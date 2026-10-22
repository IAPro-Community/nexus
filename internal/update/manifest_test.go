package update

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManifestValidationRejectsExpiredDowngradeAndWrongTarget(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "0.5.1",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:     time.Now().UTC().Add(-time.Minute).Format(time.RFC3339),
		KeyID:         "test",
		Artifacts: map[string]Artifact{
			"linux_amd64": {SHA256: strings.Repeat("a", 64), Size: 3},
		},
	}
	if err := manifest.Validate(ManifestPolicy{Channel: "beta", CurrentVersion: "0.5.0", Target: "linux_amd64"}); err == nil {
		t.Fatal("expected expired manifest to be rejected")
	}

	manifest.ExpiresAt = time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	manifest.Version = "0.4.9"
	if err := manifest.Validate(ManifestPolicy{Channel: "beta", CurrentVersion: "0.5.0", Target: "linux_amd64"}); err == nil {
		t.Fatal("expected downgrade manifest to be rejected")
	}

	manifest.Version = "0.5.1"
	if err := manifest.Validate(ManifestPolicy{Channel: "beta", CurrentVersion: "0.5.0", Target: "darwin_arm64"}); err == nil {
		t.Fatal("expected missing target artifact to be rejected")
	}
}

func TestManifestValidationAcceptsCompatibilityAndTarget(t *testing.T) {
	manifest := Manifest{
		SchemaVersion:     1,
		Channel:           "stable",
		Version:           "1.2.0",
		ReleaseDate:       time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:         time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		MinNexusVersion:   "1.0.0",
		MinMaestroVersion: "0.14.0",
		KeyID:             "test",
		Artifacts:         map[string]Artifact{"linux_amd64": {SHA256: strings.Repeat("a", 64), Size: 3}},
	}
	if err := manifest.Validate(ManifestPolicy{Channel: "stable", CurrentVersion: "1.1.0", NexusVersion: "1.1.0", MaestroVersion: "0.14.1", Target: "linux_amd64"}); err != nil {
		t.Fatalf("expected compatible manifest to validate: %v", err)
	}
}

func TestUpdaterRollbackKeepsCurrentBinaryWhenRestoreFails(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "nexus")
	if err := os.WriteFile(binPath, []byte("current"), 0755); err != nil {
		t.Fatal(err)
	}
	updater := NewUpdater(binPath, tempDir)
	if _, err := updater.ApplyUpdate("1.0.0", "1.1.0", []byte("next"), sha256Hex([]byte("next"))); err != nil {
		t.Fatalf("apply update: %v", err)
	}
	if err := os.Remove(binPath); err != nil {
		t.Fatal(err)
	}
	if _, err := updater.Rollback(); err != nil {
		t.Fatalf("rollback should restore backup: %v", err)
	}
	data, err := os.ReadFile(binPath)
	if err != nil || string(data) != "current" {
		t.Fatalf("expected current binary restored, got %q, err=%v", data, err)
	}
}

func TestUpdaterApplyManifestRejectsSameVersion(t *testing.T) {
	dir := t.TempDir()
	updater := NewUpdater(filepath.Join(dir, "nexus"), dir)
	content := []byte("same-version-payload")
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "1.0.0-beta.10",
		Artifacts: map[string]Artifact{
			"linux_amd64": {
				URL:    "https://example.invalid/cli",
				Size:   int64(len(content)),
				SHA256: sha256Hex(content),
			},
		},
	}
	_, err := updater.ApplyManifest(manifest, ManifestPolicy{
		Channel:        "beta",
		CurrentVersion: "1.0.0-beta.10",
		Target:         "linux_amd64",
	}, content)
	if err != ErrNoUpdateAvailable {
		t.Fatalf("same-version update must fail closed, got %v", err)
	}
	if _, err := os.Stat(updater.BinaryPath); !os.IsNotExist(err) {
		t.Fatalf("same-version release must not replace executable: %v", err)
	}
}

func TestUpdaterApplyManifestBindsTargetChecksum(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "nexus")
	if err := os.WriteFile(binPath, []byte("current"), 0755); err != nil {
		t.Fatal(err)
	}
	data := []byte("signed-artifact")
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "1.1.0",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         "test",
		Artifacts: map[string]Artifact{"linux_amd64": {
			Size: int64(len(data)), SHA256: sha256Hex(data),
		}},
	}
	updater := NewUpdater(binPath, tempDir)
	if _, err := updater.ApplyManifest(manifest, ManifestPolicy{Channel: "beta", CurrentVersion: "1.0.0", Target: "linux_amd64"}, data); err != nil {
		t.Fatalf("expected manifest-bound update to succeed: %v", err)
	}
	bad := append([]byte(nil), data...)
	bad[0] = 'x'
	if _, err := updater.ApplyManifest(manifest, ManifestPolicy{Channel: "beta", CurrentVersion: "1.0.0", Target: "linux_amd64"}, bad); err == nil {
		t.Fatal("expected artifact bound to manifest checksum to fail")
	}
}

