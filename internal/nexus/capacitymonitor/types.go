package capacitymonitor

import (
	"crypto/rand"
	"errors"
	"math"
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

// Validate ensures 0 <= exhausted < critical < warning < recovery <= 100
// with finite values. NaN passes every comparison, so it is rejected explicitly.
func (t Thresholds) Validate() error {
	for _, v := range []float64{t.Warning, t.Critical, t.Exhausted, t.Recovery} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return errors.New("invalid thresholds: values must be finite")
		}
	}
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

// RealJitter adds 0-10% randomized jitter. The bound holds for every positive
// base: sub-millisecond bases jitter by a fraction of a nanosecond slice.
type RealJitter struct{}

func (RealJitter) Duration(base time.Duration) time.Duration {
	if base <= 0 {
		return base
	}
	tenth := int64(base / 10)
	if tenth <= 0 {
		tenth = 1
	}
	n, err := rand.Int(rand.Reader, big.NewInt(tenth))
	if err != nil {
		return base
	}
	add := time.Duration(n.Int64())
	if add > 0 && base > time.Duration(math.MaxInt64)-add {
		return time.Duration(math.MaxInt64)
	}
	return base + add
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
	IsLeader            bool            `json:"is_leader"`
	LeaderOwner         string          `json:"leader_owner"`
	Running             bool            `json:"running"`
	ConsecutiveFailures int             `json:"consecutive_failures"`
	MonitoredResources  int             `json:"monitored_resources"`
	LastTickAt          time.Time       `json:"last_tick_at"`
	Resources           []ResourceState `json:"resources"`
}

// ProbeRegistry provides access to safe read-only quota probes for providers.
type ProbeRegistry interface {
	For(provider string) (quota.Probe, bool)
	Providers() []string
}

// Dependencies supplies injected components to the monitor.
// Activity/power detectors are intentionally absent in v1 (see slice doc);
// the scheduler treats the host as neither active nor low-power until T8.
// Recommender is optional: when nil the monitor publishes transitions without
// a candidate (never a phantom recommendation). When set, WARNING/CRITICAL/
// EXHAUSTED publications include the advisory candidate + reason; the monitor
// never migrates workloads itself.
// Dispatcher is the optional single-dispatcher gate shared with the legacy
// QuotaDropMonitor. When nil it allows every announcement (previous
// behavior); when set, the first loop to claim a condition per cycle wins
// and the other stays silent, so the bus never carries duplicates.
type Dependencies struct {
	Profiles       func() ([]model.Profile, error)
	ProbeRegistry  ProbeRegistry
	Store          *store.Store
	Bus            *events.Bus
	Clock          Clock
	JitterSource   JitterSource
	Recommender    Recommender
	Dispatcher     *Coordinator
	MaxConcurrency int
	OwnerID        string
	LeaseTTL       time.Duration
	BaseInterval   time.Duration
}
