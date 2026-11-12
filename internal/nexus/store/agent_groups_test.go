package store

import (
	"strings"
	"testing"
)

func TestAgentGroupsPersistOrderedMembershipAndMoveWithoutChangingAgents(t *testing.T) {
	s := openTestStore(t)
	project, err := s.CreateProject(Project{Name: "Groups", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.CreateAgent(Agent{ProjectID: project.ID, Name: "First"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateAgent(Agent{ProjectID: project.ID, Name: "Second"})
	if err != nil {
		t.Fatal(err)
	}
	group, err := s.CreateAgentGroup(project.ID, "Release", "Ship v1", "/worktrees/release", "agents")
	if err != nil {
		t.Fatal(err)
	}
	group.AgentIDs = []string{second.ID, first.ID}
	group, err = s.UpdateAgentGroup(project.ID, group)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(group.AgentIDs, ",") != second.ID+","+first.ID {
		t.Fatalf("membership order = %v", group.AgentIDs)
	}

	other, err := s.CreateAgentGroup(project.ID, "Feature", "Implement grouping", "", "agent-groups")
	if err != nil {
		t.Fatal(err)
	}
	other.AgentIDs = []string{first.ID}
	if _, err := s.UpdateAgentGroup(project.ID, other); err != nil {
		t.Fatal(err)
	}

	groups, err := s.ListAgentGroups(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || len(groups[0].AgentIDs) != 1 || groups[0].AgentIDs[0] != second.ID ||
		len(groups[1].AgentIDs) != 1 || groups[1].AgentIDs[0] != first.ID {
		t.Fatalf("persisted memberships = %+v", groups)
	}
	gotFirst, err := s.GetAgent(first.ID, project.ID)
	if err != nil || gotFirst.ID != first.ID {
		t.Fatalf("agent identity was not preserved: agent=%+v err=%v", gotFirst, err)
	}
	if _, err := s.UpdateAgentGroup("other-project", other); err == nil {
		t.Fatal("cross-project group update unexpectedly succeeded")
	}
}

func TestAgentGroupUpdateRejectsDuplicateAndForeignAgentAtomically(t *testing.T) {
	s := openTestStore(t)
	project, err := s.CreateProject(Project{Name: "Groups", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	foreignProject, err := s.CreateProject(Project{Name: "Foreign", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := s.CreateAgent(Agent{ProjectID: project.ID, Name: "Local"})
	if err != nil {
		t.Fatal(err)
	}
	foreignAgent, err := s.CreateAgent(Agent{ProjectID: foreignProject.ID, Name: "Foreign"})
	if err != nil {
		t.Fatal(err)
	}
	group, err := s.CreateAgentGroup(project.ID, "Group", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	group.AgentIDs = []string{agent.ID, agent.ID}
	if _, err := s.UpdateAgentGroup(project.ID, group); err == nil {
		t.Fatal("duplicate membership unexpectedly accepted")
	}
	group.AgentIDs = []string{foreignAgent.ID}
	if _, err := s.UpdateAgentGroup(project.ID, group); err == nil {
		t.Fatal("foreign project agent unexpectedly accepted")
	}
	groups, err := s.ListAgentGroups(project.ID)
	if err != nil || len(groups) != 1 || len(groups[0].AgentIDs) != 0 {
		t.Fatalf("failed update changed persisted group: groups=%+v err=%v", groups, err)
	}
}

func TestDeleteAgentGroupPreservesAgentIdentity(t *testing.T) {
	s := openTestStore(t)
	project, err := s.CreateProject(Project{Name: "Groups", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := s.CreateAgent(Agent{ProjectID: project.ID, Name: "Stable"})
	if err != nil {
		t.Fatal(err)
	}
	group, err := s.CreateAgentGroup(project.ID, "Temporary", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	group.AgentIDs = []string{agent.ID}
	if _, err := s.UpdateAgentGroup(project.ID, group); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteAgentGroup(project.ID, group.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAgent(agent.ID, project.ID); err != nil {
		t.Fatalf("deleting a group changed Agent identity: %v", err)
	}
	groups, err := s.ListAgentGroups(project.ID)
	if err != nil || len(groups) != 0 {
		t.Fatalf("groups after deletion=%+v err=%v", groups, err)
	}
}
