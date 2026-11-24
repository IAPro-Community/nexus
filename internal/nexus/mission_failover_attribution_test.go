package nexus

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/launcher"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	profilepkg "github.com/IAPro-Community/nexus/internal/profile"
	"github.com/IAPro-Community/nexus/internal/testutil"
)

// TestMissionFailoverCostObservationAttributionFollowsExecutingRuntime pins the
// F2.8 cost-attribution contract for runtime model reassignment (issue #228,
// item #20): after a mission runtime exhausts its quota and the work is
// reassigned to another profile, every cost observation recorded by
// executeAgentPrompt (mission_execution.go) carries the provider/profile/
// runtime of whoever actually executed that attempt.
//
// Real behaviors this test fixes in place:
//   - A quota failure surfaced by the provider process (after the launch
//     boundary) DOES record a cost observation, with Status "FAILED" — the
//     failed attempt is never relabeled as a completed one.
//   - The reassigned attempt records its own observation with its own runtime
//     generation; nothing from attempt A (agent, runtime, generation, account
//     scope) leaks into attempt B's evidence, and vice versa.
//   - Token/cost values stay NULL (CostKind UNKNOWN): no cost is invented.
//
// Note: this test drives executeAgentPrompt directly for the two attempts of
// one mission because the runner-level reassignment mechanism (quota failure ->
// StateRemediating -> re-entry into StateAllocating -> re-allocation) is
// covered by TestFaultInjectionQuotaFailoverReallocatesAndKeepsAttribution in
// internal/nexus/runner; the runner state machine itself never records cost
// observations.
func TestMissionFailoverCostObservationAttributionFollowsExecutingRuntime(t *testing.T) {
	root := t.TempDir()
	hostHome := filepath.Join(root, "host-home")
	binDir := filepath.Join(root, "bin")
	runtimeDir := filepath.Join(root, "runtime")
	for _, dir := range []string{hostHome, binDir, runtimeDir, filepath.Join(root, "config")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatalf("create isolated test directory: %v", err)
		}
	}

	t.Setenv("HOME", hostHome)
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("NEXUS_CONFIG_DIR", filepath.Join(root, "config"))
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// The quota-a profile fails its turn with the production 429 surface; the
	// quota-b profile completes its turn. Both are the same fixture CLI; the
	// provider home distinguishes the accounts.
	fixturePath := testutil.WriteSpecBinary(t, binDir, "codex")
	t.Setenv("FIXTURE_CLI_SPEC", `{"version":"codex fixture 1.0","responses":[{"when_env_contains":[["CODEX_HOME","quota-a"]],"stdout":["Error: 429 Too Many Requests. Rate limit reached. Usage limit reached for profile quota-a."],"exit_code":1,"sleep_ms":1000},{"when_env_contains":[["HOME","quota-a"]],"stdout":["Error: 429 Too Many Requests. Rate limit reached. Usage limit reached for profile quota-a."],"exit_code":1,"sleep_ms":1000}],"stdout":["FIXTURE_FAILOVER_TURN_COMPLETED"],"sleep_ms":1000}`)
	if resolved, err := exec.LookPath("codex"); err != nil || resolved != fixturePath {
		t.Fatalf("runtime would not resolve to the isolated fixture CLI: path=%q err=%v", resolved, err)
	}

	n := openTestNexus(t)
	n.maestroStatus = func() MaestroStatus { return MaestroStatus{Available: false, Mode: MaestroOff} }
	n.launcher = standaloneSkillFixtureLauncher{inner: launcher.NewLauncher()}
	st, err := n.OpenProject()
	if err != nil {
		t.Fatalf("open isolated Nexus store: %v", err)
	}
	workspace := filepath.Join(root, "workspace")
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		t.Fatalf("create fixture workspace: %v", err)
	}
	project, err := st.CreateProject(store.Project{Name: "Mission failover attribution", CanonicalPath: workspace})
	if err != nil {
		t.Fatalf("create fixture project: %v", err)
	}

	createProfileAgent := func(name, profile string) store.Agent {
		t.Helper()
		agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: name, Role: "implementer"})
		if err != nil {
			t.Fatalf("create fixture Agent %s: %v", name, err)
		}
		cfg := AgentConfig{Provider: "codex", Profile: profile, Workspace: workspace}
		revision, err := st.AddRevision(agent.ID, cfg.ConfigJSON())
		if err != nil {
			t.Fatalf("persist fixture Agent config for %s: %v", name, err)
		}
		agent.CurrentRevisionID = revision.ID
		if err := st.UpdateAgent(agent); err != nil {
			t.Fatalf("select fixture Agent revision for %s: %v", name, err)
		}
		return agent
	}
	agentA := createProfileAgent("Quota A", "quota-a")
	agentB := createProfileAgent("Quota B", "quota-b")

	missionRunID := "run_failover_attribution"
	policy := agentPromptPolicy{
		Contract:     runner.DefaultAutonomyContract(),
		MissionRunID: missionRunID,
		PackageID:    "pkg-1",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Attempt 1: quota-a executes and its turn ends in the injected quota
	// exhaustion. The failure must stay a typed failure and keep its own
	// runtime evidence.
	executionA, errA := n.executeAgentPrompt(ctx, agentA.ID, workspace, "attempt one under quota-a", policy)
	if errA == nil {
		t.Fatal("quota-exhausted attempt must not be reported as success")
	}
	if executionA == nil || executionA.RuntimeID == "" {
		t.Fatalf("failed attempt must keep its runtime evidence: %+v (err=%v)", executionA, errA)
	}
	if executionA.Failure.Kind != model.FailureRateLimit {
		t.Fatalf("injected 429 surface must classify as a typed rate-limit/quota failure, got %s", executionA.Failure.Kind)
	}

	// Attempt 2: the reassigned profile quota-b executes the same package and
	// completes. Its evidence must be its own.
	executionB, errB := n.executeAgentPrompt(ctx, agentB.ID, workspace, "attempt two under quota-b", policy)
	if errB != nil {
		t.Fatalf("reassigned attempt must complete: %v", errB)
	}
	if executionB.RuntimeID == "" || executionB.RuntimeID == executionA.RuntimeID {
		t.Fatalf("reassigned attempt must have its own runtime identity: A=%q B=%q", executionA.RuntimeID, executionB.RuntimeID)
	}
	if !strings.Contains(executionB.Output, "FIXTURE_FAILOVER_TURN_COMPLETED") {
		t.Fatalf("reassigned attempt output missing completion evidence: %q", executionB.Output)
	}

	generationA, err := st.CurrentGeneration(agentA.ID)
	if err != nil {
		t.Fatalf("load runtime generation for quota-a attempt: %v", err)
	}
	generationB, err := st.CurrentGeneration(agentB.ID)
	if err != nil {
		t.Fatalf("load runtime generation for quota-b attempt: %v", err)
	}
	if generationA.RuntimeID != executionA.RuntimeID || generationB.RuntimeID != executionB.RuntimeID {
		t.Fatalf("runtime generations must stay bound to their own attempts: A=%s/%s B=%s/%s",
			generationA.RuntimeID, executionA.RuntimeID, generationB.RuntimeID, executionB.RuntimeID)
	}
	if generationA.ID == generationB.ID {
		t.Fatalf("failover attempts must not share a runtime generation: %s", generationA.ID)
	}

	observations, err := st.ListCostObservations(store.CostObservationFilter{MissionRunID: missionRunID})
	if err != nil {
		t.Fatalf("list cost observations for the mission: %v", err)
	}
	if len(observations) != 2 {
		t.Fatalf("each attempt must record exactly one cost observation, got %d: %+v", len(observations), observations)
	}

	byProfile := map[string]store.CostObservation{}
	for _, obs := range observations {
		if obs.Source != "runtime_observation" {
			t.Fatalf("unexpected observation source %q: %+v", obs.Source, obs)
		}
		byProfile[obs.Profile] = obs
	}
	obsA, okA := byProfile["quota-a"]
	obsB, okB := byProfile["quota-b"]
	if !okA || !okB {
		t.Fatalf("observations must be attributed per attempt profile: %+v", observations)
	}

	// The quota-failed attempt stays FAILED in the ledger; the reassigned one
	// is the only COMPLETED evidence.
	if obsA.Status != "FAILED" {
		t.Fatalf("quota-exhausted attempt must remain FAILED in the ledger, got %q", obsA.Status)
	}
	if obsB.Status != "COMPLETED" {
		t.Fatalf("reassigned attempt must be COMPLETED, got %q", obsB.Status)
	}

	// Each observation carries the identity of who executed that attempt.
	if obsA.AgentID != agentA.ID || obsB.AgentID != agentB.ID {
		t.Fatalf("cost observations must follow the executing agent: A=%s/%s B=%s/%s", obsA.AgentID, agentA.ID, obsB.AgentID, agentB.ID)
	}
	if obsA.RuntimeID != executionA.RuntimeID || obsB.RuntimeID != executionB.RuntimeID {
		t.Fatalf("cost observations must follow the executing runtime: A=%s/%s B=%s/%s", obsA.RuntimeID, executionA.RuntimeID, obsB.RuntimeID, executionB.RuntimeID)
	}
	if obsA.RuntimeGenerationID != generationA.ID || obsB.RuntimeGenerationID != generationB.ID {
		t.Fatalf("cost observations must follow their own runtime generation: A=%s/%s B=%s/%s",
			obsA.RuntimeGenerationID, generationA.ID, obsB.RuntimeGenerationID, generationB.ID)
	}
	if obsA.Provider != "codex" || obsB.Provider != "codex" {
		t.Fatalf("provider attribution lost: A=%q B=%q", obsA.Provider, obsB.Provider)
	}
	if obsA.MissionRunID != missionRunID || obsB.MissionRunID != missionRunID || obsA.PackageID != "pkg-1" || obsB.PackageID != "pkg-1" {
		t.Fatalf("both attempts must stay attributed to the same mission package: %+v %+v", obsA, obsB)
	}

	// Account scope follows each profile's own account; nothing crosses over.
	for _, attempt := range []struct {
		profile string
		obs     store.CostObservation
	}{{"quota-a", obsA}, {"quota-b", obsB}} {
		info := profilepkg.GetAccountInfo("codex", attempt.profile)
		if info.AccountScope.Verifiable() {
			if attempt.obs.AccountScope != info.AccountScope.Key() {
				t.Fatalf("account scope mismatch for %s: obs=%q expected=%q", attempt.profile, attempt.obs.AccountScope, info.AccountScope.Key())
			}
		} else if attempt.obs.AccountScope != "" {
			t.Fatalf("unverifiable account must not claim a scope for %s: %q", attempt.profile, attempt.obs.AccountScope)
		}
	}
	if obsA.AccountScope != obsB.AccountScope {
		t.Fatalf("attempts must not borrow each other's account scope: A=%q B=%q", obsA.AccountScope, obsB.AccountScope)
	}

	// Cost Intelligence is observational: no tokens or cost may be invented
	// for either attempt.
	for _, obs := range []store.CostObservation{obsA, obsB} {
		if obs.CostMicroUSD != nil || obs.InputTokens != nil || obs.OutputTokens != nil {
			t.Fatalf("cost observation must not invent token/cost evidence: %+v", obs)
		}
		if obs.CostKind != store.CostKindUnknown {
			t.Fatalf("unmeasured observation must stay CostKind UNKNOWN, got %q", obs.CostKind)
		}
	}
}
