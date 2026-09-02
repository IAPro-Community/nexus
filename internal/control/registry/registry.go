package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
)

// Registry manages in-memory and persistent state of AI Control runtimes.
type Registry struct {
	mu           sync.RWMutex
	filePath     string
	sessions     map[string]RuntimeSession
	lastModified time.Time
}

var (
	singletonMu     sync.RWMutex
	defaultRegistry *Registry
	regOnce         sync.Once
)

// DefaultRegistry returns the singleton registry instance.
func DefaultRegistry() *Registry {
	singletonMu.RLock()
	reg := defaultRegistry
	singletonMu.RUnlock()
	if reg != nil {
		return reg
	}

	singletonMu.Lock()
	defer singletonMu.Unlock()

	if defaultRegistry == nil {
		regOnce.Do(func() {
			dataDir, err := config.DataDir()
			if err != nil {
				dataDir = filepath.Join(os.TempDir(), "ai-control")
			}
			path := filepath.Join(dataDir, "runtimes.json")
			defaultRegistry = NewRegistry(path)
		})
	}
	return defaultRegistry
}

// ResetDefaultRegistryForTest resets the singleton registry for isolated testing.
func ResetDefaultRegistryForTest() {
	singletonMu.Lock()
	defer singletonMu.Unlock()
	regOnce = sync.Once{}
	defaultRegistry = nil
}

// NewRegistry initializes a Registry storing state at the given file path.
func NewRegistry(filePath string) *Registry {
	r := &Registry{
		filePath: filePath,
		sessions: make(map[string]RuntimeSession),
	}
	_ = r.load()
	return r
}

func loadFromDisk(filePath string) (map[string]RuntimeSession, error) {
	sessions := make(map[string]RuntimeSession)
	if filePath == "" {
		return sessions, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return sessions, nil
		}
		return nil, err
	}
	var list []RuntimeSession
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	for _, s := range list {
		sessions[s.RuntimeID] = s
	}
	return sessions, nil
}

func (r *Registry) load() error {
	if r.filePath == "" {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(r.filePath), 0700)
	unlock, err := acquireFileLock(r.filePath)
	if err != nil {
		return err
	}
	defer unlock()

	m, err := loadFromDisk(r.filePath)
	if err != nil {
		return err
	}
	r.sessions = m
	if fi, err := os.Stat(r.filePath); err == nil {
		r.lastModified = fi.ModTime()
	}
	return nil
}

func (r *Registry) syncIfNeededLocked() {
	if r.filePath == "" {
		return
	}
	fi, err := os.Stat(r.filePath)
	if err != nil {
		return
	}
	if fi.ModTime().After(r.lastModified) {
		_ = r.load()
	}
}

