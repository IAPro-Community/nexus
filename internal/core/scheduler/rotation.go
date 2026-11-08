package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
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
	evaluations := s.EvaluateAll(provider, workspace, candidates, accounts, excludeProfiles)
	eligible := make([]model.Profile, 0, len(evaluations))
	for _, evaluation := range evaluations {
		if evaluation.Eligible {
			eligible = append(eligible, evaluation.Profile)
		}
	}
	if len(eligible) == 0 {
		return &SelectionResult{Evaluations: evaluations}, fmt.Errorf("no eligible %s profiles for balanced rotation", provider)
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

	provider = strings.ToLower(strings.TrimSpace(provider))
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
