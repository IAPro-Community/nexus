package nexus

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/handoff"
	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/memoryrecovery"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func migrationTestNexus(t *testing.T) (*Nexus, string) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("XDG_CONFIG_HOME", dataDir+"/config")
	t.Setenv("XDG_DATA_HOME", dataDir+"/data")
	t.Setenv("XDG_STATE_HOME", dataDir+"/state")
	t.Setenv("HOME", dataDir+"/home")
	st, err := store.Open(filepath.Join(dataDir, "nexus.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	project, err := st.CreateProject(store.Project{Name: "Mig", CanonicalPath: dataDir + "/workspace"})
	if err != nil {
		t.Fatal(err)
	}
	n := &Nexus{st: st}
	return n, project.ID
}

func resetMigrationSeams(t *testing.T, exporter func(context.Context, string, handoff.MaestroMemoryWriteRequest) (handoff.MaestroMemoryWriteResult, error)) {
	t.Helper()
	originalExporter, originalReader := maestroMemoryExporter, maestroMemoryReader
	t.Cleanup(func() {
		maestroMemoryExporter, maestroMemoryReader = originalExporter, originalReader
	})
	maestroMemoryExporter = exporter
	if exporter == nil {
		maestroMemoryReader = func(ctx context.Context, workspace, taskID, search string, limit int) (handoff.MaestroMemoryReadResult, error) {
			return handoff.MaestroMemoryReadResult{}, errors.New("no bridge in this test")
		}
	}
}

func addMigrationMemories(t *testing.T, s *store.Store, projectID string) {
	t.Helper()
	if _, err := s.UpsertIntelligenceMemory(store.IntelligenceMemory{ProjectID: projectID, Key: "quality.testing", Value: "run go test ./...", Source: "explicit"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpsertIntelligenceMemory(store.IntelligenceMemory{ProjectID: projectID, Key: "pending.promotion", Value: "never exported", Source: "composer"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := coreconfig.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.IntelligenceMemories = append(cfg.IntelligenceMemories, coreconfig.IntelligenceMemory{
		ID: "imem_install1", Key: "environment.stack", Value: "linux docker", Source: "composer",
	})
	if err := coreconfig.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestPlanMemoryMigrationIsEmptyWithoutMemories(t *testing.T) {
	n, projectID := migrationTestNexus(t)
	_ = projectID
	plan, err := n.PlanMemoryMigration(context.Background(), "/workspace")
	if err != nil {
		t.Fatalf("empty dry-run must succeed: %v", err)
	}
	if !plan.DryRun || plan.InstallCount != 0 || len(plan.Projects) != 0 || plan.SourceDigestSHA256 == "" {
		t.Fatalf("unexpected empty plan: %+v", plan)
	}
	dataDir, _ := coreconfig.DataDir()
	if inspection, err := memoryrecovery.Inspect(dataDir); err != nil || len(inspection.Operations) != 0 || inspection.LockPresent {
		t.Fatalf("dry-run must not create a journal operation: %+v %v", inspection, err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "migration-backups")); !os.IsNotExist(err) {
		t.Fatal("dry-run must not create backups")
	}
}

func TestPlanMemoryMigrationCountsProjectsAndInstall(t *testing.T) {
	n, projectID := migrationTestNexus(t)
	addMigrationMemories(t, n.st, projectID)
	plan, err := n.PlanMemoryMigration(context.Background(), "/workspace")
	if err != nil {
		t.Fatalf("dry-run failed: %v", err)
	}
	if plan.InstallCount != 1 || len(plan.Projects) != 1 || plan.Projects[0].Count != 1 {
		t.Fatalf("pending.* rows and empty rows must be excluded: %+v", plan)
	}
	if plan.Projects[0].ProjectID != projectID {
		t.Fatalf("unexpected project id: %+v", plan)
	}
	second, err := n.PlanMemoryMigration(context.Background(), "/workspace")
	if err != nil || second.SourceDigestSHA256 != plan.SourceDigestSHA256 {
		t.Fatalf("digest must be deterministic: %v vs %v, err %v", plan.SourceDigestSHA256, second.SourceDigestSHA256, err)
	}
}

func TestMigrateMemoryRefusesUnconfirmedAndHasNoSideEffects(t *testing.T) {
	n, projectID := migrationTestNexus(t)
	addMigrationMemories(t, n.st, projectID)
	resetMigrationSeams(t, func(ctx context.Context, workspace string, request handoff.MaestroMemoryWriteRequest) (handoff.MaestroMemoryWriteResult, error) {
		t.Fatal("exporter must not be called before confirmation")
		return handoff.MaestroMemoryWriteResult{}, errors.New("unreachable")
	})
	dataDir, _ := coreconfig.DataDir()
	if _, err := n.MigrateIntelligenceMemoryToMaestro(context.Background(), "/workspace", false, "Migration without confirmation"); err == nil {
		t.Fatal("unconfirmed migration must fail")
	}
	if inspection, err := memoryrecovery.Inspect(dataDir); err != nil || len(inspection.Operations) != 0 || inspection.LockPresent {
		t.Fatalf("unconfirmed migration must not create a journal operation: %+v %v", inspection, err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "migration-backups")); !os.IsNotExist(err) {
		t.Fatal("unconfirmed migration must not create backups")
	}
	gateFile := filepath.Join(dataDir, "memory-migration-state.json")
	if _, err := os.Stat(gateFile); !os.IsNotExist(err) {
		t.Fatal("unconfirmed migration must not latch the gate")
	}
}

func TestMigrateMemoryRefusesShortReasonWithoutSideEffects(t *testing.T) {
	n, projectID := migrationTestNexus(t)
	addMigrationMemories(t, n.st, projectID)
	resetMigrationSeams(t, nil)
	if _, err := n.MigrateIntelligenceMemoryToMaestro(context.Background(), "/workspace", true, "short"); err == nil {
		t.Fatal("short reason must fail the preflight")
	}
}

func recordedResult() handoff.MaestroMemoryWriteResult {
	return handoff.MaestroMemoryWriteResult{
		SchemaVersion: 1, Authority: "maestro", Status: "RECORDED",
		ID:    "obs_0123456789abcdef",
		Scope: handoff.MaestroMemoryScope{Level: "workspace", RepositoryID: "repo_migration"},
	}
}

func TestMigrateMemoryHappyPathLatchesGate(t *testing.T) {
	n, projectID := migrationTestNexus(t)
	addMigrationMemories(t, n.st, projectID)
	exported := map[string]int{}
	resetMigrationSeams(t, func(ctx context.Context, workspace string, request handoff.MaestroMemoryWriteRequest) (handoff.MaestroMemoryWriteResult, error) {
		exported[request.IdempotencyKey]++
		if !request.Confirmed || request.ScopeLevel != "workspace" || request.ProjectPath != workspace {
			return handoff.MaestroMemoryWriteResult{}, errors.New("invalid export request")
		}
		result := recordedResult()
		result.ID = "obs_" + strings.ReplaceAll(strings.Repeat("ab", 8), "ab", "ab")[:16]
		return result, nil
	})
	maestroMemoryReader = func(ctx context.Context, workspace, taskID, search string, limit int) (handoff.MaestroMemoryReadResult, error) {
		return handoff.MaestroMemoryReadResult{Items: []handoff.MaestroMemoryObservation{{ID: "obs_0123456789abcdef", Summary: search, Scope: handoff.MaestroMemoryScope{Level: "workspace", RepositoryID: "repo"}}}}, nil
	}
	result, err := n.MigrateIntelligenceMemoryToMaestro(context.Background(), "/workspace", true, "Migrate project habits to Maestro")
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if !result.GateClosed || result.InstallCount != 1 || len(result.Projects) != 1 || result.Projects[0].Exported != 1 {
		t.Fatalf("unexpected migration result: %+v", result)
	}
	if len(exported) != 2 {
		t.Fatalf("pending.* row must not be exported: %v", exported)
	}
	for _, count := range exported {
		if count != 1 {
			t.Fatalf("duplicate export detected: %v", exported)
		}
	}
	dataDir, _ := coreconfig.DataDir()
	if _, err := os.Stat(filepath.Join(dataDir, "memory-migration-state.json")); err != nil {
		t.Fatalf("gate file must exist: %v", err)
	}
	inspection, err := memoryrecovery.Inspect(dataDir)
	if err != nil || inspection.Blocked {
		t.Fatalf("journal clean after verified migration: %+v %v", inspection, err)
	}
	if _, err := n.st.UpsertIntelligenceMemory(store.IntelligenceMemory{ProjectID: projectID, Key: "quality.testing", Value: "changed"}); !errors.Is(err, store.ErrLegacyMemoryWritesDisabled) {
		t.Fatalf("writers must be disabled after migration: %v", err)
	}
	if _, err := n.MigrateIntelligenceMemoryToMaestro(context.Background(), "/workspace", true, "Migrate project habits again"); !errors.Is(err, ErrMemoryAlreadyMigrated) {
		t.Fatalf("second migration must be refused by the gate: %v", err)
	}
}

func TestRecoverMemoryMigrationResumesInterruptedOperationWithoutDuplicates(t *testing.T) {
	n, projectID := migrationTestNexus(t)
	addMigrationMemories(t, n.st, projectID)
	seen := map[string]int{}
	failSecond := true
	resetMigrationSeams(t, func(ctx context.Context, workspace string, request handoff.MaestroMemoryWriteRequest) (handoff.MaestroMemoryWriteResult, error) {
		seen[request.IdempotencyKey]++
		if failSecond && seen[request.IdempotencyKey] == 1 && len(seen) == 2 {
			failSecond = false
			return handoff.MaestroMemoryWriteResult{}, errors.New("bridge exploded")
		}
		result := recordedResult()
		if seen[request.IdempotencyKey] > 1 {
			result.Status = "REPLAYED"
		}
		return result, nil
	})
	maestroMemoryReader = func(ctx context.Context, workspace, taskID, search string, limit int) (handoff.MaestroMemoryReadResult, error) {
		return handoff.MaestroMemoryReadResult{Items: []handoff.MaestroMemoryObservation{{ID: "obs_0123456789abcdef", Summary: search, Scope: handoff.MaestroMemoryScope{Level: "workspace", RepositoryID: "repo"}}}}, nil
	}
	if _, err := n.MigrateIntelligenceMemoryToMaestro(context.Background(), "/workspace", true, "Migrate project habits now"); err == nil {
		t.Fatal("interrupted migration must fail")
	}
	result, err := n.RecoverMemoryMigration(context.Background(), "/workspace", true, "Recover interrupted memory migration")
	if err != nil {
		t.Fatalf("recovery failed: %v", err)
	}
	if !result.GateClosed || result.InstallCount != 1 || len(result.Projects) != 1 {
		t.Fatalf("unexpected recovery result: %+v", result)
	}
	for key, count := range seen {
		if count > 2 {
			t.Fatalf("idempotency key %s exported %d times", key, count)
		}
	}
	dataDir, _ := coreconfig.DataDir()
	inspection, err := memoryrecovery.Inspect(dataDir)
	if err != nil || inspection.Blocked {
		t.Fatalf("recovered journal must be clean: %+v %v", inspection, err)
	}
	if _, err := n.st.UpsertIntelligenceMemory(store.IntelligenceMemory{ProjectID: projectID, Key: "quality.testing", Value: "changed"}); !errors.Is(err, store.ErrLegacyMemoryWritesDisabled) {
		t.Fatalf("gate must be latched after recovery: %v", err)
	}
}

func TestRecoverMemoryMigrationRefusesWithoutInterruptedOperation(t *testing.T) {
	n, projectID := migrationTestNexus(t)
	addMigrationMemories(t, n.st, projectID)
	resetMigrationSeams(t, func(ctx context.Context, workspace string, request handoff.MaestroMemoryWriteRequest) (handoff.MaestroMemoryWriteResult, error) {
		return handoff.MaestroMemoryWriteResult{}, errors.New("bridge exploded")
	})
	// The very first export fails, so nothing reached the canonical store: the
	// journal aborts cleanly and no recovery operation exists.
	if _, err := n.MigrateIntelligenceMemoryToMaestro(context.Background(), "/workspace", true, "Migrate project habits now"); err == nil {
		t.Fatal("interrupted migration must fail")
	}
	if _, err := n.RecoverMemoryMigration(context.Background(), "/workspace", true, "Recover interrupted memory migration"); err == nil {
		t.Fatal("recovery must be refused without an interrupted operation")
	}
	if _, err := n.st.UpsertIntelligenceMemory(store.IntelligenceMemory{ProjectID: projectID, Key: "quality.extra", Value: "fresh fact"}); err != nil {
		t.Fatalf("clean abort must keep legacy writers functional: %v", err)
	}
}

func TestMigrateMemoryExportFailureLeavesNoDuplicatesAndNoGate(t *testing.T) {
	n, projectID := migrationTestNexus(t)
	addMigrationMemories(t, n.st, projectID)
	// INSTALL memory sorts before the project row, so a one-shot failure on
	// the second entry proves the partial-export path.
	callCount := 0
	resetMigrationSeams(t, func(ctx context.Context, workspace string, request handoff.MaestroMemoryWriteRequest) (handoff.MaestroMemoryWriteResult, error) {
		callCount++
		if callCount == 2 {
			return handoff.MaestroMemoryWriteResult{}, errors.New("bridge exploded")
		}
		result := recordedResult()
		result.Status = "REPLAYED"
		return result, nil
	})
	maestroMemoryReader = func(ctx context.Context, workspace, taskID, search string, limit int) (handoff.MaestroMemoryReadResult, error) {
		return handoff.MaestroMemoryReadResult{Items: []handoff.MaestroMemoryObservation{{ID: "obs_0123456789abcdef", Summary: search, Scope: handoff.MaestroMemoryScope{Level: "workspace", RepositoryID: "repo"}}}}, nil
	}
	if _, err := n.MigrateIntelligenceMemoryToMaestro(context.Background(), "/workspace", true, "Migrate project habits now"); err == nil {
		t.Fatal("export failure must fail the migration")
	}
	dataDir, _ := coreconfig.DataDir()
	gateFile := filepath.Join(dataDir, "memory-migration-state.json")
	if _, err := os.Stat(gateFile); !os.IsNotExist(err) {
		t.Fatal("failed migration must not latch the gate")
	}
	inspection, err := memoryrecovery.Inspect(dataDir)
	if err != nil || !inspection.Blocked {
		t.Fatalf("partial export must keep the journal for review: %+v %v", inspection, err)
	}
	// The retained journal lock fences legacy writers until an operator
	// reviews the partially applied operation.
	if _, err := n.st.UpsertIntelligenceMemory(store.IntelligenceMemory{ProjectID: projectID, Key: "quality.new", Value: "fresh fact"}); !errors.Is(err, memoryrecovery.ErrLegacyWriteFenced) {
		t.Fatalf("partial export must fence legacy writers: %v", err)
	}
	// A new migration is refused while the journal needs review.
	if _, err := n.MigrateIntelligenceMemoryToMaestro(context.Background(), "/workspace", true, "Retry migration after failure"); err == nil {
		t.Fatal("migration must be refused while the journal needs review")
	}
}
