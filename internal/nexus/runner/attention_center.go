package runner

import (
	"context"
	"sort"
)

// AttentionCenter is a read model / query surface over canonical MissionRun
// state. It does not own business truth; it projects AttentionItems from the
// RunRepository.
type AttentionCenter struct {
	repo RunRepository
}

// NewAttentionCenter creates an AttentionCenter backed by the given repository.
func NewAttentionCenter(repo RunRepository) *AttentionCenter {
	return &AttentionCenter{repo: repo}
}

// AttentionGroup organizes attention items by category for the UI.
type AttentionGroup struct {
	NeedsYou   []*AttentionItem `json:"needs_you"`
	Completed  []*AttentionItem `json:"completed"`
	Failed     []*AttentionItem `json:"failed"`
	AllItems   []*AttentionItem `json:"all_items"`
	TotalNeeds int              `json:"total_needs"`
}

// ListAttention returns all active attention items across all mission runs,
// grouped by category. This is the primary query for the Attention Center UI.
func (ac *AttentionCenter) ListAttention(ctx context.Context) (*AttentionGroup, error) {
	runs, err := ac.repo.ListRuns(ctx)
	if err != nil {
		return nil, err
	}
	group := &AttentionGroup{}
	for _, run := range runs {
		item := AttentionFromRun(run)
		if item == nil {
			continue
		}
		group.AllItems = append(group.AllItems, item)
		switch item.Level {
		case AttentionRequireUser:
			group.NeedsYou = append(group.NeedsYou, item)
			group.TotalNeeds++
		case AttentionInApp, AttentionNotify:
			switch run.State {
			case StateCompletedVerified:
				group.Completed = append(group.Completed, item)
			default:
				group.Failed = append(group.Failed, item)
			}
		}
	}
	// Sort each group: most recent first.
	sort.Slice(group.NeedsYou, func(i, j int) bool {
		return group.NeedsYou[i].Age < group.NeedsYou[j].Age
	})
	sort.Slice(group.Completed, func(i, j int) bool {
		return group.Completed[i].Age < group.Completed[j].Age
	})
	sort.Slice(group.Failed, func(i, j int) bool {
		return group.Failed[i].Age < group.Failed[j].Age
	})
	return group, nil
}

// GetAttention returns the attention item for a specific mission run.
func (ac *AttentionCenter) GetAttention(ctx context.Context, runID string) (*AttentionItem, error) {
	run, err := ac.repo.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	return AttentionFromRun(run), nil
}

// NeedsUserCount returns the total count of missions requiring human action.
func (ac *AttentionCenter) NeedsUserCount(ctx context.Context) (int, error) {
	runs, err := ac.repo.ListRuns(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, run := range runs {
		if ClassifyAttention(run) == AttentionRequireUser {
			count++
		}
	}
	return count, nil
}
