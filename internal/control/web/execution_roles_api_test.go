package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestExecutionRoleAPIUsesRuntimeBindingWithoutMutatingAgentRevision(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()

	projectBody, _ := json.Marshal(map[string]string{"name": "Execution Role API", "path": t.TempDir()})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects", bytes.NewReader(projectBody))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create project status=%d", resp.StatusCode)
	}
	var project store.Project
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	agentBody, _ := json.Marshal(map[string]string{"name": "Persistent Dev", "role": "senior-developer"})
	req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/projects/"+project.ID+"/agents", bytes.NewReader(agentBody))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent status=%d", resp.StatusCode)
	}
	var agent store.Agent
	if err := json.NewDecoder(resp.Body).Decode(&agent); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	st, err := nexus.Default().OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	revision, err := st.AddRevision(agent.ID, `{"provider":"codex","profile":"default"}`)
	if err != nil {
		t.Fatal(err)
	}
	agent.CurrentRevisionID = revision.ID
	if err := st.UpdateAgent(agent); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddGeneration(store.RuntimeGeneration{
		AgentID: agent.ID, RevisionID: revision.ID, RuntimeID: "rt_execution_role_api",
		Provider: "codex", Profile: "default", StartedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	resp, err = client.Get(base + "/api/v1/execution-roles")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("execution roles status=%d", resp.StatusCode)
	}
	var roles []nexus.ExecutionRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(roles) != 6 {
		t.Fatalf("expected 6 execution roles, got %d", len(roles))
	}

	resp, err = client.Get(base + "/api/v1/runtimes/rt_execution_role_api/execution-role")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("default role status=%d", resp.StatusCode)
	}
	var initial nexus.RuntimeExecutionRoleView
	if err := json.NewDecoder(resp.Body).Decode(&initial); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if initial.Persisted || initial.Role.ID != "build" {
		t.Fatalf("unexpected default role: %#v", initial)
	}

	before, _ := st.ListRevisions(agent.ID)
	roleBody, _ := json.Marshal(map[string]string{"role_id": "review"})
	req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/runtimes/rt_execution_role_api/execution-role", bytes.NewReader(roleBody))
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set role status=%d", resp.StatusCode)
	}
	var changed nexus.RuntimeExecutionRoleView
	if err := json.NewDecoder(resp.Body).Decode(&changed); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if !changed.Persisted || changed.Role.ID != "review" || changed.Binding.Enforcement != nexus.EnforcementBestEffort {
		t.Fatalf("unexpected changed role: %#v", changed)
	}

	after, _ := st.ListRevisions(agent.ID)
	if len(after) != len(before) || after[0].ID != before[0].ID {
		t.Fatalf("execution-role API mutated AgentRevision: before=%#v after=%#v", before, after)
	}

	req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/runtimes/rt_execution_role_api/execution-role", bytes.NewReader(roleBody))
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("execution-role POST without CSRF must be 403, got %d", resp.StatusCode)
	}
}
