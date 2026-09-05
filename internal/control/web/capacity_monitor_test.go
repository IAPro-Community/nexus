package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/nexus/store"
)

func TestCapacityMonitorEndpoints(t *testing.T) {
	client, srv, csrfToken := csrfClient(t)
	st := srv.api.auth // or nexus store

	// 1. GET /api/v1/capacity-monitor
	resp, err := client.Get(srv.URL() + "/api/v1/capacity-monitor")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/capacity-monitor failed: err=%v, code=%d", err, resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 2. GET /api/v1/capacity-monitor/resources
	resp, err = client.Get(srv.URL() + "/api/v1/capacity-monitor/resources")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/v1/capacity-monitor/resources failed: err=%v, code=%d", err, resp.StatusCode)
	}
	var resList []any
	bodyBytes, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err := json.Unmarshal(bodyBytes, &resList); err != nil {
		t.Fatalf("unmarshal resources: %v, body: %s", err, string(bodyBytes))
	}

	// 3. GET & PUT /api/v1/capacity-monitor/preferences
	resp, err = client.Get(srv.URL() + "/api/v1/capacity-monitor/preferences")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET preferences failed: code=%d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	putBody := store.DefaultCapacityPreferences()
	putBody.WarningThreshold = 18.0
	b, _ := json.Marshal(putBody)

	req, _ := http.NewRequest(http.MethodPut, srv.URL()+"/api/v1/capacity-monitor/preferences", bytes.NewReader(b))
	req.Header.Set(csrfHeaderName, csrfToken)
	resp, err = client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT preferences failed: code=%d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 4. Alert ack and snooze
	req, _ = http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/capacity-monitor/alerts/alert-123/ack", bytes.NewReader([]byte(`{}`)))
	req.Header.Set(csrfHeaderName, csrfToken)
	resp, err = client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("POST ack failed: code=%d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	req, _ = http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/capacity-monitor/alerts/alert-123/snooze", bytes.NewReader([]byte(`{"duration_seconds": 1800}`)))
	req.Header.Set(csrfHeaderName, csrfToken)
	resp, err = client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("POST snooze failed: code=%d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 5. POST /api/v1/agents/{id}/capacity-switch without confirmation must fail
	unconfirmedBody := map[string]any{
		"agent_id":                   "agent-test-1",
		"recommendation_fingerprint": "fp-123",
		"provider":                   "agy",
		"profile":                    "default",
		"confirmed":                  false,
	}
	b, _ = json.Marshal(unconfirmedBody)
	req, _ = http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/agents/agent-test-1/capacity-switch", bytes.NewReader(b))
	req.Header.Set(csrfHeaderName, csrfToken)
	resp, err = client.Do(req)
	if err != nil || resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unconfirmed switch must return 400 Bad Request, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
	_ = st
}
