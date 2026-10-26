package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
	"github.com/IAPro-Community/nexus/internal/update"
)

func newTestClient(t *testing.T) (*http.Client, *Server) {
	t.Helper()
	srv, err := NewServer(ServerOptions{Host: "127.0.0.1", Port: 0})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	go func() { _ = srv.Start() }()
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	body := strings.NewReader(`{"token":"` + srv.bootstrap + `"}`)
	resp, err := client.Post(srv.URL()+"/api/v1/auth/bootstrap", "application/json", body)
	if err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bootstrap POST returned %d", resp.StatusCode)
	}
	return client, srv
}

type interventionRestartExecutor struct {
	blockFirstAllocation bool
	allocationStarted    chan struct{}
	effects              *atomic.Int32
}

func (e *interventionRestartExecutor) Allocate(ctx context.Context, _ *runner.MissionRun, _ *runner.PackageRun) (runner.AllocationResult, error) {
	if e.blockFirstAllocation {
		close(e.allocationStarted)
		<-ctx.Done()
		return runner.AllocationResult{}, ctx.Err()
	}
	return runner.AllocationResult{AgentID: "agent-restart-e2e", Workspace: "."}, nil
}

func (*interventionRestartExecutor) Compile(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.PromptArtifact, error) {
	return runner.PromptArtifact{VersionID: "prompt-restart-e2e", Content: "continue safely"}, nil
}

func (e *interventionRestartExecutor) Execute(context.Context, *runner.MissionRun, *runner.PackageRun, string) (runner.ExecutionResult, error) {
	e.effects.Add(1)
	return runner.ExecutionResult{RuntimeID: "runtime-restart-e2e"}, nil
}

func (*interventionRestartExecutor) Review(context.Context, *runner.MissionRun, *runner.PackageRun) (runner.ReviewVerdict, error) {
	return runner.ReviewVerdict{Approved: true, ReviewerAgentID: "reviewer-restart-e2e", ReviewedAt: time.Now().UTC()}, nil
}

func newInterventionRouteServer(auth *AuthManager, service *nexus.Nexus) *httptest.Server {
	handler := &NexusHandler{
		auth:  auth,
		nexus: service,
		runs:  nexus.NewRunApplicationService(service),
	}
	server := &Server{auth: auth}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/runs/", server.routeRun(handler))
	return httptest.NewServer(mux)
}

func requestResolveIntervention(t *testing.T, client *http.Client, baseURL, sessionID, csrfToken, runID string) *http.Response {
	t.Helper()
	body := strings.NewReader(`{"intervention_id":"intervention-http-restart","version":3,"option_id":"replan-package","resolved_by":"test-user"}`)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/runs/"+runID+"/resolve-intervention", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST resolve intervention: %v", err)
	}
	return resp
}

