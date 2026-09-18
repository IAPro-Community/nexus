package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEffectiveExposureLoopbackWithoutTunnel(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	if got := auth.EffectiveExposure(); got != ExposureLoopback {
		t.Fatalf("exposure=%s want %s", got, ExposureLoopback)
	}
	if auth.RequiresSecureCookie() {
		t.Fatal("loopback without tunnel must not require Secure cookies")
	}
	if !auth.BootstrapReusable() {
		t.Fatal("loopback without tunnel must allow reusable bootstrap")
	}
}

func TestEffectiveExposureLoopbackWithTunnelIsPublicRemote(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	auth.SetTunnelActive(true)
	if got := auth.EffectiveExposure(); got != ExposurePublicRemote {
		t.Fatalf("tunnel on loopback bind must be PUBLIC_REMOTE, got %s", got)
	}
	if !auth.RequiresSecureCookie() {
		t.Fatal("PUBLIC_REMOTE must require Secure cookies even on loopback bind")
	}
	if auth.BootstrapReusable() {
		t.Fatal("PUBLIC_REMOTE bootstrap must not be reusable")
	}
}

func TestEffectiveExposurePrivateBind(t *testing.T) {
	auth, _, err := NewAuthManager("192.168.1.10", "13000")
	if err != nil {
		t.Fatal(err)
	}
	if got := auth.EffectiveExposure(); got != ExposurePrivateNetwork {
		t.Fatalf("exposure=%s want %s", got, ExposurePrivateNetwork)
	}
	if !auth.RequiresSecureCookie() {
		t.Fatal("private network bind must require Secure cookies")
	}
}

func TestRemoteBootstrapExpires(t *testing.T) {
	auth, token, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	auth.SetTunnelActive(true)
	fixed := time.Now()
	auth.clock = func() time.Time { return fixed }
	auth.bootstrapCreatedAt = fixed.Add(-remoteBootstrapTTL - time.Second)
	if sess, ok := auth.ExchangeBootstrapToken(token); ok || sess != nil {
		t.Fatal("expired remote bootstrap must reject")
	}
}

func TestRemoteSessionsAreNotPersistedAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	first, token, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	first.SetTunnelActive(true)
	sess, ok := first.ExchangeBootstrapToken(token)
	if !ok || sess == nil {
		t.Fatal("first remote bootstrap must succeed")
	}
	// Force a persist attempt — PUBLIC_REMOTE must refuse to write sessions.
	if err := first.persistLocked(); err != nil {
		t.Fatal(err)
	}
	second, _, err := NewAuthManagerWithStore("127.0.0.1", "13000", dir)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	req.Header.Set("Authorization", "Bearer "+sess.ID)
	if second.AuthenticateRequest(req) != nil {
		t.Fatal("sessions minted under PUBLIC_REMOTE must not restore after restart")
	}
}

func TestServerCookieSecureFollowsEffectiveExposure(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{auth: auth, loopback: true}
	if s.cookieSecure() {
		t.Fatal("loopback without tunnel must not set Secure")
	}
	auth.SetTunnelActive(true)
	if !s.cookieSecure() {
		t.Fatal("PUBLIC_REMOTE must set Secure cookies")
	}
}
