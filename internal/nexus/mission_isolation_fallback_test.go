package nexus

import (
	"context"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func isolationTestAccount(id, provider, profile string) ProviderAccount {
	return ProviderAccount{
		ID: id, Provider: provider, Profile: profile, DisplayName: provider + " " + profile,
		Authenticated: true, Available: true, Health: "healthy",
		QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.8,
		Capabilities: map[string]string{"headless": "SUPPORTED", "submit_prompt": "SUPPORTED"},
	}
}

// Only codex carries a provider-native verified sandbox today; opencode and
// the other CLIs stay UNVERIFIED until their vendors ship enforcement.
func TestMissionIsolationCandidatesFallsBackToVerifiedPool(t *testing.T) {
	accounts := []ProviderAccount{
		isolationTestAccount("opencode-1", "opencode", "default"),
		isolationTestAccount("codex-1", "codex", "main"),
	}
	candidates := accounts[:1]
	contract := runner.DefaultAutonomyContract()

	eligible, rejected, fallback := missionIsolationCandidates(accounts, candidates, contract, AffinityPrefer)
	if !fallback {
		t.Fatal("non-pinned affinity must fall back to the verified pool when the desired provider is isolation-incompatible")
	}
	if len(eligible) != 1 || eligible[0].Provider != "codex" {
		t.Fatalf("expected the verified codex account, got %+v (rejected=%v)", eligible, rejected)
	}
	if len(rejected) == 0 || !strings.Contains(rejected[0], "opencode:default") {
		t.Fatalf("fallback must keep the rejection explanation for audit: %v", rejected)
	}

	// A pinned identity is the maestro's explicit choice and never drifts.
	eligiblePin, rejectedPin, fallbackPin := missionIsolationCandidates(accounts, candidates, contract, AffinityPin)
	if fallbackPin || len(eligiblePin) != 0 || len(rejectedPin) == 0 {
		t.Fatalf("pinned runtime must stay blocked on isolation rejection: eligible=%v fallback=%v rejected=%v", eligiblePin, fallbackPin, rejectedPin)
	}
}

func TestMissionIsolationCandidatesBlocksWhenWholePoolUnverified(t *testing.T) {
	accounts := []ProviderAccount{isolationTestAccount("opencode-1", "opencode", "default")}
	eligible, rejected, fallback := missionIsolationCandidates(accounts, accounts, runner.DefaultAutonomyContract(), AffinityAuto)
	if fallback || len(eligible) != 0 || len(rejected) != 1 {
		t.Fatalf("pool without any verified provider must stay blocked: eligible=%v fallback=%v rejected=%v", eligible, fallback, rejected)
	}
}

func TestExecutionPreviewFallsBackToVerifiedProvider(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Isolation fallback preview", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := n.CreateWorkPlan(context.Background(), project.ID, "Fallback plan", "", []store.PlanPhase{{
		ID: "phase", Title: "Build", Order: 1,
		Packages: []store.WorkPackage{{
			ID: "step", Title: "Build", Goal: "Build feature", Status: "READY",
			Role: "implementer", AssignmentStrategy: "AUTO", ResourcePolicy: "BALANCED",
			Provider: "opencode", Profile: "default",
		}},
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}

	accounts := []ProviderAccount{
		isolationTestAccount("opencode-1", "opencode", "default"),
		isolationTestAccount("codex-1", "codex", "main"),
	}
	preview, err := n.previewPlanExecutionWithContract(
		context.Background(), st, plan, accounts, nil, runner.DefaultAutonomyContract(), true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Ready {
		t.Fatalf("preview must fall back to the verified pool instead of blocking: %+v", preview.Blockers)
	}
	step := preview.Steps[0]
	if !strings.Contains(step.Resource.Reason, "rejected by the approved isolation contract") {
		t.Fatalf("fallback must document the drift in the resource reason: %q", step.Resource.Reason)
	}
	if !strings.Contains(step.Resource.Reason, "opencode:default") {
		t.Fatalf("fallback reason must name the rejected provider: %q", step.Resource.Reason)
	}

	// Without any verified provider in the pool the fail-closed gate holds.
	previewBlocked, err := n.previewPlanExecutionWithContract(
		context.Background(), st, plan, accounts[:1], nil, runner.DefaultAutonomyContract(), true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if previewBlocked.Ready {
		t.Fatal("preview without a verified provider must stay blocked")
	}
	found := false
	for _, blocker := range previewBlocked.Blockers {
		if strings.Contains(blocker, "no provider satisfies the approved Mission isolation contract") {
			found = true
		}
	}
	if !found {
		t.Fatalf("blocked preview must explain the isolation contract: %+v", previewBlocked.Blockers)
	}
}
