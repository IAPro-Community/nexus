package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/classifier"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/nexus/autonomyguard"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	profilepkg "github.com/IAPro-Community/nexus/internal/profile"
)

type agentPromptExecution struct {
	RuntimeID string
	Output    string
	Failure   model.Failure
}

type agentPromptPolicy struct {
	Contract     runner.AutonomyContract
	Review       bool
	MissionRunID string
	PackageID    string
}

// executeAgentPrompt runs one autonomous, headless provider turn under a real
// persistent Agent identity and real RuntimeGeneration. It deliberately starts
// a fresh provider process; durable continuity for autonomous work is the code
// worktree + immutable prompt/evidence, not a fabricated cross-provider resume.
func (n *Nexus) executeAgentPrompt(ctx context.Context, agentID, workspace, prompt string, policy agentPromptPolicy) (*agentPromptExecution, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("agent prompt is required")
	}
	st, err := n.OpenProject()
	if err != nil {
		return nil, err
	}
	agent, err := st.GetAgent(agentID, "")
	if err != nil {
		return nil, err
	}

	// Record invocation start for observability.
	startedAt := time.Now().UTC()
	goalID := fmt.Sprintf("goal_%d", startedAt.UnixNano())
	invocation := AgentInvocation{
		GoalID:    goalID,
		AgentID:   agent.ID,
		ProjectID: agent.ProjectID,
		Status:    AgentInvocationRunning,
		StartedAt: startedAt,
	}
	_ = RecordAgentInvocation(st, invocation)

	cfg, err := currentAgentConfig(st, agent)
	if err != nil {
		return nil, err
	}
	if cfg.Provider == "" || cfg.Profile == "" {
		return nil, fmt.Errorf("agent %s has no persisted provider/profile", agentID)
	}
	if strings.TrimSpace(workspace) == "" {
		project, err := st.GetProject(agent.ProjectID)
		if err != nil {
			return nil, err
		}
		workspace, err = n.resolveExecutionWorkspace(ctx, project, agent, cfg)
		if err != nil {
			return nil, err
		}
	}

	d, err := n.controlDrivers().Get(cfg.Provider)
	if err != nil {
		return nil, err
	}
	profile := model.Profile{Provider: cfg.Provider, Name: cfg.Profile}
	caps := d.EffectiveCaps(ctx, profile)
	if caps.Headless.Status != driver.CapabilitySupported || caps.SubmitPrompt.Status != driver.CapabilitySupported {
		return nil, fmt.Errorf("provider %s:%s cannot execute autonomous prompt: headless=%s submit_prompt=%s", cfg.Provider, cfg.Profile, caps.Headless.Status, caps.SubmitPrompt.Status)
	}
	if cfg.AgentSpec.HasCustomBehavior() {
		compiledContext, compileErr := intelligence.NewNexusEngine(nil).CompileExecutionContext(ctx, intelligence.ExecutionContextRequest{
			Agent:   cfg.AgentSpec,
			Task:    intelligence.WorkPackageContext{Title: "Direct Agent request", Goal: prompt, Role: cfg.AgentSpec.Role},
			Runtime: intelligence.RuntimeConstraints{Provider: cfg.Provider, Model: cfg.Model, Workspace: workspace, Isolation: cfg.Isolation, Capabilities: []string{"headless", "submit_prompt"}},
		})
		if compileErr != nil {
			return nil, fmt.Errorf("compile agent execution context: %w", compileErr)
		}
		prompt = compiledContext.SystemInstructions + "\n\n" + compiledContext.TaskInstructions
	}
	kickoffArgs, err := d.BuildKickoffArgs(ctx, profile, prompt)
	if err != nil {
		return nil, fmt.Errorf("build provider kickoff args: %w", err)
	}
	kickoffArgs, err = missionProviderArgs(cfg.Provider, kickoffArgs, policy)
	if err != nil {
		return nil, err
	}

	// Do not silently steal an unrelated live terminal session. Mission agents
	// are expected to be dedicated or stopped between headless turns.
	if gen, gerr := st.CurrentGeneration(agent.ID); gerr == nil && gen.RuntimeID != "" && n.runtimeAlive(gen.RuntimeID) {
		return nil, fmt.Errorf("agent %s already has a live runtime %s; take control or stop it before autonomous execution", agent.ID, gen.RuntimeID)
	}

	var guardDir string
	pathPrepend := []string(nil)
	if policy.Contract.DisallowDestructiveGit || !policy.Contract.AllowGitPush || !policy.Contract.AllowDeploy || !policy.Contract.AllowExternalNetwork || !policy.Contract.AllowSecretAccess || !policy.Contract.AllowPaidServices {
		guardDir, err = os.MkdirTemp("", "nexus-autonomy-guards-*")
		if err != nil {
			return nil, fmt.Errorf("create autonomy command guards: %w", err)
		}
		defer os.RemoveAll(guardDir)
		if _, err := autonomyguard.WriteCommandGuards(guardDir, autonomyguard.Policy{
			DisallowDestructiveGit: policy.Contract.DisallowDestructiveGit,
			AllowGitPush:           policy.Contract.AllowGitPush,
			AllowDeploy:            policy.Contract.AllowDeploy,
			AllowExternalNetwork:   policy.Contract.AllowExternalNetwork,
			AllowSecretAccess:      policy.Contract.AllowSecretAccess,
			AllowPaidServices:      policy.Contract.AllowPaidServices,
		}); err != nil {
			return nil, fmt.Errorf("prepare autonomy command guards: %w", err)
		}
		pathPrepend = []string{guardDir}
	}

	sess, err := n.launcher.Launch(ctx, launcher.LaunchOptions{
		AgentID:     agent.ID,
		ProjectID:   agent.ProjectID,
		ProviderID:  cfg.Provider,
		ProfileID:   cfg.Profile,
		Workspace:   workspace,
		Args:        kickoffArgs,
		Model:       cfg.Model,
		Environment: cfg.Environment,
		Isolation:   cfg.Isolation,
		Options:     cfg.Options,
		PathPrepend: pathPrepend,
	})
	if err != nil {
		return nil, fmt.Errorf("launch autonomous agent runtime: %w", err)
	}

	revisionID := agent.CurrentRevisionID
	if revisionID == "" {
		rev, revErr := st.AddRevision(agent.ID, cfg.ConfigJSON())
		if revErr != nil {
			n.stopRuntime(sess.RuntimeID)
			return nil, revErr
		}
		revisionID = rev.ID
	}
	generation, err := st.AddGeneration(store.RuntimeGeneration{
		AgentID:    agent.ID,
		RevisionID: revisionID,
		RuntimeID:  sess.RuntimeID,
		Provider:   cfg.Provider,
		Profile:    cfg.Profile,
		Continuity: "NEW_SESSION",
		StartedAt:  time.Now().UTC(),
		State:      "RUNNING",
	})
	if err != nil {
		n.stopRuntime(sess.RuntimeID)
		return nil, fmt.Errorf("persist autonomous runtime generation: %w", err)
	}

	agent.Status = store.AgentWorking
	agent.ContinuityStatus = "NEW_SESSION"
	now := time.Now().UTC()
	agent.LastStartedAt = &now
	_ = st.UpdateAgent(agent)
	n.notifyRuntimeChanged(agent.ID, "", sess.RuntimeID, cfg.Provider, cfg.Profile, "NEW_SESSION")
	n.notifyContinuity(agent.ID, "NEW_SESSION")
	n.notifyAgentState(agent.ID, store.AgentWorking)

	output, runErr := n.captureRuntimeOutput(ctx, sess.RuntimeID)
	stoppedAt := time.Now().UTC()
	_ = st.StopGeneration(generation.ID, stoppedAt)
	agent.Status = store.AgentStopped
	if runErr != nil {
		n.stopRuntime(sess.RuntimeID)
		agent.Status = store.AgentFailed
	}
	_ = st.UpdateAgent(agent)
	n.notifyAgentState(agent.ID, agent.Status)

	// Record invocation completion for observability.
	finishedAt := time.Now().UTC()
	invocation.FinishedAt = &finishedAt
	status := "COMPLETED"
	if runErr != nil {
		invocation.Status = AgentInvocationFailed
		invocation.Errors = []string{runErr.Error()}
		status = "FAILED"
	} else {
		invocation.Status = AgentInvocationCompleted
	}
	_ = RecordAgentInvocation(st, invocation)

	// Cost Intelligence is observational: duration and attribution are known
	// here, while token/cost fields remain NULL until a trustworthy provider
	// source reports them. UNKNOWN must never be serialized as zero.
	accountScope := ""
	if info := profilepkg.GetAccountInfo(cfg.Provider, cfg.Profile); info.AccountScope.Verifiable() {
		accountScope = info.AccountScope.Key()
	}
	_, _ = st.RecordCostObservation(store.CostObservation{
		ProjectID:           agent.ProjectID,
		MissionRunID:        policy.MissionRunID,
		PackageID:           policy.PackageID,
		InvocationID:        goalID,
		AgentID:             agent.ID,
		RuntimeGenerationID: generation.ID,
		RuntimeID:           sess.RuntimeID,
		Provider:            cfg.Provider,
		Profile:             cfg.Profile,
		Model:               cfg.Model,
		Branch:              executionBranch(ctx, workspace),
		WorkspaceKind:       cfg.Isolation,
		DurationMS:          max(finishedAt.Sub(startedAt).Milliseconds(), 0),
		CostKind:            store.CostKindUnknown,
		Source:              "runtime_observation",
		Status:              status,
		AccountScope:        accountScope,
		ObservedAt:          finishedAt,
	})

	if runErr != nil {
		failure := classifier.Classify(runErr, output)
		return &agentPromptExecution{RuntimeID: sess.RuntimeID, Output: output, Failure: failure},
			fmt.Errorf("autonomous runtime %s failed: %w", sess.RuntimeID, runErr)
	}
	return &agentPromptExecution{RuntimeID: sess.RuntimeID, Output: output}, nil
}

