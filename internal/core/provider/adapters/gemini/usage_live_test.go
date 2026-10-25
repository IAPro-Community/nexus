package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/oauthutil"
)

func TestParseGeminiQuotaFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "retrieve_user_quota_v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var response geminiQuotaResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}

	windows := parseGeminiQuotaBuckets(response.Buckets)
	if len(windows) != 2 {
		t.Fatalf("got %d windows, want 2", len(windows))
	}
	if windows[0].Group != "gemini-2.5-pro" || windows[0].Kind != "requests" {
		t.Fatalf("unexpected first window: %+v", windows[0])
	}
	if windows[0].Remaining == nil || *windows[0].Remaining != 42 {
		t.Fatalf("remaining amount not parsed: %+v", windows[0].Remaining)
	}
	if windows[0].RemainingPercent == nil || *windows[0].RemainingPercent != 42 {
		t.Fatalf("remaining percent not parsed: %+v", windows[0].RemainingPercent)
	}
	if windows[1].Remaining != nil {
		t.Fatalf("absent remainingAmount must remain unknown: %+v", windows[1].Remaining)
	}
	if windows[1].RemainingPercent == nil || *windows[1].RemainingPercent != 100 {
		t.Fatalf("full bucket percent not parsed: %+v", windows[1].RemainingPercent)
	}
}

func TestGeminiUsageKillSwitchReturnsUnknownWithoutUsageJSONPassthrough(t *testing.T) {
	home := setupGeminiProfile(t, "disabled")
	t.Setenv("NEXUS_GEMINI_USAGE_API", "0")
	legacy := model.UsageSnapshot{
		ProviderID: "gemini",
		ProfileID:  "disabled",
		Status:     model.UsageLive,
		Source:     model.SourceOfficialAPI,
	}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "usage.json"), data, 0600); err != nil {
		t.Fatal(err)
	}

	snapshot := New().RefreshUsage(context.Background(), model.Profile{Name: "disabled"})
	if snapshot.Status != model.UsageUnknown || snapshot.Source != model.SourceNone {
		t.Fatalf("kill switch returned invented usage: %+v", snapshot)
	}
}

