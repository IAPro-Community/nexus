package versionprobe

import (
	"errors"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestResultForProbeFailureDoesNotExposeCapturedOutput(t *testing.T) {
	result := resultFromProbe("gemini", "SyntaxError: private diagnostic", errors.New("exit status 1"))
	if !result.Installed || result.BinaryPath != "gemini" || result.Version != "unknown" {
		t.Fatalf("unexpected installed provider result: %+v", result)
	}
	if result.Error == "" || strings.Contains(result.Error, "private diagnostic") {
		t.Fatalf("probe failure should be reported without copying combined output: %+v", result)
	}
}

func TestResultForProbeSuccessTrimsVersion(t *testing.T) {
	result := resultFromProbe("codex", "  codex-cli 1.2.3\r\n", nil)
	want := model.DetectionResult{Installed: true, Version: "codex-cli 1.2.3", BinaryPath: "codex"}
	if result != want {
		t.Fatalf("result=%+v, want %+v", result, want)
	}
}

func TestResultForEmptyProbeOutputIsUnknown(t *testing.T) {
	result := resultFromProbe("agy", " \r\n", nil)
	if !result.Installed || result.Version != "unknown" || result.Error == "" {
		t.Fatalf("empty output should remain installed but unverified: %+v", result)
	}
}
