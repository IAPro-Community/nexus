package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNexProactivityClaimDedupeCooldownQuietHoursAndSnooze(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "nex-proactive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	p, err := db.GetNexProactivityPreferences("")
	if err != nil || p.Mode != "manual" {
		t.Fatalf("manual default: %+v %v", p, err)
	}
	claimed, err := db.TryClaimNexSuggestion("", "mission-a:blocked", now)
	if err != nil || claimed {
		t.Fatalf("manual cannot claim: %v %v", claimed, err)
	}
	p = NexProactivityPreferences{Mode: "proactive", QuietStart: "22:00", QuietEnd: "08:00", TimeZone: "UTC", MinIntervalMinutes: 30}
	if err := db.SetNexProactivityPreferences("", p); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		t     time.Time
		quiet bool
	}{
		{time.Date(2026, 11, 17, 23, 0, 0, 0, time.UTC), true},
		{time.Date(2026, 11, 18, 7, 59, 0, 0, time.UTC), true},
		{now, false},
	} {
		if quiet := NexQuietHours(p, tc.t); quiet != tc.quiet {
			t.Fatalf("quiet hours for %v = %v, expected %v", tc.t, quiet, tc.quiet)
		}
	}
	if claimed, err = db.TryClaimNexSuggestion("", "mission-a:blocked", now); err != nil || !claimed {
		t.Fatalf("first claim: %v %v", claimed, err)
	}
	if claimed, err = db.TryClaimNexSuggestion("", "mission-a:blocked", now.Add(time.Minute)); err != nil || claimed {
		t.Fatalf("duplicate claim: %v %v", claimed, err)
	}
	if claimed, err = db.TryClaimNexSuggestion("", "mission-b:blocked", now.Add(10*time.Minute)); err != nil || claimed {
		t.Fatalf("cooldown: %v %v", claimed, err)
	}
	if err := db.SetNexSuggestionDecision("", "mission-a:blocked", "SNOOZED", 90, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if claimed, err = db.TryClaimNexSuggestion("", "mission-a:blocked", now.Add(31*time.Minute)); err != nil || claimed {
		t.Fatalf("snooze should block: %v %v", claimed, err)
	}
	if claimed, err = db.TryClaimNexSuggestion("", "mission-a:blocked", now.Add(100*time.Minute)); err != nil || !claimed {
		t.Fatalf("expired snooze should allow reissue: %v %v", claimed, err)
	}
	if err := db.SetNexSuggestionDecision("", "mission-a:blocked", "DISMISSED", 0, now.Add(101*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if claimed, err = db.TryClaimNexSuggestion("", "mission-a:blocked", now.Add(24*time.Hour)); err != nil || claimed {
		t.Fatalf("dismiss should remain closed: %v %v", claimed, err)
	}
}

func TestNexProactivityRejectsInvalidSettingsAndCrossScopeClaims(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "nex-proactive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	p := NexProactivityPreferences{Mode: "proactive", QuietStart: "25:00", QuietEnd: "08:00", TimeZone: "UTC", MinIntervalMinutes: 5}
	if err := db.SetNexProactivityPreferences("", p); err == nil {
		t.Fatal("invalid clock accepted")
	}
	p.QuietStart = "22:00"
	p.TimeZone = "invalid/timezone"
	if err := db.SetNexProactivityPreferences("", p); err == nil {
		t.Fatal("unknown timezone accepted")
	}
	p.TimeZone = "UTC"
	if err := db.SetNexProactivityPreferences("", p); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	if claimed, err := db.TryClaimNexSuggestion("", "alert:rate-limit", now); err != nil || !claimed {
		t.Fatalf("claim global: %v %v", claimed, err)
	}
	if err := db.SetNexSuggestionDecision("different-project", "alert:rate-limit", "DISMISSED", 0, now); err == nil {
		t.Fatal("cross-project decision must be rejected")
	}
}
