package agy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// agyIdentityFixture builds an isolated AGY profile with a valid,
// non-expired OAuth token and (optionally) an active Google account. It
// returns the profile, the account scope a refresh would carry, and the
// profile root — where SaveUsageForScope persists usage.json.
func agyIdentityFixture(t *testing.T, name, activeEmail string) (model.Profile, model.AccountScope, string, string) {
	t.Helper()

	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)

	root := filepath.Join(dataDir, "profiles", "agy", name)
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(home, ".gemini"), 0700); err != nil {
		t.Fatal(err)
	}
	if activeEmail != "" {
		if err := os.WriteFile(
			filepath.Join(home, ".gemini", "google_accounts.json"),
			[]byte(fmt.Sprintf(`{"active":%q}`, activeEmail)), 0600,
		); err != nil {
			t.Fatal(err)
		}
	}

	// A valid, non-expired OAuth token is the only authentication evidence the
	// live probe accepts (google_accounts.json alone fails closed).
	token := fmt.Sprintf(`{"token":{"access_token":"at-test","refresh_token":"rt-test","expiry":%q}}`,
		time.Now().Add(2*time.Hour).UTC().Format(time.RFC3339Nano))
	if err := os.WriteFile(filepath.Join(home, ".gemini", "antigravity-oauth-token"), []byte(token), 0600); err != nil {
		t.Fatal(err)
	}

	scope := model.AccountScope{
		ProviderID:      "agy",
		ProfileID:       name,
		AccountID:       "scope-account-id-" + name,
		IdentityVersion: "1",
	}
	return model.Profile{Provider: "agy", Name: name, AccountScope: scope}, scope, root, home
}

