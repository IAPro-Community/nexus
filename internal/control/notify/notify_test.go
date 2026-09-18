package notify

import (
	"strings"
	"testing"
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
