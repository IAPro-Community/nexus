package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLoopbackReauthMintsSessionWithDeviceCookie(t *testing.T) {
	srv := newLoopbackTestServer(t)
	cookies := bootstrapLoopback(t, srv)

	req := newLoopbackReauthRequest(t, srv, "127.0.0.1:54321")
	attachCookies(req, cookies)
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
	foundSession, foundDevice := false, false
	for _, c := range res.Result().Cookies() {
		if c.Name == sessionCookieName && c.Value != "" {
			foundSession = true
		}
		if c.Name == deviceCookieName && c.Value != "" {
			foundDevice = true
		}
	}
	if !foundSession {
		t.Fatal("expected session cookie")
	}
	if !foundDevice {
		t.Fatal("expected device cookie")
	}
}

func TestLoopbackReauthRejectsAnonymousPost(t *testing.T) {
	srv := newLoopbackTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/loopback-reauth", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLoopbackReauthRejectsMissingDeviceCookie(t *testing.T) {
	srv := newLoopbackTestServer(t)
	req := newLoopbackReauthRequest(t, srv, "127.0.0.1:54321")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLoopbackReauthRejectsSpoofedOrigin(t *testing.T) {
	srv := newLoopbackTestServer(t)
	cookies := bootstrapLoopback(t, srv)
	req := newLoopbackReauthRequest(t, srv, "127.0.0.1:54321")
	attachCookies(req, cookies)
	req.Header.Set("Origin", "http://127.0.0.1:9")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLoopbackReauthRejectsCrossSiteFetchMetadata(t *testing.T) {
	srv := newLoopbackTestServer(t)
	cookies := bootstrapLoopback(t, srv)
	req := newLoopbackReauthRequest(t, srv, "127.0.0.1:54321")
	attachCookies(req, cookies)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLoopbackReauthRejectsNonLoopbackClient(t *testing.T) {
	srv := newLoopbackTestServer(t)
	cookies := bootstrapLoopback(t, srv)
	req := newLoopbackReauthRequest(t, srv, "192.168.1.20:54321")
	attachCookies(req, cookies)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLoopbackReauthRejectsTunnel(t *testing.T) {
	srv := newLoopbackTestServer(t)
	cookies := bootstrapLoopback(t, srv)
	srv.auth.SetTunnelActive(true)
	req := newLoopbackReauthRequest(t, srv, "127.0.0.1:54321")
	attachCookies(req, cookies)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLoopbackReauthRejectsPrivateBind(t *testing.T) {
	srv := newLoopbackTestServer(t)
	cookies := bootstrapLoopback(t, srv)
	srv.auth.listenHost = "192.168.1.20"
	req := newLoopbackReauthRequest(t, srv, "127.0.0.1:54321")
	attachCookies(req, cookies)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLoopbackReauthIgnoresExpiredBootstrapSecret(t *testing.T) {
	srv := newLoopbackTestServer(t)
	cookies := bootstrapLoopback(t, srv)
	srv.auth.bootstrapCreatedAt = time.Now().Add(-loopbackBootstrapTTL - time.Second)
	if _, ok := srv.auth.ExchangeBootstrapToken(srv.bootstrap); ok {
		t.Fatal("expired bootstrap must not mint a session")
	}
	req := newLoopbackReauthRequest(t, srv, "127.0.0.1:54321")
	attachCookies(req, cookies)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestLoopbackLogoutClearsDeviceCookie(t *testing.T) {
	srv := newLoopbackTestServer(t)
	cookies := bootstrapLoopback(t, srv)
	var sessionID, deviceToken string
	for _, c := range cookies {
		if c.Name == sessionCookieName {
			sessionID = c.Value
		}
		if c.Name == deviceCookieName {
			deviceToken = c.Value
		}
	}
	if sessionID == "" || deviceToken == "" {
		t.Fatal("bootstrap must set session and device cookies")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/session/logout", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Host = loopbackHost(t, srv)
	req.Header.Set("Origin", "http://"+loopbackHost(t, srv))
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	sessObj := srv.auth.AuthenticateRequest(req)
	if sessObj == nil {
		t.Fatal("expected authenticated session before logout")
	}
	req.Header.Set(csrfHeaderName, sessObj.CSRFToken)
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", res.Code, res.Body.String())
	}
	cleared := false
	for _, c := range res.Result().Cookies() {
		if c.Name == deviceCookieName && (c.MaxAge < 0 || c.Value == "") {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("expected device cookie to be cleared on logout")
	}
}

func TestAuthBootstrapRejectsSpoofedOrigin(t *testing.T) {
	srv := newLoopbackTestServer(t)
	body := strings.NewReader(`{"token":"` + srv.bootstrap + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/bootstrap", body)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Host = loopbackHost(t, srv)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:9")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func newLoopbackTestServer(t *testing.T) *Server {
	t.Helper()
	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	t.Cleanup(func() {
		_ = srv.Shutdown(context.Background())
	})
	return srv
}

func loopbackHost(t *testing.T, srv *Server) string {
	t.Helper()
	u, err := url.Parse(srv.URL())
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	return u.Host
}

func newLoopbackReauthRequest(t *testing.T, srv *Server, remoteAddr string) *http.Request {
	t.Helper()
	host := loopbackHost(t, srv)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/loopback-reauth", nil)
	req.RemoteAddr = remoteAddr
	req.Host = host
	req.Header.Set("Origin", "http://"+host)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	return req
}

func bootstrapLoopback(t *testing.T, srv *Server) []*http.Cookie {
	t.Helper()
	body := strings.NewReader(`{"token":"` + srv.bootstrap + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/bootstrap", body)
	req.RemoteAddr = "127.0.0.1:54321"
	req.Host = loopbackHost(t, srv)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	srv.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("bootstrap status=%d body=%s", res.Code, res.Body.String())
	}
	return res.Result().Cookies()
}

func attachCookies(req *http.Request, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		if cookie != nil {
			req.AddCookie(cookie)
		}
	}
}
