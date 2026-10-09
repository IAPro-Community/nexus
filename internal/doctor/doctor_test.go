package doctor

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

func TestBuildReportIsReadOnlyAndContainsStableChecks(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	before, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	report := BuildReport("test", map[string]model.DetectionResult{"codex": {Installed: false, Error: "missing"}}, nexusruntime.CredentialCapability{Status: nexusruntime.CredentialUnsupported, Mechanism: "test"})
	after, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) || len(report.Checks) == 0 {
		t.Fatalf("doctor mutated state or returned no checks: before=%d after=%d", len(before), len(after))
	}
}

func TestBuildReportWarnsWhenInstalledProviderVersionProbeFails(t *testing.T) {
	report := BuildReport("test", map[string]model.DetectionResult{
		"gemini": {
			Installed:  true,
			Version:    "unknown",
			BinaryPath: "gemini",
			Error:      "version probe failed: exit status 1",
		},
	}, nexusruntime.CredentialCapability{Status: nexusruntime.CredentialUnsupported, Mechanism: "test"})

	for _, check := range report.Checks {
		if check.ID != "provider.gemini" {
			continue
		}
		if check.Status != Warn || !strings.Contains(check.Summary, "version probe failed") {
			t.Fatalf("failed installed provider probe must be WARN, got %+v", check)
		}
		if check.Remediation == "" {
			t.Fatalf("failed provider probe needs remediation: %+v", check)
		}
		return
	}
	t.Fatal("provider.gemini check missing")
}

func TestBuildReportWarnsWhenInstalledProviderVersionIsUnknown(t *testing.T) {
	report := BuildReport("test", map[string]model.DetectionResult{
		"gemini": {Installed: true, Version: "unknown", BinaryPath: "gemini"},
	}, nexusruntime.CredentialCapability{Status: nexusruntime.CredentialUnsupported, Mechanism: "test"})

	for _, check := range report.Checks {
		if check.ID == "provider.gemini" {
			if check.Status != Warn || !strings.Contains(check.Summary, "version unavailable") {
				t.Fatalf("unknown installed provider version must be WARN, got %+v", check)
			}
			return
		}
	}
	t.Fatal("provider.gemini check missing")
}

func TestBuildReportDoesNotClaimDesktopShellRuntimeWasVerified(t *testing.T) {
	report := BuildReport("test", nil, nexusruntime.CredentialCapability{
		Status:    nexusruntime.CredentialUnsupported,
		Mechanism: "test",
	})
	for _, check := range report.Checks {
		if check.ID == "desktop.shell" {
			if check.Status != Skipped {
				t.Fatalf("desktop shell must remain unverified without native smoke, got %s", check.Status)
			}
			return
		}
	}
	t.Fatal("desktop.shell check missing")
}

func TestBuildReportPlatformChecksAreEvidenceBound(t *testing.T) {
	t.Setenv("NEXUS_DOCKER", "")
	t.Setenv("NEXUS_COMPAT_DOCKER", "")
	t.Setenv("NEXUS_TEST_NOT_CONTAINER", "1")
	report := BuildReport("test", nil, nexusruntime.CredentialCapability{
		Status:    nexusruntime.CredentialUnsupported,
		Mechanism: "test",
	})
	for _, check := range report.Checks {
		if check.ID == "platform.conpty" && stdruntime.GOOS != "windows" {
			t.Fatalf("ConPTY check must not be emitted on %s", stdruntime.GOOS)
		}
		if check.ID == "platform.webview2" && stdruntime.GOOS != "windows" {
			t.Fatalf("WebView2 check must not be emitted on %s", stdruntime.GOOS)
		}
		if check.ID == "platform.webkitgtk" && stdruntime.GOOS != "linux" {
			t.Fatalf("WebKitGTK check must not be emitted on %s", stdruntime.GOOS)
		}
	}
}

func TestBuildReportContainerSkipsNativeDesktopProbes(t *testing.T) {
	t.Setenv("NEXUS_TEST_NOT_CONTAINER", "")
	t.Setenv("NEXUS_DOCKER", "1")
	report := BuildReport("test", map[string]model.DetectionResult{
		"claude": {Installed: true, Version: "fixture"},
	}, nexusruntime.CredentialCapability{Status: nexusruntime.CredentialUnsupported, Mechanism: "test"})

	var desktop, webview, runtimeCheck *Check
	for i := range report.Checks {
		c := &report.Checks[i]
		switch c.ID {
		case "desktop.shell":
			desktop = c
		case "platform.webview2":
			webview = c
		case "runtime.container":
			runtimeCheck = c
		}
	}
	if runtimeCheck == nil || runtimeCheck.Status != Pass {
		t.Fatalf("expected runtime.container PASS, got %#v", runtimeCheck)
	}
	if desktop == nil || desktop.Status != Skipped || !strings.Contains(desktop.Summary, "N/A in container") {
		t.Fatalf("desktop.shell must be N/A in container, got %#v", desktop)
	}
	if webview == nil || webview.Status != Skipped {
		t.Fatalf("platform.webview2 must be SKIPPED in container, got %#v", webview)
	}
}

func TestHasWebView2RegistrationRequiresRuntimeNameAndVersion(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want bool
	}{
		{
			name: "webview2 runtime",
			out: "HKEY_LOCAL_MACHINE\\SOFTWARE\\WOW6432Node\\Microsoft\\EdgeUpdate\\Clients\\{F3017226-E4C5}\n" +
				"    name    REG_SZ    Microsoft Edge WebView2 Runtime\n" +
				"    pv      REG_SZ    154.0.4258.37\n",
			want: true,
		},
		{
			name: "edge browser only",
			out: "HKEY_LOCAL_MACHINE\\SOFTWARE\\WOW6432Node\\Microsoft\\EdgeUpdate\\Clients\\{56EB18F8-B008}\n" +
				"    name    REG_SZ    Microsoft Edge\n" +
				"    pv      REG_SZ    154.0.4258.48\n",
			want: false,
		},
		{
			name: "filtered version output omits runtime name",
			out: "HKEY_LOCAL_MACHINE\\SOFTWARE\\WOW6432Node\\Microsoft\\EdgeUpdate\\Clients\\{F3017226-E4C5}\n" +
				"    pv      REG_SZ    154.0.4258.37\n",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasWebView2Registration(tt.out); got != tt.want {
				t.Fatalf("hasWebView2Registration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteBundleAllowlistAndRedactionSurface(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doctor.zip")
	report := Report{Schema: "nexus.doctor/v1", Version: "test", Providers: map[string]model.DetectionResult{"codex": {Installed: true, BinaryPath: "/tmp/codex"}}}
	if err := report.WriteBundle(path); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if len(archive.File) != 2 {
		t.Fatalf("unexpected bundle entries: %d", len(archive.File))
	}
	for _, entry := range archive.File {
		if entry.Name != "report.json" && entry.Name != "MANIFEST.txt" {
			t.Fatalf("unexpected bundle entry %q", entry.Name)
		}
		reader, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if string(data) == "SECRET_CANARY" {
			t.Fatal("diagnostic bundle leaked secret canary")
		}
	}
}
