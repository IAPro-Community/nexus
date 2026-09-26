package nexus

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

// MaestroVersion is the contract version for Maestro integration.
const MaestroVersion = "1.0.0"

// MaestroMode represents the Maestro integration mode.
type MaestroMode string

const (
	MaestroOff         MaestroMode = "OFF"
	MaestroAssist      MaestroMode = "ASSIST"
	MaestroOrchestrate MaestroMode = "ORCHESTRATE"
)

// MaestroLifecycle identifies the phase for which optional Maestro guidance
// is requested. Nexus owns execution and resource allocation; Maestro only
// advises the phase-specific process.
type MaestroLifecycle string

const (
	MaestroLifecyclePlan     MaestroLifecycle = "PLAN"
	MaestroLifecycleTask     MaestroLifecycle = "TASK"
	MaestroLifecycleRecovery MaestroLifecycle = "RECOVERY"
	MaestroLifecycleVerify   MaestroLifecycle = "VERIFY"
)

// MaestroSkillDesc describes a single canonical skill provided by Maestro.
type MaestroSkillDesc struct {
	ID          string   `json:"id"`
	Version     string   `json:"version,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category,omitempty"`
	Risk        string   `json:"risk,omitempty"`
	Triggers    []string `json:"triggers,omitempty"`
	Aliases     []string `json:"aliases,omitempty"`
	Prompt      string   `json:"prompt,omitempty"`
}

// SkillSource identifies where a skill was discovered. It is deliberately
// part of the API contract: discovery is not the same thing as installation.
type SkillSource string

const (
	SkillSourceCanonical SkillSource = "canonical"
	SkillSourceCommunity SkillSource = "community"
	SkillSourceCodex     SkillSource = "codex"
)

type SkillAvailability string

const (
	SkillAvailable      SkillAvailability = "AVAILABLE"
	SkillSynchronizable SkillAvailability = "SYNCHRONIZABLE"
	SkillTaskOnly       SkillAvailability = "TASK_ONLY"
)

// CatalogSkill is the honest, UI-facing skill contract. Contract contains the
// complete SKILL.md content when it is safe to apply it to one task.
type CatalogSkill struct {
	MaestroSkillDesc
	Source       SkillSource       `json:"source"`
	Availability SkillAvailability `json:"availability"`
	Mode         string            `json:"activation_mode"`
	Root         string            `json:"root,omitempty"`
	Copies       int               `json:"copies"`
	Contract     string            `json:"contract,omitempty"`
}

type SkillCatalog struct {
	Operational []CatalogSkill `json:"operational"`
	Library     []CatalogSkill `json:"library"`
	Counts      struct {
		Operational int `json:"operational"`
		Library     int `json:"library"`
		Copies      int `json:"copies"`
	} `json:"counts"`
}

type SkillSyncPreview struct {
	ID      string   `json:"id"`
	DryRun  bool     `json:"dry_run"`
	Tool    string   `json:"tool"`
	Roots   []string `json:"roots"`
	Skills  []string `json:"skills"`
	Command string   `json:"command"`
	Output  string   `json:"output,omitempty"`
}

const maxSkillSyncOutput = 1 << 20

type boundedOutput struct {
	buf bytes.Buffer
	max int
}

// Write returns error when the accumulated output exceeds the configured
// limit; exec.Cmd.Run() failure is expected in this case and the caller
// should inspect the bounded buffer for whatever partial output succeeded.
func (b *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > b.max-b.buf.Len() {
		return 0, fmt.Errorf("sync output exceeds %d bytes", b.max)
	}
	return b.buf.Write(p)
}

// catalogCache provides a short-lived in-memory cache for the skill catalog.
// SKILL.md files are expensive to read on every call; 30 s TTL is enough to
// avoid redundant I/O within a single interactive session while remaining
// fresh enough for edits made outside the process.
type catalogCache struct {
	mu        sync.RWMutex
	catalog   *SkillCatalog
	expiresAt time.Time
}

const catalogCacheTTL = 30 * time.Second

var globalCatalogCache catalogCache

