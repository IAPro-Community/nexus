package web

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
)

func authenticatedTestClient(t *testing.T, srv *Server) (*http.Client, string) {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	body := strings.NewReader(`{"token":"` + srv.bootstrap + `"}`)
	resp, err := client.Post(srv.URL()+"/api/v1/auth/bootstrap", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bootstrap POST returned %d", resp.StatusCode)
	}
	sessResp, err := client.Get(srv.URL() + "/api/v1/session")
	if err != nil {
		t.Fatal(err)
	}
	defer sessResp.Body.Close()
	var session struct {
		Authenticated bool   `json:"authenticated"`
		CSRFToken     string `json:"csrf_token"`
	}
	if err := json.NewDecoder(sessResp.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	if !session.Authenticated || session.CSRFToken == "" {
		t.Fatalf("expected authenticated session, got %+v", session)
	}
	return client, session.CSRFToken
}

func newStartedTestServer(t *testing.T) *Server {
	t.Helper()
	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Start() }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })
	return srv
}

func TestSessionRotateRouteReplacesSessionAndCSRF(t *testing.T) {
	srv := newStartedTestServer(t)
	client, oldCSRF := authenticatedTestClient(t, srv)

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/session/rotate", nil)
	req.Header.Set(csrfHeaderName, oldCSRF)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected rotate 200, got %d", resp.StatusCode)
	}
	var payload struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.CSRFToken == "" || payload.CSRFToken == oldCSRF {
		t.Fatalf("expected fresh CSRF token, old=%q new=%q", oldCSRF, payload.CSRFToken)
	}
}

func TestServer_LoopbackCookieSurvivesRestart(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	first, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = first.Start() }()
	client, _ := authenticatedTestClient(t, first)
	port := first.listener.Addr().(*net.TCPAddr).Port
	if err := first.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	second, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: port})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = second.Start() }()
	t.Cleanup(func() { _ = second.Shutdown(context.Background()) })

	sessResp, err := client.Get(second.URL() + "/api/v1/session")
	if err != nil {
		t.Fatal(err)
	}
	defer sessResp.Body.Close()
	var payload struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.NewDecoder(sessResp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Authenticated {
		t.Fatal("browser cookie must still authenticate after nexus web restart on loopback")
	}
}

func TestSessionLogoutRouteRevokesSession(t *testing.T) {
	srv := newStartedTestServer(t)
	client, csrf := authenticatedTestClient(t, srv)

	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/session/logout", nil)
	req.Header.Set(csrfHeaderName, csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected logout 204, got %d", resp.StatusCode)
	}

	sessionResp, err := client.Get(srv.URL() + "/api/v1/session")
	if err != nil {
		t.Fatal(err)
	}
	defer sessionResp.Body.Close()
	var payload struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.NewDecoder(sessionResp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Authenticated {
		t.Fatal("expected session to be revoked after logout")
	}
}
