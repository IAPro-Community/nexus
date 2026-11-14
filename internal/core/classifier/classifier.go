package classifier

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

var (
	rateLimitPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(rate[ -]?limit(ed)?|too many requests|http 429|status 429)\b`),
		regexp.MustCompile(`(?i)\b(quota exceeded|usage limit reached|exceeded your current quota)\b`),
		regexp.MustCompile(`(?i)\b(capacity reached|temporarily rate-limited)\b`),
		regexp.MustCompile(`(?i)\bresource_exhausted\b`),
		// Codex TUI limit screen, evidence-backed by the supervised host
		// watcher (internal/control/host/quota_pattern.go): keep the exit
		// classifier aligned so a direct launch classifies identically.
		regexp.MustCompile(`(?i)\byou'?ve (?:hit|reached) your usage limit\b`),
		regexp.MustCompile(`(?i)\blimit will reset (?:at|in)\b`),
	}

	authPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(auth(entication)? required|unauthorized|http 401|status 401)\b`),
		regexp.MustCompile(`(?i)\b(token expired|session expired|invalid token|invalid api key)\b`),
		regexp.MustCompile(`(?i)\b(please login|run .*login|not logged in)\b`),
		regexp.MustCompile(`(?i)\b(access denied|permission denied|forbidden|http 403)\b`),
	}

	quotaExhaustedPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(insufficient credits|out of credits|balance too low|billing limit)\b`),
		regexp.MustCompile(`(?i)\b(credit balance is too low|plan quota exhausted)\b`),
		// AGY (Gemini CLI) weekly/individual quota: "Individual quota reached.
		// Please upgrade your subscription to increase your limits." — without
		// this the failure classifies as Unknown and failover never rotates.
		regexp.MustCompile(`(?i)\bindividual quota reached\b`),
		regexp.MustCompile(`(?i)\bplease upgrade your subscription\b`),
	}

	networkPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(connection refused|network unreachable|dial tcp|no route to host)\b`),
		regexp.MustCompile(`(?i)\b(tls handshake timeout|i/o timeout|connection reset by peer)\b`),
	}

	providerErrorPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(internal server error|http 500|http 502|http 503|http 504)\b`),
		regexp.MustCompile(`(?i)\b(service unavailable|bad gateway|gateway timeout|model overloaded)\b`),
	}

	retrySecondsRegex = regexp.MustCompile(`(?i)(?:retry|try again|wait)\s+(?:in|after)?\s*(\d+)\s*(?:s|sec|seconds)`)
	retryMinutesRegex = regexp.MustCompile(`(?i)(?:retry|try again|wait)\s+(?:in|after)?\s*(\d+)\s*(?:m|min|minutes)`)
	retryHoursRegex   = regexp.MustCompile(`(?i)(?:retry|try again|wait)\s+(?:in|after)?\s*(\d+)\s*(?:h|hr|hours)`)
	// AGY weekly reset: "Resets in 167h58m45s" (also "58m45s", "45s").
	resetCompoundRegex = regexp.MustCompile(`(?i)resets?\s+in\s+(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?`)
)

// Classify inspects an error and output log to determine the failure kind and retry parameters.
func Classify(err error, output string) model.Failure {
	combined := ""
	if err != nil {
		combined += err.Error() + "\n"
	}
	combined += output

	if strings.TrimSpace(combined) == "" {
		return model.Failure{Kind: model.FailureNone}
	}

	// 1. Check Rate Limit
	for _, p := range rateLimitPatterns {
		if p.MatchString(combined) {
			dur := ExtractRetryDuration(combined)
			var resetAt *time.Time
			if dur > 0 {
				t := time.Now().Add(dur)
				resetAt = &t
			}
			return model.Failure{
				Kind:       model.FailureRateLimit,
				Message:    "Rate limit exceeded on provider",
				RetryAfter: &dur,
				ResetAt:    resetAt,
			}
		}
	}

	// 2. Check Quota Exhaustion
	for _, p := range quotaExhaustedPatterns {
		if p.MatchString(combined) {
			// Quota windows carry their own reset (e.g. AGY weekly "Resets in
			// 167h58m45s"): surface it so cooldown keeps the account out of
			// automatic rotation until the provider reopens the window.
			dur := ExtractRetryDuration(combined)
			var resetAt *time.Time
			if dur > 0 {
				t := time.Now().Add(dur)
				resetAt = &t
			}
			return model.Failure{
				Kind:       model.FailureQuota,
				Message:    "Account quota / credit limit reached",
				RetryAfter: &dur,
				ResetAt:    resetAt,
			}
		}
	}

	// 3. Check Auth Failure
	for _, p := range authPatterns {
		if p.MatchString(combined) {
			return model.Failure{
				Kind:    model.FailureAuth,
				Message: "Authentication required or credentials expired",
			}
		}
	}

	// 4. Check Network Failure
	for _, p := range networkPatterns {
		if p.MatchString(combined) {
			return model.Failure{
				Kind:    model.FailureNetwork,
				Message: "Network connection failure to AI provider",
			}
		}
	}

	// 5. Check Provider Transient Error
	for _, p := range providerErrorPatterns {
		if p.MatchString(combined) {
			return model.Failure{
				Kind:    model.FailureProvider,
				Message: "AI provider service unavailable or overloaded",
			}
		}
	}

	return model.Failure{
		Kind:    model.FailureUnknown,
		Message: "CLI command returned non-zero exit",
	}
}

// ExtractRetryDuration attempts to parse retry cooldown intervals from error messages.
func ExtractRetryDuration(s string) time.Duration {
	// Provider reset signals (e.g. AGY "Resets in 167h58m45s") are the most
	// authoritative duration hint: they state when the quota window itself
	// reopens, not when to retry a transient request.
	if m := resetCompoundRegex.FindStringSubmatch(s); len(m) > 1 {
		hours, _ := strconv.Atoi(m[1])
		minutes, _ := strconv.Atoi(m[2])
		seconds, _ := strconv.Atoi(m[3])
		if total := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second; total > 0 {
			return total
		}
	}
	if m := retrySecondsRegex.FindStringSubmatch(s); len(m) > 1 {
		if val, err := strconv.Atoi(m[1]); err == nil && val > 0 {
			return time.Duration(val) * time.Second
		}
	}
	if m := retryMinutesRegex.FindStringSubmatch(s); len(m) > 1 {
		if val, err := strconv.Atoi(m[1]); err == nil && val > 0 {
			return time.Duration(val) * time.Minute
		}
	}
	if m := retryHoursRegex.FindStringSubmatch(s); len(m) > 1 {
		if val, err := strconv.Atoi(m[1]); err == nil && val > 0 {
			return time.Duration(val) * time.Hour
		}
	}
	return 0
}
