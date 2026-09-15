package capacitymonitor

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// State represents the capacity health state of a profile/group/window.
type State string

const (
	StateUnknown   State = "UNKNOWN"
	StateHealthy   State = "HEALTHY"
	StateWarning   State = "WARNING"
	StateCritical  State = "CRITICAL"
	StateExhausted State = "EXHAUSTED"
	StateRecovered State = "RECOVERED"
)

// Thresholds defines capacity boundaries for warning, critical, exhausted, and recovery.
type Thresholds struct {
	Warning   float64 `json:"warning"`
	Critical  float64 `json:"critical"`
	Exhausted float64 `json:"exhausted"`
	Recovery  float64 `json:"recovery"`
}

// DefaultThresholds returns official standard Nexus thresholds (percent remaining).
func DefaultThresholds() Thresholds {
	return Thresholds{
		Warning:   20.0,
		Critical:  10.0,
		Exhausted: 0.0,
		Recovery:  25.0,
	}
}

// Validate ensures 0 <= exhausted < critical < warning < recovery <= 100.
func (t Thresholds) Validate() error {
	if t.Exhausted < 0 || t.Exhausted >= t.Critical {
		return errors.New("invalid thresholds: exhausted must be >= 0 and < critical")
	}
	if t.Critical >= t.Warning {
		return errors.New("invalid thresholds: critical must be < warning")
	}
	if t.Warning >= t.Recovery {
		return errors.New("invalid thresholds: warning must be < recovery")
	}
	if t.Recovery > 100 {
		return errors.New("invalid thresholds: recovery must be <= 100")
	}
	return nil
}

// Clock abstracts time measurement for deterministic testing.
type Clock interface {
	Now() time.Time
}

// RealClock uses standard system time.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

// JitterSource adds bounded random jitter to intervals.
type JitterSource interface {
	Duration(base time.Duration) time.Duration
}

// RealJitter adds 0-10% randomized jitter.
type RealJitter struct{}

func (RealJitter) Duration(base time.Duration) time.Duration {
	if base <= 0 {
		return base
	}
	maxJitterMs := int64(base.Milliseconds() / 10)
	if maxJitterMs <= 0 {
		maxJitterMs = 1000
	}
	n, err := rand.Int(rand.Reader, big.NewInt(maxJitterMs))
	if err != nil {
		return base
	}
	return base + time.Duration(n.Int64())*time.Millisecond
}

// ResourceState provides a snapshot of a monitored resource.
type ResourceState struct {
	Key              string    `json:"key"`
	Provider         string    `json:"provider"`
	Profile          string    `json:"profile"`
	QuotaGroup       string    `json:"quota_group"`
	WindowKind       string    `json:"window_kind"`
	State            State     `json:"state"`
	RemainingPercent *float64  `json:"remaining_percent,omitempty"`
	ResetDescription string    `json:"reset_description,omitempty"`
	LastObservedAt   time.Time `json:"last_observed_at"`
	TransitionedAt   time.Time `json:"transitioned_at"`
}

// Status represents the live diagnostic status of the capacity monitor.
type Status struct {
	IsLeader           bool            `json:"is_leader"`
	LeaderOwner        string          `json:"leader_owner"`
	Running            bool            `json:"running"`
	MonitoredResources int             `json:"monitored_resources"`
	LastTickAt         time.Time       `json:"last_tick_at"`
	Resources          []ResourceState `json:"resources"`
}

// ProbeRegistry provides access to safe read-only quota probes for providers.
type ProbeRegistry interface {
	For(provider string) (quota.Probe, bool)
	Providers() []string
}

// Dependencies supplies injected components to the monitor.
// Activity/power detectors are intentionally absent in v1 (see slice doc);
// the scheduler treats the host as neither active nor low-power until T8.
type Dependencies struct {
	Profiles       func() ([]model.Profile, error)
	ProbeRegistry  ProbeRegistry
	Store          *store.Store
	Bus            *events.Bus
	Clock          Clock
	JitterSource   JitterSource
	MaxConcurrency int
	OwnerID        string
	LeaseTTL       time.Duration
	BaseInterval   time.Duration
}
