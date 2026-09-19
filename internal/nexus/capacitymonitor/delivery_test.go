package capacitymonitor

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func stubLastEvent(fp string, at time.Time) *store.CapacityEventRecord {
	return &store.CapacityEventRecord{
		ID: "evt-last", Fingerprint: fp, EventType: "QUOTA_LOW",
		Provider: "codex", Profile: "kiver",
		Severity:  "WARNING",
		EmittedAt: at,
	}
}

func TestResetCycleKeyPrefersWindowReset(t *testing.T) {
	reset := time.Date(2026, 9, 20, 1, 0, 0, 0, time.UTC)
	exp := reset.Add(2 * time.Hour)
	if got, want := ResetCycleKey(&reset, exp), ResetCycleKey(&reset, time.Time{}); got != want || got == "" {
		t.Fatalf("window reset must win over expiry: %q vs %q", got, want)
	}
}

func TestResetCycleKeyFallsBackToExpiry(t *testing.T) {
	exp := time.Date(2026, 9, 20, 1, 0, 0, 0, time.UTC)
	if got := ResetCycleKey(nil, exp); got == "" {
		t.Fatal("expiry must produce a non-empty reset cycle")
	}
	if a, b := ResetCycleKey(nil, exp), ResetCycleKey(nil, exp.Add(time.Hour)); a == b {
		t.Fatal("different expiries must produce different reset cycles (janela nova)")
	}
}

func TestResetCycleKeyEmptyWithoutTimeEvidence(t *testing.T) {
	// resetDesc propositalmente IGNORADO: é texto de display e muda a cada
	// refresh ("2h 00m" -> "1h 59m") — usá-lo quebraria o dedup
	// (espelha QuotaDropMonitor.checkWindow). Sem evidência temporal, o ciclo
	// é desconhecido ("") e o dedup cai para o estado.
	if got := ResetCycleKey(nil, time.Time{}); got != "" {
		t.Fatalf("cycle without time evidence = %q, want empty", got)
	}
	a := DeriveFingerprint("codex", "kiver", "g", "5h", "WARNING", ResetCycleKey(nil, time.Time{}))
	b := DeriveFingerprint("codex", "kiver", "g", "5h", "WARNING", ResetCycleKey(nil, time.Time{}))
	if a != b {
		t.Fatal("same condition must fingerprint identically")
	}
}

func TestShouldDeliverFirstSight(t *testing.T) {
	d := ShouldDeliver(StateWarning, "fp-1", nil, time.Now().UTC(), time.Hour, false)
	if !d.Deliver {
		t.Fatalf("first sight must deliver: %+v", d)
	}
}

func TestShouldDeliverSuppressesDuplicate(t *testing.T) {
	now := time.Now().UTC()
	last := stubLastEvent("fp-1", now.Add(-time.Minute))
	d := ShouldDeliver(StateWarning, "fp-1", last, now, time.Hour, false)
	if d.Deliver {
		t.Fatalf("same-condition repeat must suppress (sem WARNING x4): %+v", d)
	}
}

func TestShouldDeliverReAlertsOnNewFingerprint(t *testing.T) {
	now := time.Now().UTC()
	// Mudança de estado, janela nova, recurso diferente e recovery produzem
	// fingerprints distintos (severity/resetCycle/recurso) — todos re-alertam.
	last := stubLastEvent("fp-old", now.Add(-time.Minute))
	for name, fp := range map[string]string{
		"state change": "fp-critical",
		"new window":   "fp-newwindow",
		"new resource": "fp-other",
		"recovery":     "fp-recovered",
	} {
		if d := ShouldDeliver(StateCritical, fp, last, now, time.Hour, false); !d.Deliver {
			t.Fatalf("%s must re-alert: %+v", name, d)
		}
	}
}

func TestShouldDeliverReminderRepeat(t *testing.T) {
	now := time.Now().UTC()
	last := stubLastEvent("fp-1", now.Add(-2*time.Hour))
	if d := ShouldDeliver(StateWarning, "fp-1", last, now, time.Hour, false); !d.Deliver {
		t.Fatalf("elapsed reminder interval must repeat: %+v", d)
	}
}

func TestShouldDeliverExplicitRepeat(t *testing.T) {
	now := time.Now().UTC()
	last := stubLastEvent("fp-1", now.Add(-time.Minute))
	if d := ShouldDeliver(StateWarning, "fp-1", last, now, time.Hour, true); !d.Deliver {
		t.Fatalf("explicit repeat must deliver: %+v", d)
	}
}
