package opencode

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/oauthutil"
)

const (
	opencodeZenModelsURL = "https://opencode.ai/zen/v1/models"
	opencodeUsageTimeout = 12 * time.Second
	opencodeUsageMaxBody = 1 << 20
)

var (
	opencodeUsageHTTPClient = oauthutil.HTTPClient(opencodeUsageTimeout)
	opencodeZenURL          = opencodeZenModelsURL
)

type openCodeAuthFile map[string]openCodeAuthEntry

type openCodeAuthEntry struct {
	Type     string `json:"type"`
	Key      string `json:"key"`
	Token    string `json:"token"`
	Access   string `json:"access"`
	Refresh  string `json:"refresh"`
	Provider string `json:"provider"`
}

func (a *Adapter) GetUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	// OpenCode has no balance API. Paint/list paths stay off the network and
	// never touch the host auth store.
	return model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Status:     model.UsageUnsupported,
		Source:     model.SourceNone,
		FetchedAt:  time.Now(),
	}
}

func (a *Adapter) RefreshUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	return a.probeUsage(ctx, p)
}

func (a *Adapter) probeUsage(ctx context.Context, p model.Profile) model.UsageSnapshot {
	snap := model.UsageSnapshot{
		ProviderID: string(a.ID()),
		ProfileID:  p.Name,
		Status:     model.UsageUnsupported,
		Source:     model.SourceNone,
		FetchedAt:  time.Now(),
	}
	if oauthutil.KillSwitch("NEXUS_OPENCODE_USAGE_API") {
		return snap
	}

	home, err := config.ProfileHome(string(a.ID()), p.Name)
	if err != nil {
		snap.Status = model.UsageUnknown
		snap.Error = err.Error()
		return snap
	}
	authPath := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	data, err := os.ReadFile(authPath)
	if err != nil {
		// Fail closed: never probe with the host's unmanaged auth.json — that
		// would leak host API keys into an isolated profile quota read.
		snap.Status = model.UsageUnknown
		snap.Error = "opencode auth.json not found in profile home"
		return snap
	}
	var auth openCodeAuthFile
	if json.Unmarshal(data, &auth) != nil || len(auth) == 0 {
		snap.Status = model.UsageUnknown
		snap.Error = "opencode auth.json is invalid"
		return snap
	}

	// Prefer Anthropic OAuth upstream when present — reuse is handled by
	// returning UNSUPPORTED here with a clear note; Claude adapter owns the
	// live Anthropic probe for Claude profiles. OpenCode Zen has no balance API.
	if entry, ok := auth["anthropic"]; ok && strings.EqualFold(entry.Type, "oauth") {
		snap.Plan = "opencode/anthropic"
		snap.Error = "OpenCode Anthropic upstream: use a Claude profile for live Anthropic quota"
		return snap
	}
	if entry, ok := auth["opencode"]; ok {
		apiKey := firstNonEmpty(entry.Key, entry.Token, entry.Access)
		if apiKey == "" {
			return snap
		}
		status, probeErr := probeZenModels(ctx, apiKey)
		snap.Plan = "opencode/zen"
		snap.Source = model.SourceOfficialAPI
		switch status {
		case http.StatusOK:
			snap.Status = model.UsageUnsupported
			snap.Error = "OpenCode Zen does not expose spend/balance to API keys"
		case http.StatusUnauthorized, http.StatusForbidden:
			snap.Status = model.UsageUnknown
			snap.Error = "OpenCode Zen authentication failed"
		case http.StatusTooManyRequests:
			snap.Status = model.UsageRateLimited
			snap.Error = "OpenCode Zen rate limited"
		default:
			snap.Status = model.UsageUnknown
			if probeErr != "" {
				snap.Error = probeErr
			} else {
				snap.Error = "OpenCode Zen probe failed"
			}
		}
		return snap
	}

	snap.Error = "OpenCode upstream has no attributable quota source"
	return snap
}

func probeZenModels(ctx context.Context, apiKey string) (int, string) {
	probeCtx, cancel := context.WithTimeout(ctx, opencodeUsageTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, opencodeZenURL, nil)
	if err != nil {
		return 0, err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := opencodeUsageHTTPClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, opencodeUsageMaxBody))
	return resp.StatusCode, ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
