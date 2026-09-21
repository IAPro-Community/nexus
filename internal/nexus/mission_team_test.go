package nexus

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestMissionTeamSelectionKeepsAtomicTasksDirect(t *testing.T) {
	decision := DecideIntent("Troque o texto Login para Entrar.", nil)
	selection := SelectMissionTeam("Troque o texto Login para Entrar.", decision)
	if !selection.Direct || selection.Preset != "" || selection.CostClass != "DIRECT" {
		t.Fatalf("atomic task was over-processed: %+v", selection)
	}
}

func TestMissionTeamSelectionRespectsDelegationOff(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "MissionTeamOff", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	proposal, err := n.DecomposePromptIntoFlowProposal(context.Background(), FlowDecompositionRequest{ProjectID: project.ID, Goal: "Implemente uma feature completa com backend e frontend", DelegationMode: DelegationOff})
	if err != nil {
		t.Fatal(err)
	}
	selection, err := ReadMissionTeamSelectionFacts(proposal.Flow.StructuredFacts)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Direct || selection.Preset != "" || !strings.Contains(selection.Reason, "OFF") {
		t.Fatalf("delegation OFF must keep the Lead path: %+v", selection)
	}
}

func TestMissionTeamSelectionChoosesExplainablePresets(t *testing.T) {
	tests := []struct {
		goal   string
		preset MissionTeamPreset
	}{
		{goal: "Implemente autenticação com backend, frontend e testes e deixe pronta para produção", preset: PresetFeatureDelivery},
		{goal: "Corrija o bug quebrado no fluxo de login", preset: PresetBugFix},
		{goal: "Refatore o módulo de sessões", preset: PresetRefactor},
		{goal: "Faça uma mudança arquitetural no sistema de autenticação", preset: PresetArchitectureChange},
		{goal: "Prepare o release candidate para merge", preset: PresetReleaseCandidate},
	}
	for _, test := range tests {
		decision := DecideIntent(test.goal, nil)
		selection := SelectMissionTeam(test.goal, decision)
		if selection.Preset != test.preset || selection.Direct || selection.Reason == "" || len(selection.Signals) == 0 {
			t.Fatalf("goal %q selection=%+v want preset=%s", test.goal, selection, test.preset)
		}
	}
}

func TestMissionTeamPresetFactsRoundTripAndRejectInvalidPreset(t *testing.T) {
	selection := SelectMissionTeam("Implemente uma feature completa", DecideIntent("Implemente uma feature completa", nil))
	facts, err := PersistMissionTeamFacts(map[string]string{"existing": "keep"}, selection)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadMissionTeamSelectionFacts(facts)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Preset != selection.Preset || facts["existing"] != "keep" || facts[MissionTeamPresetFactKey] == "" {
		t.Fatalf("selection round trip lost durable data: decoded=%+v facts=%v", decoded, facts)
	}
	facts[MissionTeamSelectionFactKey] = `{"preset":"NOT_A_PRESET"}`
	if _, err := ReadMissionTeamSelectionFacts(facts); err == nil {
		t.Fatal("invalid mission team preset must fail closed")
	}
}

