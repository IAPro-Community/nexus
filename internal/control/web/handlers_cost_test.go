package web

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestCostEndpointsPreserveUnknownEvidence(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)

	client, srv, _ := csrfClient(t)
	st := nexus.Default().Store()
	if st == nil {
		t.Fatal("cost endpoint test requires Nexus store")
	}
	now := time.Now().UTC()
	knownCost := int64(12500)
	input := int64(400)
	output := int64(100)
	for _, obs := range []store.CostObservation{
		{
			ProjectID:    "project-a",
			MissionRunID: "mission-a",
			AgentID:      "agent-a",
			Provider:     "codex",
			Profile:      "work",
			InputTokens:  &input,
			OutputTokens: &output,
			DurationMS:   500,
			CostMicroUSD: &knownCost,
			CostKind:     store.CostKindProviderReported,
			Source:       "provider_response",
			Status:       "COMPLETED",
			ObservedAt:   now.Add(-time.Minute),
		},
		{
			ProjectID:    "project-a",
			MissionRunID: "mission-b",
			AgentID:      "agent-b",
			Provider:     "claude",
			Profile:      "default",
			DurationMS:   900,
			CostKind:     store.CostKindUnknown,
			Source:       "runtime_observation",
			Status:       "COMPLETED",
			ObservedAt:   now,
		},
	} {
		if _, err := st.RecordCostObservation(obs); err != nil {
			t.Fatal(err)
		}
	}

	resp, err := client.Get(srv.URL() + "/api/v1/cost/summary?days=7&project_id=project-a")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("summary status=%d, want 200", resp.StatusCode)
	}
	var summary nexus.CostSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		t.Fatal(err)
	}
	if summary.Aggregate.ObservationCount != 2 || summary.Aggregate.CostMicroUSD != knownCost {
		t.Fatalf("summary aggregate=%+v", summary.Aggregate)
	}
	if summary.CostEvidence != nexus.CostEvidencePartial || summary.TokenEvidence != nexus.CostEvidencePartial {
		t.Fatalf("evidence cost=%s tokens=%s, want PARTIAL/PARTIAL", summary.CostEvidence, summary.TokenEvidence)
	}

	resp2, err := client.Get(srv.URL() + "/api/v1/cost/observations?mission_run_id=mission-b")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	var observations []store.CostObservation
	if err := json.NewDecoder(resp2.Body).Decode(&observations); err != nil {
		t.Fatal(err)
	}
	if len(observations) != 1 || observations[0].CostKind != store.CostKindUnknown {
		t.Fatalf("filtered observations=%+v", observations)
	}
	if observations[0].CostMicroUSD != nil || observations[0].InputTokens != nil {
		t.Fatalf("unknown evidence was serialized as measured: %+v", observations[0])
	}
}

func TestCostSummaryRejectsInvalidDays(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)
	client, srv, _ := csrfClient(t)
	resp, err := client.Get(srv.URL() + "/api/v1/cost/summary?days=nope")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", resp.StatusCode)
	}
}

func TestCostPreferencesEndpointsRoundTripAndRejectInvalidPolicy(t *testing.T) {
	nexus.ResetDefaultForTest()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_STATE_DIR", t.TempDir())
	t.Cleanup(nexus.ResetDefaultForTest)

	client, srv, csrf := csrfClient(t)

	resp, err := client.Get(srv.URL() + "/api/v1/cost/preferences")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("GET preferences status=%d, want 200", resp.StatusCode)
	}
	var defaults store.CostPreferences
	if err := json.NewDecoder(resp.Body).Decode(&defaults); err != nil {
		resp.Body.Close()
		t.Fatal(err)
	}
	resp.Body.Close()
	if !defaults.Enabled || defaults.WindowDays != 7 || defaults.SoftLimitMicroUSD != 0 {
		t.Fatalf("unexpected cost defaults: %+v", defaults)
	}

	prefs := defaults
	prefs.SoftLimitMicroUSD = 1_000_000
	prefs.HardLimitMicroUSD = 2_000_000
	prefs.EmergencyLimitMicroUSD = 3_000_000
	body, _ := json.Marshal(prefs)
	resp = putJSON(t, client, csrf, srv.URL()+"/api/v1/cost/preferences", body)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("PUT preferences status=%d, want 200", resp.StatusCode)
	}
	var saved store.CostPreferences
	if err := json.NewDecoder(resp.Body).Decode(&saved); err != nil {
		resp.Body.Close()
		t.Fatal(err)
	}
	resp.Body.Close()
	if saved.HardLimitMicroUSD != 2_000_000 || saved.EmergencyLimitMicroUSD != 3_000_000 {
		t.Fatalf("saved preferences=%+v", saved)
	}

	invalid := saved
	invalid.SoftLimitMicroUSD = 2_000_000
	invalid.HardLimitMicroUSD = 1_000_000
	body, _ = json.Marshal(invalid)
	resp = putJSON(t, client, csrf, srv.URL()+"/api/v1/cost/preferences", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("inverted thresholds status=%d, want 400", resp.StatusCode)
	}
}
