package notify

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/executil"
)

// Payload is a desktop notification request from the Nexus host process.
type Payload struct {
	Title      string
	Body       string
	Tag        string
	Category   Category
	EpisodeKey string
}

// Category classifies a native popup for the installation-wide notification policy.
type Category string

const (
	CategoryActionRequired     Category = "action_required"
	CategoryAuthPermission     Category = "auth_permission"
	CategoryCriticalFailure    Category = "critical_failure"
	CategoryQuotaExhausted     Category = "quota_exhausted"
	CategoryQuotaLow           Category = "quota_low"
	CategoryCompletionRecovery Category = "completion_recovery"
	CategoryRuntimeConnection  Category = "runtime_connection"
)

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

const maxSeenNotificationKeys = 2048

type throttledNotifier struct {
	inner Notifier
	mu    sync.Mutex
	seen  map[string]struct{}
	order []string
}

// NewThrottled retains its historical name for source compatibility. Delivery
// is now deduplicated by event episode identity, so a new episode is immediate
// even when it occurs within the old 30-second cooldown.
func NewThrottled(inner Notifier, _ time.Duration) Notifier {
	return &throttledNotifier{
		inner: inner,
		seen:  make(map[string]struct{}),
	}
}

func (t *throttledNotifier) Notify(payload Payload) error {
	if !CategoryEnabled(payload.Category) {
		return nil
	}
	key := strings.TrimSpace(payload.EpisodeKey)
	if key == "" {
		key = strings.TrimSpace(payload.Tag)
	}
	if key == "" {
		key = payload.Title + "|" + payload.Body
	}
	if payload.Category != "" {
		key = string(payload.Category) + "|" + key
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.seen[key]; ok {
		return nil
	}
	if err := t.inner.Notify(payload); err != nil {
		return err
	}
	if len(t.order) >= maxSeenNotificationKeys {
		oldest := t.order[0]
		copy(t.order, t.order[1:])
		t.order = t.order[:len(t.order)-1]
		delete(t.seen, oldest)
	}
	t.seen[key] = struct{}{}
	t.order = append(t.order, key)
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
		return windowsToastCommand(title, body).Run()
	default:
		return nil
	}
}

func windowsToastCommand(title, body string) *exec.Cmd {
	cmd := executil.Command("powershell", "-NoProfile", "-Command", windowsToastScript)
	cmd.Env = append(os.Environ(),
		"NEXUS_TOAST_TITLE="+title,
		"NEXUS_TOAST_BODY="+body,
	)
	return cmd
}

const windowsToastScript = `$Title = $env:NEXUS_TOAST_TITLE
$Body = $env:NEXUS_TOAST_BODY
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$text = $template.GetElementsByTagName('text')
$text.Item(0).AppendChild($template.CreateTextNode($Title)) > $null
$text.Item(1).AppendChild($template.CreateTextNode($Body)) > $null
$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Nexus').Show($toast)`

var (
	defaultMu    sync.Mutex
	defaultN     Notifier = NewThrottled(desktopNotifier{}, 30*time.Second)
	policyMu     sync.RWMutex
	nativePolicy = Policy{
		Categories: map[Category]bool{
			CategoryActionRequired: true, CategoryAuthPermission: true,
			CategoryCriticalFailure: true, CategoryQuotaExhausted: true,
			CategoryQuotaLow: false, CategoryCompletionRecovery: false,
			CategoryRuntimeConnection: false,
		},
		QuietHoursStartUTC:        "22:00",
		QuietHoursEndUTC:          "08:00",
		ExhaustedBypassQuietHours: true,
	}
)

// Policy controls categorized native popup delivery for one Nexus installation.
type Policy struct {
	Categories                map[Category]bool
	QuietHoursEnabled         bool
	QuietHoursStartUTC        string
	QuietHoursEndUTC          string
	ExhaustedBypassQuietHours bool
}

// SetCategoryPolicy updates the installation-wide native popup preferences.
func SetCategoryPolicy(policy Policy) {
	policyMu.Lock()
	defer policyMu.Unlock()
	nativePolicy = Policy{Categories: make(map[Category]bool, len(policy.Categories)),
		QuietHoursEnabled:         policy.QuietHoursEnabled,
		QuietHoursStartUTC:        policy.QuietHoursStartUTC,
		QuietHoursEndUTC:          policy.QuietHoursEndUTC,
		ExhaustedBypassQuietHours: policy.ExhaustedBypassQuietHours}
	for category, enabled := range policy.Categories {
		nativePolicy.Categories[category] = enabled
	}
}

// CategoryEnabled returns whether a categorized native popup is enabled.
// Uncategorized legacy notifications remain enabled until their producer is migrated.
func CategoryEnabled(category Category) bool {
	if category == "" {
		return true
	}
	policyMu.RLock()
	policy := nativePolicy
	enabled, known := policy.Categories[category]
	policyMu.RUnlock()
	if known && !enabled {
		return false
	}
	if !known || !policy.QuietHoursEnabled || bypassesQuietHours(category, policy) {
		return true
	}
	return !isQuietNow(policy.QuietHoursStartUTC, policy.QuietHoursEndUTC, time.Now().UTC())
}

func bypassesQuietHours(category Category, policy Policy) bool {
	if category == CategoryQuotaExhausted {
		return policy.ExhaustedBypassQuietHours
	}
	return category == CategoryActionRequired || category == CategoryAuthPermission || category == CategoryCriticalFailure
}

func isQuietNow(start, end string, now time.Time) bool {
	startTime, startErr := time.Parse("15:04", strings.TrimSpace(start))
	endTime, endErr := time.Parse("15:04", strings.TrimSpace(end))
	if startErr != nil || endErr != nil {
		return false
	}
	minute := now.Hour()*60 + now.Minute()
	startMinute := startTime.Hour()*60 + startTime.Minute()
	endMinute := endTime.Hour()*60 + endTime.Minute()
	if startMinute == endMinute {
		return false
	}
	if startMinute < endMinute {
		return minute >= startMinute && minute < endMinute
	}
	return minute >= startMinute || minute < endMinute
}

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
