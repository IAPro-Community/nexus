package driver

import (
	"reflect"
	"testing"
)

func TestApplyLaunchConfigurationModelFlags(t *testing.T) {
	tests := []struct {
		provider string
		model    string
		want     []string
	}{
		{"codex", "gpt-5.6-sol", []string{"--model", "gpt-5.6-sol"}},
		{"claude", "opus", []string{"--model", "opus"}},
		{"gemini", "gemini-2.5-pro", []string{"--model", "gemini-2.5-pro"}},
		{"opencode", "ollama/deepseek-r1", []string{"--model", "ollama/deepseek-r1"}},
		{"agy", "claude-sonnet-4-20250514", []string{"--model", "claude-sonnet-4-20250514"}},
	}
	for _, tt := range tests {
		got, err := ApplyLaunchConfiguration(tt.provider, tt.model, nil, nil)
		if err != nil {
			t.Fatalf("%s: %v", tt.provider, err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("%s args=%v want %v", tt.provider, got, tt.want)
		}
	}
}

func TestApplyLaunchConfigurationRejectsUnsupportedModel(t *testing.T) {
	if _, err := ApplyLaunchConfiguration("cursor", "some-model", nil, nil); err == nil {
		t.Fatal("Cursor model override must not be silently ignored")
	}
}

func TestApplyLaunchConfigurationExplicitExtraArgs(t *testing.T) {
	got, err := ApplyLaunchConfiguration("codex", "", map[string]any{"extra_args": []any{"--sandbox", "workspace-write"}}, []string{"resume", "sess"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"resume", "sess", "--sandbox", "workspace-write"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args=%v want %v", got, want)
	}
}

func TestApplyLaunchConfigurationIgnoresUIModeOption(t *testing.T) {
	got, err := ApplyLaunchConfiguration("agy", "", map[string]any{"mode": "YOLO", "extra_args": []any{"--prompt", "hi"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--prompt", "hi"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args=%v want %v", got, want)
	}
}

func TestApplyLaunchConfigurationRejectsUnknownOption(t *testing.T) {
	if _, err := ApplyLaunchConfiguration("codex", "", map[string]any{"temperature": 0.7}, nil); err == nil {
		t.Fatal("unknown provider options must not be silently ignored")
	}
}

func TestApplyLaunchConfigurationCanonicalAlias(t *testing.T) {
	gotAGY, err := ApplyLaunchConfiguration("agy", "", nil, []string{"--yolo"})
	if err != nil {
		t.Fatal(err)
	}
	wantAGY := []string{"--dangerously-skip-permissions"}
	if !reflect.DeepEqual(gotAGY, wantAGY) {
		t.Fatalf("agy args=%v want %v", gotAGY, wantAGY)
	}

	gotCodex, err := ApplyLaunchConfiguration("codex", "", nil, []string{"--yolo"})
	if err != nil {
		t.Fatal(err)
	}
	wantCodex := []string{"--dangerously-bypass-approvals-and-sandbox"}
	if !reflect.DeepEqual(gotCodex, wantCodex) {
		t.Fatalf("codex args=%v want %v", gotCodex, wantCodex)
	}
}

func TestApplyLaunchConfigurationNormalizesConfiguredYoloForCodex(t *testing.T) {
	got, err := ApplyLaunchConfiguration("codex", "", map[string]any{
		"mode":       "YOLO",
		"extra_args": []any{"--yolo", "--sandbox", "workspace-write", "--add-dir", "one", "--add-dir", "two"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--dangerously-bypass-approvals-and-sandbox",
		"--sandbox", "workspace-write",
		"--add-dir", "one", "--add-dir", "two",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("configured args=%v want %v", got, want)
	}
}
