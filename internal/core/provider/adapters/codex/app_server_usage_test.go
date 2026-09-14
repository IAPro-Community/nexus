package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// TestMain keeps unit tests hermetic: the real Codex app-server must never be
// spawned from a test process. Protocol tests install a fake via fakeAppServer.
func TestMain(m *testing.M) {
	if mode := os.Getenv("NEXUS_TEST_APP_SERVER_MODE"); mode != "" {
		runFakeAppServer(mode)
		return
	}
	appServerCommand = func(ctx context.Context, codexHome string) (*exec.Cmd, error) {
		return nil, errors.New("app-server disabled in tests")
	}
	os.Exit(m.Run())
}

// fakeAppServer routes the probe to this test binary running in helper mode.
func fakeAppServer(t *testing.T, mode string) {
	t.Helper()
	prev := appServerCommand
	appServerCommand = func(ctx context.Context, codexHome string) (*exec.Cmd, error) {
		cmd := exec.CommandContext(ctx, os.Args[0])
		cmd.Env = append(os.Environ(),
			"NEXUS_TEST_APP_SERVER_MODE="+mode,
			"NEXUS_TEST_APP_SERVER_HOME="+codexHome,
		)
		return cmd, nil
	}
	resetAppServerProbeCache()
	t.Cleanup(func() {
		appServerCommand = prev
		resetAppServerProbeCache()
	})
}

func codexTestEnv(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	return dataDir
}

const (
	fakeAccountID = "9cba8371-db95-468c-ad00-02e9ceddcb3e"
	fakeEmail     = "kiver.omegaedu@gmail.com"
)

func gmailProfile(t *testing.T, dataDir string) model.Profile {
	t.Helper()
	home := filepath.Join(dataDir, "profiles", "codex", "kivergmail", "home")
	writeAuth(t, filepath.Join(home, "auth.json"), fakeAccountID, fakeEmail)
	return model.Profile{Provider: "codex", Name: "kivergmail"}
}

func TestAppServerReturnsOfficialLiveQuota(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "ok")

	snap, ok := New().fetchUsageFromAppServer(context.Background(), p)
	if !ok {
		t.Fatal("expected official snapshot")
	}
	if snap.Source != model.SourceOfficialAPI {
		t.Fatalf("source=%s want OFFICIAL_API", snap.Source)
	}
	if snap.Status != model.UsageLive {
		t.Fatalf("status=%s want LIVE", snap.Status)
	}
	if len(snap.Windows) != 2 {
		t.Fatalf("windows=%d want 2", len(snap.Windows))
	}
	if snap.Windows[0].Kind != "5h" || *snap.Windows[0].RemainingPercent != 0 {
		t.Fatalf("5h=%s/%v want 5h/0", snap.Windows[0].Kind, *snap.Windows[0].RemainingPercent)
	}
	if snap.Windows[1].Kind != "weekly" || *snap.Windows[1].RemainingPercent != 56 {
		t.Fatalf("weekly=%s/%v want weekly/56", snap.Windows[1].Kind, *snap.Windows[1].RemainingPercent)
	}
	if snap.Account != fakeAccountID {
		t.Fatalf("account=%q want %q", snap.Account, fakeAccountID)
	}
	if snap.Plan != "ChatGPT Plus" {
		t.Fatalf("plan=%q", snap.Plan)
	}
	if time.Since(snap.FetchedAt) > time.Minute {
		t.Fatalf("FetchedAt=%v should be now", snap.FetchedAt)
	}
}

func TestAppServerSkipsNotificationsAndOutOfOrderIDs(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "noisy")

	snap, ok := New().fetchUsageFromAppServer(context.Background(), p)
	if !ok {
		t.Fatal("expected official snapshot despite notification noise")
	}
	if *snap.Windows[0].RemainingPercent != 0 {
		t.Fatalf("remaining=%v", *snap.Windows[0].RemainingPercent)
	}
}

func TestAppServerRejectsMissingAccountID(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "missing-account")

	if _, ok := New().fetchUsageFromAppServer(context.Background(), p); ok {
		t.Fatal("response without accountId must be rejected")
	}
}