func executionBranch(ctx context.Context, workspace string) string {
	if strings.TrimSpace(workspace) == "" {
		return ""
	}
	branchCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(branchCtx, "git", "-C", workspace, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return ""
	}
	return branch
}

func missionProviderArgs(provider string, args []string, policy agentPromptPolicy) ([]string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	out := append([]string(nil), args...)
	providerSpec, ok := spec.Get(provider)
	if !ok {
		return nil, fmt.Errorf("provider %s is not approved for autonomous coding", provider)
	}
	if policy.Review {
		if len(providerSpec.ReviewMode) == 0 {
			return nil, fmt.Errorf("provider %s has no verified read-only headless review mode", provider)
		}
		if provider == "codex" && len(out) > 1 && out[0] == "exec" {
			return append(append([]string(nil), providerSpec.ReviewMode...), out...), nil
		}
		return append(out, providerSpec.ReviewMode...), nil
	}

	// A coding Mission must not hang waiting for an approval prompt. Auto-approval
	// is an explicit contract decision and is restricted by Nexus to the Agent's
	// isolated workspace plus prompt-level no-push/no-deploy boundaries.
	if !policy.Contract.AllowToolAutoApproval {
		return nil, fmt.Errorf("%s autonomous coding requires allow_tool_auto_approval", provider)
	}
	switch provider {
	case "agy", "claude", "codex", "gemini":
		// Codex parses exec options before its positional prompt. Appending
		// approval flags after the prompt makes current Codex releases report
		// "unexpected argument" and prevents the autonomous runtime from
		// starting. Keep the prompt last while preserving the public spec.
		if provider == "codex" && len(out) > 1 && out[0] == "exec" {
			return append(append([]string(nil), providerSpec.AutoApproval...), out...), nil
		}
		return append(out, providerSpec.AutoApproval...), nil
	case "opencode":
		if len(out) > 0 && out[0] == "run" {
			out = append(append([]string{"run"}, providerSpec.AutoApproval...), out[1:]...)
		} else {
			out = append(append([]string(nil), providerSpec.AutoApproval...), out...)
		}
		return out, nil
	case "cursor":
		return append(out, providerSpec.AutoApproval...), nil
	default:
		return nil, fmt.Errorf("provider %s is not approved for autonomous coding", provider)
	}
}

