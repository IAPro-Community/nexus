package store

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestNexConversationDurabilityIsolationAndDeletion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nexus.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	a, err := st.CreateProject(Project{Name: "A", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.CreateProject(Project{Name: "B", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.StoreNexConversationExchange(a.ID, "request-A-001", "status?", "Ativo.", "nexus_snapshot"); err != nil {
		t.Fatal(err)
	}
	if _, err = st.StoreNexConversationExchange("", "request-G-001", "ajuda", "Disponível.", "nexus_snapshot"); err != nil {
		t.Fatal(err)
	}
	if v, err := st.StoreNexConversationExchange(a.ID, "request-A-001", "status?", "Outra resposta.", "nexus_snapshot"); err != nil || v != "Ativo." {
		t.Fatalf("dedup retry: %q, %v", v, err)
	}
	if _, err := st.StoreNexConversationExchange(a.ID, "request-A-001", "mensagem diferente", "Ativo.", "nexus_snapshot"); err == nil {
		t.Fatal("idempotency conflict must fail")
	}
	if turns, err := st.ListNexConversation(b.ID); err != nil || len(turns) != 0 {
		t.Fatalf("cross-project leak: %v %v", turns, err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if turns, err := st.ListNexConversation(a.ID); err != nil || len(turns) != 2 || turns[0].Role != "user" || turns[1].Content != "Ativo." {
		t.Fatalf("reopen transcript: %+v %v", turns, err)
	}
	if err := st.DeleteNexConversation(a.ID); err != nil {
		t.Fatal(err)
	}
	if turns, err := st.ListNexConversation(a.ID); err != nil || len(turns) != 0 {
		t.Fatalf("delete did not remove transcript: %+v %v", turns, err)
	}
	if turns, err := st.ListNexConversation(""); err != nil || len(turns) != 2 {
		t.Fatalf("global transcript changed: %+v %v", turns, err)
	}
}

func TestNexConversationLimits(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "limited.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if _, err := st.StoreNexConversationExchange("", "req-small", "hello", "answer", "nexus_snapshot"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.StoreNexConversationExchange("", "req-broken", strings.Repeat("x", 8001), "answer", "nexus_snapshot"); err == nil {
		t.Fatal("must reject huge user message")
	}
	if _, err := st.StoreNexConversationExchange("", "req-unknown", "hi", "ok", "fabricated"); err == nil {
		t.Fatal("must reject invented source")
	}
	for i := 0; i < 90; i++ {
		key := fmt.Sprintf("req-%08d", i)
		if _, err := st.StoreNexConversationExchange("", key, "hello", "answer", "nexus_snapshot"); err != nil {
			t.Fatal(err)
		}
	}
	turns, err := st.ListNexConversation("")
	if err != nil || len(turns) != nexConversationHistoryLimit {
		t.Fatalf("bounded list: len=%d error=%v", len(turns), err)
	}
	var retained int
	if err := st.DB().QueryRow(`SELECT COUNT(*) FROM nex_conversation_turns`).Scan(&retained); err != nil || retained > 160 {
		t.Fatalf("retention exceeded: %d %v", retained, err)
	}
}
