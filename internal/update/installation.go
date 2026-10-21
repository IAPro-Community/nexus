package update

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// InstallationMethod identifies how IAPro Nexus was installed on the system.
type InstallationMethod string

const (
	MethodStandalone InstallationMethod = "STANDALONE"
	MethodDesktop    InstallationMethod = "DESKTOP"
	MethodNSIS       InstallationMethod = "NSIS"
	MethodPKG        InstallationMethod = "PKG"
	MethodDEB        InstallationMethod = "DEB"
	MethodRPM        InstallationMethod = "RPM"
	MethodMSIX       InstallationMethod = "MSIX"
	MethodWinget     InstallationMethod = "WINGET"
	MethodHomebrew   InstallationMethod = "HOMEBREW"
	MethodStore      InstallationMethod = "STORE"
	MethodUnknown    InstallationMethod = "UNKNOWN"
)

// IsPackageManaged returns true if the installation is managed by an OS package manager
// or external installer, meaning in-place binary overwrite should be avoided in favor
// of delegating to the package manager or guiding the user.
func (m InstallationMethod) IsPackageManaged() bool {
	switch m {
	case MethodDEB, MethodRPM, MethodMSIX, MethodWinget, MethodHomebrew, MethodStore, MethodNSIS, MethodPKG, MethodDesktop:
		return true
	default:
		return false
	}
}

// AllowsSelfUpdate returns true if the Nexus core/CLI/desktop can safely replace its
// own binary directly via the UpdateService without breaking package manager integrity.
func (m InstallationMethod) AllowsSelfUpdate() bool {
	return m == MethodStandalone
}

// UpgradeInstruction returns the recommended CLI or package manager command to upgrade.
func (m InstallationMethod) UpgradeInstruction() string {
	switch m {
	case MethodDEB:
		return "sudo apt-get update && sudo apt-get --only-upgrade install nexus"
	case MethodRPM:
		return "sudo dnf upgrade nexus || sudo yum upgrade nexus"
	case MethodHomebrew:
		return "brew upgrade nexus"
	case MethodWinget:
		return "winget upgrade Nexus"
	case MethodNSIS:
		return "Download and run the latest Nexus Windows Setup installer."
	case MethodDesktop:
		return "Download the verified Nexus desktop installer from GitHub Releases and restart the app after installation."
	case MethodPKG:
		return "Download and run the latest notarized Nexus macOS PKG installer."
	case MethodMSIX, MethodStore:
		return "Updates are managed automatically by the application store."
	case MethodStandalone:
		return "nexus update"
	default:
		return "Consult your system administrator or download the latest release."
	}
}

// DetectInstallationMethod attempts to detect how the current binary was installed.
func DetectInstallationMethod(execPath string) InstallationMethod {
	if env := os.Getenv("NEXUS_INSTALL_METHOD"); env != "" {
		return InstallationMethod(strings.ToUpper(strings.TrimSpace(env)))
	}

	cleanPath := filepath.Clean(execPath)
	lowerPath := strings.ToLower(cleanPath)

	// A running Desktop/HUD process must never replace itself with the CLI
	// artifact selected by the updater's goos_goarch lookup. Even an unpacked
	// Wails executable has to update via its native bundle/installer.
	baseName := strings.ToLower(filepath.Base(cleanPath))
	if baseName == "nexus-desktop" || baseName == "nexus-desktop.exe" ||
		baseName == "nexus-hud" || baseName == "nexus-hud.exe" {
		return MethodDesktop
	}

	// Check marker files in the directory containing the binary
	dir := filepath.Dir(cleanPath)
	if _, err := os.Stat(filepath.Join(dir, ".nexus-nsis")); err == nil {
		return MethodNSIS
	}
	if _, err := os.Stat(filepath.Join(dir, ".nexus-pkg")); err == nil {
		return MethodPKG
	}
	if _, err := os.Stat(filepath.Join(dir, ".nexus-deb")); err == nil {
		return MethodDEB
	}
	if _, err := os.Stat(filepath.Join(dir, ".nexus-rpm")); err == nil {
		return MethodRPM
	}
	if _, err := os.Stat(filepath.Join(dir, ".nexus-standalone")); err == nil {
		return MethodStandalone
	}

	// Homebrew paths on macOS and Linux
	if strings.Contains(cleanPath, "/Cellar/nexus") ||
		strings.Contains(cleanPath, "/opt/homebrew/") ||
		strings.Contains(cleanPath, "/usr/local/Cellar/") {
		return MethodHomebrew
	}

	// Windows package / installer heuristics
	if runtime.GOOS == "windows" {
		if strings.Contains(lowerPath, "windowsapps") {
			return MethodMSIX
		}
		if strings.Contains(lowerPath, "programs\\nexus") ||
			strings.Contains(lowerPath, "program files\\nexus") ||
			strings.Contains(lowerPath, "programs\\iapro nexus") ||
			strings.Contains(lowerPath, "program files\\iapro nexus") {
			return MethodNSIS
		}
	}

	// Linux FHS standard paths often indicate distribution package installs
	if runtime.GOOS == "linux" {
		if strings.HasPrefix(cleanPath, "/usr/bin/") || strings.HasPrefix(cleanPath, "/usr/sbin/") || strings.HasPrefix(cleanPath, "/usr/local/bin/") {
			if _, err := os.Stat("/var/lib/dpkg/status"); err == nil {
				return MethodDEB
			}
			if _, err := os.Stat("/var/lib/rpm"); err == nil {
				return MethodRPM
			}
			return MethodDEB
		}
	}

	// Default user local bin installs (like ~/.local/bin/nexus) created by install.sh / install.ps1
	return MethodStandalone
}
