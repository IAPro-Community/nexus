package app

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	"github.com/IAPro-Community/nexus/internal/profile"
	"github.com/IAPro-Community/nexus/internal/testutil"
)

func captureStdout(f func() error) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	var buf bytes.Buffer
	copyDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(&buf, r)
		copyDone <- copyErr
	}()

	runErr := f()
	closeErr := w.Close()
	copyErr := <-copyDone
	_ = r.Close()

	if runErr != nil {
		return buf.String(), runErr
	}
	if closeErr != nil {
		return buf.String(), closeErr
	}
	return buf.String(), copyErr
}

func TestUpdateCommandFailsClosedWhenApplyFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "registry unavailable", http.StatusInternalServerError)
	}))
	defer ts.Close()

	t.Setenv("NEXUS_UPDATE_REGISTRY_URL", ts.URL+"/update-manifest.json")
	t.Setenv("NEXUS_INSTALL_METHOD", "STANDALONE")

	_, err := captureStdout(func() error { return updateCmd(nil) })
	if err == nil || !strings.Contains(err.Error(), "update could not be applied") {
		t.Fatalf("update command must return a non-zero error when apply fails, got %v", err)
	}
}

func TestCaptureStdoutDrainsLargeOutput(t *testing.T) {
	const size = 256 * 1024
	expected := strings.Repeat("nexus-output\n", size/len("nexus-output\n"))

	out, err := captureStdout(func() error {
		_, writeErr := os.Stdout.WriteString(expected)
		return writeErr
	})
	if err != nil {
		t.Fatalf("captureStdout returned error: %v", err)
	}
	if out != expected {
		t.Fatalf("captured output mismatch: got %d bytes, want %d", len(out), len(expected))
	}
}

func TestTopLevelCommandDispatchContracts(t *testing.T) {
	setupTestEnvironment(t)
	cases := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{name: "help", args: []string{"help"}},
		{name: "version", args: []string{"version"}},
		{name: "provider help", args: []string{"help", "codex"}},
		{name: "unknown command", args: []string{"not-a-command"}, wantError: true},
		{name: "missing plan id", args: []string{"plan", "show"}, wantError: true},
		{name: "missing compile arguments", args: []string{"plan", "compile"}, wantError: true},
		{name: "missing run arguments", args: []string{"plan", "run"}, wantError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := captureStdout(func() error { return Run(tc.args) })
			if (err != nil) != tc.wantError {
				t.Fatalf("args=%v error=%v wantError=%v", tc.args, err, tc.wantError)
			}
		})
	}
}

func TestHelpDocumentsDispatchedCommandFamilies(t *testing.T) {
	setupTestEnvironment(t)
	out, err := captureStdout(func() error { return Run([]string{"help"}) })
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{
		"web", "projects", "agents", "plan", "providers", "profiles", "start",
		"stop", "attach", "handoff", "continue", "usage", "doctor", "security",
		"sessions", "workspaces", "paths", "rename", "current", "run", "export",
		"issue-report", "update", "maestro", "completion", "version",
	} {
		if !strings.Contains(out, " "+command) {
			t.Errorf("help does not document dispatched command %q", command)
		}
	}
}

