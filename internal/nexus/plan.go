package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"

	"sort"
	"time"
)

// CreateWorkPlan creates a new structured engineering work plan.
func (n *Nexus) CreateWorkPlan(ctx context.Context, projectID, title, description string, phases []store.PlanPhase, facts map[string]string) (*store.WorkPlan, error) {
	return NewPlanApplicationService(n).Create(ctx, projectID, title, description, phases, facts)
}

// MaterializePromptArtifactAsFlow creates a first-class Flow from the exact
// finalized Composer artifact. The artifact id/version remains in structured
// facts so later revisions retain durable lineage instead of transporting a
// prompt through an opaque UI string.
func (n *Nexus) MaterializePromptArtifactAsFlow(ctx context.Context, artifactID string) (*store.WorkPlan, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	artifact, err := st.GetPromptArtifact(artifactID)
	if err != nil {
		return nil, err
	}
	session, err := st.GetComposerSession(artifact.SessionID)
	if err != nil {
		return nil, err
	}
	brief, manifest, err := composerArtifactExecutionContext(*artifact, *session)
	if err != nil {
		return nil, err
	}
	title := firstNonEmpty(brief.Intent.Objective, brief.Goal, "Composer Flow")
	facts := map[string]string{
		"nexus.source_artifact_id":         artifact.ID,
		"nexus.source_artifact_revision":   fmt.Sprintf("%d", artifact.Version),
		"nexus.source_artifact_hash":       artifact.Hash,
		"nexus.source_composer_session_id": session.ID,
		"composer.handoff.schema_version":  "1",
	}
	if strings.TrimSpace(manifest.Fingerprint) != "" {
		facts["nexus.source_context_manifest_fingerprint"] = manifest.Fingerprint
	}
	// Freeze the decision provenance with the artifact that was actually
	// approved. The Flow may project these fields, but cannot silently promote
	// model proposals and hypotheses to instructions approved by the user.
	if handoff, marshalErr := json.Marshal(composerHandoffFromArtifact(*artifact, *session, brief)); marshalErr == nil {
		facts["composer.handoff.v1"] = string(handoff)
	}
	if motivation, marshalErr := json.Marshal(brief.Motivation); marshalErr == nil {
		facts["composer.motivation_map"] = string(motivation)
	}
	for key, value := range map[string]string{
		"composer.context":   composerExecutionContextJSON(session.ProjectID, brief, manifest),
		"composer.skill_ids": artifact.SkillIDsJSON,
	} {
		if strings.TrimSpace(value) != "" {
			facts[key] = value
		}
	}
	for _, unknown := range brief.Unknowns {
		if unknown.Status != PromptUnknownAnswered && unknown.Status != PromptUnknownConfirmed {
			continue
		}
		if strings.TrimSpace(unknown.Answer) == "" {
			continue
		}
		facts["composer.fact."+unknown.Field] = unknown.Answer
	}
	for i, fact := range brief.Facts {
		// Do not flatten INFERRED/ASSUMED/PROPOSED statements into authoritative
		// WorkPlan facts. They remain explicitly typed in composer.context and
		// composer.handoff.v1 so the executor can review provenance.
		if strings.TrimSpace(fact.Value) == "" || fact.Status != ComposerFactConfirmed {
			continue
		}
		key := firstNonEmpty(fact.Field, fmt.Sprintf("composer.fact.%d", i))
		if _, exists := facts[key]; !exists {
			facts[key] = fact.Value
		}
	}
	// Do not enrich this frozen artifact with fresh semantic-memory values.
	// Composer's approved context was captured when the artifact was finalized;
	// new Maestro facts must not silently rewrite the approved handoff.

	// The immutable PromptArtifact is the ONLY source prompt for this handoff.
	// The planner may propose activities, but cannot rewrite approved Composer
	// decisions or silently fall back to one generic work package.
	sourcePrompt := strings.TrimSpace(artifact.Content)
	if sourcePrompt == "" {
		return nil, fmt.Errorf("Composer artifact %s has no finalized prompt", artifact.ID)
	}
	frozenHandoff := composerHandoffFromArtifact(*artifact, *session, brief)
	intent, outlines, generatedFacts, err := n.generateSemanticFlowOutlines(ctx, session.ProjectID, sourcePrompt, map[string]any{
		"source":               "FINALIZED_COMPOSER_ARTIFACT",
		"source_artifact_id":   artifact.ID,
		"source_artifact_version": artifact.Version,
		"source_artifact_hash": artifact.Hash,
		"approved_handoff":    frozenHandoff,
		"context_manifest_fingerprint": manifest.Fingerprint,
	})
	if err != nil {
		return nil, fmt.Errorf("decompose finalized Composer artifact %s: %w", artifact.ID, err)
	}
	for key, value := range generatedFacts {
		if _, frozen := facts[key]; !frozen {
			facts[key] = value
		}
	}
	facts["scope"] = intent.Scope
	facts["risk_level"] = intent.RiskLevel

	// Reuse the SAME work package materialization as direct Flow. Only the
	// transport-specific Composer metadata and frozen provenance are added.
	packages := packagesFromOutlines(sourcePrompt, outlines)
	project, err := st.GetProject(session.ProjectID)
	if err != nil {
		return nil, err
	}
	verification := detectVerificationCommands(project.CanonicalPath)
	skills := decodeStringArray(artifact.SkillIDsJSON)
	for i := range packages {
		if len(packages[i].VerificationRequirements) == 0 {
			packages[i].VerificationRequirements = append([]string(nil), verification...)
		}
		packages[i].TaskRequirements = taskRequirementsJSON(packages[i].Goal, packages[i].Role)
		packages[i].SkillIDs = append([]string(nil), skills...)
		packages[i].MaestroSkills = append([]string(nil), skills...)
	}
	if len(packages) == 1 {
		// Direct/atomic execution must preserve the standalone complete prompt.
		// Multi-step work is compiled from each task's scoped goal instead.
		packages[0].CompiledPrompt = artifact.Content
	}
	phase := store.PlanPhase{
		ID: "phase_" + ids.NewRuntimeID(), Title: "Composer Flow",
		Order: 1, Packages: packages,
	}
	return n.CreateWorkPlan(ctx, session.ProjectID, title,
		"Materialized from Composer PromptArtifact via Flow decomposition",
		[]store.PlanPhase{phase}, facts)
}

