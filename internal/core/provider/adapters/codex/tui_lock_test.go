package codex

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestTryAcquireTUILockNonBlocking(t *testing.T) {
	home := t.TempDir()
	held, err := AcquireTUILock(home)
	if err != nil {
		t.Fatalf("AcquireTUILock: %v", err)
	}
	defer func() { _ = held.Release() }()

	if !IsTUILocked(home) {
		t.Fatal("expected IsTUILocked=true while lock is held")
	}

	lock, ok, err := TryAcquireTUILock(home)
	if err != nil {
		t.Fatalf("TryAcquireTUILock: %v", err)
	}
	if ok || lock != nil {
		t.Fatal("expected non-blocking acquire to fail while TUI holds the lock")
	}
}

func TestTUILeasesShareAndMaintenanceIsExclusive(t *testing.T) {
	home := t.TempDir()
	first, err := AcquireTUILease(home)
	if err != nil {
		t.Fatalf("AcquireTUILease first: %v", err)
	}
	defer func() { _ = first.Release() }()
	second, err := AcquireTUILease(home)
	if err != nil {
		t.Fatalf("second TUI lease must coexist: %v", err)
	}
	defer func() { _ = second.Release() }()
	maintenance, ok, err := TryAcquireMaintenanceLock(home)
	if err != nil {
		t.Fatalf("TryAcquireMaintenanceLock: %v", err)
	}
	if ok || maintenance != nil {
		t.Fatal("maintenance lock must be blocked while any TUI lease is active")
	}
}

func TestMaintenanceLockBlocksNewTUILeases(t *testing.T) {
	home := t.TempDir()
	maintenance, err := AcquireMaintenanceLock(home)
	if err != nil {
		t.Fatalf("AcquireMaintenanceLock: %v", err)
	}
	defer func() { _ = maintenance.Release() }()
	lease, ok, err := TryAcquireTUILease(home)
	if err != nil {
		t.Fatalf("TryAcquireTUILease: %v", err)
	}
	if ok || lease != nil {
		t.Fatal("new TUI lease must be blocked by maintenance")
	}
}

func TestRunAppServerRateLimitsRespectsTUILock(t *testing.T) {
	home := t.TempDir()
	lock, err := AcquireTUILock(home)
	if err != nil {
		t.Fatalf("AcquireTUILock: %v", err)
	}
	defer func() { _ = lock.Release() }()

	var spawned atomic.Int32
	prev := appServerCommand
	t.Cleanup(func() { appServerCommand = prev })
	appServerCommand = func(ctx context.Context, codexHome string) (*exec.Cmd, error) {
		spawned.Add(1)
		return prev(ctx, codexHome)
	}

	_, err = runAppServerRateLimits(context.Background(), home)
	if err == nil || !strings.Contains(err.Error(), "TUI") {
		t.Fatalf("expected TUI busy error, got %v", err)
	}
	if spawned.Load() != 0 {
		t.Fatalf("must not spawn app-server under TUI lock; spawned=%d", spawned.Load())
	}
}

func TestGetUsageSkipsMigrationAndAppServerWhileTUILocked(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_DATA_DIR", data)
	t.Setenv("NEXUS_CODEX_APP_SERVER", "1")

	profileName := "busy"
	home := filepath.Join(data, "profiles", "codex", profileName, "home")
	if err := os.MkdirAll(filepath.Join(home, "sessions"), 0700); err != nil {
		t.Fatal(err)
	}
	writeAuth(t, filepath.Join(home, "auth.json"), "acct-busy", "busy@example.test")

	hostSessions := filepath.Join(t.TempDir(), "host-sessions")
	if err := os.MkdirAll(hostSessions, 0700); err != nil {
		t.Fatal(err)
	}
	threadPath := filepath.Join(home, "thread_history_1.sqlite")
	if err := os.Symlink(filepath.Join(hostSessions, "thread_history_1.sqlite"), threadPath); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	writeRollout(t, filepath.Join(home, "sessions", "rollout-busy.jsonl"), 12, 40, now.Add(time.Hour).Unix(), now, "")

	lock, err := AcquireTUILock(home)
	if err != nil {
		t.Fatalf("AcquireTUILock: %v", err)
	}
	defer func() { _ = lock.Release() }()

	var spawned atomic.Int32
	prev := appServerCommand
	t.Cleanup(func() { appServerCommand = prev })
	appServerCommand = func(ctx context.Context, codexHome string) (*exec.Cmd, error) {
		spawned.Add(1)
		return prev(ctx, codexHome)
	}

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "codex", Name: profileName})
	if spawned.Load() != 0 {
		t.Fatalf("app-server must not spawn while TUI lock is held; spawned=%d", spawned.Load())
	}
	if len(snap.Windows) == 0 {
		t.Fatalf("expected rollout windows while TUI locked, got %+v", snap)
	}
	if fi, err := os.Lstat(threadPath); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("GetUsage must not replace thread_history symlink while TUI is locked: err=%v mode=%v", err, fi)
	}
}