func TestImportHostAuthCopiesGeminiOAuthToIsolatedProfile(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("AI_CLI_DATA_DIR", dataDir)
	t.Setenv("AI_CLI_CONFIG_DIR", t.TempDir())
	hostHome := t.TempDir()
	t.Setenv("AI_REAL_HOME", hostHome)
	t.Setenv("HOME", hostHome)
	hostGemini := filepath.Join(hostHome, ".gemini")
	if err := os.MkdirAll(hostGemini, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{
		"access_token": "access", "refresh_token": "refresh",
		"expiry_date": time.Now().Add(time.Hour).UnixMilli(),
		"id_token":    "synthetic-id-token", "scope": "openid email", "token_type": "Bearer",
	})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(hostGemini, "oauth_creds.json")
	if err := os.WriteFile(source, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	profile := model.Profile{Provider: "gemini", Name: "imported"}
	adapter := New()
	copied, err := adapter.ImportHostAuth(t.Context(), profile)
	if err != nil || !copied {
		t.Fatalf("import = %v, %v; want true, nil", copied, err)
	}
	if !adapter.InspectAuth(t.Context(), profile).Authenticated {
		t.Fatal("imported Gemini profile was not authenticated")
	}
	home, err := config.ProfileHome("gemini", profile.Name)
	if err != nil {
		t.Fatal(err)
	}
	imported, err := os.ReadFile(filepath.Join(home, ".gemini", "oauth_creds.json"))
	if err != nil || string(imported) != string(raw) {
		t.Fatalf("destination did not receive host credentials: err=%v", err)
	}
	unchanged, err := os.ReadFile(source)
	if err != nil || string(unchanged) != string(raw) {
		t.Fatalf("host Gemini credentials changed: err=%v", err)
	}
}

func TestGeminiUsageLoadsProjectAndRetrievesOfficialQuota(t *testing.T) {
	home := setupGeminiProfile(t, "work")
	writeGeminiCredentials(t, home, map[string]any{
		"access_token":  "access-1",
		"refresh_token": "refresh-1",
		"expiry_date":   time.Now().Add(time.Hour).UnixMilli(),
		"token_type":    "Bearer",
	})
	if err := os.WriteFile(
		filepath.Join(home, ".gemini", "google_accounts.json"),
		[]byte(`{"active":"person@example.com"}`),
		0600,
	); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.ReadFile(filepath.Join("testdata", "retrieve_user_quota_v1.json"))
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer access-1" {
			t.Errorf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		switch request.URL.Path {
		case "/v1internal:loadCodeAssist":
			_, _ = writer.Write([]byte(`{"cloudaicompanionProject":"project-123"}`))
		case "/v1internal:retrieveUserQuota":
			body, _ := io.ReadAll(request.Body)
			if !strings.Contains(string(body), `"project":"project-123"`) {
				t.Errorf("retrieve body has no discovered project: %s", body)
			}
			_, _ = writer.Write(fixture)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	setGeminiTestEndpoints(t, server.URL)

	snapshot := New().RefreshUsage(context.Background(), model.Profile{Name: "work"})
	if snapshot.Status != model.UsageLive || snapshot.Source != model.SourceOfficialAPI {
		t.Fatalf("unexpected snapshot status: %+v", snapshot)
	}
	if snapshot.Account != "person@example.com" || !snapshot.AccountScope.Verifiable() {
		t.Fatalf("account scope was not resolved from Google account: %+v", snapshot)
	}
	if len(snapshot.Windows) != 2 {
		t.Fatalf("got %d windows, want 2", len(snapshot.Windows))
	}
}

func TestGeminiUsageRefreshesExpiredOAuthTokenAtomically(t *testing.T) {
	home := setupGeminiProfile(t, "expired")
	writeGeminiCredentials(t, home, map[string]any{
		"access_token":  "expired-access",
		"refresh_token": "refresh-2",
		"expiry_date":   time.Now().Add(-time.Hour).UnixMilli(),
		"token_type":    "Bearer",
		"id_token":      jwtWithEmail(t, "jwt@example.com"),
		"extra":         "preserved",
	})
	t.Setenv("NEXUS_GEMINI_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("NEXUS_GEMINI_OAUTH_CLIENT_SECRET", "client-secret")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "env-project")

	var tokenCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token":
			tokenCalls.Add(1)
			if err := request.ParseForm(); err != nil {
				t.Error(err)
			}
			if request.Form.Get("grant_type") != "refresh_token" ||
				request.Form.Get("refresh_token") != "refresh-2" ||
				request.Form.Get("client_id") != "client-id" ||
				request.Form.Get("client_secret") != "client-secret" {
				t.Errorf("unexpected refresh form: %v", request.Form)
			}
			_, _ = writer.Write([]byte(`{"access_token":"fresh-access","expires_in":3600,"token_type":"Bearer"}`))
		case "/v1internal:retrieveUserQuota":
			if request.Header.Get("Authorization") != "Bearer fresh-access" {
				t.Errorf("quota request used stale token: %q", request.Header.Get("Authorization"))
			}
			_, _ = writer.Write([]byte(
				`{"buckets":[{"modelId":"gemini-2.5-pro","remainingFraction":0.5,"tokenType":"REQUESTS"}]}`,
			))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	setGeminiTestEndpoints(t, server.URL)

	snapshot := New().RefreshUsage(context.Background(), model.Profile{Name: "expired"})
	if snapshot.Status != model.UsageLive || snapshot.Account != "jwt@example.com" {
		t.Fatalf("unexpected refreshed snapshot: %+v", snapshot)
	}
	if tokenCalls.Load() != 1 {
		t.Fatalf("got %d token calls, want 1", tokenCalls.Load())
	}
	updated, err := os.ReadFile(filepath.Join(home, ".gemini", "oauth_creds.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), `"access_token": "fresh-access"`) ||
		!strings.Contains(string(updated), `"extra": "preserved"`) {
		t.Fatalf("credentials were not atomically updated while preserving fields: %s", updated)
	}
	if _, err := os.Stat(filepath.Join(home, ".gemini", "oauth_creds.json.bak")); err != nil {
		t.Fatalf("atomic write backup missing: %v", err)
	}
}

func TestGeminiOAuthInvalidGrantWritesRevokedSentinel(t *testing.T) {
	home := setupGeminiProfile(t, "revoked")
	writeGeminiCredentials(t, home, map[string]any{
		"access_token":  "expired-access",
		"refresh_token": "bad-refresh",
		"expiry_date":   time.Now().Add(-time.Hour).UnixMilli(),
	})
	t.Setenv("NEXUS_GEMINI_OAUTH_CLIENT_ID", "client-id")
	t.Setenv("NEXUS_GEMINI_OAUTH_CLIENT_SECRET", "client-secret")

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer server.Close()
	setGeminiTestEndpoints(t, server.URL)

	snapshot := New().RefreshUsage(context.Background(), model.Profile{Name: "revoked"})
	if snapshot.Status != model.UsageUnknown || !strings.Contains(snapshot.Error, "invalid_grant") {
		t.Fatalf("invalid grant did not return UNKNOWN: %+v", snapshot)
	}
	revoked, reason := oauthutil.IsRevoked(filepath.Join(home, ".gemini"))
	if !revoked || reason != hashGeminiRefreshToken("bad-refresh") {
		t.Fatalf("revoked sentinel mismatch: revoked=%v reason=%q", revoked, reason)
	}
}

func setupGeminiProfile(t *testing.T, profile string) string {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("NEXUS_GEMINI_USAGE_API", "1")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT_ID", "")
	t.Setenv("NEXUS_GEMINI_OAUTH_CLIENT_ID", "")
	t.Setenv("NEXUS_GEMINI_OAUTH_CLIENT_SECRET", "")
	home := filepath.Join(dataDir, "profiles", "gemini", profile, "home")
	if err := os.MkdirAll(filepath.Join(home, ".gemini"), 0700); err != nil {
		t.Fatal(err)
	}
	return home
}

func writeGeminiCredentials(t *testing.T, home string, credentials map[string]any) {
	t.Helper()
	data, err := json.Marshal(credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gemini", "oauth_creds.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func setGeminiTestEndpoints(t *testing.T, serverURL string) {
	t.Helper()
	oldBaseURL := geminiUsageBaseURL
	oldTokenURL := geminiTokenURL
	oldClient := geminiUsageHTTPClient
	geminiUsageBaseURL = serverURL
	geminiTokenURL = serverURL + "/token"
	geminiUsageHTTPClient = http.DefaultClient
	t.Cleanup(func() {
		geminiUsageBaseURL = oldBaseURL
		geminiTokenURL = oldTokenURL
		geminiUsageHTTPClient = oldClient
	})
}

func jwtWithEmail(t *testing.T, email string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"email": email})
	if err != nil {
		t.Fatal(err)
	}
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}
