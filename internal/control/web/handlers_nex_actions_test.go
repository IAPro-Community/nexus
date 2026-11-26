package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

func TestNexActionEndpoints(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "0")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	h := &NexusHandler{auth: auth, nexActions: nexus.NewNexActionService(filepath.Join(t.TempDir(), "r.json"))}
	srv := &Server{auth: auth}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/nex/actions", srv.authMiddleware(h.handleNexActions))
	mux.HandleFunc("/api/v1/nex/actions/propose", srv.authMiddleware(h.handleNexActionPropose))
	mux.HandleFunc("/api/v1/nex/actions/confirm", srv.authMiddleware(h.handleNexActionConfirm))
	ts := httptest.NewServer(mux)
	defer ts.Close()

	do := func(method, path string, body any, csrf bool) (*http.Response, []byte) {
		var rdr *bytes.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		} else {
			rdr = bytes.NewReader(nil)
		}
		req, _ := http.NewRequest(method, ts.URL+path, rdr)
		req.Header.Set("Authorization", "Bearer "+sess.ID)
		if csrf {
			req.Header.Set("X-CSRF-Token", sess.CSRFToken)
		}
		res, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(res.Body)
		return res, buf.Bytes()
	}

	unauth, _ := http.Get(ts.URL + "/api/v1/nex/actions")
	_ = unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauth.StatusCode)
	}
	if res, _ := do(http.MethodPost, "/api/v1/nex/actions/propose", map[string]any{"action_id": "shell"}, false); res.StatusCode != http.StatusForbidden {
		t.Fatalf("missing CSRF expected 403, got %d", res.StatusCode)
	}
	if res, _ := do(http.MethodPost, "/api/v1/nex/actions/propose", map[string]any{"action_id": "push"}, true); res.StatusCode != http.StatusForbidden {
		t.Fatalf("forbidden action expected 403, got %d", res.StatusCode)
	}

	res, body := do(http.MethodPost, "/api/v1/nex/actions/propose", map[string]any{"action_id": "open_nexus_surface", "params": map[string]string{"target": "agents"}}, true)
	var created struct {
		Receipt nexus.NexActionReceipt `json:"receipt"`
	}
	_ = json.Unmarshal(body, &created)
	if res.StatusCode != http.StatusCreated || created.Receipt.Status != nexus.NexActionStatusProposed {
		t.Fatalf("propose failed: %d %s", res.StatusCode, body)
	}
	res, body = do(http.MethodPost, "/api/v1/nex/actions/confirm", map[string]any{"receipt_id": created.Receipt.ReceiptID, "approved": true}, true)
	var done struct {
		Receipt nexus.NexActionReceipt `json:"receipt"`
	}
	_ = json.Unmarshal(body, &done)
	if res.StatusCode != http.StatusOK || done.Receipt.Status != nexus.NexActionStatusExecuted {
		t.Fatalf("confirm failed: %d %s", res.StatusCode, body)
	}

	d, body := do(http.MethodPost, "/api/v1/nex/actions/propose", map[string]any{"action_id": "ack_capacity_alert"}, true)
	_ = json.Unmarshal(body, &created)
	_ = d
	_, body = do(http.MethodPost, "/api/v1/nex/actions/confirm", map[string]any{"receipt_id": created.Receipt.ReceiptID, "approved": false}, true)
	_ = json.Unmarshal(body, &done)
	if done.Receipt.Status != nexus.NexActionStatusDenied {
		t.Fatalf("deny flow failed: %s", body)
	}
	if res, _ := do(http.MethodGet, "/api/v1/nex/actions", nil, false); res.StatusCode != http.StatusOK {
		t.Fatalf("list expected 200, got %d", res.StatusCode)
	}
}