func TestAppServerRejectsMissingCodexHome(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "missing-home")

	if _, ok := New().fetchUsageFromAppServer(context.Background(), p); ok {
		t.Fatal("initialize without codexHome must be rejected")
	}
}

func TestAppServerRejectsForeignAccount(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "wrong-account")

	if _, ok := New().fetchUsageFromAppServer(context.Background(), p); ok {
		t.Fatal("snapshot from a different accountId must be rejected")
	}
}

func TestAppServerRejectsForeignCodexHome(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "wrong-home")

	if _, ok := New().fetchUsageFromAppServer(context.Background(), p); ok {
		t.Fatal("server bound to another codex home must be rejected")
	}
}

func TestAppServerHandlesRPCErrorAndGarbage(t *testing.T) {
	for _, mode := range []string{"rpc-error", "garbage", "empty-result", "no-primary"} {
		t.Run(mode, func(t *testing.T) {
			dataDir := codexTestEnv(t)
			p := gmailProfile(t, dataDir)
			fakeAppServer(t, mode)
			if _, ok := New().fetchUsageFromAppServer(context.Background(), p); ok {
				t.Fatalf("mode %s must not produce a snapshot", mode)
			}
		})
	}
}

func TestAppServerBackendBlockWinsOverPercentages(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "blocked")

	snap, ok := New().fetchUsageFromAppServer(context.Background(), p)
	if !ok {
		t.Fatal("expected snapshot")
	}
	if snap.Status != model.UsageRateLimited {
		t.Fatalf("status=%s want RATE_LIMITED when backend blocks ordinary usage", snap.Status)
	}
	if snap.Error == "" {
		t.Fatal("expected a reason for the block")
	}
	// Percentages look healthy; the backend signal must still win.
	if *snap.Windows[0].RemainingPercent != 90 {
		t.Fatalf("remaining=%v want the reported 90", *snap.Windows[0].RemainingPercent)
	}
}

func TestAppServerPrefersCodexBucket(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "multi-bucket")

	snap, ok := New().fetchUsageFromAppServer(context.Background(), p)
	if !ok {
		t.Fatal("expected snapshot")
	}
	if *snap.Windows[0].RemainingPercent != 70 {
		t.Fatalf("remaining=%v want 70 from the codex bucket, not the legacy view", *snap.Windows[0].RemainingPercent)
	}
}

func TestAppServerTimeoutDoesNotHang(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "hang")

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, ok := New().fetchUsageFromAppServer(ctx, p); ok {
		t.Fatal("hanging server must not produce a snapshot")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("probe took %v; caller context must bound it", elapsed)
	}
}

func TestAppServerProbeCoalescesConcurrentReaders(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)

	var mu sync.Mutex
	spawned := 0
	prev := appServerCommand
	appServerCommand = func(ctx context.Context, codexHome string) (*exec.Cmd, error) {
		mu.Lock()
		spawned++
		mu.Unlock()
		cmd := exec.CommandContext(ctx, os.Args[0])
		cmd.Env = append(os.Environ(),
			"NEXUS_TEST_APP_SERVER_MODE=slow-ok",
			"NEXUS_TEST_APP_SERVER_HOME="+codexHome,
		)
		return cmd, nil
	}
	resetAppServerProbeCache()
	t.Cleanup(func() {
		appServerCommand = prev
		resetAppServerProbeCache()
	})

	adapter := New()
	var wg sync.WaitGroup
	results := make([]bool, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, ok := adapter.appServerUsage(context.Background(), p, false)
			results[idx] = ok
		}(i)
	}
	wg.Wait()

	for i, ok := range results {
		if !ok {
			t.Fatalf("reader %d got no snapshot", i)
		}
	}
	mu.Lock()
	got := spawned
	mu.Unlock()
	if got != 1 {
		t.Fatalf("spawned %d app-servers; concurrent readers must coalesce onto one", got)
	}
}

