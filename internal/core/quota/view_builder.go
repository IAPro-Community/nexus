package quota

import (
	"fmt"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

// BuildQuotaView transforms a UsageSnapshot into a QuotaView.
// It detects model groups from the window Group field and assigns display labels.
func BuildQuotaView(snap model.UsageSnapshot, account, plan string) QuotaView {
	qv := QuotaView{
		Provider:  snap.ProviderID,
		Profile:   snap.ProfileID,
		Account:   account,
		Plan:      plan,
		Status:    string(snap.Status),
		Source:    string(snap.Source),
		FetchedAt: snap.FetchedAt,
	}

	if len(snap.Windows) == 0 {
		// No windows: produce a single empty group with UNKNOWN status.
		qv.ModelGroups = []ModelGroup{{Name: "", Windows: []Window{{
			Kind:      "unknown",
			Label:     "Quota",
			Remaining: 0,
			Status:    string(snap.Status),
			Bar:       RenderBar(0, string(snap.Status), 10),
		}}}}
		qv.ComputeAvailability()
		return qv
	}

	// Determine if we have multiple model groups.
	groupKeys := GroupKeys(snap.Windows)
	multiGroup := len(groupKeys) > 1

	// Build windows grouped by Group field.
	// Preserve insertion order of groupKeys (already sorted by GroupDisplayName).
	groupedWindows := make(map[string][]Window)
	for _, w := range snap.Windows {
		gKey := w.Group
		remaining := 0.0
		if w.RemainingPercent != nil {
			remaining = *w.RemainingPercent
			if remaining < 0 {
				remaining = 0
			}
			if remaining > 100 {
				remaining = 100
			}
		}

		label := WindowLabel(w.Kind)
		if multiGroup && gKey != "" {
			// When multiple groups, prefix label with group context.
			// The group name is shown as a header, so label stays as-is.
		}

		bar := RenderBarWithPercent(remaining, string(snap.Status), 10)

		// For multi-group AGY: if the group is "gemini" and kind is "5h"/"weekly",
		// add model context to the label when ambiguous.
		displayLabel := label

		appended := Window{
			Kind:      w.Kind,
			Label:     displayLabel,
			Remaining: remaining,
			ResetDesc: w.ResetDescription,
			Status:    string(snap.Status),
			Bar:       bar,
		}
		groupedWindows[gKey] = append(groupedWindows[gKey], appended)
	}

	// Build ModelGroups in stable order.
	for _, gKey := range groupKeys {
		displayName := GroupDisplayName(gKey)
		// When there's only one group with key "" and no other groups,
		// use the provider-specific model name as group display only if attributable.
		if !multiGroup && gKey == "" && IsAttributableModelName(snap) {
			displayName = snap.ModelName
		}
		qv.ModelGroups = append(qv.ModelGroups, ModelGroup{
			Key:     gKey,
			Name:    displayName,
			Windows: groupedWindows[gKey],
		})
	}

	SortModelGroups(qv.ModelGroups)

	// If single group with empty name and attributable model name available, set it.
	if len(qv.ModelGroups) == 1 && qv.ModelGroups[0].Name == "" && IsAttributableModelName(snap) {
		qv.ModelGroups[0].Name = snap.ModelName
	}

	// Compute availability from all windows.
	qv.ComputeAvailability()

	return qv
}

// BuildQuotaViewFromDetails constructs a QuotaView from QuotaDetails (legacy wrapper).
// Used during migration when consumers still produce QuotaDetails.
func BuildQuotaViewFromDetails(provider, profile, account, plan, status, modelName string, windows []model.UsageWindow) QuotaView {
	snap := model.UsageSnapshot{
		ProviderID: provider,
		ProfileID:  profile,
		Status:     model.UsageStatus(status),
		FetchedAt:  time.Now(),
		Windows:    windows,
		ModelName:  modelName,
	}
	return BuildQuotaView(snap, account, plan)
}

// BottleneckScore computes an effective capacity score for scheduling.
// Returns (effectiveCapacity, bottleneckKind, avgRemaining).
// Prefer accounts with more usable model families, then higher total capacity
// (usableGroups*100 + avg group remaining). Bottleneck kind remains the global
// tightest window for diagnostics only.
func BottleneckScore(qv *QuotaView) (float64, string, float64) {
	_, bottleneckKind := qv.Bottleneck()
	total, groups := qv.TotalGroupRemaining()
	if groups == 0 {
		return 50.0, "", 50.0
	}
	avgRemaining := total / float64(groups)
	effective, ok := qv.EffectiveCapacityScore()
	if !ok {
		return 50.0, bottleneckKind, avgRemaining
	}
	return effective, bottleneckKind, avgRemaining
}

// RenderSummary returns a one-line summary of the quota view for compact displays.
func RenderSummary(qv *QuotaView) string {
	if qv == nil || len(qv.ModelGroups) == 0 {
		return "UNKNOWN"
	}
	status := qv.Status
	if status == "" {
		status = "unknown"
	}
	if summary := qv.CompactGroupSummary(); summary != "" {
		return fmt.Sprintf("%s (%s)", summary, status)
	}
	bottleneck, kind := qv.Bottleneck()
	if qv.HasMultipleGroups() {
		return fmt.Sprintf("%.0f%% [%s] (%d groups, %s)", bottleneck, kind, len(qv.ModelGroups), status)
	}
	return fmt.Sprintf("%.0f%% [%s] (%s)", bottleneck, kind, status)
}
