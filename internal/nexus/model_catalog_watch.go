package nexus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

const (
	ModelWatchFreshnessFresh   = "FRESH"
	ModelWatchFreshnessStale   = "STALE"
	ModelWatchFreshnessUnknown = "UNKNOWN"

	ModelChangeAdded          = "ADDED"
	ModelChangeRemoved        = "REMOVED"
	ModelChangeDeprecated     = "DEPRECATED"
	ModelChangeDefaultChanged = "DEFAULT_CHANGED"
	ModelChangeUnknownSource  = "UNKNOWN_SOURCE"

	modelWatchStaleAfter = 10 * time.Minute
	modelWatchMaxChanges = 100
)

type ModelWatchEntry struct {
	Provider   string    `json:"provider"`
	Profile    string    `json:"profile"`
	ModelID    string    `json:"model_id"`
	IsDefault  bool      `json:"is_default"`
	Deprecated bool      `json:"deprecated"`
	Source     string    `json:"source"`
	ObservedAt time.Time `json:"observed_at"`
}

type ModelCatalogChange struct {
	Fingerprint     string    `json:"fingerprint"`
	Kind            string    `json:"kind"`
	Provider        string    `json:"provider"`
	Profile         string    `json:"profile"`
	ModelID         string    `json:"model_id"`
	PreviousDefault string    `json:"previous_default,omitempty"`
	NewDefault      string    `json:"new_default,omitempty"`
	Source          string    `json:"source"`
	ObservedAt      time.Time `json:"observed_at"`
}

type ModelCatalogWatchSnapshot struct {
	Freshness  string               `json:"freshness"`
	ObservedAt *time.Time           `json:"observed_at,omitempty"`
	Entries    []ModelWatchEntry    `json:"entries"`
	Changes    []ModelCatalogChange `json:"changes"`
}

type modelWatchPersistedState struct {
	HasObserved bool                 `json:"has_observed"`
	ObservedAt  time.Time            `json:"observed_at,omitempty"`
	Entries     []ModelWatchEntry    `json:"entries"`
	Changes     []ModelCatalogChange `json:"changes"`
}

// ModelCatalogWatchService maintains a canonical, read-only model catalog
// observation cache without probing external provider CLIs or APIs.
type ModelCatalogWatchService struct {
	mu          sync.Mutex
	filePath    string
	hasObserved bool
	observedAt  time.Time
	entries     map[string]ModelWatchEntry
	defaults    map[string]string
	changes     []ModelCatalogChange
}

func NewModelCatalogWatchService(filePath string) *ModelCatalogWatchService {
	if strings.TrimSpace(filePath) == "" {
		dataDir, err := config.DataDir()
		if err != nil || strings.TrimSpace(dataDir) == "" {
			dataDir = filepath.Join(os.TempDir(), "iapro-nexus")
		}
		filePath = filepath.Join(dataDir, "nex-model-catalog-watch.json")
	}
	svc := &ModelCatalogWatchService{
		filePath: filePath,
		entries:  make(map[string]ModelWatchEntry),
		defaults: make(map[string]string),
		changes:  make([]ModelCatalogChange, 0),
	}
	svc.loadLocked()
	return svc
}

func (s *ModelCatalogWatchService) loadLocked() {
	raw, err := os.ReadFile(s.filePath)
	if err != nil {
		return
	}
	var st modelWatchPersistedState
	if err := json.Unmarshal(raw, &st); err != nil {
		return
	}
	s.hasObserved = st.HasObserved
	s.observedAt = st.ObservedAt
	for _, e := range st.Entries {
		key := modelEntryKey(e.Provider, e.Profile, e.ModelID)
		s.entries[key] = e
		if e.IsDefault {
			s.defaults[modelWatchAccountKey(e.Provider, e.Profile)] = e.ModelID
		}
	}
	if len(st.Changes) > 0 {
		s.changes = append(s.changes, st.Changes...)
		if len(s.changes) > modelWatchMaxChanges {
			s.changes = append([]ModelCatalogChange{}, s.changes[len(s.changes)-modelWatchMaxChanges:]...)
		}
	}
}

