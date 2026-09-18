package driver

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/testutil"
)

func TestAGYBuildCommandDoesNotStartKeyringByDefault(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_AGY_ENABLE_SECRET_SERVICE", "")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/run/user/1000/bus")
	t.Setenv("GNOME_KEYRING_CONTROL", "/run/user/1000/keyring")
	t.Setenv("XDG_CONFIG_HOME", "/host/config")
	t.Setenv("XDG_CACHE_HOME", "/host/cache")
	binDir := t.TempDir()
	bin := testutil.WriteFakeBinary(t, binDir, "agy", "")
	t.Setenv("PATH", binDir)

	gotBin, gotArgs, env, err := (NewAGYDriver()).BuildCommand(context.Background(), model.Profile{Name: "work", Provider: "agy"}, []string{"-p", "hello"})
	if err != nil {
		t.Fatalf("BuildCommand: %v", err)
	}
	if gotBin != bin {
		t.Fatalf("binary=%q, want direct AGY binary %q", gotBin, bin)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "-p" {
		t.Fatalf("args=%v, want direct provider args", gotArgs)
	}
	values := map[string]string{}
	for _, item := range env {
		if k, v, ok := strings.Cut(item, "="); ok {
			values[k] = v
		}
	}
	if got := values["DBUS_SESSION_BUS_ADDRESS"]; got != "unix:path=/dev/null" {
		t.Fatalf("DBUS_SESSION_BUS_ADDRESS=%q, want unreachable address so keyringAuth cannot hit the host bus", got)
	}
	if _, ok := values["GNOME_KEYRING_CONTROL"]; ok {
		t.Fatal("GNOME_KEYRING_CONTROL must not leak into supervised AGY launches")
	}
	if !strings.HasSuffix(values["HOME"], filepath.Join("profiles", "agy", "work", "home")) && !strings.Contains(values["HOME"], filepath.Join("agy", "work", "home")) {
		t.Fatalf("HOME=%q, want the isolated profile home", values["HOME"])
	}
	if values["XDG_CONFIG_HOME"] != filepath.Join(values["HOME"], ".config") {
		t.Fatalf("XDG_CONFIG_HOME=%q, want profile-local config under HOME", values["XDG_CONFIG_HOME"])
	}
	if values["AI_HOST_DBUS_SESSION_BUS_ADDRESS"] != "unix:path=/run/user/1000/bus" {
		t.Fatalf("AI_HOST_DBUS must preserve the host bus for the browser helper, got %q", values["AI_HOST_DBUS_SESSION_BUS_ADDRESS"])
	}
}
