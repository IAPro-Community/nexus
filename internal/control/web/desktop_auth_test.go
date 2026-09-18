package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDesktopBootstrapRequiresDesktopOrigin(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}

	sess, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	auth.SetDesktopSession(sess)

	srv := &Server{
		auth: auth,
		url:  "http://127.0.0.1:13000",
	}

	// 1. Request without Origin or Referer -> Must be rejected with 403 Forbidden
	req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/bootstrap", nil)
	req.Host = "127.0.0.1:13000"
	rec := httptest.NewRecorder()

	srv.handleDesktopBootstrap(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-desktop request, got %d", rec.Code)
	}

	// 2. Request with spoofed malicious referer/origin -> Must be rejected with 403 Forbidden
	for _, badOrigin := range []string{
		"wails://evil.com",
		"wails://wails.localhost.evil.com",
		"wails://attacker",
		"http://wails.localhost.evil.com",
		"https://wails.localhost.attacker",
		"http://notwails.localhost",
		"http://evil.com",
	} {
		reqBad := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/bootstrap", nil)
		reqBad.Host = "127.0.0.1:13000"
		reqBad.Header.Set("Origin", badOrigin)
		recBad := httptest.NewRecorder()
		srv.handleDesktopBootstrap(recBad, reqBad)
		if recBad.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for bad origin %s, got %d", badOrigin, recBad.Code)
		}

		reqBadRef := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/bootstrap", nil)
		reqBadRef.Host = "127.0.0.1:13000"
		reqBadRef.Header.Set("Referer", badOrigin+"/index.html")
		recBadRef := httptest.NewRecorder()
		srv.handleDesktopBootstrap(recBadRef, reqBadRef)
		if recBadRef.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden for bad referer %s, got %d", badOrigin, recBadRef.Code)
		}
	}

	// 3. Origin is not an authentication factor; a local process can spoof it.
	reqDesktop := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/bootstrap", nil)
	reqDesktop.Host = "127.0.0.1:13000"
	reqDesktop.Header.Set("Origin", "wails://wails")
	recDesktop := httptest.NewRecorder()

	srv.handleDesktopBootstrap(recDesktop, reqDesktop)
	if recDesktop.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for origin-only desktop request, got %d", recDesktop.Code)
	}

	// 4. A provisioned desktop session must be presented explicitly.
	reqDesktop.Header.Set("Authorization", "Bearer "+sess.ID)
	recDesktop = httptest.NewRecorder()
	srv.handleDesktopBootstrap(recDesktop, reqDesktop)
	if recDesktop.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for authenticated desktop request, got %d", recDesktop.Code)
	}
	if !strings.Contains(recDesktop.Body.String(), sess.ID) {
		t.Fatalf("expected response to contain session ID %s, got %s", sess.ID, recDesktop.Body.String())
	}

	// 5. Request with invalid Origin and valid Referer -> Must be rejected with 403 Forbidden (Origin takes precedence)
	reqConflict := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/bootstrap", nil)
	reqConflict.Host = "127.0.0.1:13000"
	reqConflict.Header.Set("Origin", "wails://evil.com")
	reqConflict.Header.Set("Referer", "wails://wails/index.html")
	recConflict := httptest.NewRecorder()
	srv.handleDesktopBootstrap(recConflict, reqConflict)
	if recConflict.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when Origin is invalid despite valid Referer, got %d", recConflict.Code)
	}

	// 6. A valid Referer without the explicit session must not bootstrap.
	reqRefererOnly := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/bootstrap", nil)
	reqRefererOnly.Host = "127.0.0.1:13000"
	reqRefererOnly.Header.Set("Referer", "wails://wails/index.html")
	recRefererOnly := httptest.NewRecorder()
	srv.handleDesktopBootstrap(recRefererOnly, reqRefererOnly)
	if recRefererOnly.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for referer-only desktop request, got %d", recRefererOnly.Code)
	}
}