func TestResolveInterventionHTTPDecisionSurvivesServerRestartExactlyOnce(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "nexus.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "HTTP restart", CanonicalPath: t.TempDir()})
	if err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "HTTP restart plan"})
	if err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	now := time.Now().UTC()
	contract := runner.DefaultAutonomyContract()
	contract.RequireVerification = false
	contract.MaxTotalIterations = 20
	run := &runner.MissionRun{
		ID: "run-http-restart", PlanID: plan.ID, PlanRevision: plan.CurrentRevision, ProjectID: project.ID,
		State: runner.StateBlockedNeedsUser, Contract: contract, StartedAt: now, UpdatedAt: now,
		PackageRuns: []runner.PackageRun{{
			ID: "pkg-http-restart", PackageID: "pkg-http-restart", State: runner.StateFailed,
			DispatchState: runner.DispatchFailedBeforeDispatch,
		}},
		NeedsHuman: &runner.HumanIntervention{
			ID: "intervention-http-restart", Version: 3, MissionID: "run-http-restart",
			TaskID: "pkg-http-restart", Timestamp: now,
			Options: []runner.InterventionOption{{ID: "replan-package", Operation: runner.InterventionReplanPackage, PackageID: "pkg-http-restart"}},
		},
	}
	payload, err := json.Marshal(run)
	if err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	if err := st.UpsertMissionRun(store.MissionRunRecord{
		ID: run.ID, PlanID: run.PlanID, ProjectID: run.ProjectID, State: string(run.State),
		PayloadJSON: string(payload), CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		_ = st.Close()
		t.Fatal(err)
	}

	var effects atomic.Int32
	firstExecutor := &interventionRestartExecutor{blockFirstAllocation: true, allocationStarted: make(chan struct{}), effects: &effects}
	firstService := nexus.NewWithStoreAndExecutor(st, firstExecutor)
	t.Cleanup(firstService.Shutdown)
	authDir := t.TempDir()
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
	firstServer := newInterventionRouteServer(firstAuth, firstService)
	t.Cleanup(firstServer.Close)
	client := firstServer.Client()
	response := requestResolveIntervention(t, client, firstServer.URL, session.ID, session.CSRFToken, run.ID)
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		firstServer.Close()
		firstService.Shutdown()
		t.Fatalf("resolve status=%d, want %d", response.StatusCode, http.StatusOK)
	}
	response.Body.Close()
	select {
	case <-firstExecutor.allocationStarted:
	case <-time.After(2 * time.Second):
		firstServer.Close()
		firstService.Shutdown()
		t.Fatal("first worker did not stop at the pre-dispatch allocation barrier")
	}
	if got := effects.Load(); got != 0 {
		firstServer.Close()
		firstService.Shutdown()
		t.Fatalf("external side effects before simulated restart=%d, want 0", got)
	}

	duplicate := requestResolveIntervention(t, client, firstServer.URL, session.ID, session.CSRFToken, run.ID)
	if duplicate.StatusCode != http.StatusOK {
		duplicate.Body.Close()
		firstServer.Close()
		firstService.Shutdown()
		t.Fatalf("duplicate resolution status=%d, want %d", duplicate.StatusCode, http.StatusOK)
	}
	duplicate.Body.Close()
	events, err := st.ListMissionRunEvents(run.ID)
	if err != nil || len(events) != 2 {
		firstServer.Close()
		firstService.Shutdown()
		t.Fatalf("durable resolution outbox events=%d err=%v, want exactly 2", len(events), err)
	}
	for _, event := range events {
		if event.IdempotencyKey == "" || (event.EventType != "human_intervention.resolved" && event.EventType != "mission.resume_requested") {
			firstServer.Close()
			firstService.Shutdown()
			t.Fatalf("unexpected intervention outbox event: %+v", event)
		}
	}

	firstServer.Close()
	preRestart, err := firstService.Runner().GetRun(ctx, run.ID)
	if err != nil {
		firstService.Shutdown()
		t.Fatal(err)
	}
	if preRestart.State != runner.StateExecuting || len(preRestart.PackageRuns) != 1 || preRestart.PackageRuns[0].State != runner.StateAllocating || preRestart.ResumeRequest == nil || preRestart.ResumeRequest.Status != "STARTED" {
		firstService.Shutdown()
		t.Fatalf("restart checkpoint=%+v, want durable EXECUTING with package ALLOCATING and resume STARTED", preRestart)
	}
	if preRestart.NeedsHuman == nil || preRestart.NeedsHuman.Resolution == nil {
		firstService.Shutdown()
		t.Fatalf("human resolution was not durable at restart checkpoint: %+v", preRestart.NeedsHuman)
	}
	for _, event := range events {
		if event.IdempotencyKey != preRestart.NeedsHuman.Resolution.IdempotencyKey {
			firstService.Shutdown()
			t.Fatalf("outbox event %s key=%q differs from resolution key=%q", event.EventType, event.IdempotencyKey, preRestart.NeedsHuman.Resolution.IdempotencyKey)
		}
	}
	stopCtx, stopCancel := context.WithTimeout(ctx, 2*time.Second)
	if err := firstService.ShutdownContext(stopCtx); err != nil {
		stopCancel()
		t.Fatal(err)
	}
	stopCancel()

	reopened, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	secondExecutor := &interventionRestartExecutor{effects: &effects}
	secondService := nexus.NewWithStoreAndExecutor(reopened, secondExecutor)
	t.Cleanup(secondService.Shutdown)
	secondAuth, _, err := NewAuthManagerWithStore("127.0.0.1", "0", authDir)
	if err != nil {
		secondService.Shutdown()
		t.Fatal(err)
	}
	persistedAfterRestart, err := secondService.Runner().GetRun(ctx, run.ID)
	if err != nil {
		secondService.Shutdown()
		t.Fatal(err)
	}
	if persistedAfterRestart.State != runner.StateExecuting || len(persistedAfterRestart.PackageRuns) != 1 || persistedAfterRestart.PackageRuns[0].State != runner.StateAllocating || persistedAfterRestart.ResumeRequest == nil || persistedAfterRestart.ResumeRequest.Status != "STARTED" {
		secondService.Shutdown()
		t.Fatalf("reopened checkpoint=%+v, want EXECUTING with package ALLOCATING and resume STARTED", persistedAfterRestart)
	}
	secondServer := newInterventionRouteServer(secondAuth, secondService)
	t.Cleanup(secondServer.Close)
	if err := secondService.RecoverMissionRuns(ctx); err != nil {
		secondServer.Close()
		secondService.Shutdown()
		t.Fatalf("recover mission after restart: %v", err)
	}
	waitCtx, waitCancel := context.WithTimeout(ctx, 5*time.Second)
	final, err := secondService.WaitMissionWorker(waitCtx, run.ID)
	waitCancel()
	if err != nil {
		secondServer.Close()
		secondService.Shutdown()
		t.Fatalf("wait for recovered mission: %v", err)
	}
	if final.State != runner.StateCompletedVerified || final.ResumeRequest == nil || final.ResumeRequest.Status != "COMPLETED" {
		secondServer.Close()
		secondService.Shutdown()
		t.Fatalf("recovered mission did not reach verified terminal state: %+v", final)
	}
	if got := effects.Load(); got != 1 {
		secondServer.Close()
		secondService.Shutdown()
		t.Fatalf("external execution count after restart=%d, want exactly 1", got)
	}
	events, err = reopened.ListMissionRunEvents(run.ID)
	if err != nil || len(events) != 2 {
		secondServer.Close()
		secondService.Shutdown()
		t.Fatalf("outbox events after restart=%d err=%v, want exactly 2", len(events), err)
	}
	getRequest, err := http.NewRequest(http.MethodGet, secondServer.URL+"/api/v1/runs/"+run.ID, nil)
	if err != nil {
		secondServer.Close()
		secondService.Shutdown()
		t.Fatal(err)
	}
	getRequest.Header.Set("Authorization", "Bearer "+session.ID)
	getResponse, err := secondServer.Client().Do(getRequest)
	if err != nil {
		secondServer.Close()
		secondService.Shutdown()
		t.Fatalf("GET recovered mission from restarted server: %v", err)
	}
	var serverFinal runner.MissionRun
	decodeErr := json.NewDecoder(getResponse.Body).Decode(&serverFinal)
	getResponse.Body.Close()
	secondServer.Close()
	if decodeErr != nil || getResponse.StatusCode != http.StatusOK || serverFinal.State != runner.StateCompletedVerified {
		secondService.Shutdown()
		t.Fatalf("restarted server response status=%d state=%s decode_err=%v", getResponse.StatusCode, serverFinal.State, decodeErr)
	}
	if err := secondService.ShutdownContext(ctx); err != nil {
		t.Fatal(err)
	}
}

