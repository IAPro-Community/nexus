package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	sdk "github.com/IAPro-Community/IAPro-SDK/providers"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

const defaultDetectTimeout = 5 * time.Second

// Registry manages registered provider adapters in a thread-safe manner.
type Registry struct {
	bindings *sdk.Bindings[Provider]
}

var (
	defaultRegMu   sync.Mutex
	defaultReg     *Registry
	builtinFactors []func() Provider
)

// NewRegistry creates a new Provider Registry.
func NewRegistry() *Registry {
	return &Registry{
		bindings: sdk.NewBindings[Provider](),
	}
}

// RegisterBuiltin registers a constructor for a built-in provider adapter so
// DefaultRegistry stays extensible without adapter-layer import cycles.
func RegisterBuiltin(factory func() Provider) {
	if factory == nil {
		return
	}
	defaultRegMu.Lock()
	defer defaultRegMu.Unlock()
	builtinFactors = append(builtinFactors, factory)
	if defaultReg != nil {
		if p := factory(); p != nil {
			_ = defaultReg.Register(p)
		}
	}
}

// DefaultRegistry returns the singleton registry pre-populated with built-in providers.
func DefaultRegistry() *Registry {
	defaultRegMu.Lock()
	defer defaultRegMu.Unlock()
	if defaultReg == nil {
		defaultReg = NewRegistry()
		for _, factory := range builtinFactors {
			if p := factory(); p != nil {
				_ = defaultReg.Register(p)
			}
		}
	}
	return defaultReg
}

// ResetDefaultRegistryForTest rebuilds the singleton registry from built-in factories.
func ResetDefaultRegistryForTest() {
	defaultRegMu.Lock()
	defer defaultRegMu.Unlock()
	defaultReg = NewRegistry()
	for _, factory := range builtinFactors {
		if p := factory(); p != nil {
			_ = defaultReg.Register(p)
		}
	}
}

// Register adds a provider adapter to the registry.
func (r *Registry) Register(p Provider) error {
	if p == nil {
		return fmt.Errorf("provider is required")
	}
	return r.bindings.Register(string(p.ID()), p)
}

func (r *Registry) Unregister(id string) { r.bindings.Unregister(id) }

func (r *Registry) Get(id string) (Provider, bool) { return r.bindings.Get(id) }

func (r *Registry) List() []Provider {
	bindings := r.bindings.List()
	out := make([]Provider, 0, len(bindings))
	for _, entry := range bindings {
		out = append(out, entry.Value)
	}
	return out
}

func (r *Registry) DetectAll(ctx context.Context) map[string]model.DetectionResult {
	return sdk.DetectBindings(ctx, r.bindings.List(), defaultDetectTimeout, 4, func(ctx context.Context, p Provider) model.DetectionResult {
		return p.Detect(ctx)
	})
}
