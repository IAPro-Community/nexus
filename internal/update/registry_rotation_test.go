package update

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func signTestManifest(t *testing.T, priv ed25519.PrivateKey, keyID, version string) ([]byte, string) {
	t.Helper()
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       version,
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		KeyID:         keyID,
		Artifacts:     map[string]Artifact{"linux_amd64": {SHA256: "ab", Size: 2}},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return raw, hex.EncodeToString(ed25519.Sign(priv, raw))
}

// TestKeyRotationAcceptsSuccessorAndRevokesPredecessor proves the trust
// model supports rotation without a flag day: the successor verifies while
// both are trusted, and the predecessor stops verifying after revocation.
func TestKeyRotationAcceptsSuccessorAndRevokesPredecessor(t *testing.T) {
	pubOld, privOld := generateTestKeyPair(t)
	pubNew, privNew := generateTestKeyPair(t)
	ring := NewKeyRing()
	ring.AddKey("nexus-signing-key-2026-v1", pubOld)

	oldBytes, oldSig := signTestManifest(t, privOld, "nexus-signing-key-2026-v1", "1.0.0-beta.1")
	if _, err := ring.VerifyManifest(oldBytes, oldSig); err != nil {
		t.Fatalf("current key must verify: %v", err)
	}

	// Rollout phase: both keys trusted.
	ring.AddKey("nexus-signing-key-2026-v2", pubNew)
	newBytes, newSig := signTestManifest(t, privNew, "nexus-signing-key-2026-v2", "1.0.0-beta.2")
	if _, err := ring.VerifyManifest(newBytes, newSig); err != nil {
		t.Fatalf("successor must verify while trusted: %v", err)
	}
	if _, err := ring.VerifyManifest(oldBytes, oldSig); err != nil {
		t.Fatalf("predecessor must verify until revoked: %v", err)
	}

	// Completion phase: predecessor revoked.
	ring.RemoveKey("nexus-signing-key-2026-v1")
	if _, err := ring.VerifyManifest(newBytes, newSig); err != nil {
		t.Fatalf("successor must verify after rotation: %v", err)
	}
	_, err := ring.VerifyManifest(oldBytes, oldSig)
	if !errors.Is(err, ErrUntrustedKeyID) {
		t.Fatalf("revoked key must fail with ErrUntrustedKeyID, got %v", err)
	}
}

func TestArtifactRegistryBackends(t *testing.T) {
	gh := ResolveArtifactRegistry("", "beta")
	if _, ok := gh.(GitHubReleasesRegistry); !ok {
		t.Fatalf("default backend = %T, want GitHubReleasesRegistry", gh)
	}
	u, err := gh.ManifestURL("beta")
	if err != nil || u == "" {
		t.Fatalf("github manifest url: %q, %v", u, err)
	}
	if gh.BackendName() == "" {
		t.Fatal("backend name must be set")
	}

	t.Setenv("NEXUS_UPDATE_REGISTRY_URL", "https://cdn.example.invalid/beta/manifest.json")
	st := ResolveArtifactRegistry("", "beta")
	if _, ok := st.(StaticURLRegistry); !ok {
		t.Fatalf("env backend = %T, want StaticURLRegistry", st)
	}
	u, err = st.ManifestURL("beta")
	if err != nil || u != "https://cdn.example.invalid/beta/manifest.json" {
		t.Fatalf("static manifest url: %q, %v", u, err)
	}

	explicit := ResolveArtifactRegistry("https://explicit.invalid/m.json", "beta")
	u, err = explicit.ManifestURL("stable")
	if err != nil || u != "https://explicit.invalid/m.json" {
		t.Fatalf("explicit manifest url: %q, %v", u, err)
	}

	bad := StaticURLRegistry{URL: "nota-url"}
	if _, err := bad.ManifestURL("beta"); err == nil {
		t.Fatal("non-http static URL must fail")
	}
	empty := StaticURLRegistry{}
	if _, err := empty.ManifestURL("beta"); err == nil {
		t.Fatal("empty static URL must fail")
	}
}
