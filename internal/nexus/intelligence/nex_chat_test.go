package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNexChatUsesConfiguredEndpointWithoutTools(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("decode: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Posso ajudar."}}]}`))
	}))
	defer server.Close()

	p := NewOpenAIProvider(server.URL, "test-key", "test-model")
	answer, err := p.NexReply(context.Background(), "persona Nex", "status factual")
	if err != nil || answer != "Posso ajudar." {
		t.Fatalf("NexReply = %q, %v", answer, err)
	}
	if _, ok := captured["tools"]; ok {
		t.Fatal("Nex conversation must not grant model tools")
	}
	if captured["model"] != "test-model" {
		t.Fatalf("unexpected model: %+v", captured)
	}
}

func TestCLIProviderNexChatRequiresVerifiedReadOnlyRunner(t *testing.T) {
	regularCalled := false
	withoutReadOnly := NewCLIProvider("claude", "default", true, func(context.Context, string) (string, error) {
		regularCalled = true
		return "should-not-run", nil
	})
	if withoutReadOnly.SupportsNexReadOnly() {
		t.Fatal("expected CLIProvider without read-only runner not to advertise Nex read-only support")
	}
	_, err := withoutReadOnly.NexReply(context.Background(), "system", "status")
	if !errors.Is(err, ErrNexChatRequiresReadOnly) {
		t.Fatalf("expected ErrNexChatRequiresReadOnly, got %v", err)
	}
	if regularCalled {
		t.Fatal("regular headless runner must not execute for NexReply without read-only sandbox")
	}

	readOnlyCalled := false
	withReadOnly := NewCLIProviderWithReadOnly("codex", "work", true,
		func(context.Context, string) (string, error) {
			regularCalled = true
			return "mutating-runner", nil
		},
		func(ctx context.Context, prompt string) (string, error) {
			readOnlyCalled = true
			return "Resumo factual pronto.", nil
		},
	)
	if !withReadOnly.SupportsNexReadOnly() {
		t.Fatal("expected CLIProvider with read-only runner to advertise Nex read-only support")
	}
	answer, err := withReadOnly.NexReply(context.Background(), "persona Nex", "como estao minhas cotas?")
	if err != nil {
		t.Fatalf("unexpected NexReply error: %v", err)
	}
	if answer != "Resumo factual pronto." {
		t.Fatalf("unexpected answer: %q", answer)
	}
	if !readOnlyCalled || regularCalled {
		t.Fatalf("expected only readOnlyCalled=true regularCalled=false, got readOnly=%v regular=%v", readOnlyCalled, regularCalled)
	}
}
