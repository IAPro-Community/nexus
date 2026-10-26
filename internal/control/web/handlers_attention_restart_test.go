package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

type multiMissionAttentionExecutor struct {
	mu         sync.Mutex
	allocates  map[string]int
	executions map[string]int
}

func newMultiMissionAttentionExecutor() *multiMissionAttentionExecutor {
	return &multiMissionAttentionExecutor{
		allocates:  make(map[string]int),
		executions: make(map[string]int),
	}
}

func (e *multiMissionAttentionExecutor) Allocate(_ context.Context, run *runner.MissionRun, pkg *runner.PackageRun) (runner.AllocationResult, error) {
	e.mu.Lock()
	e.allocates[pkg.PackageID]++
	e.mu.Unlock()
	return runner.AllocationResult{AgentID: "agent-" + pkg.PackageID, Workspace: run.Workspace}, nil
}

func (*multiMissionAttentionExecutor) Compile(_ context.Context, _ *runner.MissionRun, pkg *runner.PackageRun) (runner.PromptArtifact, error) {
	return runner.PromptArtifact{VersionID: "prompt-" + pkg.PackageID, Content: "deterministic attention fixture"}, nil
}

func (e *multiMissionAttentionExecutor) Execute(_ context.Context, _ *runner.MissionRun, pkg *runner.PackageRun, _ string) (runner.ExecutionResult, error) {
	e.mu.Lock()
	e.executions[pkg.PackageID]++
	attempt := e.executions[pkg.PackageID]
	e.mu.Unlock()

	switch pkg.PackageID {
	case "pkg-a":
		return runner.ExecutionResult{}, errors.New("deterministic unknown provider outcome")
	case "pkg-c":
		if attempt == 1 {
			return runner.ExecutionResult{
				RuntimeID: "runtime-c-quota",
				Failure:   model.Failure{Kind: model.FailureRateLimit, Message: "fixture rate limit"},
			}, errors.New("deterministic typed rate limit")
		}
	case "pkg-e":
		return runner.ExecutionResult{}, runner.MarkDispatchNotSent(errors.New("deterministic pre-dispatch failure"))
	}
	return runner.ExecutionResult{RuntimeID: "runtime-" + pkg.PackageID}, nil
}

func (*multiMissionAttentionExecutor) Review(_ context.Context, _ *runner.MissionRun, _ *runner.PackageRun) (runner.ReviewVerdict, error) {
	return runner.ReviewVerdict{Approved: true, ReviewerAgentID: "reviewer-fixture", ReviewedAt: time.Now().UTC()}, nil
}

func (e *multiMissionAttentionExecutor) counts(packageID string) (allocates, executions int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.allocates[packageID], e.executions[packageID]
}

func newAttentionRouteServer(auth *AuthManager, service *nexus.Nexus) *httptest.Server {
	handler := &NexusHandler{
		auth:  auth,
		nexus: service,
		runs:  nexus.NewRunApplicationService(service),
	}
	server := &Server{auth: auth}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/attention", server.authMiddleware(handler.handleAttentionCenter))
	return httptest.NewServer(mux)
}

func requestAttentionGroup(t *testing.T, client *http.Client, baseURL, sessionID, query string) (int, runner.AttentionGroup) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/attention"+query, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sessionID != "" {
		request.Header.Set("Authorization", "Bearer "+sessionID)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("GET attention %q: %v", query, err)
	}
	defer response.Body.Close()
	var group runner.AttentionGroup
	if response.StatusCode == http.StatusOK {
		if err := json.NewDecoder(response.Body).Decode(&group); err != nil {
			t.Fatalf("decode attention response: %v", err)
		}
	}
	return response.StatusCode, group
}

