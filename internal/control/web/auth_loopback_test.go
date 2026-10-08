package web

import (
	"testing"
	"time"
)

func TestExchangeBootstrapTokenReusableOnLoopback(t *testing.T) {
	auth, token, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}

	first, ok := auth.ExchangeBootstrapToken(token)
	if !ok || first == nil {
		t.Fatal("first loopback bootstrap exchange must succeed")
	}
	second, ok := auth.ExchangeBootstrapToken(token)
	if !ok || second == nil {
		t.Fatal("loopback bootstrap token must remain reusable for local re-auth")
	}
	if first.ID == second.ID {
		t.Fatal("each exchange must mint a distinct session")
	}
	if auth.usedBootstrap {
		t.Fatal("loopback listen must not mark bootstrap as consumed")
	}
}

func TestExchangeBootstrapTokenOneTimeOnPrivateBind(t *testing.T) {
	auth, token, err := NewAuthManager("192.168.1.10", "13000")
	if err != nil {
		t.Fatal(err)
	}

	first, ok := auth.ExchangeBootstrapToken(token)
	if !ok || first == nil {
		t.Fatal("first private-bind bootstrap exchange must succeed")
	}
	if !auth.usedBootstrap {
		t.Fatal("private/remote bind must consume bootstrap after first use")
	}
	second, ok := auth.ExchangeBootstrapToken(token)
	if ok || second != nil {
		t.Fatal("private/remote bootstrap must be one-time")
	}
}

func TestIsLoopbackListen(t *testing.T) {
	cases := map[string]bool{
		"":          true,
		"localhost": true,
		"127.0.0.1": true,
		"::1":       true,
		"10.0.0.5":  false,
		"0.0.0.0":   false,
	}
	for host, want := range cases {
		auth := &AuthManager{listenHost: host}
		if got := auth.isLoopbackListen(); got != want {
			t.Fatalf("host %q: got %v want %v", host, got, want)
		}
	}
}

func TestExchangeBootstrapTokenOneTimeWhenTunnelActiveOnLoopback(t *testing.T) {
	auth, token, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}

	// Without tunnel: loopback bootstrap is reusable
	first, ok := auth.ExchangeBootstrapToken(token)
	if !ok || first == nil {
		t.Fatal("first loopback bootstrap exchange must succeed")
	}
	second, ok := auth.ExchangeBootstrapToken(token)
	if !ok || second == nil {
		t.Fatal("loopback bootstrap must be reusable without tunnel")
	}
	if first.ID == second.ID {
		t.Fatal("each exchange must mint a distinct session")
	}

	// Simulate tunnel activation — reset for clean state
	auth2, token2, err := NewAuthManager("127.0.0.1", "13001")
	if err != nil {
		t.Fatal(err)
	}
	auth2.SetTunnelActive(true)

	first2, ok := auth2.ExchangeBootstrapToken(token2)
	if !ok || first2 == nil {
		t.Fatal("first tunnel-active bootstrap exchange must succeed")
	}
	if !auth2.usedBootstrap {
		t.Fatal("tunnel-active loopback must consume bootstrap")
	}
	second2, ok := auth2.ExchangeBootstrapToken(token2)
	if ok || second2 != nil {
		t.Fatal("tunnel-active bootstrap must be one-time even on loopback")
	}
}

func TestSetTunnelActiveControlsConsumption(t *testing.T) {
	auth, token, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}

	// Tunnel not active: reusable
	_, ok := auth.ExchangeBootstrapToken(token)
	if !ok {
		t.Fatal("first exchange must succeed")
	}
	_, ok = auth.ExchangeBootstrapToken(token)
	if !ok {
		t.Fatal("reusable on loopback without tunnel")
	}

	// Activate tunnel: now one-time
	auth.SetTunnelActive(true)
	auth.usedBootstrap = false // reset for clean test
	_, ok = auth.ExchangeBootstrapToken(token)
	if !ok {
		t.Fatal("first tunnel-active exchange must succeed")
	}
	_, ok = auth.ExchangeBootstrapToken(token)
	if ok {
		t.Fatal("must be one-time when tunnel is active")
	}
}

func TestTunnelInviteWorksAfterServerBootstrapExpired(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil { t.Fatal(err) }
	auth.SetTunnelActive(true)
	now := time.Now()
	auth.clock = func() time.Time { return now }
	auth.bootstrapCreatedAt = now.Add(-remoteBootstrapTTL - time.Hour)
	token, err := auth.IssueTunnelInvitation("example.trycloudflare.com")
	if err != nil { t.Fatal(err) }
	if sess, ok := auth.ExchangeBootstrapTokenForHost(token, "example.trycloudflare.com"); !ok || sess == nil {
		t.Fatal("fresh invitation must work even when process bootstrap is expired")
	}
	if _, ok := auth.ExchangeBootstrapTokenForHost(token, "example.trycloudflare.com"); ok {
		t.Fatal("one-time invitation reused")
	}
}

func TestTunnelInviteHostExpiryAndStop(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil { t.Fatal(err) }
	auth.SetTunnelActive(true)
	now := time.Now()
	auth.clock = func() time.Time { return now }
	token, err := auth.IssueTunnelInvitation("example.trycloudflare.com")
	if err != nil { t.Fatal(err) }
	if _, ok := auth.ExchangeBootstrapTokenForHost(token, "evil.trycloudflare.com"); ok {
		t.Fatal("wrong hostname should be rejected")
	}
	now = now.Add(remoteBootstrapTTL + time.Second)
	if _, ok := auth.ExchangeBootstrapTokenForHost(token, "example.trycloudflare.com"); ok {
		t.Fatal("expired invitation accepted")
	}
	token, err = auth.IssueTunnelInvitation("example.trycloudflare.com")
	if err != nil { t.Fatal(err) }
	auth.SetTunnelActive(false)
	if _, ok := auth.ExchangeBootstrapTokenForHost(token, "example.trycloudflare.com"); ok {
		t.Fatal("stopped tunnel must revoke invitation")
	}
	if _, err := auth.IssueTunnelInvitation("example.trycloudflare.com"); err == nil {
		t.Fatal("cannot issue invitation when tunnel is offline")
	}
}

func TestExchangeBootstrapTokenExpiresOnLoopback(t *testing.T) {
	auth, token, err := NewAuthManager("127.0.0.1", "13000")
	if err != nil {
		t.Fatal(err)
	}
	auth.bootstrapCreatedAt = time.Now().Add(-loopbackBootstrapTTL - time.Second)
	if _, ok := auth.ExchangeBootstrapToken(token); ok {
		t.Fatal("expired loopback bootstrap must be rejected")
	}
}
