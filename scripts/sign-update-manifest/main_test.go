package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteSignedManifestSignsPublishedBytes(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dist := t.TempDir()
	jsonBytes := []byte(`{"schema_version":1,"channel":"stable","version":"1.0.0"}`)

	if err := writeSignedManifest(dist, jsonBytes, privateKey); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(dist, "update-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	signatureText, err := os.ReadFile(filepath.Join(dist, "update-manifest.sig"))
	if err != nil {
		t.Fatal(err)
	}
	signature, err := hex.DecodeString(strings.TrimSpace(string(signatureText)))
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(publicKey, manifestBytes, signature) {
		t.Fatal("signature does not cover the published manifest bytes")
	}
}

func TestArtifactKeysMapGoReleaserAndDesktopNames(t *testing.T) {
	cases := map[string][]string{
		"nexus_Linux_x86_64.tar.gz":           {"linux_amd64"},
		"nexus_Darwin_arm64.tar.gz":           {"darwin_arm64"},
		"nexus_Windows_x86_64.zip":            {"windows_amd64"},
		"nexus-desktop_Linux_x86_64.tar.gz":   {"desktop_linux_amd64"},
		"nexus-desktop_Darwin_arm64.zip":      {"desktop_darwin_arm64"},
		"nexus-desktop_Windows_x86_64.zip":    {"desktop_windows_amd64"},
		"nexus-hud_Linux_x86_64.tar.gz":       {"hud_linux_amd64"},
		"nexus-hud_Darwin_arm64.zip":          {"hud_darwin_arm64"},
		"nexus-hud_Windows_x86_64.zip":        {"hud_windows_amd64"},
		"nexus-setup-Windows_x86_64.exe":      {"windows_amd64_nsis"},
		"nexus-installer_Darwin_arm64.pkg":    {"darwin_arm64_pkg"},
		"nexus_0.5.0-beta.23_linux_amd64.deb": {"linux_amd64_deb"},
		"nexus_0.5.0-beta.23_linux_amd64.rpm": {"linux_amd64_rpm"},
	}
	for name, mustContain := range cases {
		keys := artifactKeys(name)
		for _, want := range mustContain {
			found := false
			for _, got := range keys {
				if got == want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("%s keys=%v missing %s", name, keys, want)
			}
		}
		if artifactTarget(name) == "" {
			t.Fatalf("%s missing target", name)
		}
	}
}

func TestHUDArtifactKeysDoNotCollideWithCLI(t *testing.T) {
	cases := map[string]string{
		"nexus-hud_Linux_x86_64.tar.gz": "linux_amd64",
		"nexus-hud_Darwin_arm64.zip":     "darwin_arm64",
		"nexus-hud_Windows_x86_64.zip":   "windows_amd64",
	}
	for name, forbidden := range cases {
		for _, key := range artifactKeys(name) {
			if key == forbidden {
				t.Fatalf("%s must not claim CLI manifest key %s: %v", name, forbidden, artifactKeys(name))
			}
		}
	}
}

func TestAllowedArtifactBaseURL(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"https://github.com/IAPro-Community/nexus-releases/releases/download/v1.0.0", true},
		{"http://127.0.0.1:8080/v1.0.0", true},
		{"http://localhost:8080/v1.0.0", true},
		{"http://[::1]:8080/v1.0.0", true},
		{"http://example.com/v1.0.0", false},
		{"ftp://127.0.0.1/v1.0.0", false},
		{"/relative/path", false},
	}
	for _, tc := range cases {
		u, err := url.Parse(tc.raw)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.raw, err)
		}
		if got := allowedArtifactBaseURL(u); got != tc.want {
			t.Errorf("allowedArtifactBaseURL(%q)=%v, want %v", tc.raw, got, tc.want)
		}
	}
}

func TestReleaseChannelClassification(t *testing.T) {
	cases := map[string]string{
		"v1.0.0-alpha.1": "beta",
		"v1.0.0-beta.1":  "beta",
		"v1.0.0-rc.1":    "beta",
		"v1.0.0":         "stable",
	}
	for version, want := range cases {
		if got := channel(version); got != want {
			t.Errorf("channel(%q)=%q, want %q", version, got, want)
		}
	}
}
