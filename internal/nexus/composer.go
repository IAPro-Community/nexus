package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/nexus/maestrogates"
	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

type ComposerSessionView struct {
	Session   store.ComposerSession         `json:"session"`
	Brief     LivingBrief                   `json:"brief"`
	Turns     []store.ComposerTurn          `json:"turns"`
	Skills    []store.ComposerSkillProposal `json:"skills"`
	Artifacts []store.PromptArtifact        `json:"artifacts,omitempty"`
}

func (n *Nexus) CreateComposerSession(ctx context.Context, projectID, goal string) (*ComposerSessionView, error) {
	return n.CreateComposerSessionWithPrompt(ctx, projectID, goal, "")
}

func (n *Nexus) CreateComposerSessionWithPrompt(_ context.Context, projectID, goal, sourcePrompt string) (*ComposerSessionView, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	brief := newComposerBrief(goal, sourcePrompt)
	brief.Context.Project = project.Name
	if readiness, err := n.ObserveContextReadiness(projectID); err == nil {
		brief.Context.Evidence = appendUniqueAll(brief.Context.Evidence, []string{
			"Project: " + project.Name,
			"Branch: " + firstNonEmpty(readiness.CurrentFingerprint.Branch, "unknown"),
			"HEAD: " + firstNonEmpty(readiness.CurrentFingerprint.Head, "unknown"),
		})
	}
	refreshComposerBrief(&brief)
	briefJSON, _ := json.Marshal(brief)
	sessionState := composerSessionStateFromBrief(brief)
	session, err := st.CreateComposerSession(store.ComposerSession{
		ProjectID:          projectID,
		Title:              firstNonEmpty(brief.Intent.Objective, brief.Goal),
		State:              sessionState,
		ContextFingerprint: project.CanonicalPath,
		BriefJSON:          string(briefJSON),
	})
	if err != nil {
		return nil, err
	}
	return n.composeSessionView(st, *session, brief)
}

