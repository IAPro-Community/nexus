package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/launchenv"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	projection "github.com/IAPro-Community/nexus/internal/core/provider/agentprojection"
	opencodeprojection "github.com/IAPro-Community/nexus/internal/core/provider/agentprojection/opencode"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

type AgentProjectionPlanView struct {
	RuntimeID           string                  `json:"runtime_id"`
	RuntimeGenerationID string                  `json:"runtime_generation_id"`
	AgentID             string                  `json:"agent_id"`
	RoleID              string                  `json:"role_id"`
	Provider            string                  `json:"provider"`
	Profile             string                  `json:"profile"`
	ProviderVersion     string                  `json:"provider_version"`
	VersionSource       string                  `json:"version_source"`
	TargetKind          string                  `json:"target_kind"`
	TargetWorkspace     string                  `json:"target_workspace"`
	CustomCommand       bool                    `json:"custom_command"`
	Capabilities        projection.Capabilities `json:"capabilities"`
	Plan                *projection.Plan        `json:"plan"`
}

type AgentProjectionDiscovery struct {
	Provider        string `json:"provider"`
	ProviderVersion string `json:"provider_version"`
	TargetKind      string `json:"target_kind"`
	AgentName       string `json:"agent_name"`
	Discovered      bool   `json:"discovered"`
	Status          string `json:"status"`
	Evidence        string `json:"evidence,omitempty"`
	Error           string `json:"error,omitempty"`
}

type AgentProjectionActivation struct {
	Provider          string `json:"provider"`
	ProviderVersion   string `json:"provider_version"`
	TargetKind        string `json:"target_kind"`
	AgentName         string `json:"agent_name"`
	Activated         bool   `json:"activated"`
	Status            string `json:"status"`
	Evidence          string `json:"evidence,omitempty"`
	Error             string `json:"error,omitempty"`
	ExternalExecution bool   `json:"external_execution"`
}

type AgentProjectionEnforcement struct {
	Provider          string `json:"provider"`
	ProviderVersion   string `json:"provider_version"`
	TargetKind        string `json:"target_kind"`
	AgentName         string `json:"agent_name"`
	Enforced          bool   `json:"enforced"`
	Status            string `json:"status"`
	Tool              string `json:"tool,omitempty"`
	CanaryAbsent      bool   `json:"canary_absent"`
	Evidence          string `json:"evidence,omitempty"`
	Error             string `json:"error,omitempty"`
	ExternalExecution bool   `json:"external_execution"`
}

type AgentProjectionApplyView struct {
	Plan      AgentProjectionPlanView   `json:"plan"`
	Receipt   *projection.Receipt       `json:"receipt"`
	Discovery *AgentProjectionDiscovery `json:"discovery,omitempty"`
}