func (c *MaestroClient) ApplySyncPreview(ctx context.Context, preview SkillSyncPreview) (*SkillSyncPreview, error) {
	current, err := c.SyncPreview()
	if err != nil {
		return nil, err
	}
	if preview.ID == "" || preview.ID != current.ID || filepath.Clean(preview.Tool) != filepath.Clean(current.Tool) {
		return nil, fmt.Errorf("sync preview is stale or not an official script")
	}
	base := strings.ToLower(filepath.Base(current.Tool))
	if base != "sync-skills.sh" && base != "sync-skills.ps1" {
		return nil, fmt.Errorf("sync tool is not executable on this platform")
	}
	// Allowlist: the tool must reside under a known .orquestrador root so that
	// an attacker who controls an arbitrary file on PATH cannot trick Nexus
	// into executing it with --apply.
	allowed := false
	toolAbs, _ := filepath.Abs(current.Tool)
	for _, dir := range findOrquestradorDirs() {
		if strings.HasPrefix(toolAbs, filepath.Clean(dir)+string(filepath.Separator)) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("sync tool path %q is outside known orquestrador roots", current.Tool)
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd, err := syncScriptCommand(ctx, current.Tool, "--apply")
	if err != nil {
		return nil, err
	}
	var output boundedOutput
	output.max = maxSkillSyncOutput
	cmd.Stdout = &output
	cmd.Stderr = &output // capture stderr inside the bounded buffer
	err = cmd.Run()
	result := *current
	result.DryRun = false
	result.Output = output.buf.String()
	// Invalidate the catalog cache so the next Catalog() call picks up newly
	// synced skills.
	globalCatalogCache.mu.Lock()
	globalCatalogCache.catalog = nil
	globalCatalogCache.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("official skill sync failed: %w", err)
	}
	return &result, nil
}

func syncScriptInvocation(tool, action string, lookPath func(string) (string, error)) (string, []string, error) {
	tool = filepath.Clean(strings.TrimSpace(tool))
	action = strings.TrimSpace(action)
	if tool == "." || action == "" {
		return "", nil, fmt.Errorf("sync script and action are required")
	}
	if lookPath == nil {
		return "", nil, fmt.Errorf("sync script executable resolver is required")
	}

	base := strings.ToLower(filepath.Base(tool))
	switch {
	case strings.HasSuffix(base, ".sh"):
		bash, err := lookPath("bash")
		if err != nil {
			return "", nil, fmt.Errorf("bash is required to execute %s: %w", base, err)
		}
		return bash, []string{tool, action}, nil
	case strings.HasSuffix(base, ".ps1"):
		for _, candidate := range []string{"pwsh", "powershell.exe", "powershell"} {
			shell, err := lookPath(candidate)
			if err != nil {
				continue
			}
			args := []string{"-NoProfile", "-NonInteractive"}
			if strings.HasPrefix(strings.ToLower(filepath.Base(shell)), "powershell") {
				args = append(args, "-ExecutionPolicy", "Bypass")
			}
			args = append(args, "-File", tool, action)
			return shell, args, nil
		}
		return "", nil, fmt.Errorf("PowerShell is required to execute %s", base)
	default:
		return "", nil, fmt.Errorf("unsupported Maestro sync script %q", base)
	}
}

func syncScriptCommand(ctx context.Context, tool, action string) (*exec.Cmd, error) {
	binary, args, err := syncScriptInvocation(tool, action, exec.LookPath)
	if err != nil {
		return nil, err
	}
	return exec.CommandContext(ctx, binary, args...), nil
}

func (c *MaestroClient) SyncPreview() (*SkillSyncPreview, error) {
	dirs := findOrquestradorDirs()
	if len(dirs) == 0 {
		return nil, fmt.Errorf("maestro library root not found")
	}
	root := filepath.Clean(dirs[0])
	for _, candidate := range []string{filepath.Join(root, "sync-skills.sh"), filepath.Join(root, "sync-skills.ps1")} {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		base := filepath.Base(candidate)
		if base != "sync-skills.sh" && base != "sync-skills.ps1" {
			continue
		}
		catalog := c.Catalog()
		skills := make([]string, 0, len(catalog.Library))
		for _, skill := range catalog.Library {
			if skill.Availability == SkillSynchronizable {
				skills = append(skills, skill.ID)
			}
		}
		preview := &SkillSyncPreview{ID: fmt.Sprintf("%x", sha256.Sum256([]byte(candidate+fmt.Sprintf("%d", info.ModTime().Unix())+strings.Join(skills, "\n")))), DryRun: true, Tool: candidate, Roots: dirs, Skills: skills, Command: base + " --dry-run"}
		var output boundedOutput
		output.max = maxSkillSyncOutput
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd, cmdErr := syncScriptCommand(ctx, candidate, "--dry-run")
		if cmdErr != nil {
			cancel()
			return nil, cmdErr
		}
		cmd.Stdout = &output
		cmd.Stderr = &output
		err = cmd.Run()
		cancel()
		if err != nil {
			return nil, fmt.Errorf("official skill sync dry-run failed: %w", err)
		}
		preview.Output = output.buf.String()
		return preview, nil
	}
	return nil, fmt.Errorf("official sync-skills script not found")
}

