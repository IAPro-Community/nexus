package capacitymonitor

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestPredictPaceTrustAndIsolation(t *testing.T) {
	now := time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)
	reset := now.Add(2 * time.Hour)
	makeSample := func(minutes int, used float64) store.CapacityPaceSample {
		return store.CapacityPaceSample{
			Provider: "codex", Profile: "account-1", AccountScope: "codex/verified-1",
			QuotaGroup: "default", WindowKind: "5h",
			ResetAt: reset, ObservedAt: now.Add(time.Duration(minutes) * time.Minute),
			UsedPercent: used, Source: "OFFICIAL_API",
		}
	}
	samples := []store.CapacityPaceSample{
		makeSample(-40, 20), makeSample(-30, 32),
		makeSample(-20, 44), makeSample(-10, 56), makeSample(0, 68),
	}
	got := PredictPace(samples, now)
	if got == nil || got.Status != "RISING" || got.ExhaustsAt == nil ||
		got.Confidence != "MEDIUM" || !got.ExhaustsAt.Before(reset) {
		t.Fatalf("expected qualified rising forecast, got %+v", got)
	}
	latestPct := 68.0
	latest := store.ObservationRecord{
		UsedPercent: &latestPct,
		Provider:    "codex", Profile: "account-1", AccountScope: "codex/verified-1",
		QuotaGroup: "default", WindowKind: "5h", Status: "LIVE", FetchedAt: now,
	}
	if f := ForecastForObservation(samples, latest, now); f == nil || f.Status != "RISING" {
		t.Fatalf("matching observation should be projected: %+v", f)
	}
	different := 98.0
	latest.UsedPercent = &different
	if f := ForecastForObservation(samples, latest, now); f != nil {
		t.Fatalf("divergent live quota must invalidate forecast: %+v", f)
	}
	latest.UsedPercent = &latestPct
	latest.AccountScope = "codex/verified-2"
	if f := ForecastForObservation(samples, latest, now); f != nil {
		t.Fatalf("different account inherited quota history: %+v", f)
	}
	latest.AccountScope = "codex/verified-1"
	latest.Status = "CACHED"
	if f := ForecastForObservation(samples, latest, now); f != nil {
		t.Fatalf("cached status must not produce live forecast: %+v", f)
	}
	latest.Status = "LIVE"
	latest.FetchedAt = now.Add(-30 * time.Minute)
	if f := ForecastForObservation(samples, latest, now); f != nil {
		t.Fatalf("stale observation produced forecast: %+v", f)
	}
}

func TestPredictPaceHonorsResetAndRejectsBadSeries(t *testing.T) {
	now := time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)
	reset := now.Add(15 * time.Minute)
	sample := func(mins int, pct float64) store.CapacityPaceSample {
		return store.CapacityPaceSample{
			Provider: "codex", Profile: "account-1", AccountScope: "verified",
			WindowKind: "5h", ResetAt: reset, ObservedAt: now.Add(time.Duration(mins) * time.Minute),
			UsedPercent: pct,
		}
	}
	slow := []store.CapacityPaceSample{sample(-30, 30), sample(-15, 31), sample(0, 32)}
	if got := PredictPace(slow, now); got == nil || got.Status != "RESET_FIRST" ||
		got.ExhaustsAt != nil {
		t.Fatalf("reset must precede exhaustion: %+v", got)
	}
	mixed := append([]store.CapacityPaceSample(nil), slow...)
	mixed[1].ResetAt = reset.Add(time.Hour)
	if got := PredictPace(mixed, now); got != nil {
		t.Fatalf("reset-cycle mixing accepted: %+v", got)
	}
	refilled := []store.CapacityPaceSample{sample(-30, 70), sample(-15, 20), sample(0, 40)}
	if got := PredictPace(refilled, now); got != nil {
		t.Fatalf("unexpected refill trend: %+v", got)
	}
	if got := PredictPace(slow[:2], now); got != nil {
		t.Fatalf("too few samples: %+v", got)
	}
}

func TestPaceSourceTrustBoundary(t *testing.T) {
	for _, source := range []model.UsageSource{model.SourceOfficialAPI, model.SourceCLIOutput, model.SourceResponseHeader} {
		if !trustedPaceSource(source) {
			t.Errorf("%s is direct provider evidence", source)
		}
	}
	for _, source := range []model.UsageSource{model.SourceNone, model.SourceLocalFiles, model.SourceObservation, ""} {
		if trustedPaceSource(source) {
			t.Errorf("%s must not generate forecast", source)
		}
	}
}
