package runner

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryRunRepositoryHonorsCanceledContext(t *testing.T) {
	repo := NewMemoryRunRepository()
	run := &MissionRun{ID: "run_context", PlanID: "plan_context", ProjectID: "project_context"}
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
}

func TestMemoryEvidenceRepositoryHonorsCanceledContext(t *testing.T) {
	repo := NewMemoryRunRepository()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	wantCanceled := func(t *testing.T, err error) {
		t.Helper()
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
	}

	wantCanceled(t, repo.SaveContextCapsule(ctx, nil))
	_, err := repo.GetContextCapsule(ctx, "run", "step")
	wantCanceled(t, err)
	wantCanceled(t, repo.SaveWorkReceipt(ctx, nil))
	_, err = repo.GetWorkReceipt(ctx, "run", "step")
	wantCanceled(t, err)
}
