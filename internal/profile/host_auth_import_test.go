package profile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	coreprovider "github.com/IAPro-Community/nexus/internal/core/provider"
	"github.com/IAPro-Community/nexus/internal/core/provider/authcopy"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
)

type fakeHostSessionProvider struct {
	installed       bool
	source          bool
	invalidImported bool
}

func (f fakeHostSessionProvider) ID() model.ProviderID { return model.ProviderCodex }
func (f fakeHostSessionProvider) Name() string         { return "fake codex" }
func (f fakeHostSessionProvider) Detect(context.Context) model.DetectionResult {
	return model.DetectionResult{Installed: f.installed}
}
func (f fakeHostSessionProvider) Capabilities() model.Capabilities { return model.Capabilities{} }
func (f fakeHostSessionProvider) Prepare(context.Context, model.Profile) error {
	return nil
}
func (f fakeHostSessionProvider) Run(context.Context, model.Profile, []string) (model.Failure, error) {
	return model.Failure{}, nil
}
func (f fakeHostSessionProvider) Login(context.Context, model.Profile) error  { return nil }
func (f fakeHostSessionProvider) Logout(context.Context, model.Profile) error { return nil }
func (f fakeHostSessionProvider) InspectAuth(_ context.Context, p model.Profile) model.AccountInfo {
	home, _ := Home(p.Provider, p.Name)
	if _, err := os.Stat(filepath.Join(home, "imported-session")); err == nil && !f.invalidImported {
		return model.AccountInfo{Authenticated: true, Email: "host@example.test"}
	}
	return model.AccountInfo{Authenticated: false}
}
func (f fakeHostSessionProvider) ImportHostAuth(_ context.Context, p model.Profile) (bool, error) {
	home, _ := Home(p.Provider, p.Name)
	if _, err := os.Stat(filepath.Join(home, "reserved-auth")); err == nil {
		return false, authcopy.ErrDestinationExists
	}
	if !f.source {
		return false, nil
	}
	if err := os.WriteFile(filepath.Join(home, "imported-session"), []byte("fixture"), 0o600); err != nil {
		return false, err
	}
	return true, nil
}

var _ coreprovider.Provider = fakeHostSessionProvider{}
var _ coreprovider.AuthProvider = fakeHostSessionProvider{}
var _ coreprovider.HostAuthImporter = fakeHostSessionProvider{}

type fakeAuthenticatedProviderWithoutImporter struct{}

func (fakeAuthenticatedProviderWithoutImporter) ID() model.ProviderID { return model.ProviderCursor }
func (fakeAuthenticatedProviderWithoutImporter) Name() string         { return "Cursor CLI fixture" }
func (fakeAuthenticatedProviderWithoutImporter) Detect(context.Context) model.DetectionResult {
	return model.DetectionResult{Installed: true}
}
func (fakeAuthenticatedProviderWithoutImporter) Capabilities() model.Capabilities {
	return model.Capabilities{}
}
func (fakeAuthenticatedProviderWithoutImporter) Prepare(context.Context, model.Profile) error {
	return nil
}
func (fakeAuthenticatedProviderWithoutImporter) Run(context.Context, model.Profile, []string) (model.Failure, error) {
	return model.Failure{}, nil
}
func (fakeAuthenticatedProviderWithoutImporter) Login(context.Context, model.Profile) error {
	return nil
}
func (fakeAuthenticatedProviderWithoutImporter) Logout(context.Context, model.Profile) error {
	return nil
}
func (fakeAuthenticatedProviderWithoutImporter) InspectAuth(context.Context, model.Profile) model.AccountInfo {
	return model.AccountInfo{Authenticated: true, Status: "Authenticated"}
}

var _ coreprovider.Provider = fakeAuthenticatedProviderWithoutImporter{}
var _ coreprovider.AuthProvider = fakeAuthenticatedProviderWithoutImporter{}

func setupHostImportTest(t *testing.T) {
	t.Helper()
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
}

