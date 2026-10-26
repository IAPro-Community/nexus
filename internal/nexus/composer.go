package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// composerBootstrapTimeout bounds Create intelligence enrichment. Overridable in tests.
var composerBootstrapTimeout = intelligence.ComposerBootstrapTimeout

type ComposerSessionView struct {
	Session         store.ComposerSession         `json:"session"`
	Brief           LivingBrief                   `json:"brief"`
	Turns           []store.ComposerTurn          `json:"turns"`
	Skills          []store.ComposerSkillProposal `json:"skills"`
	Artifacts       []store.PromptArtifact        `json:"artifacts,omitempty"`
	FlowSuitability FlowSuitabilityAssessment     `json:"flow_suitability"`
	PromptReview    ComposerPromptReview          `json:"prompt_review,omitempty"`
	ContextManifest ComposerContextManifest       `json:"context_manifest"`
}

// ComposerAttachmentRef is the durable image reference embedded in a turn payload.
type ComposerAttachmentRef struct {
	ID       string `json:"id"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	URL      string `json:"url,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type FlowSuitability string

const (
	FlowDirectFit           FlowSuitability = "DIRECT_FIT"
	FlowBeneficial          FlowSuitability = "FLOW_BENEFICIAL"
	FlowStronglyRecommended FlowSuitability = "FLOW_STRONGLY_RECOMMENDED"
)

type FlowSuitabilityAssessment struct {
	Result  FlowSuitability `json:"result"`
	Reasons []string        `json:"reasons"`
	Signals []string        `json:"signals"`
}

type ComposerPromptReview struct {
	Missing        []string `json:"missing,omitempty"`
	Contradictions []string `json:"contradictions,omitempty"`
	Diff           []string `json:"diff,omitempty"`
}

func ReviewComposerPrompt(brief LivingBrief) ComposerPromptReview {
	review := ComposerPromptReview{}
	if strings.TrimSpace(brief.SourcePrompt) == "" {
		return review
	}
	if len(brief.Scope.InScope) == 0 {
		review.Missing = append(review.Missing, "scope")
	}
	if len(brief.Quality.AcceptanceCriteria) == 0 {
		review.Missing = append(review.Missing, "acceptance_criteria")
	}
	if len(brief.Quality.Verification) == 0 {
		review.Missing = append(review.Missing, "verification")
	}
	lower := strings.ToLower(brief.SourcePrompt)
	if (strings.Contains(lower, "must ") || strings.Contains(lower, "deve ")) && (strings.Contains(lower, "must not") || strings.Contains(lower, "não deve") || strings.Contains(lower, "nao deve")) {
		review.Contradictions = append(review.Contradictions, "prompt contains both mandatory and prohibitive instructions; confirm precedence")
	}
	if len(review.Missing) > 0 {
		review.Diff = append(review.Diff, "Composer added structured fields: "+strings.Join(review.Missing, ", "))
	}
	return review
}

func AssessComposerFlowSuitability(brief LivingBrief) FlowSuitabilityAssessment {
	signals, reasons := []string{}, []string{}
	if len(brief.Scope.InScope) >= 2 {
		signals = append(signals, "multiple-workstreams")
		reasons = append(reasons, "há múltiplas frentes no escopo")
	}
	if len(brief.Quality.Review) > 0 {
		signals = append(signals, "independent-review")
		reasons = append(reasons, "há revisão adicional")
	}
	if len(brief.Constraints.Security)+len(brief.Risks) > 0 {
		signals = append(signals, "risk-or-gates")
		reasons = append(reasons, "há risco ou restrição que beneficia gates")
	}
	if len(signals) >= 3 {
		return FlowSuitabilityAssessment{FlowStronglyRecommended, reasons, signals}
	}
	if len(signals) > 0 {
		return FlowSuitabilityAssessment{FlowBeneficial, reasons, signals}
	}
	return FlowSuitabilityAssessment{FlowDirectFit, []string{"execução direta é suficiente para este escopo"}, signals}
}

func (n *Nexus) CreateComposerSession(ctx context.Context, projectID, goal string) (*ComposerSessionView, error) {
	return n.CreateComposerSessionWithPrompt(ctx, projectID, goal, "")
}

func (n *Nexus) CreateComposerSessionWithPrompt(ctx context.Context, projectID, goal, sourcePrompt string) (*ComposerSessionView, error) {
	return n.CreateComposerSessionWithPromptProgress(ctx, projectID, goal, sourcePrompt, nil)
}

func (n *Nexus) CreateComposerSessionWithPromptProgress(ctx context.Context, projectID, goal, sourcePrompt string, sink ComposerProgressSink) (*ComposerSessionView, error) {
	return n.createComposerSessionWithPromptProgress(ctx, projectID, goal, sourcePrompt, ComposerInterviewStandard, sink)
}

func (n *Nexus) CreateComposerSessionWithPromptInterviewModeProgress(
	ctx context.Context,
	projectID, goal, sourcePrompt string,
	interviewMode ComposerInterviewMode,
	sink ComposerProgressSink,
) (*ComposerSessionView, error) {
	return n.createComposerSessionWithPromptProgress(ctx, projectID, goal, sourcePrompt, interviewMode, sink)
}

func (n *Nexus) createComposerSessionWithPromptProgress(
	ctx context.Context,
	projectID, goal, sourcePrompt string,
	interviewMode ComposerInterviewMode,
	sink ComposerProgressSink,
) (*ComposerSessionView, error) {
	progress := progressOrNoop(sink)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	progress.Step("brief", "started")
	brief := newComposerBriefWithInterviewMode(goal, sourcePrompt, interviewMode)
	brief.Context.Project = project.Name
	progress.Step("brief", "done")

	progress.Step("git_context", "started")
	if readiness, err := n.ObserveContextReadiness(projectID); err == nil {
		brief.Context.Evidence = appendUniqueAll(brief.Context.Evidence, []string{
			"Project: " + project.Name,
			"Branch: " + firstNonEmpty(readiness.CurrentFingerprint.Branch, "unknown"),
			"HEAD: " + firstNonEmpty(readiness.CurrentFingerprint.Head, "unknown"),
		})
		progress.Step("git_context", "done")
	} else {
		progress.Step("git_context", "skipped")
	}

	progress.Step("memories", "started")
	if facts, factErr := n.MemoryFactsMap(ctx, projectID); factErr == nil {
		ApplyMemoriesToBrief(&brief, facts)
		progress.Step("memories", "done")
	} else {
		progress.Step("memories", "skipped")
	}

	progress.Step("project_intel", "started")
	groundedContext, groundedContextErr := n.BoundedProjectIntelligenceContext(ctx, projectID)
	if groundedContextErr == nil {
		applySnapshotFactsToBrief(&brief, groundedContext)
		progress.Step("project_intel", "done")
	} else {
		progress.Step("project_intel", "skipped")
	}

	progress.Step("context_discovery", "started")
	manifest, manifestErr := n.buildComposerContextManifest(ctx, projectID, brief, groundedContext, groundedContextErr, true)
	if manifestErr != nil {
		progress.Step("context_discovery", "skipped")
	} else {
		progress.Step("context_discovery", "done")
	}

	// Heuristics (newComposerBrief -> mergeTextIntoBrief) run first; enrich the
	// brief with provider-extracted structure only when a provider is available.
	// Bound the provider budget so a hung headless CLI cannot strand Create on
	// "Starting…" forever — heuristics already seeded the brief above.
	bootCtx, bootCancel := context.WithTimeout(ctx, composerBootstrapTimeout)
	progress.Step("extract_brief", "started")
	discoveryCompleted := n.enrichComposerBriefFromText(bootCtx, projectID, &brief, firstNonEmpty(sourcePrompt, goal), &manifest)
	if bootCtx.Err() != nil {
		progress.Step("extract_brief", "skipped")
	} else {
		progress.Step("extract_brief", "done")
	}
	progress.Step("unknowns", "started")
	// A modern provider can return recommendations and material questions in
	// the same extraction call. Fall back to legacy two-call analysis only when
	// the provider did not advertise a complete discovery response.
	if !discoveryCompleted {
		n.seedComposerIntelligenceUnknowns(bootCtx, projectID, &brief, &manifest)
	}
	if bootCtx.Err() != nil {
		progress.Step("unknowns", "skipped")
	} else {
		progress.Step("unknowns", "done")
	}
	bootCancel()

	progress.Step("persist", "started")
	refreshComposerBrief(&brief)
	// Memory reuse: a valid durable preference must not be re-asked. Mark
	// memory-covered unknowns with Source MEMORY after the final refresh; the
	// current request already won any conflict inside ApplyMemoriesToBrief.
	if memFacts, memErr := n.MemoryFactsMap(ctx, projectID); memErr == nil {
		MarkMemoryCoveredUnknowns(&brief, memFacts)
	}
	if finalManifest, finalManifestErr := n.buildComposerContextManifest(ctx, projectID, brief, groundedContext, groundedContextErr, true); finalManifestErr == nil {
		manifest = finalManifest
	}
	briefJSON, _ := json.Marshal(brief)
	sessionState := composerSessionStateFromBrief(brief)
	session, err := st.CreateComposerSession(store.ComposerSession{
		ProjectID:           projectID,
		Title:               firstNonEmpty(brief.Intent.Objective, brief.Goal),
		State:               sessionState,
		ContextFingerprint:  project.CanonicalPath,
		BriefJSON:           string(briefJSON),
		ContextManifestJSON: mustComposerContextManifestJSON(manifest),
	})
	if err != nil {
		progress.Step("persist", "error")
		return nil, err
	}
	if catalog, catalogErr := n.skillCatalogForProject(projectID); catalogErr == nil {
		reconcileComposerSkillProposals(st, session.ID, brief, catalog)
	}
	// Start with a useful, inspectable discovery summary rather than a blank
	// conversation plus a mandatory question. This is derived from the LivingBrief
	// and contains no claimed repository findings that were not grounded.
	initialReply := composeAssistantReply(brief)
	if initialReply != "" {
		if _, err := st.AppendComposerTurn(store.ComposerTurn{
			SessionID: session.ID, Role: store.ComposerAssistant, Content: initialReply,
		}); err != nil {
			progress.Step("persist", "error")
			return nil, err
		}
	}
	progress.Step("persist", "done")
	return n.composeSessionView(st, *session, brief)
}

func reconcileComposerSkillProposals(st *store.Store, sessionID string, brief LivingBrief, catalog nexusskills.Catalog) {
	existing, err := st.ListComposerSkillProposals(sessionID)
	if err != nil {
		return
	}
	byID := make(map[string]store.ComposerSkillProposal, len(existing))
	for _, proposal := range existing {
		byID[proposal.SkillID] = proposal
	}

	ranked := rankComposerSkillCandidates(catalog, brief)
	current := make(map[string]struct{}, len(ranked))
	for _, candidate := range ranked {
		skill := candidate.Skill
		current[skill.ID] = struct{}{}
		if previous, ok := byID[skill.ID]; ok && previous.State != store.ComposerSkillSuggested {
			continue
		}
		_, _ = st.UpsertComposerSkillProposal(store.ComposerSkillProposal{
			SessionID:           sessionID,
			SkillID:             skill.ID,
			State:               store.ComposerSkillSuggested,
			Reason:              strings.Join(candidate.Reasons, "; "),
			Applicability:       firstNonEmpty(skill.Description, "Compatible with this Composer brief."),
			Risk:                skill.Risk,
			Source:              string(skill.Source),
			Version:             firstNonEmpty(skill.Version, "1.0.0"),
			Available:           true,
			RecommendationScore: candidate.Score,
		})
	}
	for _, proposal := range existing {
		if proposal.State != store.ComposerSkillSuggested {
			continue
		}
		if _, stillRelevant := current[proposal.SkillID]; stillRelevant {
			continue
		}
		_ = st.DeleteSuggestedComposerSkillProposal(sessionID, proposal.SkillID)
	}
}
func (n *Nexus) GetComposerSession(ctx context.Context, id string) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(id)
	if err != nil {
		return nil, err
	}
	brief := decodeComposerBrief(session.BriefJSON)
	refreshComposerBrief(&brief)
	return n.composeSessionView(st, *session, brief)
}

