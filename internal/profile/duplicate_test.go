package profile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/adapters/codex"
)

func writeFakeCodexAuth(t *testing.T, authPath, accountID, email string) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"email":                       email,
		"https://api.openai.com/auth": map[string]string{"chatgpt_account_id": accountID, "chatgpt_plan_type": "plus"},
	})
	if err != nil {
		t.Fatal(err)
	}
	jwt := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	raw, err := json.Marshal(map[string]any{
		"auth_mode": "chatgpt",
		"tokens": map[string]string{
			"id_token":      jwt,
			"access_token":  "access",
			"refresh_token": "refresh",
			"account_id":    accountID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(authPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAdoptHostAuthSkipsWhenAccountAlreadyOwnedByAnotherProfile(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	hostHome := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)

	writeFakeCodexAuth(t, filepath.Join(hostHome, ".codex", "auth.json"), "acct-shared-1", "owner@example.com")

	if _, err := Create("codex", "first"); err != nil {
		t.Fatalf("Create first: %v", err)
	}
	firstInfo := GetAccountInfo("codex", "first")
	if !firstInfo.Authenticated {
		t.Fatalf("first profile should adopt host auth, got %+v", firstInfo)
	}

	if _, err := Create("codex", "second"); err != nil {
		t.Fatalf("Create second: %v", err)
	}
	secondInfo := GetAccountInfo("codex", "second")
	if secondInfo.Authenticated {
		t.Fatalf("second profile must NOT duplicate already-registered host account, got %+v", secondInfo)
	}
}

func TestValidateNoDuplicateAccountDetectsSameCodexAccountAndEmail(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("NEXUS_REAL_HOME", t.TempDir())

	if _, err := Create("codex", "alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := Create("codex", "beta"); err != nil {
		t.Fatal(err)
	}
	alphaHome, _ := Home("codex", "alpha")
	betaHome, _ := Home("codex", "beta")
	writeFakeCodexAuth(t, filepath.Join(alphaHome, "auth.json"), "acct-dup", "same@example.com")
	writeFakeCodexAuth(t, filepath.Join(betaHome, "auth.json"), "acct-dup", "same@example.com")

	err := ValidateNoDuplicateAccount(context.Background(), "codex", "beta")
	var dupErr *DuplicateAccountError
	if !errors.As(err, &dupErr) {
		t.Fatalf("expected DuplicateAccountError, got %v", err)
	}
	if dupErr.ExistingProfile != "alpha" {
		t.Fatalf("ExistingProfile=%q want alpha", dupErr.ExistingProfile)
	}

	// Non-Codex email matching is case-insensitive.
	if _, err := Create("agy", "work"); err != nil {
		t.Fatal(err)
	}
	if _, err := Create("agy", "clone"); err != nil {
		t.Fatal(err)
	}
	workHome, _ := Home("agy", "work")
	cloneHome, _ := Home("agy", "clone")
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	writeAGY := func(home, email string) {
		tokenDir := filepath.Join(home, ".gemini", "antigravity-cli")
		_ = os.MkdirAll(tokenDir, 0o700)
		_ = os.WriteFile(filepath.Join(home, ".gemini", "google_accounts.json"), []byte(`{"active":"`+email+`"}`), 0o600)
		_ = os.WriteFile(filepath.Join(tokenDir, "antigravity-oauth-token"), []byte(`{"token":{"access_token":"ya29.x","refresh_token":"1//r","expiry":"`+future+`"}}`), 0o600)
	}
	writeAGY(workHome, "User@Example.COM")
	writeAGY(cloneHome, "user@example.com")

	err = ValidateNoDuplicateAccount(context.Background(), "agy", "clone")
	if !errors.As(err, &dupErr) || dupErr.ExistingProfile != "work" {
		t.Fatalf("expected AGY DuplicateAccountError referencing work, got %v", err)
	}
}

func TestFindDuplicateAccountGroups(t *testing.T) {
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("NEXUS_REAL_HOME", t.TempDir())

	for _, name := range []string{"a", "b", "c"} {
		if _, err := Create("codex", name); err != nil {
			t.Fatal(err)
		}
	}
	aHome, _ := Home("codex", "a")
	bHome, _ := Home("codex", "b")
	cHome, _ := Home("codex", "c")
	writeFakeCodexAuth(t, filepath.Join(aHome, "auth.json"), "acct-1", "dup@example.com")
	writeFakeCodexAuth(t, filepath.Join(bHome, "auth.json"), "acct-1", "dup@example.com")
	writeFakeCodexAuth(t, filepath.Join(cHome, "auth.json"), "acct-2", "other@example.com")

	groups := FindDuplicateAccountGroups(context.Background(), []model.Profile{
		{Provider: "codex", Name: "b"},
		{Provider: "codex", Name: "a"},
		{Provider: "codex", Name: "c"},
	})
	if len(groups) != 1 {
		t.Fatalf("groups=%d want 1 (%+v)", len(groups), groups)
	}
	if len(groups[0].Profiles) != 2 || groups[0].Profiles[0] != "a" || groups[0].Profiles[1] != "b" {
		t.Fatalf("unexpected duplicate profiles: %+v", groups[0].Profiles)
	}
}

func setupIsolatedProfileEnv(t *testing.T) {
	t.Helper()
	dataDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_MANAGER_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("AI_CLI_CONFIG_DIR", cfgDir)
	t.Setenv("AI_MANAGER_CONFIG_DIR", cfgDir)
	t.Setenv("NEXUS_REAL_HOME", t.TempDir())
}

func TestCanonicalAccountIdentityCursorPrefersExternalID(t *testing.T) {
	id := CanonicalAccountIdentity("cursor", model.AccountInfo{
		Authenticated:     true,
		Email:             "User@Example.COM",
		ExternalAccountID: "auth0|User_123",
	})
	if id != "auth0|user_123" {
		t.Fatalf("got %q want auth0|user_123", id)
	}
	fallback := CanonicalAccountIdentity("cursor", model.AccountInfo{
		Authenticated: true,
		Email:         "User@Example.COM",
	})
	if fallback != "user@example.com" {
		t.Fatalf("got %q want user@example.com", fallback)
	}
}

func TestHasDuplicateAccounts(t *testing.T) {
	profiles := []model.Profile{
		{Provider: "codex", Name: "one"},
		{Provider: "codex", Name: "two"},
		{Provider: "codex", Name: "disabled", Disabled: true},
	}
	noDup := map[string]model.AccountInfo{
		"codex:one":      {Authenticated: true, ExternalAccountID: "acct-1"},
		"codex:two":      {Authenticated: true, ExternalAccountID: "acct-2"},
		"codex:disabled": {Authenticated: true, ExternalAccountID: "acct-1"},
	}
	if HasDuplicateAccounts(profiles, noDup) {
		t.Fatal("expected HasDuplicateAccounts=false when accounts differ or duplicate is disabled")
	}
	withDup := map[string]model.AccountInfo{
		"codex:one": {Authenticated: true, ExternalAccountID: "acct-1"},
		"codex:two": {Authenticated: true, ExternalAccountID: "acct-1"},
	}
	if !HasDuplicateAccounts(profiles[:2], withDup) {
		t.Fatal("expected HasDuplicateAccounts=true")
	}
}

func TestDeduplicateAccountsKeepsHealthyDefaultAndMigratesBindings(t *testing.T) {
	setupIsolatedProfileEnv(t)

	for _, name := range []string{"old", "keeper", "expired"} {
		if _, err := Create("codex", name); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"old", "keeper", "expired"} {
		home, _ := Home("codex", name)
		writeFakeCodexAuth(t, filepath.Join(home, "auth.json"), "acct-shared", "shared@example.com")
	}
	// Mark "expired" with cached AUTH_REQUIRED snapshot so it loses to healthy candidates.
	expiredRoot, _ := Root("codex", "expired")
	expiredSnap, _ := json.Marshal(model.UsageSnapshot{
		ProviderID:     "codex",
		ProfileID:      "expired",
		Status:         model.UsageError,
		Source:         model.SourceOfficialAPI,
		DiagnosticCode: "AUTH_REQUIRED",
		FetchedAt:      time.Now(),
	})
	if err := os.WriteFile(filepath.Join(expiredRoot, "usage.json"), expiredSnap, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.SetDefaultProfile("codex", "keeper"); err != nil {
		t.Fatal(err)
	}
	if err := config.BindWorkspace("/work/proj", "codex", "old"); err != nil {
		t.Fatal(err)
	}

	cleaned, err := DeduplicateAccounts(context.Background())
	if err != nil {
		t.Fatalf("DeduplicateAccounts: %v", err)
	}
	if len(cleaned) != 1 || cleaned[0].KeptProfile != "keeper" || len(cleaned[0].RemovedProfiles) != 2 {
		t.Fatalf("unexpected cleanup result: %+v", cleaned)
	}
	if Exists("codex", "old") || Exists("codex", "expired") || !Exists("codex", "keeper") {
		t.Fatalf("unexpected surviving profiles after dedupe")
	}
	def, _ := config.GetDefaultProfile("codex")
	if def != "keeper" {
		t.Fatalf("default profile=%q want keeper", def)
	}
	bound := config.GetBinding("/work/proj", "codex")
	if bound != "keeper" {
		t.Fatalf("binding=%q want keeper", bound)
	}
}

func TestDeduplicateAccountsSkipsLockedCodexProfile(t *testing.T) {
	setupIsolatedProfileEnv(t)

	for _, name := range []string{"primary", "locked-dup"} {
		if _, err := Create("codex", name); err != nil {
			t.Fatal(err)
		}
		home, _ := Home("codex", name)
		writeFakeCodexAuth(t, filepath.Join(home, "auth.json"), "acct-lock", "lock@example.com")
	}
	if err := config.SetDefaultProfile("codex", "primary"); err != nil {
		t.Fatal(err)
	}
	lockedHome, _ := Home("codex", "locked-dup")
	lease, err := codex.AcquireTUILease(lockedHome)
	if err != nil {
		t.Fatalf("AcquireTUILease: %v", err)
	}
	t.Cleanup(func() { _ = lease.Release() })

	cleaned, err := DeduplicateAccounts(context.Background())
	if err != nil {
		t.Fatalf("DeduplicateAccounts: %v", err)
	}
	if len(cleaned) != 1 || cleaned[0].KeptProfile != "primary" || len(cleaned[0].SkippedLocked) != 1 || cleaned[0].SkippedLocked[0] != "locked-dup" {
		t.Fatalf("expected locked-dup to be skipped cleanly, got %+v", cleaned)
	}
	if !Exists("codex", "locked-dup") {
		t.Fatal("locked-dup must not be deleted while TUI lease is active")
	}
}
