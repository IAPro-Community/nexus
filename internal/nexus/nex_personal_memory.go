package nexus

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

const (
	nexPersonalScope             = "@personal"
	nexPersonalMemoryMaxEntries  = 100
	nexPersonalMemoryMaxKeyChars = 96
	nexPersonalMemoryMaxValChars = 500
)

type NexPersonalMemoryEntry struct {
	ID         string    `json:"id"`
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	Scope      string    `json:"scope"`
	Provenance string    `json:"provenance"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type nexPersonalMemoryDisk struct {
	Entries []NexPersonalMemoryEntry `json:"entries"`
}

// NexPersonalMemoryStore stores user-approved @personal facts separately from
// project semantic memory and Maestro ledgers, redacting secrets before write.
type NexPersonalMemoryStore struct {
	mu       sync.Mutex
	filePath string
	entries  map[string]NexPersonalMemoryEntry
}

func NewNexPersonalMemoryStore(filePath string) *NexPersonalMemoryStore {
	if strings.TrimSpace(filePath) == "" {
		dataDir, err := config.DataDir()
		if err != nil || strings.TrimSpace(dataDir) == "" {
			dataDir = filepath.Join(os.TempDir(), "iapro-nexus")
		}
		filePath = filepath.Join(dataDir, "nex-personal-memory.json")
	}
	st := &NexPersonalMemoryStore{
		filePath: filePath,
		entries:  make(map[string]NexPersonalMemoryEntry),
	}
	st.loadLocked()
	return st
}

func (s *NexPersonalMemoryStore) loadLocked() {
	raw, err := os.ReadFile(s.filePath)
	if err != nil {
		return
	}
	var disk nexPersonalMemoryDisk
	if err := json.Unmarshal(raw, &disk); err != nil {
		return
	}
	for _, entry := range disk.Entries {
		if strings.TrimSpace(entry.ID) == "" || strings.TrimSpace(entry.Key) == "" {
			continue
		}
		entry.Scope = nexPersonalScope
		s.entries[entry.ID] = entry
	}
}

func (s *NexPersonalMemoryStore) saveLocked() error {
	if strings.TrimSpace(s.filePath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return err
	}
	list := s.sortedListLocked()
	raw, err := json.MarshalIndent(nexPersonalMemoryDisk{Entries: list}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

func (s *NexPersonalMemoryStore) sortedListLocked() []NexPersonalMemoryEntry {
	out := make([]NexPersonalMemoryEntry, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].UpdatedAt.After(out[j].UpdatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func randomMemoryID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "mem-" + hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("mem-%d", time.Now().UnixNano())
}

func trimValidUTF8(s string, maxChars int) string {
	s = strings.TrimSpace(s)
	if maxChars <= 0 || len(s) <= maxChars {
		return s
	}
	return strings.TrimSpace(strings.ToValidUTF8(s[:maxChars], ""))
}

func (s *NexPersonalMemoryStore) List() []NexPersonalMemoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sortedListLocked()
}

func (s *NexPersonalMemoryStore) Upsert(key, value, provenance string, now time.Time) (NexPersonalMemoryEntry, error) {
	cleanKey := trimValidUTF8(security.Redact(key), nexPersonalMemoryMaxKeyChars)
	cleanVal := trimValidUTF8(security.Redact(value), nexPersonalMemoryMaxValChars)
	if cleanKey == "" || cleanVal == "" {
		return NexPersonalMemoryEntry{}, fmt.Errorf("memory key and value are required")
	}
	if provenance != "assistant_approved" {
		provenance = "user_explicit"
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, existing := range s.entries {
		if strings.EqualFold(existing.Key, cleanKey) {
			existing.Key = cleanKey
			existing.Value = cleanVal
			existing.Provenance = provenance
			existing.UpdatedAt = now
			s.entries[id] = existing
			if err := s.saveLocked(); err != nil {
				return NexPersonalMemoryEntry{}, err
			}
			return existing, nil
		}
	}

	if len(s.entries) >= nexPersonalMemoryMaxEntries {
		sorted := s.sortedListLocked()
		oldest := sorted[len(sorted)-1]
		delete(s.entries, oldest.ID)
	}

	entry := NexPersonalMemoryEntry{
		ID:         randomMemoryID(),
		Key:        cleanKey,
		Value:      cleanVal,
		Scope:      nexPersonalScope,
		Provenance: provenance,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	s.entries[entry.ID] = entry
	if err := s.saveLocked(); err != nil {
		return NexPersonalMemoryEntry{}, err
	}
	return entry, nil
}

func (s *NexPersonalMemoryStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.entries[strings.TrimSpace(id)]; !ok {
		return false
	}
	delete(s.entries, strings.TrimSpace(id))
	_ = s.saveLocked()
	return true
}

func (s *NexPersonalMemoryStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = make(map[string]NexPersonalMemoryEntry)
	return s.saveLocked()
}

func (s *NexPersonalMemoryStore) BoundedSummary(maxChars int) string {
	s.mu.Lock()
	list := s.sortedListLocked()
	s.mu.Unlock()
	if len(list) == 0 || maxChars <= 0 {
		return ""
	}
	var b strings.Builder
	for _, entry := range list {
		line := entry.Key + ": " + entry.Value
		if b.Len() > 0 {
			line = " | " + line
		}
		if b.Len()+len(line) > maxChars {
			break
		}
		b.WriteString(line)
	}
	return b.String()
}
