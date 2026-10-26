package store

import (
    "database/sql"
    "errors"
    "testing"
)

func deletionProject(t *testing.T, st *Store) *Project {
    t.Helper()
    project, err := st.CreateProject(Project{Name: "Delete planning fixture", CanonicalPath: t.TempDir()})
    if err != nil { t.Fatal(err) }
    return project
}

func TestDeleteComposerSessionCascadesOnlyItsUnusedDraft(t *testing.T) {
    st := openTestStore(t)
    project := deletionProject(t, st)
    session, err := st.CreateComposerSession(ComposerSession{ProjectID: project.ID, Title: "Temporary idea"})
    if err != nil { t.Fatal(err) }
    other, err := st.CreateComposerSession(ComposerSession{ProjectID: project.ID, Title: "Keep this"})
    if err != nil { t.Fatal(err) }
    if _, err := st.AppendComposerTurn(ComposerTurn{SessionID: session.ID, Role: ComposerUser, Content: "First idea"}); err != nil { t.Fatal(err) }
    artifact, err := st.CreatePromptArtifact(PromptArtifact{SessionID: session.ID, Content: "Plan the solution"})
    if err != nil { t.Fatal(err) }
    if _, err := st.CreatePromptVariant(PromptVariant{ArtifactID: artifact.ID, Variant: "summary", Content: "Short plan"}); err != nil { t.Fatal(err) }

    if err := st.DeleteComposerSessionExpected(session.ID, session.Revision+1); !errors.Is(err, ErrPlanRevisionConflict) {
        t.Fatalf("stale revision must fail, got %v", err)
    }
    if _, err := st.GetComposerSession(session.ID); err != nil { t.Fatalf("stale revision deleted session: %v", err) }
    if err := st.DeleteComposerSessionExpected(session.ID, session.Revision); err != nil { t.Fatal(err) }
    if _, err := st.GetComposerSession(session.ID); !errors.Is(err, sql.ErrNoRows) { t.Fatalf("session still present: %v", err) }
    if _, err := st.GetPromptArtifact(artifact.ID); !errors.Is(err, sql.ErrNoRows) { t.Fatalf("artifact still present: %v", err) }
    if _, err := st.GetComposerSession(other.ID); err != nil { t.Fatalf("unrelated session lost: %v", err) }
    var turns, variants int
    if err := st.DB().QueryRow(`SELECT COUNT(*) FROM composer_turns WHERE session_id=?`, session.ID).Scan(&turns); err != nil { t.Fatal(err) }
    if err := st.DB().QueryRow(`SELECT COUNT(*) FROM composer_prompt_variants WHERE artifact_id=?`, artifact.ID).Scan(&variants); err != nil { t.Fatal(err) }
    if turns != 0 || variants != 0 { t.Fatalf("orphan records: turns=%d variants=%d", turns, variants) }
}

func TestDeleteComposerSessionRefusesLinkedWorkPlan(t *testing.T) {
    st := openTestStore(t)
    project := deletionProject(t, st)
    session, err := st.CreateComposerSession(ComposerSession{ProjectID: project.ID, Title: "Plan with Flow"})
    if err != nil { t.Fatal(err) }
    artifact, err := st.CreatePromptArtifact(PromptArtifact{SessionID: session.ID, Content: "Implement workflow"})
    if err != nil { t.Fatal(err) }
    plan, err := st.CreateWorkPlan(WorkPlan{
        ProjectID: project.ID, Title: "Linked Flow",
        StructuredFacts: map[string]string{"nexus.source_artifact_id": artifact.ID},
    })
    if err != nil { t.Fatal(err) }
    if err := st.DeleteComposerSessionExpected(session.ID, session.Revision); !errors.Is(err, ErrComposerDeleteProtected) {
        t.Fatalf("must keep Composer lineage while Flow exists: %v", err)
    }
    if _, err := st.GetComposerSession(session.ID); err != nil { t.Fatal(err) }
    if err := st.DeleteWorkPlanExpected(plan.ID, plan.CurrentRevision); err != nil { t.Fatal(err) }
    if err := st.DeleteComposerSessionExpected(session.ID, session.Revision); err != nil { t.Fatal(err) }
}