// csrfClient returns an authenticated client plus the CSRF token.
func csrfClient(t *testing.T) (*http.Client, *Server, string) {
	t.Helper()
	client, srv := newTestClient(t)
	resp, err := client.Get(srv.URL() + "/api/v1/session")
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	var sess struct {
		Authenticated bool   `json:"authenticated"`
		CSRFToken     string `json:"csrf_token"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&sess)
	resp.Body.Close()
	if !sess.Authenticated || sess.CSRFToken == "" {
		t.Fatal("session not authenticated / no CSRF token")
	}
	return client, srv, sess.CSRFToken
}

func TestUpdatePlanReturnsConflictForStaleRevision(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	n := nexus.Default()
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Plan revision conflict", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	created, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "Original"})
	if err != nil {
		t.Fatal(err)
	}

	stale := *created
	current := *created
	current.Title = "Saved current edit"
	if _, _, err := n.UpdateWorkPlan(context.Background(), current, "current edit"); err != nil {
		t.Fatalf("persist current edit: %v", err)
	}
	stale.Title = "Stale edit"
	body, err := json.Marshal(map[string]any{
		"plan":           stale,
		"change_summary": "stale edit",
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPut, srv.URL()+"/api/v1/plans/"+created.ID, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-CSRF-Token", csrf)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("PUT stale plan: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("stale plan response status=%d, want %d", response.StatusCode, http.StatusConflict)
	}

	getResponse, err := client.Get(srv.URL() + "/api/v1/plans/" + created.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer getResponse.Body.Close()
	var latest struct {
		Plan store.WorkPlan `json:"plan"`
	}
	if err := json.NewDecoder(getResponse.Body).Decode(&latest); err != nil {
		t.Fatal(err)
	}
	if latest.Plan.CurrentRevision != 2 || latest.Plan.Title != "Saved current edit" {
		t.Fatalf("stale HTTP update changed the plan: revision=%d title=%q", latest.Plan.CurrentRevision, latest.Plan.Title)
	}
}

func TestHostSessionImportRequiresCSRF(t *testing.T) {
	client, srv, _ := csrfClient(t)
	resp, err := client.Post(srv.URL()+"/api/v1/profiles/import-host-sessions", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("post host session import without CSRF token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for host session import without CSRF token, got %d", resp.StatusCode)
	}
}

func TestResolveRunInterventionRejectsMalformedAndNonPositiveVersion(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "malformed JSON", body: `{"intervention_id":`, want: "invalid body"},
		{name: "trailing JSON value", body: `{"intervention_id":"intv","version":1,"option_id":"verify"} {}`, want: "invalid body"},
		{name: "missing version", body: `{"intervention_id":"intv","option_id":"verify"}`, want: "intervention_id, version and option_id are required"},
		{name: "zero version", body: `{"intervention_id":"intv","version":0,"option_id":"verify"}`, want: "intervention_id, version and option_id are required"},
		{name: "negative version", body: `{"intervention_id":"intv","version":-1,"option_id":"verify"}`, want: "intervention_id, version and option_id are required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/runs/missing-run/resolve-intervention", strings.NewReader(test.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("X-CSRF-Token", csrf)
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("resolve intervention: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status=%d, want %d", resp.StatusCode, http.StatusBadRequest)
			}
			var payload struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload.Error != test.want {
				t.Fatalf("error=%q, want %q", payload.Error, test.want)
			}
		})
	}
}

func TestNexusProjectsAndAgentsAPI(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()

	// Create a project from a temp dir.
	projDir := t.TempDir()
	body, _ := json.Marshal(map[string]string{"name": "Omega API", "path": projDir})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects", bytes.NewReader(body))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	var proj store.Project
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&proj)
	resp.Body.Close()
	if proj.ID == "" || len(proj.ID) < 5 {
		t.Fatalf("invalid project id: %q", proj.ID)
	}
	if proj.MaestroMode != store.MaestroOff {
		t.Errorf("default maestro mode expected OFF, got %q", proj.MaestroMode)
	}

	// List projects.
	resp, err = client.Get(base + "/api/v1/projects")
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	var projects []store.Project
	_ = json.NewDecoder(resp.Body).Decode(&projects)
	resp.Body.Close()
	if len(projects) < 1 {
		t.Fatal("expected at least one project")
	}

	// Create an agent in the project.
	abody, _ := json.Marshal(map[string]string{"name": "Backend Developer"})
	req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/projects/"+proj.ID+"/agents", bytes.NewReader(abody))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	var agent store.Agent
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent expected 201, got %d", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&agent)
	resp.Body.Close()
	if agent.ID == "" || len(agent.ID) < 5 {
		t.Fatalf("invalid agent id: %q", agent.ID)
	}

	// List agents.
	resp, err = client.Get(base + "/api/v1/projects/" + proj.ID + "/agents")
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	var agents []store.Agent
	_ = json.NewDecoder(resp.Body).Decode(&agents)
	resp.Body.Close()
	if len(agents) != 1 || agents[0].ID != agent.ID {
		t.Fatalf("expected agent %s, got %+v", agent.ID, agents)
	}

	// Agent detail (generations/lineage empty) + effective state fields.
	resp, err = client.Get(base + "/api/v1/agents/" + agent.ID)
	if err != nil {
		t.Fatalf("agent detail: %v", err)
	}
	var detail map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&detail)
	resp.Body.Close()
	if detail["agent"] == nil || detail["generations"] == nil {
		t.Fatalf("agent detail missing fields: %+v", detail)
	}
	if _, hasState := detail["effective_state"]; !hasState {
		t.Errorf("agent detail must include effective_state")
	}
	if rec, _ := detail["recoverable"].(bool); rec {
		t.Errorf("stopped agent with no generation must not be recoverable")
	}

	// Recover on an agent with no runtime must fail cleanly (not 500/panic).
	recReq, _ := http.NewRequest(http.MethodPost, base+"/api/v1/agents/"+agent.ID+"/recover", nil)
	recReq.Header.Set("X-CSRF-Token", csrf)
	recResp, err := client.Do(recReq)
	if err != nil {
		t.Fatalf("recover agent (no runtime): %v", err)
	}
	_ = recResp.Body.Close()
	if recResp.StatusCode == http.StatusInternalServerError {
		t.Errorf("recover without runtime must not 500, got %d", recResp.StatusCode)
	}

	// Layout round-trip with monotonic revision and conflict handling.
	lbody, _ := json.Marshal(map[string]any{"layout": `{"openAgents":["` + agent.ID + `"]}`, "revision": 0})
	req, _ = http.NewRequest(http.MethodPut, base+"/api/v1/projects/"+proj.ID+"/layout", bytes.NewReader(lbody))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("save layout: %v", err)
	}
	var saveResp map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&saveResp)
	resp.Body.Close()
	if saveResp["status"] != "saved" || saveResp["revision"].(float64) < 1 {
		t.Fatalf("unexpected save layout response: %v", saveResp)
	}

	// Stale revision save returns 409 Conflict.
	staleBody, _ := json.Marshal(map[string]any{"layout": `{"openAgents":[]}`, "revision": 999})
	req, _ = http.NewRequest(http.MethodPut, base+"/api/v1/projects/"+proj.ID+"/layout", bytes.NewReader(staleBody))
	req.Header.Set("X-CSRF-Token", csrf)
	conflictResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("stale layout save error: %v", err)
	}
	if conflictResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on stale revision, got %d", conflictResp.StatusCode)
	}
	conflictResp.Body.Close()

	resp, err = client.Get(base + "/api/v1/projects/" + proj.ID + "/layout")
	if err != nil {
		t.Fatalf("get layout: %v", err)
	}
	var layoutResp map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&layoutResp)
	resp.Body.Close()
	layoutStr, _ := layoutResp["layout"].(string)
	if layoutStr == "" || !bytes.Contains([]byte(layoutStr), []byte(agent.ID)) {
		t.Fatalf("layout roundtrip failed: %v", layoutResp)
	}

	// Agent terminal WS without active runtime must fail cleanly (404).
	wsResp, err := client.Get(base + "/api/v1/agents/" + agent.ID + "/terminal")
	if err != nil {
		t.Fatalf("agent terminal (no runtime): %v", err)
	}
	_ = wsResp.Body.Close()
	if wsResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for terminal without runtime, got %d", wsResp.StatusCode)
	}

	// Delete agent.
	dreq, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/agents/"+agent.ID, nil)
	dreq.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(dreq)
	if err != nil {
		t.Fatalf("delete agent: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("delete agent expected 200, got %d", resp.StatusCode)
	}

	// DELETE is idempotent for stale UI/surface references.
	dreq, _ = http.NewRequest(http.MethodDelete, base+"/api/v1/agents/"+agent.ID, nil)
	dreq.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(dreq)
	if err != nil {
		t.Fatalf("repeat delete agent: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("repeat delete agent expected 200, got %d", resp.StatusCode)
	}

	// Delete project.
	pdreq, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/projects/"+proj.ID, nil)
	pdreq.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(pdreq)
	if err != nil {
		t.Fatalf("delete project: %v", err)
	}
	resp.Body.Close()
}

func TestSystemDoctorAPIUsesSharedReadOnlyReport(t *testing.T) {
	client, srv, _ := csrfClient(t)
	resp, err := client.Get(srv.URL() + "/api/v1/system/doctor")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("doctor status: %d", resp.StatusCode)
	}
	var report struct {
		Schema string `json:"schema"`
		Checks []struct {
			ID string `json:"id"`
		} `json:"checks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.Schema != "nexus.doctor/v1" || len(report.Checks) == 0 {
		t.Fatalf("unexpected doctor report: %+v", report)
	}
}

func TestSystemDoctorUsesHandlerDriverRegistry(t *testing.T) {
	registry := driver.NewRegistry()
	registry.Register(driver.NewFakeDriver())
	h := &NexusHandler{drivers: registry}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/system/doctor", nil)

	h.handleSystemDoctor(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("doctor status: %d", recorder.Code)
	}

	var report struct {
		Providers map[string]json.RawMessage `json:"providers"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) != 0 {
		t.Fatalf("providers = %#v, want no non-fake providers from injected registry", report.Providers)
	}
}

// TestNexusCSRFEnforcement verifies that all mutating Nexus REST routes reject
// requests missing or invalid CSRF tokens (P0-2).
func TestNexusCSRFEnforcement(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()

	// Create a project for sub-route testing.
	projDir := t.TempDir()
	body, _ := json.Marshal(map[string]string{"name": "CSRF Test", "path": projDir})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects", bytes.NewReader(body))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	var proj store.Project
	_ = json.NewDecoder(resp.Body).Decode(&proj)
	resp.Body.Close()

	// Create an agent for sub-route testing.
	abody, _ := json.Marshal(map[string]string{"name": "CSRF Agent"})
	req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/projects/"+proj.ID+"/agents", bytes.NewReader(abody))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	var agent store.Agent
	_ = json.NewDecoder(resp.Body).Decode(&agent)
	resp.Body.Close()

	// Helper: assert mutating request without CSRF returns 403.
	assertNoCSRF := func(method, path string, body []byte) {
		t.Helper()
		var req *http.Request
		if body != nil {
			req, _ = http.NewRequest(method, base+path, bytes.NewReader(body))
		} else {
			req, _ = http.NewRequest(method, base+path, nil)
		}
		// No X-CSRF-Token header.
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", method, path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s without CSRF: expected 403, got %d", method, path, resp.StatusCode)
		}
	}

	// Helper: assert mutating request with wrong CSRF returns 403.
	assertBadCSRF := func(method, path string, body []byte) {
		t.Helper()
		var req *http.Request
		if body != nil {
			req, _ = http.NewRequest(method, base+path, bytes.NewReader(body))
		} else {
			req, _ = http.NewRequest(method, base+path, nil)
		}
		req.Header.Set("X-CSRF-Token", "forged-token-value")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s (bad CSRF): %v", method, path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s with bad CSRF: expected 403, got %d", method, path, resp.StatusCode)
		}
	}

	// --- Nexus routes (routeProject / routeAgent) ---
	// Project PATCH without CSRF.
	assertNoCSRF(http.MethodPatch, "/api/v1/projects/"+proj.ID, []byte(`{"name":"x"}`))
	assertBadCSRF(http.MethodPatch, "/api/v1/projects/"+proj.ID, []byte(`{"name":"x"}`))

	// Project DELETE without CSRF.
	assertNoCSRF(http.MethodDelete, "/api/v1/projects/"+proj.ID, nil)
	assertBadCSRF(http.MethodDelete, "/api/v1/projects/"+proj.ID, nil)

	// Project layout PUT without CSRF.
	assertNoCSRF(http.MethodPut, "/api/v1/projects/"+proj.ID+"/layout", []byte(`{"layout":"{}"}`))
	assertBadCSRF(http.MethodPut, "/api/v1/projects/"+proj.ID+"/layout", []byte(`{"layout":"{}"}`))

	// Agent create POST without CSRF.
	assertNoCSRF(http.MethodPost, "/api/v1/projects/"+proj.ID+"/agents", []byte(`{"name":"x"}`))
	assertBadCSRF(http.MethodPost, "/api/v1/projects/"+proj.ID+"/agents", []byte(`{"name":"x"}`))

	// Agent PATCH without CSRF.
	assertNoCSRF(http.MethodPatch, "/api/v1/agents/"+agent.ID, []byte(`{"name":"x"}`))
	assertBadCSRF(http.MethodPatch, "/api/v1/agents/"+agent.ID, []byte(`{"name":"x"}`))

	// Agent DELETE without CSRF.
	assertNoCSRF(http.MethodDelete, "/api/v1/agents/"+agent.ID, nil)
	assertBadCSRF(http.MethodDelete, "/api/v1/agents/"+agent.ID, nil)

	// Agent start/stop/recover POST without CSRF.
	assertNoCSRF(http.MethodPost, "/api/v1/agents/"+agent.ID+"/start", []byte(`{"provider":"fake"}`))
	assertBadCSRF(http.MethodPost, "/api/v1/agents/"+agent.ID+"/start", []byte(`{"provider":"fake"}`))

	assertNoCSRF(http.MethodPost, "/api/v1/agents/"+agent.ID+"/stop", nil)
	assertBadCSRF(http.MethodPost, "/api/v1/agents/"+agent.ID+"/stop", nil)

	assertNoCSRF(http.MethodPost, "/api/v1/agents/"+agent.ID+"/recover", nil)
	assertBadCSRF(http.MethodPost, "/api/v1/agents/"+agent.ID+"/recover", nil)

	// --- authMiddleware routes (projects list) ---
	// Project list POST without CSRF.
	projectBody, _ := json.Marshal(map[string]string{"name": "x", "path": projDir})
	assertNoCSRF(http.MethodPost, "/api/v1/projects", projectBody)
	assertBadCSRF(http.MethodPost, "/api/v1/projects", projectBody)

	// --- GET requests must NOT require CSRF ---
	getReq, _ := http.NewRequest(http.MethodGet, base+"/api/v1/projects/"+proj.ID, nil)
	getResp, err := client.Do(getReq)
	if err != nil {
		t.Fatalf("GET project: %v", err)
	}
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Errorf("GET project without CSRF should be 200, got %d", getResp.StatusCode)
	}

	getAgentReq, _ := http.NewRequest(http.MethodGet, base+"/api/v1/agents/"+agent.ID, nil)
	getAgentResp, err := client.Do(getAgentReq)
	if err != nil {
		t.Fatalf("GET agent: %v", err)
	}
	getAgentResp.Body.Close()
	if getAgentResp.StatusCode != http.StatusOK {
		t.Errorf("GET agent without CSRF should be 200, got %d", getAgentResp.StatusCode)
	}

	// Cleanup: delete agent then project (with CSRF).
	dreq, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/agents/"+agent.ID, nil)
	dreq.Header.Set("X-CSRF-Token", csrf)
	dresp, _ := client.Do(dreq)
	dresp.Body.Close()

	pdreq, _ := http.NewRequest(http.MethodDelete, base+"/api/v1/projects/"+proj.ID, nil)
	pdreq.Header.Set("X-CSRF-Token", csrf)
	pdresp, _ := client.Do(pdreq)
	pdresp.Body.Close()
}

func TestAgentConfigApplyRejectsUnallocatedProvider(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	projectDir := t.TempDir()
	projectBody, _ := json.Marshal(map[string]string{"name": "Config validation", "path": projectDir})
	projectReq, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/projects", bytes.NewReader(projectBody))
	projectReq.Header.Set("X-CSRF-Token", csrf)
	projectResp, err := client.Do(projectReq)
	if err != nil {
		t.Fatal(err)
	}
	defer projectResp.Body.Close()
	if projectResp.StatusCode != http.StatusCreated {
		t.Fatalf("create project: got %d", projectResp.StatusCode)
	}
	var project store.Project
	if err := json.NewDecoder(projectResp.Body).Decode(&project); err != nil {
		t.Fatal(err)
	}

	agentReq, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/projects/"+project.ID+"/agents", bytes.NewBufferString(`{"name":"Agent"}`))
	agentReq.Header.Set("X-CSRF-Token", csrf)
	agentResp, err := client.Do(agentReq)
	if err != nil {
		t.Fatal(err)
	}
	defer agentResp.Body.Close()
	if agentResp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent: got %d", agentResp.StatusCode)
	}
	var agent store.Agent
	if err := json.NewDecoder(agentResp.Body).Decode(&agent); err != nil {
		t.Fatal(err)
	}

	applyReq, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/agents/"+agent.ID+"/config/apply", bytes.NewBufferString(`{"provider":"not-a-provider","profile":"default"}`))
	applyReq.Header.Set("X-CSRF-Token", csrf)
	applyResp, err := client.Do(applyReq)
	if err != nil {
		t.Fatal(err)
	}
	defer applyResp.Body.Close()
	if applyResp.StatusCode != http.StatusConflict {
		t.Fatalf("unallocated provider must be rejected, got %d", applyResp.StatusCode)
	}
}

func TestResourceAllocationRejectsAutomaticPolicy(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	request, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/resources/select", bytes.NewBufferString(`{"agent_id":"agt_example","provider":"codex","profile":"default","policy":"BALANCED"}`))
	request.Header.Set("X-CSRF-Token", csrf)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("manual allocation endpoint must reject automatic policy, got %d", response.StatusCode)
	}
}

func TestResourcesRefreshReturnsAccounts(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/resources/refresh", nil)
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from resources refresh, got %d", resp.StatusCode)
	}
	var payload struct {
		Accounts  []any  `json:"accounts"`
		Policy    string `json:"policy"`
		Refreshed bool   `json:"refreshed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode refresh payload: %v", err)
	}
	if !payload.Refreshed {
		t.Fatal("expected refreshed=true")
	}
	if payload.Accounts == nil {
		t.Fatal("expected accounts array (possibly empty), got null")
	}
	if payload.Policy == "" {
		t.Fatal("expected non-empty policy")
	}
}

