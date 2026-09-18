package main

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFindBootstrapRequiresTokenAndRedactsNothingInReturnedValue(t *testing.T) {
	bootstrap := findBootstrap("URL: http://127.0.0.1:13000\nBootstrap: http://127.0.0.1:13000/?token=abc123\n")
	if bootstrap == "" || !strings.Contains(bootstrap, "token=abc123") {
		t.Fatalf("bootstrap URL was not detected: %q", bootstrap)
	}
	if got := findBootstrap("Bootstrap: http://127.0.0.1:13000/\n"); got != "" {
		t.Fatalf("accepted tokenless bootstrap URL: %q", got)
	}
}

func TestParseBootstrapURLAcceptsPersistedFragmentToken(t *testing.T) {
	got := parseBootstrapURL("http://127.0.0.1:13000/#nexus_bootstrap=abc123")
	if got == "" || !strings.Contains(got, "#nexus_bootstrap=abc123") {
		t.Fatalf("persisted bootstrap URL was not accepted: %q", got)
	}
	if got := parseBootstrapURL("http://127.0.0.1:13000/"); got != "" {
		t.Fatalf("accepted tokenless URL: %q", got)
	}
}

func TestBootstrapTokenFromURLExchangesFragmentToken(t *testing.T) {
	parsed, err := url.Parse("http://127.0.0.1:13000/#nexus_bootstrap=abc123")
	if err != nil {
		t.Fatal(err)
	}
	got, err := bootstrapTokenFromURL(parsed)
	if err != nil || got != "abc123" {
		t.Fatalf("fragment token = %q, err=%v", got, err)
	}

	parsed, err = url.Parse("http://127.0.0.1:13000/?token=query123")
	if err != nil {
		t.Fatal(err)
	}
	got, err = bootstrapTokenFromURL(parsed)
	if err != nil || got != "query123" {
		t.Fatalf("query token = %q, err=%v", got, err)
	}

	parsed, err = url.Parse("http://127.0.0.1:13000/")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bootstrapTokenFromURL(parsed); err == nil {
		t.Fatal("tokenless bootstrap URL was accepted")
	}
}

func TestCopyProfileTreeSkipsSymlinksAndCycles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires platform privileges on Windows")
	}
	source := t.TempDir()
	destination := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "auth.json"), []byte("redacted-test-fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(".", filepath.Join(source, "self")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(source, "auth.json"), filepath.Join(source, "auth-link.json")); err != nil {
		t.Fatal(err)
	}

	if err := copyProfileTree(source, destination); err != nil {
		t.Fatalf("copyProfileTree followed a symlink: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "auth.json")); err != nil {
		t.Fatalf("regular profile file was not copied: %v", err)
	}
	for _, name := range []string{"self", "auth-link.json"} {
		if _, err := os.Lstat(filepath.Join(destination, name)); !os.IsNotExist(err) {
			t.Fatalf("symlink %q was copied into isolated profile", name)
		}
	}
}

func TestNotAuthenticatedIsMachineClassifiable(t *testing.T) {
	err := errors.Join(errNotAuthenticated, errors.New("provider unavailable"))
	if !errors.Is(err, errNotAuthenticated) {
		t.Fatal("authentication limitation is not classifiable")
	}
}

func TestResolveE2ERootUsesExplicitNonTemporaryDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil && runtime.GOOS != "windows" {
		t.Fatal(err)
	}
	t.Setenv("NEXUS_E2E_ROOT", root)

	got, remove, err := resolveE2ERoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != root || remove {
		t.Fatalf("explicit root = %q, remove=%t; want %q, false", got, remove, root)
	}
}

func TestResolveE2ERootRejectsNonEmptyExplicitDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sentinel"), []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEXUS_E2E_ROOT", root)

	if _, _, err := resolveE2ERoot(); err == nil {
		t.Fatal("accepted a non-empty explicit E2E root")
	}
}

func TestResolveE2ERootRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires platform privileges on Windows")
	}
	target := t.TempDir()
	parent := t.TempDir()
	root := filepath.Join(parent, "root-link")
	if err := os.Symlink(target, root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEXUS_E2E_ROOT", root)

	if _, _, err := resolveE2ERoot(); err == nil {
		t.Fatal("accepted a symlink as explicit E2E root")
	}
}

func TestResolveE2ERootRejectsOpenPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not enforced on Windows")
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NEXUS_E2E_ROOT", root)

	if _, _, err := resolveE2ERoot(); err == nil {
		t.Fatal("accepted an explicit E2E root with open permissions")
	}
}

func TestResolveE2ERootCreatesDisposableTemporaryDirectory(t *testing.T) {
	t.Setenv("NEXUS_E2E_ROOT", "")

	root, remove, err := resolveE2ERoot()
	if err != nil {
		t.Fatal(err)
	}
	if !remove {
		t.Fatal("temporary E2E root was not marked disposable")
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
}

func TestResolveE2EProfileSourceUsesHostHomeNotProviderHome(t *testing.T) {
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", filepath.Join(hostHome, "profiles", "codex", "isolated", "home"))
	t.Setenv("NEXUS_E2E_PROFILE_SOURCE", "")

	got := resolveE2EProfileSource()
	want := filepath.Join(hostHome, ".local", "share", "ai-manager")
	if got != want {
		t.Fatalf("resolveE2EProfileSource=%q, want %q", got, want)
	}
}

func TestResolveE2EProfileSourceHonorsExplicitOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "profiles-source")
	t.Setenv("AI_REAL_HOME", t.TempDir())
	t.Setenv("NEXUS_E2E_PROFILE_SOURCE", override)

	if got := resolveE2EProfileSource(); got != override {
		t.Fatalf("resolveE2EProfileSource=%q, want explicit override %q", got, override)
	}
}

func TestProviderMarkerRequiresLiteralMarkerAndSanitizesANSI(t *testing.T) {
	if providerMarkerSeen("banner only\nNEXUS_E2E_O") {
		t.Fatal("accepted partial provider marker")
	}
	if !providerMarkerSeen("banner\n\x1b[32mNEXUS_E2E_OK\x1b[0m\n") {
		t.Fatal("did not accept literal marker with ANSI repaint")
	}
	if providerMarkerSeen("Reply with exactly NEXUS_E2E_OK and nothing else.") {
		t.Fatal("accepted marker echoed inside the prompt")
	}
	if providerMarkerSeen("NEXUS_E2E_NOT_OK") {
		t.Fatal("accepted non-marker output")
	}
	if strings.Contains(sanitizeTranscript("TOKEN=secret-value"), "secret-value") {
		t.Fatal("transcript secret was not redacted")
	}
}
