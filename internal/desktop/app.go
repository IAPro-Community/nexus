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
	controlnotify "github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/nexus"
	"github.com/wailsapp/wails/v3/pkg/application"
)

var ErrCapabilityUnavailable = errors.New("desktop capability unavailable")

const (
	DesktopWindowName = "main"
	HUDWindowName     = "hud"
)

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

// App is the Wails v3 service exposed to the frontend. Domain state remains in
// Nexus Core; this service owns native OS presentation only.
type App struct {
	mu            sync.RWMutex
	ctx           context.Context
	dialogHandler NativeDialogHandler
	capabilities  Capabilities
	windowManager *WindowStateManager
	windowName    string
	bootstrap     BootstrapInfo
	voiceMu       sync.Mutex
	voiceCapture  *nativeVoiceCaptureState
}

// NewApp preserves the test/direct-consumer constructor and targets the main
// desktop window.
func NewApp(dialogHandler NativeDialogHandler, windowManager *WindowStateManager, bootstrap ...BootstrapInfo) *App {
	return NewAppForWindow(dialogHandler, windowManager, DesktopWindowName, bootstrap...)
}

// NewAppForWindow creates a bridge bound to one named Wails v3 window.
func NewAppForWindow(dialogHandler NativeDialogHandler, windowManager *WindowStateManager, windowName string, bootstrap ...BootstrapInfo) *App {
	if windowManager == nil {
		windowManager = NewWindowStateManager("")
	}
	if strings.TrimSpace(windowName) == "" {
		windowName = DesktopWindowName
	}
	var b BootstrapInfo
	if len(bootstrap) > 0 {
		b = bootstrap[0]
	}
	return &App{
		dialogHandler: dialogHandler,
		capabilities:  DefaultCapabilities(),
		windowManager: windowManager,
		windowName:    windowName,
		bootstrap:     b,
	}
}

func (a *App) nativeWindow() (application.Window, bool) {
	app := application.Get()
	if app == nil {
		return nil, false
	}
	return app.Window.GetByName(a.windowName)
}

// ServiceStartup is Wails v3's lifecycle hook. It is not exposed as a binding.
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()
	return nil
}

// ServiceShutdown is Wails v3's lifecycle hook. It is not exposed as a binding.
func (a *App) ServiceShutdown() error {
	a.cancelNativeVoiceCapture()
	nexus.Default().Shutdown()
	return nil
}

// GetBootstrapInfo returns loopback connection details and pre-seeded auth tokens.
func (a *App) GetBootstrapInfo() BootstrapInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.bootstrap
}

// GetCapabilities returns capabilities proven by the current native shell.
func (a *App) GetCapabilities() Capabilities {
	a.mu.RLock()
	capabilities := a.capabilities
	a.mu.RUnlock()

	if application.Get() != nil {
		capabilities.FilePicker = true
		capabilities.FolderPicker = true
		capabilities.Tray = true
		capabilities.NativeMenus = true
		capabilities.AutoStart = runtime.GOOS == "linux" || runtime.GOOS == "windows" || runtime.GOOS == "darwin"
	}
	return capabilities
}

// SelectDirectory opens Wails v3's native OS directory picker.
func (a *App) SelectDirectory(title string) (string, error) {
	a.mu.RLock()
	dh := a.dialogHandler
	ctx := a.ctx
	a.mu.RUnlock()
	if dh != nil {
		return dh.SelectDirectory(ctx, title)
	}
	if app := application.Get(); app != nil {
		dialog := app.Dialog.OpenFile().
			SetTitle(title).
			CanChooseFiles(false).
			CanChooseDirectories(true).
			CanCreateDirectories(true)
		if window, ok := a.nativeWindow(); ok {
			dialog.AttachToWindow(window)
		}
		return dialog.PromptForSingleSelection()
	}
	return fallbackSelectDirectory(title)
}

// SelectFile opens Wails v3's native OS file picker.
func (a *App) SelectFile(opts FilePickerOptions) (string, error) {
	a.mu.RLock()
	dh := a.dialogHandler
	ctx := a.ctx
	a.mu.RUnlock()
	if dh != nil {
		return dh.SelectFile(ctx, opts)
	}
	if app := application.Get(); app != nil {
		dialog := app.Dialog.OpenFile().SetTitle(opts.Title).CanChooseFiles(true)
		if opts.DefaultPath != "" {
			dialog.SetDirectory(opts.DefaultPath)
		}
		for _, filter := range opts.Filters {
			dialog.AddFilter(filter.Name, strings.Join(filter.Extensions, ";"))
		}
		if window, ok := a.nativeWindow(); ok {
			dialog.AttachToWindow(window)
		}
		return dialog.PromptForSingleSelection()
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

// OpenExternal opens a validated HTTP(S) URL using the Wails v3 browser manager.
func (a *App) OpenExternal(raw string) error {
	parsed, err := neturl(raw)
	if err != nil {
		return err
	}
	if app := application.Get(); app != nil {
		return app.Browser.OpenURL(parsed.String())
	}

	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", parsed.String()}
	case "darwin":
		cmd = "open"
		args = []string{parsed.String()}
	default:
		return browser.OpenAppWindow(parsed.String())
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

func (a *App) IsMaximized() bool {
	if window, ok := a.nativeWindow(); ok {
		return window.IsMaximised()
	}
	return false
}

func (a *App) MinimizeWindow() {
	if window, ok := a.nativeWindow(); ok {
		window.Minimise() //nolint:misspell // Wails v3 API uses British spelling.
	}
}

func (a *App) MaximizeWindow() {
	if window, ok := a.nativeWindow(); ok {
		window.Maximise() //nolint:misspell // Wails v3 API uses British spelling.
	}
}

func (a *App) UnmaximizeWindow() {
	if window, ok := a.nativeWindow(); ok {
		window.UnMaximise()
	}
}

func (a *App) CloseWindow() {
	if window, ok := a.nativeWindow(); ok {
		window.Close()
	}
}

func (a *App) QuitApp() {
	if app := application.Get(); app != nil {
		app.Quit()
	}
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
	case "windows":
		return controlnotify.Default().Notify(controlnotify.Payload{Title: opts.Title, Body: opts.Body, Tag: opts.Title})
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