func TestAgentsCommandUsesSinglePositionalProjectArgument(t *testing.T) {
	setupTestEnvironment(t)
	n := nexus.Default()
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	first, err := st.CreateProject(store.Project{Name: "First", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	second, err := st.CreateProject(store.Project{Name: "Second", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateAgent(store.Agent{ProjectID: first.ID, Name: "First agent"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateAgent(store.Agent{ProjectID: second.ID, Name: "Second agent"}); err != nil {
		t.Fatal(err)
	}
	if err := st.TouchProject(first.ID); err != nil {
		t.Fatal(err)
	}
	out, err := captureStdout(func() error { return Run([]string{"agents", second.ID}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "("+second.ID+")") || strings.Contains(out, "("+first.ID+")") {
		t.Fatalf("agents did not honor positional project id: %s", out)
	}
}

func setupTestEnvironment(t *testing.T) (binDir, testOut string) {
	t.Helper()
	nexus.ResetDefaultForTest()
	data := t.TempDir()
	cfg := t.TempDir()
	state := t.TempDir()
	binDir = t.TempDir()
	testOut = filepath.Join(t.TempDir(), "test_out.txt")

	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)
	t.Setenv("AI_CLI_STATE_DIR", state)
	t.Setenv("AI_TEST_OUT", testOut)

	writeExe := func(name, body string) {
		testutil.WriteFakeBinaryWithScript(t, binDir, name, body)
	}

	writeExe("codex", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=codex" > "$AI_TEST_OUT"
  echo "home=$CODEX_HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("agy", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=agy" > "$AI_TEST_OUT"
  echo "home=$HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("claude", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=claude" > "$AI_TEST_OUT"
  echo "home=$HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("opencode", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=opencode" > "$AI_TEST_OUT"
  echo "home=$HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("gemini", `if [ -n "$AI_TEST_OUT" ]; then
  echo "provider=gemini" > "$AI_TEST_OUT"
  echo "home=$HOME" >> "$AI_TEST_OUT"
  echo "cwd=$PWD" >> "$AI_TEST_OUT"
  for arg in "$@"; do
    echo "arg=$arg" >> "$AI_TEST_OUT"
  done
fi`)

	writeExe("dbus-run-session", `if [ "$1" = "--" ]; then shift; fi
exec "$@"`)
	writeExe("gnome-keyring-daemon", `cat >/dev/null
exit 0`)

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	// Cleanup callbacks run in LIFO order. Register singleton shutdown after
	// TempDir and environment cleanups so SQLite closes before Windows tries to
	// remove nexus.db and while the configured data directory is still active.
	t.Cleanup(nexus.ResetDefaultForTest)
	return binDir, testOut
}

func TestControlPlaneCLICommands(t *testing.T) {
	_, testOut := setupTestEnvironment(t)

	// 1. Version
	out, err := captureStdout(func() error {
		return Run([]string{"version"})
	})
	if err != nil || !strings.Contains(out, "Nexus") {
		t.Fatalf("version failed: %s, %v", out, err)
	}

	// 2. Help
	out, err = captureStdout(func() error {
		return Run([]string{"help"})
	})
	if err != nil || !strings.Contains(out, "Usage:") {
		t.Fatalf("help failed: %s, %v", out, err)
	}

	// 3. Providers listing
	out, err = captureStdout(func() error {
		return Run([]string{"providers"})
	})
	if err != nil || !strings.Contains(out, "Codex") || !strings.Contains(out, "Claude Code") {
		t.Fatalf("providers failed: %s, %v", out, err)
	}

	// 4. Add profiles
	if err := Run([]string{"add", "codex", "work", "--no-login"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"add", "codex", "personal", "--no-login"}); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"add", "agy", "google-1", "--no-login"}); err != nil {
		t.Fatal(err)
	}

	// 5. Profiles listing
	out, err = captureStdout(func() error {
		return Run([]string{"profiles"})
	})
	if err != nil || !strings.Contains(out, "work") || !strings.Contains(out, "google-1") {
		t.Fatalf("profiles failed: %s, %v", out, err)
	}

	// 6. JSON output support
	out, err = captureStdout(func() error {
		return Run([]string{"providers", "--json"})
	})
	if err != nil || !strings.Contains(out, `"installed": true`) {
		t.Fatalf("providers --json failed: %s, %v", out, err)
	}

	// 7. Inspect
	out, err = captureStdout(func() error {
		return Run([]string{"inspect", "codex:work"})
	})
	if err != nil || !strings.Contains(out, `"provider": "codex"`) {
		t.Fatalf("inspect failed: %s, %v", out, err)
	}

	// 8. Bindings
	if err := Run([]string{"bind", "codex:work"}); err != nil {
		t.Fatal(err)
	}
	out, err = captureStdout(func() error {
		return Run([]string{"bindings"})
	})
	if err != nil || !strings.Contains(out, "work") {
		t.Fatalf("bindings failed: %s, %v", out, err)
	}
	if err := Run([]string{"unbind", "codex"}); err != nil {
		t.Fatal(err)
	}

	// 9. Explain
	out, err = captureStdout(func() error {
		return Run([]string{"explain", "codex"})
	})
	if err != nil || !strings.Contains(out, "Smart Account Selection") {
		t.Fatalf("explain failed: %s, %v", out, err)
	}

	// 10. Run explicit profile
	_ = os.Remove(testOut)
	if err := Run([]string{"codex:personal", "--model", "gpt-5.6-sol", "-p", "hello"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(testOut)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	home, _ := profile.Home("codex", "personal")
	if !strings.Contains(got, "home="+home) || !strings.Contains(got, "arg=--model") {
		t.Fatalf("run explicit profile failed:\n%s", got)
	}

	// 11. Run resume with provider native syntax
	_ = os.Remove(testOut)
	if err := Run([]string{"resume", "session-uuid-1234", "codex:work"}); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(testOut)
	got = string(raw)
	if !strings.Contains(got, "arg=resume") || !strings.Contains(got, "arg=session-uuid-1234") {
		t.Fatalf("resume codex failed (must use 'codex resume <id>'):\n%s", got)
	}

	// 12. Security command
	out, err = captureStdout(func() error {
		return Run([]string{"security"})
	})
	if err != nil || !strings.Contains(out, "Isolation Preset:") {
		t.Fatalf("security failed: %s, %v", out, err)
	}

	// 13. Doctor command
	out, err = captureStdout(func() error {
		return Run([]string{"doctor"})
	})
	if err != nil || !strings.Contains(out, "Nexus Diagnostics") {
		t.Fatalf("doctor failed: %s, %v", out, err)
	}

	// 14. Completion commands (bash, zsh, fish, powershell, pwsh)
	for _, sh := range []string{"bash", "zsh", "fish", "powershell", "pwsh"} {
		out, err = captureStdout(func() error {
			return Run([]string{"completion", sh})
		})
		if err != nil || len(out) == 0 {
			t.Fatalf("completion %s failed: %s, %v", sh, out, err)
		}
	}

	// 15. AI Control CLI commands
	out, err = captureStdout(func() error {
		return Run([]string{"control", "running", "--json"})
	})
	if err != nil || !strings.Contains(out, "[") {
		t.Fatalf("control running failed: %s, %v", out, err)
	}

	out, err = captureStdout(func() error {
		return Run([]string{"control", "cleanup"})
	})
	if err != nil || !strings.Contains(out, "Cleaned up") {
		t.Fatalf("control cleanup failed: %s, %v", out, err)
	}

	out, err = captureStdout(func() error {
		return Run([]string{"control", "doctor", "--json"})
	})
	if err != nil || !strings.Contains(out, "drivers") {
		t.Fatalf("control doctor failed: %s, %v", out, err)
	}
}

func TestQuotaGroupStatusKeepsUsableModelGroupAvailable(t *testing.T) {
	qv := quota.QuotaView{Status: "CACHED", ModelGroups: []quota.ModelGroup{
		{Name: "Gemini Models", Windows: []quota.Window{{Kind: "5h", Remaining: 8}, {Kind: "weekly", Remaining: 17}}},
		{Name: "Claude & GPT Models", Windows: []quota.Window{{Kind: "claude_5h", Remaining: 0}, {Kind: "claude_weekly", Remaining: 0}}},
	}}
	qv.ComputeAvailability()
	if !qv.IsAvailable() || quotaGroupStatus(qv.ModelGroups[0], qv.Status) != "COM QUOTA" || quotaGroupStatus(qv.ModelGroups[1], qv.Status) != "SEM QUOTA" {
		t.Fatalf("expected mixed group availability, got profile=%v reasons=%+v", qv.IsAvailable(), qv.AvailReasons)
	}
}

func TestPerformSystemUpdateDoesNotClaimNexusBinaryUpdated(t *testing.T) {
	binDir := t.TempDir()
	writeExecutable := func(name, body string) {
		t.Helper()
		testutil.WriteFakeBinaryWithScript(t, binDir, name, body)
	}
	writeExecutable("npm", "exit 0")
	writeExecutable("orquestrador-maestro", "if [ \"$1\" = \"version\" ]; then echo 0.1.0; fi")
	t.Setenv("PATH", binDir)
	t.Setenv("HOME", t.TempDir())

	result := PerformSystemUpdate()
	if result.NexusUpdated {
		t.Fatal("update must not claim that the Nexus binary was updated when no binary update ran")
	}
	if !strings.Contains(result.Error, "Nexus binary update") {
		t.Fatalf("missing honest Nexus update status: %+v", result)
	}
}

func TestMaestroStatusIsExplicitAndDegradedWithoutMaestro(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("NEXUS_ORQUESTRADOR_DIR", filepath.Join(t.TempDir(), "missing-maestro"))
	out, err := captureStdout(func() error {
		return Run([]string{"maestro", "status"})
	})
	if err != nil {
		t.Fatalf("maestro status failed: %v", err)
	}
	if !strings.Contains(out, "Maestro: unavailable") || !strings.Contains(out, "degraded mode") {
		t.Fatalf("expected honest degraded Maestro status, got %q", out)
	}
}

func TestQuotaUnknownLabelReturnsStatusAwarePlaceholder(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{string(model.UsageUnknown), "— desconhecido"},
		{string(model.UsageError), "— erro"},
		{string(model.UsageRateLimited), "— rate limited"},
		{"CACHED", "—"},
		{"", "—"},
	}
	for _, tt := range tests {
		if got := quotaUnknownLabel(tt.status); got != tt.want {
			t.Errorf("quotaUnknownLabel(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}