func TestImportHostSessionRegistersVerifiedProfileAndIsIdempotent(t *testing.T) {
	setupHostImportTest(t)
	adapter := fakeHostSessionProvider{installed: true, source: true}
	result, profiles := importHostSession(t.Context(), adapter, nil)
	if result.Status != HostSessionImported || result.Profile != "codex" {
		t.Fatalf("first import = %+v, want imported codex profile", result)
	}
	if len(profiles) != 1 {
		t.Fatalf("tracked profiles = %d, want 1", len(profiles))
	}
	cfg, err := config.LoadConfig()
	if err != nil || cfg.Defaults["codex"] != "codex" {
		t.Fatalf("default profile = %q, %v", cfg.Defaults["codex"], err)
	}

	result, _ = importHostSession(t.Context(), adapter, profiles)
	if result.Status != HostSessionAlreadyAuthenticated {
		t.Fatalf("repeat import = %+v, want already authenticated", result)
	}
}

func TestImportHostSessionRecognizesAuthenticatedProfileWithoutImporter(t *testing.T) {
	setupHostImportTest(t)
	profile, err := Create("cursor", "existing")
	if err != nil {
		t.Fatal(err)
	}
	profiles := []model.Profile{profile}
	adapter := fakeAuthenticatedProviderWithoutImporter{}
	if _, ok := any(adapter).(coreprovider.HostAuthImporter); ok {
		t.Fatal("fixture unexpectedly implements HostAuthImporter")
	}

	result, updated := importHostSession(t.Context(), adapter, profiles)
	if result.Status != HostSessionAlreadyAuthenticated || result.Profile != profile.Name {
		t.Fatalf("import = %+v, want authenticated profile without import", result)
	}
	if len(updated) != 1 || updated[0].Name != profile.Name {
		t.Fatalf("profiles after import = %+v, want the existing profile unchanged", updated)
	}
}

func TestImportHostSessionDoesNotLeaveEmptyProfileWithoutSource(t *testing.T) {
	setupHostImportTest(t)
	result, profiles := importHostSession(
		t.Context(), fakeHostSessionProvider{installed: true}, nil,
	)
	if result.Status != HostSessionNoSource {
		t.Fatalf("import = %+v, want no session", result)
	}
	if len(profiles) != 0 {
		t.Fatalf("profiles after unsuccessful import = %d, want 0", len(profiles))
	}
	stored, err := List()
	if err != nil || len(stored) != 0 {
		t.Fatalf("persisted profiles after unsuccessful import = %v, %v", stored, err)
	}
}

func TestImportHostSessionReportsFailedWhenCopiedAuthCannotBeVerified(t *testing.T) {
	setupHostImportTest(t)
	result, profiles := importHostSession(
		t.Context(),
		fakeHostSessionProvider{installed: true, source: true, invalidImported: true},
		nil,
	)
	if result.Status != HostSessionFailed {
		t.Fatalf("import = %+v, want failed verification status", result)
	}
	if len(profiles) != 0 {
		t.Fatalf("profiles after failed verification = %d, want 0", len(profiles))
	}
	stored, err := List()
	if err != nil || len(stored) != 0 {
		t.Fatalf("persisted profiles after failed verification = %v, %v", stored, err)
	}
}

