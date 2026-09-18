package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCommandWindowsPrefersPathAndBuildsCmdLauncher(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "codex.cmd")
	if err := os.WriteFile(artifact, []byte("@echo off\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{
		"PATH":         dir,
		"PATHEXT":      ".EXE;.CMD;.BAT;.PS1",
		"COMSPEC":      `C:\\Windows\\System32\\cmd.exe`,
		"APPDATA":      filepath.Join(dir, "appdata"),
		"LOCALAPPDATA": filepath.Join(dir, "local"),
		"USERPROFILE":  filepath.Join(dir, "profile"),
	}
	getenv := func(key string) string { return env[key] }
	stat := func(path string) (os.FileInfo, error) { return os.Stat(path) }
	lookPath := func(path string) (string, error) { return path, errors.New("not used") }

	resolved, err := resolveCommand("codex", "windows", getenv, stat, lookPath)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ArtifactPath != artifact || resolved.Kind != CommandCmd {
		t.Fatalf("unexpected resolution: %+v", resolved)
	}
	if resolved.LauncherPath != env["COMSPEC"] || len(resolved.PrefixArgs) != 4 {
		t.Fatalf("unexpected launcher: %+v", resolved)
	}
}

func TestResolveCommandWindowsSupportsPowerShellShim(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "claude.ps1")
	if err := os.WriteFile(artifact, []byte("Write-Output ok\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"PATH": dir, "PATHEXT": ".PS1", "USERPROFILE": dir}
	getenv := func(key string) string { return env[key] }
	stat := func(path string) (os.FileInfo, error) { return os.Stat(path) }
	lookPath := func(path string) (string, error) { return path, errors.New("launcher unavailable in test") }

	resolved, err := resolveCommand("claude", "windows", getenv, stat, lookPath)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Kind != CommandPowerShell || len(resolved.PrefixArgs) != 2 || resolved.PrefixArgs[0] != "-File" {
		t.Fatalf("unexpected PowerShell resolution: %+v", resolved)
	}
}

func TestResolveCommandUnixUsesLookPath(t *testing.T) {
	lookPath := func(path string) (string, error) { return "/usr/local/bin/" + path, nil }
	resolved, err := resolveCommand("codex", "linux", func(string) string { return "" }, nil, lookPath)
	if err != nil || resolved.LauncherPath != "/usr/local/bin/codex" || resolved.Kind != CommandExecutable {
		t.Fatalf("unexpected Unix resolution: %+v, %v", resolved, err)
	}
}

func TestPathListSeparatorForGOOS(t *testing.T) {
	if got := pathListSeparatorForGOOS("windows"); got != ";" {
		t.Fatalf("windows separator = %q, want %q", got, ";")
	}
	for _, goos := range []string{"linux", "darwin", "freebsd", ""} {
		if got := pathListSeparatorForGOOS(goos); got != ":" {
			t.Fatalf("separator for %q = %q, want %q", goos, got, ":")
		}
	}
}

func TestResolveCommandWindowsSearchesEveryPathEntry(t *testing.T) {
	empty := t.TempDir()
	dir := t.TempDir()
	artifact := filepath.Join(dir, "codex.exe")
	if err := os.WriteFile(artifact, []byte("MZ"), 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{
		"PATH":         empty + ";" + dir,
		"PATHEXT":      ".COM;.EXE;.BAT;.CMD;.PS1",
		"APPDATA":      filepath.Join(dir, "appdata"),
		"LOCALAPPDATA": filepath.Join(dir, "local"),
		"USERPROFILE":  filepath.Join(dir, "profile"),
	}
	getenv := func(key string) string { return env[key] }
	stat := func(path string) (os.FileInfo, error) { return os.Stat(path) }
	lookPath := func(path string) (string, error) { return path, errors.New("not used") }

	resolved, err := resolveCommand("codex", "windows", getenv, stat, lookPath)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ArtifactPath != artifact || resolved.Kind != CommandExecutable {
		t.Fatalf("binary in second PATH entry not found: %+v", resolved)
	}
}

func TestResolveCommandWindowsEmptyPathIsNotFound(t *testing.T) {
	for _, pathValue := range []string{"", "   "} {
		env := map[string]string{
			"PATH":         pathValue,
			"PATHEXT":      ".EXE",
			"APPDATA":      "",
			"LOCALAPPDATA": "",
			"USERPROFILE":  "",
		}
		getenv := func(key string) string { return env[key] }
		stat := func(path string) (os.FileInfo, error) { return os.Stat(path) }
		lookPath := func(path string) (string, error) { return path, errors.New("not used") }

		if _, err := resolveCommand("codex", "windows", getenv, stat, lookPath); err == nil {
			t.Fatalf("empty PATH %q must fail, got nil error", pathValue)
		}
	}
}