func TestAppServerProbeIsolatesProfiles(t *testing.T) {
	dataDir := codexTestEnv(t)
	gmail := gmailProfile(t, dataDir)
	omegaHome := filepath.Join(dataDir, "profiles", "codex", "omega", "home")
	writeAuth(t, filepath.Join(omegaHome, "auth.json"), "802cf130-605a-4489-94f4-a6570dbdf8e7", "kiver@omegasistemas.net.br")
	omega := model.Profile{Provider: "codex", Name: "omega"}

	fakeAppServer(t, "per-home")
	adapter := New()

	g, ok := adapter.appServerUsage(context.Background(), gmail, false)
	if !ok {
		t.Fatal("gmail probe failed")
	}
	o, ok := adapter.appServerUsage(context.Background(), omega, false)
	if !ok {
		t.Fatal("omega probe failed")
	}
	if g.Account == o.Account {
		t.Fatalf("both profiles reported %q; probes must not share cache", g.Account)
	}
	if *g.Windows[0].RemainingPercent == *o.Windows[0].RemainingPercent {
		t.Fatal("profiles must not share quota values")
	}
}

func TestAppServerDisabledByKillSwitch(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "ok")
	t.Setenv("NEXUS_CODEX_APP_SERVER", "0")

	if _, ok := New().fetchUsageFromAppServer(context.Background(), p); ok {
		t.Fatal("kill switch must disable the official probe")
	}
}

func TestAppServerRequiresVerifiableAccountID(t *testing.T) {
	dataDir := codexTestEnv(t)
	// Auth without chatgpt_account_id: the response could not be verified, so the
	// probe must not run at all.
	home := filepath.Join(dataDir, "profiles", "codex", "legacy", "home")
	writeAuth(t, filepath.Join(home, "auth.json"), "", "legacy@example.com")

	spawned := false
	prev := appServerCommand
	appServerCommand = func(ctx context.Context, codexHome string) (*exec.Cmd, error) {
		spawned = true
		return nil, errors.New("must not be called")
	}
	resetAppServerProbeCache()
	t.Cleanup(func() {
		appServerCommand = prev
		resetAppServerProbeCache()
	})

	if _, ok := New().fetchUsageFromAppServer(context.Background(), model.Profile{Provider: "codex", Name: "legacy"}); ok {
		t.Fatal("must not produce a snapshot without a verifiable account id")
	}
	if spawned {
		t.Fatal("must not spawn an app-server for an unverifiable identity")
	}
}

func TestAppServerFallsBackToRolloutWhenUnavailable(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "rpc-error")

	home := filepath.Join(dataDir, "profiles", "codex", "kivergmail", "home")
	now := time.Now()
	writeRollout(t, filepath.Join(home, "sessions", "rollout-fallback.jsonl"), 25, 41, now.Add(time.Hour).Unix(), now, "")

	snap := New().GetUsage(context.Background(), p)
	if snap.Source != model.SourceObservation {
		t.Fatalf("source=%s want OBSERVATION fallback", snap.Source)
	}
	if *snap.Windows[0].RemainingPercent != 75 {
		t.Fatalf("remaining=%v want 75 from rollout", *snap.Windows[0].RemainingPercent)
	}
}

func TestAppServerWinsOverRollout(t *testing.T) {
	dataDir := codexTestEnv(t)
	p := gmailProfile(t, dataDir)
	fakeAppServer(t, "ok")

	home := filepath.Join(dataDir, "profiles", "codex", "kivergmail", "home")
	now := time.Now()
	// Stale-but-recent rollout claiming plenty of quota.
	writeRollout(t, filepath.Join(home, "sessions", "rollout-stale.jsonl"), 10, 10, now.Add(time.Hour).Unix(), now, "")

	snap := New().GetUsage(context.Background(), p)
	if snap.Source != model.SourceOfficialAPI {
		t.Fatalf("source=%s want OFFICIAL_API to win over rollout", snap.Source)
	}
	if *snap.Windows[0].RemainingPercent != 0 {
		t.Fatalf("remaining=%v want the official 0", *snap.Windows[0].RemainingPercent)
	}
}

// --- fake server ---