func TestPrepareUsesCanonicalHomeOnly(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_DATA_DIR", data)
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	if err := New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "canon"}); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(data, "profiles", "codex", "canon", "home")
	if _, err := os.Stat(filepath.Join(home, "sessions")); err != nil {
		t.Fatalf("canonical sessions missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "config.toml")); err != nil {
		t.Fatalf("canonical config missing: %v", err)
	}
	if fi, err := os.Stat(filepath.Join(home, ".codex", "sessions")); err == nil && fi.IsDir() {
		t.Fatal("Prepare must not create a second .codex/sessions tree")
	}
}

func TestPrepareMigratesLegacyConfigBeforeCreatingCanonicalConfig(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_DATA_DIR", data)
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	home := filepath.Join(data, "profiles", "codex", "legacy-config", "home")
	legacyConfig := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(legacyConfig), 0700); err != nil {
		t.Fatal(err)
	}
	const contents = "model = \"legacy-model\"\n"
	if err := os.WriteFile(legacyConfig, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}

	if err := New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "legacy-config"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatalf("read canonical config: %v", err)
	}
	if !strings.Contains(string(got), contents) {
		t.Fatalf("canonical config = %q, want migrated legacy contents containing %q", got, contents)
	}
}

func TestAcquireTUILockTimeoutFailsFastWhenHeld(t *testing.T) {
	home := t.TempDir()
	held, err := AcquireTUILock(home)
	if err != nil {
		t.Fatalf("AcquireTUILock: %v", err)
	}
	defer func() { _ = held.Release() }()

	start := time.Now()
	lock, err := AcquireTUILockTimeout(home, 80*time.Millisecond)
	if !errors.Is(err, ErrTUIBusy) {
		t.Fatalf("err=%v lock=%v want ErrTUIBusy", err, lock)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("timeout waited %s, want fail-fast", time.Since(start))
	}
}

func TestPrepareFailsFastWhenTUILocked(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_DATA_DIR", data)
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	home := filepath.Join(data, "profiles", "codex", "locked", "home")
	held, err := AcquireTUILock(home)
	if err != nil {
		t.Fatalf("AcquireTUILock: %v", err)
	}
	defer func() { _ = held.Release() }()

	start := time.Now()
	err = New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "locked"})
	if !errors.Is(err, ErrTUIBusy) {
		t.Fatalf("Prepare err=%v want ErrTUIBusy", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("Prepare blocked for %s", time.Since(start))
	}
}

func TestPrepareReusesReadyProfileWhileTUILeaseIsActive(t *testing.T) {
	data := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_MANAGER_DATA_DIR", data)
	t.Setenv("NEXUS_DATA_DIR", data)
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)
	home := filepath.Join(data, "profiles", "codex", "ready", "home")
	if err := os.MkdirAll(filepath.Join(home, "sessions"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("model = \"ready\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	lease, err := AcquireTUILease(home)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lease.Release() }()
	if err := New().Prepare(context.Background(), model.Profile{Provider: "codex", Name: "ready"}); err != nil {
		t.Fatalf("ready profile should be reusable under a TUI lease: %v", err)
	}
}

func TestEnvCODEXHome(t *testing.T) {
	if got := EnvCODEXHome([]string{"PATH=/bin", "CODEX_HOME=/profiles/codex/work/home", "HOME=/profiles/codex/work/home"}); got != "/profiles/codex/work/home" {
		t.Fatalf("EnvCODEXHome=%q", got)
	}
	if got := EnvCODEXHome([]string{"HOME=/tmp"}); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}
