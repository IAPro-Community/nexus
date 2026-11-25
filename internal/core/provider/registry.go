package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

const defaultDetectTimeout = 5 * time.Second

// Registry manages registered provider adapters in a thread-safe manner.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

var (
	defaultRegMu   sync.Mutex
	defaultReg     *Registry
	builtinFactors []func() Provider
)

// NewRegistry creates a new Provider Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
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

// Unregister removes a provider adapter by ID (primarily for tests).
func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, strings.ToLower(strings.TrimSpace(id)))
}

// Get looks up a provider by ID.
func (r *Registry) Get(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.providers[strings.ToLower(strings.TrimSpace(id))]
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

// DetectAll queries installation and version status for all registered providers
// concurrently, bounding each probe with defaultDetectTimeout.
func (r *Registry) DetectAll(ctx context.Context) map[string]model.DetectionResult {
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.RLock()
	snapshot := make(map[string]Provider, len(r.providers))
	for id, p := range r.providers {
		snapshot[id] = p
	}
	r.mu.RUnlock()

	results := make(map[string]model.DetectionResult, len(snapshot))
	var resMu sync.Mutex
	var wg sync.WaitGroup
	for id, p := range snapshot {
		wg.Add(1)
		go func(provID string, prov Provider) {
			defer wg.Done()
			probeCtx, cancel := context.WithTimeout(ctx, defaultDetectTimeout)
			defer cancel()
			det := prov.Detect(probeCtx)
			resMu.Lock()
			results[provID] = det
			resMu.Unlock()
		}(id, p)
	}
	wg.Wait()
	return results
}
