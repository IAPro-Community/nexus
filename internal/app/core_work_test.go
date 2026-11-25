package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestHasActiveWorkIn(t *testing.T) {
	if busy, reason := hasActiveWorkIn(nil, nil, false); busy || reason != "" {
		t.Fatalf("expected idle when empty, got busy=%v reason=%q", busy, reason)
	}

	reg := registry.NewRegistry(filepath.Join(t.TempDir(), "registry.json"))
	if err := reg.Register(registry.RuntimeSession{
		RuntimeID:  "rt-live",
		ProviderID: "codex",
		ProfileID:  "work",
		State:      registry.StateRunning,
		UpdatedAt:  time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if busy, _ := hasActiveWorkIn(reg, nil, false); !busy {
		t.Fatal("expected busy=true with active runtime")
	}

	st, err := store.Open(filepath.Join(t.TempDir(), "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	proj, err := st.CreateProject(store.Project{Name: "P", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{
		ProjectID: proj.ID,
		Title:     "Plan",
		Phases: []store.PlanPhase{{
			ID:       "phase-1",
			Title:    "Phase",
			Packages: []store.WorkPackage{{ID: "pkg-1", Title: "Pkg", Goal: "Goal"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertMissionRun(store.MissionRunRecord{
		ID:          "run-paused",
		PlanID:      plan.ID,
		ProjectID:   proj.ID,
		State:       string(runner.StatePaused),
		PayloadJSON: `{"id":"run-paused"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if busy, _ := hasActiveWorkIn(nil, st, true); busy {
		t.Fatal("paused mission run should not report active work")
	}

	if err := st.UpsertMissionRun(store.MissionRunRecord{
		ID:          "run-exec",
		PlanID:      plan.ID,
		ProjectID:   proj.ID,
		State:       string(runner.StateExecuting),
		PayloadJSON: `{"id":"run-exec"}`,
	}); err != nil {
		t.Fatal(err)
	}
	if busy, _ := hasActiveWorkIn(nil, st, true); !busy {
		t.Fatal("executing mission run should report active work")
	}
	if busy, _ := hasActiveWorkIn(nil, st, false); busy {
		t.Fatal("missionsEnabled=false should not count mission runs as active work")
	}
}
