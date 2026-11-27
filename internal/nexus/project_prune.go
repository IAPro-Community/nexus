package nexus

import (
	"context"
	"fmt"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// ProjectPruneResult reports an explicit-prune run. Pruning never removes a
// project that still has user evidence: an agent, a saved layout, events, or
// having been opened from the UI (last_opened_at set).
type ProjectPruneResult struct {
	IncludeOpen bool                          `json:"include_open"`
	Candidates  []store.ProjectPruneCandidate `json:"candidates"`
	Removed     []store.ProjectPruneCandidate `json:"removed"`
	Skipped     []store.ProjectPruneCandidate `json:"skipped"`
}

// ListProjectsToPrune returns the read-only candidate list used by the dry run.
// The CLI only deletes after an explicit --apply.
func (s *ProjectApplicationService) ListProjectsToPrune(ctx context.Context) ([]store.ProjectPruneCandidate, error) {
	st, err := s.store(ctx)
	if err != nil {
		return nil, err
	}
	return st.ListProjectPruneCandidates(ctx)
}

// PruneProjects deletes the candidate rows. It is the single write path for
// pruning: every row is re-verified against the same evidence rules before the
// delete, so a concurrent agent/layout/event creation is never destroyed.
func (s *ProjectApplicationService) PruneProjects(ctx context.Context, includeOpened bool) (ProjectPruneResult, error) {
	if err := ctx.Err(); err != nil {
		return ProjectPruneResult{}, err
	}
	st, err := s.store(ctx)
	if err != nil {
		return ProjectPruneResult{}, err
	}
	candidates, err := st.ListProjectPruneCandidates(ctx)
	if err != nil {
		return ProjectPruneResult{}, err
	}
	result := ProjectPruneResult{
		IncludeOpen: includeOpened,
		Candidates:  candidates,
		Removed:     []store.ProjectPruneCandidate{},
		Skipped:     []store.ProjectPruneCandidate{},
	}
	for _, candidate := range candidates {
		evident := candidate.AgentCount > 0 ||
			candidate.LayoutSaved ||
			candidate.EventCount > 0 ||
			(candidate.LastOpenedAt != nil && !includeOpened)
		if evident {
			result.Skipped = append(result.Skipped, candidate)
			continue
		}
		if err := st.DeleteProjectRow(ctx, candidate.ID); err != nil {
			return result, fmt.Errorf("prune project %s: %w", candidate.ID, err)
		}
		result.Removed = append(result.Removed, candidate)
	}
	return result, nil
}
