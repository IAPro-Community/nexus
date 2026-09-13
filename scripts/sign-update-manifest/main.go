package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type artifact struct {
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Target string `json:"target,omitempty"`
}

type manifest struct {
	SchemaVersion int                 `json:"schema_version"`
	Channel       string              `json:"channel"`
	Version       string              `json:"version"`
	ReleaseDate   string              `json:"release_date"`
	ExpiresAt     string              `json:"expires_at"`
	KeyID         string              `json:"key_id"`
	Artifacts     map[string]artifact `json:"artifacts"`
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func main() {
	var dirs stringList
	flag.Var(&dirs, "dist", "artifact directory (repeatable; default: dist)")
	version := flag.String("version", "", "release version")
	keyID := flag.String("key-id", "", "trusted public-key identifier")
	privateKey := flag.String("private-key", "", "base64 or hex Ed25519 private key")
	publicKey := flag.String("public-key", os.Getenv("NEXUS_UPDATE_PUBLIC_KEY"), "hex Ed25519 public key corresponding to the private key")
	baseURL := flag.String("base-url", "", "absolute release artifact base URL")
	flag.Parse()
	if *version == "" || *keyID == "" || *privateKey == "" || *publicKey == "" {
		fmt.Fprintln(os.Stderr, "version, key-id, private-key and public-key are required")
		os.Exit(2)
	}
	if len(dirs) == 0 {
		dirs = stringList{"dist"}
	}
	if *baseURL == "" {
		*baseURL = "https://github.com/kivervinicius/ai-cli/releases/download/v" + strings.TrimPrefix(*version, "v")
	}
	parsedBase, err := url.Parse(*baseURL)
	if err != nil || parsedBase.Scheme != "https" || parsedBase.Host == "" {
		panic("base-url must be an absolute https URL")
	}
	base := strings.TrimRight(*baseURL, "/")
	key, err := decodePrivateKey(*privateKey)
	if err != nil {
		panic(err)
	}
	pub, err := decodePublicKey(*publicKey)
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(pub, key.Public().(ed25519.PublicKey)) {
		panic("public key does not match private key")
	}
	m := manifest{
		SchemaVersion: 1,
		Channel:       channel(*version),
		Version:       strings.TrimPrefix(*version, "v"),
		ReleaseDate:   time.Now().UTC().Format(time.RFC3339),
		ExpiresAt:     time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339),
		KeyID:         *keyID,
		Artifacts:     map[string]artifact{},
	}
	outDir := dirs[0]
	for _, dist := range dirs {
		entries, err := os.ReadDir(dist)
		if err != nil {
			panic(err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			lower := strings.ToLower(name)
			if strings.HasSuffix(lower, ".txt") || strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".sig") {
				continue
			}
			path := filepath.Join(dist, name)
			info, err := entry.Info()
			if err != nil {
				panic(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				panic(err)
			}
			art := artifact{
				URL:    base + "/" + url.PathEscape(name),
				Size:   info.Size(),
				SHA256: fmt.Sprintf("%x", sha256.Sum256(data)),
				Target: artifactTarget(name),
			}
			for _, keyName := range artifactKeys(name) {
				m.Artifacts[keyName] = art
			}
		}
	}
	bytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := writeSignedManifest(outDir, bytes, key); err != nil {
		panic(err)
	}
}

func writeSignedManifest(dist string, jsonBytes []byte, key ed25519.PrivateKey) error {
	// Sign exactly the bytes that are published. The trailing newline is part of
	// the manifest body and must be covered by Ed25519 verification.
	manifestBytes := append(append([]byte(nil), jsonBytes...), '\n')
	sig := ed25519.Sign(key, manifestBytes)
	if err := os.WriteFile(filepath.Join(dist, "update-manifest.json"), manifestBytes, 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dist, "update-manifest.sig"), []byte(hex.EncodeToString(sig)+"\n"), 0644)
}

func decodePrivateKey(value string) (ed25519.PrivateKey, error) {
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && len(decoded) == ed25519.PrivateKeySize {
		return ed25519.PrivateKey(decoded), nil
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key must be base64 or hex Ed25519 key")
	}
	return ed25519.PrivateKey(decoded), nil
}

func decodePublicKey(value string) (ed25519.PublicKey, error) {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil || len(decoded) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key must be a hex Ed25519 key")
	}
	return ed25519.PublicKey(decoded), nil
}

func channel(version string) string {
	if strings.Contains(strings.ToLower(version), "beta") {
		return "beta"
	}
	return "stable"
}

func artifactTarget(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		return "tar.gz"
	case strings.HasSuffix(lower, ".zip"):
		return "zip"
	case strings.HasSuffix(lower, ".exe") && strings.Contains(lower, "setup"):
		return "nsis"
	case strings.HasSuffix(lower, ".deb"):
		return "deb"
	case strings.HasSuffix(lower, ".rpm"):
		return "rpm"
	default:
		return "binary"
	}
}

// artifactKeys returns stable lookup keys for installers and nexus update.
// Primary key is goos_goarch (or desktop_/package variants). A filename slug is
// also registered so install.sh can resolve by archive basename.
func artifactKeys(name string) []string {
	lower := strings.ToLower(name)
	slug := strings.ReplaceAll(strings.TrimSuffix(strings.TrimSuffix(lower, ".tar.gz"), filepath.Ext(lower)), "-", "_")
	keys := []string{slug}

	osArch, kind := classifyArtifact(lower)
	if osArch == "" {
		return uniqueKeys(keys)
	}
	switch kind {
	case "cli":
		keys = append(keys, osArch)
	case "desktop":
		keys = append(keys, "desktop_"+osArch)
	case "nsis":
		keys = append(keys, osArch+"_nsis", "windows_amd64_nsis")
	case "deb":
		keys = append(keys, osArch+"_deb")
	case "rpm":
		keys = append(keys, osArch+"_rpm")
	}
	return uniqueKeys(keys)
}

func classifyArtifact(lowerName string) (osArch string, kind string) {
	osName, arch := detectOSArch(lowerName)
	if osName == "" || arch == "" {
		return "", ""
	}
	osArch = osName + "_" + arch
	switch {
	case strings.Contains(lowerName, "nexus-setup") || (strings.HasSuffix(lowerName, ".exe") && strings.Contains(lowerName, "setup")):
		return osArch, "nsis"
	case strings.HasSuffix(lowerName, ".deb"):
		return osArch, "deb"
	case strings.HasSuffix(lowerName, ".rpm"):
		return osArch, "rpm"
	case strings.Contains(lowerName, "nexus-desktop"):
		return osArch, "desktop"
	default:
		return osArch, "cli"
	}
}

func detectOSArch(lowerName string) (osName, arch string) {
	switch {
	case strings.Contains(lowerName, "linux"):
		osName = "linux"
	case strings.Contains(lowerName, "darwin") || strings.Contains(lowerName, "macos"):
		osName = "darwin"
	case strings.Contains(lowerName, "windows") || strings.Contains(lowerName, "win64") || strings.Contains(lowerName, "win32"):
		osName = "windows"
	}
	switch {
	case strings.Contains(lowerName, "x86_64") || strings.Contains(lowerName, "amd64") || strings.Contains(lowerName, "x64"):
		arch = "amd64"
	case strings.Contains(lowerName, "arm64") || strings.Contains(lowerName, "aarch64"):
		arch = "arm64"
	case strings.Contains(lowerName, "i386") || strings.Contains(lowerName, "386"):
		arch = "386"
	}
	return osName, arch
}

func uniqueKeys(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, k := range in {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}
