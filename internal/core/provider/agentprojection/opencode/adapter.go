package opencode

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	projection "github.com/IAPro-Community/nexus/internal/core/provider/agentprojection"
)

const (
	providerID      = "opencode"
	manifestSchema  = 1
	managedMarker   = "<!-- nexus-agent-projection: opencode/"
	manifestDirName = "agent-projection"
)

var (
	roleIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	applyMu       sync.Mutex
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ProviderID() string { return providerID }

// AgentName returns the deterministic host-native name materialized for a
// Nexus ExecutionRole.
func AgentName(roleID string) (string, error) {
	roleID = strings.ToLower(strings.TrimSpace(roleID))
	if !roleIDPattern.MatchString(roleID) {
		return "", fmt.Errorf("invalid OpenCode projection role id %q", roleID)
	}
	return "nexus-" + roleID, nil
}

// DiscoveryArgs returns a non-LLM command that asks OpenCode to enumerate the
// agents visible to the current isolated/custom target environment.
func DiscoveryArgs(providerVersion string) ([]string, error) {
	major, ok := parseMajor(providerVersion)
	if !ok {
		return nil, fmt.Errorf("OpenCode provider version %q is unknown", providerVersion)
	}
	if major >= 2 {
		return []string{"debug", "agents"}, nil
	}
	return []string{"agent", "list"}, nil
}

// ActivationProbeArgs returns an explicit non-interactive OpenCode invocation
// that selects the projected primary-capable agent. It does invoke the
// configured model, so callers must require an explicit user/certification
// opt-in before executing it.
func ActivationProbeArgs(providerVersion, agentName, marker string) ([]string, error) {
	if _, ok := parseMajor(providerVersion); !ok {
		return nil, fmt.Errorf("OpenCode provider version %q is unknown", providerVersion)
	}
	agentName = strings.TrimSpace(agentName)
	if !strings.HasPrefix(agentName, "nexus-") || len(agentName) <= len("nexus-") {
		return nil, fmt.Errorf("invalid projected OpenCode agent name %q", agentName)
	}
	marker = strings.TrimSpace(marker)
	if marker == "" {
		return nil, errors.New("activation probe marker is required")
	}
	prompt := "Reply with exactly " + marker + " and nothing else. Do not call tools."
	return []string{"run", "--agent", agentName, prompt}, nil
}

func (a *Adapter) Capabilities(_ context.Context, providerVersion string) projection.Capabilities {
	major, ok := parseMajor(providerVersion)
	if !ok {
		unknown := projection.CapabilityEvidence{
			Status:      projection.CapabilityUnknown,
			Tested:      false,
			Evidence:    "OpenCode version could not be parsed; native agent format is not assumed",
			ProviderVer: providerVersion,
		}
		return projection.Capabilities{
			Definitions:     unknown,
			PrimaryAgent:    unknown,
			Subagents:       unknown,
			InSessionSwitch: unknown,
			DynamicReload:   unknown,
			ToolPermissions: unknown,
			ModelOverride:   unknown,
			ProjectScope: projection.CapabilityEvidence{
				Status:   projection.CapabilityUnsupported,
				Tested:   true,
				Evidence: "this milestone intentionally permits NEXUS_PROFILE scope only",
			},
			UserScope: projection.CapabilityEvidence{
				Status:   projection.CapabilityUnsupported,
				Tested:   true,
				Evidence: "this milestone intentionally permits NEXUS_PROFILE scope only",
			},
			RemoteProtocol: projection.CapabilityEvidence{
				Status:   projection.CapabilityUnsupported,
				Tested:   true,
				Evidence: "remote/A2A projection is outside the OpenCode adapter milestone",
			},
		}
	}

	versioned := func(status projection.CapabilityStatus, tested bool, evidence string) projection.CapabilityEvidence {
		return projection.CapabilityEvidence{
			Status:      status,
			Tested:      tested,
			Evidence:    evidence,
			ProviderVer: providerVersion,
		}
	}

	format := "OpenCode V1 permission/bash Markdown"
	if major >= 2 {
		format = "OpenCode V2 permissions/shell Markdown"
	}

	return projection.Capabilities{
		Definitions: versioned(
			projection.CapabilityPartial,
			true,
			format+" renderer and isolated profile path are covered by Nexus tests; host discovery still requires runtime probe",
		),
		PrimaryAgent: versioned(
			projection.CapabilityNotTested,
			false,
			"mode=all is rendered, but primary selection is not promoted until a live OpenCode probe succeeds",
		),
		Subagents: versioned(
			projection.CapabilityNotTested,
			false,
			"mode=all is rendered, but subagent invocation is not promoted until a live OpenCode probe succeeds",
		),
		InSessionSwitch: versioned(
			projection.CapabilityNotTested,
			false,
			"OpenCode exposes agent switching, but Nexus has not verified hot activation for an already-running session",
		),
		DynamicReload: versioned(
			projection.CapabilityUnknown,
			false,
			"no assumption is made that an existing session reloads newly materialized agents",
		),
		ToolPermissions: versioned(
			projection.CapabilityNotTested,
			false,
			"version-aware read-only permission rules are rendered but require live enforcement evidence",
		),
		ModelOverride: versioned(
			projection.CapabilityNotTested,
			false,
			"model can be rendered in agent frontmatter but is not runtime-probed",
		),
		ProjectScope: projection.CapabilityEvidence{
			Status:      projection.CapabilityUnsupported,
			Tested:      true,
			Evidence:    "project export is intentionally disabled in the first OpenCode adapter milestone",
			ProviderVer: providerVersion,
		},
		UserScope: projection.CapabilityEvidence{
			Status:      projection.CapabilityUnsupported,
			Tested:      true,
			Evidence:    "real user-home export is intentionally disabled in the first OpenCode adapter milestone",
			ProviderVer: providerVersion,
		},
		RemoteProtocol: projection.CapabilityEvidence{
			Status:      projection.CapabilityUnsupported,
			Tested:      true,
			Evidence:    "this adapter materializes local OpenCode agents only",
			ProviderVer: providerVersion,
		},
	}
}

func (a *Adapter) Plan(ctx context.Context, req projection.Request) (*projection.Plan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	normalized, major, err := normalizeRequest(req)
	if err != nil {
		return nil, err
	}
	artifacts, err := a.Render(ctx, normalized)
	if err != nil {
		return nil, err
	}
	renderer := rendererVersion(major)
	root, _ := filepath.Abs(normalized.ProviderHome)

	manifest, err := loadManifest(root)
	if err != nil {
		return nil, err
	}

	plan := &projection.Plan{
		Provider:        providerID,
		Profile:         normalized.Profile,
		RoleID:          normalized.RoleID,
		Scope:           projection.ScopeNexusProfile,
		Strategy:        projection.ActivationNexusOverlay,
		ManagedRoot:     root,
		RendererVersion: renderer,
		SourceHash:      normalized.SourceHash,
		Artifacts:       artifacts,
		Warnings: []string{
			"native agent materialization is available, but runtime activation remains NEXUS_OVERLAY until activation/enforcement is E2E-proven",
		},
	}

	for _, artifact := range artifacts {
		rel, err := safeRelative(root, artifact.Path)
		if err != nil {
			plan.Conflicts = append(plan.Conflicts, projection.Conflict{
				Path: artifact.Path, Kind: projection.ConflictUnsafePath, Detail: err.Error(),
			})
			continue
		}
		if err := ensureNoSymlinkComponents(root, filepath.Dir(artifact.Path)); err != nil {
			plan.Conflicts = append(plan.Conflicts, projection.Conflict{
				Path: artifact.Path, Kind: projection.ConflictUnsafePath, Detail: err.Error(),
			})
			continue
		}
		info, statErr := os.Lstat(artifact.Path)
		if errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if statErr != nil {
			return nil, fmt.Errorf("inspect projected artifact %s: %w", artifact.Path, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			plan.Conflicts = append(plan.Conflicts, projection.Conflict{
				Path: artifact.Path, Kind: projection.ConflictUnsafePath, Detail: "target is a symbolic link",
			})
			continue
		}
		if !info.Mode().IsRegular() {
			plan.Conflicts = append(plan.Conflicts, projection.Conflict{
				Path: artifact.Path, Kind: projection.ConflictUnsafePath, Detail: "target is not a regular file",
			})
			continue
		}
		data, err := os.ReadFile(artifact.Path)
		if err != nil {
			return nil, fmt.Errorf("read projected artifact %s: %w", artifact.Path, err)
		}
		observed := projection.ContentHash(data)
		receiptHash, managed := manifest.Artifacts[filepath.ToSlash(rel)]
		state, decision := projection.ResolveOwnership(true, managed, observed, receiptHash)
		if decision == projection.DecisionConflict {
			kind := projection.ConflictManagedModified
			if state == projection.OwnershipUnmanaged {
				kind = projection.ConflictUnmanaged
			}
			plan.Conflicts = append(plan.Conflicts, projection.Conflict{
				Path:   artifact.Path,
				Kind:   kind,
				Detail: fmt.Sprintf("ownership state %s; Nexus will not overwrite this file", state),
			})
		}
	}

	return plan, nil
}

func (a *Adapter) Render(ctx context.Context, req projection.Request) ([]projection.GeneratedArtifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	normalized, major, err := normalizeRequest(req)
	if err != nil {
		return nil, err
	}
	root, _ := filepath.Abs(normalized.ProviderHome)
	agentName, err := AgentName(normalized.RoleID)
	if err != nil {
		return nil, err
	}
	target := filepath.Join(root, ".config", "opencode", "agents", agentName+".md")
	if _, err := safeRelative(root, target); err != nil {
		return nil, err
	}

	content := renderAgentMarkdown(normalized, major)
	return []projection.GeneratedArtifact{{
		Path:        target,
		Content:     content,
		Mode:        0o600,
		ContentHash: projection.ContentHash(content),
	}}, nil
}

func (a *Adapter) Apply(ctx context.Context, plan *projection.Plan) (*projection.Receipt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validatePlan(plan); err != nil {
		return nil, err
	}
	if len(plan.Conflicts) > 0 {
		return nil, fmt.Errorf("OpenCode projection has %d conflict(s); refusing to apply", len(plan.Conflicts))
	}

	applyMu.Lock()
	defer applyMu.Unlock()

	root, _ := filepath.Abs(plan.ManagedRoot)
	if err := ensureSafeRoot(root); err != nil {
		return nil, err
	}
	manifest, err := loadManifest(root)
	if err != nil {
		return nil, err
	}

	type backup struct {
		path    string
		existed bool
		content []byte
		mode    fs.FileMode
	}
	backups := make([]backup, 0, len(plan.Artifacts))
	rollback := func() {
		for i := len(backups) - 1; i >= 0; i-- {
			b := backups[i]
			if !b.existed {
				_ = os.Remove(b.path)
				continue
			}
			_ = writeAtomic(b.path, b.content, b.mode)
		}
	}

	for _, artifact := range plan.Artifacts {
		if err := ctx.Err(); err != nil {
			rollback()
			return nil, err
		}
		rel, err := safeRelative(root, artifact.Path)
		if err != nil {
			rollback()
			return nil, err
		}
		if err := ensureNoSymlinkComponents(root, filepath.Dir(artifact.Path)); err != nil {
			rollback()
			return nil, err
		}

		var b backup
		b.path = artifact.Path
		info, statErr := os.Lstat(artifact.Path)
		switch {
		case errors.Is(statErr, os.ErrNotExist):
			b.mode = artifact.Mode
		case statErr != nil:
			rollback()
			return nil, statErr
		default:
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				rollback()
				return nil, fmt.Errorf("refusing to replace non-regular OpenCode agent file %s", artifact.Path)
			}
			data, readErr := os.ReadFile(artifact.Path)
			if readErr != nil {
				rollback()
				return nil, readErr
			}
			observed := projection.ContentHash(data)
			receiptHash, managed := manifest.Artifacts[filepath.ToSlash(rel)]
			_, decision := projection.ResolveOwnership(true, managed, observed, receiptHash)
			if decision == projection.DecisionConflict {
				rollback()
				return nil, fmt.Errorf("OpenCode agent file changed since plan or is unmanaged: %s", artifact.Path)
			}
			b.existed = true
			b.content = data
			b.mode = info.Mode().Perm()
		}
		backups = append(backups, b)

		if err := os.MkdirAll(filepath.Dir(artifact.Path), 0o700); err != nil {
			rollback()
			return nil, err
		}
		if err := ensureNoSymlinkComponents(root, filepath.Dir(artifact.Path)); err != nil {
			rollback()
			return nil, err
		}
		if err := writeAtomic(artifact.Path, artifact.Content, artifact.Mode); err != nil {
			rollback()
			return nil, err
		}
		manifest.Artifacts[filepath.ToSlash(rel)] = artifact.ContentHash
	}

	manifest.Schema = manifestSchema
	manifest.Provider = providerID
	manifest.RendererVersion = plan.RendererVersion
	if err := saveManifest(root, manifest); err != nil {
		rollback()
		return nil, fmt.Errorf("save OpenCode projection manifest: %w", err)
	}

	now := time.Now().UTC()
	receipt := &projection.Receipt{
		ID:              newReceiptID(),
		Provider:        providerID,
		Profile:         plan.Profile,
		RoleID:          plan.RoleID,
		Scope:           projection.ScopeNexusProfile,
		Strategy:        plan.Strategy,
		ManagedRoot:     root,
		RendererVersion: plan.RendererVersion,
		SourceHash:      plan.SourceHash,
		AppliedAt:       now,
	}
	for _, artifact := range plan.Artifacts {
		receipt.Artifacts = append(receipt.Artifacts, projection.ArtifactReceipt{
			Path: artifact.Path, ContentHash: artifact.ContentHash,
		})
	}
	return receipt, nil
}

func (a *Adapter) Verify(ctx context.Context, receipt *projection.Receipt) (*projection.Verification, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateReceipt(receipt); err != nil {
		return nil, err
	}
	root, _ := filepath.Abs(receipt.ManagedRoot)
	manifest, err := loadManifest(root)
	if err != nil {
		return nil, err
	}

	result := &projection.Verification{
		Verified: true,
		Warnings: []string{
			"artifact hashes and ownership are verified; OpenCode discovery/selection/permission enforcement still require a live provider probe",
		},
	}
	for _, artifact := range receipt.Artifacts {
		rel, relErr := safeRelative(root, artifact.Path)
		if relErr != nil {
			result.Verified = false
			result.Errors = append(result.Errors, relErr.Error())
			continue
		}
		info, statErr := os.Lstat(artifact.Path)
		if statErr != nil {
			result.Verified = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", artifact.Path, statErr))
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			result.Verified = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s is not a regular managed file", artifact.Path))
			continue
		}
		data, readErr := os.ReadFile(artifact.Path)
		if readErr != nil {
			result.Verified = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", artifact.Path, readErr))
			continue
		}
		hash := projection.ContentHash(data)
		if hash != artifact.ContentHash {
			result.Verified = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s content hash mismatch", artifact.Path))
		}
		if manifest.Artifacts[filepath.ToSlash(rel)] != artifact.ContentHash {
			result.Verified = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s ownership manifest mismatch", artifact.Path))
		}
		if !bytes.Contains(data, []byte(managedMarker)) {
			result.Verified = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s is missing Nexus managed marker", artifact.Path))
		}
	}
	return result, nil
}

func (a *Adapter) Remove(ctx context.Context, receipt *projection.Receipt) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateReceipt(receipt); err != nil {
		return err
	}

	applyMu.Lock()
	defer applyMu.Unlock()

	root, _ := filepath.Abs(receipt.ManagedRoot)
	manifest, err := loadManifest(root)
	if err != nil {
		return err
	}

	type backup struct {
		path    string
		content []byte
		mode    fs.FileMode
	}
	var backups []backup

	for _, artifact := range receipt.Artifacts {
		rel, err := safeRelative(root, artifact.Path)
		if err != nil {
			return err
		}
		expected, managed := manifest.Artifacts[filepath.ToSlash(rel)]
		if !managed || expected != artifact.ContentHash {
			return fmt.Errorf("refusing to remove unmanaged or superseded OpenCode agent file %s", artifact.Path)
		}
		info, err := os.Lstat(artifact.Path)
		if err != nil {
			return fmt.Errorf("inspect OpenCode agent file %s: %w", artifact.Path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to remove non-regular OpenCode agent file %s", artifact.Path)
		}
		data, err := os.ReadFile(artifact.Path)
		if err != nil {
			return err
		}
		if projection.ContentHash(data) != artifact.ContentHash {
			return fmt.Errorf("refusing to remove modified OpenCode agent file %s", artifact.Path)
		}
		backups = append(backups, backup{path: artifact.Path, content: data, mode: info.Mode().Perm()})
	}

	for _, b := range backups {
		if err := os.Remove(b.path); err != nil {
			for _, restore := range backups {
				if _, statErr := os.Stat(restore.path); errors.Is(statErr, os.ErrNotExist) {
					_ = writeAtomic(restore.path, restore.content, restore.mode)
				}
			}
			return err
		}
		rel, _ := safeRelative(root, b.path)
		delete(manifest.Artifacts, filepath.ToSlash(rel))
	}
	if err := saveManifest(root, manifest); err != nil {
		for _, restore := range backups {
			_ = writeAtomic(restore.path, restore.content, restore.mode)
		}
		return fmt.Errorf("save OpenCode projection manifest after removal: %w", err)
	}
	return nil
}

type ownershipManifest struct {
	Schema          int               `json:"schema"`
	Provider        string            `json:"provider"`
	RendererVersion string            `json:"renderer_version"`
	Artifacts       map[string]string `json:"artifacts"`
}

func normalizeRequest(req projection.Request) (projection.Request, int, error) {
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	if req.Provider == "" {
		req.Provider = providerID
	}
	if req.Provider != providerID {
		return req, 0, fmt.Errorf("OpenCode projection cannot handle provider %q", req.Provider)
	}
	if req.Scope == "" {
		req.Scope = projection.ScopeNexusProfile
	}
	if req.Scope != projection.ScopeNexusProfile {
		return req, 0, fmt.Errorf("OpenCode projection scope %s is unsupported; only NEXUS_PROFILE is allowed in this milestone", req.Scope)
	}
	req.Profile = strings.TrimSpace(req.Profile)
	if req.Profile == "" {
		return req, 0, errors.New("OpenCode projection profile is required")
	}
	req.ProviderHome = strings.TrimSpace(req.ProviderHome)
	if req.ProviderHome == "" {
		return req, 0, errors.New("OpenCode projection provider_home is required")
	}
	root, err := filepath.Abs(req.ProviderHome)
	if err != nil {
		return req, 0, fmt.Errorf("resolve OpenCode provider home: %w", err)
	}
	req.ProviderHome = root

	req.RoleID = strings.ToLower(strings.TrimSpace(req.RoleID))
	if !roleIDPattern.MatchString(req.RoleID) {
		return req, 0, fmt.Errorf("invalid OpenCode projection role id %q", req.RoleID)
	}
	req.Role = strings.TrimSpace(req.Role)
	req.AccessIntent = strings.ToUpper(strings.TrimSpace(req.AccessIntent))
	switch req.AccessIntent {
	case "WRITE", "READ_ONLY", "VERIFY_ONLY":
	default:
		return req, 0, fmt.Errorf("unsupported OpenCode access intent %q", req.AccessIntent)
	}

	major, ok := parseMajor(req.ProviderVersion)
	if !ok {
		return req, 0, fmt.Errorf("OpenCode provider version %q is unknown; refusing to guess native agent syntax", req.ProviderVersion)
	}
	if major < 1 {
		return req, 0, fmt.Errorf("unsupported OpenCode provider version %q", req.ProviderVersion)
	}

	if strings.TrimSpace(req.SourceHash) == "" {
		req.SourceHash = requestHash(req)
	}
	return req, major, nil
}

func validatePlan(plan *projection.Plan) error {
	if plan == nil {
		return errors.New("OpenCode projection plan is required")
	}
	if plan.Provider != providerID {
		return fmt.Errorf("invalid OpenCode projection provider %q", plan.Provider)
	}
	if plan.Scope != projection.ScopeNexusProfile {
		return fmt.Errorf("OpenCode projection can apply only NEXUS_PROFILE scope, got %s", plan.Scope)
	}
	if strings.TrimSpace(plan.ManagedRoot) == "" {
		return errors.New("OpenCode projection managed root is required")
	}
	if len(plan.Artifacts) == 0 {
		return errors.New("OpenCode projection plan contains no artifacts")
	}
	return nil
}

func validateReceipt(receipt *projection.Receipt) error {
	if receipt == nil {
		return errors.New("OpenCode projection receipt is required")
	}
	if receipt.Provider != providerID {
		return fmt.Errorf("invalid OpenCode projection receipt provider %q", receipt.Provider)
	}
	if receipt.Scope != projection.ScopeNexusProfile {
		return fmt.Errorf("OpenCode projection receipt scope %s is unsupported", receipt.Scope)
	}
	if strings.TrimSpace(receipt.ManagedRoot) == "" {
		return errors.New("OpenCode projection receipt managed root is required")
	}
	if len(receipt.Artifacts) == 0 {
		return errors.New("OpenCode projection receipt contains no artifacts")
	}
	return nil
}

func parseMajor(version string) (int, bool) {
	version = strings.TrimSpace(strings.ToLower(version))
	version = strings.TrimPrefix(version, "opencode")
	version = strings.TrimSpace(version)
	if strings.HasPrefix(version, "v") {
		version = strings.TrimPrefix(version, "v")
	}
	if version == "" {
		return 0, false
	}
	part := version
	if idx := strings.IndexAny(part, ".-+ "); idx >= 0 {
		part = part[:idx]
	}
	major, err := strconv.Atoi(part)
	return major, err == nil && major > 0
}

func rendererVersion(major int) string {
	if major >= 2 {
		return "opencode-v2-agent-md/1"
	}
	return "opencode-v1-agent-md/1"
}

func renderAgentMarkdown(req projection.Request, major int) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("description: ")
	b.WriteString(yamlQuoted("Nexus " + req.RoleID + " execution role"))
	b.WriteString("\nmode: all\n")
	if req.Model != "" {
		b.WriteString("model: ")
		b.WriteString(yamlQuoted(req.Model))
		b.WriteString("\n")
	}
	if major >= 2 {
		renderV2Permissions(&b, req.AccessIntent)
	} else {
		renderV1Permissions(&b, req.AccessIntent)
	}
	b.WriteString("---\n\n")
	b.WriteString(managedMarker)
	b.WriteString(rendererVersion(major))
	b.WriteString(" -->\n")
	b.WriteString("You are operating as the Nexus ")
	b.WriteString(req.RoleID)
	b.WriteString(" execution role")
	if req.Role != "" {
		b.WriteString(" (specialization: ")
		b.WriteString(req.Role)
		b.WriteString(")")
	}
	b.WriteString(".\n")
	b.WriteString("Access intent: ")
	b.WriteString(req.AccessIntent)
	b.WriteString(".\n")
	b.WriteString("Do not expand authorization beyond the current Nexus task/runtime contract.\n")

	switch req.AccessIntent {
	case "READ_ONLY":
		b.WriteString("Do not modify files or run shell commands. Use read/search capabilities only.\n")
	case "VERIFY_ONLY":
		b.WriteString("Do not edit files directly. Shell verification requires host approval and must remain scoped to verification.\n")
	}

	appendSection(&b, "Instructions", req.Instructions)
	appendSection(&b, "Responsibilities", req.Responsibilities)
	appendSection(&b, "Capabilities", req.Capabilities)
	appendSection(&b, "Constraints", req.Constraints)

	return []byte(b.String())
}

