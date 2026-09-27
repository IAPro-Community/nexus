package web

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/executil"
)

var tunnelURLRegex = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

const (
	cloudflaredBinDir  = "bin"
	cloudflaredBin     = "cloudflared"
	cloudflaredWinBin  = "cloudflared.exe"
	cloudflaredVersion = "2026.8.2"
	cloudflaredBase    = "https://github.com/cloudflare/cloudflared/releases/download/" + cloudflaredVersion
)

// cloudflaredChecksums contains known-good SHA-256 hashes for cloudflared binaries.
// Update when cloudflared releases a new version.
// Source: https://github.com/cloudflare/cloudflared/releases
var cloudflaredChecksums = map[string]string{
	// Official SHA-256 from https://github.com/cloudflare/cloudflared/releases/tag/2026.8.2
	"cloudflared-linux-amd64":       "fcfb02b575a52ca1af2e3267af4e1517bcdeb30ac48c834c69abaed3c0576ad2",
	"cloudflared-linux-arm64":       "7747d94570fb390cf47dcb4f9555c193c6355cda9793f0d878d9049e5d6a7790",
	"cloudflared-darwin-amd64.tgz":  "b0f770e1e0b281399a57219b840fd8eef1cc25387a404124248157ea2073727a",
	"cloudflared-darwin-arm64.tgz":  "b61054d3d6326ea558cb49826eebf5676e0d0a36d51b546975096ca3e0e3c89d",
	"cloudflared-windows-amd64.exe": "c29eee2b121f5436a642eed69fd9767da7e7b8c510fa50aaa130337f931357b5",
}

// Tunnel represents a running Cloudflare Quick Tunnel.
type Tunnel struct {
	URL     string
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	mu      sync.Mutex
	stopped bool
	// OnExit runs once after the cloudflared process exits (including crashes).
	// Callers use it to disarm tunnel auth mode when the public URL dies.
	OnExit func()
}

// Stop terminates the cloudflared process and waits for it to exit.
func (t *Tunnel) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return nil
	}
	t.stopped = true
	if t.cancel != nil {
		t.cancel()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		if err := t.cmd.Process.Kill(); err != nil {
			return err
		}
	}
	// Wait for process to be reaped (cmd.Wait called in background goroutine)
	return nil
}

// WaitForTunnel polls the tunnel URL until it's reachable or context expires.
// Call this AFTER the local web server is started.
func (t *Tunnel) WaitForTunnel(ctx context.Context) error {
	return waitForTunnelReady(ctx, t.URL)
}

func cloudflaredBinaryName() string {
	if runtime.GOOS == "windows" {
		return cloudflaredWinBin
	}
	return cloudflaredBin
}

func cloudflaredDir() (string, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, cloudflaredBinDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cloudflared directory: %w", err)
	}
	return dir, nil
}

func cloudflaredPath() (string, error) {
	dir, err := cloudflaredDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, cloudflaredBinaryName()), nil
}

func cloudflaredDownloadURL() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64":
	default:
		arch = runtime.GOARCH
	}
	if osName == "darwin" {
		// Official macOS assets are gzip tarballs, not bare binaries.
		return fmt.Sprintf("%s/cloudflared-darwin-%s.tgz", cloudflaredBase, arch)
	}
	suffix := ""
	if osName == "windows" {
		suffix = ".exe"
	}
	return fmt.Sprintf("%s/cloudflared-%s-%s%s", cloudflaredBase, osName, arch, suffix)
}

func cloudflaredChecksumKey() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64":
	default:
		return ""
	}
	if osName == "darwin" {
		return fmt.Sprintf("cloudflared-darwin-%s.tgz", arch)
	}
	name := fmt.Sprintf("cloudflared-%s-%s", osName, arch)
	if osName == "windows" {
		name += ".exe"
	}
	return name
}

func cloudflaredBinarySidecar(binPath string) string {
	return binPath + ".sha256"
}

