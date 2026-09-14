package nexus

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

func TestQuotaDropMonitor_StepDownAndSuppressionAtZero(t *testing.T) {
	rec := &notify.Recorder{}
	bus := events.NewBus(100)
	monitor := NewQuotaDropMonitor(rec, bus, 0.30, 0.05)

	acc := ProviderAccount{
		Provider:       "claude",
		Profile:        "work",
		DisplayName:    "Claude Pro Work",
		QuotaRemaining: 0.80,
		QuotaTotal:     1.0,
		QuotaView: &quota.QuotaView{
			Status: "LIVE",
			ModelGroups: []quota.ModelGroup{
				{
					Windows: []quota.Window{
						{ResetDesc: "3h 15m"},
					},
				},
			},
		},
	}

	// 1. Initial healthy quota (80%) -> No notification
	act := monitor.CheckAccount(acc)
	if act != nil {
		t.Fatalf("expected nil action for healthy quota, got %+v", act)
	}
	if len(rec.Payloads) != 0 {
		t.Fatalf("expected 0 notifications, got %d", len(rec.Payloads))
	}

	// 2. Drop to 30% (reaches lowThreshold) -> Must notify
	acc.QuotaRemaining = 0.30
	act = monitor.CheckAccount(acc)
	if act == nil || act.Kind != string(events.EventQuotaLow) {
		t.Fatalf("expected QUOTA_LOW at 30%%, got %+v", act)
	}
	if act.RemainingPercent != 30 {
		t.Fatalf("expected RemainingPercent 30, got %v", act.RemainingPercent)
	}
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected 1 notification in recorder, got %d", len(rec.Payloads))
	}

	// 3. Tiny drop to 28% (< 5% delta) -> Anti-spam must suppress
	acc.QuotaRemaining = 0.28
	act = monitor.CheckAccount(acc)
	if act != nil {
		t.Fatalf("expected suppression for tiny drop from 30%% to 28%%, got %+v", act)
	}
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected recorder count to stay at 1, got %d", len(rec.Payloads))
	}

	// 4. Significant drop to 20% (milestone crossed and delta >= 5%) -> Must notify
	acc.QuotaRemaining = 0.20
	act = monitor.CheckAccount(acc)
	if act == nil || act.Kind != string(events.EventQuotaLow) {
		t.Fatalf("expected QUOTA_LOW at 20%%, got %+v", act)
	}
	if act.RemainingPercent != 20 {
		t.Fatalf("expected RemainingPercent 20, got %v", act.RemainingPercent)
	}
	if len(rec.Payloads) != 2 {
		t.Fatalf("expected 2 notifications in recorder, got %d", len(rec.Payloads))
	}

	// 5. Drop to 10% -> Must notify
	acc.QuotaRemaining = 0.10
	act = monitor.CheckAccount(acc)
	if act == nil || act.Kind != string(events.EventQuotaLow) {
		t.Fatalf("expected QUOTA_LOW at 10%%, got %+v", act)
	}
	if act.RemainingPercent != 10 {
		t.Fatalf("expected RemainingPercent 10, got %v", act.RemainingPercent)
	}
	if len(rec.Payloads) != 3 {
		t.Fatalf("expected 3 notifications in recorder, got %d", len(rec.Payloads))
	}

	// 6. Drop to 5% -> Must notify
	acc.QuotaRemaining = 0.05
	act = monitor.CheckAccount(acc)
	if act == nil || act.Kind != string(events.EventQuotaLow) {
		t.Fatalf("expected QUOTA_LOW at 5%%, got %+v", act)
	}
	if len(rec.Payloads) != 4 {
		t.Fatalf("expected 4 notifications in recorder, got %d", len(rec.Payloads))
	}

	// 7. Drop to 0% -> Must notify QUOTA_EXHAUSTED
	acc.QuotaRemaining = 0.0
	act = monitor.CheckAccount(acc)
	if act == nil || act.Kind != string(events.EventQuotaExhausted) {
		t.Fatalf("expected QUOTA_EXHAUSTED at 0%%, got %+v", act)
	}
	if act.RemainingPercent != 0 {
		t.Fatalf("expected RemainingPercent 0, got %v", act.RemainingPercent)
	}
	if len(rec.Payloads) != 5 {
		t.Fatalf("expected 5 notifications in recorder, got %d", len(rec.Payloads))
	}

	// 8. Repeated checks at 0% -> MUST NOT NOTIFY ANY MORE ("e não avisar mais")
	for i := 0; i < 5; i++ {
		act = monitor.CheckAccount(acc)
		if act != nil {
			t.Fatalf("iteration %d: expected SILENCE after 0%% reached, got %+v", i, act)
		}
	}
	if len(rec.Payloads) != 5 {
		t.Fatalf("expected recorder count to strictly remain 5, got %d", len(rec.Payloads))
	}

	// 9. Quota Reset / Recovery to 100% -> Auto-Rearm
	acc.QuotaRemaining = 1.0
	act = monitor.CheckAccount(acc)
	if act != nil {
		t.Fatalf("expected nil when resetting quota to 1.0, got %+v", act)
	}

	// 10. After reset, dropping to 25% must notify again!
	acc.QuotaRemaining = 0.25
	act = monitor.CheckAccount(acc)
	if act == nil || act.Kind != string(events.EventQuotaLow) {
		t.Fatalf("expected monitor to re-arm and notify at 25%% after recovery, got %+v", act)
	}
	if len(rec.Payloads) != 6 {
		t.Fatalf("expected 6 total notifications after re-arm, got %d", len(rec.Payloads))
	}
}