func renderV1Permissions(b *strings.Builder, accessIntent string) {
	switch accessIntent {
	case "READ_ONLY":
		b.WriteString("permission:\n  edit: deny\n  bash: deny\n")
	case "VERIFY_ONLY":
		b.WriteString("permission:\n  edit: deny\n  bash: ask\n")
	}
}

func renderV2Permissions(b *strings.Builder, accessIntent string) {
	switch accessIntent {
	case "READ_ONLY":
		b.WriteString("permissions:\n")
		b.WriteString("  - action: edit\n    resource: ")
		b.WriteString(yamlQuoted("*"))
		b.WriteString("\n    effect: deny\n")
		b.WriteString("  - action: shell\n    resource: ")
		b.WriteString(yamlQuoted("*"))
		b.WriteString("\n    effect: deny\n")
	case "VERIFY_ONLY":
		b.WriteString("permissions:\n")
		b.WriteString("  - action: edit\n    resource: ")
		b.WriteString(yamlQuoted("*"))
		b.WriteString("\n    effect: deny\n")
		b.WriteString("  - action: shell\n    resource: ")
		b.WriteString(yamlQuoted("*"))
		b.WriteString("\n    effect: ask\n")
	}
}

func appendSection(b *strings.Builder, title string, items []string) {
	clean := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			clean = append(clean, item)
		}
	}
	if len(clean) == 0 {
		return
	}
	b.WriteString("\n")
	b.WriteString(title)
	b.WriteString(":\n")
	for _, item := range clean {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
}

