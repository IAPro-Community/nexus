package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// NexConversationTurn is operational chat history, not semantic project memory.
// A paired remote Nexus client is the same local host owner; this store never
// copies conversation content to Maestro or a third-party persistence service.
type NexConversationTurn struct {
	Sequence  int64     `json:"sequence"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Source    string    `json:"source,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

const nexConversationHistoryLimit = 60

func nexConversationScope(projectID string) string {
	if projectID == "" {
		return "@global"
	}
	return "project:" + projectID
}

// ListNexConversation returns only the most recent bounded turns, chronologically.
func (s *Store) ListNexConversation(projectID string) ([]NexConversationTurn, error) {
	rows, err := s.db.Query(`SELECT seq,role,content,source,created_at
		FROM nex_conversation_turns WHERE project_scope=?
		ORDER BY seq DESC LIMIT ?`, nexConversationScope(projectID), nexConversationHistoryLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]NexConversationTurn, 0)
	for rows.Next() {
		var turn NexConversationTurn
		var created string
		if err := rows.Scan(&turn.Sequence, &turn.Role, &turn.Content, &turn.Source, &created); err != nil {
			return nil, err
		}
		turn.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, fmt.Errorf("invalid Nex conversation timestamp: %w", err)
		}
		out = append(out, turn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// FindNexConversationReply makes a retry with the same request ID idempotent.
// A caller must never replace or reinterpret an already completed exchange.
func (s *Store) FindNexConversationReply(projectID, requestID, message string) (string, bool, error) {
	var original string
	err := s.db.QueryRow(`SELECT content FROM nex_conversation_turns
		WHERE project_scope=? AND request_id=? AND role='user'`,
		nexConversationScope(projectID), requestID).Scan(&original)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if original != message {
		return "", false, fmt.Errorf("Nex request ID conflict")
	}
	var reply string
	err = s.db.QueryRow(`SELECT content FROM nex_conversation_turns
		WHERE project_scope=? AND request_id=? AND role='assistant'`,
		nexConversationScope(projectID), requestID).Scan(&reply)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return reply, true, nil
}

// StoreNexConversationExchange commits both turns atomically after the provider
// returns. Failed/cancelled requests do not leave a misleading partial history.
// requestID deduplicates client retries after uncertain network completion.
func (s *Store) StoreNexConversationExchange(projectID, requestID, user, reply, source string) (string, error) {
	if len(requestID) < 8 || len(requestID) > 128 || strings.TrimSpace(requestID) != requestID {
		return "", fmt.Errorf("invalid Nex request ID")
	}
	if user == "" || len(user) > 8000 || reply == "" || len(reply) > 8000 {
		return "", fmt.Errorf("invalid Nex message size")
	}
	if source != "nexus_snapshot" && source != "configured_model" {
		return "", fmt.Errorf("invalid Nex source")
	}
	scope := nexConversationScope(projectID)
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var previousUser, previousReply string
	err = tx.QueryRow(`SELECT content FROM nex_conversation_turns WHERE project_scope=?
		AND request_id=? AND role='user'`, scope, requestID).Scan(&previousUser)
	if err == nil {
		if previousUser != user {
			return "", fmt.Errorf("Nex request ID conflict")
		}
		if err := tx.QueryRow(`SELECT content FROM nex_conversation_turns WHERE project_scope=?
			AND request_id=? AND role='assistant'`, scope, requestID).Scan(&previousReply); err != nil {
			return "", fmt.Errorf("incomplete Nex exchange: %w", err)
		}
		return previousReply, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	var linkedProject any
	if projectID != "" {
		var count int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM projects WHERE id=?`, projectID).Scan(&count); err != nil {
			return "", err
		}
		if count != 1 {
			return "", ErrNotFound
		}
		linkedProject = projectID
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, turn := range []struct{ role, content, source string }{
		{"user", user, ""},
		{"assistant", reply, source},
	} {
		if _, err := tx.Exec(`INSERT INTO nex_conversation_turns
			(project_scope,project_id,request_id,role,content,source,created_at)
			VALUES(?,?,?,?,?,?,?)`, scope, linkedProject, requestID,
			turn.role, turn.content, turn.source, now); err != nil {
			return "", err
		}
	}
	// Retain a maximum of 160 turns per scope; no unbounded transcripts.
	if _, err := tx.Exec(`DELETE FROM nex_conversation_turns WHERE seq IN (
		SELECT seq FROM nex_conversation_turns WHERE project_scope=?
		ORDER BY seq DESC LIMIT -1 OFFSET 160)`, scope); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return reply, nil
}

// DeleteNexConversation permanently removes operational dialogue for one scope.
func (s *Store) DeleteNexConversation(projectID string) error {
	_, err := s.db.Exec(`DELETE FROM nex_conversation_turns WHERE project_scope=?`,
		nexConversationScope(projectID))
	return err
}
