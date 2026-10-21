package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/buildinfo"
)

// Service provides a unified update engine across CLI, Web, and Desktop.
type Service struct {
	mu          sync.RWMutex
	keyRing     *KeyRing
	registryURL string
	httpClient  *http.Client
	currentVer  string
	currentSHA  string
	channel     string
	execPath    string
	method      InstallationMethod
	workTracker WorkTracker
}

// WorkTracker reports whether there is active execution that shouldn't be abruptly killed.
type WorkTracker interface {
	HasActiveWork() (bool, string)
}

// ServiceConfig configures the unified Update Service.
type ServiceConfig struct {
	RegistryURL string
	KeyRing     *KeyRing
	CurrentVer  string
	CurrentSHA  string
	Channel     string
	ExecPath    string
	Method      InstallationMethod
	HTTPClient  *http.Client
	WorkTracker WorkTracker
}

// CheckResult details the availability and metadata of an update.
type CheckResult struct {
	CurrentVersion     string             `json:"current_version"`
	LatestVersion      string             `json:"latest_version"`
	LatestSHA          string             `json:"latest_sha,omitempty"`
	Channel            string             `json:"channel"`
	InstallationMethod InstallationMethod `json:"installation_method"`
	UpdateAvailable    bool               `json:"update_available"`
	AllowsSelfUpdate   bool               `json:"allows_self_update"`
	Instruction        string             `json:"instruction,omitempty"`
	Changelog          string             `json:"changelog,omitempty"`
	Target             string             `json:"target,omitempty"`
	TargetArtifact     *Artifact          `json:"target_artifact,omitempty"`
	InstallerArtifact  *Artifact          `json:"installer_artifact,omitempty"`
	ActiveWork         bool               `json:"active_work"`
	ActiveWorkReason   string             `json:"active_work_reason,omitempty"`
}

// NewService creates a new unified Update Service.
func NewService(cfg ServiceConfig) *Service {
	ver := cfg.CurrentVer
	if ver == "" {
		ver = buildinfo.Version
	}
	currentSHA := cfg.CurrentSHA
	if currentSHA == "" {
		currentSHA = buildinfo.Commit
	}
	ch := cfg.Channel
	if ch == "" {
		ch = channelForVersion(ver)
	}
	execP := cfg.ExecPath
	if execP == "" {
		if p, err := os.Executable(); err == nil {
			execP = p
		}
	}
	meth := cfg.Method
	if meth == "" {
		meth = DetectInstallationMethod(execP)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	regURL := ResolveRegistryURL(cfg.RegistryURL, ch)
	kr := cfg.KeyRing
	if kr == nil {
		kr = NewKeyRing()
	}

	return &Service{
		keyRing:     kr,
		registryURL: regURL,
		httpClient:  client,
		currentVer:  ver,
		currentSHA:  currentSHA,
		channel:     ch,
		execPath:    execP,
		method:      meth,
		workTracker: cfg.WorkTracker,
	}
}

func channelForVersion(version string) string {
	lower := strings.ToLower(strings.TrimSpace(version))
	if strings.Contains(lower, "-alpha.") ||
		strings.Contains(lower, "-beta.") ||
		strings.Contains(lower, "-rc.") {
		return "beta"
	}
	return "stable"
}

// Check inspects the registry and evaluates update eligibility.
func (s *Service) Check(ctx context.Context) (*CheckResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := &CheckResult{
		CurrentVersion:     s.currentVer,
		Channel:            s.channel,
		InstallationMethod: s.method,
		AllowsSelfUpdate:   s.method.AllowsSelfUpdate(),
		Instruction:        s.method.UpgradeInstruction(),
	}

	if s.workTracker != nil {
		hasWork, reason := s.workTracker.HasActiveWork()
		res.ActiveWork = hasWork
		res.ActiveWorkReason = reason
	}

	target := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	res.Target = target

	manifest, err := s.fetchManifest(ctx)
	if err != nil {
		res.LatestVersion = s.currentVer
		return res, err
	}

	res.LatestVersion = manifest.Version
	res.LatestSHA = manifest.SourceSHA
	res.Changelog = manifest.Changelog

	policy := ManifestPolicy{
		Channel:        s.channel,
		CurrentVersion: s.currentVer,
		Target:         target,
		Now:            time.Now().UTC(),
	}

	if err := manifest.Validate(policy); err != nil {
		if errors.Is(err, ErrManifestDowngrade) {
			res.UpdateAvailable = false
			return res, nil
		}
		return res, fmt.Errorf("manifest validation failed: %w", err)
	}

	if compareVersions(manifest.Version, s.currentVer) > 0 ||
		(s.channel == "beta" && compareVersions(manifest.Version, s.currentVer) == 0 &&
			isNewerSignedBetaSHA(s.currentSHA, manifest.SourceSHA)) {
		res.UpdateAvailable = true
		art := manifest.Artifacts[target]
		res.TargetArtifact = &art

		// Native Desktop/HUD upgrades must go through their platform installer.
		// The primary goos_goarch artifact is the CLI and must not replace a
		// running GUI executable. These optional package entries are only
		// considered after the full signed manifest has been verified.
		var installerKey string
		switch runtime.GOOS {
		case "windows":
			installerKey = target + "_nsis"
		case "darwin":
			installerKey = target + "_pkg"
		case "linux":
			installerKey = target + "_deb"
		}
		if installer, ok := manifest.Artifacts[installerKey]; ok &&
			installer.ExtractAction() == "manual" && installer.URL != "" &&
			installer.Size > 0 && len(installer.SHA256) == 64 {
			if _, err := hex.DecodeString(installer.SHA256); err == nil {
				res.InstallerArtifact = &installer
			}
		}
	}

	return res, nil
}

// isNewerSignedBetaSHA recognizes rolling beta builds with the same semantic
// version. A known 7+ character build commit and a verified full manifest
// object ID are required. Local "dev"/"unknown" binaries do not qualify.
func isNewerSignedBetaSHA(current, signed string) bool {
	current = strings.ToLower(strings.TrimSpace(current))
	signed = strings.ToLower(strings.TrimSpace(signed))
	if len(current) < 7 || (len(signed) != 40 && len(signed) != 64) ||
		current == "unknown" || current == "dev" {
		return false
	}
	// Git's --short output often has an odd number of hexadecimal digits.
	for _, digit := range current {
		if !strings.ContainsRune("0123456789abcdef", digit) {
			return false
		}
	}
	if _, err := hex.DecodeString(signed); err != nil {
		return false
	}
	return !strings.HasPrefix(signed, current) && !strings.HasPrefix(current, signed)
}

// Apply downloads, verifies, and installs the update when permitted by InstallationMethod.
func (s *Service) Apply(ctx context.Context) (*Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.method.AllowsSelfUpdate() {
		return nil, fmt.Errorf("installation method %s does not allow self-update; use %s", s.method, s.method.UpgradeInstruction())
	}

	if s.workTracker != nil {
		hasWork, reason := s.workTracker.HasActiveWork()
		if hasWork {
			return nil, fmt.Errorf("cannot apply update: active work in progress (%s)", reason)
		}
	}

	manifest, err := s.fetchManifest(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	target := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	policy := ManifestPolicy{
		Channel:        s.channel,
		CurrentVersion: s.currentVer,
		Target:         target,
		Now:            time.Now().UTC(),
	}

	if err := manifest.Validate(policy); err != nil {
		return nil, fmt.Errorf("manifest policy rejected: %w", err)
	}

	art, ok := manifest.Artifacts[target]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrManifestTarget, target)
	}

	// Check if artifact requires manual installation
	if art.ExtractAction() == "manual" {
		return nil, fmt.Errorf("artifact type %s requires manual installation: %s", art.Target, s.method.UpgradeInstruction())
	}

	data, err := s.downloadArtifact(ctx, art)
	if err != nil {
		return nil, err
	}

	// Manifest size/sha256 bind the downloaded blob (archive or raw binary).
	if art.Size != int64(len(data)) {
		return nil, fmt.Errorf("artifact size mismatch: expected %d, got %d", art.Size, len(data))
	}
	if err := NewUpdater(s.execPath, "").VerifyArtifactChecksum(data, art.SHA256); err != nil {
		return nil, err
	}

	// Extract binary if archive format
	var binaryData []byte
	if art.Target == TargetTarGz || art.Target == TargetZip {
		extracted, err := ExtractBinary(data, art.Target)
		if err != nil {
			return nil, fmt.Errorf("archive extraction failed: %w", err)
		}
		binaryData = extracted.Data
	} else {
		binaryData = data
	}

	updater := NewUpdater(s.execPath, dataDirForUpdates())
	sum := sha256.Sum256(binaryData)
	return updater.ApplyUpdate(s.currentVer, manifest.Version, binaryData, hex.EncodeToString(sum[:]))
}

