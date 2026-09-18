package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
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
		"nexus-setup-Windows_x86_64.exe":      {"windows_amd64_nsis"},
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
