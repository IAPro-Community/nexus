package store

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCapacityMigrationApplied(t *testing.T) {
	s := openTestStore(t)
	v, err := s.SchemaVersion()
	if err != nil {
		t.Fatalf("SchemaVersion() = %v", err)
	}
	if v < 21 {
		t.Fatalf("schema version = %d, want >= 21 (capacity monitor)", v)
	}
	for _, table := range []string{
		"capacity_observations", "capacity_states", "capacity_monitor_lease",
		"capacity_events", "capacity_delivery_receipts", "capacity_recommendations",
		"capacity_preferences",
	} {
		var name string
		if err := s.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
}

func TestCapacityPreferencesDefaultsAndRoundTrip(t *testing.T) {
	s := openTestStore(t)
	got, err := s.GetCapacityPreferences()
	if err != nil {
		t.Fatalf("GetCapacityPreferences() = %v", err)
	}
	want := DefaultCapacityPreferences()
	if !got.Enabled || got.BackgroundEnabled {
		t.Fatalf("defaults = enabled:%v background:%v, want true/false", got.Enabled, got.BackgroundEnabled)
	}
	if got.WarningThreshold != want.WarningThreshold || got.CriticalThreshold != want.CriticalThreshold {
		t.Fatalf("default thresholds = %.1f/%.1f, want %.1f/%.1f",
			got.WarningThreshold, got.CriticalThreshold, want.WarningThreshold, want.CriticalThreshold)
	}

	custom := want
	custom.BackgroundEnabled = true
	custom.WarningThreshold = 30
	custom.QuietHoursEnabled = true
	if err := s.PutCapacityPreferences(custom); err != nil {
		t.Fatalf("PutCapacityPreferences() = %v", err)
	}
	got, err = s.GetCapacityPreferences()
	if err != nil {
		t.Fatalf("GetCapacityPreferences() after put = %v", err)
	}
	if !got.BackgroundEnabled || got.WarningThreshold != 30 || !got.QuietHoursEnabled {
		t.Fatalf("round-trip prefs mismatch: %+v", got)
	}
}

func TestCapacityObservationUpsertAndList(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	rem := 42.5
	rec := ObservationRecord{
		Provider: "codex", Profile: "kiver", QuotaGroup: "claude_gpt", WindowKind: "5h",
		Status: "LIVE", Source: "OFFICIAL_API",
		RemainingPercent: &rem, FetchedAt: now, ResetDescription: "resets 01:00",
	}
	if err := s.UpsertCapacityObservation(rec); err != nil {
		t.Fatalf("UpsertCapacityObservation() = %v", err)
	}
	// Re-upsert same key with new values must overwrite, not duplicate.
	rem2 := 10.0
	rec.RemainingPercent = &rem2
	rec.Status = "RATE_LIMITED"
	if err := s.UpsertCapacityObservation(rec); err != nil {
		t.Fatalf("re-upsert = %v", err)
	}
	all, err := s.ListCapacityObservations()
	if err != nil {
		t.Fatalf("ListCapacityObservations() = %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("observations = %d, want 1 (upsert, not insert)", len(all))
	}
	got := all[0]
	if got.Status != "RATE_LIMITED" || got.RemainingPercent == nil || *got.RemainingPercent != 10.0 {
		t.Fatalf("observation not overwritten: %+v", got)
	}
	if got.FetchedAt.Unix() != now.Unix() {
		t.Fatalf("fetched_at = %v, want %v", got.FetchedAt, now)
	}
}

func TestCapacityStateUpsertGetList(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.GetCapacityState("nope"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing key err = %v, want sql.ErrNoRows", err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	rec := CapacityStateRecord{
		Key: "codex:kiver:claude_gpt:5h", Provider: "codex", Profile: "kiver",
		QuotaGroup: "claude_gpt", WindowKind: "5h",
		State: "WARNING", Reason: "crossed 20%", TransitionedAt: now, UpdatedAt: now,
	}
	if err := s.UpsertCapacityState(rec); err != nil {
		t.Fatalf("UpsertCapacityState() = %v", err)
	}
	got, err := s.GetCapacityState(rec.Key)
	if err != nil {
		t.Fatalf("GetCapacityState() = %v", err)
	}
	if got.State != "WARNING" || got.Reason != "crossed 20%" {
		t.Fatalf("state mismatch: %+v", got)
	}
	rec.State = "CRITICAL"
	if err := s.UpsertCapacityState(rec); err != nil {
		t.Fatalf("re-upsert state = %v", err)
	}
	all, err := s.ListCapacityStates()
	if err != nil {
		t.Fatalf("ListCapacityStates() = %v", err)
	}
	if len(all) != 1 || all[0].State != "CRITICAL" {
		t.Fatalf("states = %+v, want single CRITICAL", all)
	}
}

func TestCapacityLeaseLifecycle(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC()
	if cur, err := s.GetCapacityLease(); err != nil || cur != nil {
		t.Fatalf("initial lease = %+v,%v, want nil,nil", cur, err)
	}
	lease, held, err := s.AcquireCapacityLease("owner-a", now, time.Minute)
	if err != nil || !held {
		t.Fatalf("first acquire = %v,%v, want lease,true", lease, held)
	}
	if lease.Owner != "owner-a" {
		t.Fatalf("owner = %q, want owner-a", lease.Owner)
	}
	// Live lease owned by someone else must not be stolen.
	if _, held, err := s.AcquireCapacityLease("owner-b", now.Add(10*time.Second), time.Minute); err != nil || held {
		t.Fatalf("steal attempt = held:%v err:%v, want false,nil", held, err)
	}
	// Expired lease can be taken over.
	taken, held, err := s.AcquireCapacityLease("owner-b", now.Add(2*time.Minute), time.Minute)
	if err != nil || !held || taken.Owner != "owner-b" {
		t.Fatalf("takeover = %+v,%v,%v, want owner-b,true,nil", taken, held, err)
	}
	// Renewal by non-owner must fail.
	if err := s.RenewCapacityLease("owner-a", now.Add(3*time.Minute), time.Minute); err == nil {
		t.Fatal("expected renewal error for non-owner, got nil")
	}
	if err := s.RenewCapacityLease("owner-b", now.Add(3*time.Minute), time.Minute); err != nil {
		t.Fatalf("renewal by owner = %v, want nil", err)
	}
	// Release by non-owner is a no-op; release by owner frees.
	if err := s.ReleaseCapacityLease("owner-a"); err != nil {
		t.Fatalf("release by non-owner = %v, want nil", err)
	}
	if cur, _ := s.GetCapacityLease(); cur == nil || cur.Owner != "owner-b" {
		t.Fatalf("lease after foreign release = %+v, want owner-b", cur)
	}
	if err := s.ReleaseCapacityLease("owner-b"); err != nil {
		t.Fatalf("release by owner = %v, want nil", err)
	}
	if cur, err := s.GetCapacityLease(); err != nil || cur != nil {
		t.Fatalf("lease after release = %+v,%v, want nil,nil", cur, err)
	}
	// Non-positive TTLs fail fast instead of writing instantly-dead leases.
	if _, _, err := s.AcquireCapacityLease("x", time.Now().UTC(), 0); err == nil {
		t.Fatal("expected error for zero acquire TTL, got nil")
	}
	if err := s.RenewCapacityLease("x", time.Now().UTC(), -time.Second); err == nil {
		t.Fatal("expected error for negative renew TTL, got nil")
	}
}

func TestPruneCapacityHistory(t *testing.T) {
	s := openTestStore(t)
	if err := s.PruneCapacityHistory(time.Now().UTC()); err != nil {
		t.Fatalf("prune empty = %v, want nil", err)
	}
	// Fixed-width seeds (the storage invariant): variable-width Go layouts
	// would reintroduce the second-boundary ordering hazard under test.
	old := formatStoreTime(time.Now().UTC().Add(-48 * time.Hour))
	fresh := formatStoreTime(time.Now().UTC())
	for id, ts := range map[string]string{"old-1": old, "new-1": fresh} {
		if _, err := s.db.Exec(`INSERT INTO capacity_events(id, fingerprint, event_type, provider, profile, severity, payload_json, emitted_at)
			VALUES(?, 'fp', 'QUOTA_LOW', 'codex', 'kiver', 'warning', '{}', ?)`, id, ts); err != nil {
			t.Fatalf("seed event %s: %v", id, err)
		}
	}
	for id, ts := range map[string]string{"old-r": old, "new-r": fresh} {
		if _, err := s.db.Exec(`INSERT INTO capacity_delivery_receipts(alert_id, channel, fingerprint, status, delivered_at)
			VALUES(?, 'native', 'fp', 'ok', ?)`, id, ts); err != nil {
			t.Fatalf("seed receipt %s: %v", id, err)
		}
	}
	if err := s.PruneCapacityHistory(time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatalf("prune = %v", err)
	}
	var remaining int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM capacity_events`).Scan(&remaining); err != nil {
		t.Fatalf("count: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("remaining events = %d, want 1 (fresh only)", remaining)
	}
	var id string
	if err := s.db.QueryRow(`SELECT id FROM capacity_events`).Scan(&id); err != nil || id != "new-1" {
		t.Fatalf("remaining id = %q,%v, want new-1", id, err)
	}
	var receipts int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM capacity_delivery_receipts`).Scan(&receipts); err != nil {
		t.Fatalf("receipt count: %v", err)
	}
	if receipts != 1 {
		t.Fatalf("remaining receipts = %d, want 1 (fresh only)", receipts)
	}
}

func TestCapacityLeaseConcurrentTakeover(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC()
	if _, held, err := s.AcquireCapacityLease("holder", now.Add(-time.Hour), time.Minute); err != nil || !held {
		t.Fatalf("seed expired lease: %v,%v", held, err)
	}
	const racers = 8
	var wg sync.WaitGroup
	wins := make(chan bool, racers)
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, held, err := s.AcquireCapacityLease(fmt.Sprintf("racer-%d", n), now, time.Minute)
			if err != nil {
				t.Errorf("racer %d error: %v (transient backend errors fail loudly, never silently lost)", n, err)
				return
			}
			wins <- held
		}(i)
	}
	wg.Wait()
	close(wins)
	count := 0
	for w := range wins {
		if w {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("takeover winners = %d, want exactly 1 (atomic lease)", count)
	}
}

