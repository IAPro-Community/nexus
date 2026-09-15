package quota

import (
	"context"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

type fakeCapacityProbe struct {
	desc ProbeDescriptor
	snap model.UsageSnapshot
	err  error
}

func (p *fakeCapacityProbe) Descriptor() ProbeDescriptor { return p.desc }

func (p *fakeCapacityProbe) Probe(ctx context.Context, prof model.Profile) (model.UsageSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return model.UsageSnapshot{}, err
	}
	if p.err != nil {
		return model.UsageSnapshot{}, p.err
	}
	out := p.snap
	out.ProviderID = prof.Provider
	out.ProfileID = prof.Name
	return out, nil
}

func validProbeDescriptor() ProbeDescriptor {
	return ProbeDescriptor{
		Sources:         []model.UsageSource{model.SourceOfficialAPI},
		RequiresNetwork: true,
		ReadOnly:        true,
		MinimumInterval: time.Minute,
		Confidence:      model.UsageLive,
		TTL:             5 * time.Minute,
		Granularity:     ProbeGranularityGroup,
	}
}

func TestProbeDescriptorAcceptsValid(t *testing.T) {
	if err := validProbeDescriptor().Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestProbeDescriptorRejectsNonReadOnly(t *testing.T) {
	desc := validProbeDescriptor()
	desc.ReadOnly = false
	if err := desc.Validate(); err == nil {
		t.Fatal("expected error when ReadOnly is false, got nil")
	}
}

func TestProbeDescriptorRejectsUnknownGranularity(t *testing.T) {
	desc := validProbeDescriptor()
	desc.Granularity = ProbeGranularity("PLANET")
	if err := desc.Validate(); err == nil {
		t.Fatal("expected error for unknown granularity, got nil")
	}
}

func TestProbeDescriptorRejectsNegativeIntervals(t *testing.T) {
	desc := validProbeDescriptor()
	desc.TTL = -time.Second
	if err := desc.Validate(); err == nil {
		t.Fatal("expected error for negative TTL, got nil")
	}
}

func TestDisplayQuotaSubjectPrefersExactModel(t *testing.T) {
	snap := model.UsageSnapshot{
		ProviderID: "gemini",
		ModelName:  "gemini-2.5-pro",
		Status:     model.UsageLive,
	}
	if got := DisplayQuotaSubject(snap, "gemini"); got != "gemini-2.5-pro" {
		t.Fatalf("subject = %q, want attributable model name", got)
	}
}

func TestDisplayQuotaSubjectDoesNotInventModel(t *testing.T) {
	snap := model.UsageSnapshot{
		ProviderID: "gemini",
		ModelName:  "",
		Status:     model.UsageUnknown,
	}
	if got := DisplayQuotaSubject(snap, "gemini"); got != "gemini" {
		t.Fatalf("subject = %q, want quota group 'gemini'", got)
	}
	if got := DisplayQuotaSubject(snap, ""); got != "gemini" {
		t.Fatalf("subject = %q, want provider fallback 'gemini'", got)
	}
	empty := model.UsageSnapshot{}
	if got := DisplayQuotaSubject(empty, ""); got != "unknown" {
		t.Fatalf("subject = %q, want 'unknown', never empty", got)
	}
}

func TestIsAttributableModelName(t *testing.T) {
	if !IsAttributableModelName(model.UsageSnapshot{ModelName: "gpt-5.6-sol"}) {
		t.Fatal("expected true for non-empty model name")
	}
	if IsAttributableModelName(model.UsageSnapshot{ModelName: "   "}) {
		t.Fatal("expected false for blank model name")
	}
	if IsAttributableModelName(model.UsageSnapshot{}) {
		t.Fatal("expected false for empty snapshot")
	}
}

func TestFakeProbeBindsProfileAndHonorsCancel(t *testing.T) {
	p := &fakeCapacityProbe{desc: validProbeDescriptor(), snap: model.UsageSnapshot{Status: model.UsageLive}}
	got, err := p.Probe(context.Background(), model.Profile{Provider: "codex", Name: "kiver"})
	if err != nil {
		t.Fatalf("Probe() = %v, want nil", err)
	}
	if got.ProviderID != "codex" || got.ProfileID != "kiver" {
		t.Fatalf("profile binding = %q/%q, want codex/kiver", got.ProviderID, got.ProfileID)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Probe(ctx, model.Profile{}); err == nil {
		t.Fatal("expected context error, got nil")
	}
}
