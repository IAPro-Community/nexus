package profile

import (
	"context"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
)

// CapacityProbeRegistry bridges the Capacity Monitor to the canonical usage
// observation cache. Live collection remains owned by QuotaMonitorService,
// which already serializes provider refreshes and protects Codex TUI ownership.
// The probes here are deliberately read-only and never start provider CLIs,
// HTTP requests or app-server processes.
type CapacityProbeRegistry struct {
	probes map[string]quota.Probe
}

// NewCapacityProbeRegistry returns the providers for which Nexus already has a
// production usage collector. Unsupported providers remain absent so the
// Capacity Monitor projects UNKNOWN rather than inventing support.
func NewCapacityProbeRegistry() *CapacityProbeRegistry {
	return &CapacityProbeRegistry{probes: map[string]quota.Probe{
		"codex":  newObservedUsageProbe("codex", []model.UsageSource{model.SourceOfficialAPI, model.SourceObservation}),
		"agy":    newObservedUsageProbe("agy", []model.UsageSource{model.SourceCLIOutput, model.SourceLocalFiles}),
		"claude": newObservedUsageProbe("claude", []model.UsageSource{model.SourceOfficialAPI, model.SourceLocalFiles}),
		"gemini": newObservedUsageProbe("gemini", []model.UsageSource{model.SourceOfficialAPI, model.SourceLocalFiles}),
	}}
}

func (r *CapacityProbeRegistry) For(provider string) (quota.Probe, bool) {
	if r == nil {
		return nil, false
	}
	probe, ok := r.probes[strings.ToLower(strings.TrimSpace(provider))]
	return probe, ok
}

func (r *CapacityProbeRegistry) Providers() []string {
	if r == nil {
		return nil
	}
	// Keep the order stable for diagnostics and tests.
	order := []string{"codex", "agy", "claude", "gemini"}
	out := make([]string, 0, len(order))
	for _, provider := range order {
		if _, ok := r.probes[provider]; ok {
			out = append(out, provider)
		}
	}
	return out
}

type observedUsageProbe struct {
	provider string
	sources  []model.UsageSource
	read     func(provider, profile string) model.UsageSnapshot
}

func newObservedUsageProbe(provider string, sources []model.UsageSource) quota.Probe {
	return &observedUsageProbe{
		provider: strings.ToLower(strings.TrimSpace(provider)),
		sources:  append([]model.UsageSource(nil), sources...),
		read:     PeekUsageSnapshot,
	}
}

func (p *observedUsageProbe) Descriptor() quota.ProbeDescriptor {
	return quota.ProbeDescriptor{
		Sources:         append([]model.UsageSource(nil), p.sources...),
		RequiresNetwork: false,
		ReadOnly:        true,
		MinimumInterval: 15 * time.Second,
		Confidence:      model.UsageCached,
		TTL:             5 * time.Minute,
		Granularity:     quota.ProbeGranularityGroup,
	}
}

func (p *observedUsageProbe) Probe(ctx context.Context, profile model.Profile) (model.UsageSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return model.UsageSnapshot{}, err
	}
	provider := strings.ToLower(strings.TrimSpace(profile.Provider))
	if provider == "" {
		provider = p.provider
	}
	if provider != p.provider || strings.TrimSpace(profile.Name) == "" {
		return model.UsageSnapshot{}, quota.ErrProbeUnavailable
	}

	snapshot := p.read(p.provider, profile.Name)
	// A read-only observation probe never upgrades evidence. Even if a legacy
	// cache entry still carries LIVE, crossing this boundary makes it CACHED.
	if snapshot.Status == model.UsageLive {
		snapshot.Status = model.UsageCached
	}
	if !snapshot.AccountScope.Verifiable() && profile.AccountScope.Verifiable() {
		snapshot.AccountScope = profile.AccountScope
	}
	return snapshot, nil
}
