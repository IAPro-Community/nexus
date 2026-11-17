package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestNexConversationEndpointRequiresAuthCSRFAndScopes(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Cleanup(nexus.ResetDefaultForTest)
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	client, srv, csrf := csrfClient(t)
	base := srv.URL() + "/api/v1/nex/conversation"
	res, err := client.Get(base)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET status %d", res.StatusCode)
	}
	res.Body.Close()
	deleteReq, _ := http.NewRequest(http.MethodDelete, base, nil)
	res, err = client.Do(deleteReq)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("DELETE without csrf: %d", res.StatusCode)
	}
	deleteReq, _ = http.NewRequest(http.MethodDelete, base, nil)
	deleteReq.Header.Set("X-CSRF-Token", csrf)
	res, err = client.Do(deleteReq)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("DELETE with csrf: %d", res.StatusCode)
	}
	unknown, err := client.Get(base + "?project_id=unknown_project")
	if err != nil {
		t.Fatal(err)
	}
	unknown.Body.Close()
	if unknown.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown scope status: %d", unknown.StatusCode)
	}
}

func TestNexAssistantDeterministicChatPersistsWhenOptedIn(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Cleanup(nexus.ResetDefaultForTest)
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	client, srv, csrf := csrfClient(t)
	body, _ := json.Marshal(map[string]any{
		"message": "status", "project_id": "", "persist": true,
		"request_id": "test-nex-request-0001",
	})
	send := func() string {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/nex/assistant", bytes.NewReader(body))
		req.Header.Set("X-CSRF-Token", csrf)
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("persist chat status %d", res.StatusCode)
		}
		var reply struct{ Answer string `json:"answer"` }
		if err := json.NewDecoder(res.Body).Decode(&reply); err != nil {
			t.Fatal(err)
		}
		return reply.Answer
	}
	if a, b := send(), send(); a == "" || a != b {
		t.Fatalf("dedup response mismatch: %q %q", a, b)
	}
	res, err := client.Get(srv.URL()+"/api/v1/nex/conversation")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var data struct{ Turns []struct{ Role string `json:"role"` } `json:"turns"` }
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil || len(data.Turns) != 2 {
		t.Fatalf("Nex conversation not persisted: len=%d err=%v", len(data.Turns), err)
	}
}
