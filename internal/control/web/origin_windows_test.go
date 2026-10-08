package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestWindowsWailsVirtualHostSupportsAuthenticatedProjectCreation(t *testing.T) {
	srv := newStartedTestServer(t)
	session, err := srv.CreateDesktopSession()
	if err != nil {
		t.Fatalf("create desktop session: %v", err)
	}

	bootstrapReq, err := http.NewRequest(http.MethodGet, srv.URL()+"/api/v1/desktop/bootstrap", nil)
	if err != nil {
		t.Fatal(err)
	}
	bootstrapReq.Host = "wails.localhost"
	bootstrapReq.Header.Set("Origin", "http://wails.localhost")
	bootstrapReq.Header.Set("Authorization", "Bearer "+session.ID)
	bootstrapResp, err := http.DefaultClient.Do(bootstrapReq)
	if err != nil {
		t.Fatalf("desktop bootstrap through Wails host: %v", err)
	}
	bootstrapResp.Body.Close()
	if bootstrapResp.StatusCode != http.StatusOK {
		t.Fatalf("desktop bootstrap through Wails host returned HTTP %d", bootstrapResp.StatusCode)
	}

	body, err := json.Marshal(map[string]string{
		"name": "Windows Wails Project",
		"path": t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	createReq, err := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/projects", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	createReq.Host = "wails.localhost"
	createReq.Header.Set("Origin", "http://wails.localhost")
	createReq.Header.Set("Authorization", "Bearer "+session.ID)
	createReq.Header.Set("X-CSRF-Token", session.CSRFToken)
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("project creation through Wails host: %v", err)
	}
	createResp.Body.Close()
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("project creation through Wails host returned HTTP %d", createResp.StatusCode)
	}

	for _, tc := range []struct {
		host   string
		origin string
	}{
		{"wails.localhost", "http://wails.localhost.evil.com"},
		{"wails.localhost.evil.com", "http://wails.localhost.evil.com"},
		{"wails.localhost", "http://evil.com"},
	} {
		req, err := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/projects", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Host = tc.host
		req.Header.Set("Origin", tc.origin)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request with host=%q origin=%q: %v", tc.host, tc.origin, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("host=%q origin=%q returned HTTP %d, want 403", tc.host, tc.origin, resp.StatusCode)
		}
	}
}
