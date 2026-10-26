package notify

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRecorderCapturesPayload(t *testing.T) {
	rec := &Recorder{}
	SetDefault(rec)
	t.Cleanup(func() { SetDefault(nil) })

	if err := Default().Notify(Payload{Title: "Nexus · demo", Body: "Deseja continuar?", Tag: "fp1"}); err != nil {
		t.Fatalf("notify: %v", err)
	}
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(rec.Payloads))
	}
	if rec.Payloads[0].Body != "Deseja continuar?" {
		t.Fatalf("unexpected body: %q", rec.Payloads[0].Body)
	}
}

func TestThrottleSuppressesSameTag(t *testing.T) {
	rec := &Recorder{}
	n := NewThrottled(rec, 0) // uses default 30s
	_ = n.Notify(Payload{Title: "a", Body: "b", Tag: "same"})
	_ = n.Notify(Payload{Title: "a", Body: "b", Tag: "same"})
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected throttle to keep 1 notify, got %d", len(rec.Payloads))
	}
}

func TestThrottleAllowsRetryAfterTransportFailure(t *testing.T) {
	inner := &flakyNotifier{failuresRemaining: 1}
	n := NewThrottled(inner, time.Minute)
	payload := Payload{Title: "Nexus", Body: "Atenção", Tag: "mission-1"}
	if err := n.Notify(payload); err == nil {
		t.Fatal("first transport attempt should fail")
	}
	if err := n.Notify(payload); err != nil {
		t.Fatalf("retry after failure should reach the transport: %v", err)
	}
	if inner.calls != 2 {
		t.Fatalf("expected both attempts to reach the transport, got %d", inner.calls)
	}
	if err := n.Notify(payload); err != nil {
		t.Fatal(err)
	}
	if inner.calls != 2 {
		t.Fatalf("successful delivery should activate throttling, got %d calls", inner.calls)
	}
}

func TestIsAvailableDelegatesToNativeCapability(t *testing.T) {
	available := &capabilityNotifier{available: false}
	if IsAvailable(available) {
		t.Fatal("unavailable native channel was reported available")
	}
	if !IsAvailable(&Recorder{}) {
		t.Fatal("notifier without explicit capability should remain available")
	}
}

func TestDesktopNotificationSmoke(t *testing.T) {
	if os.Getenv("NEXUS_ENABLE_NATIVE_NOTIFICATION_SMOKE") != "1" {
		t.Skip("set NEXUS_ENABLE_NATIVE_NOTIFICATION_SMOKE=1 to send a harmless native notification")
	}
	n := desktopNotifier{}
	if !n.Available() {
		t.Skip("native desktop notification capability is unavailable")
	}
	if err := n.Notify(Payload{Title: "Nexus · homologação", Body: "Notificação de teste, sem dados da missão."}); err != nil {
		t.Fatalf("native notification transport rejected the smoke payload: %v", err)
	}
}

type capabilityNotifier struct{ available bool }

func (*capabilityNotifier) Notify(Payload) error { return nil }
func (n *capabilityNotifier) Available() bool    { return n.available }

type flakyNotifier struct {
	failuresRemaining int
	calls             int
}

func (n *flakyNotifier) Notify(Payload) error {
	n.calls++
	if n.failuresRemaining > 0 {
		n.failuresRemaining--
		return errors.New("temporary transport failure")
	}
	return nil
}

func TestNativeNotificationScriptDoesNotInterpolatePayload(t *testing.T) {
	for _, payload := range []Payload{
		{Title: `title"; exit`, Body: "body\n$(Get-ChildItem)"},
		{Title: "normal", Body: `$(whoami)`},
	} {
		if strings.Contains(windowsToastScript, payload.Title) || strings.Contains(windowsToastScript, payload.Body) {
			t.Fatalf("Windows notification script must not contain payload data: %q", payload)
		}
	}
	if !strings.Contains(windowsToastScript, "$Title") || !strings.Contains(windowsToastScript, "$Body") {
		t.Fatal("Windows notification script must consume explicit parameters")
	}
}
