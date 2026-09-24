package capacitymonitor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

// stubRecommender returns a fixed advisory recommendation.
type stubRecommender struct {
	rec   Recommendation
	err   error
	calls int
}

func (s *stubRecommender) RecommendForLowResource(ctx context.Context, atRisk ResourceKey, reason string) (Recommendation, error) {
	s.calls++
	if s.err != nil {
		return Recommendation{}, s.err
	}
	return s.rec, nil
}

// T4: resource low → recommendation available com reason no evento publicado.
// Recomendação ≠ migração automática: o teste só exige o dado publicado.
func TestRecommendationIncludedOnLowResource(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(5, model.UsageLive),
	}
	m, _, counter := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)
	recommender := &stubRecommender{rec: Recommendation{
		Provider: "claude", Profile: "backup", DisplayName: "backup",
		Score: 82.5, Reason: "healthy fallback with live quota", Available: true,
	}}
	m.deps.Recommender = recommender

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	if counter.count() != 1 {
		t.Fatalf("events = %d, want 1", counter.count())
	}
	data := counter.events[0].Data
	if data["recommended_provider"] != "claude" || data["recommended_profile"] != "backup" {
		t.Fatalf("recommendation candidate missing: %v", data)
	}
	if r, _ := data["recommendation_reason"].(string); r == "" {
		t.Fatalf("recommendation reason empty: %v", data)
	}
	if recommender.calls != 1 {
		t.Fatalf("recommender calls = %d, want exactly 1 per transition", recommender.calls)
	}
	history, err := m.deps.Store.ListCapacityEvents(10)
	if err != nil || len(history) != 1 {
		t.Fatalf("capacity history = %+v, %v; want one durable event", history, err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(history[0].PayloadJSON), &payload); err != nil {
		t.Fatalf("decode durable payload: %v", err)
	}
	if payload["recommended_provider"] != "claude" || payload["recommended_profile"] != "backup" {
		t.Fatalf("durable recommendation missing: %v", payload)
	}
	if payload["recommendation_reason"] != "healthy fallback with live quota" {
		t.Fatalf("durable recommendation reason = %v", payload["recommendation_reason"])
	}
}

// Sem Recommender injetado, o publish continua sem recomendação (sem phantom).
func TestNoRecommendationWithoutRecommender(t *testing.T) {
	probe := &stubProbe{
		desc: quota.ProbeDescriptor{ReadOnly: true, Granularity: quota.ProbeGranularityGroup},
		snap: liveSnapshot(5, model.UsageLive),
	}
	m, _, counter := testMonitor(t,
		[]model.Profile{{Provider: "codex", Name: "kiver"}},
		&stubRegistry{probes: map[string]quota.Probe{"codex": probe}}, nil)

	if err := m.Tick(context.Background()); err != nil {
		t.Fatalf("Tick() = %v", err)
	}
	if counter.count() != 1 {
		t.Fatalf("events = %d, want 1", counter.count())
	}
	if _, ok := counter.events[0].Data["recommended_provider"]; ok {
		t.Fatalf("unexpected recommendation without recommender: %v", counter.events[0].Data)
	}
}
