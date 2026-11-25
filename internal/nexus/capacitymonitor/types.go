package capacitymonitor

import (
	"crypto/rand"
	"math"
	"math/big"
	"time"

	sdkcapacity "github.com/IAPro-Community/IAPro-SDK/capacity"
	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// State is the shared SDK health state for a provider/account/group/window.
type State = sdkcapacity.State

const (
	StateUnknown   = sdkcapacity.StateUnknown
	StateHealthy   = sdkcapacity.StateHealthy
	StateWarning   = sdkcapacity.StateWarning
	StateCritical  = sdkcapacity.StateCritical
	StateExhausted = sdkcapacity.StateExhausted
	StateRecovered = sdkcapacity.StateRecovered
)

// Thresholds is the shared SDK capacity threshold contract.
type Thresholds = sdkcapacity.Thresholds

// DefaultThresholds returns official standard Nexus thresholds (percent remaining).
func DefaultThresholds() Thresholds {
	return sdkcapacity.DefaultThresholds()
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
