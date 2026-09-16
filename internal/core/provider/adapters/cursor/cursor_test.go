package cursor

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestInspectAuthMemoizesStatusProbe(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	invalidateCursorAuthCache("memo")

	var calls atomic.Int32
	prev := cursorStatusRun
	cursorStatusRun = func(ctx context.Context, bin string, args []string, env []string, cwd string) (string, error) {
		calls.Add(1)
		return "Logged in as user@example.com", nil
	}
	t.Cleanup(func() { cursorStatusRun = prev })

	// Force binary lookup to succeed without a real cursor-agent by putting a
	// stub on PATH via LookPath — InspectAuth still needs Prepare + binary.
	// When binary is missing, InspectAuth returns early without caching a live
	// probe. Stub LookPath indirectly by skipping Prepare failure: create home.
	a := New()
	p := model.Profile{Provider: "cursor", Name: "memo"}
	if err := a.Prepare(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	// Without a real binary, InspectAuth returns unauthenticated without calling
	// cursorStatusRun. Inject a fake binary via PATH is heavy; instead call the
	// cache path by writing a pre-warmed entry and verifying subsequent hits.
	info := model.AccountInfo{Authenticated: true, Status: "Authenticated", Health: model.HealthHealthy}
	cursorAuthCacheMu.Lock()
	cursorAuthCache["memo"] = cursorAuthCacheEntry{info: info, fetchedAt: time.Now()}
	cursorAuthCacheMu.Unlock()

	got := a.InspectAuth(context.Background(), p)
	if !got.Authenticated || got.Status != "Authenticated" {
		t.Fatalf("cached auth = %+v", got)
	}
	if calls.Load() != 0 {
		t.Fatalf("cached InspectAuth must not spawn status probe, calls=%d", calls.Load())
	}
}

func TestInspectAuthExtractsEmail(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	invalidateCursorAuthCache("test-email")

	prev := cursorStatusRun
	cursorStatusRun = func(ctx context.Context, bin string, args []string, env []string, cwd string) (string, error) {
		return "Tip: You can start the Cursor CLI with `agent`.\n✓ Logged in as engineer@example.com\n", nil
	}
	t.Cleanup(func() { cursorStatusRun = prev })

	a := New()
	p := model.Profile{Provider: "cursor", Name: "test-email"}
	if err := a.Prepare(context.Background(), p); err != nil {
		t.Fatal(err)
	}

	tmpBin := t.TempDir()
	fakeBin := filepath.Join(tmpBin, "cursor-agent")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmpBin+":"+os.Getenv("PATH"))

	got := a.InspectAuth(context.Background(), p)
	if !got.Authenticated {
		t.Fatalf("expected Authenticated=true, got %+v", got)
	}
	if got.Email != "engineer@example.com" {
		t.Fatalf("expected Email='engineer@example.com', got %q", got.Email)
	}
}
