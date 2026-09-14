package nexus

import (
	"context"
	"errors"
	"testing"

	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestMissionApplicationServiceCRUDAndDetail(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Missions", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	service := NewMissionApplicationService(n)

	mission, err := service.Create(context.Background(), &store.Mission{ProjectID: project.ID, Name: "Ship", Goal: "Ship safely"})
	if err != nil {
		t.Fatal(err)
	}
	if mission.ID == "" || mission.Status == "" {
		t.Fatalf("mission was not persisted: %#v", mission)
	}
	if err := service.CreateTask(context.Background(), &store.MissionTask{MissionID: mission.ID, Name: "Build"}); err != nil {
		t.Fatal(err)
	}
	agent, err := st.CreateAgent(store.Agent{ProjectID: project.ID, Name: "Builder"})
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := st.ListTasks(mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Assign(context.Background(), &store.MissionAssignment{MissionID: mission.ID, TaskID: tasks[0].ID, AgentID: agent.ID}); err != nil {
		t.Fatal(err)
	}
	detail, err := service.Detail(context.Background(), mission.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Tasks) != 1 || len(detail.Assignments) != 1 || detail.Stats.Total != 1 {
		t.Fatalf("unexpected mission detail: %#v", detail)
	}

	listed, err := service.List(context.Background(), project.ID)
	if err != nil || len(listed) != 1 {
		t.Fatalf("missions = %#v, err = %v", listed, err)
	}

	if err := service.Delete(context.Background(), mission.ID); err != nil {
		t.Fatal(err)
	}
}

func TestMissionApplicationServiceRejectsCanceledContext(t *testing.T) {
	n := openTestNexus(t)
	service := NewMissionApplicationService(n)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.List(ctx, "project"); !errors.Is(err, context.Canceled) {
		t.Fatalf("list error = %v, want context canceled", err)
	}
}

func TestMissionApplicationServiceRejectsCrossProjectAssignments(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	projectA, err := st.CreateProject(store.Project{Name: "Mission A", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	projectB, err := st.CreateProject(store.Project{Name: "Mission B", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	service := NewMissionApplicationService(n)
	missionA, err := service.Create(context.Background(), &store.Mission{ProjectID: projectA.ID, Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	missionB, err := service.Create(context.Background(), &store.Mission{ProjectID: projectB.ID, Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.CreateTask(context.Background(), &store.MissionTask{MissionID: missionA.ID, Name: "A task"}); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateTask(context.Background(), &store.MissionTask{MissionID: missionB.ID, Name: "B task"}); err != nil {
		t.Fatal(err)
	}
	agentA, err := st.CreateAgent(store.Agent{ProjectID: projectA.ID, Name: "A agent"})
	if err != nil {
		t.Fatal(err)
	}
	agentB, err := st.CreateAgent(store.Agent{ProjectID: projectB.ID, Name: "B agent"})
	if err != nil {
		t.Fatal(err)
	}
	tasksA, err := st.ListTasks(missionA.ID)
	if err != nil {
		t.Fatal(err)
	}
	tasksB, err := st.ListTasks(missionB.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Assign(context.Background(), &store.MissionAssignment{
		MissionID: missionA.ID,
		TaskID:    tasksB[0].ID,
		AgentID:   agentA.ID,
	}); err == nil {
		t.Fatal("expected task from another mission/project to be rejected")
	}
	if err := service.Assign(context.Background(), &store.MissionAssignment{
		MissionID: missionA.ID,
		TaskID:    tasksA[0].ID,
		AgentID:   agentB.ID,
	}); err == nil {
		t.Fatal("expected agent from another project to be rejected")
	}
}

func TestMissionApplicationServiceRejectsTerminalStatusPatch(t *testing.T) {
	n := openTestNexus(t)
	st, err := n.OpenProject()
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Terminal Patch", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	service := NewMissionApplicationService(n)
	mission, err := service.Create(context.Background(), &store.Mission{ProjectID: project.ID, Name: "Ship", Goal: "Ship safely"})
	if err != nil {
		t.Fatal(err)
	}

	for _, status := range []string{store.MissionCompleted, "COMPLETED_VERIFIED", "FAILED", "FAILED_NO_PROGRESS"} {
		_, err := service.Update(context.Background(), mission.ID, MissionPatch{Status: &status})
		if !errors.Is(err, ErrTerminalMissionStatusPatch) {
			t.Fatalf("status %s: error = %v, want ErrTerminalMissionStatusPatch", status, err)
		}
	}

	name := "Renamed"
	updated, err := service.Update(context.Background(), mission.ID, MissionPatch{Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Renamed" {
		t.Fatalf("expected name patch to succeed, got %#v", updated)
	}
	ready := store.MissionReady
	updated, err = service.Update(context.Background(), mission.ID, MissionPatch{Status: &ready})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != store.MissionReady {
		t.Fatalf("expected non-terminal status patch, got %#v", updated)
	}
}
