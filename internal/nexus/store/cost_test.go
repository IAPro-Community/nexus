package store

import (
	"testing"
	"time"
)

func int64ptr(v int64) *int64 { return &v }

func TestCostObservationRoundTripPreservesUnknownAndKnownEvidence(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	known, err := s.RecordCostObservation(CostObservation{
		ProjectID:           "project-1",
		MissionRunID:        "run-1",
		PackageID:           "pkg-1",
		InvocationID:        "inv-1",
		AgentID:             "agent-1",
		RuntimeGenerationID: "gen-1",
		RuntimeID:           "runtime-1",
		Provider:            "codex",
		Profile:             "work",
		Model:               "gpt-5.6",
		Branch:              "feat/test",
		WorkspaceKind:       "worktree",
		InputTokens:         int64ptr(100),
		CachedInputTokens:   int64ptr(20),
		OutputTokens:        int64ptr(30),
		ReasoningTokens:     int64ptr(10),
		DurationMS:          1250,
		CostMicroUSD:        int64ptr(4200),
		CostKind:            CostKindProviderReported,
		Source:              "provider_response",
		Status:              "COMPLETED",
		ObservedAt:          now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if known.ID == "" {
		t.Fatal("cost observation must get a durable id")
	}
	if _, err := s.RecordCostObservation(CostObservation{
		ProjectID:  "project-1",
		Provider:   "claude",
		Profile:    "default",
		DurationMS: 250,
		CostKind:   CostKindUnknown,
		Source:     "runtime_observation",
		Status:     "COMPLETED",
		ObservedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := s.ListCostObservations(CostObservationFilter{ProjectID: "project-1", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("observations=%d, want 2", len(rows))
	}
	if rows[0].CostKind != CostKindUnknown || rows[0].CostMicroUSD != nil || rows[0].InputTokens != nil {
		t.Fatalf("unknown evidence was fabricated: %+v", rows[0])
	}
	if rows[1].CostMicroUSD == nil || *rows[1].CostMicroUSD != 4200 || rows[1].InputTokens == nil || *rows[1].InputTokens != 100 {
		t.Fatalf("known evidence lost: %+v", rows[1])
	}
}

func TestCostObservationRejectsInvalidNegativeEvidence(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.RecordCostObservation(CostObservation{
		CostKind:     CostKindInferred,
		Source:       "test",
		InputTokens:  int64ptr(-1),
		ObservedAt:   time.Now().UTC(),
	}); err == nil {
		t.Fatal("negative token evidence must be rejected")
	}
	if _, err := s.RecordCostObservation(CostObservation{
		CostKind:    "FAKE",
		Source:      "test",
		ObservedAt:  time.Now().UTC(),
	}); err == nil {
		t.Fatal("unknown cost kind must be rejected")
	}
}

func TestCostObservationFiltersByMissionProviderAndTime(t *testing.T) {
	s := openTestStore(t)
	base := time.Now().UTC().Add(-time.Hour)
	for _, obs := range []CostObservation{
		{ProjectID: "p", MissionRunID: "m1", Provider: "codex", Profile: "a", CostKind: CostKindUnknown, Source: "test", ObservedAt: base},
		{ProjectID: "p", MissionRunID: "m2", Provider: "claude", Profile: "b", CostKind: CostKindUnknown, Source: "test", ObservedAt: base.Add(30 * time.Minute)},
		{ProjectID: "p", MissionRunID: "m1", Provider: "codex", Profile: "a", CostKind: CostKindUnknown, Source: "test", ObservedAt: base.Add(45 * time.Minute)},
	} {
		if _, err := s.RecordCostObservation(obs); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := s.ListCostObservations(CostObservationFilter{
		MissionRunID: "m1",
		Provider:     "codex",
		Since:        base.Add(15 * time.Minute),
		Limit:        10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].MissionRunID != "m1" || rows[0].Provider != "codex" {
		t.Fatalf("filtered rows=%+v, want one m1/codex observation", rows)
	}
}
