package host

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/agentwire"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestSessionHostKeepsNativePermissionUntilAuthoritativeSettlement(t *testing.T) {
	dir := t.TempDir()
	reg := registry.NewRegistry(filepath.Join(dir, "registry.json"))
	sess := registry.RuntimeSession{
		RuntimeID: "reducer-host-1", ProviderID: "opencode", ProfileID: "default",
		State: registry.StateRunning, Workspace: dir, StartedAt: time.Now(),
		ControlLevel: registry.ControlLevelEvents, Labels: map[string]string{"nexus.agent_protocol": "acp"},
	}
	if err := reg.Register(sess); err != nil {
		t.Fatal(err)
	}
	sh, err := NewSessionHost(Config{Session: sess, Registry: reg, Binary: "mock", Cwd: dir})
	if err != nil {
		t.Fatal(err)
	}
	sh.observeStructuredSignal(agentwire.WireSignal{State: "blocked", Kind: "permission", Summary: "Approve a safe edit?", InteractionID: "req-1"}, "acp")
	blocked, ok := reg.Get(sess.RuntimeID)
	if !ok || blocked.AttentionKind != AttentionNeedsUser {
		t.Fatalf("permission not visible: %+v", blocked)
	}
	sh.observeStructuredSignal(agentwire.WireSignal{State: "working", Summary: "Tool in progress"}, "acp")
	stillBlocked, _ := reg.Get(sess.RuntimeID)
	if stillBlocked.AttentionKind != AttentionNeedsUser {
		t.Fatalf("a progress event incorrectly resolved the permission: %+v", stillBlocked)
	}
	snap, ok := sh.wireReducer.Get(sess.RuntimeID)
	if !ok || len(snap.Attention) != 1 {
		t.Fatalf("reducer lost pending permission: %+v", snap)
	}
	sh.observeStructuredSignal(agentwire.WireSignal{State: "done", Summary: "Agent turn settled"}, "acp")
	done, _ := reg.Get(sess.RuntimeID)
	snap, _ = sh.wireReducer.Get(sess.RuntimeID)
	if len(snap.Attention) != 0 || done.AttentionKind == AttentionNeedsUser {
		t.Fatalf("authoritative completion did not reconcile: %+v %+v", snap, done)
	}
}