func (n *Nexus) ListComposerSessions(ctx context.Context, projectID string) ([]store.ComposerSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	return st.ListComposerSessions(projectID)
}

func (n *Nexus) AddComposerTurn(ctx context.Context, sessionID, role, content string) (*ComposerSessionView, error) {
	return n.AddComposerTurnExpected(ctx, sessionID, role, content, 0, nil)
}

func (n *Nexus) AddComposerTurnExpected(ctx context.Context, sessionID, role, content string, expectedRevision int, attachments []ComposerAttachmentRef) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	if session.State == store.ComposerFinalized {
		return nil, fmt.Errorf("composer session is finalized; create a new session to keep refining the prompt")
	}
	payloadJSON := ""
	kind := ""
	if len(attachments) > 0 {
		kind = "user"
		encoded, encErr := json.Marshal(map[string]any{"attachments": attachments})
		if encErr != nil {
			return nil, encErr
		}
		payloadJSON = string(encoded)
	}
	if _, err := st.AppendComposerTurn(store.ComposerTurn{
		SessionID:   sessionID,
		Role:        role,
		Content:     content,
		Kind:        kind,
		PayloadJSON: payloadJSON,
	}); err != nil {
		return nil, err
	}
	brief := decodeComposerBrief(session.BriefJSON)
	intelligenceNotice := ""
	if role == store.ComposerUser {
		if _, hasQuestion := selectComposerNextUnknownForBrief(brief); hasQuestion {
			brief.Interview.QuestionsAnswered++
		}
		mergeTextIntoBrief(&brief, content, "USER")
		if facts, factErr := n.MemoryFactsMap(ctx, session.ProjectID); factErr == nil {
			ApplyMemoriesToBrief(&brief, facts)
		}
		if shouldEnrichComposerTurn(brief, content) {
			turnCtx, turnCancel := context.WithTimeout(ctx, intelligence.ComposerTurnIntelligenceTimeout)
			if provider, providerErr := n.ConfiguredIntelligenceProvider(turnCtx, session.ProjectID); providerErr == nil && provider.Available(turnCtx) {
				contextData := map[string]any{
					"project_id": session.ProjectID,
					"brief":      compactComposerIntelligenceBrief(brief),
				}
				if manifest, manifestErr := n.BuildComposerContextManifest(turnCtx, session.ProjectID, brief); manifestErr == nil {
					contextData["context_manifest"] = manifest.IntelligenceContext()
				}
				analysis, analyzeErr := provider.AnalyzeIntent(turnCtx, content, contextData)
				if analyzeErr != nil {
					intelligenceNotice = turnIntelligenceNotice(analyzeErr)
				}
				if analysis != nil {
					brief.Intent.Objective = firstNonEmpty(analysis.Intent, brief.Intent.Objective)
					brief.Intent.DesiredOutcome = firstNonEmpty(analysis.Intent, brief.Intent.DesiredOutcome)
					brief.Constraints.Technical = appendUniqueAll(brief.Constraints.Technical, analysis.Constraints)
					for _, assumption := range analysis.Assumptions {
						brief.Assumptions = upsertAssumption(brief.Assumptions, PromptAssumption{ID: "intelligence-" + fmt.Sprint(len(brief.Assumptions)+1), Value: assumption, Confidence: "MEDIUM", Status: "INFERRED"})
					}
					var unknowns []intelligence.AmbiguityItem
					var unknownErr error
					if contextual, contextualOK := provider.(intelligence.ContextualAmbiguityEvaluator); contextualOK {
						unknowns, unknownErr = contextual.EvaluateAmbiguitiesWithContext(turnCtx, analysis, contextData)
					} else {
						unknowns, unknownErr = provider.EvaluateAmbiguities(turnCtx, analysis)
					}
					if unknownErr != nil && intelligenceNotice == "" {
						intelligenceNotice = turnIntelligenceNotice(unknownErr)
					}
					MergeIntelligenceUnknowns(&brief, unknowns)
					if extraction, extractErr := provider.ExtractBrief(turnCtx, content, contextData); extractErr == nil {
						applyBriefExtraction(&brief, extraction)
					} else if intelligenceNotice == "" {
						intelligenceNotice = turnIntelligenceNotice(extractErr)
					}
				}
			} else if providerErr != nil && !errors.Is(providerErr, intelligence.ErrIntelligenceUnavailable) {
				intelligenceNotice = turnIntelligenceNotice(providerErr)
			}
			turnCancel()
		}
		refreshComposerBrief(&brief)
		if memFacts, memErr := n.MemoryFactsMap(ctx, session.ProjectID); memErr == nil {
			MarkMemoryCoveredUnknowns(&brief, memFacts)
		}
		response := strings.TrimSpace(strings.Join([]string{intelligenceNotice, composeAssistantReply(brief)}, "\n\n"))
		if strings.TrimSpace(response) != "" {
			if _, err := st.AppendComposerTurn(store.ComposerTurn{SessionID: sessionID, Role: store.ComposerAssistant, Content: response}); err != nil {
				return nil, err
			}
		}
	}
	return n.persistComposerSessionExpected(ctx, st, session, brief, expectedRevision)
}