type agentProjectionContext struct {
	st              *store.Store
	generation      store.RuntimeGeneration
	agent           store.Agent
	project         store.Project
	config          AgentConfig
	role            ExecutionRole
	mode            AgentMode
	providerHome    string
	version         string
	versionSource   string
	targetKind      string
	targetWorkspace string
	custom          bool
	adapter         projection.Adapter
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
		RuntimeID:           runtimeID,
		RuntimeGenerationID: resolved.generation.ID,
		AgentID:             resolved.agent.ID,
		RoleID:              resolved.role.ID,
		Provider:            resolved.generation.Provider,
		Profile:             resolved.generation.Profile,
		ProviderVersion:     resolved.version,
		VersionSource:       resolved.versionSource,
		TargetKind:          resolved.targetKind,
		TargetWorkspace:     resolved.targetWorkspace,
		CustomCommand:       resolved.custom,
		Capabilities:        caps,
		Plan:                plan,
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
	rollbackPersistedProjection := func() {
		_ = resolved.adapter.Remove(context.Background(), receipt)
		_ = resolved.st.MarkAgentProjectionRemoved(receipt.ID, time.Now().UTC())
	}

	// Ensure an explicit durable runtime role exists before attaching the
	// projection. This remains NEXUS_OVERLAY/BEST_EFFORT until activation and
	// permission probes prove stronger semantics.
	roleView, err := n.SetRuntimeExecutionRole(ctx, runtimeID, resolved.role.ID, strings.TrimSpace(activatedBy))
	if err != nil {
		rollbackPersistedProjection()
		return nil, fmt.Errorf("persist execution role before projection link: %w", err)
	}
	if err := resolved.st.AttachRuntimeExecutionRoleProjection(roleView.Binding.RuntimeGenerationID, receipt.ID); err != nil {
		rollbackPersistedProjection()
		return nil, fmt.Errorf("attach projection receipt to execution role: %w", err)
	}

	caps := resolved.adapter.Capabilities(ctx, resolved.version)
	discovery := n.probeAgentProjectionDiscovery(ctx, resolved)
	return &AgentProjectionApplyView{
		Plan: AgentProjectionPlanView{
			RuntimeID:           runtimeID,
			RuntimeGenerationID: resolved.generation.ID,
			AgentID:             resolved.agent.ID,
			RoleID:              resolved.role.ID,
			Provider:            resolved.generation.Provider,
			Profile:             resolved.generation.Profile,
			ProviderVersion:     resolved.version,
			VersionSource:       resolved.versionSource,
			TargetKind:          resolved.targetKind,
			TargetWorkspace:     resolved.targetWorkspace,
			CustomCommand:       resolved.custom,
			Capabilities:        caps,
			Plan:                plan,
		},
		Receipt:   receipt,
		Discovery: discovery,
	}, nil
}

// ProbeRuntimeAgentProjectionDiscovery asks the provider target to enumerate
// native agents without invoking an LLM. Discovery is evidence that OpenCode
// sees the materialized file; it is not evidence that the current session has
// activated that agent or that its permissions were enforced.
func (n *Nexus) ProbeRuntimeAgentProjectionDiscovery(ctx context.Context, runtimeID string) (*AgentProjectionDiscovery, error) {
	resolved, err := n.resolveAgentProjectionContext(ctx, runtimeID)
	if err != nil {
		return nil, err
	}
	result := n.probeAgentProjectionDiscovery(ctx, resolved)
	if result.Error != "" {
		return result, fmt.Errorf("%s", result.Error)
	}
	return result, nil
}

func (n *Nexus) probeAgentProjectionDiscovery(ctx context.Context, resolved *agentProjectionContext) *AgentProjectionDiscovery {
	result := &AgentProjectionDiscovery{
		Provider:        resolved.generation.Provider,
		ProviderVersion: resolved.version,
		TargetKind:      resolved.targetKind,
		Status:          "NOT_VERIFIED",
	}
	if !strings.EqualFold(resolved.generation.Provider, "opencode") {
		result.Error = fmt.Sprintf("native discovery probe is not implemented for provider %q", resolved.generation.Provider)
		return result
	}
	agentName, err := opencodeprojection.AgentName(resolved.role.ID)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.AgentName = agentName
	discoveryArgs, err := opencodeprojection.DiscoveryArgs(resolved.version)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	var bin string
	var args []string
	var env []string
	var cwd string

	if resolved.custom {
		template := agentModeStringOption(resolved.config.Options, "target_probe_template")
		if template == "" {
			result.Error = "custom target has no target_probe_template; discovery cannot be proved"
			return result
		}
		bin, args, err = launcher.ResolveCommandTemplateContext(template, launcher.CommandTemplateContext{
			Cwd:         resolved.project.CanonicalPath,
			TargetCwd:   resolved.targetWorkspace,
			Args:        discoveryArgs,
			Provider:    resolved.generation.Provider,
			Profile:     resolved.generation.Profile,
			ProfileHome: resolved.providerHome,
		})
		env = launchenv.Merge(os.Environ(), resolved.config.Environment, nil)
		cwd = resolved.project.CanonicalPath
	} else {
		driver, driverErr := n.controlDrivers().Get(resolved.generation.Provider)
		if driverErr != nil {
			result.Error = driverErr.Error()
			return result
		}
		bin, args, env, err = driver.BuildCommand(probeCtx, model.Profile{
			Provider: resolved.generation.Provider,
			Name:     resolved.generation.Profile,
		}, discoveryArgs)
		cwd = resolved.project.CanonicalPath
	}
	if err != nil {
		result.Error = fmt.Sprintf("resolve discovery command: %v", err)
		return result
	}

	output, runErr := nexusruntime.RunCommandCapture(probeCtx, bin, args, env, cwd)
	if runErr != nil {
		result.Error = fmt.Sprintf("OpenCode native-agent discovery failed: %v", runErr)
		return result
	}
	if !containsAgentName(output, agentName) {
		result.Error = fmt.Sprintf("OpenCode did not list projected agent %q", agentName)
		return result
	}

	result.Discovered = true
	result.Status = "VERIFIED"
	result.Evidence = fmt.Sprintf("OpenCode listed native agent %q", agentName)
	return result
}