func TestSystemUpdateReturnsJSONNot501(t *testing.T) {
	prev := performSystemUpdate
	performSystemUpdate = func() nexus.UpdateResult {
		return nexus.UpdateResult{
			NexusUpdated:   false,
			NexusVersion:   "0.5.0-test",
			MaestroUpdated: true,
			MaestroVersion: "0.1.11",
			Error:          "Nexus binary update was not performed",
		}
	}
	t.Cleanup(func() { performSystemUpdate = prev })

	client, srv, csrf := csrfClient(t)
	req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/maestro/update", bytes.NewBufferString(`{"product":"maestro","target_version":"latest","confirmed":true}`))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var result nexus.UpdateResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.MaestroVersion != "0.1.11" {
		t.Fatalf("maestro version: %+v", result)
	}
	if result.NexusUpdated {
		t.Fatal("must not claim Nexus binary updated")
	}
	if result.Error == "" {
		t.Fatal("expected honest nexus binary note")
	}
}

func TestMaestroUpdateRequiresExplicitContract(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	for _, body := range []string{`{}`, `{"product":"nexus","target_version":"latest","confirmed":true}`, `{"product":"maestro","target_version":"latest","confirmed":false}`} {
		req, _ := http.NewRequest(http.MethodPost, srv.URL()+"/api/v1/maestro/update", bytes.NewBufferString(body))
		req.Header.Set("X-CSRF-Token", csrf)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("invalid Maestro update contract %s returned %d", body, resp.StatusCode)
		}
	}
}