func (n *Nexus) UpdateComposerSkillState(ctx context.Context, sessionID, skillID, state string) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	if err := st.SetComposerSkillState(sessionID, skillID, state); err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	return n.composeSessionView(st, *session, decodeComposerBrief(session.BriefJSON))
}

func (n *Nexus) FinalizeComposerSession(ctx context.Context, sessionID string, selectedSkills []string, confirmGaps bool) (*store.PromptArtifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	brief := decodeComposerBrief(session.BriefJSON)
	refreshComposerBrief(&brief)
	blockingOpen := false
	for _, u := range brief.Unknowns {
		if u.Severity == PromptUnknownBlocking && u.Status == PromptUnknownOpen {
			blockingOpen = true
			break
		}
	}
	if blockingOpen && !confirmGaps {
		return nil, fmt.Errorf("composer has blocking gaps; confirm gaps before finalizing")
	}
	if PromptIsShallow(brief) && len(brief.OpenQuestions) > 0 && !confirmGaps {
		return nil, fmt.Errorf("composer has open questions; confirm gaps before finalizing")
	}
	catalog, err := n.skillCatalogForProject(session.ProjectID)
	if err != nil {
		return nil, err
	}
	skills, err := st.ListComposerSkillProposals(sessionID)
	if err != nil {
		return nil, err
	}
	validatedSkills, err := n.validateComposerSelectedSkills(selectedSkills, skills, catalog)
	if err != nil {
		return nil, err
	}
	proposalIDs := make(map[string]struct{}, len(skills))
	for _, proposal := range skills {
		proposalIDs[proposal.SkillID] = struct{}{}
	}
	for _, skillID := range validatedSkills {
		if _, exists := proposalIDs[skillID]; exists {
			continue
		}
		skill, ok := catalog.Resolve(skillID)
		if !ok {
			return nil, fmt.Errorf("selected skill %s is no longer available", skillID)
		}
		proposal, proposalErr := st.UpsertComposerSkillProposal(store.ComposerSkillProposal{
			SessionID: sessionID, SkillID: skill.ID, State: store.ComposerSkillSuggested,
			Reason: "Selected from the source-agnostic Nexus Skill Catalog.", Applicability: skill.Description,
			Risk: skill.Risk, Source: string(skill.Source), Version: skill.Version, Available: true,
		})
		if proposalErr != nil {
			return nil, proposalErr
		}
		skills = append(skills, *proposal)
		proposalIDs[skillID] = struct{}{}
	}
	manifest := decodeComposerContextManifest(session.ContextManifestJSON)
	if refreshedManifest, manifestErr := n.BuildComposerContextManifest(ctx, session.ProjectID, brief); manifestErr == nil {
		manifest = refreshedManifest
		session.ContextManifestJSON = mustComposerContextManifestJSON(manifest)
	}
	compiled, err := compileComposerPromptWithCatalog(renderComposerExecutionPrompt(brief, manifest), validatedSkills, catalog)
	if err != nil {
		return nil, err
	}
	frozenSkills, err := frozenSkillContractsFromCatalog(catalog, validatedSkills)
	if err != nil {
		return nil, err
	}
	contextJSON, _ := json.Marshal(map[string]any{
		"project_id": session.ProjectID, "brief": brief, "context_manifest": manifest,
		"skill_contracts": frozenSkills,
	})
	skillJSON, _ := json.Marshal(compiled.ValidatedSkills)
	artifact, err := st.CreatePromptArtifact(store.PromptArtifact{
		SessionID:    sessionID,
		Content:      compiled.CompiledPrompt,
		ContextJSON:  string(contextJSON),
		SkillIDsJSON: string(skillJSON),
	})
	if err != nil {
		return nil, err
	}
	for _, skill := range skills {
		switch {
		case containsString(validatedSkills, skill.SkillID):
			_ = st.SetComposerSkillState(sessionID, skill.SkillID, store.ComposerSkillApplied)
		case skill.State == store.ComposerSkillApplied:
			_ = st.SetComposerSkillState(sessionID, skill.SkillID, store.ComposerSkillAccepted)
		}
	}
	session.State = store.ComposerFinalized
	session.BriefJSON = mustJSON(brief)
	if err := st.UpdateComposerSession(*session); err != nil {
		return nil, err
	}
	return artifact, nil
}