func TestCapacityPreferencesCorruptRowErrors(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.db.Exec(`INSERT INTO capacity_preferences(id, enabled, updated_at)
		VALUES(1, 1, 'not-a-time')`); err != nil {
		t.Fatalf("seed corrupt prefs: %v", err)
	}
	if _, err := s.GetCapacityPreferences(); err == nil {
		t.Fatal("expected error for corrupt prefs timestamp, got nil")
	}
}

func TestCapacityLeaseCorruptRowErrors(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.db.Exec(`INSERT INTO capacity_monitor_lease(id, owner, acquired_at, renewed_at, expires_at)
		VALUES(1, 'x', 'not-a-time', 'not-a-time', 'not-a-time')`); err != nil {
		t.Fatalf("seed corrupt row: %v", err)
	}
	if _, err := s.GetCapacityLease(); err == nil {
		t.Fatal("expected error for corrupt lease timestamps, got nil (never zero-time silent)")
	}
}

func TestCapacityStateCorruptRow(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.db.Exec(`INSERT INTO capacity_states(key, provider, profile, state, transitioned_at, updated_at)
		VALUES('k', 'p', 'q', 'HEALTHY', 'not-a-time', 'not-a-time')`); err != nil {
		t.Fatalf("seed corrupt row: %v", err)
	}
	// Decisions fail loudly...
	if _, err := s.GetCapacityState("k"); err == nil {
		t.Fatal("expected error for corrupt state timestamps, got nil")
	}
	// ...while diagnostics skip the row instead of blanking the listing.
	all, err := s.ListCapacityStates()
	if err != nil {
		t.Fatalf("list with corrupt row = %v, want skip-not-fail", err)
	}
	if len(all) != 0 {
		t.Fatalf("listed = %d corrupt rows, want 0", len(all))
	}
}

