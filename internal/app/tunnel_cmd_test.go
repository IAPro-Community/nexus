package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readTunnelCommandSource(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("tunnel_cmd.go"))
	if err != nil {
		t.Fatal(err)
	}
	return string(source)
}

// The CLI is an entry point that opens a public tunnel. It must follow the
// same transport selection as the Settings button, otherwise the stable-URL
// configuration would be silently ignored and the CLI would hand out a random
// hostname that breaks the app installed on the phone.
func TestTunnelCommandUsesConfiguredTransport(t *testing.T) {
	text := readTunnelCommandSource(t)
	if strings.Contains(text, "web.StartTunnel(") {
		t.Fatal("the tunnel command must not start a Quick Tunnel directly; use web.StartConfiguredTunnel")
	}
	if !strings.Contains(text, "web.StartConfiguredTunnel(") {
		t.Fatal("the tunnel command must start the transport selected by the environment")
	}
}

// The command must also say which transport it is opening, so the operator
// does not have to infer it from the hostname.
func TestTunnelCommandReportsTheSelectedTransport(t *testing.T) {
	text := readTunnelCommandSource(t)
	if !strings.Contains(text, "web.NamedTunnelConfigFromEnv()") {
		t.Fatal("the tunnel command must report the Named Tunnel when the stable-URL configuration is present")
	}
	if !strings.Contains(text, "Named Tunnel") || !strings.Contains(text, "Quick Tunnel") {
		t.Fatal("the tunnel command must name both transports")
	}
}

// The help text must document the stable-URL configuration: it is the only
// place an operator reads before the first run.
func TestTunnelHelpDocumentsTheStableURL(t *testing.T) {
	text := readTunnelCommandSource(t)
	for _, token := range []string{"NEXUS_TUNNEL_TOKEN", "NEXUS_TUNNEL_HOSTNAME"} {
		if !strings.Contains(text, token) {
			t.Fatalf("the tunnel help must document %s", token)
		}
	}
}