// ProbeRuntimeAgentProjectionActivation proves that OpenCode can launch the
// projected agent by selecting it explicitly in a fresh non-interactive run.
// This invokes the configured model and may consume quota/cost, so it is never
// called implicitly by Plan, Apply or Verify.
func (n *Nexus) ProbeRuntimeAgentProjectionActivation(ctx context.Context, runtimeID string, confirmExternalExecution bool) (*AgentProjectionActivation, error) {
	resolved, err := n.resolveAgentProjectionContext(ctx, runtimeID)
	if err != nil {
		return nil, err
	}
	result := &AgentProjectionActivation{
		Provider:          resolved.generation.Provider,
		ProviderVersion:   resolved.version,
		TargetKind:        resolved.targetKind,
		Status:            "NOT_VERIFIED",
		ExternalExecution: true,
	}
	if !confirmExternalExecution {
		result.Error = "activation probe invokes the configured provider model and may consume quota/cost; explicit confirmation is required"
		return result, fmt.Errorf("%s", result.Error)
	}
	if !strings.EqualFold(resolved.generation.Provider, "opencode") {
		result.Error = fmt.Sprintf("native activation probe is not implemented for provider %q", resolved.generation.Provider)
		return result, fmt.Errorf("%s", result.Error)
	}

	binding, err := resolved.st.RuntimeExecutionRole(resolved.generation.ID)
	if err != nil {
		result.Error = fmt.Sprintf("activation requires a persisted runtime execution role: %v", err)
		return result, fmt.Errorf("%s", result.Error)
	}
	if strings.TrimSpace(binding.ProjectionReceiptID) == "" {
		result.Error = "activation requires an applied native projection receipt"
		return result, fmt.Errorf("%s", result.Error)
	}
	record, err := resolved.st.AgentProjectionReceipt(binding.ProjectionReceiptID)
	if err != nil {
		result.Error = fmt.Sprintf("load active projection receipt: %v", err)
		return result, fmt.Errorf("%s", result.Error)
	}
	if record.RemovedAt != nil || record.RuntimeGenerationID != resolved.generation.ID || record.RoleID != resolved.role.ID {
		result.Error = "active projection receipt does not match the runtime generation and execution role"
		return result, fmt.Errorf("%s", result.Error)
	}
	verification, err := n.VerifyAgentProjection(ctx, record.ID)
	if err != nil {
		result.Error = fmt.Sprintf("verify projection before activation: %v", err)
		return result, fmt.Errorf("%s", result.Error)
	}
	if !verification.Verified {
		result.Error = "projection filesystem/ownership verification failed before activation"
		return result, fmt.Errorf("%s", result.Error)
	}
	discovery := n.probeAgentProjectionDiscovery(ctx, resolved)
	if discovery == nil || !discovery.Discovered {
		detail := "projected agent was not discovered by OpenCode"
		if discovery != nil && strings.TrimSpace(discovery.Error) != "" {
			detail = discovery.Error
		}
		result.Error = detail
		return result, fmt.Errorf("%s", result.Error)
	}

	agentName, err := opencodeprojection.AgentName(resolved.role.ID)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	result.AgentName = agentName
	activationMarker := fmt.Sprintf("NEXUS_ACTIVATION_OK_%d", time.Now().UTC().UnixNano())
	activationArgs, err := opencodeprojection.ActivationProbeArgs(resolved.version, agentName, activationMarker)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	probeCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	var bin string
	var args []string
	var env []string
	var cwd string
	if resolved.custom {
		template := agentModeStringOption(resolved.config.Options, "target_probe_template")
		if template == "" {
			result.Error = "custom target has no target_probe_template; activation cannot be proved"
			return result, fmt.Errorf("%s", result.Error)
		}
		bin, args, err = launcher.ResolveCommandTemplateContext(template, launcher.CommandTemplateContext{
			Cwd:         resolved.project.CanonicalPath,
			TargetCwd:   resolved.targetWorkspace,
			Args:        activationArgs,
			Provider:    resolved.generation.Provider,
			Profile:     resolved.generation.Profile,
			ProfileHome: resolved.providerHome,
		})
		env = launchenv.Merge(os.Environ(), resolved.config.Environment, nil)
		cwd = resolved.project.CanonicalPath
	} else {
		driver, driverErr := n.controlDrivers().Get(resolved.generation.Provider)
		if driverErr != nil {
			result.Error = driverErr.Error()
			return result, driverErr
		}
		bin, args, env, err = driver.BuildCommand(probeCtx, model.Profile{
			Provider: resolved.generation.Provider,
			Name:     resolved.generation.Profile,
		}, activationArgs)
		cwd = resolved.project.CanonicalPath
	}
	if err != nil {
		result.Error = fmt.Sprintf("resolve activation command: %v", err)
		return result, fmt.Errorf("%s", result.Error)
	}

	output, runErr := nexusruntime.RunCommandCapture(probeCtx, bin, args, env, cwd)
	if runErr != nil {
		result.Error = fmt.Sprintf("OpenCode native-agent activation failed: %v", runErr)
		return result, fmt.Errorf("%s", result.Error)
	}
	if !strings.Contains(output, activationMarker) {
		result.Error = "OpenCode run completed but did not return the activation marker from the projected agent"
		return result, fmt.Errorf("%s", result.Error)
	}

	if err := resolved.st.MarkRuntimeExecutionRoleActivation(
		resolved.generation.ID,
		record.ID,
		string(projection.ActivationNativeOnLaunch),
	); err != nil {
		result.Error = fmt.Sprintf("persist native activation evidence: %v", err)
		return result, fmt.Errorf("%s", result.Error)
	}
	result.Activated = true
	result.Status = "VERIFIED"
	result.Evidence = fmt.Sprintf("OpenCode launched projected native agent %q and returned the expected activation marker", agentName)
	return result, nil
}