func (r *Registry) saveLocked(mutators ...func(map[string]RuntimeSession)) error {
	if r.filePath == "" {
		if len(mutators) > 0 {
			for _, fn := range mutators {
				if fn != nil {
					fn(r.sessions)
				}
			}
		}
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(r.filePath), 0700)

	unlock, err := acquireFileLock(r.filePath)
	if err != nil {
		return err
	}
	defer unlock()

	// 1. Under cross-process flock, reload runtimes.json from disk into fresh map
	freshMap, err := loadFromDisk(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to reload runtimes from disk: %w", err)
	}

	// 2. Merge/apply the mutation onto that fresh map
	if len(mutators) > 0 {
		for _, fn := range mutators {
			if fn != nil {
				fn(freshMap)
			}
		}
	} else {
		for k, v := range r.sessions {
			freshMap[k] = v
		}
	}

	// 3. Write updated state atomically
	list := make([]RuntimeSession, 0, len(freshMap))
	for _, s := range freshMap {
		// Launch-only metadata must never be persisted: Env may contain
		// API keys/tokens from the parent environment and Args may embed
		// prompts or session IDs. The in-memory map keeps them; disk gets none.
		p := s
		p.Env = nil
		p.Args = nil
		p.Binary = ""
		list = append(list, p)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	tmp := fmt.Sprintf("%s.tmp.%d", r.filePath, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(data, '\n'), 0600); err != nil {
		return err
	}

	if err := os.Rename(tmp, r.filePath); err != nil {
		return err
	}

	// 4. Update in-memory cache and timestamp
	r.sessions = freshMap
	if fi, err := os.Stat(r.filePath); err == nil {
		r.lastModified = fi.ModTime()
	}
	return nil
}

// Register adds or updates a runtime session.
func (r *Registry) Register(s RuntimeSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s.RuntimeID == "" {
		return errors.New("runtime ID cannot be empty")
	}
	now := time.Now()
	if s.StartedAt.IsZero() {
		s.StartedAt = now
	}
	s.UpdatedAt = now

	return r.saveLocked(func(fresh map[string]RuntimeSession) {
		fresh[s.RuntimeID] = s
	})
}

// Get retrieves a runtime session by ID.
func (r *Registry) Get(runtimeID string) (RuntimeSession, bool) {
	r.mu.Lock()
	r.syncIfNeededLocked()
	defer r.mu.Unlock()

	s, ok := r.sessions[runtimeID]
	return s, ok
}

// UpdateState updates the lifecycle state of a runtime.
func (r *Registry) UpdateState(runtimeID string, state RuntimeState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var notFound bool
	err := r.saveLocked(func(fresh map[string]RuntimeSession) {
		s, ok := fresh[runtimeID]
		if !ok {
			notFound = true
			return
		}
		s.State = state
		s.UpdatedAt = time.Now()
		fresh[runtimeID] = s
	})
	if err != nil {
		return err
	}
	if notFound {
		return fmt.Errorf("runtime %q not found", runtimeID)
	}
	return nil
}

// UpdateProviderSessionID updates the underlying provider session ID.
func (r *Registry) UpdateProviderSessionID(runtimeID, providerSessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var notFound bool
	err := r.saveLocked(func(fresh map[string]RuntimeSession) {
		s, ok := fresh[runtimeID]
		if !ok {
			notFound = true
			return
		}
		s.ProviderSessionID = providerSessionID
		s.UpdatedAt = time.Now()
		fresh[runtimeID] = s
	})
	if err != nil {
		return err
	}
	if notFound {
		return fmt.Errorf("runtime %q not found", runtimeID)
	}
	return nil
}

// UpdateTitle updates the human-friendly title of a runtime session.
func (r *Registry) UpdateTitle(runtimeID, title string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var notFound bool
	err := r.saveLocked(func(fresh map[string]RuntimeSession) {
		s, ok := fresh[runtimeID]
		if !ok {
			notFound = true
			return
		}
		s.Title = title
		s.DynamicTitle = title
		s.UpdatedAt = time.Now()
		fresh[runtimeID] = s
	})
	if err != nil {
		return err
	}
	if notFound {
		return fmt.Errorf("runtime %q not found", runtimeID)
	}
	return nil
}

// UpdateAttention updates the attention metadata and dynamic title of a runtime session.
func (r *Registry) UpdateAttention(runtimeID string, state RuntimeState, reason, context, projectName, taskSummary, dynamicTitle string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var notFound bool
	err := r.saveLocked(func(fresh map[string]RuntimeSession) {
		s, ok := fresh[runtimeID]
		if !ok {
			notFound = true
			return
		}
		if state != "" {
			s.State = state
		}
		s.AttentionReason = reason
		s.AttentionContext = context
		if projectName != "" {
			s.ProjectName = projectName
		}
		if taskSummary != "" {
			s.LastTaskSummary = taskSummary
		}
		if dynamicTitle != "" {
			s.DynamicTitle = dynamicTitle
			s.Title = dynamicTitle
		}
		s.UpdatedAt = time.Now()
		fresh[runtimeID] = s
	})
	if err != nil {
		return err
	}
	if notFound {
		return fmt.Errorf("runtime %q not found", runtimeID)
	}
	return nil
}

// List returns all registered runtime sessions.
func (r *Registry) List() []RuntimeSession {
	r.mu.Lock()
	r.syncIfNeededLocked()
	defer r.mu.Unlock()

	result := make([]RuntimeSession, 0, len(r.sessions))
	for _, s := range r.sessions {
		result = append(result, s)
	}
	return result
}

// ListActive returns all active (running, starting, detached) sessions.
func (r *Registry) ListActive() []RuntimeSession {
	r.mu.Lock()
	r.syncIfNeededLocked()
	defer r.mu.Unlock()

	result := []RuntimeSession{}
	for _, s := range r.sessions {
		if s.IsActive() {
			result = append(result, s)
		}
	}
	return result
}

// Delete removes a runtime from the registry.
func (r *Registry) Delete(runtimeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.saveLocked(func(fresh map[string]RuntimeSession) {
		delete(fresh, runtimeID)
	})
}

// Reload explicitly reloads the registry state from disk.
func (r *Registry) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.load()
}
