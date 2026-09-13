package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/provider/oauthutil"
	"github.com/kivervinicius/ai-cli/internal/core/provider/spec"
)

const (
	claudeUsageEndpoint = "https://api.anthropic.com/api/oauth/usage"
	claudeUsageBeta     = "oauth-2025-04-20"
	claudeUsageTimeout  = 12 * time.Second
	claudeUsageMaxBody  = 1 << 20
)

var (
	claudeUsageHTTPClient = oauthutil.HTTPClient(claudeUsageTimeout)
	claudeUsageURL        = claudeUsageEndpoint
)

type claudeUsagePeriod struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

type claudeUsageResponse struct {
	FiveHour       *claudeUsagePeriod `json:"five_hour"`
	SevenDay       *claudeUsagePeriod `json:"seven_day"`
	SevenDaySonnet *claudeUsagePeriod `json:"seven_day_sonnet"`
	SevenDayOpus   *claudeUsagePeriod `json:"seven_day_opus"`
}

func (a *Adapter) getLiveUsage(ctx context.Context, p model.Profile, force bool) model.UsageSnapshot {
	_ = force // Refresh always forces; reserved for future TTL short-circuits.
	snap := newClaudeUsageSnapshot(p)
	if oauthutil.KillSwitch("NEXUS_CLAUDE_USAGE_API") {
		return snap
	}

	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	credentials, found, err := readClaudeOAuthCredentials(home)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	if !found {
		snap.Status = model.UsageUnsupported
		return snap
	}

	snap.Account = claudeAccountEmail(home, credentials.Email)
	snap.Plan = credentials.SubscriptionType
	if p.AccountScope.Verifiable() {
		snap.AccountScope = p.AccountScope
	}

	probeCtx, cancel := context.WithTimeout(ctx, claudeUsageTimeout)
	defer cancel()
	if credentials.HasExpiry && !time.Now().Before(credentials.ExpiresAt) &&
		strings.TrimSpace(credentials.RefreshToken) != "" {
		if refreshed, refreshErr := refreshClaudeOAuth(probeCtx, home); refreshErr == nil {
			credentials = refreshed
			snap.Plan = refreshed.SubscriptionType
			if snap.Account == "" {
				snap.Account = claudeAccountEmail(home, refreshed.Email)
			}
		}
		// Refresh is best-effort. The current access token may still be accepted
		// due to clock skew or a token rotation race.
	}
	if strings.TrimSpace(credentials.AccessToken) == "" {
		snap.Error = "Claude OAuth credentials do not contain an access token"
		return snap
	}

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, claudeUsageURL, nil)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	req.Header.Set("Authorization", "Bearer "+credentials.AccessToken)
	req.Header.Set("anthropic-beta", claudeUsageBeta)
	req.Header.Set("User-Agent", "claude-code/"+claudeCodeVersion(probeCtx))
	req.Header.Set("Content-Type", "application/json")

	resp, err := claudeUsageHTTPClient.Do(req)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, claudeUsageMaxBody))
	if err != nil {
		snap.Error = err.Error()
		return snap
	}

	var usage claudeUsageResponse
	parsed := json.Unmarshal(body, &usage) == nil
	if resp.StatusCode == http.StatusTooManyRequests {
		snap.Status = model.UsageRateLimited
		snap.Source = model.SourceOfficialAPI
		snap.Confidence = model.UsageConfidenceHigh
		if parsed {
			snap.Windows = claudeUsageWindows(usage)
		}
		snap.Error = "Anthropic usage API rate limited the request"
		return snap
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		snap.Error = "Claude OAuth authorization is no longer valid; login required"
		return snap
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		snap.Error = fmt.Sprintf("Anthropic usage API returned HTTP %d", resp.StatusCode)
		return snap
	}
	if !parsed {
		snap.Error = "Anthropic usage API returned invalid JSON"
		return snap
	}

	snap.Windows = claudeUsageWindows(usage)
	snap.Status = model.UsageLive
	snap.Source = model.SourceOfficialAPI
	snap.Confidence = model.UsageConfidenceHigh
	return snap
}

func newClaudeUsageSnapshot(p model.Profile) model.UsageSnapshot {
	return model.UsageSnapshot{
		ProviderID: string(model.ProviderClaude),
		ProfileID:  p.Name,
		Status:     model.UsageUnknown,
		Source:     model.SourceNone,
		FetchedAt:  time.Now(),
	}
}

func claudeUsageWindows(usage claudeUsageResponse) []model.UsageWindow {
	windows := make([]model.UsageWindow, 0, 4)
	appendWindow := func(period *claudeUsagePeriod, kind, group string) {
		if period == nil {
			return
		}
		used := clampPercent(period.Utilization)
		remaining := 100 - used
		window := model.UsageWindow{
			Kind:             kind,
			Group:            group,
			UsedPercent:      &used,
			RemainingPercent: &remaining,
		}
		if reset, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(period.ResetsAt)); err == nil {
			reset = reset.UTC()
			window.ResetTime = &reset
			window.ResetDescription = reset.Format(time.RFC3339)
		}
		windows = append(windows, window)
	}
	appendWindow(usage.FiveHour, "5h", "")
	appendWindow(usage.SevenDay, "weekly", "")
	appendWindow(usage.SevenDayOpus, "weekly", "opus")
	appendWindow(usage.SevenDaySonnet, "weekly", "sonnet")
	return windows
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func claudeAccountEmail(home, credentialEmail string) string {
	if email := strings.TrimSpace(credentialEmail); email != "" {
		return email
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude.json"))
	if err != nil {
		return ""
	}
	var identity struct {
		OAuthAccount struct {
			Email string `json:"email"`
		} `json:"oauthAccount"`
	}
	if json.Unmarshal(data, &identity) != nil {
		return ""
	}
	return strings.TrimSpace(identity.OAuthAccount.Email)
}

func claudeCodeVersion(_ context.Context) string {
	// Never spawn `claude --version` on the usage hot path — that alone can
	// add multi-second stalls while the TUI/list waits on quota.
	known := spec.MustGet("claude").KnownVersions
	if len(known) > 0 && strings.TrimSpace(known[0]) != "" {
		return strings.TrimSpace(known[0])
	}
	return "unknown"
}
