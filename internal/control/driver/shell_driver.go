package driver

import (
	"context"
	"os"
	"runtime"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

// ShellDriver adapts the existing supervised terminal runtime to a normal
// project shell. It deliberately has no Agent or provider-session identity.
type ShellDriver struct{ FakeDriver }

func NewShellDriver() *ShellDriver        { return &ShellDriver{} }
func (d *ShellDriver) ProviderID() string { return "shell" }

func (d *ShellDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	bin, _, _, err := d.BuildCommand(ctx, model.Profile{Provider: "shell"}, nil)
	return model.DetectionResult{Installed: err == nil, BinaryPath: bin, Version: "system-shell"}, err
}

func (d *ShellDriver) BuildCommand(_ context.Context, _ model.Profile, _ []string) (string, []string, []string, error) {
	if runtime.GOOS == "windows" {
		bin := os.Getenv("ComSpec")
		if bin == "" {
			bin = "cmd.exe"
		}
		return bin, []string{"/Q"}, os.Environ(), nil
	}
	bin := os.Getenv("SHELL")
	if bin == "" {
		bin = "/bin/sh"
	}
	return bin, []string{"-i"}, os.Environ(), nil
}
