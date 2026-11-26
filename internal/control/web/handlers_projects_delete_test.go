package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func openTestWebProjectHarness(t *testing.T) (*nexus.Nexus, *NexusHandler, store.Project) {
	t.Helper()
	t.Setenv("NEXUS_CONFIG_DIR", t.TempDir())
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NEXUS_REAL_HOME", "")

	registry.ResetDefaultRegistryForTest()
	t.Cleanup(registry.ResetDefaultRegistryForTest)

	st, err := nexus.OpenStore()
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	n := nexus.NewWithStoreAndExecutor(st, nil)
	proj, err := st.CreateProject(store.Project{Name: "Web Delete Proj", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	handler := &NexusHandler{
		nexus:    n,
		projects: nexus.NewProjectApplicationService(n),
	}
	return n, handler, proj
}

func TestHandleProjectDeleteConflictAndResolutionModes(t *testing.T) {
	n, handler, proj := openTestWebProjectHarness(t)
	st, _ := n.OpenProject()
	agent, err := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Session AGY"})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	if err := registry.DefaultRegistry().Register(registry.RuntimeSession{
		RuntimeID:  "rt-del-1",
		ProviderID: "agy",
		ProfileID:  "default",
		State:      registry.StateRunning,
		PID:        os.Getpid(),
		StartedAt:  time.Now(),
		Transport:  "mock",
	}); err != nil {
		t.Fatalf("register runtime session: %v", err)
	}
	if _, err := st.AddGeneration(store.RuntimeGeneration{
		AgentID:    agent.ID,
		RevisionID: "rev-1",
		RuntimeID:  "rt-del-1",
		Provider:   "agy",
		Profile:    "default",
		StartedAt:  time.Now(),
	}); err != nil {
		t.Fatalf("add generation: %v", err)
	}

	// Plain DELETE returns 409 with code=project_has_live_agents and live_agents list.
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+proj.ID, nil)
	rec := httptest.NewRecorder()
	handler.handleProjectDetail(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d body=%s", rec.Code, rec.Body.String())
	}
	var conflictPayload struct {
		Code       string                       `json:"code"`
		ProjectID  string                       `json:"project_id"`
		LiveAgents []nexus.ProjectLiveAgentInfo `json:"live_agents"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &conflictPayload); err != nil {
		t.Fatalf("decode 409 payload: %v", err)
	}
	if conflictPayload.Code != "project_has_live_agents" || len(conflictPayload.LiveAgents) != 1 || conflictPayload.LiveAgents[0].Provider != "agy" || conflictPayload.LiveAgents[0].Profile != "default" {
		t.Fatalf("unexpected conflict payload: %+v", conflictPayload)
	}

	// Invalid mode returns 400.
	badReq := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+proj.ID+"?mode=unknown", nil)
	badRec := httptest.NewRecorder()
	handler.handleProjectDetail(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid mode, got %d", badRec.Code)
	}

	// Detach mode succeeds with 200 and reports detached_agents.
	detReq := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/"+proj.ID+"?mode=detach", nil)
	detRec := httptest.NewRecorder()
	handler.handleProjectDetail(detRec, detReq)
	if detRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on detach delete, got %d body=%s", detRec.Code, detRec.Body.String())
	}
	var detRes nexus.ProjectDeleteResult
	if err := json.Unmarshal(detRec.Body.Bytes(), &detRes); err != nil {
		t.Fatalf("decode detach response: %v", err)
	}
	if detRes.Mode != "detach" || len(detRes.DetachedAgents) != 1 {
		t.Fatalf("unexpected detach result: %+v", detRes)
	}
}
