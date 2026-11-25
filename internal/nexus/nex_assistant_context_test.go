package nexus

import (
	"strings"
	"testing"
)

func TestBuildNexContextSummaryRedactsAndBounds(t *testing.T) {
	state := HUDState{
		Providers: []HUDProvider{{Provider: "claude"}},
		Agents:    []HUDAgent{{ID: "a1"}},
		Cost:      &HUDCostSummary{BudgetLevel: "OK", Evidence: "PARTIAL"},
	}
	noPersonal := BuildNexContextSummary(state, "", 200)
	if strings.Contains(noPersonal, "personal=") || !strings.Contains(noPersonal, "providers=1") {
		t.Fatalf("unexpected summary without personal memory: %q", noPersonal)
	}

	secret := "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz"
	withPersonal := BuildNexContextSummary(state, "token="+secret+" "+strings.Repeat("x", 200), 120)
	if strings.Contains(withPersonal, secret) {
		t.Fatal("secret leaked into context summary")
	}
	if len(withPersonal) > 120 {
		t.Fatalf("context summary exceeded maxChars=120: %d", len(withPersonal))
	}
}
