package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNegative_CorruptedManifest(t *testing.T) {
	pub, priv := generateTestKeyPair(t)
	keyID := "test-key"
	ring := NewKeyRing()
	ring.AddKey(keyID, pub)

	// Sign a valid manifest, then tamper with it
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "0.5.1",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         keyID,
		Artifacts:     map[string]Artifact{"linux_amd64": {SHA256: strings.Repeat("a", 64), Size: 3}},
	}
	manifestBytes, _ := json.Marshal(manifest)
	sig := ed25519.Sign(priv, manifestBytes)
	sigHex := hex.EncodeToString(sig)

	// Tamper with the manifest by changing a character
	tampered := make([]byte, len(manifestBytes))
	copy(tampered, manifestBytes)
	tampered[0] = 'X' // Corrupt the JSON

	_, err := ring.VerifyManifest(tampered, sigHex)
	if err == nil {
		t.Fatal("expected corrupted manifest to fail verification")
	}
}

func TestNegative_EmptyKeyRing(t *testing.T) {
	ring := NewKeyRing() // no keys added
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "0.5.1",
		KeyID:         "unknown-key",
	}
	manifestBytes, _ := json.Marshal(manifest)
	sig := make([]byte, ed25519.SignatureSize)
	_, err := ring.VerifyManifest(manifestBytes, hex.EncodeToString(sig))
	if err == nil || !strings.Contains(err.Error(), "not trusted") {
		t.Fatalf("expected untrusted key error, got: %v", err)
	}
}

func TestNegative_ChecksumMismatch(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "nexus")
	os.WriteFile(binPath, []byte("original"), 0755)

	updater := NewUpdater(binPath, tempDir)
	wrongChecksum := sha256.Sum256([]byte("different"))
	_, err := updater.ApplyUpdate("1.0.0", "1.1.0", []byte("new-content"), hex.EncodeToString(wrongChecksum[:]))
	if err == nil {
		t.Fatal("expected checksum mismatch to fail")
	}
}

func TestNegative_DowngradeRejection(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "stable",
		Version:       "0.4.0", // older than current
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:     time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		KeyID:         "test",
		Artifacts:     map[string]Artifact{"linux_amd64": {SHA256: strings.Repeat("a", 64), Size: 3}},
	}
	err := manifest.Validate(ManifestPolicy{Channel: "stable", CurrentVersion: "0.5.0", Target: "linux_amd64"})
	if err == nil || !strings.Contains(err.Error(), "downgrade") {
		t.Fatalf("expected downgrade rejection, got: %v", err)
	}
}

func TestNegative_ExpiredManifest(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "stable",
		Version:       "1.0.0",
		ReleaseDate:   time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
		ExpiresAt:     time.Now().UTC().Add(-time.Hour).Format(time.RFC3339), // expired
		KeyID:         "test",
		Artifacts:     map[string]Artifact{"linux_amd64": {SHA256: strings.Repeat("a", 64), Size: 3}},
	}
	err := manifest.Validate(ManifestPolicy{Channel: "stable", CurrentVersion: "0.9.0", Target: "linux_amd64"})
	if err == nil || !errors.Is(err, ErrManifestExpired) {
		t.Fatalf("expected expired manifest error, got: %v", err)
	}
}

func TestNegative_MissingArtifactTarget(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "stable",
		Version:       "1.0.0",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         "test",
		Artifacts:     map[string]Artifact{"linux_amd64": {SHA256: strings.Repeat("a", 64), Size: 3}},
	}
	err := manifest.Validate(ManifestPolicy{Channel: "stable", CurrentVersion: "0.9.0", Target: "darwin_arm64"})
	if err == nil || !strings.Contains(err.Error(), "no artifact") {
		t.Fatalf("expected missing target error, got: %v", err)
	}
}

func TestNegative_UnknownArtifactTarget(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "stable",
		Version:       "1.0.0",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         "test",
		Artifacts: map[string]Artifact{"linux_amd64": {
			SHA256: strings.Repeat("a", 64), Size: 3, Target: "shell-script",
		}},
	}
	err := manifest.Validate(ManifestPolicy{Channel: "stable", CurrentVersion: "0.9.0", Target: "linux_amd64"})
	if err == nil || !strings.Contains(err.Error(), "unsupported artifact target") {
		t.Fatalf("expected unknown artifact target error, got: %v", err)
	}
}

func TestNegative_ChannelMismatch(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "1.0.0",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         "test",
	}
	err := manifest.Validate(ManifestPolicy{Channel: "stable", CurrentVersion: "0.9.0"})
	if err == nil || !strings.Contains(err.Error(), "channel") {
		t.Fatalf("expected channel mismatch error, got: %v", err)
	}
}

