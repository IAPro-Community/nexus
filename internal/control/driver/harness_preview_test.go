package driver

import (
	"context"
	"reflect"
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
