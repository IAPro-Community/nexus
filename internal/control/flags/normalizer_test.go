package flags

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeYolo(t *testing.T) {
	tests := []struct {
		provider string
		args     []string
		expected []string
	}{
		{
			provider: "agy",
			args:     []string{"--yolo"},
			expected: []string{"--dangerously-skip-permissions"},
		},
		{
			provider: "claude",
			args:     []string{"-y", "--prompt", "hello"},
			expected: []string{"--dangerously-skip-permissions", "--prompt", "hello"},
		},
		{
			provider: "codex",
			args:     []string{"--yolo"},
			expected: []string{"--dangerously-bypass-approvals-and-sandbox"},
		},
		{
			provider: "agy",
			args:     []string{"--dangerously-skip-permissions", "--yolo"},
			expected: []string{"--dangerously-skip-permissions"},
		},
		{
			provider: "agy",
			args:     []string{"--mode", "plan"},
			expected: []string{"--mode", "plan"},
		},
		{
			provider: "agy",
			args:     []string{"--plan"},
			expected: []string{"--mode", "plan"},
		},
		{
			provider: "agy",
			args:     []string{"--continue"},
			expected: []string{"--continue"},
		},
		{
			provider: "codex",
			args:     []string{"-c"},
			expected: []string{"-c"},
		},
		{
			provider: "codex",
			args:     []string{"--continue"},
			expected: []string{"resume", "--last"},
		},
		{
			provider: "codex",
			args:     []string{"-c", `model_reasoning_effort="high"`},
			expected: []string{"-c", `model_reasoning_effort="high"`},
		},
		{
			provider: "codex",
			args:     []string{"-p", "work"},
			expected: []string{"-p", "work"},
		},
		{
			provider: "agy",
			args:     []string{"--resume", "sess-123"},
			expected: []string{"--conversation=sess-123"},
		},
		{
			provider: "codex",
			args:     []string{"-r", "sess-123"},
			expected: []string{"resume", "sess-123"},
		},
		{
			provider: "agy",
			args:     []string{"--print"},
			expected: []string{"--print"},
		},
		{
			provider: "codex",
			args:     []string{"--print"},
			expected: []string{"exec"},
		},
		{
			provider: "codex",
			args:     []string{"-p"},
			expected: []string{"-p"},
		},
		{
			provider: "agy",
			args:     []string{"--effort", "low"},
			expected: []string{"--effort", "low"},
		},
		{
			provider: "codex",
			args:     []string{"--effort", "high", "-c", "foo.toml"},
			expected: []string{"-c", `model_reasoning_effort="high"`, "-c", "foo.toml"},
		},
	}

	for _, tt := range tests {
		got := Normalize(tt.provider, tt.args, nil)
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("Normalize(%q, %v) = %v; want %v", tt.provider, tt.args, got, tt.expected)
		}
	}
}

func TestNormalizeCustomUserAliases(t *testing.T) {
	userAliases := map[string]map[string][]string{
		"--fast": {
			"agy":   []string{"--effort", "low"},
			"codex": []string{"-c", "model_reasoning_effort=low"},
		},
	}

	gotAGY := Normalize("agy", []string{"--fast"}, userAliases)
	wantAGY := []string{"--effort", "low"}
	if !reflect.DeepEqual(gotAGY, wantAGY) {
		t.Errorf("got %v, want %v", gotAGY, wantAGY)
	}

	gotCodex := Normalize("codex", []string{"--fast"}, userAliases)
	wantCodex := []string{"-c", "model_reasoning_effort=low"}
	if !reflect.DeepEqual(gotCodex, wantCodex) {
		t.Errorf("got %v, want %v", gotCodex, wantCodex)
	}
}

func TestRenderMergedHelp(t *testing.T) {
	nativeHelp := "Usage: agy [options]\n  --help show help"
	merged := RenderMergedHelp("agy", nativeHelp)
	if !strings.Contains(merged, "Nexus · Canonical Aliases for AGY") {
		t.Errorf("expected header with provider, got %s", merged)
	}
	if !strings.Contains(merged, "--yolo") {
		t.Errorf("expected --yolo in table, got %s", merged)
	}
	if !strings.Contains(merged, "--dangerously-skip-permissions") {
		t.Errorf("expected translation in table, got %s", merged)
	}
	if !strings.Contains(merged, "Official AGY CLI Help:") {
		t.Errorf("expected official help section, got %s", merged)
	}
	if !strings.Contains(merged, nativeHelp) {
		t.Errorf("expected native help text, got %s", merged)
	}
}
