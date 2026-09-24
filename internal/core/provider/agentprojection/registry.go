package agentprojection

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

func NewRegistry() *Registry {
	return &Registry{adapters: map[string]Adapter{}}
}

func (r *Registry) Register(adapter Adapter) error {
	if adapter == nil {
		return fmt.Errorf("agent projection adapter is required")
	}
	id := strings.ToLower(strings.TrimSpace(adapter.ProviderID()))
	if id == "" {
		return fmt.Errorf("agent projection provider id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[id]; exists {
		return fmt.Errorf("agent projection adapter %q already registered", id)
	}
	r.adapters[id] = adapter
	return nil
}

func (r *Registry) Get(provider string) (Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(provider))]
	return adapter, ok
}

func (r *Registry) Providers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.adapters))
	for id := range r.adapters {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
