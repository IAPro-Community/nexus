package store

import (
	"testing"
	"time"
)

func TestCapacityPaceSamplesScopedBoundedAndPrunable(t *testing.T) {
	s := openTestStore(t)
	now := time.Date(2026, 10, 14, 15, 2, 0, 0, time.UTC)
	reset := now.Add(2 * time.Hour)
	base := CapacityPaceSample{
		Provider: "codex", Profile: "main", AccountScope: "verified-1",
		WindowKind: "5h", ResetAt: reset, ObservedAt: now, UsedPercent: 30,
		Source: "OFFICIAL_API",
	}
	if err := s.RecordCapacityPaceSample(base); err != nil { t.Fatal(err) }
	base.ObservedAt = now.Add(time.Minute)
	base.UsedPercent = 33
	if err := s.RecordCapacityPaceSample(base); err != nil { t.Fatal(err) }
	base.AccountScope = "verified-2"
	if err := s.RecordCapacityPaceSample(base); err != nil { t.Fatal(err) }
	base.AccountScope = "verified-1"
	base.ResetAt = reset.Add(5 * time.Hour)
	if err := s.RecordCapacityPaceSample(base); err != nil { t.Fatal(err) }
	base.AccountScope = "" // never store anonymous data
	if err := s.RecordCapacityPaceSample(base); err != nil { t.Fatal(err) }
	records, err := s.ListCapacityPaceSamples(now.Add(-time.Hour))
	if err != nil { t.Fatal(err) }
	if len(records) != 3 { t.Fatalf("want 3 isolated samples, got %d: %+v", len(records), records) }
	found := 0
	for _, record := range records {
		if record.AccountScope == "verified-1" && record.ResetAt.Equal(reset) {
			found++
			if record.UsedPercent != 33 { t.Fatalf("5-minute bucket not updated: %+v", record) }
		}
	}
	if found != 1 { t.Fatalf("want one current scope/cycle sample, got %d", found) }
	if _, err := s.PruneCapacityPaceSamples(now.Add(time.Minute)); err != nil { t.Fatal(err) }
	records, err = s.ListCapacityPaceSamples(now.Add(-time.Hour))
	if err != nil { t.Fatal(err) }
	if len(records) != 0 { t.Fatalf("prune should delete historical samples, got %d", len(records)) }
}
