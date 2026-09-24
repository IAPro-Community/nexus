package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/launchenv"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	projection "github.com/IAPro-Community/nexus/internal/core/provider/agentprojection"
	opencodeprojection "github.com/IAPro-Community/nexus/internal/core/provider/agentprojection/opencode"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type AgentProjectionPlanView struct {
	RuntimeID       string                  `json:"runtime_id"`
	RuntimeGenerationID string              `json:"runtime_generation_id"`
	AgentID         string                  `json:"agent_id"`
	RoleID          string                  `json:"role_id"`
	Provider        string                  `json:"provider"`
	Profile         string                  `json:"profile"`
	ProviderVersion string                  `json:"provider_version"`
	VersionSource   string                  `json:"version_source"`
	TargetKind      string                  `json:"target_kind"`
	CustomCommand   bool                    `json:"custom_command"`
	Capabilities    projection.Capabilities `json:"capabilities"`
	Plan            *projection.Plan        `json:"plan"`
}

type AgentProjectionApplyView struct {
	Plan    AgentProjectionPlanView `json:"plan"`
	Receipt *projection.Receipt     `json:"receipt"`
}

type agentProjectionContext struct {
	st          *store.Store
	generation  store.RuntimeGeneration
	agent       store.Agent
	project     store.Project
	config      AgentConfig
	role        ExecutionRole
	mode        AgentMode
	providerHome string
	version     string
	versionSource string
	targetKind  string
	custom      bool
	adapter     projection.Adapter
}

// PlanRuntimeAgentProjection computes the native-agent projection for the
// runtime's current ExecutionRole without mutating filesystem or durable state.
func (n *Nexus) PlanRuntimeAgentProjection(ctx context.Context, runtimeID string) (*AgentProjectionPlanView, error) {
	resolved, err := n.resolveAgentProjectionContext(ctx, runtimeID)
	if err != nil {
		return nil, err
	}

	req := resolved.projectionRequest()
	plan, err := resolved.adapter.Plan(ctx, req)
	if err != nil {
		return nil, err
	}
	caps := resolved.adapter.Capabilities(ctx, resolved.version)

	return &AgentProjectionPlanView{
		RuntimeID:            runtimeID,
		RuntimeGenerationID: resolved.generation.ID,
		AgentID:              resolved.agent.ID,
		RoleID:               resolved.role.ID,
		Provider:             resolved.generation.Provider,
		Profile:              resolved.generation.Profile,
		ProviderVersion:      resolved.version,
		VersionSource:        resolved.versionSource,
		TargetKind:           resolved.targetKind,
		CustomCommand:        resolved.custom,
		Capabilities:         caps,
		Plan:                 plan,
	}, nil
}

