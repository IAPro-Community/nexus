package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ViewGeometry represents the visual state of a single view/window.
type ViewGeometry struct {
	ViewID       string  `json:"viewId"`
	LogicalKey   string  `json:"logicalKey,omitempty"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Width        float64 `json:"width"`
	Height       float64 `json:"height"`
	ZIndex       int     `json:"zIndex,omitempty"`
	Minimized    bool    `json:"minimized,omitempty"`
	Maximized    bool    `json:"maximized,omitempty"`
}

// WorkspaceVisualState represents the complete visual state of a workspace desktop.
type WorkspaceVisualState struct {
	Version         int             `json:"version"`
	ProjectID       string          `json:"projectId"`
	ActiveView      string          `json:"activeView,omitempty"`
	PresentationMode bool          `json:"presentationMode,omitempty"`
	FocusedStack    string          `json:"focusedStack,omitempty"`
	Views           []ViewGeometry  `json:"views"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

// VisualStore persists workspace visual state across restarts.
type VisualStore struct {
	mu       sync.RWMutex
	filePath string
	states   map[string]*WorkspaceVisualState // projectID -> state
}

// NewVisualStore creates a new visual state store.
func NewVisualStore(filePath string) *VisualStore {
	s := &VisualStore{
		filePath: filePath,
		states:   make(map[string]*WorkspaceVisualState),
	}
	_ = s.load()
	return s
}

// DefaultVisualStore returns the global visual state store.
func DefaultVisualStore() *VisualStore {
	return defaultVisualStore
}

var (
	defaultVisualStore *VisualStore
	visualStoreOnce    sync.Once
)

func init() {
	visualStoreOnce.Do(func() {
		dataDir, err := os.UserHomeDir()
		if err != nil {
			dataDir = os.TempDir()
		}
		nexusDir := filepath.Join(dataDir, ".local", "share", "nexus")
		_ = os.MkdirAll(nexusDir, 0700)
		defaultVisualStore = NewVisualStore(filepath.Join(nexusDir, "workspace-visual.json"))
	})
}

// Get returns the visual state for a project, or nil if not found.
func (s *VisualStore) Get(projectID string) *WorkspaceVisualState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.states[projectID]
}

// Save persists the visual state for a project.
func (s *VisualStore) Save(state *WorkspaceVisualState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state.Version = 1
	state.UpdatedAt = time.Now()
	s.states[state.ProjectID] = state

	return s.saveLocked()
}

// Delete removes the visual state for a project.
func (s *VisualStore) Delete(projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, projectID)
	return s.saveLocked()
}

// List returns all stored visual states.
func (s *VisualStore) List() []*WorkspaceVisualState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*WorkspaceVisualState, 0, len(s.states))
	for _, state := range s.states {
		result = append(result, state)
	}
	return result
}

func (s *VisualStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var states []*WorkspaceVisualState
	if err := json.Unmarshal(data, &states); err != nil {
		return err
	}

	for _, state := range states {
		if state.ProjectID != "" {
			s.states[state.ProjectID] = state
		}
	}
	return nil
}

func (s *VisualStore) saveLocked() error {
	states := make([]*WorkspaceVisualState, 0, len(s.states))
	for _, state := range s.states {
		states = append(states, state)
	}

	data, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmpFile, s.filePath)
}
