package store

import (
	"errors"
	"math"
	"testing"
	"time"
)

func float64ptr(v float64) *float64 { return &v }

func baseObservabilityObservation(now time.Time) CostObservation {
	return CostObservation{
		ProjectID:    "project-obs",
		MissionRunID: "run-obs",
		PackageID:    "pkg-obs",
		InvocationID: "inv-obs-1",
		AgentID:      "agent-obs",
		Provider:     "codex",
		Profile:      "work",
		Model:        "gpt-5.6",
		InputTokens:  int64ptr(100),
		OutputTokens: int64ptr(30),
		DurationMS:   1250,
		CostMicroUSD: int64ptr(4200),
		CostKind:     CostKindProviderReported,
		Source:       "provider_response",
		Status:       "COMPLETED",
		SessionID:    "session-obs-1",
		TTFTMS:       int64ptr(310),
		ObservedAt:   now,
	}
}

func TestCostObservationReplayIsIdempotent(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	first, err := s.RecordCostObservation(baseObservabilityObservation(now))
	if err != nil {
		t.Fatal(err)
	}

	// Same logical (source, invocation_id) with identical immutable evidence
	// but a different id and volatile status must return the stored row.
	replay := baseObservabilityObservation(now)
	replay.ID = "cost_replay_probe"
	replay.Status = "RUNNING"
	replay.CostKind = CostKindUnknown
	existing, err := s.RecordCostObservation(replay)
	if err != nil {
		t.Fatal(err)
	}
	if existing.ID != first.ID {
		t.Fatalf("replay got id=%q, want stored id=%q", existing.ID, first.ID)
	}
	if existing.Status != "COMPLETED" || existing.CostKind != CostKindProviderReported {
		t.Fatalf("replay must return the committed row unchanged: %+v", existing)
	}

	rows, err := s.ListCostObservations(CostObservationFilter{ProjectID: "project-obs", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("observations=%d, want exactly 1 after replay", len(rows))
	}
}

func TestCostObservationReplayConflictKeepsOriginalEvidence(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	first, err := s.RecordCostObservation(baseObservabilityObservation(now))
	if err != nil {
		t.Fatal(err)
	}

	conflicting := baseObservabilityObservation(now)
	conflicting.CostMicroUSD = int64ptr(9900)
	_, err = s.RecordCostObservation(conflicting)
	if !errors.Is(err, ErrCostObservationConflict) {
		t.Fatalf("err=%v, want ErrCostObservationConflict", err)
	}

	// Divergent observed_at is immutable evidence too.
	divergentTime := baseObservabilityObservation(now)
	divergentTime.ObservedAt = now.Add(time.Second)
	if _, err := s.RecordCostObservation(divergentTime); !errors.Is(err, ErrCostObservationConflict) {
		t.Fatalf("err=%v, want ErrCostObservationConflict for divergent observed_at", err)
	}

	rows, err := s.ListCostObservations(CostObservationFilter{ProjectID: "project-obs", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ID != first.ID || *rows[0].CostMicroUSD != 4200 {
		t.Fatalf("conflicting replays must not mutate stored evidence: %+v", rows)
	}
}

func TestCostObservationTTFTAndThroughputNullWhenNotMeasured(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	unmeasured := baseObservabilityObservation(now)
	unmeasured.TTFTMS = nil
	unmeasured.OutputTokensPerSecond = nil
	unmeasured.SessionID = ""
	if _, err := s.RecordCostObservation(unmeasured); err != nil {
		t.Fatal(err)
	}

	rows, err := s.ListCostObservations(CostObservationFilter{ProjectID: "project-obs", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("observations=%d, want 1", len(rows))
	}
	if rows[0].TTFTMS != nil {
		t.Fatalf("unmeasured ttft_ms must stay nil, got %d", *rows[0].TTFTMS)
	}
	if rows[0].OutputTokensPerSecond != nil {
		t.Fatalf("unmeasured output_tokens_per_second must stay nil, got %f", *rows[0].OutputTokensPerSecond)
	}
	if rows[0].SessionID != "" {
		t.Fatalf("session_id must round trip as empty, got %q", rows[0].SessionID)
	}

	// When both output tokens and duration are measured, throughput evidence
	// round trips through the store instead of being synthesized.
	measured := baseObservabilityObservation(now)
	measured.InvocationID = "inv-obs-measured"
	measured.SessionID = "session-obs-measured"
	measured.OutputTokensPerSecond = float64ptr(17.5)
	if _, err := s.RecordCostObservation(measured); err != nil {
		t.Fatal(err)
	}
	rows, err = s.ListCostObservations(CostObservationFilter{SessionID: "session-obs-measured", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].TTFTMS == nil || *rows[0].TTFTMS != 310 || rows[0].OutputTokensPerSecond == nil || *rows[0].OutputTokensPerSecond != 17.5 {
		t.Fatalf("measured ttft/tps lost on round trip: %+v", rows)
	}
}

func TestCostObservationFilterBySessionID(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	first := baseObservabilityObservation(now)
	first.SessionID = "session-a"
	second := baseObservabilityObservation(now)
	second.InvocationID = "inv-obs-2"
	second.SessionID = "session-b"
	for _, obs := range []CostObservation{first, second} {
		if _, err := s.RecordCostObservation(obs); err != nil {
			t.Fatal(err)
		}
	}

	rows, err := s.ListCostObservations(CostObservationFilter{SessionID: "session-b", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].SessionID != "session-b" || rows[0].InvocationID != "inv-obs-2" {
		t.Fatalf("session filter got %+v, want only session-b row", rows)
	}

	agg, err := s.SummarizeCostObservations(CostObservationFilter{SessionID: "session-a"})
	if err != nil {
		t.Fatal(err)
	}
	if agg.ObservationCount != 1 {
		t.Fatalf("session-filtered aggregate count=%d, want 1", agg.ObservationCount)
	}
}

func TestCostObservationRejectsInvalidTTFTAndThroughput(t *testing.T) {
	s := openTestStore(t)
	cases := []struct {
		name string
		mut  func(o *CostObservation)
	}{
		{"negative ttft", func(o *CostObservation) { o.TTFTMS = int64ptr(-1) }},
		{"negative throughput", func(o *CostObservation) { o.OutputTokensPerSecond = float64ptr(-0.5) }},
		{"nan throughput", func(o *CostObservation) { o.OutputTokensPerSecond = float64ptr(math.NaN()) }},
		{"inf throughput", func(o *CostObservation) { o.OutputTokensPerSecond = float64ptr(math.Inf(1)) }},
	}
	for _, tc := range cases {
		obs := baseObservabilityObservation(time.Now().UTC())
		obs.InvocationID = "inv-invalid-" + tc.name
		tc.mut(&obs)
		if _, err := s.RecordCostObservation(obs); err == nil {
			t.Fatalf("%s must be rejected", tc.name)
		}
	}
}

func TestCostObservationLegacyRowsWithoutInvocationStayInsertible(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)

	// Legacy writers never set invocation_id. The partial unique index must
	// exclude those rows so multiple observations per source keep landing.
	for i := 0; i < 3; i++ {
		legacy := baseObservabilityObservation(now)
		legacy.InvocationID = ""
		legacy.ObservedAt = now.Add(time.Duration(i) * time.Second)
		if _, err := s.RecordCostObservation(legacy); err != nil {
			t.Fatalf("legacy observation %d: %v", i, err)
		}
	}
	rows, err := s.ListCostObservations(CostObservationFilter{ProjectID: "project-obs", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("legacy observations=%d, want 3", len(rows))
	}

	// The dedupe index exists on the migrated store.
	dbRows, err := s.DB().Query(`SELECT name FROM sqlite_master WHERE type='index' AND name='idx_cost_observations_invocation_source'`)
	if err != nil {
		t.Fatal(err)
	}
	defer dbRows.Close()
	indexed := false
	for dbRows.Next() {
		indexed = true
	}
	if err := dbRows.Err(); err != nil {
		t.Fatal(err)
	}
	if !indexed {
		t.Fatal("partial dedupe index idx_cost_observations_invocation_source missing after migration")
	}
}