// MergeSkillCatalog deduplicates skills by stable ID while retaining the
// strongest usable source and the number of discovered copies.
func MergeSkillCatalog(groups ...[]CatalogSkill) SkillCatalog {
	merged := make(map[string]CatalogSkill)
	copies := make(map[string]int)
	priority := map[SkillAvailability]int{SkillAvailable: 3, SkillSynchronizable: 2, SkillTaskOnly: 1}
	for _, group := range groups {
		for _, skill := range group {
			id := strings.TrimSpace(skill.ID)
			if id == "" {
				continue
			}
			copies[id]++
			if old, ok := merged[id]; !ok || priority[skill.Availability] > priority[old.Availability] {
				merged[id] = skill
			}
		}
	}
	result := SkillCatalog{Operational: []CatalogSkill{}, Library: []CatalogSkill{}}
	for id, skill := range merged {
		skill.Copies = copies[id]
		result.Library = append(result.Library, skill)
		if skill.Availability == SkillAvailable {
			result.Operational = append(result.Operational, skill)
		}
	}
	sort.Slice(result.Library, func(i, j int) bool { return result.Library[i].ID < result.Library[j].ID })
	sort.Slice(result.Operational, func(i, j int) bool { return result.Operational[i].ID < result.Operational[j].ID })
	result.Counts.Operational = len(result.Operational)
	result.Counts.Library = len(result.Library)
	for _, skill := range result.Library {
		result.Counts.Copies += skill.Copies
	}
	return result
}

// MaestroCapability describes what the Maestro instance supports.
type MaestroCapability struct {
	Version   string             `json:"version"`
	Modes     []string           `json:"modes"`     // supported modes
	Skills    []MaestroSkillDesc `json:"skills"`    // available skills
	Gates     []string           `json:"gates"`     // available gate types
	Processes []string           `json:"processes"` // available process types
}

func (mc *MaestroCapability) SkillIDs() []string {
	if mc == nil {
		return nil
	}
	ids := make([]string, len(mc.Skills))
	for i, s := range mc.Skills {
		ids[i] = s.ID
	}
	return ids
}

// AdviceRequest is the structured request sent to Maestro for recommendations.
type AdviceRequest struct {
	Version   string           `json:"version"`
	Context   AdviceContext    `json:"context"`
	Intent    string           `json:"intent"`
	Scope     string           `json:"scope"` // "project" | "agent" | "task"
	Lifecycle MaestroLifecycle `json:"lifecycle"`
	Extra     map[string]any   `json:"extra,omitempty"`
}

// AdviceContext provides project/agent context for Maestro decisions.
type AdviceContext struct {
	ProjectID   string           `json:"project_id"`
	AgentID     string           `json:"agent_id,omitempty"`
	AgentStatus string           `json:"agent_status,omitempty"`
	Provider    string           `json:"provider,omitempty"`
	Profile     string           `json:"profile,omitempty"`
	Lifecycle   MaestroLifecycle `json:"lifecycle,omitempty"`
}

// AdviceResponse is the structured response from Maestro.
type AdviceResponse struct {
	Version     string           `json:"version"`
	Mode        MaestroMode      `json:"mode"`
	Lifecycle   MaestroLifecycle `json:"lifecycle"`
	Required    []Recommendation `json:"required"`
	Recommended []Recommendation `json:"recommended"`
	Optional    []Recommendation `json:"optional"`
	Explanation string           `json:"explanation,omitempty"`
	Degraded    bool             `json:"degraded,omitempty"`
}

type MaestroVerifyRequest struct {
	Version      string           `json:"version"`
	Lifecycle    MaestroLifecycle `json:"lifecycle"`
	ProjectID    string           `json:"project_id,omitempty"`
	MissionID    string           `json:"mission_id,omitempty"`
	ChangedPaths []string         `json:"changed_paths,omitempty"`
	Checks       []string         `json:"checks,omitempty"`
	EvidenceRefs []string         `json:"evidence_refs,omitempty"`
	RiskSignals  []string         `json:"risk_signals,omitempty"`
}

type MaestroVerifyResponse struct {
	Version         string           `json:"version"`
	Lifecycle       MaestroLifecycle `json:"lifecycle"`
	Passed          bool             `json:"passed"`
	RequiredActions []string         `json:"required_actions,omitempty"`
	Explanation     string           `json:"explanation,omitempty"`
	Degraded        bool             `json:"degraded,omitempty"`
}

