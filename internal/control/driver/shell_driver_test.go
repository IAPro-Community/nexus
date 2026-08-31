package driver

import (
	"context"
	"runtime"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestShellDriverBuildsPlatformShell(t *testing.T) {
	d := NewShellDriver()
	if d.ProviderID() != "shell" {
		t.Fatalf("provider id = %q, want shell", d.ProviderID())
	}
	bin, args, _, err := d.BuildCommand(context.Background(), model.Profile{Provider: "shell", Name: "default"}, nil)
	if err != nil {
		t.Fatalf("build shell command: %v", err)
	}
	if bin == "" || len(args) == 0 {
		t.Fatalf("shell command is incomplete: %q %v", bin, args)
	}
	if runtime.GOOS == "windows" && args[0] != "/Q" {
		t.Fatalf("windows shell args = %v, want /Q", args)
	}
	if runtime.GOOS != "windows" && args[0] != "-i" {
		t.Fatalf("unix shell args = %v, want -i", args)
	}
}