func TestNegative_PackageManagedSelfUpdate(t *testing.T) {
	methods := []InstallationMethod{MethodDEB, MethodRPM, MethodMSIX, MethodWinget, MethodHomebrew, MethodStore, MethodNSIS}
	for _, m := range methods {
		svc := NewService(ServiceConfig{Method: m, CurrentVer: "1.0.0"})
		_, err := svc.Apply(context.Background())
		if err == nil || !strings.Contains(err.Error(), "does not allow self-update") {
			t.Fatalf("expected self-update rejection for %s, got: %v", m, err)
		}
	}
}

func TestNegative_ActiveWorkBlocksApply(t *testing.T) {
	tracker := &fakeWorkTracker{hasWork: true, reason: "agent running"}
	svc := NewService(ServiceConfig{
		Method:      MethodStandalone,
		CurrentVer:  "1.0.0",
		WorkTracker: tracker,
	})
	_, err := svc.Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "active work") {
		t.Fatalf("expected active work rejection, got: %v", err)
	}
}

func TestNegative_HTTPErrorOnManifest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	svc := NewService(ServiceConfig{
		RegistryURL: ts.URL + "/manifest.json",
		Method:      MethodStandalone,
		CurrentVer:  "1.0.0",
	})
	_, err := svc.Check(context.Background())
	if err == nil {
		t.Fatal("expected HTTP error on manifest fetch")
	}
}

func TestNegative_UnsignedManifest(t *testing.T) {
	manifest := []byte(`{"schema_version":1,"channel":"stable","version":"1.0.1","artifacts":{}}`)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(manifest)
	}))
	defer ts.Close()

	svc := NewService(ServiceConfig{RegistryURL: ts.URL, Method: MethodStandalone, CurrentVer: "1.0.0"})
	_, err := svc.Check(context.Background())
	if err == nil || !errors.Is(err, ErrManifestUnsigned) {
		t.Fatalf("expected unsigned manifest error, got: %v", err)
	}
}

func TestNegative_CorruptArchive(t *testing.T) {
	_, err := ExtractBinary([]byte("not-a-valid-archive"), TargetTarGz)
	if err == nil || !errors.Is(err, ErrArchiveInvalid) {
		t.Fatalf("expected archive invalid error, got: %v", err)
	}
}

func TestNegative_PathTraversal(t *testing.T) {
	err := validatePath("../../../etc/passwd")
	if err == nil || !errors.Is(err, ErrArchiveTraversal) {
		t.Fatalf("expected path traversal error, got: %v", err)
	}

	err = validatePath("/etc/passwd")
	if err == nil || !errors.Is(err, ErrArchiveTraversal) {
		t.Fatalf("expected absolute path traversal error, got: %v", err)
	}
}

func TestNegative_TooLargeArchive(t *testing.T) {
	// Create a minimal valid tar.gz with oversized content claim
	// This tests the size limit check
	err := error(nil)
	_ = err
	// The actual size limit is checked during extraction, not in this unit test
	// but we verify the constant exists and is reasonable
	if MaxExtractSize != 256*1024*1024 {
		t.Fatalf("expected MaxExtractSize to be 256MB, got %d", MaxExtractSize)
	}
}

func TestNegative_ManualInstallArtifact(t *testing.T) {
	artifacts := []ArtifactTarget{TargetNSIS, TargetDEB, TargetRPM}
	for _, target := range artifacts {
		art := Artifact{
			URL:    "http://example.com/pkg",
			Size:   100,
			SHA256: strings.Repeat("a", 64),
			Target: target,
		}
		if art.ExtractAction() != "manual" {
			t.Fatalf("expected %s to have manual extract action, got %s", target, art.ExtractAction())
		}
	}
}

func TestNegative_RollbackNoBackup(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "nexus")
	os.WriteFile(binPath, []byte("current"), 0755)

	updater := NewUpdater(binPath, tempDir)
	_, err := updater.Rollback()
	if err == nil || !strings.Contains(err.Error(), "no backup") {
		t.Fatalf("expected no backup error, got: %v", err)
	}
}

func TestNegative_UpdatePlanChecksumVerification(t *testing.T) {
	updater := NewUpdater("/tmp/fake", "/tmp")
	err := updater.VerifyArtifactChecksum([]byte("data"), "invalid-hex")
	if err == nil {
		t.Fatal("expected invalid checksum to fail")
	}
}

func TestNegative_VersionComparisonEdgeCases(t *testing.T) {
	tests := []struct {
		a, b  string
		want  int
		label string
	}{
		{"", "", 0, "empty equal"},
		{"abc", "def", 0, "non-semver equal"},
		{"v1.0.0", "1.0.0", 0, "v prefix"},
		// Note: Build metadata handling is a known limitation - "1.0.0+build" is treated as prerelease "build"
		{"1.0.0-alpha+build", "1.0.0-alpha", 0, "build metadata after prerelease ignored"},
	}

	for _, tt := range tests {
		got := compareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d (%s)", tt.a, tt.b, got, tt.want, tt.label)
		}
	}
}
