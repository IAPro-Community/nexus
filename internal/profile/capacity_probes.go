package profile

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/agy"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/claude"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/codex"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/gemini"
	"github.com/kivervinicius/ai-cli/internal/core/provider/adapters/opencode"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
)

// ProbeRegistry provides access to safe, read-only probes for each provider.
type ProbeRegistry interface {
	For(provider string) (quota.Probe, bool)
	Providers() []string
}

type defaultProbeRegistry struct {
	mu     sync.RWMutex
	probes map[string]quota.Probe
}

// NewProbeRegistry constructs the default production registry of provider probes.
func NewProbeRegistry() ProbeRegistry {
	r := &defaultProbeRegistry{
		probes: make(map[string]quota.Probe),
	}

	r.probes["codex"] = &adapterProbe{
		desc: quota.ProbeDescriptor{
			Sources:         []model.UsageSource{model.SourceLocalFiles, model.SourceCLIOutput},
			RequiresNetwork: false,
			ReadOnly:        true,
			MinimumInterval: 30 * time.Second,
			Confidence:      model.UsageLive,
			TTL:             5 * time.Minute,
			Granularity:     model.UsageGranularityGroup,
		},
		fn: func(ctx context.Context, p model.Profile) (model.UsageSnapshot, error) {
			return codex.New().GetUsage(ctx, p), nil
		},
	}

	r.probes["agy"] = &adapterProbe{
		desc: quota.ProbeDescriptor{
			Sources:         []model.UsageSource{model.SourceCLIOutput, model.SourceLocalFiles},
			RequiresNetwork: false,
			ReadOnly:        true,
			MinimumInterval: 30 * time.Second,
			Confidence:      model.UsageLive,
			TTL:             5 * time.Minute,
			Granularity:     model.UsageGranularityGroup,
		},
		fn: func(ctx context.Context, p model.Profile) (model.UsageSnapshot, error) {
			return agy.New().GetUsage(ctx, p), nil
		},
	}

	r.probes["claude"] = &adapterProbe{
		desc: quota.ProbeDescriptor{
			Sources:         []model.UsageSource{model.SourceLocalFiles, model.SourceCLIOutput},
			RequiresNetwork: false,
			ReadOnly:        true,
			MinimumInterval: 30 * time.Second,
			Confidence:      model.UsageLive,
			TTL:             5 * time.Minute,
			Granularity:     model.UsageGranularityGroup,
		},
		fn: func(ctx context.Context, p model.Profile) (model.UsageSnapshot, error) {
			return claude.New().GetUsage(ctx, p), nil
		},
	}

	r.probes["opencode"] = &adapterProbe{
		desc: quota.ProbeDescriptor{
			Sources:         []model.UsageSource{model.SourceCLIOutput},
			RequiresNetwork: false,
			ReadOnly:        true,
			MinimumInterval: 30 * time.Second,
			Confidence:      model.UsageLive,
			TTL:             5 * time.Minute,
			Granularity:     model.UsageGranularityGroup,
		},
		fn: func(ctx context.Context, p model.Profile) (model.UsageSnapshot, error) {
			return opencode.New().GetUsage(ctx, p), nil
		},
	}

	r.probes["gemini"] = &adapterProbe{
		desc: quota.ProbeDescriptor{
			Sources:         []model.UsageSource{model.SourceOfficialAPI, model.SourceCLIOutput},
			RequiresNetwork: true,
			ReadOnly:        true,
			MinimumInterval: 60 * time.Second,
			Confidence:      model.UsageLive,
			TTL:             5 * time.Minute,
			Granularity:     model.UsageGranularityGroup,
		},
		fn: func(ctx context.Context, p model.Profile) (model.UsageSnapshot, error) {
			return gemini.New().GetUsage(ctx, p), nil
		},
	}

	r.probes["cursor"] = &adapterProbe{
		desc: quota.ProbeDescriptor{
			Sources:         []model.UsageSource{model.SourceLocalFiles},
			RequiresNetwork: false,
			ReadOnly:        true,
			MinimumInterval: 60 * time.Second,
			Confidence:      model.UsageEstimated,
			TTL:             10 * time.Minute,
			Granularity:     model.UsageGranularityGroup,
		},
		fn: func(ctx context.Context, p model.Profile) (model.UsageSnapshot, error) {
			return model.UsageSnapshot{
				ProviderID: "cursor",
				ProfileID:  p.Name,
				Status:     model.UsageUnknown,
				Source:     model.SourceLocalFiles,
				FetchedAt:  time.Now(),
			}, nil
		},
	}

	return r
}

func (r *defaultProbeRegistry) For(provider string) (quota.Probe, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.probes[provider]
	return p, ok
}

func (r *defaultProbeRegistry) Providers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]string, 0, len(r.probes))
	for k := range r.probes {
		list = append(list, k)
	}
	sort.Strings(list)
	return list
}

type adapterProbe struct {
	desc quota.ProbeDescriptor
	fn   func(context.Context, model.Profile) (model.UsageSnapshot, error)
}

func (p *adapterProbe) Descriptor() quota.ProbeDescriptor {
	return p.desc
}

func (p *adapterProbe) Probe(ctx context.Context, prof model.Profile) (model.UsageSnapshot, error) {
	return p.fn(ctx, prof)
}