// installFakeAgyCLI puts a fake `agy` binary at the front of PATH. The script
// runs with HOME pointed at the profile home, exactly like the real probe.
// It must only print quota lines and/or rewrite identity artifacts — never a
// usage.json, so cache-file assertions below only observe adapter persistence.
func installFakeAgyCLI(t *testing.T, script string) {
	t.Helper()
	fakeBinDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(fakeBinDir, "agy"), []byte("#!/usr/bin/env bash\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// quotaOutput is a minimal valid AGY quota response (a single family is
// accepted by agyQuotaComplete).
func quotaOutput(remaining float64) string {
	return fmt.Sprintf("printf 'gemini\\t5h\\t%g%%%%\\t2h\\n'\n", remaining)
}

func assertNoPersistedUsage(t *testing.T, root string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "usage.json"))
	if err == nil {
		t.Fatalf("nothing may be persisted for a refused refresh, got: %s", data)
	}
	if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

// TestRefreshUsageRefusesMidFlightAccountSwitch covers the F2.8 adapter-side
// gate (issue #228): the Google account switches while the live probe is in
// flight, so the probe answers with account B's quota while the profile scope
// still belongs to account A. The B snapshot must still be returned to the
// caller (the profile layer discards it), but it must never be persisted into
// this profile's cache file.
func TestRefreshUsageRefusesMidFlightAccountSwitch(t *testing.T) {
	p, _, root, _ := agyIdentityFixture(t, "midflight", "a@example.test")
	installFakeAgyCLI(t, quotaOutput(77)+
		`printf '{"active":"b@example.test"}' > "$HOME/.gemini/google_accounts.json"
`)

	snap := New().RefreshUsage(context.Background(), p)

	// Sanity: the probe really switched the identity mid-flight.
	switched, err := os.ReadFile(filepath.Join(root, "home", ".gemini", "google_accounts.json"))
	if err != nil || !strings.Contains(string(switched), "b@example.test") {
		t.Fatalf("fake probe did not perform the mid-flight account switch: %q (%v)", switched, err)
	}

	// The observation is returned normally; the discard decision stays with the
	// profile layer.
	if snap.Status != model.UsageLive {
		t.Fatalf("status=%v, want LIVE (the snapshot is returned, only persistence is gated)", snap.Status)
	}
	if snap.Account != "b@example.test" {
		t.Fatalf("account=%q, want b@example.test (post-switch probe identity)", snap.Account)
	}

	// The identity gate refused persistence: no cache file was written.
	assertNoPersistedUsage(t, root)
}

// TestRefreshUsagePersistsSameAccount is the regression guard: a legitimate
// refresh for the same authenticated account must keep persisting the live
// snapshot exactly as before the identity gate was introduced.
func TestRefreshUsagePersistsSameAccount(t *testing.T) {
	p, scope, root, _ := agyIdentityFixture(t, "sameaccount", "a@example.test")
	installFakeAgyCLI(t, quotaOutput(42))

	snap := New().RefreshUsage(context.Background(), p)
	if snap.Status != model.UsageLive {
		t.Fatalf("status=%v, want LIVE", snap.Status)
	}
	if snap.Account != "a@example.test" {
		t.Fatalf("account=%q, want a@example.test", snap.Account)
	}

	data, err := os.ReadFile(filepath.Join(root, "usage.json"))
	if err != nil {
		t.Fatalf("same-account refresh must persist the live snapshot: %v", err)
	}
	if !strings.Contains(string(data), "a@example.test") {
		t.Fatalf("persisted cache lost the account identity: %s", data)
	}
	if !strings.Contains(string(data), "\"remaining_percent\": 42") {
		t.Fatalf("persisted cache lost the live quota evidence: %s", data)
	}
	if !strings.Contains(string(data), scope.AccountID) {
		t.Fatalf("persisted cache must carry the verifiable account scope: %s", data)
	}
}

// TestRefreshUsageRefusesSnapshotWithoutProvableIdentity: when the profile
// home has no authoritative identity artifacts (no google_accounts.json, no
// jetski state, no id_token email claim), the snapshot's account cannot be
// proven to belong to this profile. Fail closed: return the snapshot, persist
// nothing.
func TestRefreshUsageRefusesSnapshotWithoutProvableIdentity(t *testing.T) {
	p, _, root, _ := agyIdentityFixture(t, "noidentity", "")
	installFakeAgyCLI(t, quotaOutput(55))

	snap := New().RefreshUsage(context.Background(), p)
	if snap.Status != model.UsageLive {
		t.Fatalf("status=%v, want LIVE (snapshot is returned even when unattributable)", snap.Status)
	}
	if snap.Account != p.Name {
		t.Fatalf("account=%q, want the %q fallback (no identity evidence available)", snap.Account, p.Name)
	}
	assertNoPersistedUsage(t, root)
}

// TestSnapshotBoundToProfile unit-tests the fail-closed gate directly.
func TestSnapshotBoundToProfile(t *testing.T) {
	p, _, _, _ := agyIdentityFixture(t, "gate", "a@example.test")
	adapter := New()

	cases := []struct {
		name   string
		snap   model.UsageSnapshot
		anchor string
		want   bool
	}{
		{"matching identity", model.UsageSnapshot{Account: "a@example.test", Status: model.UsageLive}, "a@example.test", true},
		{"case-insensitive match", model.UsageSnapshot{Account: "A@Example.Test", Status: model.UsageLive}, "a@example.test", true},
		{"foreign account", model.UsageSnapshot{Account: "b@example.test", Status: model.UsageLive}, "a@example.test", false},
		{"empty anchor", model.UsageSnapshot{Account: "a@example.test", Status: model.UsageLive}, "", false},
		{"empty account", model.UsageSnapshot{Account: "", Status: model.UsageLive}, "a@example.test", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := adapter.snapshotBoundToProfile(p, tc.snap, tc.anchor); got != tc.want {
				t.Fatalf("snapshotBoundToProfile=%v, want %v", got, tc.want)
			}
		})
	}

	// No identity artifacts at all in the profile home → anchor "" → refuse.
	pNoID, _, _, _ := agyIdentityFixture(t, "gate-noid", "")
	if anchor := adapter.profileIdentityAnchor(pNoID); anchor != "" {
		t.Fatalf("anchor=%q, want empty without identity artifacts", anchor)
	}
	if adapter.snapshotBoundToProfile(pNoID, model.UsageSnapshot{Account: "someone@example.test"}, "someone@example.test") {
		t.Fatal("post-probe re-resolution must fail closed when identity artifacts are absent")
	}
}
