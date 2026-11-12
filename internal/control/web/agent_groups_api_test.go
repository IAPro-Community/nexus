package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestAgentGroupsAPIUsesDurableProjectScopedMembership(t *testing.T) {
	client, srv, csrf := csrfClient(t)
	base := srv.URL()
	create := func(name string) store.Project {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"name": name, "path": t.TempDir()})
		request, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects", bytes.NewReader(body))
		request.Header.Set("X-CSRF-Token", csrf)
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var project store.Project
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("create project status=%d", response.StatusCode)
		}
		if err := json.NewDecoder(response.Body).Decode(&project); err != nil {
			t.Fatal(err)
		}
		return project
	}
	project := create("Groups")
	foreignProject := create("Foreign")
	createAgent := func(projectID, name string) store.Agent {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"name": name})
		request, _ := http.NewRequest(http.MethodPost, base+"/api/v1/projects/"+projectID+"/agents", bytes.NewReader(body))
		request.Header.Set("X-CSRF-Token", csrf)
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var agent store.Agent
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("create agent status=%d", response.StatusCode)
		}
		if err := json.NewDecoder(response.Body).Decode(&agent); err != nil {
			t.Fatal(err)
		}
		return agent
	}
	agent := createAgent(project.ID, "Stable Agent")
	secondAgent := createAgent(project.ID, "Second Agent")
	foreignAgent := createAgent(foreignProject.ID, "Foreign Agent")

	groupsURL := base + "/api/v1/projects/" + project.ID + "/agent-groups"
	groupBody := []byte(`{"name":"Release lane","task":"Ship v1","worktree":"../release","feature":"agent-groups"}`)
	request, _ := http.NewRequest(http.MethodPost, groupsURL, bytes.NewReader(groupBody))
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("group mutation without CSRF status=%d, want 403", response.StatusCode)
	}
	request, _ = http.NewRequest(http.MethodPost, groupsURL, bytes.NewReader(groupBody))
	request.Header.Set("X-CSRF-Token", csrf)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var group store.AgentGroup
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create group status=%d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&group); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	updateBody, _ := json.Marshal(map[string]any{
		"name": group.Name, "task": group.Task, "worktree": group.Worktree, "feature": group.Feature,
		"agent_ids": []string{secondAgent.ID, agent.ID},
	})
	request, _ = http.NewRequest(http.MethodPut, groupsURL+"/"+group.ID, bytes.NewReader(updateBody))
	request.Header.Set("X-CSRF-Token", csrf)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		t.Fatalf("update group status=%d", response.StatusCode)
	}
	var updated store.AgentGroup
	if err := json.NewDecoder(response.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if len(updated.AgentIDs) != 2 || updated.AgentIDs[0] != secondAgent.ID || updated.AgentIDs[1] != agent.ID {
		t.Fatalf("updated group membership=%v", updated.AgentIDs)
	}

	request, _ = http.NewRequest(http.MethodGet, groupsURL, nil)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var groups []store.AgentGroup
	if response.StatusCode != http.StatusOK {
		t.Fatalf("list groups status=%d", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&groups); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if len(groups) != 1 || len(groups[0].AgentIDs) != 2 || groups[0].AgentIDs[0] != secondAgent.ID || groups[0].AgentIDs[1] != agent.ID {
		t.Fatalf("listed groups=%+v", groups)
	}

	updateBody, _ = json.Marshal(map[string]any{"name": group.Name, "agent_ids": []string{foreignAgent.ID}})
	request, _ = http.NewRequest(http.MethodPut, groupsURL+"/"+group.ID, bytes.NewReader(updateBody))
	request.Header.Set("X-CSRF-Token", csrf)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("foreign Agent membership status=%d, want 400", response.StatusCode)
	}
	request, _ = http.NewRequest(http.MethodDelete, groupsURL+"/"+group.ID, nil)
	request.Header.Set("X-CSRF-Token", csrf)
	response, err = client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("delete group status=%d, want 200", response.StatusCode)
	}
	agentResponse, err := client.Get(base + "/api/v1/agents/" + agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer agentResponse.Body.Close()
	var body struct {
		Agent store.Agent `json:"agent"`
	}
	if agentResponse.StatusCode != http.StatusOK {
		t.Fatalf("Agent lookup after group deletion status=%d", agentResponse.StatusCode)
	}
	if err := json.NewDecoder(agentResponse.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Agent.ID != agent.ID {
		t.Fatalf("Agent identity after group deletion=%q, want %q", body.Agent.ID, agent.ID)
	}
}
