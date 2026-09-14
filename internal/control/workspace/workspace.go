package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

// Project represents a managed developer workspace/repository.
type Project struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Path            string    `json:"path"`
	CreatedAt       time.Time `json:"created_at"`
	LastUsedAt      time.Time `json:"last_used_at"`
	RecencySequence uint64    `json:"recency_sequence,omitempty"`
}

// Store manages persistent projects across restarts.
type Store struct {
	mu                  sync.RWMutex
	filePath            string
	projects            map[string]Project
	nextRecencySequence uint64
}

var (
	defaultStore *Store
	storeOnce    sync.Once
)

// DefaultStore returns the global workspace/project store.
func DefaultStore() *Store {
	storeOnce.Do(func() {
		dataDir, err := config.DataDir()
		if err != nil {
			dataDir = filepath.Join(os.TempDir(), "ai-control")
		}
		_ = os.MkdirAll(dataDir, 0700)
		defaultStore = NewStore(filepath.Join(dataDir, "projects.json"))
	})
	return defaultStore
}

// NewStore initializes a Project store with persistence.
func NewStore(filePath string) *Store {
	s := &Store{
		filePath: filePath,
		projects: make(map[string]Project),
	}
	_ = s.load()

	// Ensure current working directory is always present
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		_ = s.ensureCwd(cwd)
	}

	return s
}

func (s *Store) ensureCwd(cwd string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	clean, err := config.CanonicalWorkspacePath(cwd)
	if err != nil {
		clean = filepath.Clean(cwd)
	}
	for _, p := range s.projects {
		if workspacePathsEquivalent(p.Path, clean) {
			return nil
		}
	}

	name := filepath.Base(clean)
	id := makeWorkspaceID(clean)
	s.nextRecencySequence++
	s.projects[clean] = Project{
		ID:              id,
		Name:            name,
		Path:            clean,
		CreatedAt:       time.Now(),
		LastUsedAt:      time.Now(),
		RecencySequence: s.nextRecencySequence,
	}
	return s.saveLocked()
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var list []Project
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	for _, p := range list {
		if p.Path != "" {
			if canonical, err := config.CanonicalWorkspacePath(p.Path); err == nil {
				p.Path = canonical
			}
			s.projects[p.Path] = p
			if p.RecencySequence > s.nextRecencySequence {
				s.nextRecencySequence = p.RecencySequence
			}
		}
	}
	return nil
}

func (s *Store) saveLocked() error {
	var list []Project
	for _, p := range s.projects {
		list = append(list, p)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmpFile, s.filePath)
}

// List returns all registered projects sorted with most recently used first.
func (s *Store) List() []Project {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Project
	for _, p := range s.projects {
		result = append(result, p)
	}
	// Deterministic ordering: most recently used first, then ID for stability.
	sort.SliceStable(result, func(i, j int) bool {
		if !result[i].LastUsedAt.Equal(result[j].LastUsedAt) {
			return result[i].LastUsedAt.After(result[j].LastUsedAt)
		}
		if result[i].RecencySequence != result[j].RecencySequence {
			return result[i].RecencySequence > result[j].RecencySequence
		}
		return result[i].ID < result[j].ID
	})
	return result
}

// Add registers a new project path.
func (s *Store) Add(path, name string) (Project, error) {
	clean, err := config.CanonicalExistingWorkspaceDir(path)
	if err != nil {
		return Project{}, err
	}

	if name == "" {
		name = filepath.Base(clean)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	p := Project{
		ID:         makeWorkspaceID(clean),
		Name:       name,
		Path:       clean,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}
	s.nextRecencySequence++
	p.RecencySequence = s.nextRecencySequence

	s.projects[clean] = p
	if err := s.saveLocked(); err != nil {
		return Project{}, err
	}
	return p, nil
}

// Remove deletes a project from management.
func (s *Store) Remove(idOrPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	targetKey := ""
	for key, p := range s.projects {
		if p.ID == idOrPath || workspacePathsEquivalent(p.Path, idOrPath) {
			targetKey = key
			break
		}
	}

	if targetKey == "" {
		return fmt.Errorf("project %q not found", idOrPath)
	}

	delete(s.projects, targetKey)
	return s.saveLocked()
}

// Touch updates the last used timestamp of a project.
func (s *Store) Touch(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, p := range s.projects {
		if workspacePathsEquivalent(p.Path, path) {
			p.LastUsedAt = time.Now()
			s.nextRecencySequence++
			p.RecencySequence = s.nextRecencySequence
			s.projects[key] = p
			_ = s.saveLocked()
			return
		}
	}
}

// workspacePathsEquivalent compares filesystem identity when both paths exist
// and falls back to canonical path text for creation or legacy records.
func workspacePathsEquivalent(left, right string) bool {
	return config.FilesystemPathsEquivalent(left, right)
}

// makeWorkspaceID derives a stable, collision-resistant identifier from the
// canonical absolute path (symlinks resolved). Two workspaces sharing a
// basename (e.g. /home/user/company/api and /home/user/personal/api) always
// get distinct IDs.
func makeWorkspaceID(path string) string {
	clean, err := config.CanonicalWorkspacePath(path)
	if err != nil {
		clean = filepath.Clean(path)
	}
	sum := sha256.Sum256([]byte(clean))
	return "ws-" + hex.EncodeToString(sum[:16])
}
