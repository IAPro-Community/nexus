package quota

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

var (
	ErrProbeNotReadOnly    = errors.New("probe must be read-only")
	ErrProbeUnavailable    = errors.New("probe is unavailable")
	ErrProbeBadGranularity = errors.New("probe has unknown granularity")
)

// ProbeGranularity describes the attribution level of a probe observation.
// It lives in the quota package (not model) so the shared model stays untouched;
// probes opt into MODEL only when the source names an exact model.
type ProbeGranularity string

const (
	ProbeGranularityModel   ProbeGranularity = "MODEL"
	ProbeGranularityGroup   ProbeGranularity = "GROUP"
	ProbeGranularityAccount ProbeGranularity = "ACCOUNT"
)

// ProbeDescriptor declares probe capabilities, confidence, and freshness constraints.
type ProbeDescriptor struct {
	Sources         []model.UsageSource `json:"sources"`
	RequiresNetwork bool                `json:"requires_network"`
	ReadOnly        bool                `json:"read_only"`
	MinimumInterval time.Duration       `json:"minimum_interval"`
	Confidence      model.UsageStatus   `json:"confidence"`
	TTL             time.Duration       `json:"ttl"`
	Granularity     ProbeGranularity    `json:"granularity"`
}

// Validate ensures the probe meets safety invariants (read-only guarantee,
// known granularity, sane intervals).
func (d ProbeDescriptor) Validate() error {
	if !d.ReadOnly {
		return ErrProbeNotReadOnly
	}
	switch d.Granularity {
	case ProbeGranularityModel, ProbeGranularityGroup, ProbeGranularityAccount:
	default:
		return ErrProbeBadGranularity
	}
	if d.MinimumInterval < 0 || d.TTL < 0 {
		return errors.New("probe intervals must not be negative")
	}
	return nil
}

// Probe evaluates capacity for a given profile in a read-only manner.
// Implementations must honor context cancellation and must never mutate
// provider, profile, or host state.
type Probe interface {
	Descriptor() ProbeDescriptor
	Probe(context.Context, model.Profile) (model.UsageSnapshot, error)
}

// IsAttributableModelName returns true only for a non-empty model name.
// Nexus never invents model identity: without an exact source-provided name,
// callers must fall back to group or provider scope.
func IsAttributableModelName(snap model.UsageSnapshot) bool {
	return strings.TrimSpace(snap.ModelName) != ""
}

// DisplayQuotaSubject returns the exact model name if attributable, otherwise
// the quota group or provider ID. It never returns an empty string.
func DisplayQuotaSubject(snap model.UsageSnapshot, group string) string {
	if IsAttributableModelName(snap) {
		return strings.TrimSpace(snap.ModelName)
	}
	if strings.TrimSpace(group) != "" {
		return strings.TrimSpace(group)
	}
	if strings.TrimSpace(snap.ProviderID) != "" {
		return strings.TrimSpace(snap.ProviderID)
	}
	return "unknown"
}
