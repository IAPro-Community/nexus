package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func main() {
	root := os.Getenv("NEXUS_DATA_DIR")
	if root == "" {
		fatalf("NEXUS_DATA_DIR must point to an isolated data directory")
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "activate":
			activateMissions(root)
		case "complete":
			completeMissions(root)
		default:
			fatalf("unsupported fixture command %q; use activate or complete", os.Args[1])
		}
		return
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		fatalf("create isolated data directory: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		fatalf("inspect isolated data directory: %v", err)
	}
	if len(entries) != 0 {
		fatalf("refusing to seed non-empty data directory %q", root)
	}
	if err := os.MkdirAll(filepath.Join(root, "projects"), 0o700); err != nil {
		fatalf("create fixture project root: %v", err)
	}

	st, err := store.Open(filepath.Join(root, "nexus.db"))
	if err != nil {
		fatalf("open isolated fixture database: %v", err)
	}
	defer st.Close()

	now := time.Now().UTC()
	items := []struct {
		key      string
		state    runner.State
		complete bool
		needsYou bool
	}{
		{key: "a", state: runner.StateBlockedNeedsUser, needsYou: true},
		{key: "b", state: runner.StateCompletedVerified, complete: true},
		{key: "c", state: runner.StateCompletedVerified, complete: true},
		{key: "d", state: runner.StateCompletedVerified, complete: true},
		{key: "e", state: runner.StateFailedNoProgress},
	}
	if os.Getenv("NEXUS_ATTENTION_FIXTURE_PHASE") == "base" {
		items = []struct {
			key      string
			state    runner.State
			complete bool
			needsYou bool
		}{items[0], items[3], items[4]}
	}
	for i, item := range items {
		seedMission(st, root, item.key, item.state, item.complete, item.needsYou, now.Add(-time.Duration(i+1)*time.Minute))
	}
	if os.Getenv("NEXUS_ATTENTION_FIXTURE_PHASE") == "base" {
		for _, key := range []string{"b", "c"} {
			ensureFixtureProject(st, root, key)
		}
	}
}

func seedMission(st *store.Store, root, key string, state runner.State, complete, needsYou bool, updated time.Time) {
	project := ensureFixtureProject(st, root, key)
	plan, err := st.CreateWorkPlan(store.WorkPlan{
		ID:        "fixture-plan-" + key,
		ProjectID: project.ID,
		Title:     "Fixture de atenção " + key,
		Status:    "COMPLETED",
		Phases: []store.PlanPhase{{
			ID:    "phase-" + key,
			Title: "Validação isolada",
			Order: 1,
			Packages: []store.WorkPackage{{
				ID:     "package-" + key,
				Title:  "Verificar estado persistido",
				Goal:   "Apresentar o estado da missão no Attention Center",
				Role:   "implementer",
				Status: "VERIFIED",
			}},
		}},
	})
	if err != nil {
		fatalf("persist plan %s: %v", key, err)
	}
	contract := runner.DefaultAutonomyContract()
	contract.RequireVerification = false
	contract.EscalateOnFailure = true
	run := runner.MissionRun{
		ID:              "fixture-mission-" + key,
		PlanID:          plan.ID,
		PlanRevision:    plan.CurrentRevision,
		ProjectID:       project.ID,
		Workspace:       project.CanonicalPath,
		State:           state,
		Contract:        contract,
		CurrentPkgIndex: 0,
		StartedAt:       updated,
		LastProgressAt:  updated,
		UpdatedAt:       updated,
	}
	if complete {
		completed := updated.Add(time.Second)
		run.CompletedAt = &completed
	}
	if needsYou {
		run.NeedsHuman = &runner.HumanIntervention{
			ID:                 "fixture-intervention-a",
			ReasonCode:         "DISPATCH_OUTCOME_UNKNOWN",
			Summary:            "Aguarda decisão: conferir o resultado externo",
			Question:           "A execução terminou no provedor. Deseja confirmar o resultado?",
			Context:            "Fixture isolada da homologação Linux.",
			RecommendedActions: []string{"Inspecionar o resultado", "Retomar com segurança"},
			Impact:             "A Mission não continua sem uma decisão explícita.",
			MissionID:          run.ID,
			TaskID:             "package-a",
			Source:             "provider",
			Timestamp:          updated,
			Version:            1,
			Scope:              "MISSION",
		}
	}
	payload, err := json.Marshal(run)
	if err != nil {
		fatalf("encode mission %s: %v", key, err)
	}
	if err := st.UpsertMissionRun(store.MissionRunRecord{
		ID:          run.ID,
		PlanID:      plan.ID,
		ProjectID:   project.ID,
		State:       string(run.State),
		PayloadJSON: string(payload),
		CreatedAt:   updated,
		UpdatedAt:   updated,
	}); err != nil {
		fatalf("persist mission %s: %v", key, err)
	}
	fmt.Printf("seeded project=%s mission=%s state=%s\n", project.ID, run.ID, run.State)
}

func ensureFixtureProject(st *store.Store, root, key string) store.Project {
	projectID := "fixture-project-" + key
	project, err := st.GetProject(projectID)
	if err == nil {
		return project
	}
	if !errors.Is(err, store.ErrNotFound) {
		fatalf("read project %s: %v", key, err)
	}
	workspace := filepath.Join(root, "projects", key)
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		fatalf("create project %s: %v", key, err)
	}
	project, err = st.CreateProject(store.Project{
		ID:            projectID,
		Name:          "Homologação Missão " + key,
		CanonicalPath: workspace,
	})
	if err != nil {
		fatalf("persist project %s: %v", key, err)
	}
	return project
}

func activateMissions(root string) {
	st := openFixtureStore(root)
	defer st.Close()
	now := time.Now().UTC()
	seedMission(st, root, "b", runner.StateExecuting, false, false, now)
	seedMission(st, root, "c", runner.StateRemediating, false, false, now.Add(-time.Second))
}

func completeMissions(root string) {
	st := openFixtureStore(root)
	defer st.Close()
	for _, key := range []string{"b", "c"} {
		runID := "fixture-mission-" + key
		record, err := st.GetMissionRun(runID)
		if err != nil {
			fatalf("read mission %s: %v", runID, err)
		}
		var run runner.MissionRun
		if err := json.Unmarshal([]byte(record.PayloadJSON), &run); err != nil {
			fatalf("decode mission %s: %v", runID, err)
		}
		now := time.Now().UTC()
		run.State = runner.StateCompletedVerified
		run.CompletedAt = &now
		run.LastProgressAt = now
		run.UpdatedAt = now
		payload, err := json.Marshal(run)
		if err != nil {
			fatalf("encode mission %s: %v", runID, err)
		}
		if err := st.UpsertMissionRun(store.MissionRunRecord{
			ID:          record.ID,
			PlanID:      record.PlanID,
			ProjectID:   record.ProjectID,
			State:       string(run.State),
			PayloadJSON: string(payload),
		}); err != nil {
			fatalf("complete mission %s: %v", runID, err)
		}
		fmt.Printf("completed mission=%s state=%s\n", runID, run.State)
	}
}

func openFixtureStore(root string) *store.Store {
	st, err := store.Open(filepath.Join(root, "nexus.db"))
	if err != nil {
		fatalf("open isolated fixture database: %v", err)
	}
	return st
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
