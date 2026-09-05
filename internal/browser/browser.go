package browser

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Open opens a URL using host browser utilities safely.
func Open(args []string) error {
	if len(args) == 0 {
		return nil
	}
	url := args[0]
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid URL %q", url)
	}

	// Try real system browsers and /usr/bin/xdg-open (bypassing self-referencing shim)
	candidates := []string{
		"/usr/bin/xdg-open",
		"/usr/bin/google-chrome",
		"/usr/bin/firefox",
		"/usr/bin/chromium",
		"/usr/bin/x-www-browser",
		"google-chrome",
		"firefox",
		"chromium",
	}
	for _, cmd := range candidates {
		if _, err := os.Stat(cmd); err == nil {
			c := exec.Command(cmd, url)
			if hostBus := os.Getenv("AI_HOST_DBUS_SESSION_BUS_ADDRESS"); hostBus != "" {
				c.Env = append(os.Environ(), "DBUS_SESSION_BUS_ADDRESS="+hostBus)
			}
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Start()
		}
		if path, err := exec.LookPath(cmd); err == nil && !strings.Contains(path, "ai-manager/runtime/bin") {
			c := exec.Command(path, url)
			if hostBus := os.Getenv("AI_HOST_DBUS_SESSION_BUS_ADDRESS"); hostBus != "" {
				c.Env = append(os.Environ(), "DBUS_SESSION_BUS_ADDRESS="+hostBus)
			}
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Start()
		}
	}

	fmt.Printf("\nPlease open the following authentication URL in your browser:\n\n  %s\n\n", url)
	return nil
}

// OpenAppWindow opens the given URL in a dedicated standalone desktop application window frame.
func OpenAppWindow(url string) error {
	if url == "" {
		return nil
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid URL %q", url)
	}

	userDataDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "ai-manager", "desktop-profile")
	_ = os.MkdirAll(userDataDir, 0755)

	chromeCandidates := []string{
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/brave-browser",
		"google-chrome",
		"chromium",
	}

	for _, cmd := range chromeCandidates {
		if _, err := os.Stat(cmd); err == nil {
			args := []string{
				fmt.Sprintf("--app=%s", url),
				fmt.Sprintf("--user-data-dir=%s", userDataDir),
				"--class=iapro-nexus",
				"--name=iapro-nexus",
			}
			c := exec.Command(cmd, args...)
			if hostBus := os.Getenv("AI_HOST_DBUS_SESSION_BUS_ADDRESS"); hostBus != "" {
				c.Env = append(os.Environ(), "DBUS_SESSION_BUS_ADDRESS="+hostBus)
			}
			return c.Start()
		}
	}

	return Open([]string{url})
}
