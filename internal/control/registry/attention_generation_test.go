package registry

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestAttentionUpdateCannotReviveStoppedSession(t *testing.T) {
	reg := NewRegistry(filepath.Join(t.TempDir(), "runtimes.json"))
	if err := reg.Register(RuntimeSession{
		RuntimeID: "runtime-one", State: StateWaiting,
		HostGeneration: 101, ProviderGeneration: 202,
	}); err != nil {
		t.Fatal(err)
	}
	active := AttentionUpdate{
		State: StateWaiting, Reason: "QUESTION", Kind: "needs_user",
		Context: "Continue?", PromptKind: "yn", Fingerprint: "q1",
		Source: "INFERRED", Generation: "101:202",
	}
	if err := reg.UpdateAttentionMeta("runtime-one", active); err != nil {
		t.Fatal(err)
	}
	if err := reg.UpdateAttentionMeta("runtime-one", AttentionUpdate{
		State: StateStopped, Kind: "idle", Generation: "101:202",
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.UpdateAttentionMeta("runtime-one", active); err != nil {
		t.Fatalf("late terminal output should be harmless: %v", err)
	}
	sess, _ := reg.Get("runtime-one")
	if sess.State != StateStopped || sess.AttentionKind != "idle" ||
		sess.AttentionReason != "" || sess.AttentionRevision != 1 {
		t.Fatalf("stopped session was resurrected: %+v", sess)
	}
}

func TestAttentionUpdateFencesPriorHostAndProviderGeneration(t *testing.T) {
	reg := NewRegistry(filepath.Join(t.TempDir(), "runtimes.json"))
	initial := RuntimeSession{
		RuntimeID: "reused-id", State: StateWaiting,
		HostGeneration: 101, ProviderGeneration: 202,
	}
	if err := reg.Register(initial); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(RuntimeSession{
		RuntimeID: "reused-id", State: StateRunning,
		HostGeneration: 303, ProviderGeneration: 404,
	}); err != nil {
		t.Fatal(err)
	}
	late := AttentionUpdate{
		State: StateWaiting, Reason: "QUESTION", Kind: "needs_user",
		Context: "Old approval?", PromptKind: "yn", Fingerprint: "old",
		Source: "INFERRED", Generation: "101:202",
	}
	if err := reg.UpdateAttentionMeta("reused-id", late); !errors.Is(err, ErrAttentionConflict) {
		t.Fatalf("old host event must be rejected: %v", err)
	}
	if err := reg.UpdateAttentionMeta("reused-id", AttentionUpdate{
		State: StateStopped, Kind: "idle", Generation: "101:202",
	}); !errors.Is(err, ErrAttentionConflict) {
		t.Fatalf("old host shutdown must not terminate new session: %v", err)
	}
	sess, _ := reg.Get("reused-id")
	if sess.State != StateRunning || sess.AttentionRevision != 0 || sess.AttentionReason != "" {
		t.Fatalf("old generation mutated replacement runtime: %+v", sess)
	}
	late.Generation = "303:404"
	if err := reg.UpdateAttentionMeta("reused-id", late); err != nil {
		t.Fatalf("current generation should be allowed: %v", err)
	}
	sess, _ = reg.Get("reused-id")
	if sess.State != StateWaiting || sess.AttentionRevision != 1 {
		t.Fatalf("new generation did not receive question: %+v", sess)
	}
}

func TestAttentionRevisionAdvancesWhenKindChangesWithSameFingerprint(t *testing.T) {
	reg := NewRegistry("")
	if err := reg.Register(RuntimeSession{
		RuntimeID: "episode", State: StateWaiting,
		HostGeneration: 111, ProviderGeneration: 222,
		AttentionKind: "working", AttentionFingerprint: "same-text",
		AttentionRevision: 6,
	}); err != nil {
		t.Fatal(err)
	}
	if err := reg.UpdateAttentionMeta("episode", AttentionUpdate{
		State: StateWaiting, Reason: "QUESTION", Kind: "needs_user",
		Context: "Again?", PromptKind: "yn", Fingerprint: "same-text",
		Source: "INFERRED", Generation: "111:222",
	}); err != nil {
		t.Fatal(err)
	}
	sess, _ := reg.Get("episode")
	if sess.AttentionRevision != 7 {
		t.Fatalf("new episode must advance revision even with repeated text: %+v", sess)
	}
}
