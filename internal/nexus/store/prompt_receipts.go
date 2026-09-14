package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
)

type AgentPromptReceipt struct {
	ID         string    `json:"id"`
	AgentID    string    `json:"agent_id"`
	RuntimeID  string    `json:"runtime_id"`
	SkillIDs   []string  `json:"skill_ids"`
	PromptHash string    `json:"prompt_hash"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Store) RecordAgentPromptReceipt(r AgentPromptReceipt) (*AgentPromptReceipt, error) {
	if r.ID == "" {
		r.ID = "receipt_" + ids.NewRuntimeID()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	skillJSON, err := json.Marshal(r.SkillIDs)
	if err != nil {
		skillJSON = []byte("[]")
	}
	_, err = s.db.Exec(
		`INSERT INTO agent_prompt_receipts(id, agent_id, runtime_id, skill_ids, prompt_hash, source, created_at)
		 VALUES(?,?,?,?,?,?,?)`,
		r.ID, r.AgentID, r.RuntimeID, string(skillJSON), r.PromptHash, r.Source, r.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, fmt.Errorf("record prompt receipt: %w", err)
	}
	return &r, nil
}

func (s *Store) ListAgentPromptReceipts(agentID string) ([]AgentPromptReceipt, error) {
	rows, err := s.db.Query(
		`SELECT id, agent_id, runtime_id, skill_ids, prompt_hash, source, created_at
		 FROM agent_prompt_receipts WHERE agent_id=? ORDER BY created_at DESC`,
		agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list prompt receipts: %w", err)
	}
	defer rows.Close()

	var out []AgentPromptReceipt
	for rows.Next() {
		var r AgentPromptReceipt
		var rawSkills, createdAtStr string
		if err := rows.Scan(&r.ID, &r.AgentID, &r.RuntimeID, &rawSkills, &r.PromptHash, &r.Source, &createdAtStr); err != nil {
			return nil, err
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		if strings.TrimSpace(rawSkills) != "" {
			_ = json.Unmarshal([]byte(rawSkills), &r.SkillIDs)
		}
		if r.SkillIDs == nil {
			r.SkillIDs = []string{}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
