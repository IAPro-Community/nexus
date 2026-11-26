package provider

import (
	"context"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/mock"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
)

func TestProviderRegistry(t *testing.T) {
	reg := NewRegistry()

	fake1 := &mock.FakeProvider{
		ProviderID:   model.ProviderCodex,
		ProviderName: "Codex",
		DetectResult: model.DetectionResult{Installed: true, Version: "1.0.0"},
	}
	fake2 := &mock.FakeProvider{
		ProviderID:   model.ProviderClaude,
		ProviderName: "Claude Code",
		DetectResult: model.DetectionResult{Installed: false},
	}

	if err := reg.Register(fake1); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(fake2); err != nil {
		t.Fatal(err)
	}

	// Duplicate register should fail
	if err := reg.Register(fake1); err == nil {
		t.Fatal("expected error registering duplicate provider")
	}

	p, found := reg.Get("codex")
	if !found || p.Name() != "Codex" {
		t.Fatalf("expected to get Codex provider, got %+v", p)
	}

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(list))
	}

	detections := reg.DetectAll(context.Background())
	if !detections["codex"].Installed || detections["claude"].Installed {
		t.Fatalf("unexpected detection results: %+v", detections)
	}
}

func TestProviderRegistryNormalizesRegisteredIDs(t *testing.T) {
	reg := NewRegistry()
	provider := &mock.FakeProvider{
		ProviderID:   model.ProviderID("CoDeX"),
		ProviderName: "Codex",
	}
	if err := reg.Register(provider); err != nil {
		t.Fatal(err)
	}
	got, ok := reg.Get("codex")
	if !ok || got != provider {
		t.Fatalf("registered provider was not normalized for lookup: ok=%v provider=%v", ok, got)
	}
}

type slowFakeProvider struct {
	*mock.FakeProvider
	delay time.Duration
}

func (s *slowFakeProvider) Detect(ctx context.Context) model.DetectionResult {
	select {
	case <-time.After(s.delay):
		return s.FakeProvider.Detect(ctx)
	case <-ctx.Done():
		return model.DetectionResult{Installed: false, Error: ctx.Err().Error()}
	}
}

func TestProviderRegistryParallelDetectAndExtensibleSpec(t *testing.T) {
	const customID = "custom-ext-test"
	defer spec.Unregister(customID)
	if err := spec.Register(spec.Spec{Spec: func() spec.Spec {
		s := spec.MustGet("codex")
		s.Spec.ID = customID
		s.Spec.DisplayName = "Custom Ext"
		s.Spec.ShortBadge = "EXT"
		return s
	}().Spec}); err != nil {
		t.Fatalf("spec.Register failed: %v", err)
	}
	gotSpec, ok := spec.Get(customID)
	if !ok || gotSpec.DisplayName != "Custom Ext" || gotSpec.ShortBadge != "EXT" {
		t.Fatalf("unexpected custom spec lookup: ok=%v spec=%+v", ok, gotSpec)
	}

	reg := NewRegistry()
	for _, id := range []string{"slow-a", "slow-b", "slow-c"} {
		if err := reg.Register(&slowFakeProvider{
			FakeProvider: &mock.FakeProvider{
				ProviderID:   model.ProviderID(id),
				ProviderName: id,
				DetectResult: model.DetectionResult{Installed: true, Version: "1.0"},
			},
			delay: 80 * time.Millisecond,
		}); err != nil {
			t.Fatal(err)
		}
	}

	start := time.Now()
	detections := reg.DetectAll(context.Background())
	elapsed := time.Since(start)
	if len(detections) != 3 {
		t.Fatalf("got %d detections, want 3", len(detections))
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("DetectAll took %v; expected parallel execution under 200ms", elapsed)
	}
}
