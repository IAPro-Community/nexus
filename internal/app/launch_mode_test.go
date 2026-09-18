package app

import (
	"os"
	"testing"
)

func TestLaunchModeResolver_DefaultsToSupervisedInTTY(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{},
		StdinIsTTY: true,
	})
	mode := r.Resolve()
	if mode != LaunchModeSupervised {
		t.Errorf("expected SUPERVISED for TTY with no flags, got %v", mode)
	}
}

func TestLaunchModeResolver_DirectFlag(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--direct"},
		StdinIsTTY: true,
	})
	mode := r.Resolve()
	if mode != LaunchModeDirect {
		t.Errorf("expected DIRECT for --direct flag, got %v", mode)
	}
}

func TestLaunchModeResolver_SupervisedFlag(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--supervised"},
		StdinIsTTY: false, // even non-TTY
	})
	mode := r.Resolve()
	if mode != LaunchModeSupervised {
		t.Errorf("expected SUPERVISED for --supervised flag, got %v", mode)
	}
}

func TestLaunchModeResolver_PrintForcesDirect(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--print"},
		StdinIsTTY: true,
	})
	mode := r.Resolve()
	if mode != LaunchModeDirect {
		t.Errorf("expected DIRECT for --print flag, got %v", mode)
	}
}

func TestLaunchModeResolver_AgyShortPrintForcesDirect(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"-p", "hello"},
		StdinIsTTY: true,
		Provider:   "agy",
	})
	if r.Resolve() != LaunchModeDirect {
		t.Fatal("agy -p must force DIRECT like --print")
	}
	codex := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"-p", "work"},
		StdinIsTTY: true,
		Provider:   "codex",
	})
	if codex.Resolve() != LaunchModeSupervised {
		t.Fatal("codex -p is native --profile and must stay SUPERVISED on TTY")
	}
}

func TestLaunchModeResolver_PipedInputForcesDirect(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{},
		StdinIsTTY: false,
	})
	mode := r.Resolve()
	if mode != LaunchModeDirect {
		t.Errorf("expected DIRECT when stdin is not TTY (piped/CI), got %v", mode)
	}
}

func TestLaunchModeResolver_EnvOverrideSupervised(t *testing.T) {
	t.Setenv("NEXUS_LAUNCH_MODE", "supervised")
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--direct"},
		StdinIsTTY: true,
	})
	mode := r.Resolve()
	if mode != LaunchModeSupervised {
		t.Errorf("expected SUPERVISED from env override, got %v", mode)
	}
}

func TestLaunchModeResolver_EnvOverrideDirect(t *testing.T) {
	t.Setenv("NEXUS_LAUNCH_MODE", "direct")
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{},
		StdinIsTTY: true,
	})
	mode := r.Resolve()
	if mode != LaunchModeDirect {
		t.Errorf("expected DIRECT from env override, got %v", mode)
	}
}

func TestLaunchModeResolver_ContinueFlag(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--continue"},
		StdinIsTTY: true,
	})
	mode := r.Resolve()
	if mode != LaunchModeSupervised {
		t.Errorf("expected SUPERVISED for --continue in TTY, got %v", mode)
	}
}

func TestLaunchModeResolver_ResumeFlag(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--resume", "sess-123"},
		StdinIsTTY: true,
	})
	mode := r.Resolve()
	if mode != LaunchModeSupervised {
		t.Errorf("expected SUPERVISED for --resume in TTY, got %v", mode)
	}
}

func TestLaunchModeResolver_PrintWithNonTTY(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--print"},
		StdinIsTTY: false,
	})
	mode := r.Resolve()
	if mode != LaunchModeDirect {
		t.Errorf("expected DIRECT for --print with non-TTY, got %v", mode)
	}
}

func TestLaunchModeResolver_FlagsInMiddleOfArgs(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--model", "gpt-4", "--direct", "--verbose"},
		StdinIsTTY: true,
	})
	mode := r.Resolve()
	if mode != LaunchModeDirect {
		t.Errorf("expected DIRECT for --direct in middle of args, got %v", mode)
	}
}

func TestLaunchModeResolver_StripsFlags(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--direct", "--model", "gpt-4"},
		StdinIsTTY: true,
	})
	result := r.StripControlFlags()
	for _, arg := range result {
		if arg == "--direct" || arg == "--supervised" {
			t.Errorf("control flag %q should be stripped, got remaining: %v", arg, result)
		}
	}
	if len(result) != 2 {
		t.Errorf("expected 2 remaining args, got %d: %v", len(result), result)
	}
}

func TestLaunchModeResolver_NonEnvStringIgnored(t *testing.T) {
	t.Setenv("NEXUS_LAUNCH_MODE", "bogus")
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{},
		StdinIsTTY: true,
	})
	mode := r.Resolve()
	if mode != LaunchModeSupervised {
		t.Errorf("expected SUPERVISED (env bogus ignored), got %v", mode)
	}
}

func TestLaunchModeResolver_ResultReason(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{},
		StdinIsTTY: true,
	})
	result := r.ResolveWithReason()
	if result.Mode != LaunchModeSupervised {
		t.Errorf("expected SUPERVISED, got %v", result.Mode)
	}
	if result.Reason == "" {
		t.Error("expected non-empty reason")
	}
	if result.Source != "tty-default" {
		t.Errorf("expected source 'tty-default', got %q", result.Source)
	}
}

func TestLaunchModeResolver_DirectFlagReason(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--direct", "--model", "x"},
		StdinIsTTY: true,
	})
	result := r.ResolveWithReason()
	if result.Mode != LaunchModeDirect {
		t.Errorf("expected DIRECT, got %v", result.Mode)
	}
	if result.Source != "explicit-direct" {
		t.Errorf("expected source 'explicit-direct', got %q", result.Source)
	}
}

func TestLaunchModeResolver_PrintFlagReason(t *testing.T) {
	r := NewLaunchModeResolver(LaunchModeInput{
		Args:       []string{"--print"},
		StdinIsTTY: true,
	})
	result := r.ResolveWithReason()
	if result.Mode != LaunchModeDirect {
		t.Errorf("expected DIRECT, got %v", result.Mode)
	}
	if result.Source != "print-flag" {
		t.Errorf("expected source 'print-flag', got %q", result.Source)
	}
}

func TestIsTerminal_Func(t *testing.T) {
	// In test context, stdin is usually not a TTY
	result := IsTerminal()
	// We can't assert a specific value since it depends on the test runner,
	// but we can assert it doesn't panic
	_ = result
}

func TestIsTerminalFd(t *testing.T) {
	// fd 1 (stdout) in CI is usually not a TTY
	result := isTerminalFD(int(os.Stdout.Fd()))
	// Same as above - just verify no panic
	_ = result
}
