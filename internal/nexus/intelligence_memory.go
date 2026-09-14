package nexus

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/ids"
	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const (
	MemoryScopeInstall = "INSTALL"
	MemoryScopeProject = "PROJECT"

	MemorySourceComposer    = "composer"
	MemorySourceDestination = "destination"
	MemorySourceExplicit    = "explicit"
)

// Habit keys that may be promoted to durable memory. Feature-scoped fields
// (acceptance for this ticket, in-scope list) must never become INSTALL habits.
var habitMemoryKeys = map[string]bool{
	"quality.verification":    true,
	"quality.testing":         true,
	"environment.stack":       true,
	"execution.autonomy":      true,
	"composer.preferred_dest": true,
	"composer.prompt_locale":  true,
	"verification":            true,
	"testing":                 true,
}

var secretLike = regexp.MustCompile(`(?i)(api[_-]?key|token|password|secret|credential|bearer\s+[a-z0-9._-]+|/home/[^/\s]+/\.ssh)`)

// MergedIntelligenceMemory is the transport-facing memory row.
type MergedIntelligenceMemory struct {
	ID        string `json:"id"`
	Scope     string `json:"scope"`
	ProjectID string `json:"project_id,omitempty"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Source    string `json:"source,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ListIntelligenceMemories returns install + project memories for Settings/Composer chips.
func (n *Nexus) ListIntelligenceMemories(ctx context.Context, projectID string) ([]MergedIntelligenceMemory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := []MergedIntelligenceMemory{}
	cfg, err := coreconfig.LoadConfig()
	if err != nil {
		return nil, err
	}
	for _, m := range cfg.IntelligenceMemories {
		out = append(out, MergedIntelligenceMemory{
			ID: m.ID, Scope: MemoryScopeInstall, Key: m.Key, Value: m.Value,
			Source: m.Source, UpdatedAt: m.UpdatedAt,
		})
	}
	if strings.TrimSpace(projectID) == "" {
		return out, nil
	}
	st, err := n.OpenProject()
	if err != nil {
		return out, nil
	}
	projectRows, err := st.ListIntelligenceMemories(projectID)
	if err != nil {
		return nil, err
	}
	for _, m := range projectRows {
		out = append(out, MergedIntelligenceMemory{
			ID: m.ID, Scope: MemoryScopeProject, ProjectID: m.ProjectID,
			Key: m.Key, Value: m.Value, Source: m.Source,
			UpdatedAt: m.UpdatedAt.Format(time.RFC3339Nano),
		})
	}
	return out, nil
}

// UpsertIntelligenceMemory writes a habit memory. Scope INSTALL goes to config.json;
// PROJECT goes to the project SQLite store.
func (n *Nexus) UpsertIntelligenceMemory(ctx context.Context, scope, projectID, key, value, source string) (*MergedIntelligenceMemory, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key = normalizeMemoryKey(key)
	value = strings.TrimSpace(value)
	if key == "" || value == "" {
		return nil, fmt.Errorf("memory key and value are required")
	}
	if !IsHabitMemoryKey(key) {
		return nil, fmt.Errorf("memory key %q is not a promotable habit", key)
	}
	if LooksLikeSecret(value) {
		return nil, fmt.Errorf("refusing to store secret-like memory value")
	}
	if source == "" {
		source = MemorySourceExplicit
	}
	switch strings.ToUpper(scope) {
	case MemoryScopeInstall, "":
		return n.upsertInstallMemory(key, value, source)
	case MemoryScopeProject:
		if projectID == "" {
			return nil, fmt.Errorf("project_id is required for PROJECT memories")
		}
		return n.upsertProjectMemory(projectID, key, value, source)
	default:
		return nil, fmt.Errorf("unknown memory scope %q", scope)
	}
}

// DeleteIntelligenceMemory removes one memory by scope + id.
func (n *Nexus) DeleteIntelligenceMemory(ctx context.Context, scope, projectID, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch strings.ToUpper(scope) {
	case MemoryScopeInstall, "":
		cfg, err := coreconfig.LoadConfig()
		if err != nil {
			return err
		}
		next := make([]coreconfig.IntelligenceMemory, 0, len(cfg.IntelligenceMemories))
		found := false
		for _, m := range cfg.IntelligenceMemories {
			if m.ID == id {
				found = true
				continue
			}
			next = append(next, m)
		}
		if !found {
			return fmt.Errorf("memory not found")
		}
		cfg.IntelligenceMemories = next
		return coreconfig.SaveConfig(cfg)
	case MemoryScopeProject:
		st, err := n.OpenProject()
		if err != nil {
			return err
		}
		return st.DeleteIntelligenceMemory(projectID, id)
	default:
		return fmt.Errorf("unknown memory scope %q", scope)
	}
}

// MemoryFactsMap flattens install+project memories for Intelligence contextData
// and StructuredFacts. Project wins over install on the same key.
func (n *Nexus) MemoryFactsMap(ctx context.Context, projectID string) (map[string]string, error) {
	rows, err := n.ListIntelligenceMemories(ctx, projectID)
	if err != nil {
		return nil, err
	}
	facts := map[string]string{}
	for _, m := range rows {
		if m.Scope == MemoryScopeInstall {
			facts["memory.install."+m.Key] = m.Value
			if _, exists := facts[m.Key]; !exists {
				facts[m.Key] = m.Value
			}
		}
	}
	for _, m := range rows {
		if m.Scope == MemoryScopeProject {
			facts["memory.project."+m.Key] = m.Value
			facts[m.Key] = m.Value
		}
	}
	return facts, nil
}

// PromoteHabitMemory records a habit answer. After two identical values (or
// when forced), the memory becomes durable. Feature-scoped keys are ignored.
func (n *Nexus) PromoteHabitMemory(ctx context.Context, projectID, key, value, source string, force bool) error {
	key = normalizeMemoryKey(key)
	value = strings.TrimSpace(value)
	if !IsHabitMemoryKey(key) || value == "" || LooksLikeSecret(value) {
		return nil
	}
	if source == "" {
		source = MemorySourceComposer
	}
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	existing, err := st.GetIntelligenceMemoryByKey(projectID, key)
	if err == nil && existing != nil {
		if existing.Value == value || force {
			_, err := n.UpsertIntelligenceMemory(ctx, MemoryScopeProject, projectID, key, value, source)
			return err
		}
	}
	// First observation: store as project memory immediately when forced, else
	// store a pending marker so the second identical answer promotes.
	pendingKey := "pending." + key
	pending, _ := st.GetIntelligenceMemoryByKey(projectID, pendingKey)
	if force {
		_, err := n.UpsertIntelligenceMemory(ctx, MemoryScopeProject, projectID, key, value, source)
		_ = st.DeleteIntelligenceMemoryByKey(projectID, pendingKey)
		return err
	}
	if pending != nil && pending.Value == value {
		_, err := n.UpsertIntelligenceMemory(ctx, MemoryScopeProject, projectID, key, value, source)
		_ = st.DeleteIntelligenceMemoryByKey(projectID, pendingKey)
		return err
	}
	_, err = st.UpsertIntelligenceMemory(store.IntelligenceMemory{
		ProjectID: projectID, Key: pendingKey, Value: value, Source: source,
	})
	return err
}

func (n *Nexus) upsertInstallMemory(key, value, source string) (*MergedIntelligenceMemory, error) {
	cfg, err := coreconfig.LoadConfig()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	id := ""
	next := make([]coreconfig.IntelligenceMemory, 0, len(cfg.IntelligenceMemories)+1)
	replaced := false
	for _, m := range cfg.IntelligenceMemories {
		if m.Key == key {
			m.Value = value
			m.Source = source
			m.UpdatedAt = now
			if m.ID == "" {
				m.ID = "imem_" + ids.NewRuntimeID()
			}
			id = m.ID
			next = append(next, m)
			replaced = true
			continue
		}
		next = append(next, m)
	}
	if !replaced {
		id = "imem_" + ids.NewRuntimeID()
		next = append(next, coreconfig.IntelligenceMemory{
			ID: id, Key: key, Value: value, Source: source, UpdatedAt: now,
		})
	}
	cfg.IntelligenceMemories = next
	if err := coreconfig.SaveConfig(cfg); err != nil {
		return nil, err
	}
	return &MergedIntelligenceMemory{
		ID: id, Scope: MemoryScopeInstall, Key: key, Value: value, Source: source, UpdatedAt: now,
	}, nil
}

func (n *Nexus) upsertProjectMemory(projectID, key, value, source string) (*MergedIntelligenceMemory, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	row, err := st.UpsertIntelligenceMemory(store.IntelligenceMemory{
		ProjectID: projectID, Key: key, Value: value, Source: source,
	})
	if err != nil {
		return nil, err
	}
	return &MergedIntelligenceMemory{
		ID: row.ID, Scope: MemoryScopeProject, ProjectID: row.ProjectID,
		Key: row.Key, Value: row.Value, Source: row.Source,
		UpdatedAt: row.UpdatedAt.Format(time.RFC3339Nano),
	}, nil
}

func IsHabitMemoryKey(key string) bool {
	key = normalizeMemoryKey(key)
	if habitMemoryKeys[key] {
		return true
	}
	// Allow nested habit prefixes used by brief fields.
	for allowed := range habitMemoryKeys {
		if strings.HasPrefix(key, allowed+".") || strings.HasSuffix(key, "."+allowed) {
			return true
		}
	}
	return false
}

func LooksLikeSecret(value string) bool {
	return secretLike.MatchString(value)
}

func normalizeMemoryKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

// ApplyMemoriesToBrief marks habit fields covered by memory as inferred so the
// Composer refinery does not re-ask them.
func ApplyMemoriesToBrief(brief *LivingBrief, facts map[string]string) {
	if brief == nil || len(facts) == 0 {
		return
	}
	if v := firstMemoryValue(facts, "quality.verification", "verification"); v != "" {
		brief.Quality.Verification = appendUnique(brief.Quality.Verification, v)
	}
	if v := firstMemoryValue(facts, "quality.testing", "testing"); v != "" {
		brief.Quality.Testing = appendUnique(brief.Quality.Testing, v)
	}
	if v := firstMemoryValue(facts, "environment.stack"); v != "" {
		brief.Environment.Stack = appendUnique(brief.Environment.Stack, v)
	}
	if v := firstMemoryValue(facts, "execution.autonomy"); v != "" {
		brief.Execution.Autonomy = firstNonEmpty(brief.Execution.Autonomy, v)
	}
}

func firstMemoryValue(facts map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(facts[key]); v != "" {
			return v
		}
		if v := strings.TrimSpace(facts["memory.project."+key]); v != "" {
			return v
		}
		if v := strings.TrimSpace(facts["memory.install."+key]); v != "" {
			return v
		}
	}
	return ""
}