// Recommendation is a single actionable recommendation from Maestro.
type Recommendation struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"` // "action" | "config" | "security" | "process"
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Apply       string         `json:"apply"` // action identifier
	Why         string         `json:"why"`   // explanation
	Risk        string         `json:"risk"`  // "low" | "medium" | "high"
	Gates       []string       `json:"gates,omitempty"`
	Skills      []string       `json:"skills,omitempty"`
	Verify      string         `json:"verify,omitempty"` // how to verify
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// MaestroStatus represents the current Maestro integration state.
type MaestroStatus struct {
	Available    bool               `json:"available"`
	Mode         MaestroMode        `json:"mode"`
	Capabilities *MaestroCapability `json:"capabilities,omitempty"`
	LastCheck    time.Time          `json:"last_check"`
	Error        string             `json:"error,omitempty"`
}

// MaestroClient handles communication with the Maestro process.
type MaestroClient struct {
	status     MaestroStatus
	maestroBin string // path to maestro binary
}

// NewMaestroClient creates a client that discovers the Maestro binary.
func NewMaestroClient() *MaestroClient {
	bin := findMaestroBin()
	c := &MaestroClient{maestroBin: bin}
	c.checkAvailability()
	return c
}

func findMaestroBin() string {
	// LookPath with enhanced developer toolchains
	candidates := []string{"orquestrador-maestro", "maestro", "orquestrador"}
	for _, name := range candidates {
		if path, err := runtime.LookPath(name); err == nil && path != "" {
			return path
		}
	}

	// Direct paths if not in PATH
	commonPaths := []string{
		"/usr/local/bin/orquestrador-maestro",
		"/usr/local/bin/maestro",
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		commonPaths = append(commonPaths, filepath.Join(home, ".local", "bin", "maestro"))
		commonPaths = append(commonPaths, filepath.Join(home, ".local", "bin", "orquestrador-maestro"))
		commonPaths = append(commonPaths, filepath.Join(home, ".orquestrador", "bin", "orquestrador-maestro"))
		commonPaths = append(commonPaths, filepath.Join(home, ".orquestrador", "bin", "maestro"))
		matches, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "orquestrador-maestro"))
		if len(matches) > 0 {
			commonPaths = append(commonPaths, matches[len(matches)-1])
		}
	}
	for _, p := range commonPaths {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

func findOrquestradorDirs() []string {
	var dirs []string
	// Explicit override is useful for CI and non-standard installations.
	if explicit := strings.TrimSpace(os.Getenv("NEXUS_ORQUESTRADOR_DIR")); explicit != "" {
		if fi, err := os.Stat(explicit); err == nil && fi.IsDir() {
			dirs = append(dirs, filepath.Clean(explicit))
			return dirs
		}
	}
	// Host user home.
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidate := filepath.Join(home, ".orquestrador")
		if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
			dirs = append(dirs, candidate)
		}
		// Profile homes can be nested below the real user home. Walk ancestors
		// so the canonical global catalog is available without hardcoding an OS
		// or profile layout. The nearest catalog remains first and wins overrides.
		ancestor := filepath.Dir(home)
		for i := 0; i < 10 && ancestor != filepath.Dir(ancestor); i++ {
			candidate = filepath.Join(ancestor, ".orquestrador")
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				dirs = appendUniquePath(dirs, candidate)
			}
			ancestor = filepath.Dir(ancestor)
		}
	}
	// Isolated ai-cli/Nexus profile homes under the canonical cross-platform DataDir.
	if dataDir, err := coreconfig.DataDir(); err == nil && dataDir != "" {
		pattern := filepath.Join(dataDir, "profiles", "*", "*", "home", ".orquestrador")
		matches, _ := filepath.Glob(pattern)
		sort.Strings(matches)
		for _, candidate := range matches {
			if fi, err := os.Stat(candidate); err == nil && fi.IsDir() {
				dirs = appendUniquePath(dirs, candidate)
			}
		}
	}
	return dirs
}

func appendUniquePath(paths []string, candidate string) []string {
	candidate = filepath.Clean(candidate)
	for _, path := range paths {
		if filepath.Clean(path) == candidate {
			return paths
		}
	}
	return append(paths, candidate)
}

func findOrquestradorDir() string {
	dirs := findOrquestradorDirs()
	if len(dirs) == 0 {
		return ""
	}
	return dirs[0]
}