// composerHandoffContract is a versioned, self-contained interpretation of the
// immutable PromptArtifact. Structured facts carry it through the Flow editor
// and into the Mission snapshot without changing the execution engine.
type composerHandoffContract struct {
	Version         int                       `json:"version"`
	SessionID       string                    `json:"session_id"`
	ArtifactID      string                    `json:"artifact_id"`
	ArtifactVersion int                       `json:"artifact_version"`
	ArtifactHash    string                    `json:"artifact_hash"`
	Goal            string                    `json:"goal"`
	ApprovedAnswers map[string]string         `json:"approved_answers,omitempty"`
	Assumptions     []PromptAssumption        `json:"assumptions,omitempty"`
	Recommendations []ComposerFact            `json:"recommendations,omitempty"`
	Unresolved      []PromptUnknown           `json:"unresolved,omitempty"`
	Acceptance      []string                  `json:"acceptance,omitempty"`
	Risks           []string                  `json:"risks,omitempty"`
	Route           FlowSuitabilityAssessment `json:"route"`
}

func composerHandoffFromArtifact(artifact store.PromptArtifact, session store.ComposerSession, brief LivingBrief) composerHandoffContract {
	approved := map[string]string{}
	unresolved := []PromptUnknown{}
	for _, u := range brief.Unknowns {
		if (u.Status == PromptUnknownAnswered || u.Status == PromptUnknownConfirmed) && strings.TrimSpace(u.Answer) != "" {
			approved[u.Field] = u.Answer
		} else if u.Status == PromptUnknownOpen && u.Severity == PromptUnknownBlocking {
			// A confirmed-gap finalization never converts an unresolved risk into
			// an approved decision. Admission must still validate the plan.
			unresolved = append(unresolved, u)
		}
	}
	return composerHandoffContract{
		Version: 1, SessionID: session.ID, ArtifactID: artifact.ID,
		ArtifactVersion: artifact.Version, ArtifactHash: artifact.Hash,
		Goal:            firstNonEmpty(brief.Intent.Objective, brief.Goal),
		ApprovedAnswers: approved,
		Assumptions:     append([]PromptAssumption(nil), brief.Assumptions...),
		Recommendations: append([]ComposerFact(nil), brief.Recommendations...),
		Unresolved:      unresolved,
		Acceptance:      append([]string(nil), brief.Quality.AcceptanceCriteria...),
		Risks:           append([]string(nil), brief.Risks...),
		Route:           AssessComposerFlowSuitability(brief),
	}
}

// composerHandoffExecutionBlockers is intentionally independent of AI service
// availability. Admission can validate the frozen, typed Composer contract
// offline. Legacy and manually-authored plans have no handoff and are unchanged.
func composerHandoffExecutionBlockers(facts map[string]string) []string {
	raw := strings.TrimSpace(facts["composer.handoff.v1"])
	if raw == "" {
		return nil
	}
	var handoff composerHandoffContract
	if err := json.Unmarshal([]byte(raw), &handoff); err != nil || handoff.Version != 1 {
		return []string{"Composer handoff inválido ou incompatível; revise e recrie o Flow a partir do artefato aprovado"}
	}
	var blockers []string
	for _, unresolved := range handoff.Unresolved {
		if unresolved.Status != PromptUnknownOpen || unresolved.Severity != PromptUnknownBlocking {
			continue
		}
		// Include a bounded, actionable description. Never echo the entire
		// submitted prompt or potentially sensitive model output here.
		label := firstNonEmpty(strings.TrimSpace(unresolved.Question), strings.TrimSpace(unresolved.Field), "decisão necessária")
		if len([]rune(label)) > 180 {
			label = string([]rune(label)[:180])
		}
		blockers = append(blockers, "Composer: decisão bloqueante ainda não resolvida — "+label)
	}
	return blockers
}

