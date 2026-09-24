package nexus

import (
	"strings"
	"testing"
)

func TestMissionAdmissionBlockersReportsExactFailures(t *testing.T) {
	report := &FlowPreflightReport{
		Ready: false,
		Checks: []FlowPreflightCheck{
			{Key: "dag_validity", Status: "PASS", Summary: "ok"},
			{Key: "worktree_isolation", Status: "FAIL", Summary: "execução autônoma exige DefaultIsolation=worktree no projeto"},
			{Key: "resources", Status: "FAIL", Summary: "nenhum recurso de provedor disponível para admissão strict"},
		},
	}
	got := missionAdmissionBlockers(report)
	for _, want := range []string{
		"worktree_isolation: execução autônoma exige DefaultIsolation=worktree no projeto",
		"resources: nenhum recurso de provedor disponível para admissão strict",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("blockers=%q missing %q", got, want)
		}
	}
	if strings.Contains(got, "dag_validity") {
		t.Fatalf("PASS checks must not leak into blockers: %q", got)
	}
}

func TestMissionAdmissionBlockersFailsClosedWithoutSpecificCheck(t *testing.T) {
	got := missionAdmissionBlockers(&FlowPreflightReport{Ready: false})
	if !strings.Contains(got, "not ready") {
		t.Fatalf("unexpected fallback blocker: %q", got)
	}
}