func TestDesktopSessionRevocationAndExpiry(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}

	sess, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	auth.SetDesktopSession(sess)

	// Origin alone must NOT authenticate (spoofable from any local process).
	reqOriginOnly := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	reqOriginOnly.Host = "127.0.0.1:13000"
	reqOriginOnly.Header.Set("Origin", "wails://wails")
	if got := auth.AuthenticateRequest(reqOriginOnly); got != nil {
		t.Fatalf("expected Origin-only desktop request to be rejected, got %v", got)
	}

	// Explicit Bearer from CreateDesktopSession / bootstrap must authenticate.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	req.Host = "127.0.0.1:13000"
	req.Header.Set("Origin", "wails://wails")
	req.Header.Set("Authorization", "Bearer "+sess.ID)

	authed := auth.AuthenticateRequest(req)
	if authed == nil || authed.ID != sess.ID {
		t.Fatalf("expected request to authenticate with desktop session bearer, got %v", authed)
	}

	// Revoke the session
	auth.RevokeSession(sess.ID)

	// Must no longer authenticate
	if got := auth.AuthenticateRequest(req); got != nil {
		t.Fatalf("expected nil after revocation, got %v", got)
	}
	if got := auth.GetDesktopSession(); got != nil {
		t.Fatalf("expected GetDesktopSession to be nil after revocation, got %v", got)
	}

	// Test idle expiration
	sess2, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	// Artificially age the session
	sess2.LastActiveAt = time.Now().Add(-sessionIdleTTL - time.Minute)
	auth.SetDesktopSession(sess2)

	if got := auth.GetDesktopSession(); got != nil {
		t.Fatalf("expected expired desktop session to be nil, got %v", got)
	}
	reqExpired := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	reqExpired.Host = "127.0.0.1:13000"
	reqExpired.Header.Set("Authorization", "Bearer "+sess2.ID)
	if got := auth.AuthenticateRequest(reqExpired); got != nil {
		t.Fatalf("expected AuthenticateRequest to reject expired desktop session, got %v", got)
	}
}

func TestWebSocketQueryTokenRejectedWhenTunnelActive(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/rt/terminal?token="+sess.ID, nil)
	req.Host = "127.0.0.1:13000"
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	if got := auth.AuthenticateRequest(req); got == nil || got.ID != sess.ID {
		t.Fatalf("expected WS query token on loopback without tunnel, got %v", got)
	}

	auth.SetTunnelActive(true)
	if got := auth.AuthenticateRequest(req); got != nil {
		t.Fatalf("expected WS query token rejected while tunnel active, got %v", got)
	}

	reqCookie := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/rt/terminal", nil)
	reqCookie.Host = "127.0.0.1:13000"
	reqCookie.Header.Set("Connection", "Upgrade")
	reqCookie.Header.Set("Upgrade", "websocket")
	reqCookie.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	if got := auth.AuthenticateRequest(reqCookie); got == nil || got.ID != sess.ID {
		t.Fatalf("expected WS cookie auth while tunnel active, got %v", got)
	}
}

func TestWebSocketQueryTokenRejectedOnPrivateRemote(t *testing.T) {
	auth, _, err := NewAuthManager("192.168.1.10", "13000")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/rt/terminal?token="+sess.ID, nil)
	req.Host = "192.168.1.10:13000"
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	if got := auth.AuthenticateRequest(req); got != nil {
		t.Fatalf("expected WS query token rejected on private --remote bind, got %v", got)
	}

	reqCookie := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/rt/terminal", nil)
	reqCookie.Host = "192.168.1.10:13000"
	reqCookie.Header.Set("Connection", "Upgrade")
	reqCookie.Header.Set("Upgrade", "websocket")
	reqCookie.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	if got := auth.AuthenticateRequest(reqCookie); got == nil || got.ID != sess.ID {
		t.Fatalf("expected WS cookie auth on private remote, got %v", got)
	}
}

func TestDesktopSessionReplacementCleansOldSession(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}

	first, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	auth.SetDesktopSession(first)

	reqFirst := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	reqFirst.Host = "127.0.0.1:13000"
	reqFirst.Header.Set("Authorization", "Bearer "+first.ID)
	if auth.AuthenticateRequest(reqFirst) == nil {
		t.Fatal("first desktop session should be authenticated")
	}

	// Create and set a second desktop session
	second, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	auth.SetDesktopSession(second)

	// First session must be removed
	if auth.AuthenticateRequest(reqFirst) != nil {
		t.Fatal("old desktop session must be revoked when new desktop session is set")
	}

	// Second session must be active
	reqSecond := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	reqSecond.Host = "127.0.0.1:13000"
	reqSecond.Header.Set("Authorization", "Bearer "+second.ID)
	if auth.AuthenticateRequest(reqSecond) == nil {
		t.Fatal("new desktop session must be authenticated")
	}
}

func TestServerShutdownRevokesDesktopSession(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}

	sess, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	auth.SetDesktopSession(sess)

	srv := &Server{
		auth:       auth,
		httpServer: &http.Server{},
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}

	if auth.GetDesktopSession() != nil {
		t.Fatal("desktop session must be revoked on server shutdown")
	}
}