func (n *Nexus) captureRuntimeOutput(ctx context.Context, runtimeID string) (string, error) {
	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		return "", fmt.Errorf("attach to runtime output: %w", err)
	}
	defer client.Close()

	resp, err := client.Send(protocol.CmdAttach, nil)
	if err != nil {
		return "", fmt.Errorf("attach runtime stream: %w", err)
	}
	var history string
	_ = json.Unmarshal(resp.Data, &history)
	_ = client.ClearDeadline()

	type readResult struct {
		data []byte
		err  error
	}
	ch := make(chan readResult, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		data, readErr := io.ReadAll(client.Reader())
		ch <- readResult{data: data, err: readErr}
	}()

	select {
	case <-ctx.Done():
		_ = client.Close()
		// Wait for the reader goroutine to finish after connection close.
		wg.Wait()
		return history, ctx.Err()
	case rr := <-ch:
		output := history + string(rr.data)
		if rr.err != nil && !strings.Contains(strings.ToLower(rr.err.Error()), "closed") {
			return output, rr.err
		}
		// The host closes attached clients after process termination. Registry is
		// persisted cross-process, so reload and reject provider process failures.
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if sess, ok := n.runtimeRegistry().Get(runtimeID); ok {
				switch sess.State {
				case registry.StateFailed:
					return output, fmt.Errorf("provider process exited with failure")
				case registry.StateStopped:
					return output, nil
				}
			}
			time.Sleep(25 * time.Millisecond)
		}
		return output, nil
	}
}

