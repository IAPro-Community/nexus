package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMissionNotificationCursorEnqueuesOnceAndSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mission-notifications.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	seedNotificationRun(t, st, "run-1")
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	active := MissionNotificationIntent{Key: "EXECUTING:run-1", RunID: "run-1"}
	if created, err := st.ObserveMissionNotification(active, false, now); err != nil || created {
		t.Fatalf("baseline observation = created %v, err %v", created, err)
	}
	blocked := MissionNotificationIntent{
		Key: "BLOCKED_NEEDS_USER:intv-1:v1:run-1", RunID: "run-1", ProjectID: "project-1",
		SourceID: "intv-1", SourceVersion: 1, Kind: "MISSION_BLOCKED",
		Title: "Nexus — missão aguardando você", Body: "Uma missão precisa de uma decisão. Abra o Centro de Atenção.",
	}
	created, err := st.ObserveMissionNotification(blocked, true, now.Add(time.Second))
	if err != nil || !created {
		t.Fatalf("first blocker transition = created %v, err %v", created, err)
	}
	created, err = st.ObserveMissionNotification(blocked, true, now.Add(2*time.Second))
	if err != nil || created {
		t.Fatalf("duplicate blocker transition = created %v, err %v", created, err)
	}
	pending, err := st.ListPendingMissionNotificationDeliveries(now.Add(3*time.Second), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].IntentKey != blocked.Key || pending[0].Attempts != 0 {
		t.Fatalf("expected one pending native delivery, got %+v", pending)
	}
	if pending[0].Body != blocked.Body || pending[0].RunID != blocked.RunID {
		t.Fatalf("safe delivery payload changed: %+v", pending[0])
	}
	retryAt := now.Add(time.Minute)
	if err := st.RecordMissionNotificationAttempt(blocked.Key, MissionNotificationNative, MissionNotificationPending, "transport unavailable", retryAt, now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	delivery, err := st.GetMissionNotificationDelivery(blocked.Key, MissionNotificationNative)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != MissionNotificationPending || delivery.Attempts != 1 || !delivery.NextAttemptAt.Equal(retryAt) {
		t.Fatalf("retry state did not survive reopen: %+v", delivery)
	}
	if due, err := st.ListPendingMissionNotificationDeliveries(now.Add(30*time.Second), 10); err != nil || len(due) != 0 {
		t.Fatalf("retry became due too early: deliveries=%+v err=%v", due, err)
	}
	if due, err := st.ListPendingMissionNotificationDeliveries(retryAt, 10); err != nil || len(due) != 1 {
		t.Fatalf("retry not recovered after reopen: deliveries=%+v err=%v", due, err)
	}
}

func TestMissionNotificationSupersedesPendingDeliveryAfterStateChange(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "mission-notifications.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	seedNotificationRun(t, st, "run-2")
	now := time.Now().UTC()
	if _, err := st.ObserveMissionNotification(MissionNotificationIntent{Key: "EXECUTING:run-2", RunID: "run-2"}, false, now); err != nil {
		t.Fatal(err)
	}
	blocked := MissionNotificationIntent{Key: "BLOCKED:run-2", RunID: "run-2", Kind: "MISSION_BLOCKED", Title: "Nexus", Body: "Needs attention"}
	if _, err := st.ObserveMissionNotification(blocked, true, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ObserveMissionNotification(MissionNotificationIntent{Key: "EXECUTING_AGAIN:run-2", RunID: "run-2"}, false, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	delivery, err := st.GetMissionNotificationDelivery(blocked.Key, MissionNotificationNative)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != MissionNotificationSuperseded {
		t.Fatalf("stale pending delivery was not superseded: %+v", delivery)
	}
}

func seedNotificationRun(t *testing.T, st *Store, runID string) {
	t.Helper()
	project, err := st.CreateProject(Project{Name: "Notification fixture", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "Notification fixture"})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertMissionRun(MissionRunRecord{ID: runID, PlanID: plan.ID, ProjectID: project.ID, State: "EXECUTING", PayloadJSON: `{}`}); err != nil {
		t.Fatal(err)
	}
}