func TestDeleteWorkPlanPreservesMissionRunAndSnapshot(t *testing.T) {
    st := openTestStore(t)
    project := deletionProject(t, st)
    plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "Executed Flow"})
    if err != nil { t.Fatal(err) }
    if err := st.UpsertMissionRun(MissionRunRecord{
        ID: "run-delete-guard", PlanID: plan.ID, ProjectID: project.ID,
        State: "COMPLETED_VERIFIED", PayloadJSON: "{}",
    }); err != nil { t.Fatal(err) }
    revision, err := st.GetPlanRevision(plan.ID, 1)
    if err != nil { t.Fatal(err) }
    snapshot, err := st.CreateExecutionSnapshot(plan.ID, revision.ID, "{}")
    if err != nil { t.Fatal(err) }
    if err := st.DeleteWorkPlanExpected(plan.ID, plan.CurrentRevision); !errors.Is(err, ErrPlanDeleteProtected) {
        t.Fatalf("Mission history must be protected: %v", err)
    }
    if _, err := st.GetMissionRun("run-delete-guard"); err != nil { t.Fatalf("Mission was deleted: %v", err) }
    if _, err := st.GetExecutionSnapshot(snapshot.ID); err != nil { t.Fatalf("ExecutionSnapshot was deleted: %v", err) }
    if _, err := st.GetWorkPlan(plan.ID); err != nil { t.Fatalf("plan was deleted: %v", err) }
}

func TestDeleteWorkPlanRejectsScheduledAndStalePlan(t *testing.T) {
    st := openTestStore(t)
    project := deletionProject(t, st)
    plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "Scheduled Flow"})
    if err != nil { t.Fatal(err) }
    if _, err := st.CreateMissionSchedule(MissionSchedule{
        PlanID: plan.ID, ProjectID: project.ID, Mode: "AT", Status: SchedulePending,
    }); err != nil { t.Fatal(err) }
    if err := st.DeleteWorkPlanExpected(plan.ID, plan.CurrentRevision); !errors.Is(err, ErrPlanDeleteProtected) {
        t.Fatalf("schedule should prevent deletion: %v", err)
    }

    unused, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "Unused Flow"})
    if err != nil { t.Fatal(err) }
    if err := st.DeleteWorkPlanExpected(unused.ID, unused.CurrentRevision+1); !errors.Is(err, ErrPlanRevisionConflict) {
        t.Fatalf("must reject stale confirmation: %v", err)
    }
    if err := st.DeleteWorkPlanExpected(unused.ID, unused.CurrentRevision); err != nil { t.Fatal(err) }
    if _, err := st.GetWorkPlan(unused.ID); !errors.Is(err, sql.ErrNoRows) {
        t.Fatalf("unused Flow still persisted: %v", err)
    }
    if _, err := st.GetPlanRevision(unused.ID, 1); !errors.Is(err, sql.ErrNoRows) {
        t.Fatalf("Flow revision history still persisted: %v", err)
    }
}

func TestDeleteComposerRefusesPublishedDeliveryReceipt(t *testing.T) {
    st := openTestStore(t)
    project := deletionProject(t, st)
    session, err := st.CreateComposerSession(ComposerSession{ProjectID: project.ID, Title: "External delivery"})
    if err != nil { t.Fatal(err) }
    artifact, err := st.CreatePromptArtifact(PromptArtifact{SessionID: session.ID, Content: "Published specification"})
    if err != nil { t.Fatal(err) }
    if _, err := st.RecordComposerDestinationReceipt(ComposerDestinationReceipt{
        ArtifactID: artifact.ID, Destination: "AGENT", Variant: "canonical", Status: "DELIVERED",
    }); err != nil { t.Fatal(err) }
    if err := st.DeleteComposerSessionExpected(session.ID, session.Revision); !errors.Is(err, ErrComposerDeleteProtected) {
        t.Fatalf("delivery evidence must keep Composer session: %v", err)
    }
    if _, err := st.GetPromptArtifact(artifact.ID); err != nil { t.Fatalf("published artifact was deleted: %v", err) }
}

func TestDeleteWorkPlanProtectsSnapshotWithoutMissionRun(t *testing.T) {
    st := openTestStore(t)
    project := deletionProject(t, st)
    plan, err := st.CreateWorkPlan(WorkPlan{ProjectID: project.ID, Title: "Prepared execution"})
    if err != nil { t.Fatal(err) }
    revision, err := st.GetPlanRevision(plan.ID, 1)
    if err != nil { t.Fatal(err) }
    snapshot, err := st.CreateExecutionSnapshot(plan.ID, revision.ID, "{}")
    if err != nil { t.Fatal(err) }
    if err := st.DeleteWorkPlanExpected(plan.ID, plan.CurrentRevision); !errors.Is(err, ErrPlanDeleteProtected) {
        t.Fatalf("prepared execution snapshot should block deletion, got %v", err)
    }
    if _, err := st.GetExecutionSnapshot(snapshot.ID); err != nil { t.Fatalf("snapshot lost: %v", err) }
}
