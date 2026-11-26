package nexus

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestNexActionServiceAllowlistConfirmationAndReceipts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipts.json")
	svc := NewNexActionService(path)
	now := time.Date(2026, 11, 25, 15, 0, 0, 0, time.UTC)

	if len(svc.Catalog()) != 6 {
		t.Fatalf("expected 6 allowlisted actions, got %d", len(svc.Catalog()))
	}
	for _, bad := range []string{"shell", "exec", "push", "deploy", "delete_repo", "paid_fallback"} {
		if _, err := svc.Propose(bad, nil, now); !errors.Is(err, ErrNexActionForbidden) {
			t.Fatalf("%s must be forbidden, got %v", bad, err)
		}
	}
	if _, err := svc.Propose("unknown_thing", nil, now); !errors.Is(err, ErrNexActionUnknown) {
		t.Fatalf("expected unknown, got %v", err)
	}

	calls := 0
	exec := func(spec NexActionSpec, params map[string]string) (string, string) {
		calls++
		return NexActionStatusExecuted, "ok token=sk-proj-1234567890abcdefghijklmnopqrstuvwxyz"
	}
	r, err := svc.Propose("read_hud_summary", map[string]string{"k": "v"}, now)
	if err != nil || r.Status != NexActionStatusProposed {
		t.Fatalf("propose failed: %+v %v", r, err)
	}
	done, err := svc.Confirm(r.ReceiptID, true, exec, now)
	if err != nil || done.Status != NexActionStatusExecuted || calls != 1 {
		t.Fatalf("confirm failed: %+v %v calls=%d", done, err, calls)
	}
	if got := done.Summary; got == "" || nexActionContains(got, "sk-proj-") {
		t.Fatalf("summary must be redacted: %q", got)
	}
	again, _ := svc.Confirm(r.ReceiptID, true, exec, now)
	if calls != 1 || again.Status != NexActionStatusExecuted {
		t.Fatalf("double confirm must not re-execute (calls=%d)", calls)
	}

	d, _ := svc.Propose("ack_capacity_alert", nil, now)
	den, _ := svc.Confirm(d.ReceiptID, false, exec, now)
	if den.Status != NexActionStatusDenied || calls != 1 {
		t.Fatalf("deny must not execute: %+v", den)
	}

	u, _ := svc.Propose("delegate_agent_task", nil, now)
	un, _ := svc.Confirm(u.ReceiptID, true, func(NexActionSpec, map[string]string) (string, string) {
		return NexActionStatusUnavailable, "no Maestro/agent target"
	}, now)
	if un.Status != NexActionStatusUnavailable {
		t.Fatalf("expected UNAVAILABLE, got %+v", un)
	}

	if _, err := svc.Confirm("rcpt-missing", true, exec, now); !errors.Is(err, ErrNexReceiptUnknown) {
		t.Fatalf("expected unknown receipt, got %v", err)
	}

	reloaded := NewNexActionService(path)
	if len(reloaded.Receipts()) != 3 {
		t.Fatalf("expected 3 persisted receipts, got %d", len(reloaded.Receipts()))
	}
}

func nexActionContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