func yamlQuoted(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func requestHash(req projection.Request) string {
	payload := struct {
		RoleID           string   `json:"role_id"`
		Role             string   `json:"role"`
		AccessIntent     string   `json:"access_intent"`
		Instructions     []string `json:"instructions,omitempty"`
		Responsibilities []string `json:"responsibilities,omitempty"`
		Capabilities     []string `json:"capabilities,omitempty"`
		Constraints      []string `json:"constraints,omitempty"`
		Model            string   `json:"model,omitempty"`
	}{
		RoleID: req.RoleID, Role: req.Role, AccessIntent: req.AccessIntent,
		Instructions: req.Instructions, Responsibilities: req.Responsibilities,
		Capabilities: req.Capabilities, Constraints: req.Constraints, Model: req.Model,
	}
	raw, _ := json.Marshal(payload)
	return projection.ContentHash(raw)
}

func manifestPath(root string) string {
	return filepath.Join(root, ".nexus", manifestDirName, "opencode.json")
}

func loadManifest(root string) (ownershipManifest, error) {
	m := ownershipManifest{
		Schema: manifestSchema, Provider: providerID, Artifacts: map[string]string{},
	}
	path := manifestPath(root)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return m, fmt.Errorf("OpenCode projection manifest is not a regular file: %s", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, fmt.Errorf("invalid OpenCode projection manifest: %w", err)
	}
	if m.Schema != manifestSchema || m.Provider != providerID {
		return m, fmt.Errorf("unsupported OpenCode projection manifest schema/provider")
	}
	if m.Artifacts == nil {
		m.Artifacts = map[string]string{}
	}
	return m, nil
}

func saveManifest(root string, m ownershipManifest) error {
	path := manifestPath(root)
	if _, err := safeRelative(root, path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := ensureNoSymlinkComponents(root, filepath.Dir(path)); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeAtomic(path, raw, 0o600)
}

func safeRelative(root, target string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return "", err
	}
	if rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %s escapes managed OpenCode profile root %s", absTarget, absRoot)
	}
	return rel, nil
}