func TestUpdaterApplyManifestRejectsArchiveWithoutExtractionTarget(t *testing.T) {
	data := []byte("archive bytes")
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "stable",
		Version:       "1.1.0",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         "test",
		Artifacts: map[string]Artifact{"linux_amd64": {
			URL:    "https://example.invalid/releases/nexus_Linux_x86_64.tar.gz",
			Size:   int64(len(data)),
			SHA256: sha256Hex(data),
		}},
	}
	updater := NewUpdater(filepath.Join(t.TempDir(), "nexus"), t.TempDir())
	if _, err := updater.ApplyManifest(manifest, ManifestPolicy{
		Channel: "stable", CurrentVersion: "1.0.0", Target: "linux_amd64",
	}, data); err == nil || !strings.Contains(err.Error(), "extraction target") {
		t.Fatalf("expected archive without target to be rejected, got: %v", err)
	}
}

func generateTestKeyPair(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}
	return pub, priv
}

func TestVerifyManifest_ValidSignature(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	keyID := "test-key-2026-v1"
	ring := NewKeyRing()
	ring.AddKey(keyID, pub)

	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "0.5.1",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         keyID,
		Artifacts: map[string]Artifact{
			"linux_amd64": {
				URL:    "https://github.com/IAPro-Community/nexus/releases/download/v0.5.1/nexus_Linux_x86_64.tar.gz",
				Size:   1234567,
				SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		},
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}

	sig := ed25519.Sign(priv, manifestBytes)
	sigHex := hex.EncodeToString(sig)

	verified, err := ring.VerifyManifest(manifestBytes, sigHex)
	if err != nil {
		t.Fatalf("expected signature verification to succeed, got: %v", err)
	}
	if verified.Version != "0.5.1" {
		t.Errorf("expected version 0.5.1, got %s", verified.Version)
	}
	if verified.Channel != "beta" {
		t.Errorf("expected channel beta, got %s", verified.Channel)
	}
}

func TestVerifyManifest_TamperedManifest(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	keyID := "test-key-2026-v1"
	ring := NewKeyRing()
	ring.AddKey(keyID, pub)

	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "0.5.1",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         keyID,
	}

	manifestBytes, _ := json.Marshal(manifest)
	sig := ed25519.Sign(priv, manifestBytes)
	sigHex := hex.EncodeToString(sig)

	// Tamper bytes
	tamperedBytes := []byte(string(manifestBytes) + " ")

	_, err := ring.VerifyManifest(tamperedBytes, sigHex)
	if err == nil {
		t.Fatal("expected tampered manifest verification to fail, but it succeeded")
	}
}

func TestVerifyManifest_UntrustedKeyID(t *testing.T) {
	_, priv := generateTestKeyPair(t)
	ring := NewKeyRing() // empty ring

	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "0.5.1",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         "untrusted-key",
	}

	manifestBytes, _ := json.Marshal(manifest)
	sig := ed25519.Sign(priv, manifestBytes)
	sigHex := hex.EncodeToString(sig)

	_, err := ring.VerifyManifest(manifestBytes, sigHex)
	if err == nil {
		t.Fatal("expected verification to fail for untrusted key ID")
	}
}

func TestUpdatePlan_ChecksumVerificationAndRollback(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "nexus")

	// Create original binary
	origContent := []byte("nexus-v0.5.0-binary-content")
	if err := os.WriteFile(binPath, origContent, 0755); err != nil {
		t.Fatalf("failed to write original binary: %v", err)
	}

	// Target artifact
	newContent := []byte("nexus-v0.5.1-binary-content")
	sum := sha256.Sum256(newContent)
	sumHex := hex.EncodeToString(sum[:])

	updater := NewUpdater(binPath, tempDir)

	// 1. Verify Checksum matches
	if err := updater.VerifyArtifactChecksum(newContent, sumHex); err != nil {
		t.Fatalf("expected valid checksum to pass: %v", err)
	}

	// 2. Verify Checksum mismatch fails
	if err := updater.VerifyArtifactChecksum(newContent, "invalidhash"); err == nil {
		t.Fatal("expected invalid checksum to fail")
	}

	// 3. Apply update with atomic backup and receipt
	receipt, err := updater.ApplyUpdate("0.5.0", "0.5.1", newContent, sumHex)
	if err != nil {
		t.Fatalf("failed to apply update: %v", err)
	}
	if receipt.Status != StatusSuccess {
		t.Fatalf("expected status %s, got %s", StatusSuccess, receipt.Status)
	}

	// Verify new content is in place
	currBytes, _ := os.ReadFile(binPath)
	if string(currBytes) != string(newContent) {
		t.Fatalf("expected binary content %q, got %q", string(newContent), string(currBytes))
	}

	// 4. Test Rollback
	rollbackReceipt, err := updater.Rollback()
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}
	if rollbackReceipt.Status != StatusRolledBack {
		t.Fatalf("expected status %s, got %s", StatusRolledBack, rollbackReceipt.Status)
	}

	restoredBytes, _ := os.ReadFile(binPath)
	if string(restoredBytes) != string(origContent) {
		t.Fatalf("expected restored content %q, got %q", string(origContent), string(restoredBytes))
	}
}