func TestImportHostSessionPreservesPendingProfileAndUsesSeparateProfile(t *testing.T) {
	setupHostImportTest(t)
	pending, err := Create("codex", "codex")
	if err != nil {
		t.Fatal(err)
	}
	home, _ := Home("codex", pending.Name)
	if err := os.WriteFile(filepath.Join(home, "reserved-auth"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, profiles := importHostSession(
		t.Context(), fakeHostSessionProvider{installed: true, source: true}, []model.Profile{pending},
	)
	if result.Status != HostSessionImported || result.Profile != "host-import" {
		t.Fatalf("import with pending profile = %+v, want separate imported profile", result)
	}
	if len(profiles) != 2 {
		t.Fatalf("tracked profiles = %d, want 2", len(profiles))
	}
	if _, err := os.Stat(filepath.Join(home, "reserved-auth")); err != nil {
		t.Fatalf("pending profile credentials were not preserved: %v", err)
	}
}

func TestImportHostSessionSkipsUnavailableProviders(t *testing.T) {
	setupHostImportTest(t)
	result, profiles := importHostSession(t.Context(), fakeHostSessionProvider{}, nil)
	if result.Status != HostSessionNotInstalled || len(profiles) != 0 {
		t.Fatalf("unavailable provider import = %+v, profiles=%d", result, len(profiles))
	}
}

func TestImportHostSessionsComposesInstalledAdapterWithHostCredential(t *testing.T) {
	dataDir := t.TempDir()
	configDir := t.TempDir()
	hostHome := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("NEXUS_CONFIG_DIR", configDir)
	t.Setenv("NEXUS_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)
	t.Setenv("PATH", binDir)

	// Only a version-probe fixture is installed; no provider request is made.
	codexCLI := filepath.Join(binDir, "codex")
	if err := os.WriteFile(codexCLI, []byte("#!/bin/sh\nprintf 'codex 0.1.0\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	previousBinaryLookPath := spec.BinaryLookPath
	spec.BinaryLookPath = func(name string) (string, error) {
		if name == "codex" {
			return codexCLI, nil
		}
		return "", errors.New("fixture provider is not installed")
	}
	t.Cleanup(func() { spec.BinaryLookPath = previousBinaryLookPath })
	payload, err := json.Marshal(map[string]any{
		"email":                       "fixture@example.test",
		"https://api.openai.com/auth": map[string]string{"chatgpt_account_id": "fixture-account"},
	})
	if err != nil {
		t.Fatal(err)
	}
	jwt := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".fixture-signature"
	hostAuth, err := json.Marshal(map[string]any{"tokens": map[string]string{
		"id_token": jwt, "access_token": "fixture-access", "refresh_token": "fixture-refresh",
	}})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(hostHome, ".codex", "auth.json")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, hostAuth, 0o600); err != nil {
		t.Fatal(err)
	}

	results, err := ImportHostSessions(t.Context())
	if err != nil {
		t.Fatalf("import host sessions: %v", err)
	}
	var codexResult *HostSessionImportResult
	for i := range results {
		if results[i].Provider == "codex" {
			codexResult = &results[i]
			break
		}
	}
	if codexResult == nil || codexResult.Status != HostSessionImported || codexResult.Profile != "codex" {
		t.Fatalf("Codex import result = %+v, want imported codex profile", codexResult)
	}
	profileHome, err := config.ProfileHome("codex", codexResult.Profile)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(profileHome, "auth.json")
	imported, err := os.ReadFile(destination)
	if err != nil || string(imported) != string(hostAuth) {
		t.Fatalf("isolated auth differs from fixture: %v", err)
	}
	info, err := os.Stat(destination)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("imported auth mode = %v, %v; want 0600", info, err)
	}
	if got, err := os.ReadFile(source); err != nil || string(got) != string(hostAuth) {
		t.Fatalf("host credential changed: %v", err)
	}
	cfg, err := config.LoadConfig()
	if err != nil || cfg.Defaults["codex"] != codexResult.Profile {
		t.Fatalf("Codex default = %q, %v", cfg.Defaults["codex"], err)
	}
	for _, result := range results {
		if result.Provider != "codex" && result.Status != HostSessionNotInstalled {
			t.Fatalf("unexpected non-fixture provider execution: %+v", result)
		}
	}
	encodedResults, err := json.Marshal(results)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"fixture-access", "fixture-refresh", "fixture@example.test"} {
		if strings.Contains(string(encodedResults), secret) {
			t.Fatalf("host session response exposed sensitive data %q", secret)
		}
	}
}
