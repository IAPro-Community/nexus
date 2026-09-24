package nexus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestAgentProjectionTargetProbeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_AGENT_PROJECTION_PROBE") != "1" {
		return
	}
	joined := strings.Join(os.Args, " ")
	if strings.Contains(joined, "agent list") || strings.Contains(joined, "debug agents") {
		fmt.Fprintln(os.Stdout, "nexus-review")
		os.Exit(0)
	}
	fmt.Fprintln(os.Stdout, "1.18.30")
	os.Exit(0)
}

func TestCustomVPNWorkspaceProjectionLifecycle(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}

	workspace := t.TempDir()
	sharedHome := t.TempDir()
	project, err := st.CreateProject(store.Project{Name: "VPN Workspace", CanonicalPath: workspace})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{
		ProjectID: project.ID,
		Name:      "VPN Reviewer",
		Role:      "reviewer",
	})
	if err != nil {
		t.Fatal(err)
	}

	probeBinary := filepath.ToSlash(os.Args[0])
	probeTemplate := fmt.Sprintf(
		`"%s" -test.run=TestAgentProjectionTargetProbeHelper -- {args}`,
		probeBinary,
	)
	cfg := AgentConfig{
		Provider:  "opencode",
		Profile:   "default",
		Workspace: workspace,
		Environment: map[string]string{
			"GO_WANT_AGENT_PROJECTION_PROBE": "1",
		},
		AgentSpec: mustAgentMode(t, "reviewer").AgentSpec,
		Options: map[string]any{
			"mode":                  "Safe",
			"origin":                "custom",
			"execution_adapter":     "command_template",
			"command_template":      `docker exec -it -w "{cwd}" vpn-dev-workspace-terminal-1 {provider} {args}`,
			"target_kind":           "docker",
			"target_profile_home":   sharedHome,
			"target_probe_template": probeTemplate,
		},
	}
	revision, err := st.AddRevision(agent.ID, cfg.ConfigJSON())
	if err != nil {
		t.Fatal(err)
	}
	agent.CurrentRevisionID = revision.ID
	if err := st.UpdateAgent(agent); err != nil {
		t.Fatal(err)
	}
	generation, err := st.AddGeneration(store.RuntimeGeneration{
		AgentID:    agent.ID,
		RevisionID: revision.ID,
		RuntimeID:  "rt_vpn_projection",
		Provider:   "opencode",
		Profile:    "default",
		StartedAt:  time.Now().UTC(),
		State:      "RUNNING",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := n.SetRuntimeExecutionRole(context.Background(), generation.RuntimeID, "review", "test"); err != nil {
		t.Fatal(err)
	}

	plan, err := n.PlanRuntimeAgentProjection(context.Background(), generation.RuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.CustomCommand || plan.TargetKind != "docker" {
		t.Fatalf("custom target identity lost: %#v", plan)
	}
	if plan.ProviderVersion != "1.18.30" || plan.VersionSource != "TARGET_PROBE" {
		t.Fatalf("target version was not probed from custom execution environment: %#v", plan)
	}
	if plan.Plan.ManagedRoot != sharedHome {
		t.Fatalf("projection must target shared VPN profile home: got %q want %q", plan.Plan.ManagedRoot, sharedHome)
	}
	if plan.Plan.RendererVersion != "opencode-v1-agent-md/1" {
		t.Fatalf("unexpected renderer: %s", plan.Plan.RendererVersion)
	}
	if len(plan.Plan.Conflicts) != 0 {
		t.Fatalf("new shared target projection should not conflict: %#v", plan.Plan.Conflicts)
	}

	applied, err := n.ApplyRuntimeAgentProjection(context.Background(), generation.RuntimeID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if applied.Receipt == nil || applied.Receipt.ID == "" {
		t.Fatalf("missing durable projection receipt: %#v", applied)
	}
	if applied.Discovery == nil || !applied.Discovery.Discovered || applied.Discovery.Status != "VERIFIED" {
		t.Fatalf("custom VPN target did not discover projected OpenCode agent: %#v", applied.Discovery)
	}
	if applied.Discovery.AgentName != "nexus-review" {
		t.Fatalf("unexpected discovered agent name: %#v", applied.Discovery)
	}
	target := filepath.Join(sharedHome, ".config", "opencode", "agents", "nexus-review.md")
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "permission:") ||
		!strings.Contains(string(raw), "edit: deny") ||
		!strings.Contains(string(raw), "bash: deny") {
		t.Fatalf("VPN workspace reviewer did not receive V1 read-only agent permissions:\n%s", raw)
	}

	binding, err := st.RuntimeExecutionRole(generation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if binding.ProjectionReceiptID != applied.Receipt.ID {
		t.Fatalf("runtime role did not link projection receipt: %#v", binding)
	}
	if binding.Enforcement == EnforcementEnforced {
		t.Fatalf("filesystem projection must not falsely promote runtime enforcement: %#v", binding)
	}

	verification, err := n.VerifyAgentProjection(context.Background(), applied.Receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !verification.Verified || len(verification.Warnings) == 0 {
		t.Fatalf("expected filesystem verification with live-activation warning: %#v", verification)
	}

	discovery, err := n.ProbeRuntimeAgentProjectionDiscovery(context.Background(), generation.RuntimeID)
	if err != nil {
		t.Fatal(err)
	}
	if !discovery.Discovered || discovery.Status != "VERIFIED" {
		t.Fatalf("explicit discovery probe did not verify native agent: %#v", discovery)
	}

	if err := n.RemoveAgentProjection(context.Background(), applied.Receipt.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("projected OpenCode agent was not removed, stat err=%v", err)
	}
	record, err := st.AgentProjectionReceipt(applied.Receipt.ID)
	if err != nil {
		t.Fatal(err)
	}
	if record.RemovedAt == nil {
		t.Fatalf("projection removal was not persisted: %#v", record)
	}
}

func TestCustomTargetProjectionRequiresSharedProfileHome(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Remote", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Remote Dev", Role: "developer"})
	if err != nil {
		t.Fatal(err)
	}
	cfg := AgentConfig{
		Provider:  "opencode",
		Profile:   "default",
		AgentSpec: mustAgentMode(t, "developer").AgentSpec,
		Options: map[string]any{
			"execution_adapter": "command_template",
			"command_template":  `ssh devbox {provider} {args}`,
			"target_kind":       "ssh",
		},
	}
	revision, err := st.AddRevision(agent.ID, cfg.ConfigJSON())
	if err != nil {
		t.Fatal(err)
	}
	generation, err := st.AddGeneration(store.RuntimeGeneration{
		AgentID: agent.ID, RevisionID: revision.ID, RuntimeID: "rt_remote_no_home",
		Provider: "opencode", Profile: "default", StartedAt: time.Now().UTC(), State: "RUNNING",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = n.PlanRuntimeAgentProjection(context.Background(), generation.RuntimeID)
	if err == nil || !strings.Contains(err.Error(), "target_profile_home") {
		t.Fatalf("expected truthful shared-home requirement, got %v", err)
	}
}
