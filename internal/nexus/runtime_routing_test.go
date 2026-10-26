package nexus

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func testAccounts() []ProviderAccount {
	return []ProviderAccount{
		{Provider: "agy", Profile: "frontend", Authenticated: true, Available: true, QuotaRemaining: 0.8},
		{Provider: "opencode", Profile: "qa", Authenticated: true, Available: true, QuotaRemaining: 0.7},
	}
}

func TestRuntimeRoutingDecisionPersistsExplainableSelectedFields(t *testing.T) {
	policy := RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPrefer, Value: "agy"}}
	accounts := testAccounts()
	accounts[0].Scope = model.AccountScope{ProviderID: "agy", AccountID: "acct-frontend", IdentityVersion: "2"}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "coding"}, policy, accounts, testModels())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(decision)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["selected_provider"] != "agy" || payload["selected_profile"] != "frontend" || payload["selected_model"] != "cheap" {
		t.Fatalf("decision omitted selected resource projection: %s", raw)
	}
	if decision.SelectedAccountScope.AccountID != "acct-frontend" || decision.SelectedAccountScope.IdentityVersion != "2" {
		t.Fatalf("decision omitted selected account scope: %+v", decision.SelectedAccountScope)
	}
	if _, ok := payload["affinity_policy"]; !ok {
		t.Fatalf("decision omitted affinity policy: %s", raw)
	}
}

func TestMissionRoutingProjectionPersistsActualSelectionAndAffinity(t *testing.T) {
	decision := buildMissionRoutingDecision(
		&runner.PackageRun{PackageID: "pkg-1"},
		store.Agent{ID: "agent-1"},
		TaskRequirements{TaskKind: "coding"},
		ProviderAccount{Provider: "opencode", Profile: "qa", Health: "healthy", Authenticated: true, Available: true},
		AgentConfig{Model: "mimo"},
		"agy",
		"frontend",
		AffinityPrefer,
		true,
		"preferred resource unavailable; scheduler selected a healthy alternative",
	)
	if decision.AffinityPolicy.Provider.Value != "agy" || decision.AffinityPolicy.Profile.Value != "frontend" {
		t.Fatalf("affinity policy was not persisted: %+v", decision.AffinityPolicy)
	}
	if decision.SelectedProvider != "opencode" || decision.SelectedProfile != "qa" || decision.SelectedModel != "mimo" {
		t.Fatalf("actual selection was not projected: %+v", decision)
	}
	if !decision.Fallback || decision.Reason == "" {
		t.Fatalf("fallback explanation missing: %+v", decision)
	}
}

func testModels() []ModelCandidate {
	return []ModelCandidate{
		{Provider: "agy", Profile: "frontend", Model: "cheap", CostRank: 1, ReasoningRank: 1, Healthy: true, Authenticated: true, QuotaAvailable: true},
		{Provider: "agy", Profile: "frontend", Model: "strong", CostRank: 3, ReasoningRank: 3, Healthy: true, Authenticated: true, QuotaAvailable: true},
		{Provider: "opencode", Profile: "qa", Model: "mimo", CostRank: 1, ReasoningRank: 2, Healthy: true, Authenticated: true, QuotaAvailable: true},
	}
}

func TestRuntimeRoutingPreferFallsBackWithoutMutatingDesiredPolicy(t *testing.T) {
	policy := RuntimeAffinityPolicy{
		Provider: AffinityPreference{Mode: AffinityPrefer, Value: "missing"},
		Model:    AffinityPreference{Mode: AffinityPrefer, Value: "strong"},
	}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "coding"}, policy, testAccounts(), testModels())
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Fallback || decision.Actual.Provider != "agy" || decision.Actual.Model != "strong" {
		t.Fatalf("unexpected fallback decision: %+v", decision)
	}
	if decision.Desired.Provider.Value != "missing" || decision.Desired.Model.Value != "strong" {
		t.Fatalf("desired affinity was mutated: %+v", decision.Desired)
	}
}

func TestRuntimeRoutingPinFailsClosed(t *testing.T) {
	policy := RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPin, Value: "missing"}}
	_, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "coding"}, policy, testAccounts(), testModels())
	if !errors.Is(err, ErrBlockedResource) {
		t.Fatalf("expected blocked resource, got %v", err)
	}
}

