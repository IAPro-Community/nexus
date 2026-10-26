package nexus

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// ComposerApplicationService is the transport-facing Composer contract.
// Composer rules remain implemented by Nexus until their intelligence and
// Maestro dependencies can be split without duplicating the workflow.
type ComposerApplicationService struct {
	nexus *Nexus
}

func NewComposerApplicationService(n *Nexus) *ComposerApplicationService {
	return &ComposerApplicationService{nexus: n}
}

// requireIntelligence guards user-facing Composer mutations. A configured
// provider must be resolvable and available; heuristic-only elaboration is
// not a substitute for an AI planning assistant. Direct/Agent remain independent.
func (s *ComposerApplicationService) requireIntelligence(ctx context.Context, projectID string) error {
	provider, err := s.nexus.ConfiguredIntelligenceProvider(ctx, projectID)
	if err != nil {
		return fmt.Errorf("Composer requer IA configurada em Configurações > Nexus Intelligence: %w", err)
	}
	if provider == nil || !provider.Available(ctx) {
		return fmt.Errorf("Composer requer um provedor de IA disponível em Configurações > Nexus Intelligence")
	}
	return nil
}

func (s *ComposerApplicationService) requireSessionIntelligence(ctx context.Context, sessionID string) error {
	view, err := s.nexus.GetComposerSession(ctx, sessionID)
	if err != nil {
		return err
	}
	return s.requireIntelligence(ctx, view.Session.ProjectID)
}

func (s *ComposerApplicationService) Create(ctx context.Context, projectID, goal, sourcePrompt, inputMode string) (*ComposerSessionView, error) {
	return s.CreateWithProgress(ctx, projectID, goal, sourcePrompt, inputMode, nil)
}

func (s *ComposerApplicationService) CreateWithProgress(ctx context.Context, projectID, goal, sourcePrompt, inputMode string, sink ComposerProgressSink) (*ComposerSessionView, error) {
	return s.CreateWithInterviewModeProgress(ctx, projectID, goal, sourcePrompt, inputMode, string(ComposerInterviewStandard), sink)
}

func (s *ComposerApplicationService) CreateWithInterviewMode(
	ctx context.Context,
	projectID, goal, sourcePrompt, inputMode, interviewMode string,
) (*ComposerSessionView, error) {
	return s.CreateWithInterviewModeProgress(ctx, projectID, goal, sourcePrompt, inputMode, interviewMode, nil)
}

func (s *ComposerApplicationService) CreateWithInterviewModeProgress(
	ctx context.Context,
	projectID, goal, sourcePrompt, inputMode, interviewMode string,
	sink ComposerProgressSink,
) (*ComposerSessionView, error) {
	if err := s.requireIntelligence(ctx, projectID); err != nil {
		return nil, err
	}
	// A configured CLI or API endpoint may still be unauthenticated/offline.
	// Validate a bounded real model round-trip before persisting an elaboration.
	if _, err := s.nexus.ProbeIntelligence(ctx, projectID); err != nil {
		// Provider errors may contain raw CLI output, filesystem paths or remote
		// API diagnostics. Do not leak that material through an HTTP response.
		return nil, fmt.Errorf("A IA configurada não passou no teste de conexão do Composer. Verifique provedor, perfil, credenciais e conectividade em Nexus Intelligence")
	}
	view, err := s.nexus.CreateComposerSessionWithPromptInterviewModeProgress(
		ctx,
		projectID,
		goal,
		sourcePrompt,
		ComposerInterviewMode(interviewMode),
		sink,
	)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(strings.TrimSpace(inputMode), "EXISTING_PROMPT") && strings.TrimSpace(sourcePrompt) != "" {
		// Ensure review is computed against the imported prompt body.
		view.PromptReview = ReviewComposerPrompt(view.Brief)
	}
	return view, nil
}

func (s *ComposerApplicationService) List(ctx context.Context, projectID string) ([]store.ComposerSession, error) {
	return s.nexus.ListComposerSessions(ctx, projectID)
}

// Delete is a user-owned cleanup operation. It must remain available even
// when Nexus Intelligence is offline; creation/refinement still require AI.
func (s *ComposerApplicationService) Delete(ctx context.Context, sessionID string, expectedRevision int) error {
	if err := ctx.Err(); err != nil { return err }
	st, err := s.nexus.OpenProject()
	if err != nil { return err }
	return st.DeleteComposerSessionExpected(sessionID, expectedRevision)
}

func (s *ComposerApplicationService) Get(ctx context.Context, sessionID string) (*ComposerSessionView, error) {
	return s.nexus.GetComposerSession(ctx, sessionID)
}

func (s *ComposerApplicationService) AddTurn(ctx context.Context, sessionID, content string, expectedRevision int, attachments []ComposerAttachmentRef) (*ComposerSessionView, error) {
	if err := s.requireSessionIntelligence(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.nexus.AddComposerTurnExpected(ctx, sessionID, store.ComposerUser, content, expectedRevision, attachments)
}

func (s *ComposerApplicationService) UpdateSkill(ctx context.Context, sessionID, skillID, state string) (*ComposerSessionView, error) {
	if err := s.requireSessionIntelligence(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.nexus.UpdateComposerSkillState(ctx, sessionID, skillID, state)
}

func (s *ComposerApplicationService) SkillCatalog(ctx context.Context, projectID, query string) ([]ComposerSkillCatalogItem, error) {
	return s.nexus.ListComposerSkillCatalog(ctx, projectID, query)
}

func (s *ComposerApplicationService) SelectSkill(ctx context.Context, sessionID, skillID string) (*ComposerSessionView, error) {
	if err := s.requireSessionIntelligence(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.nexus.SelectComposerSkill(ctx, sessionID, skillID)
}

func (s *ComposerApplicationService) Finalize(ctx context.Context, sessionID string, selectedSkills []string, confirmGaps bool) (*store.PromptArtifact, error) {
	if err := s.requireSessionIntelligence(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.nexus.FinalizeComposerSession(ctx, sessionID, selectedSkills, confirmGaps)
}

func (s *ComposerApplicationService) Refine(ctx context.Context, sessionID, refinementGoal string) (*store.PromptArtifact, error) {
	if err := s.requireSessionIntelligence(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.nexus.RefineComposerArtifact(ctx, sessionID, refinementGoal)
}

func (s *ComposerApplicationService) ResolveUnknown(ctx context.Context, sessionID, unknownID, answer, status string, expectedRevision int) (*ComposerSessionView, error) {
	if err := s.requireSessionIntelligence(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.nexus.ResolveComposerUnknownExpected(ctx, sessionID, unknownID, answer, status, expectedRevision)
}

func (s *ComposerApplicationService) AttachContextFile(ctx context.Context, sessionID, relPath string, expectedRevision int) (*ComposerSessionView, error) {
	if err := s.requireSessionIntelligence(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.nexus.AttachComposerContextFile(ctx, sessionID, relPath, expectedRevision)
}

func (s *ComposerApplicationService) DetachContextFile(ctx context.Context, sessionID, relPath string, expectedRevision int) (*ComposerSessionView, error) {
	if err := s.requireSessionIntelligence(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.nexus.DetachComposerContextFile(ctx, sessionID, relPath, expectedRevision)
}
