package doctor

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"sort"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/executil"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

type Status string

const (
	Pass    Status = "PASS"
	Warn    Status = "WARN"
	Fail    Status = "FAIL"
	Skipped Status = "SKIPPED"
)

type Check struct {
	ID          string `json:"id"`
	Status      Status `json:"status"`
	Summary     string `json:"summary"`
	Remediation string `json:"remediation,omitempty"`
}

type Report struct {
	Schema      string                            `json:"schema"`
	GeneratedAt time.Time                         `json:"generated_at"`
	Version     string                            `json:"version"`
	OS          string                            `json:"os"`
	Arch        string                            `json:"arch"`
	Checks      []Check                           `json:"checks"`
	Providers   map[string]model.DetectionResult  `json:"providers"`
	Credentials nexusruntime.CredentialCapability `json:"credentials"`
}

// BuildReport is read-only. It inspects existing state and never creates a
// project, registry row, profile, socket or temporary persistent directory.
func BuildReport(version string, detections map[string]model.DetectionResult, capability nexusruntime.CredentialCapability) Report {
	report := Report{
		Schema:      "nexus.doctor/v1",
		GeneratedAt: time.Now().UTC(),
		Version:     version,
		OS:          stdruntime.GOOS,
		Arch:        stdruntime.GOARCH,
		Providers:   detections,
		Credentials: capability,
	}
	report.Checks = append(report.Checks, checkDirectory("data_directory", config.DataDir), checkDirectory("config_directory", config.ConfigDir), checkDirectory("state_directory", config.StateDir))
	report.Checks = append(report.Checks, Check{ID: "credentials.capability", Status: capabilityStatus(capability.Status), Summary: string(capability.Status) + " — " + capability.Mechanism, Remediation: capability.Reason})

	inContainer := runningInContainer()
	if inContainer {
		report.Checks = append(report.Checks, Check{
			ID:      "runtime.container",
			Status:  Pass,
			Summary: "Docker/container compatibility layer detected (CLI + web headless)",
		})
	}

	// Platform runtime checks
	if inContainer {
		report.Checks = append(report.Checks,
			Check{ID: "platform.pty", Status: Pass, Summary: "Unix PTY available inside container"},
			Check{ID: "platform.webview2", Status: Skipped, Summary: "N/A in container — WebView2 is a native Windows Desktop dependency"},
			Check{ID: "platform.webkitgtk", Status: Skipped, Summary: "N/A in container — WebKitGTK is a native Desktop dependency"},
			Check{ID: "platform.conpty", Status: Skipped, Summary: "N/A in container — ConPTY is a native Windows terminal dependency"},
			Check{ID: "platform.wkwebview", Status: Skipped, Summary: "N/A in container — WKWebView is a native macOS Desktop dependency"},
		)
	} else {
		switch stdruntime.GOOS {
		case "windows":
			report.Checks = append(report.Checks,
				probeWindowsConPTY(),
				probeWindowsWebView2(),
			)
		case "darwin":
			report.Checks = append(report.Checks,
				Check{ID: "platform.pty", Status: Pass, Summary: "Unix PTY supported on macOS"},
				Check{ID: "platform.wkwebview", Status: Pass, Summary: "WKWebView is provided by macOS"},
			)
		case "linux":
			report.Checks = append(report.Checks,
				Check{ID: "platform.pty", Status: Pass, Summary: "Unix PTY supported on Linux"},
				probeLinuxWebKitGTK(),
			)
		}
	}

	desktopCheck := Check{
		ID:          "desktop.shell",
		Status:      Skipped,
		Summary:     "Wails Desktop shell requires a native desktop launch to verify",
		Remediation: "run the native desktop smoke test on the target operating system",
	}
	if inContainer {
		desktopCheck.Summary = "N/A in container — nexus-desktop/Wails is not part of the Docker compatibility layer"
		desktopCheck.Remediation = "use native install.sh/install.ps1/NSIS for Desktop, or nexus web in the browser"
	}
	report.Checks = append(report.Checks, desktopCheck)

	providerIDs := make([]string, 0, len(detections))
	for id := range detections {
		providerIDs = append(providerIDs, id)
	}
	sort.Strings(providerIDs)
	for _, id := range providerIDs {
		detection := detections[id]
		status := Pass
		if !detection.Installed {
			status = Warn
		}
		report.Checks = append(report.Checks, Check{ID: "provider." + id, Status: status, Summary: providerSummary(detection)})
	}
	return report
}

