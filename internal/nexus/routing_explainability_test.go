package nexus

import (
	"encoding/json"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestDeriveRoutingSwitchReasonTyped(t *testing.T) {
	cases := []struct {
		name   string
		fb     bool
		reason string
		kind   model.FailureKind
		msg    string
		want   RoutingSwitchReason
	}{
		{"no fallback", false, "selected", "", "", SwitchReasonNone},
		{"quota", true, "quota exhausted", model.FailureQuota, "", SwitchReasonQuotaExhausted},
		{"rate", true, "429", model.FailureRateLimit, "", SwitchReasonRateLimited},
		{"cooldown", true, "cooldown until 15:04", model.FailureRateLimit, "cooldown", SwitchReasonCooldown},
		{"unauth", true, "login required", model.FailureAuth, "", SwitchReasonUnauthenticated},
		{"preferred", true, "preferred provider/profile unavailable; scheduler fallback permitted", "", "", SwitchReasonPreferredUnavailable},
		{"model", true, `preferred model "cheap" unavailable or incapable; capable fallback selected`, "", "", SwitchReasonModelIncapable},
		{"unhealthy prose", true, "provider unhealthy", "", "", SwitchReasonUnhealthy},
	}
	for _, tc := range cases {
		if got := DeriveRoutingSwitchReason(tc.fb, tc.reason, tc.kind, tc.msg); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestRoutingProjectionPreservesExplainability(t *testing.T) {
	decision := buildMissionRoutingDecision(
		&runner.PackageRun{
			PackageID: "pkg-1", Attempt: 2, RetryFrom: runner.StateAllocating,
			FailureKind: model.FailureQuota, ErrorMessage: "quota exhausted",
		},
		store.Agent{ID: "agent-1"},
		TaskRequirements{TaskKind: "coding"},
		ProviderAccount{Provider: "opencode", Profile: "qa", Health: "healthy", Authenticated: true, Available: true},
		AgentConfig{Model: "mimo"},
		"agy", "frontend", AffinityPrefer, true,
		"quota exhausted",
	)
	if decision.RequestedProvider != "agy" || decision.RequestedProfile != "frontend" {
		t.Fatalf("requested not projected: %+v", decision)
	}
	if decision.SelectedProvider != "opencode" || decision.SelectedProfile != "qa" || decision.SelectedModel != "mimo" {
		t.Fatalf("selected not projected: %+v", decision)
	}
	if !decision.Fallback {
		t.Fatalf("fallback flag missing")
	}
	if decision.Reason == "" {
		t.Fatalf("why missing")
	}
	if decision.SwitchReason != SwitchReasonQuotaExhausted {
		t.Fatalf("typed switch reason missing: got %q", decision.SwitchReason)
	}
	if len(decision.Rejected) == 0 {
		t.Fatalf("rejected missing")
	}
	raw, err := json.Marshal(decision)
	if err != nil {
		t.Fatal(err)
	}
	report, err := BuildRoutingDecisionReport(&runner.MissionRun{
		ID: "run-1", PlanID: "plan-1",
		PackageRuns: []runner.PackageRun{{PackageID: "pkg-1", Title: "T", State: runner.StateVerified, RoutingDecisionJSON: string(raw)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Decisions) != 1 {
		t.Fatalf("expected 1 decision")
	}
	got := report.Decisions[0].Decision
	if got.RequestedProvider != "agy" || got.SelectedProvider != "opencode" || !got.Fallback || got.SwitchReason != SwitchReasonQuotaExhausted {
		t.Fatalf("projection lost explainability: %+v", got)
	}
	if got.Reason == "" || len(got.Rejected) == 0 {
		t.Fatalf("projection lost why/rejected: %+v", got)
	}
}

func TestResolveRuntimeRoutingExplainabilityFields(t *testing.T) {
	policy := RuntimeAffinityPolicy{Provider: AffinityPreference{Mode: AffinityPrefer, Value: "missing"}}
	decision, err := ResolveRuntimeRouting(TaskRequirements{TaskKind: "coding"}, policy, testAccounts(), testModels())
	if err != nil {
		t.Fatal(err)
	}
	if decision.RequestedProvider != "missing" {
		t.Fatalf("requested provider missing: %+v", decision)
	}
	if decision.SelectedProvider == "" || decision.SelectedModel == "" {
		t.Fatalf("selected missing: %+v", decision)
	}
	if !decision.Fallback || decision.SwitchReason == SwitchReasonNone {
		t.Fatalf("fallback explainability missing: %+v", decision)
	}
	if decision.Reason == "" {
		t.Fatalf("why missing")
	}
	raw, err := json.Marshal(decision)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip RuntimeRoutingDecision
	if err := json.Unmarshal(raw, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if roundtrip.RequestedProvider != "missing" || roundtrip.SwitchReason != decision.SwitchReason {
		t.Fatalf("JSON roundtrip lost explainability: %+v", roundtrip)
	}
}
