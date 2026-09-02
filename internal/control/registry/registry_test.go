package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDefaultRegistryResetConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("AI_MANAGER_DATA_DIR", tmpDir)
	t.Setenv("AI_CLI_DATA_DIR", tmpDir)

	ResetDefaultRegistryForTest()
	t.Cleanup(ResetDefaultRegistryForTest)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			_ = DefaultRegistry()
			runtime.Gosched()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			ResetDefaultRegistryForTest()
			runtime.Gosched()
		}
	}()

	wg.Wait()

	if DefaultRegistry() == nil {
		t.Fatal("expected DefaultRegistry to return a singleton instance")
	}
}

func TestRegistryPersistenceAndLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "runtimes.json")

	reg := NewRegistry(dbPath)

	sess := RuntimeSession{
		RuntimeID:    "rt-001",
		ProviderID:   "codex",
		ProfileID:    "work",
		Workspace:    "/projects/omega",
		PID:          os.Getpid(),
		State:        StateRunning,
		ControlLevel: ControlLevelAPI,
		StartedAt:    time.Now(),
	}

	if err := reg.Register(sess); err != nil {
		t.Fatalf("failed to register session: %v", err)
	}

	// Verify in-memory retrieval
	got, ok := reg.Get("rt-001")
	if !ok || got.ProviderID != "codex" || got.State != StateRunning {
		t.Fatalf("unexpected retrieved session: %+v", got)
	}

	// Verify persistence reload
	reg2 := NewRegistry(dbPath)
	got2, ok2 := reg2.Get("rt-001")
	if !ok2 || got2.ProfileID != "work" {
		t.Fatalf("failed to reload persisted session: %+v", got2)
	}

	// Update state
	if err := reg2.UpdateState("rt-001", StateStopped); err != nil {
		t.Fatalf("failed to update state: %v", err)
	}
	if got3, _ := reg2.Get("rt-001"); got3.State != StateStopped {
		t.Errorf("expected state STOPPED, got %s", got3.State)
	}

	// Active list check
	if len(reg2.ListActive()) != 0 {
		t.Errorf("expected 0 active sessions after stopping")
	}

	// Delete
	if err := reg2.Delete("rt-001"); err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}
	if _, ok := reg2.Get("rt-001"); ok {
		t.Errorf("expected session to be deleted")
	}
}

func TestCleanupStale(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "runtimes.json")
	reg := NewRegistry(dbPath)

	// Register a session with a non-existent PID
	fakePID := 99999999
	_ = reg.Register(RuntimeSession{
		RuntimeID: "rt-dead",
		PID:       fakePID,
		State:     StateRunning,
	})

	cleaned, err := reg.CleanupStale()
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if cleaned != 1 {
		t.Errorf("expected 1 cleaned session, got %d", cleaned)
	}

	s, _ := reg.Get("rt-dead")
	if s.State != StateStale {
		t.Errorf("expected state STALE, got %s", s.State)
	}
}

func TestPurgeInactive(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "runtimes.json")
	reg := NewRegistry(dbPath)

	fakePID := 99999999
	_ = reg.Register(RuntimeSession{
		RuntimeID: "rt-stale-1",
		PID:       fakePID,
		State:     StateStale,
	})
	_ = reg.Register(RuntimeSession{
		RuntimeID: "rt-stopped-2",
		PID:       fakePID,
		State:     StateStopped,
	})

	purged, err := reg.PurgeInactive()
	if err != nil {
		t.Fatalf("purge failed: %v", err)
	}
	if purged != 2 {
		t.Errorf("expected 2 purged sessions, got %d", purged)
	}

	if len(reg.List()) != 0 {
		t.Errorf("expected 0 sessions remaining, got %d", len(reg.List()))
	}
}

func TestRegistry_ConcurrentMultiProcessNoLostUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "runtimes.json")

	// Instance A and Instance B represent two separate processes accessing the same runtimes.json
	regA := NewRegistry(dbPath)
	regB := NewRegistry(dbPath)

	const count = 20
	errChan := make(chan error, count*2)

	// Reg A registers rt-a-*
	go func() {
		for i := 0; i < count; i++ {
			err := regA.Register(RuntimeSession{
				RuntimeID: "rt-a-" + string(rune('a'+i)),
				PID:       os.Getpid(),
				State:     StateRunning,
			})
			errChan <- err
		}
	}()

	// Reg B registers rt-b-*
	go func() {
		for i := 0; i < count; i++ {
			err := regB.Register(RuntimeSession{
				RuntimeID: "rt-b-" + string(rune('a'+i)),
				PID:       os.Getpid(),
				State:     StateRunning,
			})
			errChan <- err
		}
	}()

	for i := 0; i < count*2; i++ {
		if err := <-errChan; err != nil {
			t.Fatalf("concurrent registration failed: %v", err)
		}
	}

	// Reg C reads directly from disk
	regC := NewRegistry(dbPath)
	listC := regC.List()
	if len(listC) != count*2 {
		t.Fatalf("lost-update detected: expected %d sessions on disk, got %d", count*2, len(listC))
	}

	// Reg A and Reg B should also see all sessions
	if len(regA.List()) != count*2 {
		t.Errorf("regA expected %d sessions, got %d", count*2, len(regA.List()))
	}
	if len(regB.List()) != count*2 {
		t.Errorf("regB expected %d sessions, got %d", count*2, len(regB.List()))
	}
}

func TestRuntimeSession_NoEnvOrSecretPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "runtimes.json")

	reg := NewRegistry(dbPath)

	sess := RuntimeSession{
		RuntimeID:    "rt-secret-1",
		ProviderID:   "codex",
		ProfileID:    "work",
		Workspace:    "/projects/omega",
		PID:          os.Getpid(),
		State:        StateRunning,
		ControlLevel: ControlLevelAPI,
		Binary:       "/usr/bin/secret-exec",
		Args:         []string{"--token", "sk-secret-token-123"},
		Env:          []string{"AWS_SECRET_ACCESS_KEY=mysecretkey", "OPENAI_API_KEY=sk-123456789"},
		StartedAt:    time.Now(),
	}

	if err := reg.Register(sess); err != nil {
		t.Fatalf("failed to register session: %v", err)
	}

	// 1. Check raw JSON marshaling of RuntimeSession struct
	rawJSON, err := json.Marshal(sess)
	if err != nil {
		t.Fatalf("failed to marshal session: %v", err)
	}
	jsonStr := string(rawJSON)
	if strings.Contains(jsonStr, "mysecretkey") || strings.Contains(jsonStr, "sk-secret-token") || strings.Contains(jsonStr, "secret-exec") {
		t.Fatalf("RuntimeSession JSON leaked sensitive fields: %s", jsonStr)
	}
	if strings.Contains(jsonStr, `"env"`) || strings.Contains(jsonStr, `"binary"`) || strings.Contains(jsonStr, `"args"`) {
		t.Fatalf("RuntimeSession JSON unexpectedly contains env/binary/args keys: %s", jsonStr)
	}

	// 2. Check runtimes.json file on disk
	diskData, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("failed to read runtimes.json: %v", err)
	}
	diskStr := string(diskData)
	if strings.Contains(diskStr, "mysecretkey") || strings.Contains(diskStr, "sk-secret-token") || strings.Contains(diskStr, "secret-exec") {
		t.Fatalf("runtimes.json leaked sensitive data to disk: %s", diskStr)
	}
	if strings.Contains(diskStr, `"env"`) || strings.Contains(diskStr, `"binary"`) || strings.Contains(diskStr, `"args"`) {
		t.Fatalf("runtimes.json contains serialized env/binary/args: %s", diskStr)
	}
}

func TestRuntimeSessionModelPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "runtimes.json")
	reg := NewRegistry(dbPath)

	sess := RuntimeSession{
		RuntimeID:         "rt-model-test",
		ProviderID:        "agy",
		ProfileID:         "kiver",
		ProviderSessionID: "sess-abc",
		Model:             "claude-sonnet-4-20250514",
		Workspace:         "/projects/alpha",
		PID:               os.Getpid(),
		State:             StateRunning,
		ControlLevel:      ControlLevelTerminal,
		StartedAt:         time.Now(),
	}

	if err := reg.Register(sess); err != nil {
		t.Fatalf("failed to register session with model: %v", err)
	}

	got, ok := reg.Get("rt-model-test")
	if !ok {
		t.Fatal("session not found in memory")
	}
	if got.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected Model 'claude-sonnet-4-20250514', got %q", got.Model)
	}

	// Reload from disk to verify JSON serialization
	reg2 := NewRegistry(dbPath)
	got2, ok2 := reg2.Get("rt-model-test")
	if !ok2 {
		t.Fatal("session not found after reloading from disk")
	}
	if got2.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected reloaded Model 'claude-sonnet-4-20250514', got %q", got2.Model)
	}
}
