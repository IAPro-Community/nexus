package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	sdkrouting "github.com/IAPro-Community/IAPro-SDK/routing"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

const rotationStateFile = "account-rotation.json"

type rotationState struct {
	LastSelected map[string]string `json:"last_selected"`
}

// SelectNextBalancedProfile delegates tier-aware :balanced ranking and cursor
// rotation to the shared IAPro-SDK routing engine, persisting the rotation
// cursor under a cross-process file lock.
func (s *Selector) SelectNextBalancedProfile(
	ctx context.Context,
	provider string,
	workspace string,
	candidates []model.Profile,
	accounts map[string]model.AccountInfo,
	excludeProfiles []string,
) (*SelectionResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	boundProfile := config.GetBinding(workspace, provider)
	defaultProfile := s.cfg.Defaults[provider]

	excludedSet := make(map[string]bool, len(excludeProfiles))
	for _, ex := range excludeProfiles {
		excludedSet[ex] = true
	}

	profileByName := make(map[string]model.Profile, len(candidates))
	routingCandidates := make([]sdkrouting.SelectionCandidate, 0, len(candidates))
	for _, p := range candidates {
		if !strings.EqualFold(strings.TrimSpace(p.Provider), provider) {
			continue
		}
		profileByName[p.Name] = p
		acc, hasAcc := accounts[p.Name]
		if !hasAcc {
			acc = model.AccountInfo{
				Authenticated: false,
				Health:        model.HealthUnknown,
			}
		}
		snap := acc.Usage
		scope := p.AccountScope
		if !scope.Verifiable() {
			scope = acc.AccountScope
		}
		if (snap.ProviderID == "" || snap.ProfileID == "") && s.quotaEng != nil {
			if scope.Verifiable() {
				snap, _ = s.quotaEng.GetCachedUsageForScope(scope)
			} else {
				snap, _ = s.quotaEng.GetCachedUsage(provider, p.Name)
			}
		}
		if (snap.ProfileID != "" && snap.ProfileID != p.Name) ||
			(snap.ProviderID != "" && !strings.EqualFold(snap.ProviderID, provider)) ||
			(snap.Source == model.SourceNone && snap.Status != model.UsageUnsupported && snap.Status != "") {
			snap = model.UsageSnapshot{
				ProviderID: provider,
				ProfileID:  p.Name,
				Status:     model.UsageUnknown,
				Source:     model.SourceNone,
			}
		}
		var isLimited bool
		if s.cooldown != nil {
			if scope.Verifiable() {
				isLimited, _ = s.cooldown.IsRateLimitedForScope(scope)
			} else {
				isLimited, _ = s.cooldown.IsRateLimited(provider, p.Name)
			}
		}
		priority := p.Priority
		if s.cfg.Priorities[provider] != nil {
			priority = s.cfg.Priorities[provider][p.Name]
		}
		disabled := p.Disabled || (s.cfg.Disabled[provider] != nil && s.cfg.Disabled[provider][p.Name])

		routingCandidates = append(routingCandidates, sdkrouting.SelectionCandidate{
			Scope:         scope,
			ProviderID:    provider,
			ProfileID:     p.Name,
			Account:       acc.Email,
			Plan:          acc.Plan,
			Authenticated: acc.Authenticated,
			Disabled:      disabled,
			Excluded:      excludedSet[p.Name],
			InCooldown:    isLimited,
			Health:        string(acc.Health),
			Priority:      priority,
			IsDefault:     defaultProfile == p.Name,
			IsBound:       boundProfile == p.Name,
			Snapshot:      snap,
		})
	}

	now := time.Now()
	req := sdkrouting.SelectionRequest{
		ProviderID: provider,
		Balanced:   true,
		TieEpsilon: 0.01,
		Now:        now,
	}
	decision := sdkrouting.RankAndSelect(req, routingCandidates)

	if err := ctx.Err(); err != nil {
		return buildBalancedSelectionResult(decision, profileByName, accounts), err
	}
	if decision.Selected == nil {
		res := buildBalancedSelectionResult(decision, profileByName, accounts)
		return res, fmt.Errorf("no eligible %s profiles for balanced rotation: %s", provider, decision.Explanation)
	}

	stateDir, err := config.StateDir()
	if err != nil {
		return buildBalancedSelectionResult(decision, profileByName, accounts), fmt.Errorf("resolve account rotation state: %w", err)
	}
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return buildBalancedSelectionResult(decision, profileByName, accounts), fmt.Errorf("create account rotation state: %w", err)
	}
	statePath := filepath.Join(stateDir, rotationStateFile)
	unlock, err := acquireRotationLock(statePath + ".lock")
	if err != nil {
		return buildBalancedSelectionResult(decision, profileByName, accounts), err
	}
	defer func() { _ = unlock() }()

	state := rotationState{LastSelected: make(map[string]string)}
	if data, readErr := os.ReadFile(statePath); readErr == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return buildBalancedSelectionResult(decision, profileByName, accounts), fmt.Errorf("decode account rotation state: %w", err)
		}
		if state.LastSelected == nil {
			state.LastSelected = make(map[string]string)
		}
	} else if !os.IsNotExist(readErr) {
		return buildBalancedSelectionResult(decision, profileByName, accounts), fmt.Errorf("read account rotation state: %w", err)
	}

	req.LastSelectedID = state.LastSelected[provider]
	decision = sdkrouting.RankAndSelect(req, routingCandidates)
	if decision.Selected == nil {
		res := buildBalancedSelectionResult(decision, profileByName, accounts)
		return res, fmt.Errorf("no eligible %s profiles for balanced rotation: %s", provider, decision.Explanation)
	}

	state.LastSelected[provider] = decision.Selected.ProfileID
	if err := writeRotationState(statePath, state); err != nil {
		return buildBalancedSelectionResult(decision, profileByName, accounts), err
	}
	return buildBalancedSelectionResult(decision, profileByName, accounts), nil
}

func buildBalancedSelectionResult(
	decision sdkrouting.SelectionDecision,
	profileByName map[string]model.Profile,
	accounts map[string]model.AccountInfo,
) *SelectionResult {
	evals := make([]CandidateEvaluation, 0, len(decision.Evaluations))
	for _, out := range decision.Evaluations {
		p := profileByName[out.ProfileID]
		acc := accounts[out.ProfileID]
		evals = append(evals, CandidateEvaluation{
			Profile:        p,
			Account:        acc,
			Usage:          acc.Usage,
			Score:          out.Score,
			Eligible:       out.Eligible,
			RejectReason:   out.RejectReason,
			ScoreBreakdown: out.ScoreBreakdown,
			ActiveTier:     out.ActiveTier,
			NeedsRefresh:   out.NeedsRefresh,
			Projection:     out.Projection,
		})
	}
	var selected *model.Profile
	if decision.Selected != nil {
		p := profileByName[decision.Selected.ProfileID]
		selected = &p
	}
	return &SelectionResult{
		SelectedProfile:       selected,
		Reason:                decision.Explanation,
		Evaluations:           evals,
		ActiveTier:            decision.ActiveTier,
		UsedAuxiliaryFallback: decision.UsedAuxiliaryFallback,
		NeedsRefresh:          decision.NeedsRefresh,
	}
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