func TestStoreTimeFixedWidthOrders(t *testing.T) {
	// Pins the storage invariant: fixed-width stamps order lexicographically
	// exactly like chronologically, including the exact-second edge where
	// variable-width layouts break (missing fraction sorts after '.').
	sec := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := formatStoreTime(sec)
	b := formatStoreTime(sec.Add(250 * time.Millisecond))
	c := formatStoreTime(sec.Add(500 * time.Millisecond))
	if a >= b || b >= c {
		t.Fatalf("ordering broken: %q < %q < %q", a, b, c)
	}
	for _, s := range []string{a, b, c} {
		if len(s) != len("2026-01-01T00:00:00.000000000Z") {
			t.Fatalf("stamp %q is not fixed width", s)
		}
	}
	if rt, err := parseStoreTime(a); err != nil || !rt.Equal(sec) {
		t.Fatalf("round-trip = %v,%v, want %v", rt, err, sec)
	}
}

func TestCapacityScopeRoundTrip(t *testing.T) {
	s := openTestStore(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	if err := s.UpsertCapacityObservation(ObservationRecord{
		Provider: "codex", Profile: "kiver", QuotaGroup: "g", WindowKind: "5h",
		Status: "LIVE", Source: "OFFICIAL_API", FetchedAt: now,
		AccountScope: "codex/acct-1/v3",
	}); err != nil {
		t.Fatalf("upsert observation: %v", err)
	}
	if err := s.UpsertCapacityState(CapacityStateRecord{
		Key: "codex|kiver|g|5h", Provider: "codex", Profile: "kiver",
		QuotaGroup: "g", WindowKind: "5h", State: "HEALTHY",
		TransitionedAt: now, UpdatedAt: now, AccountScope: "codex/acct-1/v3",
	}); err != nil {
		t.Fatalf("upsert state: %v", err)
	}
	obs, err := s.ListCapacityObservations()
	if err != nil || len(obs) != 1 || obs[0].AccountScope != "codex/acct-1/v3" {
		t.Fatalf("observation scope = %+v,%v, want codex/acct-1/v3", obs, err)
	}
	st, err := s.GetCapacityState("codex|kiver|g|5h")
	if err != nil || st.AccountScope != "codex/acct-1/v3" {
		t.Fatalf("state scope = %+v,%v, want codex/acct-1/v3", st, err)
	}
	// Re-auth under a new identity refreshes the attribute instead of leaving
	// stale attribution behind.
	if err := s.UpsertCapacityState(CapacityStateRecord{
		Key: "codex|kiver|g|5h", Provider: "codex", Profile: "kiver",
		QuotaGroup: "g", WindowKind: "5h", State: "HEALTHY",
		TransitionedAt: now, UpdatedAt: now, AccountScope: "codex/acct-9/v1",
	}); err != nil {
		t.Fatalf("re-upsert state: %v", err)
	}
	st, err = s.GetCapacityState("codex|kiver|g|5h")
	if err != nil || st.AccountScope != "codex/acct-9/v1" {
		t.Fatalf("state scope after re-auth = %+v,%v, want codex/acct-9/v1", st, err)
	}
}
