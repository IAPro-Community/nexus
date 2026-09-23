package nexus

import (
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestParseExecutionCoordinationRequests(t *testing.T) {
	run := &runner.MissionRun{ID: "run-1"}
	pkg := &runner.PackageRun{PackageID: "source", AssignedAgent: "agent-a"}
	output := "I need one durable fact before continuing.\n" +
		`NEXUS_COORDINATION_REQUEST: {"operation":"NEED_INFO","subject":"architecture.auth_endpoint","objective":"resolve canonical endpoint","preferred_role":"investigator","required_capabilities":["read_repo"]}`

	requests, err := parseExecutionCoordinationRequests(output, run, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 {
		t.Fatalf("requests=%d want 1", len(requests))
	}
	req := requests[0]
	if req.Operation != runner.CoordinationNeedInfo || req.Subject != "architecture.auth_endpoint" {
		t.Fatalf("unexpected request: %+v", req)
	}
	if req.RunID != run.ID || req.SourcePackageID != pkg.PackageID || req.SourceAgentID != pkg.AssignedAgent {
		t.Fatalf("request lineage was not injected: %+v", req)
	}
	if req.ID != "" {
		t.Fatalf("executor parser must leave durable ID generation to MissionRunner, got %q", req.ID)
	}
}

func TestParseExecutionCoordinationRequestsFailsClosedOnMalformedMarker(t *testing.T) {
	run := &runner.MissionRun{ID: "run-1"}
	pkg := &runner.PackageRun{PackageID: "source", AssignedAgent: "agent-a"}
	_, err := parseExecutionCoordinationRequests(
		`NEXUS_COORDINATION_REQUEST: {"operation":"NEED_INFO","subject":`,
		run,
		pkg,
	)
	if err == nil {
		t.Fatal("malformed coordination marker must fail closed")
	}
}

func TestCoordinationReplaySafetyRequiresStableWorkspaceAndRestrictedContract(t *testing.T) {
	contract := runner.DefaultAutonomyContract()
	safe, reason := coordinationReplaySafety(contract, "same-fingerprint", "same-fingerprint")
	if !safe || reason != "" {
		t.Fatalf("default restricted contract with stable fingerprint should be replay-safe: safe=%v reason=%q", safe, reason)
	}

	contract.AllowExternalNetwork = true
	safe, reason = coordinationReplaySafety(contract, "same-fingerprint", "same-fingerprint")
	if safe || reason == "" {
		t.Fatalf("external network permission must block automatic replay: safe=%v reason=%q", safe, reason)
	}

	contract = runner.DefaultAutonomyContract()
	safe, reason = coordinationReplaySafety(contract, "before", "after")
	if safe || reason == "" {
		t.Fatalf("workspace mutation must block automatic replay: safe=%v reason=%q", safe, reason)
	}

	safe, reason = coordinationReplaySafety(contract, "", "after")
	if safe || reason == "" {
		t.Fatalf("missing fingerprint evidence must block automatic replay: safe=%v reason=%q", safe, reason)
	}
}
