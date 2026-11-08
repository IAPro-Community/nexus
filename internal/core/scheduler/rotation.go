package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

const rotationStateFile = "account-rotation.json"

type rotationState struct {
	LastSelected map[string]string `json:"last_selected"`
}

// SelectNextBalancedProfile rotates deterministically through eligible accounts
// and persists the cursor under a cross-process lock. EvaluateAll remains the
// authority for authentication, health, quota, cooldown and disabled gates.
func (s *Selector) SelectNextBalancedProfile(
	ctx context.Context,
	provider string,
	workspace string,
	candidates []model.Profile,
	accounts map[string]model.AccountInfo,
	excludeProfiles []string,
) (*SelectionResult, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	evaluations := s.EvaluateAll(provider, workspace, candidates, accounts, excludeProfiles)
	eligible := make([]model.Profile, 0, len(evaluations))
	var rejected []string
	for i := range evaluations {
		evaluation := &evaluations[i]
		if provider == string(model.ProviderAGY) && evaluation.Eligible {
			if reason := agyBalancedQuotaEvidenceReason(evaluation.Usage, evaluation.Profile.Name, time.Now()); reason != "" {
				evaluation.Eligible = false
				evaluation.RejectReason = reason
			}
		}
		if evaluation.Eligible {
			eligible = append(eligible, evaluation.Profile)
		} else {
			rejected = append(rejected, fmt.Sprintf("%s (%s)", evaluation.Profile.Name, evaluation.RejectReason))
		}
	}
	if len(eligible) == 0 {
		message := fmt.Sprintf("no eligible %s profiles for balanced rotation", provider)
		if len(rejected) > 0 {
			message += ": " + strings.Join(rejected, ", ")
		}
		return &SelectionResult{Evaluations: evaluations}, errors.New(message)
	}
	if err := ctx.Err(); err != nil {
		return &SelectionResult{Evaluations: evaluations}, err
	}
	sort.Slice(eligible, func(i, j int) bool { return eligible[i].Name < eligible[j].Name })

	stateDir, err := config.StateDir()
	if err != nil {
		return &SelectionResult{Evaluations: evaluations}, fmt.Errorf("resolve account rotation state: %w", err)
	}
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return &SelectionResult{Evaluations: evaluations}, fmt.Errorf("create account rotation state: %w", err)
	}
	statePath := filepath.Join(stateDir, rotationStateFile)
	unlock, err := acquireRotationLock(statePath + ".lock")
	if err != nil {
		return &SelectionResult{Evaluations: evaluations}, err
	}
	defer func() { _ = unlock() }()

	state := rotationState{LastSelected: make(map[string]string)}
	if data, readErr := os.ReadFile(statePath); readErr == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return &SelectionResult{Evaluations: evaluations}, fmt.Errorf("decode account rotation state: %w", err)
		}
		if state.LastSelected == nil {
			state.LastSelected = make(map[string]string)
		}
	} else if !os.IsNotExist(readErr) {
		return &SelectionResult{Evaluations: evaluations}, fmt.Errorf("read account rotation state: %w", readErr)
	}

	last := state.LastSelected[provider]
	selectedIndex := 0
	for i, candidate := range eligible {
		if candidate.Name == last {
			selectedIndex = (i + 1) % len(eligible)
			break
		}
	}
	selected := eligible[selectedIndex]
	state.LastSelected[provider] = selected.Name
	if err := writeRotationState(statePath, state); err != nil {
		return &SelectionResult{Evaluations: evaluations}, err
	}
	return &SelectionResult{
		SelectedProfile: &selected,
		Reason:          "balanced rotation to next eligible account",
		Evaluations:     evaluations,
	}, nil
}

// agyBalancedQuotaEvidenceReason makes 5-hour quota proof a hard precondition
// for AGY automatic rotation. The generic quota gate establishes that the
// account is operationally available; this gate prevents weekly-only or
// unattributed snapshots from consuming an account whose short-window state
// is unknown.
func agyBalancedQuotaEvidenceReason(snapshot model.UsageSnapshot, profile string, now time.Time) string {
	if !strings.EqualFold(strings.TrimSpace(snapshot.ProviderID), string(model.ProviderAGY)) ||
		strings.TrimSpace(snapshot.ProfileID) != profile {
		return "AGY quota evidence is not attributable to this profile"
	}
	if snapshot.Status != model.UsageLive && snapshot.Status != model.UsageCached {
		return "AGY 5h quota requires live or recently cached evidence"
	}
	if snapshot.Source == model.SourceNone {
		return "AGY quota evidence source is unknown"
	}
	if snapshot.FetchedAt.IsZero() || now.Sub(snapshot.FetchedAt) < -time.Minute || now.Sub(snapshot.FetchedAt) > quota.DefaultTTL {
		return "AGY 5h quota evidence is missing or stale"
	}

	groups := make(map[string]bool)
	fiveHour := make(map[string]bool)
	for _, window := range snapshot.Windows {
		group, ok := agyQuotaGroup(window.Group, window.Kind)
		if !ok {
			return "AGY quota evidence contains an unrecognized model group"
		}
		groups[group] = true
		if agyFiveHourKind(group, window.Kind) && validAGYRemainingPercent(window.RemainingPercent) {
			fiveHour[group] = true
		}
	}
	if len(groups) == 0 {
		return "AGY 5h quota evidence is missing"
	}
	for group := range groups {
		if !fiveHour[group] {
			return fmt.Sprintf("AGY 5h quota evidence is missing for model group %s", group)
		}
	}
	return ""
}

func agyQuotaGroup(rawGroup, rawKind string) (string, bool) {
	group := strings.ToLower(strings.TrimSpace(rawGroup))
	switch {
	case group == "gemini" || strings.Contains(group, "gemini"):
		return "gemini", true
	case group == "claude_gpt" || strings.Contains(group, "claude") || strings.Contains(group, "gpt"):
		return "claude_gpt", true
	case group != "":
		return "", false
	}
	kind := strings.ToLower(strings.TrimSpace(rawKind))
	if kind == "claude_5h" || kind == "claude_five_hour" || kind == "claude_weekly" {
		return "claude_gpt", true
	}
	if kind == "5h" || kind == "five_hour" || kind == "weekly" {
		return "gemini", true
	}
	return "", false
}

func agyFiveHourKind(group, rawKind string) bool {
	kind := strings.ToLower(strings.TrimSpace(rawKind))
	if group == "claude_gpt" {
		return kind == "claude_5h" || kind == "claude_five_hour"
	}
	return kind == "5h" || kind == "five_hour"
}

func validAGYRemainingPercent(value *float64) bool {
	return value != nil && !math.IsNaN(*value) && !math.IsInf(*value, 0) && *value >= 0 && *value <= 100
}

func writeRotationState(path string, state rotationState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode account rotation state: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".account-rotation-*.tmp")
	if err != nil {
		return fmt.Errorf("create account rotation state temp file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("secure account rotation state: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write account rotation state: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync account rotation state: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close account rotation state: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace account rotation state: %w", err)
	}
	return nil
}