func TestRuntimeRoutingUsesCheapCapableModelForSimpleTask(t *testing.T) {
	policy := RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPrefer, Value: "agy"}}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "coding"}, policy, testAccounts(), testModels())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Actual.Model != "cheap" {
		t.Fatalf("expected cheap capable model, got %+v", decision.Actual)
	}
}

func TestRuntimeRoutingRaisesReasoningFloorForArchitecture(t *testing.T) {
	policy := RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPrefer, Value: "agy"}}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "architecture"}, policy, testAccounts(), testModels())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Actual.Model != "strong" {
		t.Fatalf("expected stronger model for architecture, got %+v", decision.Actual)
	}
}

func TestRuntimeRoutingPreferFallsBackWhenPreferredModelCannotSatisfyTask(t *testing.T) {
	policy := RuntimeAffinityPolicy{
		Provider: AffinityPreference{Mode: AffinityPrefer, Value: "agy"},
		Model:    AffinityPreference{Mode: AffinityPrefer, Value: "cheap"},
	}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "architecture"}, policy, testAccounts(), testModels())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Actual.Model != "strong" || !decision.Fallback {
		t.Fatalf("preferred incapable model must fall back to capable model: %+v", decision)
	}
	if decision.Desired.Model.Value != "cheap" {
		t.Fatalf("desired model preference was mutated: %+v", decision.Desired.Model)
	}
}

func TestMissionModelFallbackPersistsUnavailableModelProvenance(t *testing.T) {
	modelPolicy := RuntimeAffinityPolicy{Model: AffinityPreference{Mode: AffinityPrefer, Value: "cheap"}}
	requirements := TaskRequirements{TaskKind: "architecture", RuntimeAffinity: &modelPolicy}
	account := ProviderAccount{Provider: "agy", Profile: "frontend", Health: "healthy", Authenticated: true, Available: true}
	models := []ModelCandidate{
		{Provider: "agy", Profile: "frontend", Model: "cheap", CostRank: 1, ReasoningRank: 1, Healthy: true, Authenticated: true, QuotaAvailable: true},
		{Provider: "agy", Profile: "frontend", Model: "strong", CostRank: 3, ReasoningRank: 3, Healthy: true, Authenticated: true, QuotaAvailable: true},
	}

	selected, fallback, reason, err := ResolveTaskModel(requirements, modelPolicy.Model, account, models)
	if err != nil {
		t.Fatal(err)
	}
	if !fallback || selected.Model != "strong" || reason == "" {
		t.Fatalf("expected documented capable-model fallback, selected=%+v fallback=%t reason=%q", selected, fallback, reason)
	}

	routing := buildMissionRoutingDecision(
		&runner.PackageRun{PackageID: "architecture", Attempt: 1}, store.Agent{ID: "agent-1"},
		requirements, account, AgentConfig{Provider: account.Provider, Profile: account.Profile, Model: selected.Model},
		"", "", AffinityAuto, fallback, reason,
	)
	if !routing.Fallback || routing.RequestedModel != "cheap" || routing.SelectedModel != "strong" {
		t.Fatalf("model fallback provenance missing: %+v", routing)
	}
	if routing.SwitchReason != SwitchReasonModelIncapable || routing.Reason == "" {
		t.Fatalf("model fallback reason missing: %+v", routing)
	}
}

func TestRuntimeRoutingUsesDeterministicTieBreak(t *testing.T) {
	accounts := []ProviderAccount{
		{Provider: "opencode", Profile: "qa", Authenticated: true, Available: true, QuotaRemaining: 0.5},
		{Provider: "agy", Profile: "frontend", Authenticated: true, Available: true, QuotaRemaining: 0.5},
	}
	models := []ModelCandidate{
		{Provider: "agy", Profile: "frontend", Model: "zeta", CostRank: 1, ReasoningRank: 1, Healthy: true, Authenticated: true, QuotaAvailable: true},
		{Provider: "agy", Profile: "frontend", Model: "alpha", CostRank: 1, ReasoningRank: 1, Healthy: true, Authenticated: true, QuotaAvailable: true},
	}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "coding"}, RuntimeAffinityPolicy{}, accounts, models)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Actual.Provider != "agy" || decision.Actual.Model != "alpha" {
		t.Fatalf("tie-break must be deterministic by provider/profile then model: %+v", decision.Actual)
	}
}