func TestQuotaDropMonitor_UnknownQuotaIgnored(t *testing.T) {
	rec := &notify.Recorder{}
	bus := events.NewBus(100)
	monitor := NewQuotaDropMonitor(rec, bus, 0.30, 0.05)

	acc := ProviderAccount{
		Provider:       "unknown-provider",
		Profile:        "default",
		QuotaRemaining: 0,
		QuotaTotal:     1.0,
		QuotaView: &quota.QuotaView{
			Status: "UNKNOWN",
		},
	}

	act := monitor.CheckAccount(acc)
	if act != nil {
		t.Fatalf("expected nil for UNKNOWN quota status, got %+v", act)
	}
	if len(rec.Payloads) != 0 {
		t.Fatalf("expected 0 notifications, got %d", len(rec.Payloads))
	}
}

func TestQuotaDropMonitor_AlertsIndependentModelGroups(t *testing.T) {
	rec := &notify.Recorder{}
	bus := events.NewBus(20)
	monitor := NewQuotaDropMonitor(rec, bus, 0.30, 0.05)
	gemini := 28.0
	claude := 100.0
	view := &quota.QuotaView{Status: "LIVE", FetchedAt: time.Now(), ModelGroups: []quota.ModelGroup{
		{Key: "gemini", Windows: []quota.Window{{Kind: "weekly", Remaining: gemini, ResetDesc: "monday"}}},
		{Key: "claude_gpt", Windows: []quota.Window{{Kind: "weekly", Remaining: claude, ResetDesc: "monday"}}},
	}}
	act := monitor.CheckAccount(ProviderAccount{Provider: "agy", Profile: "work", QuotaView: view, QuotaTotal: 1, QuotaRemaining: 1})
	if act == nil || act.Group != "gemini" || act.Window != "weekly" || act.RemainingPercent != 28 {
		t.Fatalf("expected independent Gemini alert at 28%%, got %+v", act)
	}
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected one notification, got %d", len(rec.Payloads))
	}
	if got := len(bus.GetHistory("", 10)); got != 1 {
		t.Fatalf("expected global history, got %d", got)
	}
}