func ensureSafeRoot(root string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if info, statErr := os.Lstat(root); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("managed OpenCode profile root is a symbolic link: %s", root)
		}
		if !info.IsDir() {
			return fmt.Errorf("managed OpenCode profile root is not a directory: %s", root)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	} else if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	return nil
}

func ensureNoSymlinkComponents(root, targetDir string) error {
	rel, err := safeRelative(root, targetDir)
	if err != nil {
		return err
	}
	current, _ := filepath.Abs(root)
	if info, statErr := os.Lstat(current); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("managed root is a symbolic link: %s", current)
	}
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, os.ErrNotExist) {
			continue
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing OpenCode projection through symbolic link %s", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("OpenCode projection path component is not a directory: %s", current)
		}
	}
	return nil
}

func writeAtomic(path string, content []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".nexus-agent-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode.Perm()); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if info, statErr := os.Lstat(path); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to replace non-regular file %s", path)
		}
		backup := path + ".nexus-backup-" + newReceiptID()
		if err := os.Rename(path, backup); err != nil {
			return err
		}
		if err := os.Rename(tmpName, path); err != nil {
			_ = os.Rename(backup, path)
			return err
		}
		_ = os.Remove(backup)
		return nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}
	return os.Rename(tmpName, path)
}

func newReceiptID() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return "aprj_" + hex.EncodeToString(raw[:])
	}
	return fmt.Sprintf("aprj_%d", time.Now().UnixNano())
}
