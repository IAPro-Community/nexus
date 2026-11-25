package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestNexMemoryEndpointAuthCSRFAndRedaction(t *testing.T) {
	store := nexus.NewNexPersonalMemoryStore(filepath.Join(t.TempDir(), "mem.json"))
	auth, _, err := NewAuthManager("127.0.0.1", "0")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	handler := &NexusHandler{auth: auth, nexMemory: store}
	srv := &Server{auth: auth}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/nex/memory", srv.authMiddleware(handler.handleNexMemory))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 401 without auth
	resUnauth, err := http.Get(ts.URL + "/api/v1/nex/memory")
	if err != nil {
		t.Fatal(err)
	}
	_ = resUnauth.Body.Close()
	if resUnauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauth, got %d", resUnauth.StatusCode)
	}

	// 403 missing CSRF on POST
	secret := "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz"
	payload, _ := json.Marshal(map[string]string{"key": "lang", "value": "pt-BR " + secret})
	noCsrfReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/nex/memory", bytes.NewReader(payload))
	noCsrfReq.Header.Set("Authorization", "Bearer "+sess.ID)
	noCsrfRes, err := ts.Client().Do(noCsrfReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = noCsrfRes.Body.Close()
	if noCsrfRes.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 missing CSRF, got %d", noCsrfRes.StatusCode)
	}

	// 201 Created with CSRF and secret redacted
	postReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/nex/memory", bytes.NewReader(payload))
	postReq.Header.Set("Authorization", "Bearer "+sess.ID)
	postReq.Header.Set("X-CSRF-Token", sess.CSRFToken)
	postRes, err := ts.Client().Do(postReq)
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		Entry nexus.NexPersonalMemoryEntry `json:"entry"`
	}
	if err := json.NewDecoder(postRes.Body).Decode(&created); err != nil {
		_ = postRes.Body.Close()
		t.Fatal(err)
	}
	_ = postRes.Body.Close()
	if postRes.StatusCode != http.StatusCreated || strings.Contains(created.Entry.Value, secret) {
		t.Fatalf("unexpected POST response status=%d entry=%+v", postRes.StatusCode, created.Entry)
	}

	// GET lists the entry
	getReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/nex/memory", nil)
	getReq.Header.Set("Authorization", "Bearer "+sess.ID)
	getRes, err := ts.Client().Do(getReq)
	if err != nil {
		t.Fatal(err)
	}
	var listResp struct {
		Entries []nexus.NexPersonalMemoryEntry `json:"entries"`
	}
	_ = json.NewDecoder(getRes.Body).Decode(&listResp)
	_ = getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK || len(listResp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got status=%d resp=%+v", getRes.StatusCode, listResp)
	}

	// DELETE ?id=... removes entry
	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/nex/memory?id="+created.Entry.ID, nil)
	delReq.Header.Set("Authorization", "Bearer "+sess.ID)
	delReq.Header.Set("X-CSRF-Token", sess.CSRFToken)
	delRes, err := ts.Client().Do(delReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = delRes.Body.Close()
	if delRes.StatusCode != http.StatusOK || len(store.List()) != 0 {
		t.Fatalf("expected DELETE 200 and empty store, got %d", delRes.StatusCode)
	}
}