func TestResolveTaskModelEscalatesAfterVerificationFailure(t *testing.T) {
	account := ProviderAccount{Provider: "agy", Profile: "frontend", Health: "healthy", Authenticated: true, Available: true}
	models := []ModelCandidate{
		{Provider: "agy", Profile: "frontend", Model: "cheap", CostRank: 1, ReasoningRank: 1, Healthy: true, Authenticated: true, QuotaAvailable: true},
		{Provider: "agy", Profile: "frontend", Model: "medium", CostRank: 2, ReasoningRank: 2, Healthy: true, Authenticated: true, QuotaAvailable: true},
		{Provider: "agy", Profile: "frontend", Model: "strong", CostRank: 3, ReasoningRank: 3, Healthy: true, Authenticated: true, QuotaAvailable: true},
	}
	selected, fallback, reason, err := ResolveTaskModel(TaskRequirements{TaskKind: "coding", EscalationLevel: 1}, AffinityPreference{Mode: AffinityAuto}, account, models)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Model != "medium" || fallback || reason == "" {
		t.Fatalf("expected deterministic medium escalation, got model=%+v fallback=%t reason=%q", selected, fallback, reason)
	}
}

func TestResolveTaskModelPinUnavailableBlocks(t *testing.T) {
	account := ProviderAccount{Provider: "agy", Profile: "frontend", Health: "healthy", Authenticated: true, Available: true}
	_, _, _, err := ResolveTaskModel(TaskRequirements{TaskKind: "coding"}, AffinityPreference{Mode: AffinityPin, Value: "missing"}, account, []ModelCandidate{{Provider: "agy", Profile: "frontend", Model: "cheap"}})
	if !errors.Is(err, ErrBlockedResource) {
		t.Fatalf("expected PIN model to block, got %v", err)
	}
}

func TestResolveTaskModelPreferDoesNotDefeatTaskCapability(t *testing.T) {
	account := ProviderAccount{Provider: "agy", Profile: "frontend", Health: "healthy", Authenticated: true, Available: true}
	models := []ModelCandidate{
		{Provider: "agy", Profile: "frontend", Model: "cheap", CostRank: 1, ReasoningRank: 1, Healthy: true, Authenticated: true, QuotaAvailable: true},
		{Provider: "agy", Profile: "frontend", Model: "strong", CostRank: 3, ReasoningRank: 3, Healthy: true, Authenticated: true, QuotaAvailable: true},
	}
	selected, fallback, _, err := ResolveTaskModel(TaskRequirements{TaskKind: "architecture"}, AffinityPreference{Mode: AffinityPrefer, Value: "cheap"}, account, models)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Model != "strong" || !fallback {
		t.Fatalf("preferred incapable model must yield capable fallback: model=%+v fallback=%t", selected, fallback)
	}
}

func TestMissionRoutingProjectionPersistsTaskModelAffinity(t *testing.T) {
	modelPolicy := RuntimeAffinityPolicy{Model: AffinityPreference{Mode: AffinityPrefer, Value: "strong"}}
	decision := buildMissionRoutingDecision(
		&runner.PackageRun{PackageID: "pkg-model"},
		store.Agent{ID: "agent-model"},
		TaskRequirements{TaskKind: "coding", RuntimeAffinity: &modelPolicy},
		ProviderAccount{Provider: "agy", Profile: "frontend", Health: "healthy", Authenticated: true, Available: true},
		AgentConfig{Model: "cheap"},
		"", "", AffinityAuto, false, "selected by scheduler",
	)
	if decision.AffinityPolicy.Model.Mode != AffinityPrefer || decision.AffinityPolicy.Model.Value != "strong" {
		t.Fatalf("task model affinity was not persisted: %+v", decision.AffinityPolicy)
	}
}

