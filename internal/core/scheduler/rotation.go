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

// balanceCandidate pairs an eligible evaluation with its measured quota
// availability so the rotation can rank accounts by remaining balance.
// measured=false means the provider exposes no trustworthy capacity metric
// (OPAQUE); such accounts never outrank measured positive quota.
type balanceCandidate struct {
	evaluation *CandidateEvaluation
	measured   bool
	remaining  float64
}

// SelectNextBalancedProfile selects the eligible account with the most
// measured quota remaining and persists the cursor under a cross-process
// lock. Accounts tied at the same availability rotate fairly through the
// cursor, so equal accounts still share usage deterministically.
// EvaluateAll remains the authority for authentication, health, quota,
// cooldown and disabled gates.
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
	eligible := make([]balanceCandidate, 0, len(evaluations))
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
			view := quota.BuildQuotaView(evaluation.Usage, evaluation.Account.Email, evaluation.Account.Plan)
			remaining, measured := view.BestGroupRemaining()
			eligible = append(eligible, balanceCandidate{
				evaluation: evaluation,
				measured:   measured,
				remaining:  remaining,
			})
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

	// Rank by measured availability (most remaining first). Opaque/unmeasured
	// capacity is a separate, lower tier: it must never outrank a resource
	// backed by measured positive quota. Ties fall back to the name order so
	// the cursor rotation below stays deterministic.
	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].measured != eligible[j].measured {
			return eligible[i].measured
		}
		if eligible[i].measured && eligible[i].remaining != eligible[j].remaining {
			return eligible[i].remaining > eligible[j].remaining
		}
		return eligible[i].evaluation.Profile.Name < eligible[j].evaluation.Profile.Name
	})

	// The leader tier is the set of accounts tied at the top availability;
	// the persisted cursor rotates among them for fair reuse. An account that
	// is strictly behind on measured quota is never selected while a leader
	// has balance left.
	leaderCount := 1
	for i := 1; i < len(eligible); i++ {
		tied := eligible[i].measured == eligible[0].measured &&
			(!eligible[0].measured || eligible[i].remaining == eligible[0].remaining)
		if !tied {
			break
		}
		leaderCount++
	}
	leaders := eligible[:leaderCount]

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
	for i, candidate := range leaders {
		if candidate.evaluation.Profile.Name == last {
			selectedIndex = (i + 1) % len(leaders)
			break
		}
	}
	selected := leaders[selectedIndex].evaluation
	state.LastSelected[provider] = selected.Profile.Name
	if err := writeRotationState(statePath, state); err != nil {
		return &SelectionResult{Evaluations: evaluations}, err
	}
	reason := "selected the eligible account with the most measured quota remaining"
	if !leaders[0].measured {
		reason = "no measured quota evidence; balanced rotation among operationally eligible accounts"
	} else if leaderCount > 1 {
		reason = "balanced rotation among equally available eligible accounts"
	}
	return &SelectionResult{
		SelectedProfile: &selected.Profile,
		Reason:          reason,
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
