package capacitymonitor

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

type fakeChannel struct {
	mu        sync.Mutex
	delivered []CanonicalAlert
	err       error
}

func (c *fakeChannel) Deliver(ctx context.Context, alert CanonicalAlert) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	c.delivered = append(c.delivered, alert)
	return nil
}

func TestDeliveryRouterDeliversAndIsolatesFailures(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nexus_delivery_test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	ch1 := &fakeChannel{}
	ch2Failed := &fakeChannel{err: errors.New("native channel timeout")}

	router := NewDeliveryRouter(st, map[string]Channel{
		"in_app": ch1,
		"native": ch2Failed,
	})

	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	rem := 10.0
	alert := CanonicalAlert{
		ID:                  "alert-1",
		Fingerprint:         "fp-1",
		Provider:            "agy",
		Profile:             "work",
		Group:               "gemini",
		Window:              "5h",
		Severity:            "CRITICAL",
		RemainingPercent:    &rem,
		LockScreenSafeTitle: "Nexus Quota Alert",
		LockScreenSafeBody:  "Capacity limit reached for agy (work)",
	}

	prefs := store.DefaultCapacityPreferences()
	prefs.Enabled = true
	prefs.NativeNotificationsEnabled = true
	prefs.BrowserNotificationsEnabled = true

	ctx := context.Background()
	err = router.Route(ctx, alert, prefs, now)
	if err != nil {
		t.Fatalf("router should succeed even if one channel fails: %v", err)
	}

	// ch1 should have received the alert
	ch1.mu.Lock()
	if len(ch1.delivered) != 1 {
		t.Fatalf("expected 1 delivery in ch1, got %d", len(ch1.delivered))
	}
	ch1.mu.Unlock()

	// Delivery receipts should be persisted in store
	receipts, err := st.ListDeliveryReceipts("alert-1")
	if err != nil || len(receipts) != 2 {
		t.Fatalf("expected 2 receipts in store, got %d (err: %v)", len(receipts), err)
	}
}

func TestDeliveryRouterHonorsQuietHoursUnlessExhaustedBypass(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "nexus_quiet_test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	ch := &fakeChannel{}
	router := NewDeliveryRouter(st, map[string]Channel{
		"native": ch,
	})

	prefs := store.DefaultCapacityPreferences()
	prefs.QuietHoursEnabled = true
	prefs.QuietHoursStartUTC = "22:00"
	prefs.QuietHoursEndUTC = "08:00"
	prefs.ExhaustedBypassQuietHours = true

	// Test at 23:30 (inside quiet hours)
	quietTime := time.Date(2026, 9, 5, 23, 30, 0, 0, time.UTC)

	// Warning alert should be suppressed by quiet hours
	warningAlert := CanonicalAlert{
		ID:          "alert-warn",
		Fingerprint: "fp-warn",
		Severity:    "WARNING",
	}
	_ = router.Route(context.Background(), warningAlert, prefs, quietTime)
	ch.mu.Lock()
	if len(ch.delivered) != 0 {
		t.Fatalf("warning alert should be suppressed during quiet hours, delivered count: %d", len(ch.delivered))
	}
	ch.mu.Unlock()

	// Exhausted alert should bypass quiet hours when enabled
	exhaustedAlert := CanonicalAlert{
		ID:          "alert-exh",
		Fingerprint: "fp-exh",
		Severity:    "EXHAUSTED",
	}
	_ = router.Route(context.Background(), exhaustedAlert, prefs, quietTime)
	ch.mu.Lock()
	if len(ch.delivered) != 1 {
		t.Fatalf("exhausted alert should bypass quiet hours, delivered count: %d", len(ch.delivered))
	}
	ch.mu.Unlock()
}