type composerArtifactContextEnvelope struct {
	ProjectID       string                  `json:"project_id"`
	Brief           LivingBrief             `json:"brief"`
	ContextManifest ComposerContextManifest `json:"context_manifest"`
	SkillContracts  []FrozenSkillContract   `json:"skill_contracts,omitempty"`
}

func composerArtifactExecutionContext(artifact store.PromptArtifact, session store.ComposerSession) (LivingBrief, ComposerContextManifest, error) {
	var envelope composerArtifactContextEnvelope
	if strings.TrimSpace(artifact.ContextJSON) != "" {
		if err := json.Unmarshal([]byte(artifact.ContextJSON), &envelope); err != nil {
			return LivingBrief{}, ComposerContextManifest{}, fmt.Errorf("decode Composer artifact context: %w", err)
		}
	}
	if strings.TrimSpace(envelope.ProjectID) != "" && envelope.ProjectID != session.ProjectID {
		return LivingBrief{}, ComposerContextManifest{}, fmt.Errorf("composer artifact project %s does not match session project %s", envelope.ProjectID, session.ProjectID)
	}
	if strings.TrimSpace(envelope.ContextManifest.ProjectID) != "" && envelope.ContextManifest.ProjectID != session.ProjectID {
		return LivingBrief{}, ComposerContextManifest{}, fmt.Errorf("composer context manifest project %s does not match session project %s", envelope.ContextManifest.ProjectID, session.ProjectID)
	}
	if strings.TrimSpace(envelope.Brief.Goal) != "" || strings.TrimSpace(envelope.Brief.Intent.Objective) != "" {
		return envelope.Brief, envelope.ContextManifest, nil
	}

	// Legacy PromptArtifacts predate the frozen Composer envelope. Preserve
	// compatibility by falling back to the session brief only for those rows.
	var legacyBrief LivingBrief
	if err := json.Unmarshal([]byte(session.BriefJSON), &legacyBrief); err != nil {
		return LivingBrief{}, ComposerContextManifest{}, fmt.Errorf("decode legacy Composer brief: %w", err)
	}
	return legacyBrief, decodeComposerContextManifest(session.ContextManifestJSON), nil
}

func decodeStringArray(raw string) []string {
	var values []string
	if json.Unmarshal([]byte(raw), &values) != nil {
		return []string{}
	}
	return values
}

// GetWorkPlan fetches a plan by ID.
func (n *Nexus) GetWorkPlan(ctx context.Context, planID string) (*store.WorkPlan, error) {
	return NewPlanApplicationService(n).Get(ctx, planID)
}

// ListWorkPlans lists all work plans for a project.
func (n *Nexus) ListWorkPlans(ctx context.Context, projectID string) ([]store.WorkPlan, error) {
	return NewPlanApplicationService(n).List(ctx, projectID)
}

// UpdateWorkPlan updates a plan and records a new PlanRevision atomically.
func (n *Nexus) UpdateWorkPlan(ctx context.Context, plan store.WorkPlan, changeSummary string) (*store.WorkPlan, *store.PlanRevision, error) {
	return NewPlanApplicationService(n).Update(ctx, plan, changeSummary)
}

// DeleteWorkPlan deletes a plan by ID.
func (n *Nexus) DeleteWorkPlan(ctx context.Context, planID string) error {
	return NewPlanApplicationService(n).Delete(ctx, planID)
}

// ListPlanRevisions lists all historical revisions of a plan.
func (n *Nexus) ListPlanRevisions(ctx context.Context, planID string) ([]store.PlanRevision, error) {
	return NewPlanApplicationService(n).Revisions(ctx, planID)
}

// CompilePackagePrompt compiles the exact scoped prompt for a single WorkPackage.
func (n *Nexus) CompilePackagePrompt(ctx context.Context, planID, phaseID, packageID string) (*intelligence.PromptCompilationResult, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	plan, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, err
	}
	return n.compilePackagePromptFromPlan(ctx, plan, phaseID, packageID)
}

