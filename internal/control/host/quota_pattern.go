package host

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

const (
	// quotaPatternWindowSize bounds the rolling output window. Provider limit
	// banners live in the tail of the stream; a bounded window keeps the
	// matching cost flat for long sessions.
	quotaPatternWindowSize = 16 * 1024
	// quotaPatternMinUptime guards against startup garbage (resumed banners,
	// replayed paint) masquerading as a live limit: the runtime must have been
	// up for a moment before a pattern can accelerate the quota check.
	quotaPatternMinUptime = 15 * time.Second
)

// quotaBannerPattern is a strict, provider-scoped phrase that the provider's
// own TUI prints when the account actually hits its limit. Patterns are
// deliberately narrower than classifier failure patterns: the classifier
// judges API error output, while this window also contains the agent's own
// writing (a blog post may legitimately discuss "http 429").
type quotaBannerPattern struct {
	kind     model.FailureKind
	compiled *regexp.Regexp
}

var quotaBannerPatterns = map[string][]quotaBannerPattern{
	"codex": {
		// codex TUI limit screen.
		{kind: model.FailureRateLimit, compiled: regexp.MustCompile(`(?i)you'?ve (?:hit|reached) your usage limit`)},
		{kind: model.FailureRateLimit, compiled: regexp.MustCompile(`(?i)usage limit reached`)},
		{kind: model.FailureRateLimit, compiled: regexp.MustCompile(`(?i)limit will reset (?:at|in)`)},
		// Billing-style exhaustion.
		{kind: model.FailureQuota, compiled: regexp.MustCompile(`(?i)(?:insufficient credits|out of credits|billing limit)`)},
	},
	"agy": {
		// AGY weekly limit screen, locked from production evidence (2026-11-14):
		// "⚠ Individual quota reached. Please upgrade your subscription to
		// increase your limits. Resets in 167h58m45s." The same wording is
		// covered by the shared exit classifier; this entry lets supervised
		// runtimes accelerate the quota evidence check on the banner itself.
		{kind: model.FailureQuota, compiled: regexp.MustCompile(`(?i)individual quota reached`)},
	},
}

// matchQuotaBanner scans the window for the provider's own limit banners.
// Providers without evidence-backed patterns are disabled: no banner, no
// trigger.
func matchQuotaBanner(providerID, window string) (model.FailureKind, bool) {
	for _, pattern := range quotaBannerPatterns[strings.ToLower(strings.TrimSpace(providerID))] {
		if pattern.compiled.MatchString(window) {
			return pattern.kind, true
		}
	}
	return "", false
}

// QuotaLimitWatcher observes provider output as a read-only side effect of the
// existing stream pipeline. It never writes to the PTY, never blocks the
// fanout, and — critically — never terminates or mutates a session on its own:
// a pattern match only accelerates the quota evidence check. The continuity
// decision itself still requires confirmed quota evidence (Status RateLimited
// or exhausted balance), which keeps a model merely *discussing* rate limits
// from cutting a healthy session.
type QuotaLimitWatcher struct {
	mu         sync.Mutex
	runtimeID  string
	providerID string
	profileID  string
	lineageID  string
	window     []byte
	triggered  bool
	startedAt  time.Time
	publish    func(events.Event)
}

// NewQuotaLimitWatcher creates the watcher for one supervised runtime. The
// publish function is a test seam; production wires events.DefaultBus.
func NewQuotaLimitWatcher(runtimeID, providerID, profileID, lineageID string, startedAt time.Time, publish func(events.Event)) *QuotaLimitWatcher {
	return &QuotaLimitWatcher{
		runtimeID:  runtimeID,
		providerID: providerID,
		profileID:  profileID,
		lineageID:  lineageID,
		startedAt:  startedAt,
		publish:    publish,
	}
}

// ProcessChunk folds an output chunk into the rolling window and, when the
// provider's own limit banner shows up, publishes exactly one early
// evidence-check event per runtime lifetime.
func (w *QuotaLimitWatcher) ProcessChunk(chunk []byte) {
	if w == nil || len(chunk) == 0 {
		return
	}
	w.mu.Lock()
	if w.triggered {
		w.mu.Unlock()
		return
	}
	w.window = append(w.window, chunk...)
	if len(w.window) > quotaPatternWindowSize {
		w.window = w.window[len(w.window)-quotaPatternWindowSize:]
	}
	if time.Since(w.startedAt) < quotaPatternMinUptime {
		w.mu.Unlock()
		return
	}
	kind, matched := matchQuotaBanner(w.providerID, string(w.window))
	if !matched {
		w.mu.Unlock()
		return
	}
	w.triggered = true
	w.mu.Unlock()

	w.publishQuotaPatternEvent(kind)
}

// publishQuotaPatternEvent announces the match. The event is an input to the
// continuity pipeline, never an authorization to touch the session directly.
func (w *QuotaLimitWatcher) publishQuotaPatternEvent(kind model.FailureKind) {
	if w.publish == nil {
		return
	}
	metadata := map[string]any{
		"affected_runtime_id": w.runtimeID,
		"provider":            w.providerID,
		"profile":             w.profileID,
		"nexus.quota_trigger": "pty_pattern",
		"failure_kind":        string(kind),
	}
	summary := "Provider limit pattern detected in PTY output; quota evidence check requested"
	w.publish(events.NewEventWithCorrelation(
		w.lineageID,
		w.runtimeID,
		w.providerID,
		w.profileID,
		events.EventQuotaExhausted,
		summary,
		metadata,
	))
}

// quotaPatternPublisher returns the production publisher for a host.
func (sh *SessionHost) quotaPatternPublisher() func(events.Event) {
	return func(e events.Event) {
		events.DefaultBus().Publish(e)
	}
}

// observeOutputChunk is the single read-only observation point for the stream
// pipeline: attention detection first (existing behavior), then the quota
// pattern watcher. Both are strictly observational.
func (sh *SessionHost) observeOutputChunk(chunk []byte) {
	if sh.detector != nil {
		sh.detector.ProcessChunk(chunk)
	}
	if sh.quotaWatcher != nil {
		sh.quotaWatcher.ProcessChunk(chunk)
	}
}
