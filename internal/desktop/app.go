package desktop

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/IAPro-Community/nexus/internal/browser"
	"github.com/IAPro-Community/nexus/internal/nexus"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var ErrCapabilityUnavailable = errors.New("desktop capability unavailable")

// FileFilter specifies an extension filter for the file picker.
type FileFilter struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
}

// FilePickerOptions configures the native open file dialog.
type FilePickerOptions struct {
	Title       string       `json:"title"`
	DefaultPath string       `json:"defaultPath"`
	Filters     []FileFilter `json:"filters"`
}

// NotificationOptions defines payload for native desktop notifications.
type NotificationOptions struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	Icon   string `json:"icon"`
	Silent bool   `json:"silent"`
}

// NativeDialogHandler defines an interface for delegating native file and folder pickers.
type NativeDialogHandler interface {
	SelectDirectory(ctx context.Context, title string) (string, error)
	SelectFile(ctx context.Context, opts FilePickerOptions) (string, error)
	ShowNotification(opts NotificationOptions) error
}

// BootstrapInfo encapsulates loopback connection details and pre-seeded authentication
// for the desktop webview environment.
type BootstrapInfo struct {
	ServerURL    string `json:"serverUrl"`
	SessionToken string `json:"sessionToken"`
	CSRFToken    string `json:"csrfToken"`
}

// App is the Wails bridge struct exposed to the frontend.
// It contains ONLY native OS integration capabilities.
type App struct {
	mu            sync.RWMutex
	ctx           context.Context
	dialogHandler NativeDialogHandler
	capabilities  Capabilities
	windowManager *WindowStateManager
	bootstrap     BootstrapInfo
	voiceMu       sync.Mutex
	voiceCapture  *nativeVoiceCaptureState
}

// NewApp creates a new desktop App instance.
func NewApp(dialogHandler NativeDialogHandler, windowManager *WindowStateManager, bootstrap ...BootstrapInfo) *App {
	if windowManager == nil {
		windowManager = NewWindowStateManager("")
	}
	var b BootstrapInfo
	if len(bootstrap) > 0 {
		b = bootstrap[0]
	}
	return &App{
		dialogHandler: dialogHandler,
		capabilities:  DefaultCapabilities(),
		windowManager: windowManager,
		bootstrap:     b,
	}
}

// GetBootstrapInfo returns loopback connection details and pre-seeded auth tokens.
func (a *App) GetBootstrapInfo() BootstrapInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.bootstrap
}

// Startup is called by Wails when the application starts.
func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ctx = ctx
}

// Shutdown is called by Wails when the application terminates.
func (a *App) Shutdown(ctx context.Context) {
	a.cancelNativeVoiceCapture()
	nexus.Default().Shutdown()
}

// GetCapabilities returns the native platform capabilities.
func (a *App) GetCapabilities() Capabilities {
	a.mu.RLock()
	defer a.mu.RUnlock()

	capabilities := a.capabilities
	// Wails owns native dialogs once Startup has supplied a frontend context.
	// This keeps capability discovery honest for unit-test instances created
	// without a running desktop frontend.
	if a.ctx != nil {
		capabilities.FilePicker = true
		capabilities.FolderPicker = true
	}
	return capabilities
}

// SelectDirectory opens the native OS directory picker.
func (a *App) SelectDirectory(title string) (string, error) {
	a.mu.RLock()
	dh := a.dialogHandler
	ctx := a.ctx
	a.mu.RUnlock()

	if dh != nil {
		return dh.SelectDirectory(ctx, title)
	}
	if ctx != nil {
		return wailsruntime.OpenDirectoryDialog(ctx, wailsruntime.OpenDialogOptions{
			Title: title,
		})
	}

	// Fallback is retained for direct package consumers and tests outside Wails.
	return fallbackSelectDirectory(title)
}

// SelectFile opens the native OS file picker.
func (a *App) SelectFile(opts FilePickerOptions) (string, error) {
	a.mu.RLock()
	dh := a.dialogHandler
	ctx := a.ctx
	a.mu.RUnlock()

	if dh != nil {
		return dh.SelectFile(ctx, opts)
	}
	if ctx != nil {
		filters := make([]wailsruntime.FileFilter, 0, len(opts.Filters))
		for _, filter := range opts.Filters {
			filters = append(filters, wailsruntime.FileFilter{
				DisplayName: filter.Name,
				Pattern:     strings.Join(filter.Extensions, ";"),
			})
		}
		return wailsruntime.OpenFileDialog(ctx, wailsruntime.OpenDialogOptions{
			Title:            opts.Title,
			DefaultDirectory: opts.DefaultPath,
			Filters:          filters,
		})
	}

	return fallbackSelectFile(opts)
}

// ShowNotification triggers an OS native desktop notification.
func (a *App) ShowNotification(opts NotificationOptions) error {
	a.mu.RLock()
	dh := a.dialogHandler
	a.mu.RUnlock()

	if dh != nil {
		return dh.ShowNotification(opts)
	}

	return fallbackNotification(opts)
}

// OpenExternal opens a URL using the default system browser.
func (a *App) OpenExternal(url string) error {
	parsed, err := neturl(url)
	if err != nil {
		return err
	}
	url = parsed.String()

	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		return browser.OpenAppWindow(url)
	}
	return exec.Command(cmd, args...).Start()
}

// GetSystemTheme detects the OS dark/light mode preference.
func (a *App) GetSystemTheme() string {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
		if err == nil && strings.EqualFold(strings.TrimSpace(string(out)), "dark") {
			return "dark"
		}
		return "light"
	case "linux":
		if out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output(); err == nil {
			value := strings.ToLower(string(out))
			if strings.Contains(value, "prefer-dark") {
				return "dark"
			}
			if strings.Contains(value, "prefer-light") {
				return "light"
			}
		}
	}
	return "unknown"
}

func fallbackSelectDirectory(title string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		const script = `on run argv
POSIX path of (choose folder with prompt (item 1 of argv))
end run`
		out, err := exec.Command("osascript", "-e", script, "--", title).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	case "linux":
		if zenity, err := exec.LookPath("zenity"); err == nil {
			out, err := exec.Command(zenity, "--file-selection", "--directory", "--title", title).Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		}
	}
	return "", fmt.Errorf("%w: folder picker", ErrCapabilityUnavailable)
}

func fallbackSelectFile(opts FilePickerOptions) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		const script = `on run argv
POSIX path of (choose file with prompt (item 1 of argv))
end run`
		out, err := exec.Command("osascript", "-e", script, "--", opts.Title).Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	case "linux":
		if zenity, err := exec.LookPath("zenity"); err == nil {
			out, err := exec.Command(zenity, "--file-selection", "--title", opts.Title).Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(out)), nil
		}
	}
	return "", fmt.Errorf("%w: file picker", ErrCapabilityUnavailable)
}

func fallbackNotification(opts NotificationOptions) error {
	switch runtime.GOOS {
	case "darwin":
		const script = `on run argv
display notification (item 2 of argv) with title (item 1 of argv)
end run`
		return exec.Command("osascript", "-e", script, "--", opts.Title, opts.Body).Run()
	case "linux":
		if notifySend, err := exec.LookPath("notify-send"); err == nil {
			return exec.Command(notifySend, opts.Title, opts.Body).Run()
		}
	}
	return fmt.Errorf("%w: notifications", ErrCapabilityUnavailable)
}

func neturl(raw string) (*url.URL, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Host == "" {
		return nil, fmt.Errorf("invalid external URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("external URL scheme %q is not allowed", parsed.Scheme)
	}
	return parsed, nil
}
