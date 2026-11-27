package driver

import (
	"fmt"
	"os"
	"sync"

	sdk "github.com/IAPro-Community/IAPro-SDK/providers"
)

// Registry manages registered provider control drivers.
type Registry struct {
	mu       sync.Mutex
	bindings *sdk.Bindings[ControlDriver]
}

var (
	defaultRegistry *Registry
	regOnce         sync.Once
)

// DefaultRegistry returns the global driver registry.
func DefaultRegistry() *Registry {
	regOnce.Do(func() {
		defaultRegistry = NewRegistry()
		defaultRegistry.Register(NewCodexDriver())
		defaultRegistry.Register(NewAGYDriver())
		defaultRegistry.Register(NewClaudeDriver())
		defaultRegistry.Register(NewOpenCodeDriver())
		defaultRegistry.Register(NewCursorDriver())
		defaultRegistry.Register(NewZCodeDriver())
		defaultRegistry.Register(NewShellDriver())
		defaultRegistry.Register(NewFakeDriver())
		// Experimental harnesses are opt-in until their authenticated
		// protocol/session E2E coverage is complete. DSH is discoverable
		// but cannot be launched as an ordinary PTY.
		if os.Getenv("NEXUS_EXPERIMENTAL_HARNESSES") == "1" {
			for _, id := range []string{"pi", "omp", "mimo", "dsh"} {
				defaultRegistry.Register(NewPreviewHarnessDriver(id))
			}
		}
	})
	return defaultRegistry
}

// NewRegistry creates a new driver registry.
func NewRegistry() *Registry {
	return &Registry{
		bindings: sdk.NewBindings[ControlDriver](),
	}
}

// Register adds a driver to the registry.
func (r *Registry) Register(d ControlDriver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Preserve replacement behavior for explicit driver overrides.
	r.bindings.Unregister(d.ProviderID())
	_ = r.bindings.Register(d.ProviderID(), d)
}

func (r *Registry) Get(providerID string) (ControlDriver, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.bindings.Get(providerID)
	if !ok {
		return nil, fmt.Errorf("no control driver registered for provider %q", providerID)
	}
	return d, nil
}

func (r *Registry) List() []ControlDriver {
	r.mu.Lock()
	defer r.mu.Unlock()
	entries := r.bindings.List()
	out := make([]ControlDriver, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Value)
	}
	return out
}