func TestSystemUpdatesUsesSharedNexusUpdateService(t *testing.T) {
	previous := checkNexusUpdate
	checkNexusUpdate = func(ctx context.Context) (*update.CheckResult, error) {
		return &update.CheckResult{
			CurrentVersion:     "1.0.0",
			LatestVersion:      "1.1.0",
			UpdateAvailable:    true,
			InstallationMethod: update.MethodStandalone,
			AllowsSelfUpdate:   true,
		}, nil
	}
	t.Cleanup(func() { checkNexusUpdate = previous })

	client, srv, _ := csrfClient(t)
	resp, err := client.Get(srv.URL() + "/api/v1/system/updates")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var payload struct {
		LatestVersion   string `json:"nexus_latest_version"`
		UpdateAvailable bool   `json:"nexus_update_available"`
		UpdateError     string `json:"nexus_update_error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.LatestVersion != "1.1.0" || !payload.UpdateAvailable || payload.UpdateError != "" {
		t.Fatalf("shared Nexus update result was not exposed: %+v", payload)
	}
}

func TestOfficialNexusInstallerLinkAllowlist(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://github.com/IAPro-Community/nexus/releases/download/channel-beta/nexus-setup-Windows_x86_64.exe", true},
		{"https://github.com/IAPro-Community/nexus/releases/download/v1.0.0/nexus-installer_Darwin_arm64.pkg", true},
		{"https://github.com/attacker/nexus/releases/download/v1.0.0/payload.exe", false},
		{"https://github.com.evil.example/IAPro-Community/nexus/releases/download/v1/payload.exe", false},
		{"http://github.com/IAPro-Community/nexus/releases/download/v1/payload.exe", false},
		{"https://example.com/IAPro-Community/nexus/releases/download/v1/payload.exe", false},
		{"https://github.com@evil.example/IAPro-Community/nexus/releases/download/v1/payload.exe", false},
		{"https://github.com:444/IAPro-Community/nexus/releases/download/v1/payload.exe", false},
	}
	for _, tc := range cases {
		if got := isOfficialNexusInstallerURL(tc.url); got != tc.want {
			t.Errorf("isOfficialNexusInstallerURL(%q)=%t, want %t", tc.url, got, tc.want)
		}
	}
}

func TestSystemUpdatesExposesSignedInstallerOnlyFromOfficialRelease(t *testing.T) {
	prev := checkNexusUpdate
	url := "https://github.com/IAPro-Community/nexus/releases/download/channel-beta/nexus-setup-Windows_x86_64.exe"
	checkNexusUpdate = func(_ context.Context) (*update.CheckResult, error) {
		return &update.CheckResult{
			CurrentVersion:     "1.0.0-beta.10",
			LatestVersion:      "1.0.0-beta.11",
			Channel:            "beta",
			UpdateAvailable:    true,
			InstallationMethod: update.MethodDesktop,
			InstallerArtifact:  &update.Artifact{URL: url, Target: update.TargetNSIS},
		}, nil
	}
	t.Cleanup(func() { checkNexusUpdate = prev })

	client, srv, _ := csrfClient(t)
	resp, err := client.Get(srv.URL() + "/api/v1/system/updates")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var payload struct {
		InstallerURL string `json:"nexus_installer_url"`
		ReleaseURL   string `json:"nexus_release_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.InstallerURL != url {
		t.Fatalf("official installer missing: %+v", payload)
	}
	if payload.ReleaseURL != "https://github.com/IAPro-Community/nexus/releases/tag/channel-beta" {
		t.Fatalf("invalid official release page: %+v", payload)
	}
}

