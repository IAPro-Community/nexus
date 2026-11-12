package store

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

// ListAgentGroups returns groups and ordered persistent Agent identities for a project.
func (s *Store) ListAgentGroups(projectID string) ([]AgentGroup, error) {
	rows, err := s.db.Query(`SELECT id,project_id,name,task,worktree,feature,position,created_at,updated_at
		FROM agent_groups WHERE project_id=? ORDER BY position,created_at,id`, projectID)
	if err != nil {
		return nil, err
	}
	groups := make([]AgentGroup, 0)
	for rows.Next() {
		var group AgentGroup
		var createdAt, updatedAt string
		if err := rows.Scan(&group.ID, &group.ProjectID, &group.Name, &group.Task, &group.Worktree, &group.Feature,
			&group.Position, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		group.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		group.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		group.AgentIDs = make([]string, 0)
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range groups {
		members, err := s.db.Query(`SELECT m.agent_id FROM agent_group_members m
			JOIN agents a ON a.id=m.agent_id
			WHERE m.group_id=? AND a.project_id=? AND COALESCE(a.identity_kind,'PERSISTENT')!='EPHEMERAL_WORKER'
			ORDER BY m.position,m.agent_id`, groups[i].ID, projectID)
		if err != nil {
			return nil, err
		}
		for members.Next() {
			var id string
			if err := members.Scan(&id); err != nil {
				members.Close()
				return nil, err
			}
			groups[i].AgentIDs = append(groups[i].AgentIDs, id)
		}
		if err := members.Err(); err != nil {
			members.Close()
			return nil, err
		}
		members.Close()
	}
	return groups, nil
}

// CreateAgentGroup inserts an empty, durable project group. Membership is added atomically by UpdateAgentGroup.
func (s *Store) CreateAgentGroup(projectID, name, task, worktree, feature string) (AgentGroup, error) {
	if _, err := s.GetProject(projectID); err != nil {
		return AgentGroup{}, fmt.Errorf("agent group project must exist: %w", err)
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return AgentGroup{}, errors.New("group name must contain 1 to 120 characters")
	}
	now := time.Now().UTC()
	id := "agp_" + ids.NewRuntimeID()
	var position int
	if err := s.db.QueryRow(`SELECT COALESCE(MAX(position)+1,0) FROM agent_groups WHERE project_id=?`, projectID).Scan(&position); err != nil {
		return AgentGroup{}, err
	}
	_, err := s.db.Exec(`INSERT INTO agent_groups(id,project_id,name,task,worktree,feature,position,created_at,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?)`, id, projectID, name, strings.TrimSpace(task), strings.TrimSpace(worktree),
		strings.TrimSpace(feature), position, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return AgentGroup{}, fmt.Errorf("create agent group: %w", err)
	}
	return AgentGroup{ID: id, ProjectID: projectID, Name: name, Task: strings.TrimSpace(task),
		Worktree: strings.TrimSpace(worktree), Feature: strings.TrimSpace(feature), Position: position,
		AgentIDs: []string{}, CreatedAt: now, UpdatedAt: now}, nil
}

// UpdateAgentGroup atomically replaces group metadata and its ordered Agent membership.
// An Agent can belong to at most one group; moving it clears its previous membership in this transaction.
func (s *Store) UpdateAgentGroup(projectID string, group AgentGroup) (AgentGroup, error) {
	group.Name = strings.TrimSpace(group.Name)
	group.Task = strings.TrimSpace(group.Task)
	group.Worktree = strings.TrimSpace(group.Worktree)
	group.Feature = strings.TrimSpace(group.Feature)
	if group.Name == "" || len(group.Name) > 120 {
		return AgentGroup{}, errors.New("group name must contain 1 to 120 characters")
	}
	if len(group.AgentIDs) > 500 {
		return AgentGroup{}, errors.New("group cannot contain more than 500 agents")
	}
	seen := make(map[string]struct{}, len(group.AgentIDs))
	for _, id := range group.AgentIDs {
		if strings.TrimSpace(id) == "" {
			return AgentGroup{}, errors.New("agent id is required")
		}
		if _, ok := seen[id]; ok {
			return AgentGroup{}, fmt.Errorf("duplicate agent id %q", id)
		}
		seen[id] = struct{}{}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return AgentGroup{}, err
	}
	defer tx.Rollback()
	var createdAt string
	if err := tx.QueryRow(`SELECT id,project_id,created_at FROM agent_groups WHERE id=? AND project_id=?`,
		group.ID, projectID).Scan(&group.ID, &group.ProjectID, &createdAt); err != nil {
		return AgentGroup{}, ErrNotFound
	}
	group.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	for _, id := range group.AgentIDs {
		var count int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM agents WHERE id=? AND project_id=? AND COALESCE(identity_kind,'PERSISTENT')!='EPHEMERAL_WORKER'`,
			id, projectID).Scan(&count); err != nil {
			return AgentGroup{}, err
		}
		if count != 1 {
			return AgentGroup{}, fmt.Errorf("agent %q does not belong to project or is not persistent", id)
		}
	}
	now := time.Now().UTC()
	group.UpdatedAt = now
	if _, err := tx.Exec(`UPDATE agent_groups SET name=?,task=?,worktree=?,feature=?,updated_at=? WHERE id=? AND project_id=?`,
		group.Name, group.Task, group.Worktree, group.Feature, now.Format(time.RFC3339Nano), group.ID, projectID); err != nil {
		return AgentGroup{}, err
	}
	// Explicitly remove replaced and moved memberships before inserting the new order.
	if _, err := tx.Exec(`DELETE FROM agent_group_members WHERE group_id=? OR agent_id IN (
		SELECT agent_id FROM agent_group_members WHERE group_id=?)`, group.ID, group.ID); err != nil {
		return AgentGroup{}, err
	}
	for pos, id := range group.AgentIDs {
		if _, err := tx.Exec(`DELETE FROM agent_group_members WHERE agent_id=?`, id); err != nil {
			return AgentGroup{}, err
		}
		if _, err := tx.Exec(`INSERT INTO agent_group_members(agent_id,group_id,position,created_at,updated_at) VALUES(?,?,?,?,?)`,
			id, group.ID, pos, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
			return AgentGroup{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return AgentGroup{}, err
	}
	group.AgentIDs = append([]string{}, group.AgentIDs...)
	return group, nil
}

// DeleteAgentGroup removes a project-scoped group while preserving all Agents and sessions.
func (s *Store) DeleteAgentGroup(projectID, groupID string) error {
	result, err := s.db.Exec(`DELETE FROM agent_groups WHERE id=? AND project_id=?`, groupID, projectID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}
