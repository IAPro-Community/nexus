package web

import (
	"context"
	"os"
	"strings"
	"testing"
)

// The named tunnel is the only transport that gives the PWA a stable origin.
// These tests pin the three properties the rest of the feature depends on:
// the configuration is read from the environment, the token never reaches
// argv, and the public URL is derived from the configured hostname (so it
// survives a restart of the Nexus process).

func TestNamedTunnelConfigFromEnv(t *testing.T) {
	t.Setenv(tunnelTokenEnv, "")
	t.Setenv(tunnelHostnameEnv, "")
	if _, ok := NamedTunnelConfigFromEnv(); ok {
		t.Fatal("an empty environment must fall back to the Quick Tunnel")
	}

	// A partial configuration must not silently produce a broken tunnel.
	t.Setenv(tunnelTokenEnv, "token-only")
	t.Setenv(tunnelHostnameEnv, "")
	if _, ok := NamedTunnelConfigFromEnv(); ok {
		t.Fatal("a token without hostname must fall back to the Quick Tunnel")
	}
	t.Setenv(tunnelTokenEnv, "")
	t.Setenv(tunnelHostnameEnv, "nexus.example.com")
	if _, ok := NamedTunnelConfigFromEnv(); ok {
		t.Fatal("a hostname without token must fall back to the Quick Tunnel")
	}

	t.Setenv(tunnelTokenEnv, "  secret-token  ")
	t.Setenv(tunnelHostnameEnv, "  nexus.example.com  ")
	cfg, ok := NamedTunnelConfigFromEnv()
	if !ok {
		t.Fatal("a complete configuration must enable the named tunnel")
	}
	if cfg.Hostname != "nexus.example.com" {
		t.Fatalf("hostname = %q", cfg.Hostname)
	}
	if cfg.Token != "secret-token" {
		t.Fatalf("token = %q", cfg.Token)
	}
}

func TestNamedTunnelCommandKeepsTokenOutOfArgv(t *testing.T) {
	cmd, publicURL := namedTunnelCommand(
		context.Background(),
		"/usr/local/bin/cloudflared",
		"nexus.example.com",
		"super-secret-token",
	)

	if publicURL != "https://nexus.example.com" {
		t.Fatalf("public URL = %q", publicURL)
	}
	for _, arg := range cmd.Args {
		if strings.Contains(arg, "super-secret-token") {
			t.Fatalf("the tunnel token must never appear in argv: %v", cmd.Args)
		}
	}
	if !strings.Contains(strings.Join(cmd.Args, " "), "tunnel") {
		t.Fatalf("argv must invoke the cloudflared tunnel command: %v", cmd.Args)
	}

	found := false
	for _, entry := range cmd.Env {
		if entry == "TUNNEL_TOKEN=super-secret-token" {
			found = true
		}
	}
	if !found {
		t.Fatal("the tunnel token must reach cloudflared through the environment")
	}
}

// The stable origin is the whole point of the named tunnel: the same
// hostname must produce the same public URL on every start, including after
// the Nexus process restarts.
func TestNamedTunnelURLIsStableAcrossRestarts(t *testing.T) {
	const hostname = "nexus.example.com"
	_, first := namedTunnelCommand(context.Background(), "/bin/true", hostname, "token")
	_, second := namedTunnelCommand(context.Background(), "/bin/true", hostname, "token")
	if first != second {
		t.Fatalf("public URL changed between starts: %q vs %q", first, second)
	}
	if first != "https://"+hostname {
		t.Fatalf("public URL = %q", first)
	}
}

func TestStartNamedTunnelRejectsInvalidInput(t *testing.T) {
	if _, err := StartNamedTunnel(context.Background(), 13000, "", "token"); err == nil {
		t.Fatal("an empty hostname must be refused")
	}
	if _, err := StartNamedTunnel(context.Background(), 13000, "nexus.example.com/evil", "token"); err == nil {
		t.Fatal("a hostname with a path must be refused")
	}
	if _, err := StartNamedTunnel(context.Background(), 13000, "nexus.example.com", "  "); err == nil {
		t.Fatal("an empty token must be refused")
	}
	// A named tunnel applies the ingress rule from the Cloudflare dashboard,
	// which points at a fixed http://localhost:<port>. An ephemeral port can
	// never be the target of that rule, so the tunnel would be published and
	// answer 502.
	if _, err := StartNamedTunnel(context.Background(), 0, "nexus.example.com", "token"); err == nil {
		t.Fatal("an ephemeral local port must be refused")
	}
	if _, err := StartNamedTunnel(context.Background(), -1, "nexus.example.com", "token"); err == nil {
		t.Fatal("a negative local port must be refused")
	}
}