func (n *Nexus) GetComposerSession(_ context.Context, id string) (*ComposerSessionView, error) {
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

func (n *Nexus) ListComposerSessions(_ context.Context, projectID string) ([]store.ComposerSession, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	return st.ListComposerSessions(projectID)
}

func (n *Nexus) AddComposerTurn(_ context.Context, sessionID, role, content string) (*ComposerSessionView, error) {
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
	if _, err := st.AppendComposerTurn(store.ComposerTurn{SessionID: sessionID, Role: role, Content: content}); err != nil {
		return nil, err
	}
	brief := decodeComposerBrief(session.BriefJSON)
	if role == store.ComposerUser {
		mergeTextIntoBrief(&brief, content, "USER")
		refreshComposerBrief(&brief)
		response := composeAssistantReply(brief)
		if strings.TrimSpace(response) != "" {
			if _, err := st.AppendComposerTurn(store.ComposerTurn{SessionID: sessionID, Role: store.ComposerAssistant, Content: response}); err != nil {
				return nil, err
			}
		}
	}
	return n.persistComposerSession(st, session, brief)
}

func (n *Nexus) UpdateComposerSkillState(_ context.Context, sessionID, skillID, state string) (*ComposerSessionView, error) {
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

func (n *Nexus) FinalizeComposerSession(_ context.Context, sessionID string, selectedSkills []string, confirmGaps bool) (*store.PromptArtifact, error) {
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
	if brief.Readiness.State == PromptReadinessBlocked && !confirmGaps {
		return nil, fmt.Errorf("composer has blocking gaps; confirm gaps before finalizing")
	}
	if len(brief.OpenQuestions) > 0 && !confirmGaps {
		return nil, fmt.Errorf("composer has open questions; confirm gaps before finalizing")
	}
	skills, err := st.ListComposerSkillProposals(sessionID)
	if err != nil {
		return nil, err
	}
	validatedSkills, err := n.validateComposerSelectedSkills(selectedSkills, skills)
	if err != nil {
		return nil, err
	}
	compiled, err := compileComposerPrompt(renderCanonicalPrompt(brief), validatedSkills, n.currentMaestroStatus())
	if err != nil {
		return nil, err
	}
	contextJSON, _ := json.Marshal(map[string]any{"project_id": session.ProjectID, "brief": brief})
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

func (n *Nexus) composeSessionView(st *store.Store, session store.ComposerSession, brief LivingBrief) (*ComposerSessionView, error) {
	refreshComposerBrief(&brief)
	if session.State != store.ComposerFinalized {
		session.State = composerSessionStateFromBrief(brief)
		session.BriefJSON = mustJSON(brief)
		if err := st.UpdateComposerSession(session); err != nil {
			return nil, err
		}
	}
	turns, err := st.ListComposerTurns(session.ID, 40)
	if err != nil {
		return nil, err
	}
	skills, err := st.ListComposerSkillProposals(session.ID)
	if err != nil {
		return nil, err
	}
	skills = reconcileComposerSkillAvailability(st, session.ID, skills, n.currentMaestroStatus())
	artifacts, err := st.ListPromptArtifacts(session.ID)
	if err != nil {
		return nil, err
	}
	return &ComposerSessionView{
		Session:   session,
		Brief:     brief,
		Turns:     turns,
		Skills:    skills,
		Artifacts: artifacts,
	}, nil
}

func (n *Nexus) persistComposerSession(st *store.Store, session *store.ComposerSession, brief LivingBrief) (*ComposerSessionView, error) {
	session.Title = firstNonEmpty(brief.Intent.Objective, brief.Goal, session.Title)
	session.State = composerSessionStateFromBrief(brief)
	session.BriefJSON = mustJSON(brief)
	if err := st.UpdateComposerSession(*session); err != nil {
		return nil, err
	}
	return n.composeSessionView(st, *session, brief)
}

func composeAssistantReply(brief LivingBrief) string {
	if len(brief.OpenQuestions) > 0 {
		return "Atualizei o briefing. Próxima pergunta de maior impacto: " + brief.OpenQuestions[0]
	}
	return "Atualizei o briefing. O prompt já está consistente o bastante para finalizar."
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

func (n *Nexus) validateComposerSelectedSkills(selected []string, available []store.ComposerSkillProposal) ([]string, error) {
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
			return nil, fmt.Errorf("selected skill %s is not part of this composer session", skillID)
		}
		if item.State == store.ComposerSkillUnavailable {
			return nil, fmt.Errorf("selected skill %s is no longer available", skillID)
		}
		filtered = appendUnique(filtered, skillID)
	}
	return filtered, nil
}

func compileComposerPrompt(userPrompt string, requestedSkills []string, status MaestroStatus) (*CompiledAgentPrompt, error) {
	userPrompt = strings.TrimSpace(userPrompt)
	if userPrompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	validatedSkills := []string{}
	if len(requestedSkills) > 0 {
		catalog := []string{}
		if status.Capabilities != nil {
			catalog = status.Capabilities.SkillIDs()
		}
		var cause error
		if status.Error != "" {
			cause = fmt.Errorf("%s", status.Error)
		}
		var err error
		validatedSkills, err = maestrogates.ValidateStrict(requestedSkills, status.Available, catalog, cause)
		if err != nil {
			return nil, fmt.Errorf("skill validation failed: %w", err)
		}
	}
	return CompileAgentPrompt(userPrompt, validatedSkills, nil)
}

func reconcileComposerSkillAvailability(st *store.Store, sessionID string, skills []store.ComposerSkillProposal, status MaestroStatus) []store.ComposerSkillProposal {
	if !status.Available || status.Capabilities == nil {
		return skills
	}
	available := map[string]struct{}{}
	for _, skillID := range status.Capabilities.SkillIDs() {
		available[skillID] = struct{}{}
	}
	for i := range skills {
		if _, ok := available[skills[i].SkillID]; ok {
			continue
		}
		skills[i].State = store.ComposerSkillUnavailable
		_ = st.SetComposerSkillState(sessionID, skills[i].SkillID, store.ComposerSkillUnavailable)
	}
	return skills
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