// workspaceFingerprint is used to prove an independent reviewer did not alter
// the implementation workspace. It hashes tracked diffs and untracked file
// contents, not merely filenames.
func workspaceFingerprint(ctx context.Context, workspace string) (string, error) {
	if strings.TrimSpace(workspace) == "" {
		return "", fmt.Errorf("workspace is required")
	}
	if _, err := exec.LookPath("git"); err != nil {
		return "", err
	}
	rootOut, err := exec.CommandContext(ctx, "git", "-C", workspace, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	root := strings.TrimSpace(string(rootOut))
	h := sha256.New()

	for _, args := range [][]string{
		{"-C", root, "status", "--porcelain=v2", "-z"},
		{"-C", root, "diff", "--binary", "--no-ext-diff"},
		{"-C", root, "diff", "--binary", "--cached", "--no-ext-diff"},
	} {
		out, cmdErr := exec.CommandContext(ctx, "git", args...).Output()
		if cmdErr != nil {
			return "", cmdErr
		}
		_, _ = h.Write(out)
	}

	untrackedOut, err := exec.CommandContext(ctx, "git", "-C", root, "ls-files", "--others", "--exclude-standard", "-z").Output()
	if err != nil {
		return "", err
	}
	paths := strings.Split(string(untrackedOut), "\x00")
	sort.Strings(paths)
	for _, rel := range paths {
		if rel == "" {
			continue
		}
		clean := filepath.Clean(rel)
		if strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || filepath.IsAbs(clean) {
			continue
		}
		_, _ = h.Write([]byte(clean))
		data, readErr := os.ReadFile(filepath.Join(root, clean))
		if readErr == nil {
			_, _ = h.Write(data)
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
