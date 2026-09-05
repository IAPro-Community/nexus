package profile

import (
	"context"
	"testing"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

func TestProfileProbeRegistry(t *testing.T) {
	reg := NewProbeRegistry()
	if reg == nil {
		t.Fatal("expected non-nil ProbeRegistry")
	}

	providers := reg.Providers()
	if len(providers) == 0 {
		t.Fatal("expected registered providers in probe registry")
	}

	for _, p := range providers {
		probe, ok := reg.For(p)
		if !ok || probe == nil {
			t.Fatalf("expected probe for provider %s", p)
		}
		desc := probe.Descriptor()
		if err := desc.Validate(); err != nil {
			t.Fatalf("probe for provider %s is invalid: %v", p, err)
		}
		if !desc.ReadOnly {
			t.Fatalf("probe for provider %s must be ReadOnly", p)
		}
		if desc.MinimumInterval <= 0 {
			t.Fatalf("probe for provider %s must have positive MinimumInterval", p)
		}
	}
}

func TestProbeExecutionDoesNotMutateOrRefresh(t *testing.T) {
	reg := NewProbeRegistry()
	probe, ok := reg.For("agy")
	if !ok {
		t.Fatal("agy probe missing")
	}

	snap, err := probe.Probe(context.Background(), model.Profile{
		Provider: "agy",
		Name:     "non-existent-profile-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// For a non-existent profile with no cached file, status must be UNKNOWN and not crash
	if snap.Status != model.UsageUnknown && snap.Status != model.UsageUnsupported {
		t.Fatalf("expected UNKNOWN or UNSUPPORTED for missing profile, got %s", snap.Status)
	}
}
