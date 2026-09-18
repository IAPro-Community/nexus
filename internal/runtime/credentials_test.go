package runtime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultCredentialIsolatorPlatform(t *testing.T) {
	iso := DefaultCredentialIsolator()
	if iso == nil {
		t.Fatal("expected non-nil isolator")
	}
	if iso.Platform() != runtime.GOOS {
		t.Fatalf("expected platform %q, got %q", runtime.GOOS, iso.Platform())
	}
}

func TestCredentialIsolationWrapCommand(t *testing.T) {
	iso := DefaultCredentialIsolator()
	bin, args := iso.WrapCommand("echo", []string{"hi"})
	if bin == "" {
		t.Fatal("expected non-empty bin")
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		if bin != "echo" || len(args) != 1 || args[0] != "hi" {
			t.Fatalf("expected passthrough on %s, got bin=%q, args=%v", runtime.GOOS, bin, args)
		}
	}
}

func TestCredentialCapabilityIsTruthfulOnUnsupportedPlatforms(t *testing.T) {
	capability := DefaultCredentialIsolator().Capability()
	if capability.Mechanism == "" || capability.Status == "" {
		t.Fatalf("credential capability must be explicit: %+v", capability)
	}
	if (runtime.GOOS == "windows" || runtime.GOOS == "darwin") && capability.Status == CredentialSupported {
		t.Fatalf("unsupported native integration must not claim support: %+v", capability)
	}
}

func TestNoBrowserBinDirInstallsInertHelpers(t *testing.T) {
	data := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_CLI_DATA_DIR", data)

	dir, err := NoBrowserBinDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ai-browser", "xdg-open", "open"} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if runtime.GOOS == "windows" {
			continue
		}
		if info.Mode()&0111 == 0 {
			t.Fatalf("%s must be executable", name)
		}
	}
}

func TestDisableSessionSecretServiceFailsClosedWithoutLeakingHostBus(t *testing.T) {
	env := DisableSessionSecretService([]string{
		"PATH=/usr/bin",
		"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus",
		"AI_HOST_DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1000/bus",
		"GNOME_KEYRING_CONTROL=/run/user/1000/keyring",
		"GNOME_KEYRING_PID=1234",
	})

	values := make(map[string]string)
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[key] = value
		}
	}
	if got := values["DBUS_SESSION_BUS_ADDRESS"]; got != "unix:path=/dev/null" {
		t.Fatalf("session bus must fail closed, got %q", got)
	}
	for _, key := range []string{"AI_HOST_DBUS_SESSION_BUS_ADDRESS", "GNOME_KEYRING_CONTROL", "GNOME_KEYRING_PID"} {
		if _, ok := values[key]; ok {
			t.Errorf("sensitive desktop credential route %s must be removed", key)
		}
	}
	if got := values["PATH"]; got != "/usr/bin" {
		t.Fatalf("unrelated environment changed: PATH=%q", got)
	}
}