func (s *ModelCatalogWatchService) saveLocked() {
	if strings.TrimSpace(s.filePath) == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return
	}
	entries := s.sortedEntriesLocked()
	st := modelWatchPersistedState{
		HasObserved: s.hasObserved,
		ObservedAt:  s.observedAt,
		Entries:     entries,
		Changes:     s.changes,
	}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, s.filePath)
}

func (s *ModelCatalogWatchService) sortedEntriesLocked() []ModelWatchEntry {
	out := make([]ModelWatchEntry, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		if out[i].Profile != out[j].Profile {
			return out[i].Profile < out[j].Profile
		}
		return out[i].ModelID < out[j].ModelID
	})
	return out
}

func modelEntryKey(provider, profile, modelID string) string {
	return strings.TrimSpace(provider) + "\x00" + strings.TrimSpace(profile) + "\x00" + strings.TrimSpace(modelID)
}

func modelWatchAccountKey(provider, profile string) string {
	return strings.TrimSpace(provider) + "\x00" + strings.TrimSpace(profile)
}

func modelChangeFingerprint(kind, provider, profile, modelID, prevDef, newDef, source string) string {
	h := sha256.Sum256([]byte(strings.Join([]string{kind, provider, profile, modelID, prevDef, newDef, source}, "|")))
	return hex.EncodeToString(h[:8])
}

// ObserveEntries diffs newly observed model entries against previously known
// entries per (provider, profile), emitting deterministic changes without
// duplicating unchanged observations.
func (s *ModelCatalogWatchService) ObserveEntries(entries []ModelWatchEntry, now time.Time) []ModelCatalogChange {
	s.mu.Lock()
	defer s.mu.Unlock()

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	if !s.hasObserved && len(entries) == 0 {
		return nil
	}

	nextEntries := make(map[string]ModelWatchEntry, len(entries))
	nextDefaults := make(map[string]string)
	for _, e := range entries {
		e.Provider = strings.TrimSpace(e.Provider)
		e.Profile = strings.TrimSpace(e.Profile)
		e.ModelID = strings.TrimSpace(e.ModelID)
		e.Source = strings.TrimSpace(e.Source)
		if e.Provider == "" || e.ModelID == "" {
			continue
		}
		if e.ObservedAt.IsZero() {
			e.ObservedAt = now
		}
		k := modelEntryKey(e.Provider, e.Profile, e.ModelID)
		nextEntries[k] = e
		if e.IsDefault {
			nextDefaults[modelWatchAccountKey(e.Provider, e.Profile)] = e.ModelID
		}
	}

	emitted := make([]ModelCatalogChange, 0)
	for k, next := range nextEntries {
		prev, existed := s.entries[k]
		if !existed {
			kind := ModelChangeAdded
			if next.Source == "" || strings.EqualFold(next.Source, ModelWatchFreshnessUnknown) {
				kind = ModelChangeUnknownSource
			}
			emitted = append(emitted, ModelCatalogChange{
				Fingerprint: modelChangeFingerprint(kind, next.Provider, next.Profile, next.ModelID, "", "", next.Source),
				Kind:        kind,
				Provider:    next.Provider,
				Profile:     next.Profile,
				ModelID:     next.ModelID,
				Source:      next.Source,
				ObservedAt:  now,
			})
			continue
		}
		if !prev.Deprecated && next.Deprecated {
			emitted = append(emitted, ModelCatalogChange{
				Fingerprint: modelChangeFingerprint(ModelChangeDeprecated, next.Provider, next.Profile, next.ModelID, "", "", next.Source),
				Kind:        ModelChangeDeprecated,
				Provider:    next.Provider,
				Profile:     next.Profile,
				ModelID:     next.ModelID,
				Source:      next.Source,
				ObservedAt:  now,
			})
		}
	}

	for k, prev := range s.entries {
		if _, stillPresent := nextEntries[k]; !stillPresent {
			emitted = append(emitted, ModelCatalogChange{
				Fingerprint: modelChangeFingerprint(ModelChangeRemoved, prev.Provider, prev.Profile, prev.ModelID, "", "", prev.Source),
				Kind:        ModelChangeRemoved,
				Provider:    prev.Provider,
				Profile:     prev.Profile,
				ModelID:     prev.ModelID,
				Source:      prev.Source,
				ObservedAt:  now,
			})
		}
	}

	for accKey, newDef := range nextDefaults {
		prevDef, hadPrev := s.defaults[accKey]
		if hadPrev && prevDef != newDef {
			parts := strings.SplitN(accKey, "\x00", 2)
			prov, prof := parts[0], ""
			if len(parts) > 1 {
				prof = parts[1]
			}
			emitted = append(emitted, ModelCatalogChange{
				Fingerprint:     modelChangeFingerprint(ModelChangeDefaultChanged, prov, prof, newDef, prevDef, newDef, "hud_projection"),
				Kind:            ModelChangeDefaultChanged,
				Provider:        prov,
				Profile:         prof,
				ModelID:         newDef,
				PreviousDefault: prevDef,
				NewDefault:      newDef,
				Source:          "hud_projection",
				ObservedAt:      now,
			})
		}
	}

	sort.Slice(emitted, func(i, j int) bool {
		return emitted[i].Fingerprint < emitted[j].Fingerprint
	})

	s.hasObserved = true
	s.observedAt = now
	s.entries = nextEntries
	s.defaults = nextDefaults
	if len(emitted) > 0 {
		s.changes = append(s.changes, emitted...)
		if len(s.changes) > modelWatchMaxChanges {
			s.changes = append([]ModelCatalogChange{}, s.changes[len(s.changes)-modelWatchMaxChanges:]...)
		}
	}
	s.saveLocked()
	return emitted
}

