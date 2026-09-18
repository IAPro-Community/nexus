package update

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type fakeWorkTracker struct {
	hasWork bool
	reason  string
}

func (f *fakeWorkTracker) HasActiveWork() (bool, string) {
	return f.hasWork, f.reason
}

func TestServiceCheckAndApply(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	target := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	testData := []byte("new-binary-content-v1.0.1")
	h := sha256.Sum256(testData)
	shaHex := hex.EncodeToString(h[:])

	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "stable",
		Version:       "1.0.1",
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:     time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		KeyID:         "test-key-1",
		Changelog:     "Fixed bugs and improved performance",
		Artifacts: map[string]Artifact{
			target: {
				Size:   int64(len(testData)),
				SHA256: shaHex,
			},
		},
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	sigBytes := ed25519.Sign(priv, manifestBytes)
	sigHex := hex.EncodeToString(sigBytes)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			w.Header().Set("X-Signature-Ed25519", sigHex)
			_, _ = w.Write(manifestBytes)
			return
		}
		if r.URL.Path == "/binary" {
			_, _ = w.Write(testData)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	// Update the URL in the manifest
	manifest.Artifacts[target] = Artifact{
		URL:    ts.URL + "/binary",
		Size:   int64(len(testData)),
		SHA256: shaHex,
	}
	manifestBytes, _ = json.Marshal(manifest)
	sigHex = hex.EncodeToString(ed25519.Sign(priv, manifestBytes))

	kr := NewKeyRing()
	kr.AddKey("test-key-1", pub)

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "nexus")

	svc := NewService(ServiceConfig{
		RegistryURL: ts.URL + "/manifest.json",
		KeyRing:     kr,
		CurrentVer:  "1.0.0",
		Channel:     "stable",
		ExecPath:    binaryPath,
		Method:      MethodStandalone,
	})

	res, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if !res.UpdateAvailable {
		t.Fatal("expected update to be available")
	}
	if res.LatestVersion != "1.0.1" {
		t.Fatalf("expected latest version 1.0.1, got %s", res.LatestVersion)
	}

	receipt, err := svc.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if receipt.Status != StatusSuccess {
		t.Fatalf("expected status success, got %s", receipt.Status)
	}
}

func TestServiceBlocksPackageManagedSelfUpdate(t *testing.T) {
	svc := NewService(ServiceConfig{
		CurrentVer: "1.0.0",
		Method:     MethodDEB,
	})

	_, err := svc.Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "does not allow self-update") {
		t.Fatalf("expected package-managed error, got %v", err)
	}
}

func TestServiceBlocksUpdateWhenActiveWorkPresent(t *testing.T) {
	tracker := &fakeWorkTracker{hasWork: true, reason: "agent 'agent-1' is running a mission"}
	svc := NewService(ServiceConfig{
		CurrentVer:  "1.0.0",
		Method:      MethodStandalone,
		WorkTracker: tracker,
	})

	_, err := svc.Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "active work in progress") {
		t.Fatalf("expected active work rejection, got %v", err)
	}
}

func TestServiceRejectsUnsignedManifest(t *testing.T) {
	manifest := []byte(`{"schema_version":1,"channel":"stable","version":"1.0.1","artifacts":{}}`)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(manifest)
	}))
	defer ts.Close()

	svc := NewService(ServiceConfig{RegistryURL: ts.URL, CurrentVer: "1.0.0", Method: MethodStandalone})
	_, err := svc.Check(context.Background())
	if !errors.Is(err, ErrManifestUnsigned) {
		t.Fatalf("expected unsigned manifest rejection, got %v", err)
	}
}

func TestServiceAcceptsDetachedManifestSignature(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	target := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	manifestValue := Manifest{
		SchemaVersion: 1,
		Channel:       "stable",
		Version:       "1.0.1",
		KeyID:         "detached-key",
		Artifacts: map[string]Artifact{
			target: {Size: 1, SHA256: strings.Repeat("a", 64)},
		},
	}
	manifest, err := json.Marshal(manifestValue)
	if err != nil {
		t.Fatal(err)
	}
	sig := hex.EncodeToString(ed25519.Sign(priv, manifest))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/update-manifest.json":
			_, _ = w.Write(manifest)
		case "/update-manifest.sig":
			_, _ = w.Write([]byte(sig + "\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	kr := NewKeyRing()
	kr.AddKey("detached-key", pub)
	svc := NewService(ServiceConfig{
		RegistryURL: ts.URL + "/update-manifest.json",
		KeyRing:     kr,
		CurrentVer:  "1.0.0",
		Method:      MethodStandalone,
	})
	if _, err := svc.Check(context.Background()); err != nil {
		t.Fatalf("detached signature should be accepted: %v", err)
	}
}
