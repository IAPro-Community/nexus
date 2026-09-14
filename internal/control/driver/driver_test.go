package driver

import (
	"context"
	"reflect"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestDriverRegistryAndCapabilities(t *testing.T) {
	reg := DefaultRegistry()
	ctx := context.Background()

	providers := []string{"codex", "agy", "claude", "opencode", "gemini", "fake"}
	for _, p := range providers {
		d, err := reg.Get(p)
		if err != nil {
			t.Fatalf("driver not found for %s: %v", p, err)
		}
		if d.ProviderID() != p {
			t.Errorf("expected provider ID %s, got %s", p, d.ProviderID())
		}

		effCaps := d.EffectiveCaps(ctx, model.Profile{Name: "default"})
		if effCaps.Process.Status != CapabilitySupported || effCaps.Terminal.Status != CapabilitySupported {
			t.Errorf("driver %s expected Process and Terminal supported, got %+v", p, effCaps)
		}

		// Truthful test: Codex and OpenCode without server adapters should NOT claim StructuredEvents supported
		if p == "codex" || p == "opencode" {
			if effCaps.StructuredEvents.Status == CapabilitySupported {
				t.Errorf("driver %s must not claim StructuredEvents supported without real adapter", p)
			}
		}

		// Test CanResume with valid and empty session ID
		canResumeEmpty, reason := d.CanResume(ctx, model.Profile{Name: "default"}, "")
		if canResumeEmpty {
			t.Errorf("driver %s should reject empty session ID for resume", p)
		}
		if reason == "" {
			t.Errorf("driver %s should provide reason when resume is rejected", p)
		}

		canResumeValid, _ := d.CanResume(ctx, model.Profile{Name: "default"}, "sess-123")
		if !canResumeValid {
			t.Errorf("driver %s should allow valid session ID", p)
		}

		resumeArgs, err := d.BuildResumeArgs(ctx, model.Profile{Name: "default"}, "sess-123")
		if err != nil || len(resumeArgs) == 0 {
			t.Errorf("driver %s failed to build resume args: %v", p, err)
		}
	}

	// Unknown provider check
	if _, err := reg.Get("non-existent"); err == nil {
		t.Errorf("expected error for non-existent driver")
	}
}

func TestHeadlessKickoffArgsMatchProviderCLIContracts(t *testing.T) {
	ctx := context.Background()
	profile := model.Profile{Name: "default"}
	cases := []struct {
		name string
		d    ControlDriver
		want []string
	}{
		{name: "claude", d: NewClaudeDriver(), want: []string{"-p", "ship it"}},
		{name: "agy", d: NewAGYDriver(), want: []string{"-p", "ship it"}},
		{name: "gemini", d: NewGeminiDriver(), want: []string{"-p", "ship it"}},
		{name: "opencode", d: NewOpenCodeDriver(), want: []string{"run", "ship it"}},
		{name: "cursor", d: NewCursorDriver(), want: []string{"-p", "ship it", "--output-format", "text"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.d.BuildKickoffArgs(ctx, profile, "ship it")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("kickoff args mismatch: got %#v want %#v", got, tc.want)
			}
		})
	}
}