// compilePackagePromptFromPlan compiles from an immutable in-memory plan
// snapshot. Mission execution uses this function so edits to the live WorkPlan
// cannot mutate prompts for an already-approved run.
func (n *Nexus) compilePackagePromptFromPlan(ctx context.Context, plan *store.WorkPlan, phaseID, packageID string) (*intelligence.PromptCompilationResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("work plan snapshot is required")
	}
	var targetPkg *store.WorkPackage
	for pi := range plan.Phases {
		ph := &plan.Phases[pi]
		if phaseID != "" && ph.ID != phaseID {
			continue
		}
		for i := range ph.Packages {
			if ph.Packages[i].ID == packageID {
				targetPkg = &ph.Packages[i]
				break
			}
		}
		if targetPkg != nil {
			break
		}
	}
	if targetPkg == nil {
		return nil, fmt.Errorf("work package %s not found in plan %s", packageID, plan.ID)
	}

	requestedSkills := packageSkillIDs(*targetPkg)
	validatedSkills, err := validateGenericSkillIDs(n, plan.ProjectID, requestedSkills)
	if err != nil {
		return nil, err
	}
	catalog, err := n.skillCatalogForProject(plan.ProjectID)
	if err != nil {
		return nil, err
	}
	contracts, err := frozenSkillContractsFromCatalog(catalog, validatedSkills)
	if err != nil {
		return nil, err
	}
	return compileTargetPackagePromptForAgent(ctx, plan, targetPkg, intelligence.AgentSpec{Role: targetPkg.Role}, validatedSkills, contracts...)
}

func compileTargetPackagePromptForAgent(ctx context.Context, plan *store.WorkPlan, targetPkg *store.WorkPackage, agentSpec intelligence.AgentSpec, validatedSkills []string, frozenContracts ...FrozenSkillContract) (*intelligence.PromptCompilationResult, error) {
	guidance := guidanceForWorkPackage(targetPkg)
	compiled, err := intelligence.NewNexusEngine(nil).CompileExecutionContext(ctx, intelligence.ExecutionContextRequest{
		Agent:    agentSpec,
		Project:  intelligence.ProjectContext{Facts: plan.StructuredFacts},
		Task:     intelligence.WorkPackageContext{Title: targetPkg.Title, Goal: targetPkg.Goal, Priority: targetPkg.Priority, Role: targetPkg.Role, AcceptanceCriteria: targetPkg.AcceptanceCriteria},
		Skills:   append([]string(nil), validatedSkills...),
		Guidance: guidance,
	})
	if err != nil {
		return nil, err
	}
	result := &intelligence.PromptCompilationResult{
		PackageTitle:    targetPkg.Title,
		SystemPrompt:    compiled.SystemInstructions,
		UserPrompt:      compiled.TaskInstructions,
		Skills:          append([]string(nil), validatedSkills...),
		AcceptanceGates: append([]string(nil), targetPkg.AcceptanceCriteria...),
		Constraints:     append([]string(nil), compiled.Context...),
		EstimatedTokens: (len(compiled.SystemInstructions) + len(compiled.TaskInstructions)) / 4,
		CompiledAt:      time.Now().UTC(),
	}
	if len(validatedSkills) > 0 && len(frozenContracts) > 0 {
		selected, selectErr := frozenSkillContractsForIDs(frozenContracts, validatedSkills)
		if selectErr != nil {
			return nil, selectErr
		}
		var skillSections []string
		for _, contract := range selected {
			if strings.TrimSpace(contract.Instructions) == "" {
				continue
			}
			skillSections = append(skillSections, fmt.Sprintf(
				"### %s (source=%s version=%s skill_hash=%s contract_hash=%s)\n%s",
				contract.ID, contract.Source, contract.Version, contract.SkillHash, contract.ContractHash, contract.Instructions,
			))
		}
		if len(skillSections) > 0 {
			result.SystemPrompt += "\n\n## Frozen Nexus Skill Contracts\n" + strings.Join(skillSections, "\n\n")
			result.EstimatedTokens = (len(result.SystemPrompt) + len(result.UserPrompt)) / 4
		}
	}
	return result, nil
}

func guidanceForWorkPackage(pkg *store.WorkPackage) intelligence.ExecutionGuidance {
	if pkg == nil || strings.TrimSpace(pkg.TaskRequirements) == "" {
		return intelligence.ExecutionGuidance{}
	}
	var req TaskRequirements
	if err := json.Unmarshal([]byte(pkg.TaskRequirements), &req); err != nil || req.Guidance == nil {
		return intelligence.ExecutionGuidance{}
	}
	guidance := *req.Guidance
	guidance.Instructions = append([]string(nil), req.Guidance.Instructions...)
	guidance.Skills = append([]string(nil), req.Guidance.Skills...)
	return guidance
}

// compilePackagePromptFromExecutionSnapshot trusts only Maestro gates already
// frozen into the approved snapshot. It never consults mutable live planning state.
func compilePackagePromptFromExecutionSnapshot(ctx context.Context, plan *store.WorkPlan, phaseID, packageID string, frozenContracts []FrozenSkillContract) (*intelligence.PromptCompilationResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("work plan snapshot is required")
	}
	for pi := range plan.Phases {
		ph := &plan.Phases[pi]
		if phaseID != "" && ph.ID != phaseID {
			continue
		}
		for i := range ph.Packages {
			pkg := &ph.Packages[i]
			if pkg.ID == packageID {
				return compileTargetPackagePromptForAgent(ctx, plan, pkg, intelligence.AgentSpec{Role: pkg.Role}, packageSkillIDs(*pkg), frozenContracts...)
			}
		}
	}
	return nil, fmt.Errorf("work package %s not found in execution snapshot", packageID)
}

