package notify

import (
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
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
	if last, ok := t.lastSent[key]; ok && time.Since(last) < t.cooldown {
		t.mu.Unlock()
		return nil
	}
	t.lastSent[key] = time.Now()
	t.mu.Unlock()
	return t.inner.Notify(payload)
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
		cmd := exec.Command("notify-send", "--app-name=Nexus", title, body)
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
		return exec.Command("powershell", "-NoProfile", "-Command", windowsToastScript, title, body).Run()
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