func TestBuildMissionTeamFlowCreatesOrderedRoleBasedPipeline(t *testing.T) {
	selection := MissionTeamSelection{Version: MissionTeamVersion, Strategy: IntentPlan, Preset: PresetFeatureDelivery, ReviewRequired: true}
	flow := &FlowDefinition{ID: "flow", ProjectID: "project", StructuredFacts: map[string]string{}}
	streams := []DelegatedWorkstream{
		{ID: "backend", Goal: "backend", PreferredRoles: []string{"backend-engineer"}, OwnedPaths: []string{"internal/**"}},
		{ID: "frontend", Goal: "frontend", PreferredRoles: []string{"frontend-engineer"}, OwnedPaths: []string{"web/src/**"}},
		{ID: "qa", Goal: "qa", PreferredRoles: []string{"qa-engineer"}},
	}
	if err := BuildMissionTeamFlow(flow, selection, "feature", []string{"true"}, nil, streams); err != nil {
		t.Fatal(err)
	}
	if len(flow.Phases) != 4 || len(flow.Steps) != 5 {
		t.Fatalf("unexpected pipeline shape phases=%d steps=%d", len(flow.Phases), len(flow.Steps))
	}
	if err := ValidateFlowDAG(*flow); err != nil {
		t.Fatal(err)
	}
	if flow.Steps[0].PhaseType != string(PhasePlanning) || flow.Steps[0].Role != "architect" {
		t.Fatalf("planner phase lost role/type: %+v", flow.Steps[0])
	}
	if flow.Steps[1].PhaseType != string(PhaseImplementation) || flow.Steps[2].PhaseType != string(PhaseImplementation) {
		t.Fatalf("implementation fan-out missing: %+v", flow.Steps)
	}
	if flow.Steps[3].PhaseType != string(PhaseReview) || !flow.Steps[3].ReviewRequired || len(flow.Steps[3].Dependencies) != 2 {
		t.Fatalf("review gate missing: %+v", flow.Steps[3])
	}
	if flow.Steps[4].PhaseType != string(PhaseVerification) || flow.Steps[4].ReviewRequired || len(flow.Steps[4].Dependencies) != 1 {
		t.Fatalf("verification gate missing: %+v", flow.Steps[4])
	}
	if strings.Contains(flow.Steps[1].HandoffJSON, "codex") || strings.Contains(flow.Steps[1].HandoffJSON, "agy") {
		t.Fatal("role handoff must not hardcode a provider")
	}
	var handoff PhaseHandoff
	if err := json.Unmarshal([]byte(flow.Steps[3].HandoffJSON), &handoff); err != nil || handoff.NextExpectedAction == "" || handoff.ContextBudgetBytes <= 0 {
		t.Fatalf("semantic handoff is not bounded/structured: %s err=%v", flow.Steps[3].HandoffJSON, err)
	}
	roundTrip := WorkPlanFromFlow(*flow)
	if !reflect.DeepEqual(roundTrip.Phases[1].Packages[0].HandoffJSON, flow.Steps[1].HandoffJSON) {
		t.Fatal("handoff was lost across WorkPlan/Flow projection")
	}
}

func TestMissionTeamRuntimeOverrideRemainsPreferenceAndRoundTrips(t *testing.T) {
	selection := MissionTeamSelection{Version: MissionTeamVersion, Strategy: IntentPlan, Preset: PresetFeatureDelivery, ReviewRequired: true}
	flow := &FlowDefinition{ID: "flow", ProjectID: "project", StructuredFacts: map[string]string{}}
	override := map[string]RuntimeAffinityPolicy{
		"planning":       {Provider: AffinityPreference{Mode: AffinityPrefer, Value: "codex"}},
		"IMPLEMENTATION": {Provider: AffinityPreference{Mode: AffinityPrefer, Value: "agy"}},
	}
	if err := BuildMissionTeamFlow(flow, selection, "feature", nil, nil, nil, override); err != nil {
		t.Fatal(err)
	}
	if flow.Steps[0].DesiredProvider != "codex" || flow.Steps[1].DesiredProvider != "agy" {
		t.Fatalf("runtime preference was not projected by phase: planning=%q implementation=%q", flow.Steps[0].DesiredProvider, flow.Steps[1].DesiredProvider)
	}
	var requirements TaskRequirements
	if err := json.Unmarshal([]byte(flow.Steps[0].TaskRequirements), &requirements); err != nil {
		t.Fatal(err)
	}
	if requirements.RuntimeAffinity == nil || requirements.RuntimeAffinity.Provider.Value != "codex" {
		t.Fatalf("task requirements lost runtime preference: %+v", requirements.RuntimeAffinity)
	}
	plan := WorkPlanFromFlow(*flow)
	if plan.Phases[0].Packages[0].DesiredProvider != "codex" || plan.Phases[1].Packages[0].DesiredProvider != "agy" {
		t.Fatalf("runtime preference was lost across Flow/WorkPlan projection: %+v", plan.Phases)
	}

}

func TestMissionTeamOperationalRolesUseCanonicalAgentModes(t *testing.T) {
	bugFix, err := MissionTeamPresetDefinitionFor(PresetBugFix)
	if err != nil {
		t.Fatal(err)
	}
	if len(bugFix.Phases) == 0 || bugFix.Phases[0].RequiredRole != "investigator" {
		t.Fatalf("bug-fix diagnosis role=%q, want investigator", bugFix.Phases[0].RequiredRole)
	}

	release, err := MissionTeamPresetDefinitionFor(PresetReleaseCandidate)
	if err != nil {
		t.Fatal(err)
	}
	foundSecurity := false
	for _, phase := range release.Phases {
		if phase.Type == PhaseSecurityReview {
			foundSecurity = true
			if phase.RequiredRole != "security-reviewer" {
				t.Fatalf("security review role=%q, want security-reviewer", phase.RequiredRole)
			}
		}
	}
	if !foundSecurity {
		t.Fatal("release candidate preset must contain a security review phase")
	}
}

