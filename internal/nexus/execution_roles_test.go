package nexus

import "testing"

func TestCanonicalExecutionRolesAreBoundedAndProviderIndependent(t *testing.T) {
	roles := CanonicalExecutionRoles()
	if len(roles) != 6 {
		t.Fatalf("expected 6 canonical execution roles, got %d", len(roles))
	}
	for _, role := range roles {
		if role.ID == "" || role.BaseAgentModeID == "" {
			t.Fatalf("invalid execution role: %#v", role)
		}
		if _, ok := AgentModeForID(role.BaseAgentModeID); !ok {
			t.Fatalf("execution role %s references unknown AgentMode %s", role.ID, role.BaseAgentModeID)
		}
	}
}

func TestExecutionRoleForIDReturnsDefensiveCopy(t *testing.T) {
	role, ok := ExecutionRoleForID("review")
	if !ok {
		t.Fatal("review role not found")
	}
	role.RequiredCapabilities[0] = "mutated"

	again, ok := ExecutionRoleForID("review")
	if !ok {
		t.Fatal("review role not found on second lookup")
	}
	if again.RequiredCapabilities[0] == "mutated" {
		t.Fatal("canonical role catalog was mutated through returned slice")
	}
	if again.AccessIntent != AccessReadOnly {
		t.Fatalf("review must remain read-only, got %s", again.AccessIntent)
	}
}

func TestExecutionRoleIntelligenceContextUsesAgentModeRole(t *testing.T) {
	role, _ := ExecutionRoleForID("security")
	ctx := role.IntelligenceContext()
	if ctx == nil {
		t.Fatal("expected execution role context")
	}
	if ctx.ID != "security" || ctx.Role != "security-reviewer" || ctx.AccessIntent != "READ_ONLY" {
		t.Fatalf("unexpected context: %#v", ctx)
	}
}