func TestCompareVersionsSemverCompliance(t *testing.T) {
	tests := []struct {
		a, b  string
		want  int
		label string
	}{
		// Equal
		{"1.0.0", "1.0.0", 0, "equal stable"},
		{"0.5.0-beta.23", "0.5.0-beta.23", 0, "equal prerelease"},

		// Major/minor/patch ordering
		{"0.4.0", "0.5.0", -1, "minor upgrade"},
		{"0.5.0", "0.4.0", 1, "minor downgrade"},
		{"1.0.0", "2.0.0", -1, "major upgrade"},
		{"0.5.0", "0.5.1", -1, "patch upgrade"},

		// Prerelease ordering within same version
		{"0.5.0-beta.23", "0.5.0-beta.24", -1, "beta increment"},
		{"0.5.0-beta.24", "0.5.0-rc.1", -1, "beta to rc"},
		{"0.5.0-rc.1", "0.5.0-rc.2", -1, "rc increment"},

		// Prerelease < stable (SemVer 2.0.0 §11)
		{"0.5.0-beta.23", "0.5.0", -1, "beta < stable"},
		{"0.5.0-rc.1", "0.5.0", -1, "rc < stable"},
		{"0.5.0", "0.5.0-beta.23", 1, "stable > beta"},
		{"0.5.0", "0.5.0-rc.1", 1, "stable > rc"},

		// Leading v prefix
		{"v0.5.0", "0.5.0", 0, "v prefix ignored"},
		{"v0.5.0-beta.23", "0.5.0-beta.23", 0, "v prefix with prerelease"},

		// Cross-level comparisons
		{"0.5.0-beta.23", "0.5.1", -1, "beta < next patch"},
		{"0.5.0-rc.1", "0.5.1", -1, "rc < next patch"},
		{"1.0.0-alpha.1", "0.9.9", 1, "alpha.1 > 0.9.9 (major higher)"},
	}

	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d (%s)", tt.a, tt.b, got, tt.want, tt.label)
		}
	}
}

func TestCompareVersionsDowngradeRejection(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "0.5.0-beta.23",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:     time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		KeyID:         "test",
		Artifacts:     map[string]Artifact{"linux_amd64": {SHA256: strings.Repeat("a", 64), Size: 3}},
	}

	// Current version is newer: should reject
	if err := manifest.Validate(ManifestPolicy{Channel: "beta", CurrentVersion: "0.5.0-beta.24", Target: "linux_amd64"}); err == nil {
		t.Fatal("expected downgrade from beta.24 to beta.23 to be rejected")
	}

	// Current version is same: should pass
	manifest.Version = "0.5.0-beta.24"
	if err := manifest.Validate(ManifestPolicy{Channel: "beta", CurrentVersion: "0.5.0-beta.24", Target: "linux_amd64"}); err != nil {
		t.Fatalf("expected same version to pass: %v", err)
	}

	// Current is beta, manifest is stable (upgrade): should pass
	manifest.Version = "0.5.0"
	manifest.Channel = "stable"
	if err := manifest.Validate(ManifestPolicy{Channel: "stable", CurrentVersion: "0.5.0-beta.24", Target: "linux_amd64"}); err != nil {
		t.Fatalf("expected beta->stable upgrade to pass: %v", err)
	}
}

func TestManifestValidationRejectsMalformedSourceSHA(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "1.0.0-beta.4",
		SourceSHA:     "not-a-source-sha",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         "test",
	}
	if err := manifest.Validate(ManifestPolicy{}); err == nil || !strings.Contains(err.Error(), "source_sha") {
		t.Fatalf("expected malformed source_sha rejection, got %v", err)
	}
}

func TestManifestPreservesSourceSHA(t *testing.T) {
	const sourceSHA = "0123456789abcdef0123456789abcdef01234567"
	original := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "1.0.0-beta.4",
		SourceSHA:     sourceSHA,
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         "test",
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Manifest
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SourceSHA != sourceSHA {
		t.Fatalf("source_sha=%q want=%q", decoded.SourceSHA, sourceSHA)
	}
}
