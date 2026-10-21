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
	"io"
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

func TestChannelForVersionClassifiesPrereleases(t *testing.T) {
	cases := map[string]string{
		"1.0.0-alpha.1": "beta",
		"1.0.0-beta.1":  "beta",
		"1.0.0-rc.1":    "beta",
		"1.0.0":         "stable",
		"dev":           "stable",
	}
	for version, want := range cases {
		if got := channelForVersion(version); got != want {
			t.Errorf("channelForVersion(%q)=%q, want %q", version, got, want)
		}
	}
}

// comparisonTransport intercepts only GitHub's public ancestry lookup.
type comparisonTransport func(*http.Request) (*http.Response, error)

func (fn comparisonTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRollingBetaRequiresStrictCommitAncestry(t *testing.T) {
	current := strings.Repeat("b", 40)
	signed := strings.Repeat("a", 40)
	for _, tc := range []struct {
		name, status, body string
		want               bool
	}{
		{"new descendant", "ok", `{"status":"ahead","ahead_by":2,"behind_by":0}`, true},
		{"old ancestor", "ok", `{"status":"behind","ahead_by":0,"behind_by":2}`, false},
		{"diverged branch", "ok", `{"status":"diverged","ahead_by":2,"behind_by":1}`, false},
		{"identical commit", "ok", `{"status":"identical","ahead_by":0,"behind_by":0}`, false},
		{"ambiguous ahead", "ok", `{"status":"ahead","ahead_by":1,"behind_by":1}`, false},
		{"invalid JSON", "ok", `{`, false},
		{"API failure", "unavailable", `{"status":"ahead","ahead_by":1,"behind_by":0}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: comparisonTransport(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "api.github.com" ||
					!strings.Contains(req.URL.Path, "/compare/"+current+"..."+signed) {
					t.Errorf("unexpected ancestry destination %q", req.URL.String())
				}
				status := http.StatusOK
				if tc.status == "unavailable" {
					status = http.StatusServiceUnavailable
				}
				return &http.Response{
					StatusCode: status,
					Body:       io.NopCloser(strings.NewReader(tc.body)),
					Header:     make(http.Header),
				}, nil
			})}
			service := NewService(ServiceConfig{HTTPClient: client})
			got := service.isNewerSignedBetaCommit(context.Background(), current, signed)
			if got != tc.want {
				t.Fatalf("isNewerSignedBetaCommit = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestRollingBetaRejectsUnknownAndIdenticalSHAsWithoutNetwork(t *testing.T) {
	signed := strings.Repeat("a", 40)
	svc := NewService(ServiceConfig{HTTPClient: &http.Client{Transport: comparisonTransport(
		func(*http.Request) (*http.Response, error) {
			t.Fatal("SHA validation must not contact GitHub")
			return nil, fmt.Errorf("unexpected request")
		},
	)}})
	for _, tc := range []struct {
		current, signed string
	}{
		{"aaaaaaa", signed},
		{signed, signed},
		{"dev", signed},
		{"unknown", signed},
		{"garbage", signed},
		{"abc", signed},
		{strings.Repeat("b", 40), "abcdef1"},
		{strings.Repeat("b", 40), ""},
	} {
		if svc.isNewerSignedBetaCommit(context.Background(), tc.current, tc.signed) {
			t.Fatalf("unverifiable same-version SHA must not trigger an update: %+v", tc)
		}
	}
}

func TestServiceApplyRefusesCurrentSignedRelease(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	target := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	binary := []byte("same-build")
	sum := sha256.Sum256(binary)
	sha := strings.Repeat("a", 40)
	manifest := Manifest{
		SchemaVersion: 1,
		Channel:       "beta",
		Version:       "1.0.0-beta.10",
		SourceSHA:     sha,
		KeyID:         "test-key",
		Artifacts: map[string]Artifact{
			target: {URL: "https://example.invalid/binary", Size: int64(len(binary)), SHA256: hex.EncodeToString(sum[:])},
		},
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	signature := hex.EncodeToString(ed25519.Sign(priv, body))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manifest.json" {
			t.Fatalf("no artifact should be downloaded for an existing release: %s", r.URL.Path)
		}
		w.Header().Set("X-Signature-Ed25519", signature)
		_, _ = w.Write(body)
	}))
	defer server.Close()
	keyring := NewKeyRing()
	keyring.AddKey("test-key", pub)
	svc := NewService(ServiceConfig{
		RegistryURL: server.URL + "/manifest.json",
		KeyRing: keyring,
		CurrentVer: manifest.Version,
		CurrentSHA: sha[:7],
		Channel: "beta",
		Method: MethodStandalone,
		ExecPath: filepath.Join(t.TempDir(), "nexus"),
	})
	if _, err := svc.Apply(context.Background()); !errors.Is(err, ErrNoUpdateAvailable) {
		t.Fatalf("same SHA must not install again, got %v", err)
	}
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
	// A signed manifest may provide a separate native installer for Desktop/HUD.
	// It must not be substituted for the CLI artifact used by Apply.
	var installerKey string
	var installerType ArtifactTarget
	switch runtime.GOOS {
	case "windows":
		installerKey, installerType = target+"_nsis", TargetNSIS
	case "darwin":
		installerKey, installerType = target+"_pkg", TargetPKG
	case "linux":
		installerKey, installerType = target+"_deb", TargetDEB
	}
	if installerKey != "" {
		manifest.Artifacts[installerKey] = Artifact{
			URL:    "https://github.com/IAPro-Community/nexus/releases/download/channel-stable/installer",
			Size:   int64(len(testData)),
			SHA256: shaHex,
			Target: installerType,
		}
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
	if installerKey != "" && (res.InstallerArtifact == nil ||
		res.InstallerArtifact.Target != installerType) {
		t.Fatalf("signed native installer not detected: %+v", res.InstallerArtifact)
	}
	if res.TargetArtifact == nil || res.TargetArtifact.Target != "" {
		t.Fatalf("CLI target must remain the CLI, not the native installer: %+v", res.TargetArtifact)
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