func (c *MaestroClient) checkAvailability() {
	if c.maestroBin == "" {
		// Fallback: check if .orquestrador exists with valid skills
		dirs := findOrquestradorDirs()
		if len(dirs) > 0 {
			cap, err := c.queryCapabilitiesFromDirs(dirs)
			if err == nil && cap != nil && len(cap.Skills) > 0 {
				c.status = MaestroStatus{
					Available:    true,
					Mode:         MaestroAssist,
					Capabilities: cap,
					LastCheck:    time.Now(),
				}
				return
			}
		}
		c.status = MaestroStatus{
			Available: false,
			Mode:      MaestroOff,
			Error:     "maestro binary not found",
			LastCheck: time.Now(),
		}
		return
	}

	// Try to get capabilities / version.
	cap, err := c.queryCapabilities()
	if err != nil {
		c.status = MaestroStatus{
			Available: false,
			Mode:      MaestroOff,
			Error:     err.Error(),
			LastCheck: time.Now(),
		}
		return
	}

	c.status = MaestroStatus{
		Available:    true,
		Mode:         MaestroAssist,
		Capabilities: cap,
		LastCheck:    time.Now(),
	}
}

func (c *MaestroClient) queryCapabilitiesFromDir(orqDir string) (*MaestroCapability, error) {
	var skillDescs []MaestroSkillDesc
	manifestPath := filepath.Join(orqDir, "SKILLS_MANIFEST.json")
	if data, err := os.ReadFile(manifestPath); err == nil {
		var parsed struct {
			Skills map[string]struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Category    string   `json:"category"`
				Risk        string   `json:"risk"`
				Triggers    []string `json:"triggers"`
				Aliases     []string `json:"aliases"`
			} `json:"skills"`
		}
		if err := json.Unmarshal(data, &parsed); err == nil && len(parsed.Skills) > 0 {
			for s, meta := range parsed.Skills {
				name := meta.Name
				if name == "" {
					name = s
				}
				skillDescs = append(skillDescs, MaestroSkillDesc{
					ID:          s,
					Name:        name,
					Description: meta.Description,
					Category:    meta.Category,
					Risk:        meta.Risk,
					Triggers:    meta.Triggers,
					Aliases:     meta.Aliases,
					Prompt:      readSkillPrompt(orqDir, s),
				})
			}
		}
	}
	if len(skillDescs) == 0 {
		skillsDir := filepath.Join(orqDir, "skills")
		if entries, err := os.ReadDir(skillsDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					id := entry.Name()
					skillDescs = append(skillDescs, MaestroSkillDesc{
						ID:          id,
						Name:        id,
						Description: "",
						Category:    "custom",
						Prompt:      readSkillPrompt(orqDir, id),
					})
				}
			}
		}
	}
	sort.Slice(skillDescs, func(i, j int) bool {
		return skillDescs[i].ID < skillDescs[j].ID
	})
	return &MaestroCapability{
		Version:   "0.2.4",
		Modes:     []string{"OFF", "ASSIST", "ORCHESTRATE"},
		Skills:    skillDescs,
		Gates:     []string{},
		Processes: []string{},
	}, nil
}

func (c *MaestroClient) queryCapabilitiesFromDirs(dirs []string) (*MaestroCapability, error) {
	merged := make(map[string]MaestroSkillDesc)
	// Discover global first and active profile last so profile metadata can
	// intentionally override a canonical skill without hiding global skills.
	for i := len(dirs) - 1; i >= 0; i-- {
		cap, err := c.queryCapabilitiesFromDir(dirs[i])
		if err != nil {
			continue
		}
		for _, skill := range cap.Skills {
			merged[skill.ID] = skill
		}
	}
	if len(merged) == 0 {
		return nil, fmt.Errorf("no Maestro skills found")
	}
	skills := make([]MaestroSkillDesc, 0, len(merged))
	for _, skill := range merged {
		skills = append(skills, skill)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].ID < skills[j].ID })
	return &MaestroCapability{Version: "0.2.4", Modes: []string{"OFF", "ASSIST", "ORCHESTRATE"}, Skills: skills, Gates: []string{}, Processes: []string{}}, nil
}

