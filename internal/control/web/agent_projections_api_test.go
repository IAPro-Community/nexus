package web

import (
	"bytes"
	"net/http"
	"testing"
)

func TestAgentProjectionRoutesRequireCSRFAndAreRegistered(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	body := []byte(`{"runtime_id":"rt_missing_projection"}`)

	for _, path := range []string{
		"/api/v1/agent-projections/plan",
		"/api/v1/agent-projections/apply",
		"/api/v1/agent-projections/discovery",
		"/api/v1/agent-projections/activation",
	} {
		req, _ := http.NewRequest(http.MethodPost, base+path, bytes.NewReader(body))
		req.Header.Set("X-CSRF-Token", csrf)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			t.Fatalf("Agent Projection route %s is not registered", path)
		}
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("Agent Projection route %s status=%d, want 409 for missing runtime", path, resp.StatusCode)
		}

		req, _ = http.NewRequest(http.MethodPost, base+path, bytes.NewReader(body))
		resp, err = client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("Agent Projection route %s without CSRF status=%d, want 403", path, resp.StatusCode)
		}
	}
}
