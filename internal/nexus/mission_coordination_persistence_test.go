package nexus

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestMissionCoordinationSurvivesStoreReopenWithoutDuplicateChild(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "nexus.db")

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Coordination", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "Coordination plan"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	run := &runner.MissionRun{
		ID:           "run-coordination-restart",
		PlanID:       plan.ID,
		PlanRevision: plan.CurrentRevision,
		ProjectID:    project.ID,
		Workspace:    project.CanonicalPath,
		State:        runner.StateExecuting,
		Contract:     runner.DefaultAutonomyContract(),
		PackageRuns: []runner.PackageRun{{
			ID:        "pkgrun-source",
			PackageID: "source",
			State:     runner.StateVerified,
			StartedAt: now,
		}},
		StartedAt: now,
		UpdatedAt: now,
	}
	repoBefore := newStoreRunRepository(st)
	if err := repoBefore.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	missionRunnerBefore := runner.NewMissionRunner(repoBefore, nil)
	request, err := missionRunnerBefore.SubmitCoordinationRequest(ctx, run.ID, runner.CoordinationRequest{
		ID:              "coord-restart",
		SourcePackageID: "source",
		Operation:       runner.CoordinationDelegate,
		Subject:         "inspect persisted state",
		Objective:       "inspect persisted state independently",
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := missionRunnerBefore.MaterializeCoordinationWork(ctx, run.ID, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if child.CoordinationRequestID != request.ID || !child.DynamicWork {
		t.Fatalf("unexpected materialized child: %+v", child)
	}

	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	repoAfter := newStoreRunRepository(reopened)
	loaded, err := repoAfter.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.CoordinationRequests) != 1 || len(loaded.PackageRuns) != 2 {
		t.Fatalf("coordination lineage did not survive restart: requests=%d packages=%d", len(loaded.CoordinationRequests), len(loaded.PackageRuns))
	}
	persistedRequest := loaded.CoordinationRequests[0]
	if persistedRequest.ChildPackageID != child.PackageID || persistedRequest.MaterializedAt == nil {
		t.Fatalf("request lost child lineage across restart: %+v", persistedRequest)
	}
	persistedChild := loaded.PackageRuns[1]
	if !persistedChild.DynamicWork || persistedChild.ParentPackageID != "source" || persistedChild.CoordinationRequestID != request.ID {
		t.Fatalf("dynamic child provenance did not survive restart: %+v", persistedChild)
	}

	replayed, err := runner.NewMissionRunner(repoAfter, nil).MaterializeCoordinationWork(ctx, run.ID, request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.PackageID != child.PackageID {
		t.Fatalf("restart replay changed child identity: %q != %q", replayed.PackageID, child.PackageID)
	}
	again, err := repoAfter.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.PackageRuns) != 2 {
		t.Fatalf("restart replay duplicated dynamic work: %d packages", len(again.PackageRuns))
	}
}
