package driver

import (
	"fmt"
	"sync"
)

// Registry manages registered provider control drivers.
type Registry struct {
	mu      sync.RWMutex
	drivers map[string]ControlDriver
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
		defaultRegistry.Register(NewGeminiDriver())
		defaultRegistry.Register(NewCursorDriver())
		defaultRegistry.Register(NewFakeDriver())
		defaultRegistry.Register(NewShellDriver())
	})
	return defaultRegistry
}

// NewRegistry creates a new driver registry.
func NewRegistry() *Registry {
	return &Registry{
		drivers: make(map[string]ControlDriver),
	}
}

// Register adds a driver to the registry.
func (r *Registry) Register(d ControlDriver) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.drivers[d.ProviderID()] = d
}

// Get returns the driver for a provider ID.
func (r *Registry) Get(providerID string) (ControlDriver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, ok := r.drivers[providerID]
	if !ok {
		return nil, fmt.Errorf("no control driver registered for provider %q", providerID)
	}
	return d, nil
}

// List returns all registered drivers.
func (r *Registry) List() []ControlDriver {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []ControlDriver
	for _, d := range r.drivers {
		list = append(list, d)
	}
	return list
}