// ProbeRuntimeAgentProjectionEnforcement proves native read-only enforcement
// independently from discovery and activation. It invokes the configured model
// and therefore always requires explicit confirmation.
func (n *Nexus) ProbeRuntimeAgentProjectionEnforcement(ctx context.Context, runtimeID string, confirmExternalExecution bool) (*AgentProjectionEnforcement, error) {
	resolved, err := n.resolveAgentProjectionContext(ctx, runtimeID)
	if err != nil {
		return nil, err
	}
	result := &AgentProjectionEnforcement{
		Provider:          resolved.generation.Provider,
		ProviderVersion:   resolved.version,
		TargetKind:        resolved.targetKind,
		Status:            "NOT_VERIFIED",
		ExternalExecution: true,
	}
	if !confirmExternalExecution {
		result.Error = "enforcement probe invokes the configured provider model and may consume quota/cost; explicit confirmation is required"
		return result, fmt.Errorf("%s", result.Error)
	}
	if resolved.role.AccessIntent != AccessReadOnly && resolved.role.AccessIntent != AccessVerifyOnly {
		result.Error = fmt.Sprintf("execution role %s is not a read-only/verify-only role", resolved.role.ID)
		return result, fmt.Errorf("%s", result.Error)
	}
	if !strings.EqualFold(resolved.generation.Provider, "opencode") {
		result.Error = fmt.Sprintf("native enforcement probe is not implemented for provider %q", resolved.generation.Provider)
		return result, fmt.Errorf("%s", result.Error)
	}

	binding, err := resolved.st.RuntimeExecutionRole(resolved.generation.ID)
	if err != nil {
		result.Error = fmt.Sprintf("enforcement requires a persisted runtime execution role: %v", err)
		return result, fmt.Errorf("%s", result.Error)
	}
	if strings.TrimSpace(binding.ProjectionReceiptID) == "" {
		result.Error = "enforcement requires an applied native projection receipt"
		return result, fmt.Errorf("%s", result.Error)
	}
	if binding.ActivationStrategy != string(projection.ActivationNativeOnLaunch) {
		result.Error = "enforcement requires verified native activation first"
		return result, fmt.Errorf("%s", result.Error)
	}
	record, err := resolved.st.AgentProjectionReceipt(binding.ProjectionReceiptID)
	if err != nil {
		result.Error = fmt.Sprintf("load active projection receipt: %v", err)
		return result, fmt.Errorf("%s", result.Error)
	}
	if record.RemovedAt != nil || record.RuntimeGenerationID != resolved.generation.ID || record.RoleID != resolved.role.ID {
		result.Error = "active projection receipt does not match the runtime generation and execution role"
		return result, fmt.Errorf("%s", result.Error)
	}
	verification, err := n.VerifyAgentProjection(ctx, record.ID)
	if err != nil || !verification.Verified {
		if err != nil {
			result.Error = fmt.Sprintf("verify projection before enforcement: %v", err)
		} else {
			result.Error = "projection filesystem/ownership verification failed before enforcement"
		}
		return result, fmt.Errorf("%s", result.Error)
	}

	agentName, err := opencodeprojection.AgentName(resolved.role.ID)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	result.AgentName = agentName

	permissionArgs, permissionArgsErr := opencodeprojection.EffectivePermissionProbeArgs(resolved.version, agentName)
	nativePermissionProved := false
	permissionProbeDetail := ""
	if permissionArgsErr == nil {
		permissionCtx, permissionCancel := context.WithTimeout(ctx, 8*time.Second)
		var permissionBin string
		var permissionCommandArgs []string
		var permissionEnv []string
		var permissionCwd string
		if resolved.custom {
			template := agentModeStringOption(resolved.config.Options, "target_probe_template")
			if template != "" {
				permissionBin, permissionCommandArgs, permissionArgsErr = launcher.ResolveCommandTemplateContext(template, launcher.CommandTemplateContext{
					Cwd:         resolved.project.CanonicalPath,
					TargetCwd:   resolved.targetWorkspace,
					Args:        permissionArgs,
					Provider:    resolved.generation.Provider,
					Profile:     resolved.generation.Profile,
					ProfileHome: resolved.providerHome,
				})
				permissionEnv = launchenv.Merge(os.Environ(), resolved.config.Environment, nil)
				permissionCwd = resolved.project.CanonicalPath
			} else {
				permissionArgsErr = errors.New("custom target has no target_probe_template")
			}
		} else {
			driver, driverErr := n.controlDrivers().Get(resolved.generation.Provider)
			if driverErr != nil {
				permissionArgsErr = driverErr
			} else {
				permissionBin, permissionCommandArgs, permissionEnv, permissionArgsErr = driver.BuildCommand(permissionCtx, model.Profile{
					Provider: resolved.generation.Provider,
					Name:     resolved.generation.Profile,
				}, permissionArgs)
				permissionCwd = resolved.project.CanonicalPath
			}
		}
		if permissionArgsErr == nil {
			permissionOutput, permissionRunErr := nexusruntime.RunCommandCapture(permissionCtx, permissionBin, permissionCommandArgs, permissionEnv, permissionCwd)
			if permissionRunErr == nil {
				nativePermissionProved = opencodeprojection.ParseEffectiveReadOnlyPermission(resolved.version, agentName, permissionOutput)
				if nativePermissionProved {
					permissionProbeDetail = "effective OpenCode agent permissions disable native file mutation tools"
				}
			}
		}
		permissionCancel()
	}

	probeDir, err := os.MkdirTemp(resolved.project.CanonicalPath, ".nexus-enforcement-probe-")
	if err != nil {
		result.Error = fmt.Sprintf("create enforcement probe directory: %v", err)
		return result, fmt.Errorf("%s", result.Error)
	}
	defer os.RemoveAll(probeDir)
	hostCanary := filepath.Join(probeDir, "canary.txt")
	targetRoot := strings.TrimSpace(resolved.targetWorkspace)
	if targetRoot == "" {
		targetRoot = resolved.project.CanonicalPath
	}
	targetCanary := joinProjectionTargetPath(targetRoot, filepath.Base(probeDir), "canary.txt")

	enforcementArgs, err := opencodeprojection.EnforcementProbeArgs(resolved.version, agentName, targetCanary)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	probeCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	var bin string
	var args []string
	var env []string
	var cwd string
	if resolved.custom {
		template := agentModeStringOption(resolved.config.Options, "target_probe_template")
		if template == "" {
			result.Error = "custom target has no target_probe_template; enforcement cannot be proved"
			return result, fmt.Errorf("%s", result.Error)
		}
		bin, args, err = launcher.ResolveCommandTemplateContext(template, launcher.CommandTemplateContext{
			Cwd:         resolved.project.CanonicalPath,
			TargetCwd:   resolved.targetWorkspace,
			Args:        enforcementArgs,
			Provider:    resolved.generation.Provider,
			Profile:     resolved.generation.Profile,
			ProfileHome: resolved.providerHome,
		})
		env = launchenv.Merge(os.Environ(), resolved.config.Environment, nil)
		cwd = resolved.project.CanonicalPath
	} else {
		driver, driverErr := n.controlDrivers().Get(resolved.generation.Provider)
		if driverErr != nil {
			result.Error = driverErr.Error()
			return result, driverErr
		}
		bin, args, env, err = driver.BuildCommand(probeCtx, model.Profile{
			Provider: resolved.generation.Provider,
			Name:     resolved.generation.Profile,
		}, enforcementArgs)
		cwd = resolved.project.CanonicalPath
	}
	if err != nil {
		result.Error = fmt.Sprintf("resolve enforcement command: %v", err)
		return result, fmt.Errorf("%s", result.Error)
	}

	output, runErr := nexusruntime.RunCommandCapture(probeCtx, bin, args, env, cwd)
	evidence := opencodeprojection.ParseEnforcementProbeOutput(output)
	result.Tool = evidence.Tool
	_, statErr := os.Stat(hostCanary)
	result.CanaryAbsent = errors.Is(statErr, os.ErrNotExist)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		result.Error = fmt.Sprintf("inspect enforcement canary: %v", statErr)
		return result, fmt.Errorf("%s", result.Error)
	}
	if evidence.WriteCompleted || !result.CanaryAbsent {
		result.Error = "projected read-only agent completed a write; enforcement certification failed"
		return result, fmt.Errorf("%s", result.Error)
	}
	if runErr != nil && !evidence.WriteDenied {
		result.Error = fmt.Sprintf("OpenCode enforcement probe did not complete successfully: %v", runErr)
		return result, fmt.Errorf("%s", result.Error)
	}
	if !evidence.WriteDenied && !nativePermissionProved {
		if runErr != nil {
			result.Error = fmt.Sprintf("OpenCode enforcement probe failed without provider-backed permission evidence: %v", runErr)
		} else if evidence.WriteAttempted {
			result.Error = "OpenCode write-tool event was not an explicit permission/policy denial"
		} else {
			result.Error = "OpenCode exposed no verifiable write denial and effective permission introspection did not prove mutation tools disabled"
		}
		return result, fmt.Errorf("%s", result.Error)
	}

	now := time.Now().UTC()
	persistedEvidence := ""
	if evidence.WriteDenied {
		persistedEvidence = fmt.Sprintf("OpenCode native agent %q rejected %s by permission policy; canary_absent=true", agentName, evidence.Tool)
	} else {
		persistedEvidence = fmt.Sprintf("OpenCode native agent %q effective permission probe confirmed mutation tools disabled; canary_absent=true", agentName)
		if permissionProbeDetail != "" {
			persistedEvidence = permissionProbeDetail + "; canary_absent=true"
		}
	}
	if err := resolved.st.MarkRuntimeExecutionRoleEnforced(
		resolved.generation.ID,
		record.ID,
		persistedEvidence,
		now,
	); err != nil {
		result.Error = fmt.Sprintf("persist native enforcement evidence: %v", err)
		return result, fmt.Errorf("%s", result.Error)
	}
	result.Enforced = true
	result.Status = "VERIFIED"
	result.Evidence = persistedEvidence
	return result, nil
}