// ClarificationCheckpoint is the durable user-facing representation of a blocking ambiguity set.
type ClarificationCheckpoint struct {
	ID        string                       `json:"id"`
	ProjectID string                       `json:"project_id"`
	Goal      string                       `json:"goal"`
	Status    string                       `json:"status"`
	Intent    *intelligence.IntentAnalysis `json:"intent"`
	Unknowns  []intelligence.AmbiguityItem `json:"unknowns"`
	Facts     map[string]string            `json:"facts"`
}

// ClarificationRequiredError allows the REST layer to return a structured 409 instead of a generic 500.
type ClarificationRequiredError struct{ Checkpoint *ClarificationCheckpoint }

func (e *ClarificationRequiredError) Error() string {
	return "blocking clarification required before plan generation"
}

// GeneratePlanFromIntent uses a real configured Nexus Intelligence provider.
// It persists BLOCKING ambiguities and never falls back to fabricated packages.
// CLI providers use a single-shot PlanFromGoal call; OpenAI-compatible keeps multi-call.
//
// GeneratePlanFromIntent uses the configured intelligence provider to create a
// semantic, reviewable WorkPlan from a goal or complete prompt. Composer is not
// required. It never starts a Mission and rejects malformed AI topology.
func (n *Nexus) GeneratePlanFromIntent(ctx context.Context, projectID, goal string) (*store.WorkPlan, error) {
	intent, outlines, facts, err := n.generateSemanticFlowOutlines(ctx, projectID, goal, nil)
	if err != nil {
		return nil, err
	}
	return n.createPlanFromOutlines(ctx, projectID, goal, intent, outlines, facts)
}

// generateSemanticFlowOutlines shares the same provider, clarification,
// dependency and acceptance gates between direct Flow and finalized Composer
// artifacts. The frozenContext is never substituted with a live session.
func (n *Nexus) generateSemanticFlowOutlines(ctx context.Context, projectID, goal string, frozenContext map[string]any) (*intelligence.IntentAnalysis, []intelligence.WorkPackageOutline, map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, intelligence.PlanGenerationTimeout)
	defer cancel()

	// The Composer handoff supplies frozen, artifact-scoped context rather than
	// using a later mutable Composer session or a fresh semantic-memory read.
	contextData := frozenContext
	if contextData == nil {
		var err error
		contextData, err = n.ComposerContextData(projectID)
		if err != nil {
			return nil, nil, nil, err
		}
		if memFacts, memErr := n.MemoryFactsMap(ctx, projectID); memErr == nil && len(memFacts) > 0 {
			contextData["intelligence_memories"] = memFacts
		}
	}
	provider, err := n.ConfiguredIntelligenceProvider(ctx, projectID)
	if err != nil {
		return nil, nil, nil, err
	}
	engine := intelligence.NewNexusEngine(provider).WithContextData(contextData)

	intent, unknowns, outlines, err := engine.PlanFromGoal(ctx, goal, projectID)
	if err != nil {
		return nil, nil, nil, err
	}
	if intent == nil {
		return nil, nil, nil, fmt.Errorf("intelligence planner returned no intent")
	}

	state := &intelligence.ClarificationState{Unknowns: unknowns, StructuredFacts: map[string]string{}}
	if frozenContext == nil {
		if memFacts, memErr := n.MemoryFactsMap(ctx, projectID); memErr == nil {
			for key, value := range memFacts {
				state.StructuredFacts[key] = value
			}
			// Resolve blocking unknowns already covered by durable memories.
			for i := range state.Unknowns {
				item := &state.Unknowns[i]
				if item.IsResolved {
					continue
				}
				if value := memoryCoversAmbiguity(*item, memFacts); value != "" {
					engine.ResolveClarification(state, item.Key, value)
				}
			}
		}
	}
	// Non-blocking defaults are explicit decisions. Blocking items always require a user answer.
	for i := range state.Unknowns {
		item := state.Unknowns[i]
		if item.Level != intelligence.AmbiguityBlocking && item.DefaultChoice != "" {
			engine.ResolveClarification(state, item.Key, item.DefaultChoice)
		}
	}
	if hasBlockingUnknowns(state.Unknowns) {
		if frozenContext != nil {
			// A generic clarification continuation would lose the artifact
			// lineage. Require a revised, finalized Composer artifact instead.
			return nil, nil, nil, fmt.Errorf("Composer prompt contains unresolved blocking questions; revise and finalize the Composer artifact before converting it to Flow")
		}
		checkpoint, err := n.persistClarification(projectID, goal, intent, state.Unknowns, state.StructuredFacts)
		if err != nil {
			return nil, nil, nil, err
		}
		return nil, nil, nil, &ClarificationRequiredError{Checkpoint: checkpoint}
	}

	if len(outlines) == 0 {
		outlines, err = engine.GeneratePlan(ctx, intent, state.StructuredFacts)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	// A response from an AI provider is not proof that it is executable.
	// Fail before writing a WorkPlan when tasks, dependencies or acceptance
	// evidence are incomplete; never silently downgrade to one generic step.
	if err := validateGeneratedFlowOutlines(goal, outlines); err != nil {
		return nil, nil, nil, err
	}
	return intent, outlines, state.StructuredFacts, nil
}

