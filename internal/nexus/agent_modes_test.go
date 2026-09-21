package nexus

import "testing"

func TestCanonicalAgentModesCoverNexusOneContract(t *testing.T) {
	required := map[string]string{
		"developer": "developer",
		"senior-developer": "senior-developer",
		"reviewer": "reviewer",
		"investigator": "investigator",
		"architect": "architect",
		"security-reviewer": "security-reviewer",
		"qa-engineer": "qa-engineer",
		"performance-investigator": "performance-investigator",
		"documentation-specialist": "documentation-specialist",
	}
	modes := CanonicalAgentModes()
	for id, role := range required {
		mode, ok := AgentModeForID(id)
		if !ok {
			t.Fatalf("required Agent Mode %q is missing", id)
		}
		if mode.Role != role || mode.AgentSpec.Role != role {
			t.Fatalf("mode %s role mismatch: mode=%q spec=%q", id, mode.Role, mode.AgentSpec.Role)
		}
		if len(mode.AgentSpec.Instructions) == 0 || len(mode.AgentSpec.Responsibilities) == 0 || len(mode.AgentSpec.Capabilities) == 0 {
			t.Fatalf("mode %s lacks behavioral AgentSpec: %+v", id, mode.AgentSpec)
		}
	}
	if len(modes) < len(required) {
		t.Fatalf("mode catalog unexpectedly small: %d", len(modes))
	}
}

func TestAgentModesMarkInvestigationAndReviewReadOnlyByDefault(t *testing.T) {
	for _, id := range []string{"reviewer", "investigator", "security-reviewer", "performance-investigator"} {
		mode, ok := AgentModeForID(id)
		if !ok || !mode.ReadOnlyByDefault {
			t.Fatalf("%s must be read-only by default: %+v", id, mode)
		}
	}
}

func TestAgentModeRoleAliasesPreserveLegacyAgents(t *testing.T) {
	for legacy, want := range map[string]string{
		"code-reviewer": "reviewer",
		"docs-writer": "documentation-specialist",
		"technical-writer": "documentation-specialist",
	} {
		mode, ok := AgentModeForRole(legacy)
		if !ok || mode.ID != want {
			t.Fatalf("legacy role %q resolved to %+v, want %q", legacy, mode, want)
		}
	}
}

func TestCanonicalAgentModesReturnsDefensiveCopies(t *testing.T) {
	first := CanonicalAgentModes()
	first[0].AgentSpec.Instructions[0] = "mutated"
	first[0].RecommendedSkillIDs[0] = "mutated"
	second := CanonicalAgentModes()
	if second[0].AgentSpec.Instructions[0] == "mutated" || second[0].RecommendedSkillIDs[0] == "mutated" {
		t.Fatal("Agent Mode catalog leaked mutable slices")
	}
}