func TestQuotaDropMonitor_DoesNotRepeatWhenResetCountdownChanges(t *testing.T) {
	rec := &notify.Recorder{}
	monitor := NewQuotaDropMonitor(rec, events.NewBus(20), 0.30, 0.05)
	view := &quota.QuotaView{
		Status: "LIVE",
		ModelGroups: []quota.ModelGroup{{
			Key:     "gemini",
			Windows: []quota.Window{{Kind: "weekly", Remaining: 22, ResetDesc: "2h 00m"}},
		}},
	}
	account := ProviderAccount{Provider: "agy", Profile: "work", QuotaView: view}

	if action := monitor.CheckAccount(account); action == nil {
		t.Fatal("expected the first low-quota observation to notify")
	}

	// The provider's countdown changes, but this is still the same quota
	// window and must remain suppressed until a meaningful drop occurs.
	view.ModelGroups[0].Windows[0].ResetDesc = "1h 59m"
	if action := monitor.CheckAccount(account); action != nil {
		t.Fatalf("expected changed reset countdown to stay suppressed, got %+v", action)
	}
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected one notification, got %d", len(rec.Payloads))
	}
}

func TestQuotaDropMonitor_RateLimitedTriggersExhaustedOnce(t *testing.T) {
	rec := &notify.Recorder{}
	bus := events.NewBus(100)
	monitor := NewQuotaDropMonitor(rec, bus, 0.30, 0.05)

	acc := ProviderAccount{
		Provider:       "codex",
		Profile:        "test",
		RateLimited:    true,
		QuotaRemaining: 0.20,
		QuotaTotal:     1.0,
		QuotaView: &quota.QuotaView{
			Status: "RATE_LIMITED",
		},
	}

	act := monitor.CheckAccount(acc)
	if act == nil || act.Kind != string(events.EventQuotaExhausted) {
		t.Fatalf("expected QUOTA_EXHAUSTED for RATE_LIMITED account, got %+v", act)
	}

	// Second check must be suppressed
	act2 := monitor.CheckAccount(acc)
	if act2 != nil {
		t.Fatalf("expected suppression for subsequent checks, got %+v", act2)
	}
}

func TestQuotaDropMonitor_FailoverRecommendationAndAffectedRuntime(t *testing.T) {
	rec := &notify.Recorder{}
	bus := events.NewBus(100)
	monitor := NewQuotaDropMonitor(rec, bus, 0.30, 0.05)

	exhaustedAcc := ProviderAccount{
		Provider:       "claude",
		Profile:        "primary",
		DisplayName:    "Claude Primary",
		QuotaRemaining: 0.0,
		QuotaTotal:     1.0,
		QuotaView: &quota.QuotaView{
			Status: "OK",
		},
	}

	healthyAcc := ProviderAccount{
		Provider:       "codex",
		Profile:        "work",
		DisplayName:    "Codex Work",
		Available:      true,
		Authenticated:  true,
		Health:         "healthy",
		QuotaRemaining: 0.85,
		QuotaTotal:     1.0,
		Capabilities:   map[string]string{"headless": "supported", "submit_prompt": "supported"},
		QuotaView: &quota.QuotaView{
			Status: "OK",
		},
	}

	actions := monitor.CheckAccounts([]ProviderAccount{exhaustedAcc, healthyAcc})
	if len(actions) != 1 {
		t.Fatalf("expected 1 action for exhausted account, got %d", len(actions))
	}

	act := actions[0]
	if act.Kind != string(events.EventQuotaExhausted) {
		t.Fatalf("expected QUOTA_EXHAUSTED, got %s", act.Kind)
	}
	if act.RecommendedProvider != "codex" || act.RecommendedProfile != "work" {
		t.Fatalf("expected recommended provider codex:work, got %s:%s", act.RecommendedProvider, act.RecommendedProfile)
	}
	if act.RecommendedDisplayName != "Codex Work" {
		t.Fatalf("expected display name 'Codex Work', got %s", act.RecommendedDisplayName)
	}
}
