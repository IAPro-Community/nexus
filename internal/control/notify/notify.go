package notify

import (
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/executil"
)

// Payload is a desktop notification request from the Nexus host process.
type Payload struct {
	Title string
	Body  string
	Tag   string
}

// Notifier delivers OS notifications without depending on a browser tab.
type Notifier interface {
	Notify(payload Payload) error
}

// AvailabilityReporter lets a best-effort delivery workflow distinguish an
// unavailable desktop channel from a successful no-op. Implementations that do
// not report capability are considered available, which keeps test recorders
// and embedded adapters simple.
type AvailabilityReporter interface {
	Available() bool
}

// IsAvailable reports whether the native desktop notification channel can be
// used by the given notifier.
func IsAvailable(notifier Notifier) bool {
	if reporter, ok := notifier.(AvailabilityReporter); ok {
		return reporter.Available()
	}
	return notifier != nil
}

type throttledNotifier struct {
	inner    Notifier
	mu       sync.Mutex
	lastSent map[string]time.Time
	cooldown time.Duration
}

// NewThrottled wraps a notifier with per-tag cooldown (default 30s).
func NewThrottled(inner Notifier, cooldown time.Duration) Notifier {
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &throttledNotifier{
		inner:    inner,
		lastSent: make(map[string]time.Time),
		cooldown: cooldown,
	}
}

func (t *throttledNotifier) Notify(payload Payload) error {
	key := strings.TrimSpace(payload.Tag)
	if key == "" {
		key = payload.Title + "|" + payload.Body
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if last, ok := t.lastSent[key]; ok && time.Since(last) < t.cooldown {
		return nil
	}
	if err := t.inner.Notify(payload); err != nil {
		return err
	}
	t.lastSent[key] = time.Now()
	return nil
}

func (t *throttledNotifier) Available() bool {
	return IsAvailable(t.inner)
}

// Recorder captures notifications for tests.
type Recorder struct {
	mu       sync.Mutex
	Payloads []Payload
}

func (r *Recorder) Notify(payload Payload) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Payloads = append(r.Payloads, payload)
	return nil
}

func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Payloads = nil
}

type desktopNotifier struct{}

func (desktopNotifier) Available() bool {
	switch runtime.GOOS {
	case "linux":
		_, err := exec.LookPath("notify-send")
		return err == nil
	case "darwin":
		_, err := exec.LookPath("osascript")
		return err == nil
	case "windows":
		_, err := exec.LookPath("powershell")
		return err == nil
	default:
		return false
	}
}

func (desktopNotifier) Notify(payload Payload) error {
	title := strings.TrimSpace(payload.Title)
	body := strings.TrimSpace(payload.Body)
	if title == "" {
		title = "Nexus"
	}
	if body == "" {
		return nil
	}
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("notify-send"); err != nil {
			return nil
		}
		cmd := executil.Command("notify-send", "--app-name=Nexus", title, body)
		return cmd.Run()
	case "darwin":
		return exec.Command(
			"osascript",
			"-e",
			`on run argv
display notification (item 1 of argv) with title (item 2 of argv)
end run`,
			"--",
			body,
			title,
		).Run()
	case "windows":
		return executil.Command("powershell", "-NoProfile", "-Command", windowsToastScript, title, body).Run()
	default:
		return nil
	}
}

const windowsToastScript = `param([string]$Title, [string]$Body)
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$text = $template.GetElementsByTagName('text')
$text.Item(0).AppendChild($template.CreateTextNode($Title)) > $null
$text.Item(1).AppendChild($template.CreateTextNode($Body)) > $null
$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Nexus').Show($toast)`

var (
	defaultMu sync.Mutex
	defaultN  Notifier = NewThrottled(desktopNotifier{}, 30*time.Second)
)

// Default returns the process-wide notifier.
func Default() Notifier {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	return defaultN
}

// SetDefault replaces the process-wide notifier (tests).
func SetDefault(n Notifier) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if n == nil {
		defaultN = NewThrottled(desktopNotifier{}, 30*time.Second)
		return
	}
	defaultN = n
}
