package skills

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxSkillDiscoveryDepth      = 6
	maxSkillDiscoveryCandidates = 256
	maxSkillDiscoveryBytes      = 4 * 1024 * 1024
)

type builtinSource struct{}

func NewBuiltinSource() Source { return builtinSource{} }

func (builtinSource) ID() string { return string(SourceBuiltin) }

func (builtinSource) Available(context.Context) bool { return true }

func (builtinSource) Discover(context.Context) ([]Skill, error) {
	definitions := []struct {
		id, description, capability string
	}{
		{"coding", "Implement focused changes with tests and evidence.", "coding"},
		{"debugging", "Reproduce, trace and fix root causes systematically.", "debugging"},
		{"testing", "Design and run deterministic tests before claiming behavior.", "testing"},
		{"review", "Review changes for correctness, regressions and maintainability.", "review"},
		{"verification", "Verify acceptance criteria with reproducible evidence before completion.", "verification"},
		{"frontend", "Work within existing accessible frontend patterns and tokens.", "frontend"},
		{"backend", "Work within existing backend boundaries and typed contracts.", "backend"},
		{"security", "Apply defensive security analysis and fail-closed boundaries.", "security"},
		{"devops", "Inspect build, CI and release paths with reproducible evidence.", "devops"},
		{"architecture", "Preserve ownership boundaries and choose the smallest safe change.", "architecture"},
		{"refactoring", "Refactor only after behavior is protected by tests.", "refactoring"},
		{"delegation", "Split independent work only after contracts are frozen.", "delegation"},
		{"agent-selection", "Select or create a persistent Agent from task requirements.", "agent-selection"},
		{"resource-selection", "Resolve provider, profile and account resources under policy.", "resource-selection"},
		{"model-selection", "Choose a task-proportional model with evidence-backed escalation.", "model-selection"},
		{"context-handoff", "Carry bounded decisions, failures and verification evidence forward.", "context-handoff"},
	}
	items := make([]Skill, 0, len(definitions))
	for _, definition := range definitions {
		items = append(items, Skill{
			ID: definition.id, Name: definition.id, Description: definition.description,
			Source: SourceBuiltin, Version: "1.0.0", Capabilities: []string{definition.capability},
			Triggers: []string{definition.capability}, Risk: "low", Instructions: definition.description,
			Provenance: Provenance{SourceID: string(SourceBuiltin), Path: "builtin:" + definition.id, ObservedAt: time.Unix(0, 0).UTC()},
		})
	}
	return items, nil
}

func (s builtinSource) Resolve(ctx context.Context, id string) (Skill, bool, error) {
	items, err := s.Discover(ctx)
	if err != nil {
		return Skill{}, false, err
	}
	for _, item := range items {
		if item.ID == strings.TrimSpace(id) {
			return item, true, nil
		}
	}
	return Skill{}, false, nil
}

type directorySource struct {
	id    string
	kind  SourceID
	roots []string
}

func NewDirectorySource(id string, kind SourceID, roots []string) Source {
	return directorySource{id: strings.TrimSpace(id), kind: kind, roots: append([]string(nil), roots...)}
}

func (s directorySource) ID() string { return s.id }

func (s directorySource) Available(context.Context) bool { return len(s.roots) > 0 }

func (s directorySource) Discover(ctx context.Context) ([]Skill, error) {
	items := []Skill{}
	var totalBytes int64
	candidates := 0
	for _, root := range s.roots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		info, err := os.Lstat(root)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, resolveErr := filepath.EvalSymlinks(root)
			if resolveErr != nil {
				// A broken or cyclic optional source must not take down the
				// built-in catalog. The source remains unavailable for this scan.
				continue
			}
			root = resolved
			info, err = os.Stat(root)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return nil, err
			}
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				depth := strings.Count(filepath.ToSlash(strings.TrimPrefix(path, root)), "/")
				if depth > maxSkillDiscoveryDepth {
					return filepath.SkipDir
				}
				if path != root && skillContainsAny(entry.Name(), ".git", "node_modules", "vendor", "dist", "build", ".cache") {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Name() != "SKILL.md" {
				return nil
			}
			candidates++
			if candidates > maxSkillDiscoveryCandidates {
				return filepath.SkipAll
			}
			info, statErr := entry.Info()
			if statErr != nil || info.Size() > 256*1024 || totalBytes+info.Size() > maxSkillDiscoveryBytes {
				return nil
			}
			item, parseErr := parseSkillMarkdown(path, s.kind)
			if parseErr != nil {
				return nil
			}
			totalBytes += info.Size()
			items = append(items, item)
			return nil
		})
		if err != nil && err != filepath.SkipAll {
			return nil, err
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func (s directorySource) Resolve(ctx context.Context, id string) (Skill, bool, error) {
	items, err := s.Discover(ctx)
	if err != nil {
		return Skill{}, false, err
	}
	for _, item := range items {
		if item.ID == strings.TrimSpace(id) {
			return item, true, nil
		}
	}
	return Skill{}, false, nil
}

type externalSource struct {
	id       string
	discover func(context.Context) ([]Skill, error)
}

func NewExternalSource(id string, discover func(context.Context) ([]Skill, error)) Source {
	return externalSource{id: id, discover: discover}
}

func (s externalSource) ID() string { return s.id }

func (s externalSource) Available(context.Context) bool { return s.discover != nil }

func (s externalSource) Discover(ctx context.Context) ([]Skill, error) {
	if s.discover == nil {
		return nil, ErrSourceUnavailable
	}
	return s.discover(ctx)
}

func (s externalSource) Resolve(ctx context.Context, id string) (Skill, bool, error) {
	items, err := s.Discover(ctx)
	if err != nil {
		return Skill{}, false, err
	}
	for _, item := range items {
		if item.ID == strings.TrimSpace(id) {
			return item, true, nil
		}
	}
	return Skill{}, false, nil
}

func parseSkillMarkdown(path string, source SourceID) (Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, err
	}
	if len(data) > 256*1024 {
		return Skill{}, fmt.Errorf("skill %s exceeds size limit", path)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	frontmatter, body := splitFrontmatter(text)
	fields := parseFrontmatter(frontmatter)
	id := fields["id"]
	if id == "" {
		id = filepath.Base(filepath.Dir(path))
	}
	item := Skill{
		ID: id, Name: firstNonEmpty(fields["name"], id), Description: fields["description"], Source: source,
		Version: firstNonEmpty(fields["version"], "1.0.0"), Risk: fields["risk"],
		Capabilities: splitList(fields["capabilities"]), Triggers: splitList(fields["triggers"]),
		Dependencies: splitList(fields["dependencies"]), Permissions: splitList(fields["permissions"]),
		Instructions: strings.TrimSpace(body), Availability: AvailabilityAvailable, Mode: "task",
		Provenance: Provenance{SourceID: string(source), Path: filepath.ToSlash(path), ObservedAt: time.Now().UTC()},
	}
	return normalizeSkill(item, string(source))
}

func splitFrontmatter(text string) (string, string) {
	if !strings.HasPrefix(text, "---\n") {
		return "", text
	}
	remaining := strings.TrimPrefix(text, "---\n")
	index := strings.Index(remaining, "\n---\n")
	if index < 0 {
		return "", text
	}
	return remaining[:index], remaining[index+len("\n---\n"):]
}

func parseFrontmatter(text string) map[string]string {
	fields := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), "\"'")
	}
	return fields
}

func splitList(value string) []string {
	value = strings.TrimSpace(strings.Trim(value, "[]"))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), "\"'")
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func skillContainsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}