// RefineComposerArtifact adds an optional refinement turn to an existing
// composer session and generates a new immutable PromptArtifact revision.
// If refinementGoal is empty the session is re-compiled as-is (useful to
// refresh after resolving unknowns). The session state is reset to allow
// the new finalization pass to proceed even when it was previously finalized.
func (n *Nexus) RefineComposerArtifact(ctx context.Context, sessionID, refinementGoal string) (*store.PromptArtifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	// Reset finalized state so turns and FinalizeComposerSession do not block.
	if session.State == store.ComposerFinalized {
		session.State = store.ComposerReady
		if err := st.UpdateComposerSession(*session); err != nil {
			return nil, err
		}
	}
	if refinementGoal = strings.TrimSpace(refinementGoal); refinementGoal != "" {
		if _, err := n.AddComposerTurn(ctx, sessionID, store.ComposerUser, "Refinamento: "+refinementGoal); err != nil {
			return nil, err
		}
	}
	// Collect accepted/applied skills to carry forward.
	skills, err := st.ListComposerSkillProposals(sessionID)
	if err != nil {
		return nil, err
	}
	selectedSkillIDs := []string{}
	for _, s := range skills {
		if s.State == store.ComposerSkillAccepted || s.State == store.ComposerSkillApplied {
			selectedSkillIDs = append(selectedSkillIDs, s.SkillID)
		}
	}
	return n.FinalizeComposerSession(ctx, sessionID, selectedSkillIDs, true)
}