func TestAttentionHTTPProjectionSurvivesRestartWithProjectIsolation(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "attention.db")
	authDir := t.TempDir()
	now := time.Now().UTC()

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	projectA, err := st.CreateProject(store.Project{Name: "Attention Alpha", CanonicalPath: t.TempDir()})
	if err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	projectB, err := st.CreateProject(store.Project{Name: "Attention Beta", CanonicalPath: t.TempDir()})
	if err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	planA, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: projectA.ID, Title: "Alpha plan"})
	if err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	planB, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: projectB.ID, Title: "Beta plan"})
	if err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	completedAt := now.Add(-time.Minute)
	runs := []*runner.MissionRun{
		{
			ID: "mission-alpha-needs-user", ProjectID: projectA.ID, PlanID: planA.ID,
			State: runner.StateBlockedNeedsUser, StartedAt: now, UpdatedAt: now,
			NeedsHuman: &runner.HumanIntervention{
				ID: "intervention-alpha", MissionID: "mission-alpha-needs-user", Timestamp: now,
				Version: 1, Question: "Choose the safe next step.",
			},
		},
		{
			ID: "mission-beta-completed", ProjectID: projectB.ID, PlanID: planB.ID,
			State: runner.StateCompletedVerified, CompletedAt: &completedAt, UpdatedAt: completedAt,
		},
		{
			ID: "mission-beta-failed", ProjectID: projectB.ID, PlanID: planB.ID,
			State: runner.StateFailedNoProgress, Contract: runner.AutonomyContract{EscalateOnFailure: true},
			UpdatedAt: now,
		},
		{
			ID: "mission-alpha-executing", ProjectID: projectA.ID, PlanID: planA.ID,
			State: runner.StateExecuting, UpdatedAt: now,
		},
	}
	for _, run := range runs {
		payload, marshalErr := json.Marshal(run)
		if marshalErr != nil {
			_ = st.Close()
			t.Fatalf("marshal %s: %v", run.ID, marshalErr)
		}
		if upsertErr := st.UpsertMissionRun(store.MissionRunRecord{
			ID: run.ID, PlanID: run.PlanID, ProjectID: run.ProjectID, State: string(run.State),
			PayloadJSON: string(payload), CreatedAt: now, UpdatedAt: run.UpdatedAt,
		}); upsertErr != nil {
			_ = st.Close()
			t.Fatalf("persist %s: %v", run.ID, upsertErr)
		}
	}

	firstService := nexus.NewWithStoreAndExecutor(st, nil)
	t.Cleanup(firstService.Shutdown)
	firstAuth, _, err := NewAuthManagerWithStore("127.0.0.1", "0", authDir)
	if err != nil {
		firstService.Shutdown()
		t.Fatal(err)
	}
	session, err := firstAuth.CreateSession()
	if err != nil {
		firstService.Shutdown()
		t.Fatal(err)
	}
	firstAuth.SetDesktopSession(session)
	firstServer := newAttentionRouteServer(firstAuth, firstService)
	t.Cleanup(firstServer.Close)
	client := firstServer.Client()
	t.Cleanup(firstServer.Close)

	if status, _ := requestAttentionGroup(t, client, firstServer.URL, "", ""); status != http.StatusUnauthorized {
		t.Fatalf("unauthenticated global attention status=%d, want 401", status)
	}
	assertAttentionProjection(t, client, firstServer.URL, session.ID, projectA.ID, projectB.ID)

	firstServer.Close()
	if err := firstService.ShutdownContext(ctx); err != nil {
		t.Fatalf("shutdown first attention service: %v", err)
	}

	reopened, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen attention store: %v", err)
	}
	secondService := nexus.NewWithStoreAndExecutor(reopened, nil)
	t.Cleanup(secondService.Shutdown)
	secondAuth, _, err := NewAuthManagerWithStore("127.0.0.1", "0", authDir)
	if err != nil {
		secondService.Shutdown()
		t.Fatal(err)
	}
	secondServer := newAttentionRouteServer(secondAuth, secondService)
	t.Cleanup(secondServer.Close)
	assertAttentionProjection(t, secondServer.Client(), secondServer.URL, session.ID, projectA.ID, projectB.ID)
	if err := secondService.ShutdownContext(ctx); err != nil {
		t.Fatalf("shutdown restarted attention service: %v", err)
	}
}

