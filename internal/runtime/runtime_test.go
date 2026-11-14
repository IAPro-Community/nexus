package runtime

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestWriteTerminalKeyboardModeReset(t *testing.T) {
	var out bytes.Buffer
	writeTerminalKeyboardModeReset(&out)
	if got := out.String(); got != terminalKeyboardModeReset {
		t.Fatalf("keyboard reset = %q, want %q", got, terminalKeyboardModeReset)
	}
}

func TestTerminalKeyboardModeResetDisablesBracketedPaste(t *testing.T) {
	if !strings.Contains(terminalKeyboardModeReset, "\x1b[?2004l") {
		t.Fatalf("keyboard reset does not disable bracketed paste: %q", terminalKeyboardModeReset)
	}
}

func TestTerminalKeyboardModeResetDisablesMouseTracking(t *testing.T) {
	for _, mode := range []string{"?1000l", "?1002l", "?1003l", "?1005l", "?1006l", "?1007l", "?1015l"} {
		sequence := "\x1b[" + mode
		if !strings.Contains(terminalKeyboardModeReset, sequence) {
			t.Fatalf("keyboard reset does not disable mouse mode %s: %q", mode, terminalKeyboardModeReset)
		}
	}
}

func TestTerminalKeyboardModeResetLeavesApplicationCursorMode(t *testing.T) {
	if !strings.Contains(terminalKeyboardModeReset, "\x1b[?1l") {
		t.Fatalf("keyboard reset does not disable application cursor mode: %q", terminalKeyboardModeReset)
	}
}

func TestEnvKeyForGOOSWindowsIsCaseInsensitive(t *testing.T) {
	if got := envKeyForGOOS("Path", "windows"); got != "PATH" {
		t.Fatalf("windows env key = %q, want PATH", got)
	}
	if got := envKeyForGOOS("Path", "linux"); got != "Path" {
		t.Fatalf("linux env key changed: %q", got)
	}
}

func TestEnvSetWindowsKeyCanonicalizationHelper(t *testing.T) {
	if envKeyForGOOS("CodeX_Home", "windows") != envKeyForGOOS("CODEX_HOME", "windows") {
		t.Fatal("Windows environment keys must compare case-insensitively")
	}
}

func TestNormalizeInteractiveEnvReplacesDumbTerminal(t *testing.T) {
	env := []string{"PATH=/bin", "TERM=dumb", "LANG=pt_BR.UTF-8"}
	normalized := NormalizeInteractiveEnv(env)

	if normalized[1] != "TERM=xterm-256color" {
		t.Fatalf("expected interactive terminal type, got %q", normalized[1])
	}
	if env[1] != "TERM=dumb" {
		t.Fatalf("normalization must not mutate caller environment, got %q", env[1])
	}
}

func TestNormalizeInteractiveEnvPreservesNormalTerminal(t *testing.T) {
	env := []string{"TERM=screen-256color"}
	normalized := NormalizeInteractiveEnv(env)

	if len(normalized) != 1 || normalized[0] != env[0] {
		t.Fatalf("normal terminal environment changed: %+v", normalized)
	}
}

func TestInternalBrowserHelperCanBeRemovedWhileParentIsRunning(t *testing.T) {
	data := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("AI_CLI_DATA_DIR", data)

	binDir, err := InternalBinDir()
	if err != nil {
		t.Fatalf("InternalBinDir: %v", err)
	}
	helper := BrowserHelperPath(binDir)
	if _, err := os.Stat(helper); err != nil {
		t.Fatalf("browser helper missing: %v", err)
	}
	if err := os.Remove(helper); err != nil {
		t.Fatalf("browser helper must be independently removable while Nexus is running: %v", err)
	}
}

func TestTailBufferKeepsOnlyLastMaxBytes(t *testing.T) {
	tail := &tailBuffer{max: 8}
	if _, err := tail.Write([]byte("abcdefghij")); err != nil {
		t.Fatal(err)
	}
	if got := tail.String(); got != "cdefghij" {
		t.Fatalf("tail = %q, want %q", got, "cdefghij")
	}
}

func TestClassifyExitFailurePrefersStderrAndFallsBackToStdout(t *testing.T) {
	exitErr := errors.New("exit status 1")
	agyQuota := "⚠ Individual quota reached. Please upgrade your subscription to increase your limits. Resets in 167h58m45s."

	cases := []struct {
		name      string
		stderr    string
		stdout    string
		want      model.FailureKind
		wantReset bool
	}{
		{
			name:   "stderr quota classifies without consulting stdout",
			stderr: agyQuota,
			stdout: "",
			want:   model.FailureQuota,
		},
		{
			name:   "stderr auth wins over stdout quota wording",
			stderr: "Please run /login to authenticate with your Google account.",
			stdout: agyQuota,
			want:   model.FailureAuth,
		},
		{
			name:      "stdout-only quota falls back and carries reset",
			stderr:    "",
			stdout:    agyQuota,
			want:      model.FailureQuota,
			wantReset: true,
		},
		{
			name:   "unrelated content stays unknown",
			stderr: "",
			stdout: "the agent discussed quota policy in the report",
			want:   model.FailureUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyExitFailure(exitErr, tc.stderr, tc.stdout)
			if got.Kind != tc.want {
				t.Fatalf("kind = %s, want %s", got.Kind, tc.want)
			}
			if tc.wantReset {
				if got.ResetAt == nil {
					t.Fatal("expected ResetAt for parsed reset window")
				}
				if delta := time.Until(*got.ResetAt); delta <= 0 || delta > 168*time.Hour {
					t.Fatalf("expected weekly reset, got %v", delta)
				}
			}
		})
	}
}
