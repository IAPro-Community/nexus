package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCapacityStoreLifecycle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nexus_capacity_test.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	// 1. Observations
	rem := 15.0
	obs := ObservationRecord{
		Provider:         "agy",
		Profile:          "work",
		QuotaGroup:       "gemini",
		WindowKind:       "5h",
		Status:           "LIVE",
		Source:           "CLI_OUTPUT",
		RemainingPercent: &rem,
		FetchedAt:        now,
		ExpiresAt:        now.Add(10 * time.Minute),
		ResetDescription: "Resets in 3h",
	}
	if err := s.UpsertCapacityObservation(obs); err != nil {
		t.Fatalf("upsert observation: %v", err)
	}
	list, err := s.ListCapacityObservations()
	if err != nil || len(list) != 1 {
		t.Fatalf("list observations: %v, count=%d", err, len(list))
	}
	if list[0].Provider != "agy" || list[0].RemainingPercent == nil || *list[0].RemainingPercent != 15.0 {
		t.Fatalf("unexpected observation: %+v", list[0])
	}

	// 2. Capacity State
	state := CapacityStateRecord{
		Key:            "agy:work:gemini:5h",
		Provider:       "agy",
		Profile:        "work",
		QuotaGroup:     "gemini",
		WindowKind:     "5h",
		State:          "WARNING",
		Reason:         "Remaining quota at 15%",
		TransitionedAt: now,
		UpdatedAt:      now,
	}
	if err := s.UpsertCapacityState(state); err != nil {
		t.Fatalf("upsert state: %v", err)
	}
	gotState, err := s.GetCapacityState("agy:work:gemini:5h")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if gotState.State != "WARNING" {
		t.Fatalf("expected state WARNING, got %s", gotState.State)
	}

	// 3. Lease acquire, renew, expiry & takeover
	lease, acquired, err := s.AcquireCapacityLease("proc-1", now, time.Minute)
	if err != nil || !acquired || lease.Owner != "proc-1" {
		t.Fatalf("acquire lease proc-1 failed: err=%v, acquired=%v, owner=%s", err, acquired, lease.Owner)
	}

	// Another process cannot acquire while active
	_, acquired2, err := s.AcquireCapacityLease("proc-2", now.Add(10*time.Second), time.Minute)
	if err != nil || acquired2 {
		t.Fatalf("proc-2 should not acquire active lease: acquired=%v", acquired2)
	}

	// Takeover after expiry
	lease2, acquired3, err := s.AcquireCapacityLease("proc-2", now.Add(2*time.Minute), time.Minute)
	if err != nil || !acquired3 || lease2.Owner != "proc-2" {
		t.Fatalf("proc-2 must acquire expired lease: err=%v, acquired=%v, owner=%s", err, acquired3, lease2.Owner)
	}

	// Release
	if err := s.ReleaseCapacityLease("proc-2"); err != nil {
		t.Fatalf("release lease: %v", err)
	}

	// 4. Events and Deduplication
	ev := CapacityEventRecord{
		ID:          "ev-1",
		Fingerprint: "fp-agy-work-gemini-5h-warning",
		EventType:   "quota.threshold_crossed",
		Provider:    "agy",
		Profile:     "work",
		QuotaGroup:  "gemini",
		WindowKind:  "5h",
		Severity:    "WARNING",
		PayloadJSON: `{"remaining": 15.0}`,
		EmittedAt:   now,
	}
	if err := s.PutCapacityEvent(ev); err != nil {
		t.Fatalf("put capacity event: %v", err)
	}

	foundEv, err := s.FindCapacityEvent("fp-agy-work-gemini-5h-warning")
	if err != nil || foundEv == nil || foundEv.ID != "ev-1" {
		t.Fatalf("find capacity event: %v", err)
	}

	// 5. Delivery Receipts
	rec := DeliveryReceipt{
		AlertID:     "ev-1",
		Channel:     "native",
		Fingerprint: "fp-agy-work-gemini-5h-warning",
		Status:      "DELIVERED",
		DeliveredAt: now,
	}
	if err := s.PutDeliveryReceipt(rec); err != nil {
		t.Fatalf("put delivery receipt: %v", err)
	}

	// 6. Preferences
	prefs := CapacityPreferences{
		Enabled:                     true,
		BackgroundEnabled:           false,
		WarningThreshold:            20,
		CriticalThreshold:           10,
		ExhaustedThreshold:          0,
		RecoveryThreshold:           25,
		ReminderIntervalSeconds:     3600,
		QuietHoursEnabled:           true,
		QuietHoursStartUTC:          "22:00",
		QuietHoursEndUTC:            "08:00",
		ExhaustedBypassQuietHours:   true,
		BrowserNotificationsEnabled: true,
		NativeNotificationsEnabled:  true,
		LockScreenSafeCopy:          true,
		UpdatedAt:                   now,
	}
	if err := s.PutCapacityPreferences(prefs); err != nil {
		t.Fatalf("put preferences: %v", err)
	}
	gotPrefs, err := s.GetCapacityPreferences()
	if err != nil {
		t.Fatalf("get preferences: %v", err)
	}
	if !gotPrefs.Enabled || gotPrefs.WarningThreshold != 20 || !gotPrefs.LockScreenSafeCopy {
		t.Fatalf("unexpected preferences: %+v", gotPrefs)
	}

	// 7. Pruning 30 days
	oldTime := now.Add(-31 * 24 * time.Hour)
	oldEv := CapacityEventRecord{
		ID:          "ev-old",
		Fingerprint: "fp-old",
		EventType:   "quota.threshold_crossed",
		Provider:    "agy",
		Profile:     "work",
		QuotaGroup:  "gemini",
		WindowKind:  "5h",
		Severity:    "WARNING",
		PayloadJSON: `{}`,
		EmittedAt:   oldTime,
	}
	if err := s.PutCapacityEvent(oldEv); err != nil {
		t.Fatalf("put old event: %v", err)
	}
	if err := s.PruneCapacityHistory(now.Add(-30 * 24 * time.Hour)); err != nil {
		t.Fatalf("prune history: %v", err)
	}
	if ev, _ := s.FindCapacityEvent("fp-old"); ev != nil {
		t.Fatal("old event should have been pruned")
	}
	// Recent event and observation must remain
	if ev, _ := s.FindCapacityEvent("fp-agy-work-gemini-5h-warning"); ev == nil {
		t.Fatal("recent event must be retained")
	}
	if list, _ := s.ListCapacityObservations(); len(list) != 1 {
		t.Fatal("latest observation must be retained")
	}
}
