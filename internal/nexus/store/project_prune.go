package store

import (
	"context"
	"strings"
)

// ProjectPruneCandidate is a project row that carries no user evidence: it has
// no agents, no saved workspace layout, no events and was never opened from the
// UI (last_opened_at is NULL). Rows like these were produced by CLI invocations
// that registered their working directory implicitly.
type ProjectPruneCandidate struct {
	Project
	AgentCount  int    `json:"agent_count"`
	LayoutSaved bool   `json:"layout_saved"`
	EventCount  int    `json:"event_count"`
	Reason      string `json:"reason"`
}

// ListProjectPruneCandidates lists projects that look like implicit CLI
// registrations. Read-only: the caller decides whether to delete.
func (s *Store) ListProjectPruneCandidates(ctx context.Context) ([]ProjectPruneCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	projects, err := s.ListProjects()
	if err != nil {
		return nil, err
	}
	out := make([]ProjectPruneCandidate, 0, len(projects))
	for _, project := range projects {
		candidate, err := s.pruneCandidate(ctx, project)
		if err != nil {
			return nil, err
		}
		out = append(out, candidate)
	}
	return out, nil
}

func (s *Store) pruneCandidate(ctx context.Context, project Project) (ProjectPruneCandidate, error) {
	candidate := ProjectPruneCandidate{Project: project}
	if err := ctx.Err(); err != nil {
		return candidate, err
	}
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM agents WHERE project_id=?`, project.ID,
	).Scan(&candidate.AgentCount); err != nil {
		return candidate, err
	}
	var layout string
	err := s.db.QueryRow(
		`SELECT layout FROM project_layouts WHERE project_id=?`, project.ID,
	).Scan(&layout)
	switch {
	case err != nil:
		// No layout row at all: nothing a user configured in the shell.
		candidate.LayoutSaved = false
	default:
		trimmed := strings.TrimSpace(layout)
		candidate.LayoutSaved = trimmed != "" && trimmed != "{}"
	}
	// Events reference their project directly, or their agent (legacy rows).
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM events_metadata WHERE COALESCE(NULLIF(project_id,''),'')=?
		   OR agent_id IN (SELECT id FROM agents WHERE project_id=?)`,
		project.ID, project.ID,
	).Scan(&candidate.EventCount); err != nil {
		return candidate, err
	}
	reasons := make([]string, 0, 4)
	if project.LastOpenedAt == nil {
		reasons = append(reasons, "never opened")
	}
	if candidate.AgentCount == 0 {
		reasons = append(reasons, "no agents")
	}
	if !candidate.LayoutSaved {
		reasons = append(reasons, "no saved layout")
	}
	if candidate.EventCount == 0 {
		reasons = append(reasons, "no events")
	}
	if len(reasons) == 4 {
		candidate.Reason = strings.Join(reasons, ", ")
	}
	return candidate, nil
}

// DeleteProjectRow removes a single project row. It is the transactional
// primitive used by pruning; cascade semantics mirror DeleteProject.
func (s *Store) DeleteProjectRow(ctx context.Context, projectID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM projects WHERE id=?`, projectID)
	return err
}