func TestMissionRoutingProjectionPersistsAgentSelectionEvidence(t *testing.T) {
	decision := buildMissionRoutingDecision(
		&runner.PackageRun{PackageID: "pkg-agent"},
		store.Agent{ID: "agent-frontend"},
		TaskRequirements{TaskKind: "coding"},
		ProviderAccount{Provider: "agy", Profile: "frontend", Health: "healthy", Authenticated: true, Available: true},
		AgentConfig{Model: "cheap"},
		"", "", AffinityAuto, false, "selected by scheduler",
		agentSelectionEvidence{Score: 0.94, Confidence: "HIGH", Reason: "frontend specialization matched"},
	)
	if decision.AgentID != "agent-frontend" || decision.AgentScore != 0.94 || decision.AgentConfidence != "HIGH" || decision.AgentReason == "" {
		t.Fatalf("agent matching evidence was not persisted: %+v", decision)
	}
}

func TestMissionRoutingProjectionPersistsSkillAndMaestroGuidanceReferences(t *testing.T) {
	pkg := &runner.PackageRun{
		PackageID: "pkg-guidance",
		SkillIDs:  []string{"testing", "testing"},
	}
	req := missionTaskRequirements(&runner.PackageRun{
		PackageID:        pkg.PackageID,
		SkillIDs:         pkg.SkillIDs,
		TaskRequirements: `{"task_kind":"testing","guidance":{"enabled":true,"source":"maestro","reference":"advice:run-7","instructions":["run the independent gate"]}}`,
	})
	decision := buildMissionRoutingDecision(
		pkg,
		store.Agent{ID: "agent-qa"},
		req,
		ProviderAccount{Provider: "opencode", Profile: "qa", Health: "healthy", Authenticated: true, Available: true},
		AgentConfig{Model: "mimo"},
		"", "", AffinityAuto, false, "selected by scheduler",
	)
	if len(decision.SkillRefs) != 1 || decision.SkillRefs[0] != "testing" {
		t.Fatalf("expected deduplicated canonical skill refs, got %v", decision.SkillRefs)
	}
	if decision.MaestroGuidanceRef != "advice:run-7" {
		t.Fatalf("expected persisted Maestro guidance reference, got %q", decision.MaestroGuidanceRef)
	}
}

func TestMissionTaskRequirementsPreservesModelPolicyAndEscalation(t *testing.T) {
	pkg := &runner.PackageRun{TaskRequirements: `{"task_kind":"coding","runtime_affinity":{"model":{"mode":"PIN","value":"strong"}},"model_candidates":[{"model":"strong","cost_rank":3,"reasoning_rank":3}],"escalation_level":2}`}
	req := missionTaskRequirements(pkg)
	if req.RuntimeAffinity == nil || req.RuntimeAffinity.Model.Mode != AffinityPin || req.RuntimeAffinity.Model.Value != "strong" || len(req.ModelCandidates) != 1 || req.ModelCandidates[0].Model != "strong" || req.EscalationLevel != 2 {
		t.Fatalf("task model policy was not preserved: %+v", req)
	}
}

func TestMissionAllocatePinFailsClosedAfterRateLimit(t *testing.T) {
	accounts := []ProviderAccount{
		{Provider: "codex", Profile: "main", Authenticated: true, Available: false, RateLimited: true},
		{Provider: "agy", Profile: "backup", Authenticated: true, Available: true, QuotaRemaining: 0.9},
	}
	pkg := &runner.PackageRun{
		PackageID:       "pkg-pin",
		Provider:        "codex",
		Profile:         "main",
		DesiredProvider: "codex",
		DesiredProfile:  "main",
		ResourcePolicy:  string(PolicyManual),
		Attempt:         2,
		RetryFrom:       runner.StateAllocating,
		ErrorMessage:    "provider returned 429 rate limit",
	}
	_, err := selectMissionCandidateAccounts(accounts, pkg, AgentConfig{Provider: "codex", Profile: "main"}, AffinityPin, true)
	if !errors.Is(err, ErrBlockedResource) {
		t.Fatalf("PIN failover must fail closed, got %v", err)
	}
}

