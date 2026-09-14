package browser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SelfIsBrowserHelper returns true when the current process is already a
// browser-helper shim (ai-browser / xdg-open / nexus-browser symlink to
// the nexus binary). In that case we must NOT look up "xdg-open" via
// exec.LookPath because the PATH includes the internal bin dir where
// "xdg-open" is a symlink back to us — doing so would cause infinite
// recursion and spawn hundreds of zombie processes.
//
// Exported so callers in desktop/, control/web/, etc. can gate their own
// xdg-open invocations without duplicating the detection logic.
func SelfIsBrowserHelper() bool {
	// Fast path: env var set by main.go browser-helper entry.
	if os.Getenv("NEXUS_BROWSER_HELPER") == "1" {
		return true
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	base := filepath.Base(exe)
	return base == "xdg-open" || base == "ai-browser" || base == "nexus-browser"
}

// ResolveXdgOpen returns the absolute path to the real system xdg-open.
// If the current process is a browser helper shim, it bypasses PATH
// lookup entirely and checks well-known system locations.  Returns ""
// if no suitable binary is found.
func ResolveXdgOpen() string {
	if !SelfIsBrowserHelper() {
		// Not a shim — safe to use normal PATH lookup.
		if p, err := exec.LookPath("xdg-open"); err == nil {
			return p
		}
		return ""
	}
	// Shim: skip PATH (it contains our symlink) and probe system dirs.
	for _, p := range []string{"/usr/bin/xdg-open", "/usr/local/bin/xdg-open"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// Open opens a URL using host browser utilities safely.
func Open(args []string) error {
	if len(args) == 0 {
		return nil
	}
	url := args[0]
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid URL %q", url)
	}

	isHelper := SelfIsBrowserHelper()

	// When running as a browser helper, skip "xdg-open" in the candidate
	// list to avoid the symlink-recursion loop.  Use absolute paths for
	// system browsers so we never accidentally resolve back to ourselves.
	if isHelper {
		for _, cmd := range []string{"google-chrome", "firefox", "chromium", "brave"} {
			if path, err := exec.LookPath(cmd); err == nil {
				return startBrowser(path, url)
			}
		}
		// Fallback: try the real system xdg-open via absolute paths.
		for _, p := range []string{"/usr/bin/xdg-open", "/usr/local/bin/xdg-open"} {
			if _, err := os.Stat(p); err == nil {
				return startBrowser(p, url)
			}
		}
	} else {
		// Normal path: not a helper shim, safe to use LookPath.
		for _, cmd := range []string{"xdg-open", "google-chrome", "firefox", "chromium", "brave"} {
			if path, err := exec.LookPath(cmd); err == nil {
				return startBrowser(path, url)
			}
		}
	}

	fmt.Printf("\nPlease open the following authentication URL in your browser:\n\n  %s\n\n", url)
	return nil
}

func startBrowser(path, url string) error {
	c := exec.Command(path, url)
	if hostBus := os.Getenv("AI_HOST_DBUS_SESSION_BUS_ADDRESS"); hostBus != "" {
		c.Env = append(os.Environ(), "DBUS_SESSION_BUS_ADDRESS="+hostBus)
	}
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Start()
}

// OpenAppWindow opens the URL in a dedicated Chromium-style app window when
// available, falling back to the normal browser open path.
func OpenAppWindow(url string) error {
	if url == "" {
		return nil
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid URL %q", url)
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}
	userDataDir := filepath.Join(home, ".local", "share", "ai-manager", "desktop-profile")
	// Cookies/tokens live here: owner-only permissions.
	_ = os.MkdirAll(userDataDir, 0700)
	_ = os.Chmod(userDataDir, 0700)

	chromeCandidates := []string{
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/brave-browser",
		"google-chrome",
		"chromium",
		"brave",
	}

	for _, cmd := range chromeCandidates {
		path := cmd
		if _, err := os.Stat(cmd); err != nil {
			looked, lookErr := exec.LookPath(cmd)
			if lookErr != nil {
				continue
			}
			path = looked
		}
		args := []string{
			fmt.Sprintf("--app=%s", url),
			fmt.Sprintf("--user-data-dir=%s", userDataDir),
			"--class=iapro-nexus",
			"--name=iapro-nexus",
		}
		c := exec.Command(path, args...)
		if hostBus := os.Getenv("AI_HOST_DBUS_SESSION_BUS_ADDRESS"); hostBus != "" {
			c.Env = append(os.Environ(), "DBUS_SESSION_BUS_ADDRESS="+hostBus)
		}
		if err := c.Start(); err == nil {
			return nil
		}
	}

	return Open([]string{url})
}