// EnsureCloudflared checks if cloudflared is available and downloads it if not.
// Returns the path to the binary.
func EnsureCloudflared(ctx context.Context) (string, error) {
	// Prefer the Nexus-managed binary with SHA-256 verification. Unverified PATH
	// installs require an explicit opt-in (compromised cloudflared would own the
	// control-plane tunnel endpoint).
	binPath, err := cloudflaredPath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(binPath); err == nil {
		if verifyErr := verifyCloudflaredChecksum(binPath); verifyErr == nil {
			return binPath, nil
		}
		// Checksum mismatch — quarantine and re-download
		fmt.Fprintf(os.Stderr, "cloudflared checksum mismatch, re-downloading...\n")
		quarantine := binPath + ".quarantine"
		_ = os.Remove(quarantine)
		_ = os.Rename(binPath, quarantine)
	}

	if os.Getenv("NEXUS_CLOUDFLARED_ALLOW_PATH") == "1" {
		if path, err := exec.LookPath(cloudflaredBinaryName()); err == nil {
			slog.Warn("using PATH cloudflared without integrity check (NEXUS_CLOUDFLARED_ALLOW_PATH=1)", "path", path)
			return path, nil
		}
	}

	// Download managed binary
	fmt.Fprintf(os.Stderr, "Baixando cloudflared %s (%s/%s)...\n", cloudflaredVersion, runtime.GOOS, runtime.GOARCH)
	if err := downloadCloudflared(ctx, binPath); err != nil {
		return "", fmt.Errorf("failed to download cloudflared: %w", err)
	}

	return binPath, nil
}

// verifyCloudflaredChecksum checks the SHA-256 of a cached cloudflared binary
// against the pinned checksums for this GOOS/GOARCH (or a Darwin install sidecar).
func verifyCloudflaredChecksum(path string) error {
	if runtime.GOOS == "darwin" {
		// Darwin releases pin the .tgz digest, not the extracted binary. After
		// install we persist the extracted binary digest beside the file.
		raw, err := os.ReadFile(cloudflaredBinarySidecar(path))
		if err != nil {
			return fmt.Errorf("darwin cloudflared missing integrity sidecar: %w", err)
		}
		expected := strings.TrimSpace(string(raw))
		actual, err := fileSHA256(path)
		if err != nil {
			return err
		}
		if !strings.EqualFold(actual, expected) {
			return fmt.Errorf("cloudflared checksum mismatch: expected %s, got %s", expected, actual)
		}
		return nil
	}
	expected, ok := cloudflaredChecksums[cloudflaredChecksumKey()]
	if !ok {
		return fmt.Errorf("no checksum pinned for %s/%s; cannot verify integrity", runtime.GOOS, runtime.GOARCH)
	}
	actual, err := fileSHA256(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("cloudflared checksum mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func downloadCloudflared(ctx context.Context, destPath string) error {
	url := cloudflaredDownloadURL()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	mapKey := cloudflaredChecksumKey()
	if mapKey == "" {
		return fmt.Errorf("no checksum pinned for %s/%s; cannot verify integrity", runtime.GOOS, runtime.GOARCH)
	}
	expected, ok := cloudflaredChecksums[mapKey]
	if !ok {
		return fmt.Errorf("no checksum pinned for %s; cannot verify integrity. Update cloudflaredChecksums map", mapKey)
	}

	if runtime.GOOS == "darwin" {
		return downloadAndExtractDarwinCloudflared(resp.Body, destPath, expected)
	}

	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		f.Close()
		os.Remove(tmpPath)
	}()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), resp.Body); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("cloudflared checksum mismatch: expected %s, got %s", expected, actual)
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}
	return os.Rename(tmpPath, destPath)
}

func downloadAndExtractDarwinCloudflared(r io.Reader, destPath, expectedTGZ string) error {
	tmpTGZ := destPath + ".tgz.tmp"
	f, err := os.Create(tmpTGZ)
	if err != nil {
		return fmt.Errorf("failed to create temp archive: %w", err)
	}
	defer func() {
		f.Close()
		os.Remove(tmpTGZ)
	}()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), r); err != nil {
		return fmt.Errorf("failed to write archive: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expectedTGZ) {
		return fmt.Errorf("cloudflared tgz checksum mismatch: expected %s, got %s", expectedTGZ, actual)
	}

	af, err := os.Open(tmpTGZ)
	if err != nil {
		return err
	}
	defer af.Close()
	gz, err := gzip.NewReader(af)
	if err != nil {
		return fmt.Errorf("invalid cloudflared tgz: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	tmpBin := destPath + ".bin.tmp"
	found := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("corrupt cloudflared tgz: %w", err)
		}
		name := filepath.Base(hdr.Name)
		if hdr.Typeflag != tar.TypeReg || name != "cloudflared" {
			continue
		}
		out, err := os.OpenFile(tmpBin, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			os.Remove(tmpBin)
			return err
		}
		if err := out.Close(); err != nil {
			os.Remove(tmpBin)
			return err
		}
		found = true
		break
	}
	if !found {
		return fmt.Errorf("cloudflared binary not found inside darwin tgz")
	}
	sum, err := fileSHA256(tmpBin)
	if err != nil {
		os.Remove(tmpBin)
		return err
	}
	if err := os.WriteFile(cloudflaredBinarySidecar(destPath), []byte(sum+"\n"), 0600); err != nil {
		os.Remove(tmpBin)
		return err
	}
	if err := os.Rename(tmpBin, destPath); err != nil {
		os.Remove(tmpBin)
		return err
	}
	return nil
}

