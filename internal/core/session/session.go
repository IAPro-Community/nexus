package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

var (
	metadataMu sync.RWMutex
)

// Metadata stores user local annotations like pinned status and custom aliases.
type Metadata struct {
	Pinned      bool   `json:"pinned,omitempty"`
	CustomTitle string `json:"custom_title,omitempty"`
}

// Store coordinates universal session discovery, local annotations, and search.
type Store struct {
	metadata map[string]Metadata // key: provider:session_id
}

// NewStore creates a new Session Store and loads persisted annotations.
func NewStore() *Store {
	s := &Store{
		metadata: make(map[string]Metadata),
	}
	_ = s.load()
	return s
}

func sessionKey(provider, sessionID string) string {
	return fmt.Sprintf("%s:%s", provider, sessionID)
}

// PinSession pins or unpins a session.
func (s *Store) PinSession(provider, sessionID string, pinned bool) error {
	metadataMu.Lock()
	defer metadataMu.Unlock()

	k := sessionKey(provider, sessionID)
	m := s.metadata[k]
	m.Pinned = pinned
	s.metadata[k] = m
	return s.save()
}

// Aggregate gathers sessions from multiple providers, applies local annotations, and sorts them.
func (s *Store) Aggregate(sessions []model.Session, workspaceFilter string) []model.Session {
	metadataMu.RLock()
	defer metadataMu.RUnlock()

	var result []model.Session
	seen := make(map[string]bool)

	for _, ses := range sessions {
		if ses.ID == "" {
			continue
		}
		k := sessionKey(ses.ProviderID, ses.ID)
		if seen[k] {
			continue
		}
		seen[k] = true

		if m, exists := s.metadata[k]; exists {
			ses.Pinned = m.Pinned
			if m.CustomTitle != "" {
				ses.Title = m.CustomTitle
			}
		}

		result = append(result, ses)
	}

	cwd, _ := os.Getwd()
	if workspaceFilter == "" {
		workspaceFilter = cwd
	}

	sort.Slice(result, func(i, j int) bool {
		// 1. Pinned sessions first
		if result[i].Pinned != result[j].Pinned {
			return result[i].Pinned
		}
		// 2. Current workspace affinity
		iCwd := workspaceFilter != "" && (result[i].Workspace == workspaceFilter || strings.HasPrefix(workspaceFilter, result[i].Workspace))
		jCwd := workspaceFilter != "" && (result[j].Workspace == workspaceFilter || strings.HasPrefix(workspaceFilter, result[j].Workspace))
		if iCwd != jCwd {
			return iCwd
		}
		// 3. Most recent first
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	return result
}

// Search filters sessions by query string across title, ID, workspace, provider, or profile.
func (s *Store) Search(sessions []model.Session, query string) []model.Session {
	if strings.TrimSpace(query) == "" {
		return sessions
	}
	q := strings.ToLower(strings.TrimSpace(query))

	var matches []model.Session
	for _, ses := range sessions {
		if strings.Contains(strings.ToLower(ses.Title), q) ||
			strings.Contains(strings.ToLower(ses.ID), q) ||
			strings.Contains(strings.ToLower(ses.Workspace), q) ||
			strings.Contains(strings.ToLower(ses.ProviderID), q) ||
			strings.Contains(strings.ToLower(ses.ProfileID), q) {
			matches = append(matches, ses)
		}
	}
	return matches
}

// GroupByWorkspace aggregates sessions and workspace bindings into WorkspaceInfo structs.
func (s *Store) GroupByWorkspace(sessions []model.Session, cfg config.Config) []model.WorkspaceInfo {
	wsMap := make(map[string]*model.WorkspaceInfo)

	// Add known workspace bindings from config
	for path, bindings := range cfg.Bindings {
		wsMap[path] = &model.WorkspaceInfo{
			Path:     path,
			Bindings: bindings,
			Sessions: []model.Session{},
		}
	}

	for _, ses := range sessions {
		ws := ses.Workspace
		if ws == "" {
			ws = "(unknown workspace)"
		}
		info, exists := wsMap[ws]
		if !exists {
			bindings := cfg.Bindings[ws]
			if bindings == nil {
				bindings = make(map[string]string)
			}
			info = &model.WorkspaceInfo{
				Path:     ws,
				Bindings: bindings,
				Sessions: []model.Session{},
			}
			wsMap[ws] = info
		}
		info.Sessions = append(info.Sessions, ses)
		if ses.UpdatedAt.After(info.LastTouch) {
			info.LastTouch = ses.UpdatedAt
		}
	}

	var list []model.WorkspaceInfo
	for _, info := range wsMap {
		list = append(list, *info)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].LastTouch.After(list[j].LastTouch)
	})

	return list
}

func (s *Store) load() error {
	stateDir, err := config.StateDir()
	if err != nil {
		return err
	}
	path := filepath.Join(stateDir, "session_metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.metadata)
}

func (s *Store) save() error {
	stateDir, err := config.StateDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.metadata, "", "  ")
	if err != nil {
		return err
	}
	target := filepath.Join(stateDir, "session_metadata.json")
	temp := filepath.Join(stateDir, fmt.Sprintf("meta.tmp.%d", time.Now().UnixNano()))
	if err := os.WriteFile(temp, append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(temp, target)
}
