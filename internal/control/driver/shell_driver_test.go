package driver

import (
	"context"
	"os"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestShellDriverIsTerminalOnlyNotAIProvider(t *testing.T) {
	d := NewShellDriver()
	caps := d.EffectiveCaps(context.Background(), model.Profile{Name: "local", Provider: "shell"})
	if caps.Process.Status != CapabilitySupported || caps.Terminal.Status != CapabilitySupported || caps.Attach.Status != CapabilitySupported {
		t.Fatalf("terminal capabilities missing: %+v", caps)
	}
	if caps.SubmitPrompt.Status != CapabilityUnsupported || caps.Sessions.Status != CapabilityUnsupported || caps.Resume.Status != CapabilityUnsupported || caps.Headless.Status != CapabilityUnsupported {
		t.Fatalf("shell must not claim AI/session capability: %+v", caps)
	}
	binary, _, env, err := d.BuildCommand(context.Background(), model.Profile{}, nil)
	if err != nil || binary == "" || len(env) == 0 {
		t.Fatalf("system shell not resolved: %q env=%d err=%v", binary, len(env), err)
	}
}

func TestShellDriverFallsBackFromStaleShellPath(t *testing.T) {
	t.Setenv("SHELL", "/definitely/missing/nexus-shell")

	binary, _, _, err := NewShellDriver().BuildCommand(context.Background(), model.Profile{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if binary == "/definitely/missing/nexus-shell" {
		t.Fatal("stale SHELL path must not be selected")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("fallback shell %q is not executable: %v", binary, err)
	}
}
