package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// Registry manages registered provider adapters in a thread-safe manner.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates a new Provider Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider adapter to the registry.
func (r *Registry) Register(p Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := strings.ToLower(strings.TrimSpace(string(p.ID())))
	if id == "" {
		return fmt.Errorf("provider id is required")
	}
	if _, exists := r.providers[id]; exists {
		return fmt.Errorf("provider %q already registered", id)
	}
	r.providers[id] = p
	return nil
}

// Get looks up a provider by ID.
func (r *Registry) Get(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.providers[strings.ToLower(id)]
	return p, exists
}

// List returns all registered providers sorted by ID.
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []Provider
	for _, p := range r.providers {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID() < list[j].ID()
	})
	return list
}

// DetectAll queries installation and version status for all registered providers.
func (r *Registry) DetectAll(ctx context.Context) map[string]model.DetectionResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string]model.DetectionResult)
	for id, p := range r.providers {
		results[id] = p.Detect(ctx)
	}
	return results
}