func hasBlockingUnknowns(items []intelligence.AmbiguityItem) bool {
	for _, item := range items {
		if item.Level == intelligence.AmbiguityBlocking && !item.IsResolved {
			return true
		}
	}
	return false
}

func memoryCoversAmbiguity(item intelligence.AmbiguityItem, facts map[string]string) string {
	keys := []string{item.Key, "composer.fact." + item.Key, "memory.project." + item.Key, "memory.install." + item.Key}
	lowerKey := strings.ToLower(item.Key)
	for factKey, value := range facts {
		if strings.TrimSpace(value) == "" {
			continue
		}
		fk := strings.ToLower(factKey)
		if fk == lowerKey || strings.HasSuffix(fk, "."+lowerKey) {
			return value
		}
		if strings.Contains(lowerKey, "verif") && (strings.Contains(fk, "verif") || strings.Contains(fk, "quality.verification")) {
			return value
		}
		if strings.Contains(lowerKey, "test") && (strings.Contains(fk, "test") || strings.Contains(fk, "quality.testing")) {
			return value
		}
	}
	for _, key := range keys {
		if v := strings.TrimSpace(facts[key]); v != "" {
			return v
		}
	}
	return ""
}

func (n *Nexus) persistClarification(projectID, goal string, intent *intelligence.IntentAnalysis, unknowns []intelligence.AmbiguityItem, facts map[string]string) (*ClarificationCheckpoint, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	intentJSON, _ := json.Marshal(intent)
	unknownsJSON, _ := json.Marshal(unknowns)
	factsJSON, _ := json.Marshal(facts)
	row, err := st.CreateClarification(store.Clarification{
		ProjectID: projectID, Goal: goal, Status: store.ClarificationPending,
		IntentJSON: string(intentJSON), UnknownsJSON: string(unknownsJSON), FactsJSON: string(factsJSON),
	})
	if err != nil {
		return nil, err
	}
	return clarificationCheckpoint(row)
}

func clarificationCheckpoint(row *store.Clarification) (*ClarificationCheckpoint, error) {
	if row == nil {
		return nil, fmt.Errorf("clarification not found")
	}
	var intent intelligence.IntentAnalysis
	var unknowns []intelligence.AmbiguityItem
	facts := map[string]string{}
	if err := json.Unmarshal([]byte(row.IntentJSON), &intent); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(row.UnknownsJSON), &unknowns); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(row.FactsJSON), &facts); err != nil {
		return nil, err
	}
	return &ClarificationCheckpoint{ID: row.ID, ProjectID: row.ProjectID, Goal: row.Goal, Status: row.Status, Intent: &intent, Unknowns: unknowns, Facts: facts}, nil
}

func (n *Nexus) GetClarification(ctx context.Context, id string) (*ClarificationCheckpoint, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	row, err := st.GetClarification(id)
	if err != nil {
		return nil, err
	}
	return clarificationCheckpoint(row)
}

