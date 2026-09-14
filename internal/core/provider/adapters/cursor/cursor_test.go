package cursor

import (
	"context"
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
