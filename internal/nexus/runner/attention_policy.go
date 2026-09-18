package runner

import (
	"fmt"
	"time"
)

// AttentionLevel classifies how a mission event should be surfaced to the user.
type AttentionLevel string

const (
	// AttentionIgnore means the event is internal and should not interrupt the user.
	AttentionIgnore AttentionLevel = "IGNORE"
	// AttentionInApp means the event should appear in the Attention Center but not trigger a native notification.
	AttentionInApp AttentionLevel = "IN_APP"
	// AttentionNotify means the event should appear in the Attention Center and trigger a native notification when available.
	AttentionNotify AttentionLevel = "NOTIFY"
	// AttentionRequireUser means the mission is blocked and requires a human decision to proceed.
	AttentionRequireUser AttentionLevel = "REQUIRE_USER"
)

// AttentionItem is a read-model projection of a mission event that deserves
// user attention. It is derived from the canonical MissionRun state and does
// not own business truth.
type AttentionItem struct {
	MissionID    string             `json:"mission_id"`
	ProjectID    string             `json:"project_id"`
	State        State              `json:"state"`
	Level        AttentionLevel     `json:"level"`
	ReasonCode   string             `json:"reason_code,omitempty"`
	Summary      string             `json:"summary"`
	Question     string             `json:"question,omitempty"`
	Impact       string             `json:"impact,omitempty"`
	Recommended  []string           `json:"recommended_actions,omitempty"`
	Age          int64              `json:"age_seconds"`
	Intervention *HumanIntervention `json:"intervention,omitempty"`
}

// ClassifyAttention derives the attention level for a mission run based on its
// current state and the autonomy contract. Events that Nexus can resolve
// internally (quota failover, retries, remediation) are classified as IGNORE.
// Only states that genuinely require human time produce RequireUser.
func ClassifyAttention(run *MissionRun) AttentionLevel {
	if run == nil {
		return AttentionIgnore
	}
	switch run.State {
	case StateBlockedNeedsUser:
		return AttentionRequireUser
	case StateFailedNoProgress:
		if run.Contract.EscalateOnFailure {
			return AttentionNotify
		}
		return AttentionInApp
	case StateFailedVerification, StateFailedBudgetExceeded, StateFailed:
		if run.Contract.EscalateOnFailure {
			return AttentionNotify
		}
		return AttentionInApp
	case StateCompletedVerified:
		return AttentionInApp
	case StatePaused:
		return AttentionInApp
	default:
		return AttentionIgnore
	}
}

// AttentionFromRun projects an AttentionItem from a MissionRun.
func AttentionFromRun(run *MissionRun) *AttentionItem {
	level := ClassifyAttention(run)
	if level == AttentionIgnore {
		return nil
	}
	item := &AttentionItem{
		MissionID:    run.ID,
		ProjectID:    run.ProjectID,
		State:        run.State,
		Level:        level,
		Summary:      summarizeState(run),
		Intervention: run.NeedsHuman,
	}
	if run.NeedsHuman != nil {
		item.ReasonCode = run.NeedsHuman.ReasonCode
		item.Question = run.NeedsHuman.Question
		item.Impact = run.NeedsHuman.Impact
		item.Recommended = append([]string(nil), run.NeedsHuman.RecommendedActions...)
	}
	if run.CompletedAt != nil && !run.CompletedAt.IsZero() {
		item.Age = int64(time.Since(*run.CompletedAt).Seconds())
	} else if !run.UpdatedAt.IsZero() {
		item.Age = int64(time.Since(run.UpdatedAt).Seconds())
	}
	return item
}

// AttentionDedupKey returns a stable dedup key for notification delivery. A
// blocker that persists across multiple cycles produces exactly one key.
func AttentionDedupKey(run *MissionRun) string {
	if run == nil {
		return ""
	}
	prefix := string(run.State)
	if run.NeedsHuman != nil && run.NeedsHuman.ID != "" {
		prefix += ":" + run.NeedsHuman.ID + ":v" + fmt.Sprintf("%d", run.NeedsHuman.Version)
	}
	return prefix + ":" + run.ID
}

func summarizeState(run *MissionRun) string {
	switch run.State {
	case StateBlockedNeedsUser:
		if run.NeedsHuman != nil && run.NeedsHuman.Summary != "" {
			return run.NeedsHuman.Summary
		}
		return "Mission blocked: human decision required"
	case StateFailedNoProgress:
		return "Mission failed: no progress after remediation attempts"
	case StateFailedVerification:
		return "Mission failed: verification did not pass"
	case StateFailedBudgetExceeded:
		return "Mission failed: iteration budget exceeded"
	case StateFailed:
		return "Mission failed"
	case StateCompletedVerified:
		return "Mission completed with verification"
	case StatePaused:
		if run.PausedReason != "" {
			return "Mission paused: " + run.PausedReason
		}
		return "Mission paused"
	default:
		return "Mission in state " + string(run.State)
	}
}