// ResolveClarificationAndGeneratePlan records answers as durable facts, then continues
// the original analysis without re-asking already resolved questions.
func (n *Nexus) ResolveClarificationAndGeneratePlan(ctx context.Context, id string, answers map[string]string) (*store.WorkPlan, *ClarificationCheckpoint, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, nil, err
	}
	row, err := st.GetClarification(id)
	if err != nil {
		return nil, nil, err
	}
	checkpoint, err := clarificationCheckpoint(row)
	if err != nil {
		return nil, nil, err
	}
	contextData, err := n.ComposerContextData(checkpoint.ProjectID)
	if err != nil {
		return nil, checkpoint, err
	}
	provider, err := n.ConfiguredIntelligenceProvider(ctx, checkpoint.ProjectID)
	if err != nil {
		return nil, checkpoint, err
	}
	engine := intelligence.NewNexusEngine(provider).WithContextData(contextData)
	state := &intelligence.ClarificationState{Unknowns: checkpoint.Unknowns, StructuredFacts: checkpoint.Facts}
	for key, answer := range answers {
		if strings.TrimSpace(answer) != "" {
			engine.ResolveClarification(state, key, strings.TrimSpace(answer))
		}
	}
	unknownsJSON, _ := json.Marshal(state.Unknowns)
	factsJSON, _ := json.Marshal(state.StructuredFacts)
	row.UnknownsJSON, row.FactsJSON = string(unknownsJSON), string(factsJSON)
	if hasBlockingUnknowns(state.Unknowns) {
		row.Status = store.ClarificationPending
		if err := st.UpdateClarification(*row); err != nil {
			return nil, checkpoint, err
		}
		updated, _ := clarificationCheckpoint(row)
		return nil, updated, &ClarificationRequiredError{Checkpoint: updated}
	}
	outlines, err := engine.GeneratePlan(ctx, checkpoint.Intent, state.StructuredFacts)
	if err != nil {
		return nil, checkpoint, err
	}
	// Answering clarifications must not bypass the same semantic DAG gate.
	// Keep the checkpoint unresolved if the new plan is still untrustworthy.
	if err := validateGeneratedFlowOutlines(checkpoint.Goal, outlines); err != nil {
		return nil, checkpoint, err
	}
	row.Status = store.ClarificationResolved
	if err := st.UpdateClarification(*row); err != nil {
		return nil, checkpoint, err
	}
	plan, err := n.createPlanFromOutlines(ctx, checkpoint.ProjectID, checkpoint.Goal, checkpoint.Intent, outlines, state.StructuredFacts)
	updated, _ := clarificationCheckpoint(row)
	return plan, updated, err
}

func isSchemaPlaceholder(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	switch lower {
	case "...", "…", "title", "goal", "package title", "measurable criterion", "specific measurable objective":
		return true
	}
	if strings.HasPrefix(trimmed, "<") && strings.HasSuffix(trimmed, ">") {
		return true
	}
	if strings.Trim(lower, ".…") == "" {
		return true
	}
	return false
}

func sanitizeAcceptance(values []string) []string {
	var out []string
	for _, value := range values {
		if isSchemaPlaceholder(value) {
			continue
		}
		out = append(out, strings.TrimSpace(value))
	}
	return out
}