// ApplyRuntimeAgentProjection materializes the native agent into the isolated
// or explicitly shared target profile. It links the receipt to the runtime role
// but deliberately does not claim the native agent is active in the current
// provider session.
func (n *Nexus) ApplyRuntimeAgentProjection(ctx context.Context, runtimeID, activatedBy string) (*AgentProjectionApplyView, error) {
	resolved, err := n.resolveAgentProjectionContext(ctx, runtimeID)
	if err != nil {
		return nil, err
	}
	req := resolved.projectionRequest()
	plan, err := resolved.adapter.Plan(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(plan.Conflicts) > 0 {
		return nil, fmt.Errorf("agent projection has %d conflict(s); resolve them before apply", len(plan.Conflicts))
	}
	receipt, err := resolved.adapter.Apply(ctx, plan)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(receipt)
	if err != nil {
		_ = resolved.adapter.Remove(context.Background(), receipt)
		return nil, fmt.Errorf("serialize projection receipt: %w", err)
	}
	record := store.AgentProjectionReceiptRecord{
		ID:                  receipt.ID,
		RuntimeGenerationID: resolved.generation.ID,
		AgentID:             resolved.agent.ID,
		Provider:            receipt.Provider,
		Profile:             receipt.Profile,
		RoleID:              receipt.RoleID,
		Scope:               string(receipt.Scope),
		Strategy:            string(receipt.Strategy),
		ManagedRoot:         receipt.ManagedRoot,
		RendererVersion:     receipt.RendererVersion,
		SourceHash:          receipt.SourceHash,
		PayloadJSON:         string(payload),
		AppliedAt:           receipt.AppliedAt,
	}
	if err := resolved.st.SaveAgentProjectionReceipt(record); err != nil {
		_ = resolved.adapter.Remove(context.Background(), receipt)
		return nil, fmt.Errorf("persist projection receipt: %w", err)
	}

	// Ensure an explicit durable runtime role exists before attaching the
	// projection. This remains NEXUS_OVERLAY/BEST_EFFORT until activation and
	// permission probes prove stronger semantics.
	roleView, err := n.SetRuntimeExecutionRole(ctx, runtimeID, resolved.role.ID, strings.TrimSpace(activatedBy))
	if err != nil {
		_ = resolved.adapter.Remove(context.Background(), receipt)
		return nil, fmt.Errorf("persist execution role before projection link: %w", err)
	}
	if err := resolved.st.AttachRuntimeExecutionRoleProjection(roleView.Binding.RuntimeGenerationID, receipt.ID); err != nil {
		_ = resolved.adapter.Remove(context.Background(), receipt)
		return nil, fmt.Errorf("attach projection receipt to execution role: %w", err)
	}

	caps := resolved.adapter.Capabilities(ctx, resolved.version)
	return &AgentProjectionApplyView{
		Plan: AgentProjectionPlanView{
			RuntimeID:            runtimeID,
			RuntimeGenerationID: resolved.generation.ID,
			AgentID:              resolved.agent.ID,
			RoleID:               resolved.role.ID,
			Provider:             resolved.generation.Provider,
			Profile:              resolved.generation.Profile,
			ProviderVersion:      resolved.version,
			VersionSource:        resolved.versionSource,
			TargetKind:           resolved.targetKind,
			CustomCommand:        resolved.custom,
			Capabilities:         caps,
			Plan:                 plan,
		},
		Receipt: receipt,
	}, nil
}

func (n *Nexus) VerifyAgentProjection(ctx context.Context, receiptID string) (*projection.Verification, error) {
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	record, err := st.AgentProjectionReceipt(strings.TrimSpace(receiptID))
	if err != nil {
		return nil, err
	}
	if record.RemovedAt != nil {
		return nil, fmt.Errorf("projection receipt %s was removed", receiptID)
	}
	receipt, err := projectionReceiptFromRecord(record)
	if err != nil {
		return nil, err
	}
	adapter, err := agentProjectionAdapter(record.Provider)
	if err != nil {
		return nil, err
	}
	verification, err := adapter.Verify(ctx, receipt)
	if err != nil {
		return nil, err
	}
	if verification.Verified {
		now := time.Now().UTC()
		receipt.VerifiedAt = &now
		payload, _ := json.Marshal(receipt)
		if err := st.MarkAgentProjectionVerified(receipt.ID, now, string(payload)); err != nil {
			return nil, err
		}
	}
	return verification, nil
}

func (n *Nexus) RemoveAgentProjection(ctx context.Context, receiptID string) error {
	st, err := n.OpenProject()
	if err != nil {
		return err
	}
	record, err := st.AgentProjectionReceipt(strings.TrimSpace(receiptID))
	if err != nil {
		return err
	}
	if record.RemovedAt != nil {
		return nil
	}
	receipt, err := projectionReceiptFromRecord(record)
	if err != nil {
		return err
	}
	adapter, err := agentProjectionAdapter(record.Provider)
	if err != nil {
		return err
	}
	if err := adapter.Remove(ctx, receipt); err != nil {
		return err
	}
	return st.MarkAgentProjectionRemoved(receipt.ID, time.Now().UTC())
}

func (n *Nexus) resolveAgentProjectionContext(ctx context.Context, runtimeID string) (*agentProjectionContext, error) {
	runtimeID = strings.TrimSpace(runtimeID)
	if runtimeID == "" {
		return nil, errors.New("runtime_id is required")
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	generation, err := st.GenerationByRuntimeID(runtimeID)
	if err != nil {
		return nil, err
	}
	agent, err := st.GetAgent(generation.AgentID, "")
	if err != nil {
		return nil, err
	}
	project, err := st.GetProject(agent.ProjectID)
	if err != nil {
		return nil, err
	}

	cfg := AgentConfig{Provider: generation.Provider, Profile: generation.Profile}
	if generation.RevisionID != "" {
		if revision, revErr := st.GetRevision(generation.RevisionID); revErr == nil {
			if parsed, parseErr := ParseAgentConfig(revision.Config); parseErr == nil {
				cfg = NormalizeAgentSpec(agent, parsed)
			}
		}
	}
	if cfg.Provider == "" {
		cfg.Provider = generation.Provider
	}
	if cfg.Profile == "" {
		cfg.Profile = generation.Profile
	}
	if !strings.EqualFold(cfg.Provider, generation.Provider) {
		return nil, fmt.Errorf("runtime provider %s differs from revision provider %s", generation.Provider, cfg.Provider)
	}

	roleView, err := n.EffectiveRuntimeExecutionRole(runtimeID)
	if err != nil {
		return nil, err
	}
	role := roleView.Role
	mode, ok := AgentModeForID(role.BaseAgentModeID)
	if !ok {
		return nil, fmt.Errorf("execution role %s references unknown AgentMode %s", role.ID, role.BaseAgentModeID)
	}

	adapter, err := agentProjectionAdapter(generation.Provider)
	if err != nil {
		return nil, err
	}

	custom := strings.EqualFold(agentModeStringOption(cfg.Options, "execution_adapter"), "command_template")
	targetKind := strings.ToLower(agentModeStringOption(cfg.Options, "target_kind"))
	if targetKind == "" {
		if custom {
			targetKind = "custom"
		} else {
			targetKind = "local"
		}
	}

	providerHome := ""
	if custom {
		providerHome = agentModeStringOption(cfg.Options, "target_profile_home")
		if providerHome == "" {
			return nil, fmt.Errorf("custom %s target has no shared target_profile_home; native Agent Projection is unavailable, but Nexus overlay execution remains supported", targetKind)
		}
		if !filepath.IsAbs(providerHome) {
			return nil, fmt.Errorf("custom target_profile_home must be an absolute host-visible path")
		}
		info, statErr := os.Lstat(providerHome)
		if statErr != nil {
			return nil, fmt.Errorf("custom target_profile_home must already exist and be host-visible: %w", statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("custom target_profile_home must be a real directory, not a symlink or file")
		}
	} else {
		providerHome, err = coreconfig.ProfileHome(generation.Provider, generation.Profile)
		if err != nil {
			return nil, err
		}
	}

	version, versionSource, err := n.projectionProviderVersion(ctx, cfg, generation, project, providerHome, custom)
	if err != nil {
		return nil, err
	}

	return &agentProjectionContext{
		st: st,
		generation: generation,
		agent: agent,
		project: project,
		config: cfg,
		role: role,
		mode: mode,
		providerHome: providerHome,
		version: version,
		versionSource: versionSource,
		targetKind: targetKind,
		custom: custom,
		adapter: adapter,
	}, nil
}

func (n *Nexus) projectionProviderVersion(ctx context.Context, cfg AgentConfig, generation store.RuntimeGeneration, project store.Project, providerHome string, custom bool) (string, string, error) {
	if custom {
		if declared := agentModeStringOption(cfg.Options, "target_provider_version"); declared != "" {
			return declared, "DECLARED", nil
		}
		probeTemplate := agentModeStringOption(cfg.Options, "target_probe_template")
		if probeTemplate == "" {
			return "", "", fmt.Errorf("custom target provider version is unknown; configure target_probe_template (preferred) or target_provider_version before native Agent Projection")
		}
		probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		bin, args, err := launcher.ResolveCommandTemplateContext(probeTemplate, launcher.CommandTemplateContext{
			Cwd:         project.CanonicalPath,
			Args:        []string{"--version"},
			Provider:    generation.Provider,
			Profile:     generation.Profile,
			ProfileHome: providerHome,
		})
		if err != nil {
			return "", "", fmt.Errorf("resolve custom target probe: %w", err)
		}
		env := launchenv.Merge(os.Environ(), cfg.Environment, nil)
		out, err := nexusruntime.RunCommandCapture(probeCtx, bin, args, env, project.CanonicalPath)
		if err != nil {
			return "", "", fmt.Errorf("custom target provider version probe failed: %w (%s)", err, strings.TrimSpace(out))
		}
		version := firstNonEmptyLine(out)
		if version == "" {
			return "", "", errors.New("custom target provider version probe returned empty output")
		}
		return version, "TARGET_PROBE", nil
	}

	driver, err := n.controlDrivers().Get(generation.Provider)
	if err != nil {
		return "", "", err
	}
	detection, err := driver.Detect(ctx)
	if err != nil {
		return "", "", err
	}
	if !detection.Installed || strings.TrimSpace(detection.Version) == "" {
		return "", "", fmt.Errorf("%s provider version is not locally detectable; native Agent Projection remains unavailable", generation.Provider)
	}
	return detection.Version, "NEXUS_DRIVER", nil
}

func (c *agentProjectionContext) projectionRequest() projection.Request {
	spec := c.mode.AgentSpec
	if c.role.ID == "build" && c.config.AgentSpec.HasCustomBehavior() {
		spec = c.config.AgentSpec
	}
	instructions := append([]string(nil), spec.Instructions...)
	if persistent := strings.TrimSpace(c.config.AgentSpec.Role); persistent != "" && persistent != strings.TrimSpace(spec.Role) {
		instructions = append([]string{fmt.Sprintf("Persistent Nexus Agent specialization: %s.", persistent)}, instructions...)
	}
	return projection.Request{
		Provider:         c.generation.Provider,
		Profile:          c.generation.Profile,
		ProviderVersion:  c.version,
		ProviderHome:     c.providerHome,
		Workspace:        c.project.CanonicalPath,
		Scope:            projection.ScopeNexusProfile,
		RoleID:           c.role.ID,
		Role:             spec.Role,
		AccessIntent:     string(c.role.AccessIntent),
		Instructions:     instructions,
		Responsibilities: append([]string(nil), spec.Responsibilities...),
		Capabilities:     append([]string(nil), spec.Capabilities...),
		Constraints:      append([]string(nil), spec.Constraints...),
		Model:            c.config.Model,
	}
}

func agentProjectionAdapter(provider string) (projection.Adapter, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "opencode":
		return opencodeprojection.New(), nil
	default:
		return nil, fmt.Errorf("native Agent Projection is not implemented for provider %q in Nexus V1", provider)
	}
}

func projectionReceiptFromRecord(record store.AgentProjectionReceiptRecord) (*projection.Receipt, error) {
	var receipt projection.Receipt
	if err := json.Unmarshal([]byte(record.PayloadJSON), &receipt); err != nil {
		return nil, fmt.Errorf("decode projection receipt %s: %w", record.ID, err)
	}
	if receipt.ID != record.ID || receipt.Provider != record.Provider || receipt.ManagedRoot != record.ManagedRoot {
		return nil, fmt.Errorf("projection receipt %s durable metadata mismatch", record.ID)
	}
	return &receipt, nil
}

func firstNonEmptyLine(value string) string {
	for _, line := range strings.Split(value, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// Keep intelligence imported in this file's compile surface through the
// persistent AgentSpec contract; projection deliberately does not create a
// second specialization model.
var _ intelligence.AgentSpec
