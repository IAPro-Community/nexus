package nexus

import (
	"context"
	"errors"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestDeleteProjectWithOptionsDetachKeepsLiveRuntime(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P-Detach", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "AGY Lead"})

	sess, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}

	// Default DeleteProject refuses and exposes structured live agent metadata.
	err = n.DeleteProject(proj.ID)
	var blocked *ProjectDeleteBlockedError
	if !errors.As(err, &blocked) {
		t.Fatalf("expected *ProjectDeleteBlockedError, got %v", err)
	}
	if len(blocked.LiveAgents) != 1 || blocked.LiveAgents[0].ID != agent.ID || blocked.LiveAgents[0].Provider != "fake" {
		t.Fatalf("unexpected LiveAgents metadata: %+v", blocked.LiveAgents)
	}

	// Detach mode removes project metadata while leaving the runtime untouched.
	res, err := n.DeleteProjectWithOptions(context.Background(), proj.ID, ProjectDeleteOptions{
		Mode: ProjectDeleteDetachOnly,
	})
	if err != nil {
		t.Fatalf("DeleteProjectWithOptions(detach): %v", err)
	}
	if res.Mode != "detach" || len(res.DetachedAgents) != 1 {
		t.Fatalf("unexpected detach result: %+v", res)
	}
	if _, getErr := st.GetProject(proj.ID); getErr == nil {
		t.Fatal("expected project to be removed after detach delete")
	}
	if !n.runtimeAlive(sess.RuntimeID) {
		t.Fatal("expected live runtime to stay alive after detach delete")
	}
}

func TestDeleteProjectWithOptionsStopAgentsStopsAndDeletes(t *testing.T) {
	n := openTestNexus(t)
	st, _ := n.OpenProject()
	proj, _ := st.CreateProject(store.Project{Name: "P-Stop", CanonicalPath: t.TempDir()})
	agent, _ := st.CreateAgent(store.Agent{ProjectID: proj.ID, Name: "Codex Worker"})

	sess, err := n.StartAgent(context.Background(), agent.ID, "fake", "default")
	if err != nil {
		t.Fatal(err)
	}

	res, err := n.DeleteProjectWithOptions(context.Background(), proj.ID, ProjectDeleteOptions{
		Mode: ProjectDeleteStopAgents,
	})
	if err != nil {
		t.Fatalf("DeleteProjectWithOptions(stop_agents): %v", err)
	}
	if res.Mode != "stop_agents" || len(res.StoppedAgents) != 1 {
		t.Fatalf("unexpected stop_agents result: %+v", res)
	}
	if _, getErr := st.GetProject(proj.ID); getErr == nil {
		t.Fatal("expected project to be removed after stop_agents delete")
	}
	if n.runtimeAlive(sess.RuntimeID) {
		t.Fatal("expected runtime to be stopped after stop_agents delete")
	}
}
