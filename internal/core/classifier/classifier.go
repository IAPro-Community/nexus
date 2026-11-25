package classifier

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	sdkclassifier "github.com/IAPro-Community/IAPro-SDK/classifier"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

var (
	// Provider-specific diagnostics absent from the current shared classifier.
	// Keep this compatibility layer small and remove entries after SDK parity.
	providerRateLimitPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bresource_exhausted\b`),
		regexp.MustCompile(`(?i)\byou'?ve (?:hit|reached) your usage limit\b`),
		regexp.MustCompile(`(?i)\blimit will reset (?:at|in)\b`),
	}
	providerQuotaPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bindividual quota reached\b`),
		regexp.MustCompile(`(?i)\bplease upgrade your subscription\b`),
	}
	resetCompoundRegex     = regexp.MustCompile(`(?i)resets?\s+in\s+((?:\d+h)?(?:\d+m)?(?:\d+s)?)`)
	resetAbsoluteTimeRegex = regexp.MustCompile(`(?i)resets?\s+(?:at|@)\s+(\d{1,2})(?::(\d{2}))?\s*(am|pm)?`)
)

// Classify delegates common failure semantics to IAPro-SDK. Nexus-specific
// terminal diagnostics remain in this adapter until they are added upstream.
func Classify(err error, output string) model.Failure {
	if err == nil && strings.TrimSpace(output) == "" {
		return model.Failure{Kind: model.FailureNone}
	}
	if err == nil {
		// The SDK accepts trusted diagnostics only with a process failure. Existing
		// Nexus call sites also classify captured terminal output without an error.
		err = errors.New("provider process returned diagnostic output")
	}
	combined := err.Error() + "\n" + output

	classified := sdkclassifier.Classify(err, output)
	if classified.Kind == model.FailureUnknown {
		for _, pattern := range providerRateLimitPatterns {
			if pattern.MatchString(combined) {
				classified.Kind = model.FailureRateLimit
				classified.Message = "Rate limit exceeded on provider"
				break
			}
		}
	}
	if classified.Kind == model.FailureUnknown {
		for _, pattern := range providerQuotaPatterns {
			if pattern.MatchString(combined) {
				classified.Kind = model.FailureQuota
				classified.Message = "Account quota / credit limit reached"
				break
			}
		}
	}
	if classified.Kind == model.FailureRateLimit || classified.Kind == model.FailureQuota {
		duration := ExtractRetryDuration(combined)
		classified.RetryAfter = &duration
		if duration > 0 {
			resetAt := time.Now().Add(duration)
			classified.ResetAt = &resetAt
		}
	}
	return classified
}

// ExtractRetryDuration keeps Nexus's provider reset formats and delegates
// ordinary retry hints to the SDK's overflow-safe parser.
func ExtractRetryDuration(s string) time.Duration {
	if match := resetCompoundRegex.FindStringSubmatch(s); len(match) > 1 && strings.TrimSpace(match[1]) != "" {
		if duration, err := time.ParseDuration(match[1]); err == nil && duration > 0 {
			return duration
		}
	}
	if match := resetAbsoluteTimeRegex.FindStringSubmatch(s); len(match) > 1 {
		hour, hourErr := strconv.Atoi(match[1])
		minute, minuteErr := 0, error(nil)
		if match[2] != "" {
			minute, minuteErr = strconv.Atoi(match[2])
		}
		if hourErr == nil && minuteErr == nil {
			switch strings.ToLower(match[3]) {
			case "pm":
				if hour < 12 {
					hour += 12
				}
			case "am":
				if hour == 12 {
					hour = 0
				}
			}
			if hour >= 0 && hour < 24 && minute >= 0 && minute < 60 {
				now := time.Now()
				candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
				if !candidate.After(now) {
					candidate = candidate.Add(24 * time.Hour)
				}
				return time.Until(candidate)
			}
		}
	}
	return sdkclassifier.ExtractRetryDuration(s)
}
