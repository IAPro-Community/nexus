package quota

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

var (
	ErrProbeNotReadOnly = errors.New("probe must be read-only")
	ErrProbeUnavailable = errors.New("probe is unavailable")
)

// ProbeDescriptor declares probe capabilities, confidence, and freshness constraints.
type ProbeDescriptor struct {
	Sources         []model.UsageSource    `json:"sources"`
	RequiresNetwork bool                   `json:"requires_network"`
	ReadOnly        bool                   `json:"read_only"`
	MinimumInterval time.Duration          `json:"minimum_interval"`
	Confidence      model.UsageStatus      `json:"confidence"`
	TTL             time.Duration          `json:"ttl"`
	Granularity     model.UsageGranularity `json:"granularity"`
}

// Validate ensures the probe meets safety invariants (such as read-only guarantee).
func (d ProbeDescriptor) Validate() error {
	if !d.ReadOnly {
		return ErrProbeNotReadOnly
	}
	return nil
}

// Probe evaluates capacity for a given profile in a read-only manner.
type Probe interface {
	Descriptor() ProbeDescriptor
	Probe(context.Context, model.Profile) (model.UsageSnapshot, error)
}

// IsAttributableModelName returns true only if the snapshot has MODEL granularity and a non-empty ModelName.
func IsAttributableModelName(snap model.UsageSnapshot) bool {
	return snap.Granularity == model.UsageGranularityModel && strings.TrimSpace(snap.ModelName) != ""
}

// DisplayQuotaSubject returns the exact model name if attributable, otherwise the quota group or provider ID.
func DisplayQuotaSubject(snap model.UsageSnapshot, group string) string {
	if IsAttributableModelName(snap) {
		return snap.ModelName
	}
	if strings.TrimSpace(group) != "" {
		return strings.TrimSpace(group)
	}
	if strings.TrimSpace(snap.QuotaGroup) != "" {
		return strings.TrimSpace(snap.QuotaGroup)
	}
	if strings.TrimSpace(snap.ProviderID) != "" {
		return strings.TrimSpace(snap.ProviderID)
	}
	return "unknown"
}