// downloadArtifact streams the artifact to disk without loading entire body into memory.
func (s *Service) downloadArtifact(ctx context.Context, art Artifact) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, art.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid artifact URL: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading artifact failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("artifact download returned HTTP %d", resp.StatusCode)
	}

	// Stream with size limit
	limitedReader := io.LimitReader(resp.Body, MaxExtractSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("reading artifact body failed: %w", err)
	}

	if int64(len(data)) > MaxExtractSize {
		return nil, ErrArchiveTooLarge
	}

	return data, nil
}

func (s *Service) fetchManifest(ctx context.Context) (*Manifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.registryURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching update manifest failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("no update manifest published for channel %q yet (registry %s returned HTTP 404)", s.channel, s.registryURL)
		}
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	sig := resp.Header.Get("X-Signature-Ed25519")
	if sig == "" {
		sig = resp.Header.Get("X-Nexus-Signature")
	}

	if sig == "" {
		// Release artifacts publish the manifest and its detached signature
		// side-by-side. This fallback keeps the same trust chain usable by a
		// static registry, while still rejecting the manifest if the sidecar
		// is missing, malformed, or cannot be fetched.
		signatureURL, err := detachedSignatureURL(s.registryURL)
		if err != nil {
			return nil, ErrManifestUnsigned
		}
		sig, err = s.fetchDetachedSignature(ctx, signatureURL)
		if err != nil {
			return nil, ErrManifestUnsigned
		}
	}

	return s.keyRing.VerifyManifest(body, sig)
}

func detachedSignatureURL(manifestURL string) (string, error) {
	u, err := url.Parse(manifestURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid manifest URL")
	}
	name := path.Base(u.Path)
	if name == "." || name == "/" || name == "" {
		return "", fmt.Errorf("manifest URL has no file name")
	}
	ext := path.Ext(name)
	if ext == "" {
		name += ".sig"
	} else {
		name = strings.TrimSuffix(name, ext) + ".sig"
	}
	u.Path = path.Join(path.Dir(u.Path), name)
	return u.String(), nil
}

func (s *Service) fetchDetachedSignature(ctx context.Context, signatureURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, signatureURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("signature registry returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func dataDirForUpdates() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return dir
	}
	return filepath.Join(os.TempDir(), "nexus-update")
}
