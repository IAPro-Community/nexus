package web

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// challengeTestHandler builds a handler over a real store-backed nexus whose
// mission runs are pre-seeded through the persistence path the production
// recovery uses (UpsertMissionRun payload), then decodes them back. The
// second return value is the generated plan id the runs were attached to.
func challengeTestHandler(t *testing.T, runs []runner.MissionRun) (*NexusHandler, string) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "challenge.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	project, err := st.CreateProject(store.Project{Name: "Challenge Project", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "Challenge plan"})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}

	now := time.Now()
	for i := range runs {
		run := runs[i]
		if run.StartedAt.IsZero() {
			run.StartedAt = now
		}
		if strings.TrimSpace(run.ProjectID) == "" {
			run.ProjectID = project.ID
		}
		if strings.TrimSpace(run.PlanID) != "" {
			run.PlanID = plan.ID
		}
		run.UpdatedAt = now
		payload, err := json.Marshal(run)
		if err != nil {
			t.Fatalf("marshal run %s: %v", run.ID, err)
		}
		if err := st.UpsertMissionRun(store.MissionRunRecord{
			ID: run.ID, PlanID: run.PlanID, ProjectID: run.ProjectID, State: string(run.State),
			PayloadJSON: string(payload), CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			t.Fatalf("persist run %s: %v", run.ID, err)
		}
	}

	service := nexus.NewWithStoreAndExecutor(st, nil)
	t.Cleanup(service.Shutdown)
	auth, _, err := NewAuthManagerWithStore("127.0.0.1", "0", t.TempDir())
	if err != nil {
		t.Fatalf("auth manager: %v", err)
	}
	return &NexusHandler{auth: auth, nexus: service}, plan.ID
}

// challengeFixtureRun builds one completed participant run. ProjectID is left
// empty so the harness fills the real project id.
func challengeFixtureRun(id, lead, workspace string, passed, failed int) runner.MissionRun {
	run := runner.MissionRun{
		ID:          id,
		PlanID:      "plan-challenge",
		LeadAgentID: lead,
		Workspace:   workspace,
		State:       runner.StateCompleted,
	}
	for i := 0; i < passed; i++ {
		run.PackageRuns = append(run.PackageRuns, runner.PackageRun{
			ID: id + "-pass", WorkReceipt: &runner.WorkReceipt{
				ID: id + "-pass-r", RunID: id, StepID: "s",
				Tests: []runner.VerificationResult{{Passed: true}},
			},
		})
	}
	for i := 0; i < failed; i++ {
		run.PackageRuns = append(run.PackageRuns, runner.PackageRun{
			ID: id + "-fail", WorkReceipt: &runner.WorkReceipt{
				ID: id + "-fail-r", RunID: id, StepID: "s",
				Tests: []runner.VerificationResult{{Passed: false}},
			},
		})
	}
	return run
}

func TestPlanParallelChallengeEndpoint(t *testing.T) {
	handler, planID := challengeTestHandler(t, []runner.MissionRun{
		challengeFixtureRun("run-a", "lead-a", "/wt/a", 4, 0),
		challengeFixtureRun("run-b", "lead-b", "/wt/b", 2, 2),
	})
	body, err := json.Marshal(map[string]any{
		"criteria": []map[string]any{{"name": "tests", "kind": "tests", "weight": 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/plans/"+planID+"/parallel-challenge", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.handlePlanParallelChallenge(rec, req)
	if rec.Code != 200 {
		t.Fatalf("challenge must evaluate: %d (%s)", rec.Code, rec.Body.String())
	}
	var payload struct {
		Result nexus.ChallengeResult `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Result.Participants != 2 {
		t.Fatalf("participants=%d want 2", payload.Result.Participants)
	}
	if payload.Result.AutoMergeAllowed {
		t.Fatal("challenge must never auto-merge")
	}
	if len(payload.Result.Ranking) == 0 || payload.Result.Ranking[0] != "lead-a" {
		t.Fatalf("ranking=%v want lead-a first", payload.Result.Ranking)
	}
}

func TestPlanParallelChallengeEndpointSingleRunFails(t *testing.T) {
	handler, planID := challengeTestHandler(t, []runner.MissionRun{
		challengeFixtureRun("run-a", "lead-a", "/wt/a", 1, 0),
	})
	body := strings.NewReader(`{"criteria":[{"name":"tests","kind":"tests","weight":1}]}`)
	req := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/plans/"+planID+"/parallel-challenge", body)
	rec := httptest.NewRecorder()
	handler.handlePlanParallelChallenge(rec, req)
	if rec.Code != 422 {
		t.Fatalf("single completed run must be refused with 422: %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestPlanParallelChallengeEndpointMethodNotAllowed(t *testing.T) {
	handler, planID := challengeTestHandler(t, nil)
	req := httptest.NewRequest("GET", "http://127.0.0.1/api/v1/plans/"+planID+"/parallel-challenge", nil)
	rec := httptest.NewRecorder()
	handler.handlePlanParallelChallenge(rec, req)
	if rec.Code != 405 {
		t.Fatalf("GET must be refused: %d", rec.Code)
	}
}

func TestPlanParallelChallengeEndpointInvalidBody(t *testing.T) {
	handler, planID := challengeTestHandler(t, nil)
	req := httptest.NewRequest("POST", "http://127.0.0.1/api/v1/plans/"+planID+"/parallel-challenge", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()
	handler.handlePlanParallelChallenge(rec, req)
	if rec.Code != 400 {
		t.Fatalf("invalid JSON must be refused: %d", rec.Code)
	}
}