func packagesFromOutlines(goal string, outlines []intelligence.WorkPackageOutline) []store.WorkPackage {
	goal = strings.TrimSpace(goal)
	var pkgs []store.WorkPackage
	for i, o := range outlines {
		title := strings.TrimSpace(o.Title)
		pkgGoal := strings.TrimSpace(o.Goal)
		if isSchemaPlaceholder(title) {
			if !isSchemaPlaceholder(pkgGoal) {
				title = pkgGoal
			} else if goal != "" {
				title = goal
			} else {
				title = fmt.Sprintf("Work step %d", i+1)
			}
		}
		if isSchemaPlaceholder(pkgGoal) {
			pkgGoal = goal
			if pkgGoal == "" {
				pkgGoal = title
			}
		}
		priority := strings.TrimSpace(o.Priority)
		if priority == "" || isSchemaPlaceholder(priority) {
			priority = "HIGH"
		}
		role := strings.TrimSpace(o.Role)
		if role == "" || isSchemaPlaceholder(role) {
			role = "implementer"
		}
		pkgs = append(pkgs, store.WorkPackage{
			ID: "pkg_" + ids.NewRuntimeID(), Title: title, Goal: pkgGoal, Priority: priority,
			Status: "READY", Dependencies: append([]string(nil), o.Dependencies...), Role: role,
			// Intelligence cannot mint Maestro skill/gate IDs. Maestro assignment is a separate explicit step.
			MaestroGates: []string{}, AcceptanceCriteria: sanitizeAcceptance(o.Acceptance),
		})
	}
	if len(pkgs) == 0 {
		title := goal
		if title == "" {
			title = "Work step"
		}
		pkgs = []store.WorkPackage{{
			ID: "pkg_" + ids.NewRuntimeID(), Title: title, Goal: goal, Priority: "HIGH",
			Status: "READY", Role: "implementer", MaestroGates: []string{},
		}}
	}
	byID := make(map[string]struct{}, len(pkgs))
	byTitle := make(map[string]string, len(pkgs)*4)
	for position, pkg := range pkgs {
		byID[pkg.ID] = struct{}{}
		key := strings.ToLower(strings.TrimSpace(pkg.Title))
		if key != "" {
			if _, exists := byTitle[key]; !exists {
				byTitle[key] = pkg.ID
			}
		}
		ordinal := fmt.Sprintf("%d", position+1)
		for _, alias := range []string{ordinal, "#"+ordinal, "step "+ordinal, "etapa "+ordinal, "atividade "+ordinal} {
			byTitle[alias] = pkg.ID
		}
	}
	for i := range pkgs {
		seen := map[string]struct{}{}
		var resolved []string
		for _, dep := range pkgs[i].Dependencies {
			dep = strings.TrimSpace(dep)
			if dep == "" || isSchemaPlaceholder(dep) || dep == pkgs[i].ID {
				continue
			}
			id := dep
			if _, ok := byID[dep]; !ok {
				mapped, ok := byTitle[strings.ToLower(dep)]
				if !ok || mapped == pkgs[i].ID {
					continue
				}
				id = mapped
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			resolved = append(resolved, id)
		}
		pkgs[i].Dependencies = resolved
		if len(resolved) > 0 {
			pkgs[i].Status = "PENDING"
		}
	}
	return pkgs
}

func (n *Nexus) createPlanFromOutlines(ctx context.Context, projectID, goal string, intent *intelligence.IntentAnalysis, outlines []intelligence.WorkPackageOutline, facts map[string]string) (*store.WorkPlan, error) {
	pkgs := packagesFromOutlines(goal, outlines)
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	verification := detectVerificationCommands(project.CanonicalPath)
	for i := range pkgs {
		if len(pkgs[i].VerificationRequirements) == 0 {
			pkgs[i].VerificationRequirements = append([]string(nil), verification...)
		}
	}
	phase := store.PlanPhase{ID: "phase_" + ids.NewRuntimeID(), Title: "Execution Phase", Order: 1, Packages: pkgs}
	mergedFacts := map[string]string{}
	for k, v := range facts {
		mergedFacts[k] = v
	}
	mergedFacts["scope"] = intent.Scope
	mergedFacts["risk_level"] = intent.RiskLevel
	return n.CreateWorkPlan(ctx, projectID, intent.Intent, fmt.Sprintf("AI-generated plan for: %s", goal), []store.PlanPhase{phase}, mergedFacts)
}

// PlanRevisionDiff is a semantic diff suitable for UI/audit without relying on
// textual JSON ordering.
type PlanRevisionDiff struct {
	FromRevision       int      `json:"from_revision"`
	ToRevision         int      `json:"to_revision"`
	TitleChanged       bool     `json:"title_changed"`
	DescriptionChanged bool     `json:"description_changed"`
	AddedPackages      []string `json:"added_packages"`
	RemovedPackages    []string `json:"removed_packages"`
	ChangedPackages    []string `json:"changed_packages"`
}

func (n *Nexus) ComparePlanRevisions(ctx context.Context, planID string, fromRevision, toRevision int) (*PlanRevisionDiff, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	fromRow, err := st.GetPlanRevision(planID, fromRevision)
	if err != nil {
		return nil, err
	}
	toRow, err := st.GetPlanRevision(planID, toRevision)
	if err != nil {
		return nil, err
	}
	var from, to store.WorkPlan
	if err := json.Unmarshal([]byte(fromRow.SnapshotJSON), &from); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(toRow.SnapshotJSON), &to); err != nil {
		return nil, err
	}
	diff := &PlanRevisionDiff{FromRevision: fromRevision, ToRevision: toRevision, TitleChanged: from.Title != to.Title, DescriptionChanged: from.Description != to.Description}
	left, right := flattenPackages(from), flattenPackages(to)
	for id, pkg := range right {
		old, ok := left[id]
		if !ok {
			diff.AddedPackages = append(diff.AddedPackages, id)
			continue
		}
		oldJSON, _ := json.Marshal(old)
		newJSON, _ := json.Marshal(pkg)
		if string(oldJSON) != string(newJSON) {
			diff.ChangedPackages = append(diff.ChangedPackages, id)
		}
	}
	for id := range left {
		if _, ok := right[id]; !ok {
			diff.RemovedPackages = append(diff.RemovedPackages, id)
		}
	}
	sort.Strings(diff.AddedPackages)
	sort.Strings(diff.RemovedPackages)
	sort.Strings(diff.ChangedPackages)
	return diff, nil
}

func flattenPackages(plan store.WorkPlan) map[string]store.WorkPackage {
	out := map[string]store.WorkPackage{}
	for _, phase := range plan.Phases {
		for _, pkg := range phase.Packages {
			out[pkg.ID] = pkg
		}
	}
	return out
}

// RestoreWorkPlanRevision creates a NEW revision from a historical immutable
// snapshot; history is never rewritten or deleted.
func (n *Nexus) RestoreWorkPlanRevision(ctx context.Context, planID string, revision int) (*store.WorkPlan, *store.PlanRevision, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, nil, err
	}
	row, err := st.GetPlanRevision(planID, revision)
	if err != nil {
		return nil, nil, err
	}
	var historical store.WorkPlan
	if err := json.Unmarshal([]byte(row.SnapshotJSON), &historical); err != nil {
		return nil, nil, err
	}
	current, err := st.GetWorkPlan(planID)
	if err != nil {
		return nil, nil, err
	}
	historical.ID = current.ID
	historical.ProjectID = current.ProjectID
	historical.MissionID = current.MissionID
	historical.CurrentRevision = current.CurrentRevision
	historical.CreatedAt = current.CreatedAt
	return st.UpdateWorkPlan(historical, fmt.Sprintf("Restore from revision %d", revision))
}