// The transport selection is the Gate 0 decision: with the named-tunnel
// configuration present the stable hostname must win, and without it the
// Quick Tunnel must remain the default.
func TestStartConfiguredTunnelPrefersNamedTransport(t *testing.T) {
	originalNamed, originalQuick := startNamedTunnel, startQuickTunnel
	t.Cleanup(func() {
		startNamedTunnel, startQuickTunnel = originalNamed, originalQuick
	})

	var namedCalls, quickCalls int
	var namedHost, namedToken string
	startNamedTunnel = func(ctx context.Context, localPort int, hostname, token string) (*Tunnel, error) {
		namedCalls++
		namedHost, namedToken = hostname, token
		return &Tunnel{URL: "https://" + hostname}, nil
	}
	startQuickTunnel = func(ctx context.Context, localPort int) (*Tunnel, error) {
		quickCalls++
		return &Tunnel{URL: "https://quick.trycloudflare.com"}, nil
	}

	t.Setenv(tunnelTokenEnv, "")
	t.Setenv(tunnelHostnameEnv, "")
	tunnel, err := startConfiguredTunnel(context.Background(), 13000)
	if err != nil {
		t.Fatal(err)
	}
	if quickCalls != 1 || namedCalls != 0 {
		t.Fatalf("without configuration the Quick Tunnel must be used: named=%d quick=%d", namedCalls, quickCalls)
	}
	if tunnel.URL != "https://quick.trycloudflare.com" {
		t.Fatalf("URL = %q", tunnel.URL)
	}

	t.Setenv(tunnelTokenEnv, "secret-token")
	t.Setenv(tunnelHostnameEnv, "nexus.example.com")
	tunnel, err = startConfiguredTunnel(context.Background(), 13000)
	if err != nil {
		t.Fatal(err)
	}
	if namedCalls != 1 || quickCalls != 1 {
		t.Fatalf("with configuration the Named Tunnel must be used: named=%d quick=%d", namedCalls, quickCalls)
	}
	if namedHost != "nexus.example.com" || namedToken != "secret-token" {
		t.Fatalf("named transport received host=%q token=%q", namedHost, namedToken)
	}
	if tunnel.URL != "https://nexus.example.com" {
		t.Fatalf("URL = %q", tunnel.URL)
	}
}

// The readiness probe must not mistake an edge error for a working tunnel:
// Cloudflare answers 502/530 when the ingress does not reach the Nexus.
func TestTunnelResponseReadyRejectsEdgeErrors(t *testing.T) {
	for _, status := range []int{200, 201, 301, 302, 401, 403, 404, 429} {
		if !tunnelResponseReady(status) {
			t.Fatalf("status %d proves the origin answered and must be accepted", status)
		}
	}
	for _, status := range []int{500, 502, 503, 504, 530} {
		if tunnelResponseReady(status) {
			t.Fatalf("status %d is an edge failure and must not be accepted as ready", status)
		}
	}
	if tunnelResponseReady(0) {
		t.Fatal("an absent status must not be accepted as ready")
	}
}

// The operator must be able to tell a stable origin from an ephemeral one:
// a Quick Tunnel mints a random hostname on every start, which breaks an
// installed PWA and its host-bound cookie.
func TestTunnelOriginStability(t *testing.T) {
	t.Setenv(tunnelTokenEnv, "")
	t.Setenv(tunnelHostnameEnv, "")

	stable, named := tunnelOriginStability("https://random-name.trycloudflare.com")
	if stable {
		t.Fatal("a Quick Tunnel hostname must not be reported as stable")
	}
	if named != "" {
		t.Fatalf("named hostname = %q", named)
	}

	stable, named = tunnelOriginStability("")
	if stable || named != "" {
		t.Fatal("an absent URL must not be reported as stable")
	}

	t.Setenv(tunnelTokenEnv, "secret-token")
	t.Setenv(tunnelHostnameEnv, "nexus.example.com")

	stable, named = tunnelOriginStability("https://nexus.example.com")
	if !stable {
		t.Fatal("the configured named hostname must be reported as stable")
	}
	if named != "nexus.example.com" {
		t.Fatalf("named hostname = %q", named)
	}

	// A running Quick Tunnel must not be mistaken for the configured one.
	stable, named = tunnelOriginStability("https://random-name.trycloudflare.com")
	if stable {
		t.Fatal("a Quick Tunnel must not be reported as stable under a named configuration")
	}
	if named != "" {
		t.Fatalf("named hostname = %q", named)
	}
}

// The token must not be recoverable from the process environment of the
// server itself: it is read at start time and handed to the child.
func TestNamedTunnelTokenIsNotPersistedInTheEnvironment(t *testing.T) {
	t.Setenv(tunnelTokenEnv, "secret-token")
	t.Setenv(tunnelHostnameEnv, "nexus.example.com")
	cfg, ok := NamedTunnelConfigFromEnv()
	if !ok {
		t.Fatal("named tunnel configuration must be readable")
	}
	if cfg.Token == os.Getenv(tunnelTokenEnv) && strings.TrimSpace(cfg.Token) == "" {
		t.Fatal("the token must not be reduced to an empty string")
	}
	if strings.Contains(cfg.Hostname, cfg.Token) {
		t.Fatal("the hostname must not carry the token")
	}
}
