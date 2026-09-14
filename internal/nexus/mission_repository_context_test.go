package nexus

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestStoreRunRepositoryHonorsCanceledContext(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/nexus.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	project, err := st.CreateProject(store.Project{Name: "Context", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "Context", Phases: []store.PlanPhase{{ID: "phase", Title: "Phase", Packages: []store.WorkPackage{{ID: "package", Title: "Package", Goal: "Goal"}}}}})
	if err != nil {
		t.Fatalf("create plan: %v", err)
	}
	repo := newStoreRunRepository(st)
	run := &runner.MissionRun{ID: "run_context", PlanID: plan.ID, ProjectID: project.ID}
	if err := repo.SaveRun(context.Background(), run); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wantCanceled := func(t *testing.T, err error) {
		t.Helper()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
	}

	t.Run("save", func(t *testing.T) {
		wantCanceled(t, repo.SaveRun(ctx, run))
	})
	t.Run("get", func(t *testing.T) {
		_, err := repo.GetRun(ctx, run.ID)
		wantCanceled(t, err)
	})
	t.Run("list", func(t *testing.T) {
		_, err := repo.ListRuns(ctx)
		wantCanceled(t, err)
	})
	t.Run("acquire lease", func(t *testing.T) {
		_, err := repo.AcquireLease(ctx, run.ID, "owner", time.Minute)
		wantCanceled(t, err)
	})
	t.Run("renew lease", func(t *testing.T) {
		wantCanceled(t, repo.RenewLease(ctx, run.ID, "owner", "token", time.Minute))
	})
	t.Run("release lease", func(t *testing.T) {
		wantCanceled(t, repo.ReleaseLease(ctx, run.ID, "owner", "token"))
	})
	t.Run("save context capsule", func(t *testing.T) {
		wantCanceled(t, repo.(runner.EvidenceRepository).SaveContextCapsule(ctx, nil))
	})
	t.Run("get context capsule", func(t *testing.T) {
		_, err := repo.(runner.EvidenceRepository).GetContextCapsule(ctx, run.ID, "step")
		wantCanceled(t, err)
	})
	t.Run("save work receipt", func(t *testing.T) {
		wantCanceled(t, repo.(runner.EvidenceRepository).SaveWorkReceipt(ctx, nil))
	})
	t.Run("get work receipt", func(t *testing.T) {
		_, err := repo.(runner.EvidenceRepository).GetWorkReceipt(ctx, run.ID, "step")
		wantCanceled(t, err)
	})
}
