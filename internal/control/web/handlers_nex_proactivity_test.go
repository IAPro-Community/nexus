package web

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestNexProactiveCandidatesAreGroundedAndStable(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	until := now.Add(10 * time.Minute)
	hud := nexus.HUDState{
		ProjectID: "a",
		Missions: []nexus.HUDMission{
			{RunID: "run1", ProjectID: "a", State: "BLOCKED_NEEDS_USER",
				CurrentTaskID: "task1", NeedsUser: true, UpdatedAt: now},
			{RunID: "run2", ProjectID: "b", NeedsUser: true},
			{RunID: "run3", ProjectID: "a", NeedsUser: false},
		},
		Alerts: []nexus.HUDAlert{
			{Fingerprint: "alert1", LastObservedAt: now},
			{Fingerprint: "alert2", Acked: true},
			{Fingerprint: "alert3", SnoozedUntil: &until},
		},
	}
	got := nexProactiveCandidates(hud, now)
	if len(got) != 2 || got[0].Target != "missions" || got[1].Target != "usage" {
		t.Fatalf("incorrect proposals: %+v", got)
	}
	if got[0].Fingerprint != nexProactiveCandidates(hud, now.Add(time.Minute))[0].Fingerprint {
		t.Fatal("unchanged event changed fingerprint")
	}
	if nexCandidateFingerprint("a", "decision", "run1") == nexCandidateFingerprint("b", "decision", "run1") {
		t.Fatal("cross-project fingerprint collision")
	}
	if got := nexProactiveCandidates(nexus.HUDState{}, now); len(got) != 0 {
		t.Fatal("empty HUD must not fabricate opportunities")
	}
}
