package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newAccessCodeFixture(t *testing.T, host string) (*AuthManager, string, string) {
	t.Helper()
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	auth.SetTunnelActive(true)
	token, code, err := auth.IssueTunnelInvitation(host)
	if err != nil {
		t.Fatal(err)
	}
	return auth, token, code
}

func TestAccessCodeFormatIsCrockfordBase32(t *testing.T) {
	auth, _, code := newAccessCodeFixture(t, "example.trycloudflare.com")
	defer func() { auth.SetTunnelActive(false) }()
	if len(code) != accessCodeLength {
		t.Fatalf("code length = %d, want %d", len(code), accessCodeLength)
	}
	for _, r := range code {
		if !strings.ContainsRune(accessCodeAlphabet, r) {
			t.Fatalf("code %q contains non-Crockford rune %q", code, r)
		}
	}
	seen := map[string]bool{}
	for i := 0; i < 10; i++ {
		_, generated, err := auth.IssueTunnelInvitation("example.trycloudflare.com")
		if err != nil {
			t.Fatal(err)
		}
		if seen[generated] {
			t.Fatalf("duplicate access code %q minted twice", generated)
		}
		seen[generated] = true
		// Consume the invitation so the pending-invitation cap is never hit.
		if sess, ok := auth.ExchangeAccessCode(generated, "example.trycloudflare.com"); !ok || sess == nil {
			t.Fatalf("minted code %q must authenticate", generated)
		}
	}
}

func TestExchangeAccessCodeCreatesSingleUseRemoteSession(t *testing.T) {
	auth, _, code := newAccessCodeFixture(t, "example.trycloudflare.com")
	defer func() { auth.SetTunnelActive(false) }()
	sess, ok := auth.ExchangeAccessCode(code, "example.trycloudflare.com")
	if !ok || sess == nil {
		t.Fatal("valid pairing code must mint a session")
	}
	if !sess.NonPersistable {
		t.Fatal("remote pairing session must never persist")
	}
	if second, ok := auth.ExchangeAccessCode(code, "example.trycloudflare.com"); ok || second != nil {
		t.Fatal("pairing code must be single-use")
	}
}

func TestExchangeAccessCodeNormalizesTypedInput(t *testing.T) {
	if got := normalizeAccessCode("ab1-2l o"); got != "AB1210" {
		t.Fatalf("normalizeAccessCode = %q, want %q", got, "AB1210")
	}
	host := "example.trycloudflare.com"
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	auth.SetTunnelActive(true)
	defer func() { auth.SetTunnelActive(false) }()
	// Craft an invitation whose code contains ambiguous digits, then prove a
	// typed variant using I/L/O still authenticates.
	auth.mu.Lock()
	auth.remoteInvitations = map[string]remoteInvitation{
		"crafted": {Host: host, ExpiresAt: auth.now().Add(remoteBootstrapTTL), AccessCode: "A1B0CDEF"},
	}
	auth.mu.Unlock()
	if sess, ok := auth.ExchangeAccessCode("aib0cdef", host); !ok || sess == nil {
		t.Fatalf("typed I/L/O remap must authenticate: input aib0cdef for A1B0CDEF, ok=%v", ok)
	}
}

func TestExchangeAccessCodeRejectsWrongCodeWithoutConsuming(t *testing.T) {
	auth, _, code := newAccessCodeFixture(t, "example.trycloudflare.com")
	defer func() { auth.SetTunnelActive(false) }()
	wrong := code
	if wrong[0] == 'A' {
		wrong = "B" + wrong[1:]
	} else {
		wrong = "A" + wrong[1:]
	}
	if sess, ok := auth.ExchangeAccessCode(wrong, "example.trycloudflare.com"); ok || sess != nil {
		t.Fatal("wrong code must not authenticate")
	}
	if sess, ok := auth.ExchangeAccessCode(code, "example.trycloudflare.com"); !ok || sess == nil {
		t.Fatal("a rejected guess must not consume the real invitation")
	}
}

func TestExchangeAccessCodeBindsToTunnelHost(t *testing.T) {
	auth, _, code := newAccessCodeFixture(t, "example.trycloudflare.com")
	defer func() { auth.SetTunnelActive(false) }()
	if sess, ok := auth.ExchangeAccessCode(code, "evil.trycloudflare.com"); ok || sess != nil {
		t.Fatal("code must be bound to the invitation host")
	}
	if sess, ok := auth.ExchangeAccessCode(code, "example.trycloudflare.com"); !ok || sess == nil {
		t.Fatal("host-rejected attempt must not consume the invitation")
	}
}

func TestExchangeAccessCodeRequiresActiveTunnel(t *testing.T) {
	auth, _, code := newAccessCodeFixture(t, "example.trycloudflare.com")
	auth.SetTunnelActive(false)
	if sess, ok := auth.ExchangeAccessCode(code, "example.trycloudflare.com"); ok || sess != nil {
		t.Fatal("pairing must fail closed when the tunnel is down")
	}
}

func TestExchangeAccessCodeBruteForceGuardRevokesInvitations(t *testing.T) {
	auth, _, code := newAccessCodeFixture(t, "example.trycloudflare.com")
	defer func() { auth.SetTunnelActive(false) }()
	for i := 0; i < accessCodeMaxFailures; i++ {
		if _, ok := auth.ExchangeAccessCode("ZZZZZZZZ", "example.trycloudflare.com"); ok {
			t.Fatal("wrong code must never authenticate")
		}
	}
	if sess, ok := auth.ExchangeAccessCode(code, "example.trycloudflare.com"); ok || sess != nil {
		t.Fatal("after the failure cap, even the real code must be revoked")
	}
}

func TestAccessCodePairEndpointAuthenticatesAndRejects(t *testing.T) {
	host := "example.trycloudflare.com"
	auth, token, code := newAccessCodeFixture(t, host)
	srv := &Server{auth: auth, tunnelHost: host}

	pair := func(code string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost,
			"https://"+host+"/api/v1/auth/pair",
			strings.NewReader(`{"code":"`+code+`"}`))
		req.Host = host
		rec := httptest.NewRecorder()
		srv.handleAuthPair(rec, req)
		return rec
	}

	if rec := pair(code); rec.Code != http.StatusOK {
		t.Fatalf("valid pairing code must return 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec := pair(code); rec.Code != http.StatusUnauthorized {
		t.Fatalf("replayed pairing code must return 401, got %d", rec.Code)
	}
	if rec := pair(strings.ToUpper(code[:4]) + "-" + strings.ToUpper(code[4:])); rec.Code != http.StatusUnauthorized {
		t.Fatalf("code was single-use; normalized replay must fail, got %d", rec.Code)
	}
	if rec := pair("WRONG123"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong code must return 401, got %d", rec.Code)
	}
	if rec := pair(token); rec.Code != http.StatusUnauthorized {
		t.Fatalf("bootstrap token must not authenticate as pairing code, got %d", rec.Code)
	}
	// GET must never mint or reveal credentials.
	get := httptest.NewRequest(http.MethodGet, "https://"+host+"/api/v1/auth/pair", nil)
	rec := httptest.NewRecorder()
	srv.handleAuthPair(rec, get)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET must not pair: got %d", rec.Code)
	}
}
