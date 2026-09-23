package nexus

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestBuildHUDProjectionPreservesUnknown(t *testing.T) {
	state := BuildHUDProjection(HUDProjectionInput{
		GeneratedAt: time.Unix(10, 0).UTC(),
		Accounts: []ProviderAccount{{
			Provider: "codex",
			Profile:  "default",
			Health:   "unknown",
			QuotaView: &quota.QuotaView{
				Provider:     "codex",
				Profile:      "default",
				Status:       "UNKNOWN",
				AvailReasons: quota.AvailReasons{UnknownQuota: true},
			},
		}},
	})
	if len(state.Providers) != 1 {
		t.Fatalf("providers=%d, want 1", len(state.Providers))
	}
	got := state.Providers[0]
	if got.Health != "UNKNOWN" || got.Availability != "UNKNOWN" || got.LimitingWindow != nil {
		t.Fatalf("unknown state was invented: %+v", got)
	}
}

func TestBuildHUDProjectionUsesCanonicalQuotaBottleneck(t *testing.T) {
	now := time.Now().UTC()
	qv := quota.QuotaView{
		Provider:  "codex",
		Profile:   "work",
		Status:    "LIVE",
		Source:    "OFFICIAL",
		FetchedAt: now,
		Available: true,
		ModelGroups: []quota.ModelGroup{{
			Name: "Codex",
			Windows: []quota.Window{
				{Kind: "5h", Label: "5h", Remaining: 71, ResetDesc: "2h14", Status: "LIVE"},
				{Kind: "weekly", Label: "Weekly", Remaining: 82, ResetDesc: "Friday", Status: "LIVE"},
			},
		}},
	}
	state := BuildHUDProjection(HUDProjectionInput{
		Accounts: []ProviderAccount{{
			Provider: "codex", Profile: "work", Health: "healthy", Available: true, QuotaView: &qv,
		}},
	})
	got := state.Providers[0].LimitingWindow
	if got == nil || got.Kind != "5h" || got.RemainingPercent == nil || *got.RemainingPercent != 71 {
		t.Fatalf("limiting window=%+v, want canonical 5h/71", got)
	}
}

func TestBuildHUDProjectionKeepsCostProvenanceSeparate(t *testing.T) {
	reported := int64(1_000_000)
	estimated := int64(500_000)
	state := BuildHUDProjection(HUDProjectionInput{
		Cost: &CostSummary{
			GeneratedAt:  time.Unix(20, 0).UTC(),
			CostEvidence: CostEvidenceMeasured,
			Aggregate: store.CostAggregate{
				ObservationCount:      2,
				KnownCostObservations: 2,
				CostMicroUSD:          1_500_000,
			},
			Budget: CostBudgetAssessment{Level: CostBudgetOK},
		},
		CostObservations: []store.CostObservation{
			{CostMicroUSD: &reported, CostKind: store.CostKindProviderReported},
			{CostMicroUSD: &estimated, CostKind: store.CostKindAPIEquivalentEstimate},
		},
	})
	if state.Cost == nil || len(state.Cost.ByKind) != 2 {
		t.Fatalf("cost projection=%+v", state.Cost)
	}
	if state.Cost.ByKind[0].Kind == state.Cost.ByKind[1].Kind {
		t.Fatal("cost kinds were collapsed")
	}
}

func TestHUDPrimaryHumanInterventionBeatsWorkingAgent(t *testing.T) {
	run := &runner.MissionRun{
		ID:        "run-1",
		ProjectID: "project-1",
		State:     runner.StateBlockedNeedsUser,
		UpdatedAt: time.Now(),
		NeedsHuman: &runner.HumanIntervention{
			ID:         "intervention-1",
			ReasonCode: "REVIEW_REQUIRED",
			Resolved:   false,
		},
	}
	state := BuildHUDProjection(HUDProjectionInput{
		ProjectID: "project-1",
		Runs:      []*runner.MissionRun{run},
		Agents: []store.Agent{{
			ID: "agent-1", ProjectID: "project-1", Name: "architect", Status: "WORKING",
		}},
	})
	if state.Primary.Kind != HUDPrimaryMission || state.Primary.ID != "run-1" {
		t.Fatalf("primary=%+v, want mission", state.Primary)
	}
}

func TestHUDPrimaryCriticalAlertWins(t *testing.T) {
	remaining := 8.0
	state := BuildHUDProjection(HUDProjectionInput{
		Capacity: CapacityStatus{Resources: []CapacityResourceStatus{{
			Provider: "cursor", Profile: "work", State: "CRITICAL",
			Fingerprint: "cap:cursor:work", RemainingPercent: &remaining,
			LastObservedAt: time.Now(),
		}}},
		Runs: []*runner.MissionRun{{
			ID: "run-1", ProjectID: "project-1", State: runner.StateBlockedNeedsUser,
			NeedsHuman: &runner.HumanIntervention{ID: "h1", Resolved: false},
		}},
	})
	if state.Primary.Kind != HUDPrimaryAlert || state.Primary.ID != "cap:cursor:work" {
		t.Fatalf("primary=%+v, want critical alert", state.Primary)
	}
}

func TestLoadHUDRecommendationUsesDurableCapacityEvent(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "hud.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	payload, err := json.Marshal(map[string]any{
		"recommended_provider": "claude",
		"recommended_profile": "backup",
		"recommended_display_name": "Claude Backup",
		"recommendation_score": 82.5,
		"recommendation_reason": "healthy fallback",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.PutCapacityEvent(store.CapacityEventRecord{
		ID: "evt-1", Fingerprint: "fp-1", EventType: "QUOTA_LOW",
		Provider: "codex", Profile: "work", Severity: "CRITICAL",
		PayloadJSON: string(payload), EmittedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	rec := loadHUDRecommendation(st)
	if rec == nil {
		t.Fatal("expected durable recommendation")
	}
	if rec.Provider != "claude" || rec.Profile != "backup" || rec.Context != "CAPACITY_ALERT" {
		t.Fatalf("recommendation=%+v", rec)
	}
	if rec.Confidence != "UNKNOWN" {
		t.Fatalf("confidence=%q, want UNKNOWN when event has no confidence evidence", rec.Confidence)
	}
}
