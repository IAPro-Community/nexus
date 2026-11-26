package nexus

import (
	"context"
	"errors"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
)

func TestAdvancedFeatureRegistryRejectsUnknownAndImplicitFlags(t *testing.T) {
	t.Setenv("NEXUS_FEATURES_ENABLED", "true")
	t.Setenv("NEXUS_FEATURE_UNKNOWN", "true")
	t.Setenv("NEXUS_FEATURE_MISSIONS", "")
	if AdvancedFeatureEnabled("UNKNOWN") || AdvancedFeatureEnabled(FeatureMissions) {
		t.Fatal("unregistered or unset features must remain disabled")
	}
	t.Setenv("NEXUS_FEATURE_MISSIONS", "true")
	if !AdvancedFeatureEnabled(FeatureMissions) || !AdvancedFeatureSnapshot().Features[FeatureMissions] {
		t.Fatal("explicit opt-in must agree with effective snapshot")
	}
	t.Setenv("NEXUS_FEATURES_ENABLED", "false")
	for feature, enabled := range AdvancedFeatureSnapshot().Features {
		if enabled {
			t.Fatalf("global gate failed for %s", feature)
		}
	}
}

func TestNexV2EnabledDefaultsTrueWithExplicitOptOut(t *testing.T) {
	t.Setenv("NEXUS_FEATURES_ENABLED", "")
	t.Setenv("NEXUS_FEATURE_NEX_V2", "")
	if !NexV2Enabled() {
		t.Fatal("NEX_V2 must default to enabled")
	}
	t.Setenv("NEXUS_FEATURES_ENABLED", "false")
	if !NexV2Enabled() {
		t.Fatal("NEX_V2 must remain enabled when optional experimental flags are disabled")
	}
	t.Setenv("NEXUS_FEATURE_NEX_V2", "false")
	if NexV2Enabled() {
		t.Fatal("NEXUS_FEATURE_NEX_V2=false must disable NEX_V2 for rollback")
	}
}

func TestDisabledMissionsCannotReachDomainSideEffects(t *testing.T) {
	t.Setenv("NEXUS_FEATURES_ENABLED", "false")
	t.Setenv("NEXUS_FEATURE_MISSIONS", "true")
	// No store or executor: all guarded paths must fail before touching either.
	n := &Nexus{}
	ctx := context.Background()
	contract := runner.DefaultAutonomyContract()
	_, err := n.StartMissionRun(ctx, "plan", "", contract, true)
	if !errors.Is(err, ErrAdvancedFeatureDisabled) {
		t.Fatalf("start mission: %v", err)
	}
	_, err = n.StartMissionRunApprovedWithOptions(ctx, "plan", 1, "", contract, true, MissionAdmissionOptions{})
	if !errors.Is(err, ErrAdvancedFeatureDisabled) {
		t.Fatalf("approved start: %v", err)
	}
	_, err = n.StartDispatch(ctx, DispatchRequest{})
	if !errors.Is(err, ErrAdvancedFeatureDisabled) {
		t.Fatalf("dispatch: %v", err)
	}
	_, err = n.ResumeMissionRun(ctx, "legacy")
	if !errors.Is(err, ErrAdvancedFeatureDisabled) {
		t.Fatalf("resume: %v", err)
	}
	_, _, err = n.ResolveMissionInterventionWithOutcome(ctx, "legacy", "decision", 1, "approve", "user")
	if !errors.Is(err, ErrAdvancedFeatureDisabled) {
		t.Fatalf("intervention: %v", err)
	}
	_, err = n.ScheduleMissionWithOptions(ctx, "plan", 1, ScheduleWhenResources, nil, "", "", contract, MissionAdmissionOptions{})
	if !errors.Is(err, ErrAdvancedFeatureDisabled) {
		t.Fatalf("schedule: %v", err)
	}
	if w := n.startMissionWorker("legacy", false); w != nil {
		t.Fatal("disabled mission must not start a worker")
	}
	if err := n.RecoverMissionRuns(ctx); err != nil {
		t.Fatalf("disabled recovery should be a no-op: %v", err)
	}
	n.StartScheduleLoop()
	if n.scheduleCancel != nil {
		t.Fatal("disabled scheduler started")
	}
}
