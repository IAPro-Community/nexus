package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateListDefaultDelete(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)

	if _, err := Create("codex", "alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := Create("agy", "google-a"); err != nil {
		t.Fatal(err)
	}
	if !Exists("codex", "alpha") {
		t.Fatal("codex profile should exist")
	}
	if err := SetDefault("codex", "alpha"); err != nil {
		t.Fatal(err)
	}
	d, err := Default("codex")
	if err != nil {
		t.Fatal(err)
	}
	if d != "alpha" {
		t.Fatalf("default=%q", d)
	}
	ps, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 {
		t.Fatalf("got %d profiles, want 2", len(ps))
	}
	if err := Delete("codex", "alpha"); err != nil {
		t.Fatal(err)
	}
	if Exists("codex", "alpha") {
		t.Fatal("profile should be deleted")
	}
}

func TestEnsureRandomSecretPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := EnsureRandomSecret(path); err != nil {
		t.Fatal(err)
	}
}

func TestCursorProfilesAreSupported(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	if _, err := Create("cursor", "work"); err != nil {
		t.Fatalf("cursor profile should be supported: %v", err)
	}
	profiles, err := List()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range profiles {
		if p.Provider == "cursor" && p.Name == "work" {
			found = true
		}
	}
	if !found {
		t.Fatalf("cursor profile missing from List: %#v", profiles)
	}
}

func TestListRecoversProfileRootWithoutMetadata(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)

	// A profile root created before profile.json was written (interrupted
	// creation, legacy `use` command) must not hide the account from listings.
	root := filepath.Join(data, "profiles", "codex", "work")
	if err := os.MkdirAll(filepath.Join(root, "home"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "account-scope.json"), []byte("{\"provider_id\":\"codex\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}

	ps, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 || ps[0].Provider != "codex" || ps[0].Name != "work" {
		t.Fatalf("recovered profiles = %+v, want codex:work", ps)
	}
	if !Exists("codex", "work") {
		t.Fatal("recovered profile must be visible to Exists after List")
	}
}

func TestListIgnoresUnrelatedDirectories(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)

	root := filepath.Join(data, "profiles", "codex", "not-a-profile")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	ps, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 0 {
		t.Fatalf("profiles = %+v, want none", ps)
	}
	if _, err := os.Stat(filepath.Join(root, "profile.json")); !os.IsNotExist(err) {
		t.Fatal("unrelated directory must not gain a profile.json")
	}
}

func TestEnsureHostCredentialRespectsExplicitLogout(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)

	root := filepath.Join(data, "profiles", "codex", "work")
	if err := os.MkdirAll(filepath.Join(root, "home"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profile.json"), []byte("{\"provider\":\"codex\",\"name\":\"work\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := SuppressHostAuthLink("codex", "work"); err != nil {
		t.Fatal(err)
	}
	if _, linked := EnsureHostCredential("codex", "work"); linked {
		t.Fatal("explicit logout must keep automatic host linking off")
	}
	if _, err := os.Stat(filepath.Join(root, "home", "auth.json")); !os.IsNotExist(err) {
		t.Fatal("no credential must be written after explicit logout")
	}
}

func TestEnsureHostCredentialNeverOverwritesExistingCredential(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", data)
	t.Setenv("AI_CLI_CONFIG_DIR", cfg)

	root := filepath.Join(data, "profiles", "codex", "work")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "profile.json"), []byte("{\"provider\":\"codex\",\"name\":\"work\"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"auth_mode":"apikey","tokens":{}}`)
	if err := os.WriteFile(filepath.Join(home, "auth.json"), original, 0600); err != nil {
		t.Fatal(err)
	}
	if _, linked := EnsureHostCredential("codex", "work"); linked {
		t.Fatal("a profile with its own credential must not be linked")
	}
	got, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("credential overwritten: %q", got)
	}
}

func TestOMPProfileValidationRequiresExplicitPreview(t *testing.T) {
	t.Setenv("NEXUS_EXPERIMENTAL_HARNESSES", "")
	if err := ValidateProvider("omp"); err == nil {
		t.Fatal("OMP provider must stay gated by experimental opt-in")
	}
	t.Setenv("NEXUS_EXPERIMENTAL_HARNESSES", "1")
	if err := ValidateProvider("omp"); err != nil {
		t.Fatalf("OMP profile should be valid with opt-in: %v", err)
	}
	if err := ValidateProvider("unrecognized-harness"); err == nil {
		t.Fatal("unknown providers must not be enabled by OMP opt-in")
	}
}

func TestOMPPreviewProfileIsVisibleOnlyWhenEnabled(t *testing.T) {
	t.Setenv("AI_CLI_DATA_DIR", t.TempDir())
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("NEXUS_EXPERIMENTAL_HARNESSES", "1")
	if _, err := Create("omp", "default"); err != nil {
		t.Fatalf("create OMP preview profile: %v", err)
	}
	profiles, err := List()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range profiles {
		if p.Provider == "omp" && p.Name == "default" {
			found = true
		}
	}
	if !found {
		t.Fatal("OMP preview profile must be visible to the session selector")
	}
	t.Setenv("NEXUS_EXPERIMENTAL_HARNESSES", "")
	profiles, err = List()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range profiles {
		if p.Provider == "omp" {
			t.Fatal("OMP preview profile must not appear when feature flag is off")
		}
	}
}