func TestBetaInstallationFindsNewerSignedStableRelease(t *testing.T) {
	previousBeta := checkNexusUpdate
	previousStable := checkNexusStableUpdate
	checkNexusUpdate = func(_ context.Context) (*update.CheckResult, error) {
		return &update.CheckResult{
			CurrentVersion:  "1.0.0-beta.10",
			LatestVersion:   "1.0.0-beta.10",
			Channel:         "beta",
			UpdateAvailable: false,
		}, nil
	}
	checkNexusStableUpdate = func(_ context.Context) (*update.CheckResult, error) {
		return &update.CheckResult{
			CurrentVersion:     "1.0.0-beta.10",
			LatestVersion:      "1.0.0",
			Channel:            "stable",
			UpdateAvailable:    true,
			InstallationMethod: update.MethodDesktop,
			InstallerArtifact: &update.Artifact{
				URL:    "https://github.com/IAPro-Community/nexus/releases/download/v1.0.0/nexus-installer_Darwin_arm64.pkg",
				Target: update.TargetPKG,
			},
		}, nil
	}
	t.Cleanup(func() {
		checkNexusUpdate = previousBeta
		checkNexusStableUpdate = previousStable
	})

	client, srv, _ := csrfClient(t)
	resp, err := client.Get(srv.URL() + "/api/v1/system/updates")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var payload struct {
		Channel      string `json:"channel"`
		Version      string `json:"nexus_latest_version"`
		Available    bool   `json:"nexus_update_available"`
		InstallerURL string `json:"nexus_installer_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Channel != "stable" || payload.Version != "1.0.0" || !payload.Available {
		t.Fatalf("beta-to-stable upgrade not discovered: %+v", payload)
	}
	if payload.InstallerURL == "" {
		t.Fatal("expected native installer in verified stable release")
	}
}

func TestBetaKeepsNewerBetaWithoutConsultingStable(t *testing.T) {
	previousBeta := checkNexusUpdate
	previousStable := checkNexusStableUpdate
	checkNexusUpdate = func(_ context.Context) (*update.CheckResult, error) {
		return &update.CheckResult{
			CurrentVersion:  "1.0.0-beta.10",
			LatestVersion:   "1.0.0-beta.11",
			Channel:         "beta",
			UpdateAvailable: true,
		}, nil
	}
	checkNexusStableUpdate = func(_ context.Context) (*update.CheckResult, error) {
		t.Fatal("stable lookup should not run while beta has a newer release")
		return nil, nil
	}
	t.Cleanup(func() {
		checkNexusUpdate = previousBeta
		checkNexusStableUpdate = previousStable
	})

	client, srv, _ := csrfClient(t)
	resp, err := client.Get(srv.URL() + "/api/v1/system/updates")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var payload struct {
		Channel string `json:"channel"`
		Version string `json:"nexus_latest_version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Channel != "beta" || payload.Version != "1.0.0-beta.11" {
		t.Fatalf("beta update unexpectedly replaced by stable: %+v", payload)
	}
}

