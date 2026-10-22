package nexus

import (
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

// HUDRuntimeAttention is a privacy-conscious read projection of the canonical
// runtime registry. It deliberately excludes terminal output, prompts, tokens
// and command text. The user opens the originating terminal to act.
type HUDRuntimeAttention struct {
	RuntimeID   string    `json:"runtime_id"`
	AgentID     string    `json:"agent_id,omitempty"`
	ProjectID   string    `json:"project_id,omitempty"`
	ProjectName string    `json:"project_name,omitempty"`
	Provider    string    `json:"provider"`
	Reason      string    `json:"reason"`
	PromptKind  string    `json:"prompt_kind"`
	Fingerprint string    `json:"fingerprint"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BuildHUDRuntimeAttention consumes only already-classified attention from
// SessionHost/structured provider adapters. It does not regex terminal data,
// start a provider probe or invent an interaction from the WAITING state alone.
func BuildHUDRuntimeAttention(sessions []registry.RuntimeSession, projectID string) []HUDRuntimeAttention {
	items := make([]HUDRuntimeAttention, 0)
	projectID = strings.TrimSpace(projectID)
	for _, session := range sessions {
		if projectID != "" && session.ProjectID != projectID {
			continue
		}
		if session.State != registry.StateWaiting && session.State != registry.StateApproval {
			continue
		}
		if strings.TrimSpace(session.AttentionContext) == "" {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(session.PromptKind))
		if kind != "yn" && kind != "choice" && kind != "free_text" {
			continue
		}
		reason := strings.ToUpper(strings.TrimSpace(session.AttentionReason))
		if session.AttentionKind != "needs_user" && reason != "QUESTION" && reason != "APPROVAL" {
			continue
		}
		if session.RuntimeID == "" || !session.HostLive() {
			continue
		}
		fp := strings.TrimSpace(session.AttentionFingerprint)
		if fp == "" {
			// Legacy records still need a per-runtime stable identity, but no
			// plaintext terminal question must leak into the HUD projection.
			fp = session.RuntimeID + ":" + kind + ":" + reason
		}
		items = append(items, HUDRuntimeAttention{
			RuntimeID: session.RuntimeID, AgentID: session.AgentID,
			ProjectID: session.ProjectID, ProjectName: session.ProjectName,
			Provider: session.ProviderID, Reason: reason, PromptKind: kind,
			Fingerprint: fp, UpdatedAt: session.UpdatedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].RuntimeID < items[j].RuntimeID
	})
	if len(items) > 50 {
		items = items[:50]
	}
	return items
}
