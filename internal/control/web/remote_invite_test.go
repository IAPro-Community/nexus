package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/originpolicy"
)

func TestTunnelInviteEndpointRequiresAuthAndCSRF(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	auth.SetTunnelActive(true)
	srv := &Server{
		auth: auth,
		tunnel: &tunnelManager{tunnel: &Tunnel{URL: "https://example.trycloudflare.com"}},
	}
	handler := srv.authMiddleware(srv.handleTunnelQR)
	withoutSession := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/tunnel/invite", nil)
	withoutSession.Header.Set("Origin", "http://127.0.0.1")
	noAuth := httptest.NewRecorder()
	handler(noAuth, withoutSession)
	if noAuth.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated invitation issuance got %d", noAuth.Code)
	}

	sess, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	newRequest := func(method string, csrf bool) *http.Request {
		req := httptest.NewRequest(method, "http://127.0.0.1/api/v1/tunnel/invite", nil)
		req.Header.Set("Origin", "http://127.0.0.1")
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
		if csrf {
			req.Header.Set(csrfHeaderName, sess.CSRFToken)
		}
		return req
	}
	missingCSRF := httptest.NewRecorder()
	handler(missingCSRF, newRequest(http.MethodPost, false))
	if missingCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF accepted: %d", missingCSRF.Code)
	}
	unsafeGET := httptest.NewRecorder()
	handler(unsafeGET, newRequest(http.MethodGet, false))
	if unsafeGET.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET must not mint invite: %d", unsafeGET.Code)
	}
	ok := httptest.NewRecorder()
	handler(ok, newRequest(http.MethodPost, true))
	if ok.Code != http.StatusOK {
		t.Fatalf("valid invite POST got %d: %s", ok.Code, ok.Body.String())
	}
	var invitation struct {
		URL        string `json:"url"`
		Bootstrap  string `json:"bootstrap_url"`
		ExpiresIn  int    `json:"expires_in_seconds"`
	}
	if err := json.Unmarshal(ok.Body.Bytes(), &invitation); err != nil {
		t.Fatal(err)
	}
	if invitation.URL != "https://example.trycloudflare.com" || invitation.ExpiresIn != int(remoteBootstrapTTL.Seconds()) {
		t.Fatalf("bad invitation response: %+v", invitation)
	}
	if ok.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("invite response must not be cached")
	}
	token := strings.TrimPrefix(invitation.Bootstrap, invitation.URL+"/#nexus_bootstrap=")
	if token == invitation.Bootstrap || token == "" {
		t.Fatal("missing one-time token in fragment")
	}
	if _, accepted := auth.ExchangeBootstrapTokenForHost(token, "example.trycloudflare.com"); !accepted {
		t.Fatal("issued invitation must mint a session for the expected hostname")
	}
}

func TestTunnelBootstrapSetsSecureCookieAndKeepsSession(t *testing.T) {
	const host = "example.trycloudflare.com"
	originpolicy.RegisterTunnelHost(host)
	defer originpolicy.UnregisterTunnelHost(host)

	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	auth.SetTunnelActive(true)
	token, err := auth.IssueTunnelInvitation(host)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth, loopback: true}
	makeRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost,
			"https://"+host+"/api/v1/auth/bootstrap", strings.NewReader(`{"token":"`+token+`"}`))
		req.Header.Set("Origin", "https://"+host)
		return req
	}
	rec := httptest.NewRecorder()
	srv.handleAuthBootstrap(rec, makeRequest())
	if rec.Code != http.StatusOK {
		t.Fatalf("HTTPS tunnel bootstrap failed: %d %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly ||
		cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("tunnel must set secure HttpOnly SameSite=Strict cookie: %+v", cookies)
	}
	next := httptest.NewRequest(http.MethodGet, "https://"+host+"/api/v1/session", nil)
	next.AddCookie(cookies[0])
	if auth.AuthenticateRequest(next) == nil {
		t.Fatal("session must survive another request after bootstrap")
	}
	replay := httptest.NewRecorder()
	srv.handleAuthBootstrap(replay, makeRequest())
	if replay.Code != http.StatusUnauthorized {
		t.Fatalf("replay must be rejected, got %d", replay.Code)
	}
}

func TestLANHTTPBootstrapRequiresTLS(t *testing.T) {
	auth, token, err := NewAuthManager("192.168.1.10", "13000")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth}
	req := httptest.NewRequest(http.MethodPost,
		"http://192.168.1.10/api/v1/auth/bootstrap", strings.NewReader(`{"token":"`+token+`"}`))
	req.Header.Set("Origin", "http://192.168.1.10")
	rec := httptest.NewRecorder()
	srv.handleAuthBootstrap(rec, req)
	if rec.Code != http.StatusUpgradeRequired {
		t.Fatalf("plain HTTP LAN login must require HTTPS, got %d", rec.Code)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("must not issue a session cookie on plaintext LAN")
	}
	// A rejected insecure attempt must not consume a valid private bootstrap.
	if auth.usedBootstrap {
		t.Fatal("refused LAN login consumed bootstrap token")
	}
}

func TestExternalTunnelAdoptionAndStaleExitDoNotRevokeNewTunnel(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	auth.SetTunnelActive(true)
	srv := &Server{auth: auth, tunnelHost: "first.trycloudflare.com"}
	first := &Tunnel{URL: "https://first.trycloudflare.com"}
	if err := srv.AttachExternalTunnel(first); err != nil {
		t.Fatal(err)
	}
	host, invite, err := srv.NewTunnelInviteURL()
	if err != nil || host != first.URL {
		t.Fatalf("adopted CLI tunnel must issue invites: host=%q err=%v", host, err)
	}
	token := strings.TrimPrefix(invite, host+"/#nexus_bootstrap=")
	if _, ok := auth.ExchangeBootstrapTokenForHost(token, "first.trycloudflare.com"); !ok {
		t.Fatal("CLI-issued invite must authenticate against active tunnel")
	}
	// Simulate a new registered generation before the prior subprocess exit callback.
	second := &Tunnel{URL: "https://second.trycloudflare.com"}
	srv.tunnelManager().mu.Lock()
	srv.tunnel.tunnel = second
	srv.tunnelManager().mu.Unlock()
	first.OnExit()
	if !auth.IsTunnelActive() || srv.tunnel.tunnel != second {
		t.Fatal("stale exit callback must not disarm newer tunnel")
	}
	srv.trackTunnelExit(second)
	second.OnExit()
	if auth.IsTunnelActive() {
		t.Fatal("current tunnel exit must disarm authentication")
	}
}

func TestIPv6LoopbackBootstrapRemainsAllowed(t *testing.T) {
	auth, token, err := NewAuthManager("::1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{auth: auth, loopback: true}
	req := httptest.NewRequest(http.MethodPost,
		"http://[::1]:13000/api/v1/auth/bootstrap", strings.NewReader(`{"token":"`+token+`"}`))
	req.Header.Set("Origin", "http://[::1]:13000")
	rec := httptest.NewRecorder()
	srv.handleAuthBootstrap(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("IPv6 loopback bootstrap should be allowed, got %d: %s", rec.Code, rec.Body.String())
	}
}