func readSkillPrompt(orqDir, skillID string) string {
	path := filepath.Join(orqDir, "skills", skillID, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (c *MaestroClient) querySkillsList() ([]MaestroSkillDesc, error) {
	if c == nil || strings.TrimSpace(c.maestroBin) == "" {
		return nil, fmt.Errorf("no maestro binary")
	}
	out, err := exec.Command(c.maestroBin, "skills", "list", "--json").Output()
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Skills []MaestroSkillDesc `json:"skills"`
	}
	if err := json.Unmarshal(out, &envelope); err == nil && len(envelope.Skills) > 0 {
		return normalizeMaestroSkillList(envelope.Skills), nil
	}
	var direct []MaestroSkillDesc
	if err := json.Unmarshal(out, &direct); err != nil {
		return nil, fmt.Errorf("decode maestro skills list: %w", err)
	}
	if len(direct) == 0 {
		return nil, fmt.Errorf("maestro skills list returned no skills")
	}
	return normalizeMaestroSkillList(direct), nil
}

func normalizeMaestroSkillList(skills []MaestroSkillDesc) []MaestroSkillDesc {
	byID := make(map[string]MaestroSkillDesc, len(skills))
	for _, skill := range skills {
		skill.ID = strings.TrimSpace(skill.ID)
		if skill.ID == "" {
			continue
		}
		if strings.TrimSpace(skill.Name) == "" {
			skill.Name = skill.ID
		}
		byID[skill.ID] = skill
	}
	result := make([]MaestroSkillDesc, 0, len(byID))
	for _, skill := range byID {
		result = append(result, skill)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func mergeMaestroSkillMetadata(authoritative, fallback []MaestroSkillDesc) []MaestroSkillDesc {
	merged := make(map[string]MaestroSkillDesc, len(authoritative)+len(fallback))
	for _, skill := range fallback {
		if strings.TrimSpace(skill.ID) != "" {
			merged[skill.ID] = skill
		}
	}
	for _, skill := range authoritative {
		if strings.TrimSpace(skill.ID) == "" {
			continue
		}
		if previous, ok := merged[skill.ID]; ok {
			if skill.Name == "" {
				skill.Name = previous.Name
			}
			if skill.Description == "" {
				skill.Description = previous.Description
			}
			if skill.Category == "" {
				skill.Category = previous.Category
			}
			if skill.Risk == "" {
				skill.Risk = previous.Risk
			}
			if len(skill.Triggers) == 0 {
				skill.Triggers = append([]string(nil), previous.Triggers...)
			}
			if len(skill.Aliases) == 0 {
				skill.Aliases = append([]string(nil), previous.Aliases...)
			}
			if skill.Prompt == "" {
				skill.Prompt = previous.Prompt
			}
		}
		merged[skill.ID] = skill
	}
	result := make([]MaestroSkillDesc, 0, len(merged))
	for _, skill := range merged {
		result = append(result, skill)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (c *MaestroClient) queryCapabilities() (*MaestroCapability, error) {
	if c.maestroBin == "" {
		orqDir := findOrquestradorDir()
		if orqDir != "" {
			return c.queryCapabilitiesFromDir(orqDir)
		}
		return nil, fmt.Errorf("no maestro binary")
	}

	// 1. Prefer the machine-readable Maestro v1 contracts. Capabilities owns
	// modes/gates/processes; skills list owns the skill inventory. Local
	// manifests remain a bounded fallback/enrichment source for SKILL.md content.
	cmd := exec.Command(c.maestroBin, "capabilities", "--json")
	if out, err := cmd.Output(); err == nil {
		var cap MaestroCapability
		if err := json.Unmarshal(out, &cap); err == nil && cap.Version != "" {
			var localSkills []MaestroSkillDesc
			if dirs := findOrquestradorDirs(); len(dirs) > 0 {
				if local, localErr := c.queryCapabilitiesFromDirs(dirs); localErr == nil {
					localSkills = local.Skills
				}
			}
			if cliSkills, skillsErr := c.querySkillsList(); skillsErr == nil {
				cap.Skills = mergeMaestroSkillMetadata(cliSkills, localSkills)
			} else if len(cap.Skills) > 0 {
				cap.Skills = mergeMaestroSkillMetadata(cap.Skills, localSkills)
			} else {
				cap.Skills = localSkills
			}
			return &cap, nil
		}
	}

	// 2. Query version from CLI
	verOut, err := exec.Command(c.maestroBin, "version").Output()
	if err != nil {
		fallbackOut, _ := exec.Command(c.maestroBin, "--version").Output()
		verOut = fallbackOut
	}
	version := strings.TrimSpace(string(verOut))
	if version == "" {
		version = "0.2.4"
	}

	// 3. Resolve the explicit skills-list contract, then enrich/fallback from
	// bounded local .orquestrador roots when the installed Maestro is older.
	var localSkills []MaestroSkillDesc
	if dirs := findOrquestradorDirs(); len(dirs) > 0 {
		if cap, localErr := c.queryCapabilitiesFromDirs(dirs); localErr == nil && cap != nil {
			localSkills = cap.Skills
		}
	}
	var skillDescs []MaestroSkillDesc
	if cliSkills, skillsErr := c.querySkillsList(); skillsErr == nil {
		skillDescs = mergeMaestroSkillMetadata(cliSkills, localSkills)
	} else {
		skillDescs = localSkills
	}

	sort.Slice(skillDescs, func(i, j int) bool {
		return skillDescs[i].ID < skillDescs[j].ID
	})
	return &MaestroCapability{
		Version: version,
		Modes:   []string{"OFF", "ASSIST", "ORCHESTRATE"},
		Skills:  skillDescs,
		// Gates and processes are intentionally empty unless the Maestro binary
		// reports them through capabilities --json. Nexus never invents them.
		Gates:     []string{},
		Processes: []string{},
	}, nil
}

// Status returns the current Maestro integration status.
func (c *MaestroClient) Status() MaestroStatus {
	return c.status
}

// Catalog returns the merged operational catalog plus the local Maestro
// library. No network lookup or installation is performed here. Results are
// cached for catalogCacheTTL to avoid redundant disk reads within a single
// interactive session.
func (c *MaestroClient) Catalog() SkillCatalog {
	globalCatalogCache.mu.RLock()
	if globalCatalogCache.catalog != nil && time.Now().Before(globalCatalogCache.expiresAt) {
		result := *globalCatalogCache.catalog
		globalCatalogCache.mu.RUnlock()
		return result
	}
	globalCatalogCache.mu.RUnlock()

	groups := make([][]CatalogSkill, 0)
	for i, dir := range findOrquestradorDirs() {
		cap, err := c.queryCapabilitiesFromDir(dir)
		if err != nil || cap == nil {
			continue
		}
		source := SkillSourceCommunity
		if i == 0 {
			source = SkillSourceCanonical
		}
		availability := SkillSynchronizable
		if source == SkillSourceCanonical {
			availability = SkillAvailable
		}
		items := make([]CatalogSkill, 0, len(cap.Skills))
		for _, skill := range cap.Skills {
			items = append(items, CatalogSkill{MaestroSkillDesc: skill, Source: source, Availability: availability, Mode: "task", Root: dir, Contract: skill.Prompt})
		}
		groups = append(groups, items)
	}
	result := MergeSkillCatalog(groups...)

	globalCatalogCache.mu.Lock()
	globalCatalogCache.catalog = &result
	globalCatalogCache.expiresAt = time.Now().Add(catalogCacheTTL)
	globalCatalogCache.mu.Unlock()

	return result
}

// CatalogSkill returns a validated skill contract by ID, including skills
// that can only be attached to this task and are not globally installed.
func (c *MaestroClient) CatalogSkill(id string) (CatalogSkill, bool) {
	catalog := c.Catalog()
	for _, skill := range catalog.Library {
		if skill.ID == strings.TrimSpace(id) {
			return skill, true
		}
	}
	return CatalogSkill{}, false
}

// ListSkills returns all available Maestro skill names.
func (c *MaestroClient) ListSkills(ctx context.Context) ([]string, error) {
	if c.status.Capabilities != nil {
		return c.status.Capabilities.SkillIDs(), nil
	}
	return nil, nil
}

// GetAdvice requests recommendations from Maestro for the given context.
func (c *MaestroClient) GetAdvice(ctx AdviceContext, intent string) (*AdviceResponse, error) {
	lifecycle := normalizeMaestroLifecycle(ctx.Lifecycle)
	if !c.status.Available || c.maestroBin == "" {
		return &AdviceResponse{
			Version: MaestroVersion, Mode: MaestroOff, Lifecycle: lifecycle, Degraded: true,
		}, fmt.Errorf("maestro unavailable (MAESTRO_DEGRADED)")
	}

	// 1. Try CLI advise command if supported
	req := AdviceRequest{
		Version: MaestroVersion, Context: ctx, Intent: intent, Scope: "project", Lifecycle: lifecycle,
	}
	reqBytes, _ := json.Marshal(req)
	cmdAdvise := exec.Command(c.maestroBin, "advise", "--json")
	cmdAdvise.Stdin = stringToReader(reqBytes)
	if out, err := cmdAdvise.Output(); err == nil {
		var resp AdviceResponse
		if err := json.Unmarshal(out, &resp); err == nil && len(resp.Recommended) > 0 {
			if resp.Lifecycle == "" {
				resp.Lifecycle = lifecycle
			}
			return &resp, nil
		}
	}

	// A failed/malformed advise command means the Maestro contract is unavailable.
	// Do not synthesize skill IDs, gates or process advice inside Nexus.
	version := MaestroVersion
	if c.status.Capabilities != nil && c.status.Capabilities.Version != "" {
		version = c.status.Capabilities.Version
	}
	return &AdviceResponse{
		Version: version, Mode: MaestroOff, Lifecycle: lifecycle, Degraded: true,
	}, fmt.Errorf("maestro advise unavailable or returned an invalid contract (MAESTRO_DEGRADED)")
}

func (c *MaestroClient) Verify(ctx context.Context, req MaestroVerifyRequest) (*MaestroVerifyResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req.Version == "" {
		req.Version = MaestroVersion
	}
	req.Lifecycle = MaestroLifecycleVerify
	if err := sanitizeMaestroVerifyRequest(&req); err != nil {
		return &MaestroVerifyResponse{
			Version: req.Version, Lifecycle: MaestroLifecycleVerify, Passed: false, Degraded: true,
		}, err
	}
	if c == nil || !c.status.Available || c.maestroBin == "" {
		return &MaestroVerifyResponse{
			Version: req.Version, Lifecycle: MaestroLifecycleVerify, Passed: false, Degraded: true,
		}, fmt.Errorf("maestro unavailable (MAESTRO_DEGRADED)")
	}

	raw, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal Maestro verify request: %w", err)
	}
	verifyCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(verifyCtx, c.maestroBin, "verify", "--json")
	cmd.Stdin = stringToReader(raw)
	out, err := cmd.Output()
	if err != nil {
		return &MaestroVerifyResponse{
			Version: req.Version, Lifecycle: MaestroLifecycleVerify, Passed: false, Degraded: true,
		}, fmt.Errorf("maestro verify unavailable or failed (MAESTRO_DEGRADED): %w", err)
	}
	var resp MaestroVerifyResponse
	if err := json.Unmarshal(out, &resp); err != nil || strings.TrimSpace(resp.Version) == "" {
		return &MaestroVerifyResponse{
			Version: req.Version, Lifecycle: MaestroLifecycleVerify, Passed: false, Degraded: true,
		}, fmt.Errorf("maestro verify returned an invalid contract (MAESTRO_DEGRADED)")
	}
	resp.Lifecycle = MaestroLifecycleVerify
	return &resp, nil
}

func sanitizeMaestroVerifyRequest(req *MaestroVerifyRequest) error {
	if req == nil {
		return fmt.Errorf("maestro verify request is required")
	}
	clean := func(values []string, maxItems, maxLen int) ([]string, error) {
		if len(values) > maxItems {
			return nil, fmt.Errorf("maestro verify input exceeds %d items", maxItems)
		}
		out := make([]string, 0, len(values))
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if len(value) > maxLen {
				return nil, fmt.Errorf("maestro verify input item exceeds %d bytes", maxLen)
			}
			if LooksLikeSecret(value) {
				return nil, fmt.Errorf("refusing secret-like Maestro verify input")
			}
			out = append(out, value)
		}
		return out, nil
	}
	var err error
	if req.ChangedPaths, err = clean(req.ChangedPaths, 256, 1024); err != nil {
		return err
	}
	if req.Checks, err = clean(req.Checks, 128, 2048); err != nil {
		return err
	}
	if req.EvidenceRefs, err = clean(req.EvidenceRefs, 256, 2048); err != nil {
		return err
	}
	if req.RiskSignals, err = clean(req.RiskSignals, 128, 2048); err != nil {
		return err
	}
	if LooksLikeSecret(req.ProjectID) || LooksLikeSecret(req.MissionID) {
		return fmt.Errorf("refusing secret-like Maestro verify identifiers")
	}
	return nil
}

func normalizeMaestroLifecycle(value MaestroLifecycle) MaestroLifecycle {
	switch MaestroLifecycle(strings.ToUpper(strings.TrimSpace(string(value)))) {
	case MaestroLifecyclePlan:
		return MaestroLifecyclePlan
	case MaestroLifecycleRecovery:
		return MaestroLifecycleRecovery
	case MaestroLifecycleVerify:
		return MaestroLifecycleVerify
	default:
		return MaestroLifecycleTask
	}
}

func stringToReader(b []byte) *stringReader {
	return &stringReader{data: b, pos: 0}
}

type stringReader struct {
	data []byte
	pos  int
}

func (r *stringReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