// ResolveComposerUnknown updates the status and answer of a single unknown
// item in the living brief and re-evaluates readiness.
func (n *Nexus) ResolveComposerUnknown(ctx context.Context, sessionID, unknownID, answer, status string) (*ComposerSessionView, error) {
	return n.ResolveComposerUnknownExpected(ctx, sessionID, unknownID, answer, status, 0)
}

func (n *Nexus) ResolveComposerUnknownExpected(ctx context.Context, sessionID, unknownID, answer, status string, expectedRevision int) (*ComposerSessionView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(sessionID)
	if err != nil {
		return nil, err
	}
	brief := decodeComposerBrief(session.BriefJSON)
	found := false
	var resolvedField string
	for i := range brief.Unknowns {
		if brief.Unknowns[i].ID == unknownID || brief.Unknowns[i].Field == unknownID {
			wasOpen := brief.Unknowns[i].Status == PromptUnknownOpen
			brief.Unknowns[i].Answer = strings.TrimSpace(answer)
			brief.Unknowns[i].Status = PromptUnknownStatus(status)
			resolvedField = brief.Unknowns[i].Field
			if wasOpen && (status == string(PromptUnknownAnswered) || status == string(PromptUnknownConfirmed) || status == string(PromptUnknownDismissed)) {
				brief.Interview.QuestionsAnswered++
			}
			if status == string(PromptUnknownAnswered) || status == string(PromptUnknownConfirmed) {
				applyUnknownAnswerToBrief(&brief, brief.Unknowns[i])
			}
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("unknown %q not found in session %q", unknownID, sessionID)
	}
	refreshComposerBrief(&brief)
	view, err := n.persistComposerSessionExpected(ctx, st, session, brief, expectedRevision)
	if err != nil {
		return nil, err
	}
	if resolvedField != "" && !strings.HasPrefix(resolvedField, "interview.") &&
		(status == string(PromptUnknownAnswered) || status == string(PromptUnknownConfirmed)) {
		_ = n.PromoteHabitMemory(ctx, session.ProjectID, resolvedField, strings.TrimSpace(answer), MemorySourceComposer, false)
	}
	return view, nil
}

func applyUnknownAnswerToBrief(brief *LivingBrief, unknown PromptUnknown) {
	answer := strings.TrimSpace(unknown.Answer)
	if answer == "" {
		return
	}
	switch unknown.Field {
	case "scope.in_scope":
		brief.Scope.InScope = appendUnique(brief.Scope.InScope, answer)
	case "quality.acceptance_criteria":
		brief.Quality.AcceptanceCriteria = appendUnique(brief.Quality.AcceptanceCriteria, answer)
	case "quality.testing":
		brief.Quality.Testing = appendUnique(brief.Quality.Testing, answer)
	case "quality.verification":
		brief.Quality.Verification = appendUnique(brief.Quality.Verification, answer)
	case "quality.definition_of_done":
		brief.Quality.DefinitionOfDone = appendUnique(brief.Quality.DefinitionOfDone, answer)
	case "scope.out_of_scope":
		brief.Scope.OutOfScope = appendUnique(brief.Scope.OutOfScope, answer)
	case "risks":
		brief.Risks = appendUnique(brief.Risks, answer)
	case "execution.prohibited_actions":
		brief.Execution.ProhibitedActions = appendUnique(brief.Execution.ProhibitedActions, answer)
	case "deliverable.expected_artifacts":
		brief.Deliverable.ExpectedArtifacts = appendUnique(brief.Deliverable.ExpectedArtifacts, answer)
	case "constraints.security":
		brief.Constraints.Security = appendUnique(brief.Constraints.Security, answer)
	case "environment.infrastructure":
		brief.Environment.Infrastructure = appendUnique(brief.Environment.Infrastructure, answer)
	case "environment.stack":
		brief.Environment.Stack = appendUnique(brief.Environment.Stack, answer)
	case "context.existing_state":
		brief.Context.ExistingState = appendUnique(brief.Context.ExistingState, answer)
	default:
		brief.Decisions = appendUnique(brief.Decisions, unknown.Field+": "+answer)
	}
}

func (n *Nexus) composeSessionView(st *store.Store, session store.ComposerSession, brief LivingBrief) (*ComposerSessionView, error) {
	refreshComposerBrief(&brief)
	turns, err := st.ListComposerTurns(session.ID, 40)
	if err != nil {
		return nil, err
	}
	skills, err := st.ListComposerSkillProposals(session.ID)
	if err != nil {
		return nil, err
	}
	if catalog, catalogErr := n.skillCatalogForProject(session.ProjectID); catalogErr == nil {
		skills = reconcileComposerSkillAvailability(st, session.ID, skills, catalog)
	}
	artifacts, err := st.ListPromptArtifacts(session.ID)
	if err != nil {
		return nil, err
	}
	return &ComposerSessionView{
		Session:         session,
		Brief:           brief,
		Turns:           turns,
		Skills:          skills,
		Artifacts:       artifacts,
		FlowSuitability: AssessComposerFlowSuitability(brief),
		PromptReview:    ReviewComposerPrompt(brief),
		ContextManifest: decodeComposerContextManifest(session.ContextManifestJSON),
	}, nil
}

func (n *Nexus) persistComposerSessionExpected(ctx context.Context, st *store.Store, session *store.ComposerSession, brief LivingBrief, expectedRevision int) (*ComposerSessionView, error) {
	session.Title = firstNonEmpty(brief.Intent.Objective, brief.Goal, session.Title)
	if manifest, manifestErr := n.BuildComposerContextManifest(ctx, session.ProjectID, brief); manifestErr == nil {
		session.ContextManifestJSON = mustComposerContextManifestJSON(manifest)
	}
	if session.State != store.ComposerFinalized {
		session.State = composerSessionStateFromBrief(brief)
	}
	session.BriefJSON = mustJSON(brief)
	if err := st.UpdateComposerSessionExpected(*session, expectedRevision); err != nil {
		return nil, err
	}
	if catalog, catalogErr := n.skillCatalogForProject(session.ProjectID); catalogErr == nil {
		reconcileComposerSkillProposals(st, session.ID, brief, catalog)
	}
	if expectedRevision > 0 {
		session.Revision = expectedRevision + 1
	} else if latest, getErr := st.GetComposerSession(session.ID); getErr == nil {
		session.Revision = latest.Revision
	}
	return n.composeSessionView(st, *session, brief)
}

func composeAssistantReply(brief LivingBrief) string {
	understood := strings.TrimSpace(firstNonEmpty(brief.Intent.DesiredOutcome, brief.Intent.Objective, brief.Goal))
	var b strings.Builder
	if understood != "" {
		b.WriteString("**Objetivo compreendido:** ")
		b.WriteString(understood)
	}
	if len(brief.Recommendations) > 0 {
		b.WriteString("\n\n**Caminhos sugeridos pela IA (propostas, ainda não validadas):**")
		for i, recommendation := range brief.Recommendations {
			if i >= 3 {
				break
			}
			if recommendation.Status == ComposerFactDismissed {
				continue
			}
			b.WriteString("\n- ")
			b.WriteString(recommendation.Value)
		}
	}
	if len(brief.Scope.InScope) > 0 {
		b.WriteString("\n\n**O que já foi identificado para o plano:**")
		for i, item := range brief.Scope.InScope {
			if i >= 3 {
				break
			}
			b.WriteString("\n- ")
			b.WriteString(item)
		}
	}
	if open, hasOpen := selectComposerNextUnknownForBrief(brief); hasOpen && strings.TrimSpace(open.Question) != "" {
		b.WriteString("\n\n**Uma decisão que pode mudar a solução:** ")
		b.WriteString(open.Question)
		if strings.TrimSpace(open.Rationale) != "" {
			b.WriteString("\n")
			b.WriteString(open.Rationale)
		}
	} else {
		b.WriteString("\n\nVocê pode corrigir qualquer hipótese, pedir alternativas ou revisar o plano no painel lateral antes de enviar ao Agent ou preparar um Flow.")
	}
	return strings.TrimSpace(b.String())
}

// enrichComposerBriefFromText asks the configured intelligence provider to
// extract structured brief fields from free-form text and merges them into the
// brief. It is a no-op when no provider is available, so heuristics remain the
// deterministic fallback. Provider resolution mirrors seedComposerIntelligenceUnknowns.
func (n *Nexus) enrichComposerBriefFromText(ctx context.Context, projectID string, brief *LivingBrief, text string, manifest *ComposerContextManifest) bool {
	if brief == nil || strings.TrimSpace(text) == "" {
		return false
	}
	provider, err := n.ConfiguredIntelligenceProvider(ctx, projectID)
	if err != nil || provider == nil || !provider.Available(ctx) {
		return false
	}
	contextData := map[string]any{"project_id": projectID, "brief": compactComposerIntelligenceBrief(*brief)}
	if manifest != nil {
		contextData["context_manifest"] = manifest.IntelligenceContext()
	}
	if extraction, extractErr := provider.ExtractBrief(ctx, text, contextData); extractErr == nil && extraction != nil {
		applyBriefExtraction(brief, extraction)
		if extraction.DiscoveryComplete {
			MergeIntelligenceUnknowns(brief, extraction.Clarifications)
			return true
		}
	}
	return false
}

func (n *Nexus) seedComposerIntelligenceUnknowns(ctx context.Context, projectID string, brief *LivingBrief, manifest *ComposerContextManifest) {
	if brief == nil {
		return
	}
	provider, err := n.ConfiguredIntelligenceProvider(ctx, projectID)
	if err != nil || provider == nil || !provider.Available(ctx) {
		if PromptIsShallow(*brief) {
			MergeIntelligenceUnknowns(brief, []intelligence.AmbiguityItem{{
				Key: "quality.acceptance_criteria", Level: intelligence.AmbiguityBlocking,
				Question:  "Qual resultado observável define que isso está pronto?",
				Rationale: "Um prompt raso sem aceite produz trabalho raso.",
				SuggestedOptions: []string{
					"UI + testes verdes no caminho alterado",
					"API + contrato documentado",
					"Bug corrigido com teste de regressão",
				},
			}})
		}
		return
	}
	contextData := map[string]any{"project_id": projectID, "brief": compactComposerIntelligenceBrief(*brief)}
	if manifest != nil {
		contextData["context_manifest"] = manifest.IntelligenceContext()
	}
	goal := firstNonEmpty(brief.Goal, brief.Intent.Objective)
	analysis, analyzeErr := provider.AnalyzeIntent(ctx, goal, contextData)
	if analyzeErr != nil || analysis == nil {
		return
	}
	var unknowns []intelligence.AmbiguityItem
	if contextual, ok := provider.(intelligence.ContextualAmbiguityEvaluator); ok {
		unknowns, _ = contextual.EvaluateAmbiguitiesWithContext(ctx, analysis, contextData)
	} else {
		unknowns, _ = provider.EvaluateAmbiguities(ctx, analysis)
	}
	MergeIntelligenceUnknowns(brief, unknowns)
}

func applySnapshotFactsToBrief(brief *LivingBrief, grounded map[string]any) {
	if brief == nil || grounded == nil {
		return
	}
	rawFacts, _ := grounded["facts"].([]map[string]any)
	if rawFacts == nil {
		if anyFacts, ok := grounded["facts"].([]any); ok {
			for _, item := range anyFacts {
				if m, ok := item.(map[string]any); ok {
					rawFacts = append(rawFacts, m)
				}
			}
		}
	}
	for _, fact := range rawFacts {
		key := strings.ToLower(fmt.Sprint(fact["key"]))
		value := strings.TrimSpace(fmt.Sprint(fact["value"]))
		if value == "" || value == "<nil>" {
			continue
		}
		if strings.Contains(key, "test") || strings.Contains(key, "vitest") || strings.Contains(key, "go-test") {
			brief.Quality.Testing = appendUnique(brief.Quality.Testing, value)
		}
		if strings.Contains(key, "build") || strings.Contains(key, "command") {
			brief.Quality.Verification = appendUnique(brief.Quality.Verification, value)
		}
		if strings.Contains(key, "stack") || strings.Contains(key, "language") || strings.Contains(key, "framework") {
			brief.Environment.Stack = appendUnique(brief.Environment.Stack, value)
		}
	}
}

func decodeComposerBrief(raw string) LivingBrief {
	brief := LivingBrief{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &brief)
	}
	refreshComposerBrief(&brief)
	return brief
}