// waitForTunnelReady polls the tunnel URL until DNS propagates and it's reachable.
func waitForTunnelReady(ctx context.Context, tunnelURL string) error {
	// Extract hostname from URL
	hostname := strings.TrimPrefix(tunnelURL, "https://")
	hostname = strings.TrimSuffix(hostname, "/")
	if idx := strings.Index(hostname, "/"); idx >= 0 {
		hostname = hostname[:idx]
	}

	// Use external DNS resolvers for faster propagation check
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			// Try Cloudflare DNS first, then Google
			for _, dns := range []string{"1.1.1.1:53", "8.8.8.8:53"} {
				conn, err := d.DialContext(ctx, "udp", dns)
				if err == nil {
					return conn, nil
				}
			}
			return nil, fmt.Errorf("all DNS resolvers failed")
		},
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check DNS resolution via external resolver
			ips, err := resolver.LookupHost(ctx, hostname)
			if err != nil || len(ips) == 0 {
				continue
			}
			// DNS resolved, now try HTTP
			resp, err := client.Get(tunnelURL)
			if err == nil {
				resp.Body.Close()
				return nil
			}
		}
	}
}

// StartTunnel launches a Cloudflare Quick Tunnel pointing to the given local port.
// It blocks until the public URL is extracted or the context is canceled.
func StartTunnel(ctx context.Context, localPort int) (*Tunnel, error) {
	binPath, err := EnsureCloudflared(ctx)
	if err != nil {
		return nil, err
	}

	// The request context only governs startup. Once the tunnel URL has been
	// discovered, its lifetime belongs to Tunnel.Stop, not to the HTTP handler
	// that initiated it.
	tunnelCtx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(tunnelCtx, binPath, "tunnel",
		"--url", fmt.Sprintf("http://localhost:%d", localPort),
		"--no-autoupdate",
	)
	executil.HideConsoleWindow(cmd)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	tunnel := &Tunnel{
		cmd:    cmd,
		cancel: cancel,
	}

	// Parse stderr in background to find the public URL (one-shot)
	urlChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if match := tunnelURLRegex.FindString(line); match != "" {
				select {
				case urlChan <- match:
				default:
				}
			}
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			select {
			case errChan <- fmt.Errorf("error reading cloudflared output: %w", err):
			default:
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start cloudflared: %w", err)
	}

	// Wait for URL with timeout
	select {
	case url := <-urlChan:
		tunnel.URL = url
	case err := <-errChan:
		cancel()
		_ = cmd.Process.Kill()
		return nil, err
	case <-time.After(30 * time.Second):
		cancel()
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("timeout waiting for tunnel URL")
	case <-ctx.Done():
		cancel()
		_ = cmd.Process.Kill()
		return nil, ctx.Err()
	}

	// Reap the child process in background to avoid zombies, including when
	// cloudflared exits before the caller explicitly stops the tunnel.
	go func() {
		_ = cmd.Wait()
		stderr.Close()
		tunnel.mu.Lock()
		already := tunnel.stopped
		tunnel.stopped = true
		onExit := tunnel.OnExit
		tunnel.mu.Unlock()
		if !already && onExit != nil {
			onExit()
		}
	}()

	return tunnel, nil
}

// IsCloudflaredInstalled returns true if cloudflared is available.
func IsCloudflaredInstalled() bool {
	if path, err := exec.LookPath(cloudflaredBinaryName()); err == nil {
		return verifyCloudflaredChecksum(path) == nil
	}
	binPath, err := cloudflaredPath()
	if err != nil {
		return false
	}
	if _, err := os.Stat(binPath); err != nil {
		return false
	}
	return verifyCloudflaredChecksum(binPath) == nil
}

// CloudflaredVersion returns the version string if available.
func CloudflaredVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "version")
	executil.HideConsoleWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	firstLine := strings.SplitN(string(out), "\n", 2)
	if len(firstLine) > 0 {
		return strings.TrimSpace(firstLine[0])
	}
	return ""
}
