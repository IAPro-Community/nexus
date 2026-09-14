package nexus

import (
	"context"

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

func (s *ComposerApplicationService) Create(ctx context.Context, projectID, goal, sourcePrompt string) (*ComposerSessionView, error) {
	return s.nexus.CreateComposerSessionWithPrompt(ctx, projectID, goal, sourcePrompt)
}

func (s *ComposerApplicationService) List(ctx context.Context, projectID string) ([]store.ComposerSession, error) {
	return s.nexus.ListComposerSessions(ctx, projectID)
}

func (s *ComposerApplicationService) Get(ctx context.Context, sessionID string) (*ComposerSessionView, error) {
	return s.nexus.GetComposerSession(ctx, sessionID)
}

func (s *ComposerApplicationService) AddTurn(ctx context.Context, sessionID, content string, expectedRevision int) (*ComposerSessionView, error) {
	return s.nexus.AddComposerTurnExpected(ctx, sessionID, store.ComposerUser, content, expectedRevision)
}

func (s *ComposerApplicationService) UpdateSkill(ctx context.Context, sessionID, skillID, state string) (*ComposerSessionView, error) {
	return s.nexus.UpdateComposerSkillState(ctx, sessionID, skillID, state)
}

func (s *ComposerApplicationService) Finalize(ctx context.Context, sessionID string, selectedSkills []string, confirmGaps bool) (*store.PromptArtifact, error) {
	return s.nexus.FinalizeComposerSession(ctx, sessionID, selectedSkills, confirmGaps)
}

func (s *ComposerApplicationService) Refine(ctx context.Context, sessionID, refinementGoal string) (*store.PromptArtifact, error) {
	return s.nexus.RefineComposerArtifact(ctx, sessionID, refinementGoal)
}

func (s *ComposerApplicationService) ResolveUnknown(ctx context.Context, sessionID, unknownID, answer, status string, expectedRevision int) (*ComposerSessionView, error) {
	return s.nexus.ResolveComposerUnknownExpected(ctx, sessionID, unknownID, answer, status, expectedRevision)
}
