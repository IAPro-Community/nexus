package intelligence

import (
	"context"
	"encoding/json"
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
