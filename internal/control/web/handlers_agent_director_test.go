package web

import (
    "net/http"
    "bytes"
    "encoding/json"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestDirectorProjectionIsStrictlyReadOnlyAndCapabilitiesAreHonest(t *testing.T) {
    agent := store.Agent{ID: "agent-one", ProjectID: "project-one", Status: "WORKING"}
    view := projectDirectorAgent(agent, nil, false)
    if view.ProcessState != "UNKNOWN" || view.ProcessEvidence != "NO_LIVE_RUNTIME_OBSERVED" {
        t.Fatalf("missing generation must never imply liveness: %+v", view)
    }
    if view.RuntimeID != "" || view.ExecutionOwner != "NEXUS" || len(view.Actions) != 6 {
        t.Fatalf("unexpected baseline: %+v", view)
    }
    seen := map[string]bool{}
    for _, action := range view.Actions {
        if seen[action.Kind] { t.Fatalf("duplicate directive: %s", action.Kind) }
        seen[action.Kind] = true
        switch action.Kind {
        case "INSPECT":
            if action.Availability != "SUPPORTED" { t.Fatal("inspect must remain available") }
        case "DIRECTIVE":
            if action.Availability != "PREPARE_ONLY" { t.Fatal("directive must not claim steering") }
        case "WAIT":
            if action.Availability != "OBSERVE_ONLY" { t.Fatal("wait cannot promise a job wait") }
        case "QUEUE_NEXT", "STEER", "CANCEL":
            if action.Availability == "SUPPORTED" { t.Fatalf("%s has no verified control API", action.Kind) }
        default:
            t.Fatalf("unknown directive: %s", action.Kind)
        }
    }
    gen := store.RuntimeGeneration{ID: "generation-two", RuntimeID: "runtime-two", Provider: "opencode", Profile: "work"}
    live := projectDirectorAgent(agent, &gen, true)
    if live.ProcessState != "LIVE" || live.GenerationID != gen.ID || live.RuntimeID != gen.RuntimeID {
        t.Fatalf("live generation projection mismatch: %+v", live)
    }
    stale := projectDirectorAgent(agent, &gen, false)
    if stale.ProcessState != "NOT_LIVE" || stale.ProcessEvidence != "GENERATION_NOT_IN_LIVE_REGISTRY" {
        t.Fatalf("stale generation cannot be shown as live: %+v", stale)
    }
    if stale.AgentStatus != "WORKING" {
        t.Fatal("persisted status must be displayed as recorded, not silently overwritten")
    }
}

func TestDirectorHandlerRejectsMutationAndUnscopedLookup(t *testing.T) {
    h := &NexusHandler{}
    write := func(method, path string) *httptest.ResponseRecorder {
        rr := httptest.NewRecorder()
        h.handleAgentDirector(rr, httptest.NewRequest(method, path, strings.NewReader("{}")))
        return rr
    }
    if r := write(http.MethodPost, "/api/v1/agents/agent-one/director?project_id=project-one"); r.Code != http.StatusMethodNotAllowed {
        t.Fatalf("mutating endpoint returned %d", r.Code)
    }
    if r := write(http.MethodGet, "/api/v1/agents/agent-one/director"); r.Code != http.StatusBadRequest {
        t.Fatalf("unscoped read returned %d", r.Code)
    }
}

func TestDirectorEndpointKeepsProjectScopeAndReadOnlyAuthentication(t *testing.T) {
    client, srv, csrf := csrfClient(t)
    base := srv.URL()
    projectBody, _ := json.Marshal(map[string]string{"name": "Director test project", "path": t.TempDir()})
    req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects", bytes.NewReader(projectBody))
    req.Header.Set("X-CSRF-Token", csrf)
    response, err := client.Do(req)
    if err != nil { t.Fatal(err) }
    if response.StatusCode != http.StatusCreated {
        t.Fatalf("create project status=%d", response.StatusCode)
    }
    var project store.Project
    if err := json.NewDecoder(response.Body).Decode(&project); err != nil { t.Fatal(err) }
    response.Body.Close()

    agentBody, _ := json.Marshal(map[string]string{"name": "Director agent", "role": "developer"})
    req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/projects/"+project.ID+"/agents", bytes.NewReader(agentBody))
    req.Header.Set("X-CSRF-Token", csrf)
    response, err = client.Do(req)
    if err != nil { t.Fatal(err) }
    if response.StatusCode != http.StatusCreated {
        t.Fatalf("create agent status=%d", response.StatusCode)
    }
    var agent store.Agent
    if err := json.NewDecoder(response.Body).Decode(&agent); err != nil { t.Fatal(err) }
    response.Body.Close()

    url := base+"/api/v1/agents/"+agent.ID+"/director?project_id="+project.ID
    response, err = client.Get(url)
    if err != nil { t.Fatal(err) }
    if response.StatusCode != http.StatusOK {
        t.Fatalf("director read status=%d", response.StatusCode)
    }
    var view directorProjection
    if err := json.NewDecoder(response.Body).Decode(&view); err != nil { t.Fatal(err) }
    response.Body.Close()
    if view.AgentID != agent.ID || view.ProjectID != project.ID || view.ProcessState != "UNKNOWN" {
        t.Fatalf("unexpected director projection: %+v", view)
    }
    if response.Header.Get("Cache-Control") != "no-store" {
        t.Fatal("live runtime projection must not be cached")
    }

    response, err = client.Get(base+"/api/v1/agents/"+agent.ID+"/director?project_id=not-this-project")
    if err != nil { t.Fatal(err) }
    response.Body.Close()
    if response.StatusCode != http.StatusNotFound {
        t.Fatalf("cross-project read must fail, status=%d", response.StatusCode)
    }

    req, _ = http.NewRequest(http.MethodPost, url, strings.NewReader("{}"))
    response, err = client.Do(req)
    if err != nil { t.Fatal(err) }
    response.Body.Close()
    if response.StatusCode != http.StatusForbidden {
        t.Fatalf("mutation without CSRF must fail before routing, status=%d", response.StatusCode)
    }

    req, _ = http.NewRequest(http.MethodPost, url, strings.NewReader("{}"))
    req.Header.Set("X-CSRF-Token", csrf)
    response, err = client.Do(req)
    if err != nil { t.Fatal(err) }
    response.Body.Close()
    if response.StatusCode != http.StatusMethodNotAllowed {
        t.Fatalf("validly authenticated Director mutations must be unsupported, status=%d", response.StatusCode)
    }
}
