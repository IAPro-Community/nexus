package versionprobe

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/model"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

// Detect runs the standard CLI version probe and keeps installed-but-broken
// tools distinct from providers that reported a usable version.
func Detect(ctx context.Context, binary string) model.DetectionResult {
	output, err := nexusruntime.RunCommandCapture(ctx, binary, []string{"--version"}, os.Environ(), "")
	return resultFromProbe(binary, output, err)
}

func resultFromProbe(binary, output string, err error) model.DetectionResult {
	if err != nil {
		return model.DetectionResult{
			Installed:  true,
			Version:    "unknown",
			BinaryPath: binary,
			Error:      fmt.Sprintf("version probe failed: %v", err),
		}
	}

	version := strings.TrimSpace(output)
	if version == "" {
		return model.DetectionResult{
			Installed:  true,
			Version:    "unknown",
			BinaryPath: binary,
			Error:      "version probe returned no output",
		}
	}

	return model.DetectionResult{Installed: true, Version: version, BinaryPath: binary}
}