func TestAttentionHTTPMultiMissionQuotaRecoverySurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "attention-multi-mission.db")
	authDir := t.TempDir()
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	executor := newMultiMissionAttentionExecutor()
	firstService := nexus.NewWithStoreAndExecutor(st, executor)
	firstClosed := false
	t.Cleanup(func() {
		if !firstClosed {
			_ = firstService.ShutdownContext(context.Background())
		}
	})

	type missionFixture struct {
		id      string
		project store.Project
		plan    *store.WorkPlan
	}
	fixtures := make([]missionFixture, 0, 5)
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		project, createErr := st.CreateProject(store.Project{Name: "Attention " + id, CanonicalPath: t.TempDir()})
		if createErr != nil {
			t.Fatal(createErr)
		}
		plan, createErr := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "Mission " + id})
		if createErr != nil {
			t.Fatal(createErr)
		}
		fixtures = append(fixtures, missionFixture{id: id, project: project, plan: plan})
	}

	newRun := func(fixture missionFixture, maxRetries int) *runner.MissionRun {
		contract := runner.DefaultAutonomyContract()
		contract.RequireVerification = false
		contract.MaxRetries = maxRetries
		contract.MaxNoProgress = 3
		contract.AutoRemediate = true
		contract.EscalateOnFailure = fixture.id != "e"
		run, startErr := firstService.Runner().StartMissionRun(ctx, runner.PlanSpec{
			ID: fixture.plan.ID, ProjectID: fixture.project.ID, Revision: fixture.plan.CurrentRevision,
			Packages: []runner.PackageSpec{{ID: "pkg-" + fixture.id, Title: "Mission " + fixture.id, Goal: "Exercise durable attention state"}},
		}, fixture.project.CanonicalPath, contract, "")
		if startErr != nil {
			t.Fatalf("start mission %s: %v", fixture.id, startErr)
		}
		return run
	}
	step := func(runID string) (*runner.MissionRun, error) {
		run, _, stepErr := firstService.Runner().ExecuteNextStep(ctx, runID)
		return run, stepErr
	}
	advanceToExecuting := func(run *runner.MissionRun) *runner.MissionRun {
		for i := 0; i < 3; i++ {
			updated, stepErr := step(run.ID)
			if stepErr != nil {
				t.Fatalf("advance %s to EXECUTING at step %d: %v", run.ID, i+1, stepErr)
			}
			run = updated
		}
		if run.PackageRuns[0].State != runner.StateExecuting {
			t.Fatalf("mission %s package state=%s, want EXECUTING", run.ID, run.PackageRuns[0].State)
		}
		return run
	}
	advanceToTerminal := func(run *runner.MissionRun) *runner.MissionRun {
		for i := 0; i < 12 && !runner.IsTerminalState(run.State); i++ {
			updated, stepErr := step(run.ID)
			if updated != nil {
				run = updated
			}
			if stepErr != nil && !runner.IsTerminalState(run.State) {
				t.Fatalf("advance %s at state %s: %v", run.ID, run.State, stepErr)
			}
		}
		if !runner.IsTerminalState(run.State) {
			t.Fatalf("mission %s did not reach a terminal state: %+v", run.ID, run)
		}
		return run
	}

	runA := advanceToExecuting(newRun(fixtures[0], 3))
	if blocked, blockErr := step(runA.ID); blockErr == nil || blocked.State != runner.StateBlockedNeedsUser {
		t.Fatalf("mission A should block on unknown provider outcome: state=%v err=%v", blocked.State, blockErr)
	}
	runB := advanceToExecuting(newRun(fixtures[1], 3))
	runC := advanceToExecuting(newRun(fixtures[2], 3))
	if quotaRun, quotaErr := step(runC.ID); quotaErr != nil || quotaRun.State != runner.StateExecuting || quotaRun.PackageRuns[0].State != runner.StateRemediating || quotaRun.PackageRuns[0].RetryFrom != runner.StateAllocating {
		t.Fatalf("mission C quota failure did not persist a failover checkpoint: run=%+v err=%v", quotaRun, quotaErr)
	}
	runD := advanceToTerminal(advanceToExecuting(newRun(fixtures[3], 3)))
	if runD.State != runner.StateCompletedVerified {
		t.Fatalf("mission D state=%s, want COMPLETED_VERIFIED", runD.State)
	}
	runE := advanceToTerminal(advanceToExecuting(newRun(fixtures[4], 1)))
	if runE.State != runner.StateFailedNoProgress {
		t.Fatalf("mission E state=%s, want FAILED_NO_PROGRESS", runE.State)
	}
	if runB.State != runner.StateExecuting {
		t.Fatalf("mission B state=%s before restart, want EXECUTING", runB.State)
	}

	firstAuth, _, err := NewAuthManagerWithStore("127.0.0.1", "0", authDir)
	if err != nil {
		t.Fatal(err)
	}
	session, err := firstAuth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	firstAuth.SetDesktopSession(session)
	firstServer := newAttentionRouteServer(firstAuth, firstService)
	client := firstServer.Client()
	assertMultiMissionAttentionBeforeRestart(t, client, firstServer.URL, session.ID, runA.ID, runB.ID, runC.ID, runD.ID, runE.ID)
	if allocates, executions := executor.counts("pkg-c"); allocates != 1 || executions != 1 {
		t.Fatalf("mission C should have one initial allocation/attempt before restart, got allocations=%d executions=%d", allocates, executions)
	}

	firstServer.Close()
	if err := firstService.ShutdownContext(ctx); err != nil {
		t.Fatalf("shutdown first server/service: %v", err)
	}
	firstClosed = true

	reopened, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen multi-mission store: %v", err)
	}
	secondService := nexus.NewWithStoreAndExecutor(reopened, executor)
	secondClosed := false
	t.Cleanup(func() {
		if !secondClosed {
			_ = secondService.ShutdownContext(context.Background())
		}
	})
	secondAuth, _, err := NewAuthManagerWithStore("127.0.0.1", "0", authDir)
	if err != nil {
		t.Fatal(err)
	}
	secondServer := newAttentionRouteServer(secondAuth, secondService)
	t.Cleanup(secondServer.Close)
	if err := secondService.RecoverMissionRuns(ctx); err != nil {
		t.Fatalf("recover B/C after server restart: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	var recoveredB, recoveredC *runner.MissionRun
	for time.Now().Before(deadline) {
		recoveredB, err = secondService.Runner().GetRun(ctx, runB.ID)
		if err != nil {
			t.Fatal(err)
		}
		recoveredC, err = secondService.Runner().GetRun(ctx, runC.ID)
		if err != nil {
			t.Fatal(err)
		}
		if runner.IsTerminalState(recoveredB.State) && runner.IsTerminalState(recoveredC.State) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if recoveredB.State != runner.StateCompletedVerified || recoveredC.State != runner.StateCompletedVerified {
		t.Fatalf("B/C did not continue from durable checkpoints: B=%s C=%s", recoveredB.State, recoveredC.State)
	}
	if allocates, executions := executor.counts("pkg-b"); allocates != 1 || executions != 1 {
		t.Fatalf("B replayed completed allocation/provider work unexpectedly: allocations=%d executions=%d", allocates, executions)
	}
	if allocates, executions := executor.counts("pkg-c"); allocates != 2 || executions != 2 {
		t.Fatalf("C quota recovery did not reallocate and resume exactly once: allocations=%d executions=%d", allocates, executions)
	}
	assertMultiMissionAttentionAfterRestart(t, secondServer.Client(), secondServer.URL, session.ID, runA.ID, runB.ID, runC.ID, runD.ID, runE.ID)
	if err := secondService.ShutdownContext(ctx); err != nil {
		t.Fatalf("shutdown restarted service: %v", err)
	}
	secondClosed = true
}

func assertMultiMissionAttentionBeforeRestart(t *testing.T, client *http.Client, baseURL, sessionID string, expectedIDs ...string) {
	t.Helper()
	status, group := requestAttentionGroup(t, client, baseURL, sessionID, "")
	if status != http.StatusOK {
		t.Fatalf("multi-mission attention HTTP status=%d, want 200", status)
	}
	if len(group.AllItems) != 3 || len(group.NeedsYou) != 1 || len(group.Completed) != 1 || len(group.Failed) != 1 {
		t.Fatalf("pre-restart attention categories are incorrect: needs=%d completed=%d failed=%d", len(group.NeedsYou), len(group.Completed), len(group.Failed))
	}
	ids := attentionMissionIDs(group)
	for _, id := range []string{expectedIDs[0], expectedIDs[3], expectedIDs[4]} {
		if !ids[id] {
			t.Errorf("attention projection is missing expected mission %s", id)
		}
	}
	for _, id := range []string{expectedIDs[1], expectedIDs[2]} {
		if ids[id] {
			t.Errorf("active mission %s incorrectly generated an attention item", id)
		}
	}
}

func assertMultiMissionAttentionAfterRestart(t *testing.T, client *http.Client, baseURL, sessionID string, expectedIDs ...string) {
	t.Helper()
	status, group := requestAttentionGroup(t, client, baseURL, sessionID, "")
	if status != http.StatusOK {
		t.Fatalf("restarted multi-mission attention HTTP status=%d, want 200", status)
	}
	if len(group.AllItems) != 5 || len(group.NeedsYou) != 1 || len(group.Completed) != 3 || len(group.Failed) != 1 {
		t.Fatalf("post-restart attention categories are incorrect: needs=%d completed=%d failed=%d", len(group.NeedsYou), len(group.Completed), len(group.Failed))
	}
	states := make(map[string]runner.State, len(group.AllItems))
	for _, item := range group.AllItems {
		states[item.MissionID] = item.State
	}
	want := map[string]runner.State{
		expectedIDs[0]: runner.StateBlockedNeedsUser,
		expectedIDs[1]: runner.StateCompletedVerified,
		expectedIDs[2]: runner.StateCompletedVerified,
		expectedIDs[3]: runner.StateCompletedVerified,
		expectedIDs[4]: runner.StateFailedNoProgress,
	}
	for missionID, state := range want {
		if got := states[missionID]; got != state {
			t.Errorf("mission %s state after restart=%s, want %s", missionID, got, state)
		}
	}
}

func assertAttentionProjection(t *testing.T, client *http.Client, baseURL, sessionID, projectAID, projectBID string) {
	t.Helper()
	status, global := requestAttentionGroup(t, client, baseURL, sessionID, "")
	if status != http.StatusOK {
		t.Fatalf("authenticated global attention status=%d, want 200", status)
	}
	if len(global.AllItems) != 3 || len(global.NeedsYou) != 1 || len(global.Completed) != 1 || len(global.Failed) != 1 || global.TotalNeeds != 1 {
		t.Fatalf("global attention categories after restart are incorrect: %+v", global)
	}
	globalIDs := attentionMissionIDs(global)
	for _, id := range []string{"mission-alpha-needs-user", "mission-beta-completed", "mission-beta-failed"} {
		if !globalIDs[id] {
			t.Errorf("global attention is missing %s", id)
		}
	}
	if globalIDs["mission-alpha-executing"] {
		t.Error("executing mission leaked into global attention")
	}

	status, alpha := requestAttentionGroup(t, client, baseURL, sessionID, "?project_id="+projectAID)
	if status != http.StatusOK || len(alpha.AllItems) != 1 || alpha.AllItems[0].MissionID != "mission-alpha-needs-user" {
		t.Fatalf("Alpha project filter crossed project boundaries: status=%d group=%+v", status, alpha)
	}
	status, beta := requestAttentionGroup(t, client, baseURL, sessionID, "?project_id="+projectBID)
	if status != http.StatusOK || len(beta.AllItems) != 2 || len(beta.Completed) != 1 || len(beta.Failed) != 1 {
		t.Fatalf("Beta project filter returned incorrect categories: status=%d group=%+v", status, beta)
	}
	status, mission := requestAttentionGroup(t, client, baseURL, sessionID, "?project_id="+projectAID+"&mission_id=mission-alpha-needs-user")
	if status != http.StatusOK || len(mission.AllItems) != 1 || mission.AllItems[0].ProjectID != projectAID {
		t.Fatalf("combined project/mission filter returned cross-project data: status=%d group=%+v", status, mission)
	}
	status, conflicting := requestAttentionGroup(t, client, baseURL, sessionID, "?project_id="+projectAID+"&mission_id=mission-beta-completed")
	if status != http.StatusOK || len(conflicting.AllItems) != 0 {
		t.Fatalf("conflicting project/mission filter must be empty: status=%d group=%+v", status, conflicting)
	}
}

func attentionMissionIDs(group runner.AttentionGroup) map[string]bool {
	ids := make(map[string]bool, len(group.AllItems))
	for _, item := range group.AllItems {
		if item != nil {
			ids[item.MissionID] = true
		}
	}
	return ids
}