func joinProjectionTargetPath(root string, elems ...string) string {
	root = strings.TrimRight(strings.ReplaceAll(strings.TrimSpace(root), "\\", "/"), "/")
	parts := []string{root}
	for _, elem := range elems {
		elem = strings.Trim(strings.ReplaceAll(elem, "\\", "/"), "/")
		if elem != "" {
			parts = append(parts, elem)
		}
	}
	return strings.Join(parts, "/")
}

func containsAgentName(output, name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	pattern := `(^|[^A-Za-z0-9_-])` + regexp.QuoteMeta(name) + `([^A-Za-z0-9_-]|$)`
	return regexp.MustCompile(pattern).FindStringIndex(output) != nil
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

	// A stale receipt must never remove files owned by a newer projection. The
	// runtime role is the durable ownership pointer for the active materialized
	// projection.
	binding, bindErr := st.RuntimeExecutionRole(record.RuntimeGenerationID)
	if bindErr != nil && bindErr != store.ErrNotFound {
		return bindErr
	}
	if bindErr == nil && strings.TrimSpace(binding.ProjectionReceiptID) != "" && binding.ProjectionReceiptID != record.ID {
		return fmt.Errorf("projection receipt %s is no longer current for runtime generation %s; active receipt is %s", record.ID, record.RuntimeGenerationID, binding.ProjectionReceiptID)
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
	now := time.Now().UTC()
	if err := st.MarkAgentProjectionRemoved(receipt.ID, now); err != nil {
		return err
	}
	if bindErr == nil && binding.ProjectionReceiptID == receipt.ID {
		if err := st.DetachRuntimeExecutionRoleProjection(record.RuntimeGenerationID, receipt.ID); err != nil && err != store.ErrNotFound {
			return fmt.Errorf("detach projection receipt from execution role: %w", err)
		}
	}
	return nil
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
	targetWorkspace := agentModeStringOption(cfg.Options, "target_workspace")
	if targetWorkspace == "" {
		targetWorkspace = project.CanonicalPath
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
		st:              st,
		generation:      generation,
		agent:           agent,
		project:         project,
		config:          cfg,
		role:            role,
		mode:            mode,
		providerHome:    providerHome,
		version:         version,
		versionSource:   versionSource,
		targetKind:      targetKind,
		targetWorkspace: targetWorkspace,
		custom:          custom,
		adapter:         adapter,
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
			TargetCwd:   agentModeStringOption(cfg.Options, "target_workspace"),
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
			return "", "", fmt.Errorf("custom target provider version probe failed: %w", err)
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