// ObserveHUDState extracts already-observed model identities from canonical HUD
// projection snapshots without invoking any external CLI or API probes.
func (s *ModelCatalogWatchService) ObserveHUDState(state HUDState, now time.Time) []ModelCatalogChange {
	seen := make(map[string]ModelWatchEntry)
	addEntry := func(provider, profile, modelID, source string, isDefault bool) {
		provider = strings.TrimSpace(provider)
		profile = strings.TrimSpace(profile)
		modelID = strings.TrimSpace(modelID)
		if provider == "" || modelID == "" {
			return
		}
		k := modelEntryKey(provider, profile, modelID)
		existing, ok := seen[k]
		if ok {
			if isDefault {
				existing.IsDefault = true
			}
			seen[k] = existing
			return
		}
		seen[k] = ModelWatchEntry{
			Provider:   provider,
			Profile:    profile,
			ModelID:    modelID,
			IsDefault:  isDefault,
			Source:     source,
			ObservedAt: now,
		}
	}

	for _, p := range state.Providers {
		src := p.Source
		if src == "" {
			src = p.Freshness
		}
		if p.LimitingWindow != nil && strings.TrimSpace(p.LimitingWindow.Group) != "" {
			addEntry(p.Provider, p.Profile, p.LimitingWindow.Group, src, true)
		}
		for _, w := range p.Windows {
			if strings.TrimSpace(w.Group) != "" {
				addEntry(p.Provider, p.Profile, w.Group, src, false)
			}
		}
	}
	for _, a := range state.Agents {
		if strings.TrimSpace(a.Model) != "" {
			addEntry(a.Provider, a.Profile, a.Model, "agent_state", true)
		}
	}

	if len(seen) == 0 {
		return nil
	}
	entries := make([]ModelWatchEntry, 0, len(seen))
	for _, e := range seen {
		entries = append(entries, e)
	}
	return s.ObserveEntries(entries, now)
}

// Snapshot returns the current read-only model catalog watch state with honest
// freshness classification (UNKNOWN / STALE / FRESH) and non-nil slices.
func (s *ModelCatalogWatchService) Snapshot(now time.Time) ModelCatalogWatchSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}

	freshness := ModelWatchFreshnessUnknown
	var observedPtr *time.Time
	if s.hasObserved && !s.observedAt.IsZero() {
		t := s.observedAt.UTC()
		observedPtr = &t
		if now.Sub(t) > modelWatchStaleAfter {
			freshness = ModelWatchFreshnessStale
		} else {
			freshness = ModelWatchFreshnessFresh
		}
	}

	changes := make([]ModelCatalogChange, len(s.changes))
	copy(changes, s.changes)

	return ModelCatalogWatchSnapshot{
		Freshness:  freshness,
		ObservedAt: observedPtr,
		Entries:    s.sortedEntriesLocked(),
		Changes:    changes,
	}
}