func composerSessionStateFromBrief(brief LivingBrief) string {
	switch brief.Readiness.State {
	case PromptReadinessReady:
		return store.ComposerReady
	case PromptReadinessReadyWithAssumption:
		return store.ComposerReadyWithGaps
	default:
		return store.ComposerExploring
	}
}

func (n *Nexus) validateComposerSelectedSkills(selected []string, available []store.ComposerSkillProposal, catalog nexusskills.Catalog) ([]string, error) {
	if len(selected) == 0 {
		return nil, nil
	}
	allowed := map[string]store.ComposerSkillProposal{}
	for _, skill := range available {
		allowed[skill.SkillID] = skill
	}
	filtered := []string{}
	for _, skillID := range selected {
		skillID = strings.TrimSpace(skillID)
		if skillID == "" {
			continue
		}
		item, ok := allowed[skillID]
		if !ok {
			resolved, resolvedOK := catalog.Resolve(skillID)
			if !resolvedOK || resolved.Availability != nexusskills.AvailabilityAvailable {
				return nil, fmt.Errorf("selected skill %s is not available in the resolved Nexus catalog", skillID)
			}
			filtered = appendUnique(filtered, skillID)
			continue
		}
		if item.State == store.ComposerSkillUnavailable {
			return nil, fmt.Errorf("selected skill %s is no longer available", skillID)
		}
		filtered = appendUnique(filtered, skillID)
	}
	return filtered, nil
}

