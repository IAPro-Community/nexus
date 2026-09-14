package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestIntegrationServiceDisabledByDefault(t *testing.T) {
	st := openTestNexusStore(t)
	svc := NewIntegrationService(st)

	// Integration should be disabled by default.
	_, err := svc.Integrate(t.Context(), IntegrationRequest{
		BaseBranch: "main",
		HeadBranch: "feature",
		Workspace:  t.TempDir(),
		ProjectID:  "proj-1",
		AgentID:    "agent-1",
	})
	if err == nil {
		t.Fatal("expected error when integration is disabled")
	}
	if err.Error() != "integration is not enabled; set integration_opt_in=true in conductor policy" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegrationServiceEnabledViaPolicy(t *testing.T) {
	st := openTestNexusStore(t)

	// Enable integration in policy.
	policy := DefaultConductorPolicy()
	v := true
	policy.IntegrationOptIn = &v
	if err := PutConductorPolicy(st, policy); err != nil {
		t.Fatal(err)
	}

	svc := NewIntegrationService(st)

	// Should not error when integration is enabled (will fail on invalid workspace, but that's expected).
	_, err := svc.Integrate(t.Context(), IntegrationRequest{
		BaseBranch: "main",
		HeadBranch: "feature",
		Workspace:  "/nonexistent",
		ProjectID:  "proj-1",
		AgentID:    "agent-1",
	})
	// Will fail because workspace doesn't exist, but the policy check should pass.
	if err != nil && err.Error() == "integration is not enabled; set integration_opt_in=true in conductor policy" {
		t.Fatal("integration should be enabled in policy")
	}
}

func TestIntegrationReceiptRecording(t *testing.T) {
	st := openTestNexusStore(t)
	project, err := st.CreateProject(store.Project{Name: "Integ", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	// Record a mock receipt.
	_, err = st.RecordEventMetadata(store.EventMetadata{
		AgentID:   "agent-1",
		ProjectID: project.ID,
		Kind:      "integration_receipt",
		Summary:   `{"base_sha":"abc123","head_sha":"def456","status":"SUCCESS"}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	events, err := st.ListEventsMetadata(project.ID, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != "integration_receipt" {
		t.Fatalf("expected integration_receipt, got %s", events[0].Kind)
	}
}
