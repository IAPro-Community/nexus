package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoopbackReauthMintsSession(t *testing.T) {
	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/loopback-reauth", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["authenticated"] != true {
		t.Fatalf("expected authenticated: %#v", body)
	}
	if body["csrf_token"] == nil || body["csrf_token"] == "" {
		t.Fatalf("expected csrf_token: %#v", body)
	}
	found := false
	for _, c := range res.Result().Cookies() {
		if c.Name == sessionCookieName && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected session cookie")
	}
}

func TestLoopbackReauthRejectsNonLoopbackClient(t *testing.T) {
	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/loopback-reauth", nil)
	req.RemoteAddr = "192.168.1.20:54321"
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}