func compileComposerPromptWithCatalog(userPrompt string, requestedSkills []string, catalog nexusskills.Catalog) (*CompiledAgentPrompt, error) {
	userPrompt = strings.TrimSpace(userPrompt)
	if userPrompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	selected := make([]nexusskills.Skill, 0, len(requestedSkills))
	for _, id := range requestedSkills {
		skill, ok := catalog.Resolve(id)
		if !ok {
			return nil, fmt.Errorf("skill validation failed: skill %q is unavailable", id)
		}
		selected = append(selected, skill)
	}
	return CompileAgentPromptWithSkillContracts(userPrompt, selected), nil
}

func reconcileComposerSkillAvailability(st *store.Store, sessionID string, proposals []store.ComposerSkillProposal, catalog nexusskills.Catalog) []store.ComposerSkillProposal {
	for i := range proposals {
		proposals[i].Available = false
		if _, ok := catalog.Resolve(proposals[i].SkillID); ok {
			proposals[i].Available = true
			continue
		}
		proposals[i].State = store.ComposerSkillUnavailable
		_ = st.SetComposerSkillState(sessionID, proposals[i].SkillID, store.ComposerSkillUnavailable)
	}
	return proposals
}

func writePromptList(b *strings.Builder, title string, values []string) {
	if len(values) == 0 {
		return
	}
	b.WriteString("# ")
	b.WriteString(title)
	b.WriteString("\n")
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(value)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueAll(values []string, additions []string) []string {
	for _, value := range additions {
		values = appendUnique(values, value)
	}
	return values
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func mustJSON(brief LivingBrief) string {
	raw, _ := json.Marshal(brief)
	return string(raw)
}
