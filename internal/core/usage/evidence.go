package usage

import (
	"sort"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// ClampPercent bounds a provider percentage without changing its semantics.
func ClampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

// PercentWindowFromUsed converts USED_PERCENT evidence into the canonical
// remaining/used UsageWindow contract.
func PercentWindowFromUsed(kind, group string, used float64, resetAt *time.Time, resetDescription string) model.UsageWindow {
	used = ClampPercent(used)
	remaining := 100 - used
	return model.UsageWindow{
		Kind:             kind,
		Group:            group,
		UsedPercent:      &used,
		RemainingPercent: &remaining,
		ResetTime:        resetAt,
		ResetDescription: resetDescription,
	}
}

// PercentWindowFromRemaining converts REMAINING_PERCENT evidence without
// inverting the provider value.
func PercentWindowFromRemaining(kind, group string, remaining float64, resetAt *time.Time, resetDescription string) model.UsageWindow {
	remaining = ClampPercent(remaining)
	used := 100 - remaining
	return model.UsageWindow{
		Kind:             kind,
		Group:            group,
		UsedPercent:      &used,
		RemainingPercent: &remaining,
		ResetTime:        resetAt,
		ResetDescription: resetDescription,
	}
}

// RemainingPercent normalizes percent observations while preserving source
// semantics. Absolute/binary observations intentionally do not fabricate one.
func RemainingPercent(observation model.UsageObservation) (float64, bool) {
	switch observation.Semantics {
	case model.SemanticsUsedPercent:
		return 100 - ClampPercent(observation.Value), true
	case model.SemanticsRemainingPercent:
		return ClampPercent(observation.Value), true
	default:
		return 0, false
	}
}

// ReconcileQuotaObservations projects typed evidence for one channel into the
// legacy UsageSnapshot consumed by current UI/scheduler code. API consumption
// and cost observations are never coerced into subscription quota.
func ReconcileQuotaObservations(providerID, profileID string, channel model.ResourceChannel, observations []model.UsageObservation) model.UsageSnapshot {
	snapshot := model.UsageSnapshot{
		ProviderID: providerID,
		ProfileID:  profileID,
		Status:     model.UsageUnknown,
		Source:     model.SourceNone,
	}

	type key struct {
		group  string
		window string
	}
	latest := make(map[key]model.UsageObservation)
	var scope model.AccountScope

	for _, observation := range observations {
		if observation.Metric != model.UsageMetricQuota || observation.Channel != channel {
			continue
		}
		if observation.ProviderID != "" && observation.ProviderID != providerID {
			continue
		}
		if observation.ProfileID != "" && observation.ProfileID != profileID {
			continue
		}
		if scope.Verifiable() && observation.AccountScope.Verifiable() && observation.AccountScope.Key() != scope.Key() {
			// A single reconciliation batch must belong to exactly one durable
			// account. Mixed identities are contamination, not a signal to pick
			// whichever account happened to appear first.
			return snapshot
		}
		if !scope.Verifiable() && observation.AccountScope.Verifiable() {
			scope = observation.AccountScope
		}
		k := key{group: observation.ModelGroup, window: observation.WindowKind}
		if previous, ok := latest[k]; !ok || observation.ObservedAt.After(previous.ObservedAt) {
			latest[k] = observation
		}
	}

	if len(latest) == 0 {
		return snapshot
	}

	keys := make([]key, 0, len(latest))
	for k := range latest {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].group != keys[j].group {
			return keys[i].group < keys[j].group
		}
		return keys[i].window < keys[j].window
	})

	var newest model.UsageObservation
	for _, k := range keys {
		observation := latest[k]
		window := model.UsageWindow{
			Kind:      observation.WindowKind,
			Group:     observation.ModelGroup,
			Unit:      observation.Unit,
			ResetTime: observation.ResetAt,
		}
		switch observation.Semantics {
		case model.SemanticsUsedPercent:
			used := ClampPercent(observation.Value)
			remaining := 100 - used
			window.UsedPercent = &used
			window.RemainingPercent = &remaining
		case model.SemanticsRemainingPercent:
			remaining := ClampPercent(observation.Value)
			used := 100 - remaining
			window.UsedPercent = &used
			window.RemainingPercent = &remaining
		case model.SemanticsUsedAbsolute:
			value := observation.Value
			window.Used = &value
		case model.SemanticsRemainingAbsolute:
			value := observation.Value
			window.Remaining = &value
		default:
			continue
		}
		snapshot.Windows = append(snapshot.Windows, window)
		if newest.ObservedAt.IsZero() || observation.ObservedAt.After(newest.ObservedAt) {
			newest = observation
		}
	}

	if len(snapshot.Windows) == 0 {
		return snapshot
	}
	snapshot.AccountScope = scope
	snapshot.Status = newest.Status
	if snapshot.Status == "" {
		snapshot.Status = model.UsageEstimated
	}
	snapshot.Source = newest.Source
	if snapshot.Source == "" {
		snapshot.Source = model.SourceObservation
	}
	snapshot.Confidence = newest.Confidence
	snapshot.FetchedAt = newest.ObservedAt
	return snapshot
}