func TestMissionAllocatePreferMayOpenPoolAfterRateLimit(t *testing.T) {
	accounts := []ProviderAccount{
		{Provider: "codex", Profile: "main", Authenticated: true, Available: false, RateLimited: true},
		{Provider: "agy", Profile: "backup", Authenticated: true, Available: true, QuotaRemaining: 0.9},
	}
	pkg := &runner.PackageRun{
		PackageID:       "pkg-prefer",
		Provider:        "codex",
		Profile:         "main",
		DesiredProvider: "codex",
		DesiredProfile:  "main",
		Attempt:         2,
		RetryFrom:       runner.StateAllocating,
		ErrorMessage:    "quota exhausted",
	}
	candidates, err := selectMissionCandidateAccounts(accounts, pkg, AgentConfig{Provider: "codex", Profile: "main"}, AffinityPrefer, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Provider != "agy" {
		t.Fatalf("PREFER failover should open alternatives, got %+v", candidates)
	}
}

func TestMissionFallbackPreservesResourceAndFailureProvenance(t *testing.T) {
	cases := []struct {
		name         string
		failure      model.FailureKind
		accounts     []ProviderAccount
		wantProvider string
		wantProfile  string
		wantReason   RoutingSwitchReason
	}{
		{
			name:    "same provider profile after quota exhaustion",
			failure: model.FailureQuota,
			accounts: []ProviderAccount{
				{ID: "codex-main", Provider: "codex", Profile: "main", Authenticated: true, Available: false, RateLimited: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}},
				{ID: "codex-backup", Provider: "codex", Profile: "backup", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.45},
				{ID: "agy-backup", Provider: "agy", Profile: "backup", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.95},
			},
			wantProvider: "codex",
			wantProfile:  "backup",
			wantReason:   SwitchReasonQuotaExhausted,
		},
		{
			name:    "cross provider after rate limit with unauthenticated preferred-provider backup",
			failure: model.FailureRateLimit,
			accounts: []ProviderAccount{
				{ID: "codex-main", Provider: "codex", Profile: "main", Authenticated: true, Available: false, RateLimited: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}},
				{ID: "codex-backup", Provider: "codex", Profile: "backup", Authenticated: false, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 1},
				{ID: "agy-backup", Provider: "agy", Profile: "backup", Authenticated: true, Available: true, Health: "healthy", QuotaView: &quota.QuotaView{Status: "LIVE"}, QuotaRemaining: 0.60},
			},
			wantProvider: "agy",
			wantProfile:  "backup",
			wantReason:   SwitchReasonRateLimited,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkg := &runner.PackageRun{
				PackageID: "step-1", Attempt: 2, RetryFrom: runner.StateAllocating,
				DesiredProvider: "codex", DesiredProfile: "main", FailureKind: tc.failure,
			}
			current := AgentConfig{Provider: "codex", Profile: "main"}
			candidates, err := selectMissionCandidateAccounts(tc.accounts, pkg, current, AffinityPrefer, true)
			if err != nil {
				t.Fatal(err)
			}
			recommendation := RecommendResources(candidates, TaskRequirements{
				TaskKind: "coding", ProviderPriorities: map[string]int{"codex": 1, "agy": 2},
			}, PolicyBalanced)
			if recommendation.Recommended == nil {
				t.Fatalf("no eligible fallback: %s", recommendation.Explanation)
			}
			selected := recommendation.Recommended.Account
			if selected.Provider != tc.wantProvider || selected.Profile != tc.wantProfile {
				t.Fatalf("selected %s:%s, want %s:%s", selected.Provider, selected.Profile, tc.wantProvider, tc.wantProfile)
			}

			routing := buildMissionRoutingDecision(
				pkg, store.Agent{ID: "agent-1"}, TaskRequirements{TaskKind: "coding"},
				selected, AgentConfig{Provider: selected.Provider, Profile: selected.Profile, Model: "configured-model"},
				"codex", "main", AffinityPrefer, true, recommendation.Explanation,
			)
			if !routing.Fallback || routing.RequestedProvider != "codex" || routing.RequestedProfile != "main" {
				t.Fatalf("fallback lost requested resource provenance: %+v", routing)
			}
			if routing.SelectedProvider != tc.wantProvider || routing.SelectedProfile != tc.wantProfile {
				t.Fatalf("fallback lost selected resource provenance: %+v", routing)
			}
			if routing.SwitchReason != tc.wantReason || routing.Reason == "" || len(routing.RejectedCandidates) == 0 {
				t.Fatalf("fallback lost failure explanation: %+v", routing)
			}
		})
	}
}

