package capacitymonitor

import (
	"math"
	"sort"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// PaceForecast is an advisory projection of recent used-percent movement.
// Its estimates must never be fed back as authoritative quota evidence.
type PaceForecast struct {
	Status          string     `json:"status"` // RISING, RESET_FIRST or STABLE
	Confidence      string     `json:"confidence"`
	Samples         int        `json:"samples"`
	UsedPerHour     float64    `json:"used_per_hour"`
	ExhaustsAt      *time.Time `json:"exhausts_at,omitempty"`
	ResetAt         time.Time  `json:"reset_at"`
}

// PredictPace requires >= 3 fresh samples spanning >= 10 minutes from a
// single provider/profile/account/group/window/reset cycle. It rejects usage
// decreases, resets and ambiguous data instead of making invented estimates.
func PredictPace(samples []store.CapacityPaceSample, now time.Time) *PaceForecast {
	if len(samples) < 3 { return nil }
	list := append([]store.CapacityPaceSample(nil), samples...)
	sort.Slice(list, func(i, j int) bool { return list[i].ObservedAt.Before(list[j].ObservedAt) })
	first, last := list[0], list[len(list)-1]
	if first.AccountScope == "" || first.Provider == "" || first.Profile == "" ||
		first.WindowKind == "" || first.ResetAt.IsZero() || !first.ResetAt.After(now) ||
		last.ObservedAt.After(now.Add(time.Minute)) || now.Sub(last.ObservedAt) > 10*time.Minute {
		return nil
	}
	for i, sample := range list {
		if sample.Provider != first.Provider || sample.Profile != first.Profile ||
			sample.AccountScope != first.AccountScope || sample.QuotaGroup != first.QuotaGroup ||
			sample.WindowKind != first.WindowKind || !sample.ResetAt.Equal(first.ResetAt) ||
			sample.ObservedAt.After(now.Add(time.Minute)) || sample.ObservedAt.Before(now.Add(-24*time.Hour)) ||
			math.IsNaN(sample.UsedPercent) || math.IsInf(sample.UsedPercent, 0) ||
			sample.UsedPercent < 0 || sample.UsedPercent > 100 {
			return nil
		}
		if i > 0 && sample.UsedPercent < list[i-1].UsedPercent-1 {
			return nil // refill or changed semantics, even within the same reset cycle
		}
	}
	span := last.ObservedAt.Sub(first.ObservedAt)
	if span < 10*time.Minute { return nil }
	perHour := (last.UsedPercent - first.UsedPercent) / span.Hours()
	if perHour < 0 || math.IsNaN(perHour) || math.IsInf(perHour, 0) { return nil }
	confidence := "LOW"
	if len(list) >= 4 && span >= 20*time.Minute { confidence = "MEDIUM" }
	if len(list) >= 6 && span >= 30*time.Minute { confidence = "HIGH" }
	forecast := &PaceForecast{Status: "STABLE", Confidence: confidence, Samples: len(list),
		UsedPerHour: math.Round(perHour*100)/100, ResetAt: first.ResetAt}
	if perHour < 0.1 || last.UsedPercent >= 100 { return forecast }
	hours := (100 - last.UsedPercent) / perHour
	if hours <= 0 || hours > 24*365 { return forecast }
	exhaustsAt := last.ObservedAt.Add(time.Duration(hours*float64(time.Hour)))
	if !exhaustsAt.After(now) { return forecast }
	if exhaustsAt.Before(first.ResetAt) {
		forecast.Status = "RISING"
		forecast.ExhaustsAt = &exhaustsAt
	} else {
		forecast.Status = "RESET_FIRST"
	}
	return forecast
}

 
// ForecastForObservation selects exactly the reset cycle and account that
// produced the currently displayed LIVE observation. Older identities and
// older cycles can never appear as a current projection.
func ForecastForObservation(all []store.CapacityPaceSample, current store.ObservationRecord, now time.Time) *PaceForecast {
	if current.AccountScope == "" || current.Status != "LIVE" ||
		current.FetchedAt.IsZero() || current.FetchedAt.After(now.Add(time.Minute)) ||
		now.Sub(current.FetchedAt) > 10*time.Minute {
		return nil
	}
	var cycle time.Time
	var latest time.Time
	for _, v := range all {
		if v.Provider != current.Provider || v.Profile != current.Profile ||
			v.AccountScope != current.AccountScope || v.QuotaGroup != current.QuotaGroup ||
			v.WindowKind != current.WindowKind || !v.ResetAt.After(now) {
			continue
		}
		if v.ObservedAt.After(latest) &&
			!v.ObservedAt.Before(current.FetchedAt.Add(-10*time.Minute)) &&
			!v.ObservedAt.After(current.FetchedAt.Add(time.Minute)) {
			latest = v.ObservedAt
			cycle = v.ResetAt
		}
	}
	if cycle.IsZero() { return nil }
	selected := make([]store.CapacityPaceSample, 0, 16)
	for _, v := range all {
		if v.Provider == current.Provider && v.Profile == current.Profile &&
			v.AccountScope == current.AccountScope && v.QuotaGroup == current.QuotaGroup &&
			v.WindowKind == current.WindowKind && v.ResetAt.Equal(cycle) {
			selected = append(selected, v)
		}
	}
	return PredictPace(selected, now)
}
