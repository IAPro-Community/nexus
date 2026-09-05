package quota

import (
	"context"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

type testFakeProbe struct {
	desc ProbeDescriptor
	snap model.UsageSnapshot
	err  error
}

func (p *testFakeProbe) Descriptor() ProbeDescriptor {
	return p.desc
}

func (p *testFakeProbe) Probe(ctx context.Context, prof model.Profile) (model.UsageSnapshot, error) {
	return p.snap, p.err
}

func TestProbeDescriptorRejectsNonReadOnly(t *testing.T) {
	desc := ProbeDescriptor{
		Sources:         []model.UsageSource{model.SourceOfficialAPI},
		RequiresNetwork: true,
		ReadOnly:        false,
		MinimumInterval: time.Minute,
		Confidence:      model.UsageLive,
		TTL:             5 * time.Minute,
		Granularity:     model.UsageGranularityGroup,
	}

	if err := desc.Validate(); err == nil {
		t.Fatal("expected error when ReadOnly is false, got nil")
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

	snapWithModel := model.UsageSnapshot{
		ProviderID:  "gemini",
		ModelName:   "gemini-2.5-pro",
		Granularity: model.UsageGranularityModel,
		Status:      model.UsageLive,
	}
	if got := DisplayQuotaSubject(snapWithModel, "gemini"); got != "gemini-2.5-pro" {
		t.Fatalf("subject = %q, want attributable model name 'gemini-2.5-pro'", got)
	}

	snapUnattributableModel := model.UsageSnapshot{
		ProviderID:  "gemini",
		ModelName:   "Gemini Pro",
		Granularity: model.UsageGranularityGroup,
		Status:      model.UsageLive,
	}
	if got := DisplayQuotaSubject(snapUnattributableModel, "gemini-group"); got != "gemini-group" {
		t.Fatalf("subject = %q, want quota group when model is not attributable", got)
	}
}

func TestIsAttributableModelName(t *testing.T) {
	if IsAttributableModelName(model.UsageSnapshot{
		Granularity: model.UsageGranularityModel,
		ModelName:   "gpt-5.6-sol",
	}) != true {
		t.Fatal("expected true for model granularity with model name")
	}

	if IsAttributableModelName(model.UsageSnapshot{
		Granularity: model.UsageGranularityGroup,
		ModelName:   "gpt-5.6-sol",
	}) != false {
		t.Fatal("expected false when granularity is not model")
	}

	if IsAttributableModelName(model.UsageSnapshot{
		Granularity: model.UsageGranularityModel,
		ModelName:   "",
	}) != false {
		t.Fatal("expected false when model name is empty")
	}
}