func runFakeAppServer(mode string) {
	if mode == "hang" {
		time.Sleep(30 * time.Second)
		return
	}
	if mode == "slow-ok" {
		time.Sleep(300 * time.Millisecond)
	}
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	emit := func(v any) {
		b, _ := json.Marshal(v)
		out.Write(b)
		out.WriteByte('\n')
		out.Flush()
	}

	codexHome := os.Getenv("NEXUS_TEST_APP_SERVER_HOME")
	reportedHome := codexHome
	if mode == "wrong-home" {
		reportedHome = filepath.Join(os.TempDir(), "someone-elses-codex-home")
	}
	if mode == "missing-home" {
		reportedHome = ""
	}

	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var req struct {
			ID     *int64 `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal(sc.Bytes(), &req) != nil {
			continue
		}
		switch req.Method {
		case "initialize":
			if mode == "garbage" {
				out.WriteString("this is not json\n")
				out.Flush()
				return
			}
			initResult := map[string]any{
				"platformFamily": "unix",
			}
			if reportedHome != "" {
				initResult["codexHome"] = reportedHome
			}
			emit(map[string]any{"id": req.ID, "result": initResult})
		case "initialized":
			// no response by contract
		case appServerRateLimitsMethod:
			switch mode {
			case "rpc-error":
				emit(map[string]any{"id": req.ID, "error": map[string]any{"code": -32601, "message": "method not found"}})
			case "empty-result":
				emit(map[string]any{"id": req.ID})
			case "noisy":
				emit(map[string]any{"method": "account/rateLimits/updated", "params": map[string]any{"rateLimits": map[string]any{}}})
				emit(map[string]any{"method": "remoteControl/status/changed"})
				emit(map[string]any{"id": 99, "result": map[string]any{"unrelated": true}})
				emit(map[string]any{"id": req.ID, "result": rateLimitsPayload(fakeAccountID, 100, 44, true, nil)})
			case "wrong-account":
				emit(map[string]any{"id": req.ID, "result": rateLimitsPayload("00000000-dead-beef-0000-000000000000", 100, 44, true, nil)})
			case "missing-account":
				payload := rateLimitsPayload(fakeAccountID, 100, 44, true, nil)
				delete(payload, "accountId")
				emit(map[string]any{"id": req.ID, "result": payload})
			case "no-primary":
				emit(map[string]any{"id": req.ID, "result": map[string]any{
					"accountId":  fakeAccountID,
					"rateLimits": map[string]any{"limitId": "codex"},
				}})
			case "blocked":
				allowed := false
				emit(map[string]any{"id": req.ID, "result": rateLimitsPayload(fakeAccountID, 10, 5, true, &allowed)})
			case "multi-bucket":
				payload := rateLimitsPayload(fakeAccountID, 95, 44, true, nil)
				payload["rateLimitsByLimitId"] = map[string]any{
					"codex": bucketPayload(30, 44),
				}
				emit(map[string]any{"id": req.ID, "result": payload})
			case "per-home":
				if filepath.Base(filepath.Dir(codexHome)) == "omega" {
					emit(map[string]any{"id": req.ID, "result": rateLimitsPayload("802cf130-605a-4489-94f4-a6570dbdf8e7", 25, 41, true, nil)})
				} else {
					emit(map[string]any{"id": req.ID, "result": rateLimitsPayload(fakeAccountID, 100, 44, true, nil)})
				}
			default:
				emit(map[string]any{"id": req.ID, "result": rateLimitsPayload(fakeAccountID, 100, 44, true, nil)})
			}
			return
		}
	}
}

func bucketPayload(used5h, usedWeekly float64) map[string]any {
	now := time.Now()
	return map[string]any{
		"limitId":              "codex",
		"planType":             "plus",
		"spendControlReached":  false,
		"rateLimitReachedType": nil,
		"primary": map[string]any{
			"usedPercent":        used5h,
			"windowDurationMins": 300,
			"resetsAt":           now.Add(time.Hour).Unix(),
		},
		"secondary": map[string]any{
			"usedPercent":        usedWeekly,
			"windowDurationMins": 10080,
			"resetsAt":           now.Add(72 * time.Hour).Unix(),
		},
	}
}

func rateLimitsPayload(accountID string, used5h, usedWeekly float64, withBucket bool, ordinaryAllowed *bool) map[string]any {
	payload := map[string]any{
		"accountId":  accountID,
		"rateLimits": bucketPayload(used5h, usedWeekly),
	}
	if withBucket {
		payload["rateLimitsByLimitId"] = map[string]any{"codex": bucketPayload(used5h, usedWeekly)}
	}
	if ordinaryAllowed != nil {
		payload["ordinaryUsageAllowed"] = *ordinaryAllowed
	}
	return payload
}