func TestConfiguredModelCandidatesDoesNotInheritAgentModel(t *testing.T) {
	account := ProviderAccount{Provider: "agy", Profile: "frontend", Health: "healthy", Authenticated: true, Available: true}
	got := configuredModelCandidates(AgentConfig{Model: "opus-expensive"}, account)
	if len(got) != 0 {
		t.Fatalf("AUTO must not mint a leftover AgentConfig.Model pool: %+v", got)
	}
	got = configuredModelCandidates(AgentConfig{
		Model: "opus-expensive",
		ModelCandidates: []ModelCandidate{
			{Model: "cheap", CostRank: 1, ReasoningRank: 1},
			{Model: "opus-expensive", CostRank: 5, ReasoningRank: 5},
		},
	}, account)
	selected, _, _, err := ResolveTaskModel(TaskRequirements{TaskKind: "testing"}, AffinityPreference{Mode: AffinityAuto}, account, got)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Model != "cheap" {
		t.Fatalf("testing WorkUnit must not inherit expensive leftover model, got %s", selected.Model)
	}
}

func TestMissionAffinityModeManualIsPin(t *testing.T) {
	if got := missionAffinityMode(PolicyManual, "codex", "main"); got != AffinityPin {
		t.Fatalf("expected PIN, got %s", got)
	}
	if got := missionAffinityMode(PolicyBalanced, "codex", "main"); got != AffinityPrefer {
		t.Fatalf("expected PREFER, got %s", got)
	}
	if got := missionAffinityMode(PolicyBalanced, "", ""); got != AffinityAuto {
		t.Fatalf("expected AUTO, got %s", got)
	}
}

func TestExplainNoEligibleMissionAccountsTyped(t *testing.T) {
	err := explainNoEligibleMissionAccounts(nil, "step-1", false)
	if !errors.Is(err, ErrAllocationUnregistered) {
		t.Fatalf("empty pool error=%v want UNREGISTERED", err)
	}
	err = explainNoEligibleMissionAccounts([]ProviderAccount{{Provider: "codex", Profile: "a", Authenticated: false}}, "step-1", false)
	if !errors.Is(err, ErrAllocationUnauthenticated) {
		t.Fatalf("unauth pool error=%v want UNAUTHENTICATED", err)
	}
	err = explainNoEligibleMissionAccounts([]ProviderAccount{{
		Provider: "codex", Profile: "a", Authenticated: true, Available: false,
		QuotaView:    &quota.QuotaView{Status: "UNKNOWN"},
		AvailReasons: &quota.AvailReasons{UnknownQuota: true},
	}}, "step-1", false)
	if !errors.Is(err, ErrAllocationUnknownQuota) {
		t.Fatalf("unknown pool error=%v want UNKNOWN", err)
	}
	err = explainNoEligibleMissionAccounts([]ProviderAccount{{
		Provider: "codex", Profile: "a", Authenticated: true, RateLimited: true,
	}}, "step-1", false)
	if !errors.Is(err, ErrAllocationRateLimited) {
		t.Fatalf("rate-limit pool error=%v want RATE_LIMITED", err)
	}
	err = explainNoEligibleMissionAccounts([]ProviderAccount{{
		Provider: "codex", Profile: "a", Authenticated: true, Available: false,
	}}, "step-1", true)
	if err == nil || !strings.Contains(err.Error(), "after quota/rate limit failure") {
		t.Fatalf("failover generic error=%v", err)
	}
	err = explainNoEligibleMissionAccounts([]ProviderAccount{{
		Provider: "codex", Profile: "a", Authenticated: true, Available: false,
		QuotaTotal: 1, QuotaRemaining: 0,
		AvailReasons: &quota.AvailReasons{ExhaustedWindows: []string{"5h"}},
	}}, "step-1", false)
	if !errors.Is(err, ErrAllocationQuotaExhausted) {
		t.Fatalf("exhausted quota error=%v want QUOTA_EXHAUSTED", err)
	}
	err = explainNoEligibleMissionAccounts([]ProviderAccount{{
		Provider: "codex", Profile: "a", Authenticated: true, NetworkError: true,
	}}, "step-1", false)
	if !errors.Is(err, ErrAllocationNetwork) {
		t.Fatalf("network error=%v want NETWORK", err)
	}
}