func runningInContainer() bool {
	// Explicit override for testing environments that are not containers but
	// have cgroup entries matching container patterns (e.g. GitHub Actions
	// Ubuntu runners where /proc/1/cgroup contains "docker" or "containerd").
	if os.Getenv("NEXUS_TEST_NOT_CONTAINER") == "1" {
		return false
	}
	if os.Getenv("NEXUS_DOCKER") == "1" || os.Getenv("NEXUS_COMPAT_DOCKER") == "1" {
		return true
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	body := string(data)
	return strings.Contains(body, "docker") || strings.Contains(body, "containerd") || strings.Contains(body, "/kubepods/")
}

func probeWindowsConPTY() Check {
	// ConPTY is an OS feature, but a CLI doctor cannot prove that the current
	// desktop process successfully created a pseudo-console. Report the
	// platform prerequisite rather than claiming a runtime smoke test passed.
	return Check{
		ID:          "platform.conpty",
		Status:      Warn,
		Summary:     "ConPTY platform prerequisite detected; native terminal smoke is not run by doctor",
		Remediation: "run the native Windows Desktop/SessionHost smoke test",
	}
}

func probeWindowsWebView2() Check {
	if _, err := exec.LookPath("reg.exe"); err != nil {
		return Check{ID: "platform.webview2", Status: Warn, Summary: "WebView2 registry probe is unavailable", Remediation: "install the Microsoft WebView2 Runtime or run doctor on Windows"}
	}
	out, err := executil.Command("reg.exe", "query", `HKLM\SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients`, "/s").CombinedOutput()
	if err == nil && hasWebView2Registration(string(out)) {
		return Check{ID: "platform.webview2", Status: Pass, Summary: "Microsoft WebView2 Runtime registration detected"}
	}
	return Check{ID: "platform.webview2", Status: Warn, Summary: "Microsoft WebView2 Runtime was not detected", Remediation: "install the Microsoft WebView2 Runtime before launching Desktop"}
}

func hasWebView2Registration(output string) bool {
	const runtimeName = "microsoft edge webview2 runtime"
	nameFound, versionFound := false, false
	flush := func() bool { return nameFound && versionFound }
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(line), "HKEY_") {
			if flush() {
				return true
			}
			nameFound, versionFound = false, false
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 || !strings.EqualFold(fields[1], "REG_SZ") {
			continue
		}
		value := strings.TrimSpace(strings.Join(fields[2:], " "))
		switch strings.ToLower(fields[0]) {
		case "name":
			nameFound = strings.EqualFold(value, runtimeName)
		case "pv":
			versionFound = value != ""
		}
	}
	return flush()
}

func probeLinuxWebKitGTK() Check {
	if _, err := exec.LookPath("pkg-config"); err != nil {
		return Check{ID: "platform.webkitgtk", Status: Warn, Summary: "pkg-config is unavailable; WebKitGTK cannot be probed", Remediation: "install GTK/WebKitGTK development/runtime packages or run doctor on the target desktop host"}
	}
	for _, pkg := range []string{"webkit2gtk-4.1", "webkit2gtk-4.0"} {
		if out, err := executil.Command("pkg-config", "--modversion", pkg).Output(); err == nil {
			version := strings.TrimSpace(string(out))
			return Check{ID: "platform.webkitgtk", Status: Pass, Summary: "WebKitGTK detected: " + pkg + " " + version}
		}
	}
	return Check{ID: "platform.webkitgtk", Status: Warn, Summary: "WebKitGTK 4.0/4.1 was not detected", Remediation: "install the WebKitGTK runtime required by Wails Desktop"}
}

func checkDirectory(id string, resolver func() (string, error)) Check {
	path, err := resolver()
	if err != nil || path == "" {
		return Check{ID: id, Status: Fail, Summary: "directory could not be resolved", Remediation: "inspect Nexus directory environment configuration"}
	}
	info, err := os.Stat(path)
	if err != nil {
		return Check{ID: id, Status: Warn, Summary: "directory is not present: " + path, Remediation: "start Nexus once or create the directory with secure permissions"}
	}
	if !info.IsDir() {
		return Check{ID: id, Status: Fail, Summary: "path is not a directory: " + path}
	}
	return Check{ID: id, Status: Pass, Summary: "directory is available: " + path}
}

func capabilityStatus(status nexusruntime.CredentialCapabilityStatus) Status {
	switch status {
	case nexusruntime.CredentialSupported:
		return Pass

	case nexusruntime.CredentialDegraded, nexusruntime.CredentialUnsupported:
		return Warn
	default:
		return Fail
	}
}

func providerSummary(detection model.DetectionResult) string {
	if !detection.Installed {
		return "not installed: " + detection.Error
	}
	return fmt.Sprintf("installed (%s)", detection.Version)
}

func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// WriteBundle writes only an allowlisted report. It intentionally does not
// archive environment variables, arguments, prompts, transcripts or provider
// logs.
func (r Report) WriteBundle(path string) error {
	data, err := r.JSON()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	archive := zip.NewWriter(file)
	entry, err := archive.Create("report.json")
	if err != nil {
		return err
	}
	if _, err := entry.Write(data); err != nil {
		return err
	}
	manifest, err := archive.Create("MANIFEST.txt")
	if err != nil {
		return err
	}
	if _, err := manifest.Write([]byte("report.json\n")); err != nil {
		return err
	}
	return archive.Close()
}