func TestProjectEventsAPI(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()

	// 1. Create a project
	dir := t.TempDir()
	createBody, _ := json.Marshal(map[string]string{
		"name": "Events Project",
		"path": dir,
	})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects", bytes.NewReader(createBody))
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	defer resp.Body.Close()
	var proj store.Project
	if err := json.NewDecoder(resp.Body).Decode(&proj); err != nil {
		t.Fatalf("decode project: %v", err)
	}

	// 2. Insert test events into the store
	st := nexus.Default().Store()
	if st == nil {
		t.Fatal("store is nil")
	}
	_, err = st.RecordEventMetadata(store.EventMetadata{
		ProjectID: proj.ID,
		Kind:      "AGENT_WORKING",
		Summary:   "Secret key 12345 agent work",
	})
	if err != nil {
		t.Fatalf("record event: %v", err)
	}

	// 3. Fetch project events via API
	getReq, _ := http.NewRequest(http.MethodGet, base+"/api/v1/projects/"+proj.ID+"/events", nil)
	getResp, err := client.Do(getReq)
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", getResp.StatusCode)
	}

	var eventsList []store.EventMetadata
	if err := json.NewDecoder(getResp.Body).Decode(&eventsList); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if len(eventsList) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventsList))
	}
	if eventsList[0].Kind != "AGENT_WORKING" {
		t.Fatalf("expected AGENT_WORKING, got %s", eventsList[0].Kind)
	}
}
