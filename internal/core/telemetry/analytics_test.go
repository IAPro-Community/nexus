package telemetry

import (
	"testing"
)

func TestAnalyticsDisabledByDefaultDropsEverything(t *testing.T) {
	t.Setenv("NEXUS_ANALYTICS", "")
	var sunk []string
	SetAnalyticsSink(func(name string, _ map[string]string) { sunk = append(sunk, name) })
	defer SetAnalyticsSink(nil)
	Track("app_started", map[string]string{"os": "linux"})
	if len(sunk) != 0 {
		t.Fatal("disabled analytics must not reach the sink")
	}
	if n := AnalyticsCounts()["app_started"]; n != 0 {
		t.Fatalf("disabled analytics must not count, got %d", n)
	}
}

func TestAnalyticsOptInAllowlistsNames(t *testing.T) {
	t.Setenv("NEXUS_ANALYTICS", "1")
	var sunk []string
	SetAnalyticsSink(func(name string, _ map[string]string) { sunk = append(sunk, name) })
	defer SetAnalyticsSink(nil)
	Track("mission_started", map[string]string{"provider": "codex"})
	Track("user typed a secret prompt", map[string]string{"prompt": "sk-..."})
	Track("mission_started", map[string]string{"oversized-key-" + string(make([]byte, 200)): "x"})
	counts := AnalyticsCounts()
	if counts["mission_started"] < 2 {
		t.Fatalf("allowlisted events must count: %+v", counts)
	}
	if len(sunk) != 2 {
		t.Fatalf("only allowlisted events reach sink: %v", sunk)
	}
	if _, ok := counts["user typed a secret prompt"]; ok {
		t.Fatal("free-form names must never be recorded")
	}
}

func TestAnalyticsExplicitOptOutWins(t *testing.T) {
	t.Setenv("NEXUS_ANALYTICS", "0")
	if AnalyticsEnabled() {
		t.Fatal("explicit 0 must disable")
	}
}
