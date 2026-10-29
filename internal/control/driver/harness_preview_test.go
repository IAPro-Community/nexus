package driver

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestPreviewHarnessExplicitCapabilities(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		id      string
		resume  []string
		kickoff []string
	}{
		{"pi", []string{"--session", "session-1"}, []string{"-p", "hello"}},
		{"omp", []string{"--resume", "session-1"}, []string{"-p", "hello"}},
		{"mimo", []string{"--session", "session-1"}, []string{"run", "hello"}},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			d := NewPreviewHarnessDriver(tc.id)
			caps := d.EffectiveCaps(ctx, model.Profile{Name: "default"})
			if caps.StructuredEvents.Status == CapabilitySupported || caps.Approvals.Status == CapabilitySupported {
				t.Fatal("preview must not advertise wired structured events or approvals")
			}
			if can, _ := d.CanResume(ctx, model.Profile{}, ""); can {
				t.Fatal("empty resume id")
			}
			got, err := d.BuildResumeArgs(ctx, model.Profile{}, "session-1")
			if err != nil || !reflect.DeepEqual(got, tc.resume) {
				t.Fatalf("resume got %v: %v", got, err)
			}
			got, err = d.BuildKickoffArgs(ctx, model.Profile{}, "hello")
			if err != nil || !reflect.DeepEqual(got, tc.kickoff) {
				t.Fatalf("kickoff got %v: %v", got, err)
			}
		})
	}
	d := NewPreviewHarnessDriver("dsh")
	if d.EffectiveCaps(ctx, model.Profile{}).Terminal.Status != CapabilityUnsupported {
		t.Fatal("DSH ACP-only preview must not pretend to have validated TUI")
	}
	if _, _, _, err := d.BuildCommand(ctx, model.Profile{}, nil); err == nil {
		t.Fatal("unsupported DSH TUI launch")
	}
}

func TestOMPPreviewUsesOwnIsolatedAgentDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(home, "foreign"))
	t.Setenv("OMP_PROFILE", "inherited-host")
	t.Setenv("PI_PROFILE", "legacy-host")
	t.Setenv("PI_CODING_AGENT_SESSION_DIR", filepath.Join(home, "sessions-external"))
	t.Setenv("OMP_WORKTREE_DIR", filepath.Join(home, "worktree-external"))
	t.Setenv("PI_CONFIG_FILES", filepath.Join(home, "settings-external.yml"))
	env := previewHarnessEnv("omp", home, "nexus-default")
	values := map[string]string{}
	for _, e := range env {
		k, v, found := strings.Cut(e, "=")
		if found {
			values[k] = v
		}
	}
	if got, want := values["PI_CODING_AGENT_DIR"], filepath.Join(home, ".omp", "agent"); got != want {
		t.Fatalf("OMP native agent dir = %q, want %q", got, want)
	}
	if got := values["PI_CONFIG_DIR"]; got != ".omp" {
		t.Fatalf("OMP config root = %q", got)
	}
	if got := values["OMP_PROFILE"]; got != "" {
		t.Fatalf("OMP must ignore inherited profile %q", got)
	}
	for _, forbidden := range []string{"PI_PROFILE", "PI_CODING_AGENT_SESSION_DIR", "PI_CONFIG_FILES", "OMP_WORKTREE_DIR"} {
		if _, ok := values[forbidden]; ok {
			t.Errorf("OMP inherited unsafe %s", forbidden)
		}
	}
	if values["HOME"] != home || values["AI_PROFILE"] != "nexus-default" {
		t.Fatalf("OMP profile identity not isolated: HOME=%q AI_PROFILE=%q", values["HOME"], values["AI_PROFILE"])
	}
	piEnv := previewHarnessEnv("pi", home, "default")
	var piAgentDir string
	for _, e := range piEnv {
		if strings.HasPrefix(e, "PI_CODING_AGENT_DIR=") {
			piAgentDir = strings.TrimPrefix(e, "PI_CODING_AGENT_DIR=")
		}
	}
	if piAgentDir != filepath.Join(home, ".pi", "agent") {
		t.Fatalf("Pi agent root regressed: %q", piAgentDir)
	}
}

func TestOMPPreviewDoesNotPipeStructuredProtocolIntoPTY(t *testing.T) {
	for _, args := range [][]string{
		{"acp"}, {"--mode", "rpc"}, {"--mode=rpc-ui"}, {"--mode", "acp"}, {"--mode=rpc"},
	} {
		if !isStructuredOMPArgs(args) {
			t.Errorf("structured launch %q was allowed as PTY", args)
		}
	}
	for _, args := range [][]string{nil, {"--model", "provider/model"}, {"--mode", "text"}} {
		if isStructuredOMPArgs(args) {
			t.Errorf("normal PTY launch %q was wrongly rejected", args)
		}
	}
}
