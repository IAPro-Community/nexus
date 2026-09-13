package claude

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
)

type claudeRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn claudeRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestClaudeUsageFixtureMapsWindows(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "oauth_usage_v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var response claudeUsageResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}

	windows := claudeUsageWindows(response)
	if len(windows) != 3 {
		t.Fatalf("windows=%d want 3", len(windows))
	}
	assertClaudeWindow(t, windows[0], "5h", "", 65)
	assertClaudeWindow(t, windows[1], "weekly", "", 86)
	assertClaudeWindow(t, windows[2], "weekly", "sonnet", 77.5)
	if windows[0].ResetTime == nil || windows[0].ResetTime.Format(time.RFC3339) != "2026-09-13T22:00:00Z" {
		t.Fatalf("unexpected five-hour reset: %v", windows[0].ResetTime)
	}
}

func TestClaudeUsageWithoutOAuthBlockIsUnsupported(t *testing.T) {
	home := setupClaudeProfile(t, "api-key")
	if err := os.WriteFile(
		filepath.Join(home, ".claude", ".credentials.json"),
		[]byte(`{"apiKey":"sk-ant-test"}`),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	snap := New().GetUsage(context.Background(), model.Profile{Provider: "claude", Name: "api-key"})
	if snap.Status != model.UsageUnsupported || snap.Source != model.SourceNone {
		t.Fatalf("status/source=%s/%s want UNSUPPORTED/NONE", snap.Status, snap.Source)
	}
}

func TestClaudeUsageKillSwitchDoesNotCallHTTP(t *testing.T) {
	home := setupClaudeProfile(t, "disabled")
	writeClaudeOAuthFixture(t, home, "token")
	t.Setenv("NEXUS_CLAUDE_USAGE_API", "false")

	originalClient := claudeUsageHTTPClient
	t.Cleanup(func() { claudeUsageHTTPClient = originalClient })
	var calls atomic.Int32
	claudeUsageHTTPClient = &http.Client{Transport: claudeRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, nil
	})}

	snap := New().RefreshUsage(context.Background(), model.Profile{Provider: "claude", Name: "disabled"})
	if calls.Load() != 0 {
		t.Fatalf("HTTP calls=%d want 0", calls.Load())
	}
	if snap.Status != model.UsageUnknown || snap.Source != model.SourceNone {
		t.Fatalf("status/source=%s/%s want UNKNOWN/NONE", snap.Status, snap.Source)
	}
}

func TestClaudeUsageFetchesOfficialAPI(t *testing.T) {
	home := setupClaudeProfile(t, "live")
	writeClaudeOAuthFixture(t, home, "access-token")
	if err := os.WriteFile(
		filepath.Join(home, ".claude.json"),
		[]byte(`{"oauthAccount":{"email":"claude@example.com"}}`),
		0600,
	); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join("testdata", "oauth_usage_v1.json"))
	if err != nil {
		t.Fatal(err)
	}

	originalClient := claudeUsageHTTPClient
	t.Cleanup(func() { claudeUsageHTTPClient = originalClient })
	claudeUsageHTTPClient = &http.Client{Transport: claudeRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("method=%s want GET", req.Method)
		}
		if req.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("unexpected authorization header")
		}
		if req.Header.Get("anthropic-beta") != claudeUsageBeta {
			t.Errorf("anthropic-beta=%q", req.Header.Get("anthropic-beta"))
		}
		if !strings.HasPrefix(req.Header.Get("User-Agent"), "claude-code/") {
			t.Errorf("user-agent=%q", req.Header.Get("User-Agent"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(string(fixture))),
			Request:    req,
		}, nil
	})}

	snap := New().RefreshUsage(context.Background(), model.Profile{Provider: "claude", Name: "live"})
	if snap.Status != model.UsageLive || snap.Source != model.SourceOfficialAPI {
		t.Fatalf("status/source=%s/%s want LIVE/OFFICIAL_API (error=%q)", snap.Status, snap.Source, snap.Error)
	}
	if snap.Account != "claude@example.com" {
		t.Fatalf("account=%q", snap.Account)
	}
	if len(snap.Windows) != 3 {
		t.Fatalf("windows=%d want 3", len(snap.Windows))
	}
}

func assertClaudeWindow(t *testing.T, window model.UsageWindow, kind, group string, remaining float64) {
	t.Helper()
	if window.Kind != kind || window.Group != group {
		t.Fatalf("window kind/group=%q/%q want %q/%q", window.Kind, window.Group, kind, group)
	}
	if window.RemainingPercent == nil || *window.RemainingPercent != remaining {
		t.Fatalf("remaining=%v want %.1f", window.RemainingPercent, remaining)
	}
}

func writeClaudeOAuthFixture(t *testing.T, home, accessToken string) {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken":      accessToken,
			"refreshToken":     "refresh-token",
			"expiresAt":        time.Now().Add(time.Hour).UnixMilli(),
			"scopes":           []string{"user:profile", "user:inference"},
			"subscriptionType": "pro",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", ".credentials.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
