package app

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/control/web"
)

func TestParseControlWebArgsHelpDoesNotSelectServerStartup(t *testing.T) {
	options, err := parseControlWebArgs([]string{"--help"}, "")
	if err != nil {
		t.Fatalf("parse help: %v", err)
	}
	if !options.help {
		t.Fatal("--help must be returned as a help request")
	}
	if options.port != web.DefaultPort || options.host != "127.0.0.1" {
		t.Fatalf("help must not change startup defaults: %+v", options)
	}
}

func TestParseControlWebArgsRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "unknown flag", args: []string{"--unknown"}},
		{name: "unexpected positional argument", args: []string{"unexpected"}},
		{name: "missing port value", args: []string{"--port"}},
		{name: "invalid port value", args: []string{"--port", "not-a-port"}},
		{name: "out of range port", args: []string{"--port", "65536"}},
		{name: "missing listen value", args: []string{"--listen"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseControlWebArgs(test.args, ""); err == nil {
				t.Fatalf("parseControlWebArgs(%q) must reject the invalid input", test.args)
			}
		})
	}
}

func TestParseControlWebArgsAcceptsSupportedFlags(t *testing.T) {
	options, err := parseControlWebArgs([]string{"-p", "0", "-l", "127.0.0.1", "--no-open", "--remote"}, "")
	if err != nil {
		t.Fatalf("parse supported flags: %v", err)
	}
	if options.port != 0 || options.host != "127.0.0.1" || !options.noOpen || !options.remote {
		t.Fatalf("parsed options = %+v", options)
	}
}

func TestParseControlWebArgsUsesValidEnvironmentPort(t *testing.T) {
	options, err := parseControlWebArgs(nil, "13042")
	if err != nil {
		t.Fatalf("parse default options: %v", err)
	}
	if options.port != 13042 {
		t.Fatalf("environment port = %d, want 13042", options.port)
	}
}

func TestRunWebHelpDoesNotStartServer(t *testing.T) {
	t.Setenv("NEXUS_CONFIG_DIR", t.TempDir())
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("NEXUS_STATE_DIR", t.TempDir())
	t.Setenv("NEXUS_WEB_PORT", "43129")
	t.Setenv("NEXUS_HUD_SPAWN", "0")

	output, err := captureStdout(func() error {
		return Run([]string{"web", "--help"})
	})
	if err != nil {
		t.Fatalf("Run web help: %v", err)
	}
	if !strings.Contains(output, "Nexus Web Workspace OS") || !strings.Contains(output, "USAGE:") {
		t.Fatalf("web help output is missing usage text: %q", output)
	}
}

func TestRunWebRejectsUnknownFlagsBeforeStartup(t *testing.T) {
	t.Setenv("NEXUS_CONFIG_DIR", t.TempDir())
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("NEXUS_STATE_DIR", t.TempDir())
	t.Setenv("NEXUS_WEB_PORT", "43129")
	t.Setenv("NEXUS_HUD_SPAWN", "0")

	err := Run([]string{"web", "--unknown"})
	if err == nil || !strings.Contains(err.Error(), "invalid web option") {
		t.Fatalf("Run web with unknown flag must reject before startup, got %v", err)
	}
}

func TestWaitForOwnedContinuityClaimBlocksUntilClaimSettles(t *testing.T) {
	reg := registry.NewRegistry(t.TempDir() + "/runtimes.json")
	if err := reg.Register(registry.RuntimeSession{
		RuntimeID: "runtime-1",
		State:     registry.StateHandoff,
		Labels: map[string]string{
			registry.ContinuityInFlightLabel: "true",
			registry.ContinuityOwnerPIDLabel: strconv.Itoa(os.Getpid()),
		},
	}); err != nil {
		t.Fatalf("register runtime: %v", err)
	}

	released := make(chan struct{})
	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = reg.UpdateLabels("runtime-1", map[string]string{
			registry.ContinuityInFlightLabel: "false",
		})
		close(released)
	}()

	deadline := time.Now().Add(5 * time.Second)
	waitForOwnedContinuityClaimUntil(reg, "runtime-1", deadline)
	<-released

	session, ok := reg.Get("runtime-1")
	if !ok {
		t.Fatal("runtime must stay registered")
	}
	if session.Labels[registry.ContinuityInFlightLabel] != "false" {
		t.Fatalf("claim must be settled before the host exits: %+v", session.Labels)
	}
}

func TestWaitForOwnedContinuityClaimIgnoresForeignOwner(t *testing.T) {
	reg := registry.NewRegistry(t.TempDir() + "/runtimes.json")
	if err := reg.Register(registry.RuntimeSession{
		RuntimeID: "runtime-1",
		State:     registry.StateHandoff,
		Labels: map[string]string{
			registry.ContinuityInFlightLabel: "true",
			registry.ContinuityOwnerPIDLabel: strconv.Itoa(os.Getpid() + 1),
		},
	}); err != nil {
		t.Fatalf("register runtime: %v", err)
	}

	done := make(chan struct{})
	go func() {
		waitForOwnedContinuityClaimUntil(reg, "runtime-1", time.Now().Add(5*time.Second))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("host must not wait for a continuity claim owned by another process")
	}
}
