package gemini

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/provider/oauthutil"
)

const (
	geminiUsageAPIBase    = "https://cloudcode-pa.googleapis.com"
	geminiOAuthTokenURL   = "https://oauth2.googleapis.com/token"
	geminiOAuthExpirySkew = time.Minute
)

var (
	geminiUsageHTTPClient = oauthutil.HTTPClient(15 * time.Second)
	geminiUsageBaseURL    = geminiUsageAPIBase
	geminiTokenURL        = geminiOAuthTokenURL
	geminiUsageNow        = time.Now
)

type geminiOAuthCredentials struct {
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiryDate   json.RawMessage `json:"expiry_date"`
	TokenType    string          `json:"token_type"`
	IDToken      string          `json:"id_token"`
}

type geminiQuotaResponse struct {
	Buckets []geminiQuotaBucket `json:"buckets"`
}

type geminiQuotaBucket struct {
	ModelID           string          `json:"modelId"`
	RemainingFraction *float64        `json:"remainingFraction"`
	RemainingAmount   json.RawMessage `json:"remainingAmount"`
	ResetTime         string          `json:"resetTime"`
	TokenType         string          `json:"tokenType"`
}

func (a *Adapter) fetchLiveUsage(ctx context.Context, p model.Profile) (model.UsageSnapshot, bool) {
	snap := unknownUsageSnapshot(p)
	if oauthutil.KillSwitch("NEXUS_GEMINI_USAGE_API") {
		return snap, false
	}

	home, err := profileHome(p)
	if err != nil {
		snap.Error = err.Error()
		return snap, false
	}
	credentialsPath := filepath.Join(home, ".gemini", "oauth_creds.json")
	credentials, raw, err := readGeminiOAuthCredentials(credentialsPath)
	if err != nil {
		snap.Error = err.Error()
		return snap, false
	}
	if tokenNeedsRefresh(credentials, geminiUsageNow()) {
		credentials, raw, err = refreshGeminiAccessToken(ctx, credentialsPath, credentials, raw)
		if err != nil {
			snap.Error = err.Error()
			return snap, false
		}
	}
	if strings.TrimSpace(credentials.AccessToken) == "" {
		snap.Error = "gemini oauth access_token is unavailable"
		return snap, false
	}

	project := configuredGeminiProject(home)
	if project == "" {
		project, err = loadCodeAssistProject(ctx, credentials.AccessToken)
		if err != nil {
			snap.Error = err.Error()
			return snap, false
		}
	}

	quota, err := retrieveGeminiQuota(ctx, credentials.AccessToken, project)
	if err != nil {
		snap.Error = err.Error()
		return snap, false
	}
	windows := parseGeminiQuotaBuckets(quota.Buckets)
	if len(windows) == 0 {
		snap.Error = "gemini quota API returned no usable buckets"
		return snap, false
	}

	email := resolveGeminiEmail(home, credentials.IDToken)
	snap.Status = model.UsageLive
	snap.Source = model.SourceOfficialAPI
	snap.Confidence = model.UsageConfidenceHigh
	snap.Windows = windows
	snap.Account = email
	if p.AccountScope.Verifiable() {
		snap.AccountScope = p.AccountScope
	} else if email != "" {
		snap.AccountScope = model.AccountScope{
			ProviderID:      string(model.ProviderGemini),
			ProfileID:       p.Name,
			AccountID:       email,
			IdentityVersion: "1",
		}
	}
	return snap, true
}

func profileHome(p model.Profile) (string, error) {
	return config.ProfileHome(string(model.ProviderGemini), p.Name)
}

func readGeminiOAuthCredentials(path string) (geminiOAuthCredentials, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return geminiOAuthCredentials{}, nil, fmt.Errorf("read gemini oauth credentials: %w", err)
	}
	var credentials geminiOAuthCredentials
	if err := json.Unmarshal(data, &credentials); err != nil {
		return geminiOAuthCredentials{}, nil, fmt.Errorf("decode gemini oauth credentials: %w", err)
	}
	return credentials, data, nil
}

func tokenNeedsRefresh(credentials geminiOAuthCredentials, now time.Time) bool {
	if strings.TrimSpace(credentials.AccessToken) == "" {
		return true
	}
	expiry, ok := parseExpiryDate(credentials.ExpiryDate)
	return ok && !now.Before(expiry.Add(-geminiOAuthExpirySkew))
}

func parseExpiryDate(raw json.RawMessage) (time.Time, bool) {
	value := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if value == "" || value == "null" {
		return time.Time{}, false
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	if n > 10_000_000_000 {
		return time.UnixMilli(n), true
	}
	return time.Unix(n, 0), true
}

func refreshGeminiAccessToken(
	ctx context.Context,
	path string,
	credentials geminiOAuthCredentials,
	raw []byte,
) (geminiOAuthCredentials, []byte, error) {
	lock := oauthutil.MutexFor(path)
	lock.Lock()
	defer lock.Unlock()

	current, currentRaw, err := readGeminiOAuthCredentials(path)
	if err != nil {
		return credentials, raw, err
	}
	if !tokenNeedsRefresh(current, geminiUsageNow()) {
		return current, currentRaw, nil
	}
	refreshToken := strings.TrimSpace(current.RefreshToken)
	if refreshToken == "" {
		return current, currentRaw, fmt.Errorf("gemini access token expired and no refresh_token is available")
	}
	refreshHash := hashGeminiRefreshToken(refreshToken)
	if revoked, reason := oauthutil.IsRevoked(filepath.Dir(path)); revoked && reason == refreshHash {
		return current, currentRaw, fmt.Errorf("gemini refresh_token was rejected; re-login required")
	}
	clientID := strings.TrimSpace(os.Getenv("NEXUS_GEMINI_OAUTH_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("NEXUS_GEMINI_OAUTH_CLIENT_SECRET"))
	if clientID == "" || clientSecret == "" {
		return current, currentRaw, fmt.Errorf(
			"gemini oauth refresh requires NEXUS_GEMINI_OAUTH_CLIENT_ID and NEXUS_GEMINI_OAUTH_CLIENT_SECRET",
		)
	}

	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, geminiTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return current, currentRaw, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := geminiUsageHTTPClient.Do(request)
	if err != nil {
		return current, currentRaw, fmt.Errorf("refresh gemini oauth token: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return current, currentRaw, err
	}
	var tokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int64  `json:"expires_in"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return current, currentRaw, fmt.Errorf("gemini oauth refresh returned invalid JSON")
	}
	if response.StatusCode != http.StatusOK || tokenResponse.AccessToken == "" {
		code := strings.TrimSpace(tokenResponse.Error)
		if code == "" {
			code = fmt.Sprintf("http_%d", response.StatusCode)
		}
		if code == "invalid_grant" {
			_ = oauthutil.WriteRevokedSentinel(filepath.Dir(path), refreshHash)
		}
		return current, currentRaw, fmt.Errorf("gemini oauth refresh failed: %s", code)
	}

	var blob map[string]any
	if err := json.Unmarshal(currentRaw, &blob); err != nil {
		return current, currentRaw, err
	}
	expiresIn := tokenResponse.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	blob["access_token"] = tokenResponse.AccessToken
	blob["expiry_date"] = geminiUsageNow().UTC().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	if tokenResponse.RefreshToken != "" {
		blob["refresh_token"] = tokenResponse.RefreshToken
	}
	if tokenResponse.TokenType != "" {
		blob["token_type"] = tokenResponse.TokenType
	}
	updatedRaw, err := json.MarshalIndent(blob, "", "  ")
	if err != nil {
		return current, currentRaw, err
	}
	if err := oauthutil.AtomicWriteFile(path, updatedRaw, 0600); err != nil {
		return current, currentRaw, err
	}
	updated, _, err := readGeminiOAuthCredentials(path)
	return updated, updatedRaw, err
}

func hashGeminiRefreshToken(refreshToken string) string {
	sum := sha256.Sum256([]byte(refreshToken))
	return hex.EncodeToString(sum[:])
}

func configuredGeminiProject(home string) string {
	for _, key := range []string{"GOOGLE_CLOUD_PROJECT", "GOOGLE_CLOUD_PROJECT_ID"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	for _, path := range []string{
		filepath.Join(home, ".gemini", "settings.json"),
		filepath.Join(home, ".config", "gemini", "settings.json"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var settings any
		if json.Unmarshal(data, &settings) == nil {
			if project := findProjectID(settings); project != "" {
				return project
			}
		}
	}
	return ""
}

func loadCodeAssistProject(ctx context.Context, accessToken string) (string, error) {
	body := map[string]any{
		"metadata": map[string]string{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}
	var response any
	if err := postGeminiJSON(ctx, accessToken, "/v1internal:loadCodeAssist", body, &response); err != nil {
		return "", fmt.Errorf("load Gemini Code Assist project: %w", err)
	}
	project := findProjectID(response)
	if project == "" {
		return "", fmt.Errorf("load Gemini Code Assist project: response has no project id")
	}
	return project, nil
}

func findProjectID(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{
			"cloudaicompanionProject",
			"cloudAiCompanionProject",
			"projectId",
			"projectID",
			"project",
		} {
			if candidate, ok := typed[key]; ok {
				if text, ok := candidate.(string); ok && strings.TrimSpace(text) != "" {
					return strings.TrimSpace(text)
				}
				if nested := findProjectID(candidate); nested != "" {
					return nested
				}
			}
		}
		for _, candidate := range typed {
			if nested := findProjectID(candidate); nested != "" {
				return nested
			}
		}
	case []any:
		for _, candidate := range typed {
			if nested := findProjectID(candidate); nested != "" {
				return nested
			}
		}
	}
	return ""
}

func retrieveGeminiQuota(ctx context.Context, accessToken, project string) (geminiQuotaResponse, error) {
	var response geminiQuotaResponse
	body := map[string]string{"project": project}
	if err := postGeminiJSON(ctx, accessToken, "/v1internal:retrieveUserQuota", body, &response); err != nil {
		return response, fmt.Errorf("retrieve Gemini user quota: %w", err)
	}
	return response, nil
}

func postGeminiJSON(ctx context.Context, accessToken, path string, body, destination any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(geminiUsageBaseURL, "/")+path,
		bytes.NewReader(encoded),
	)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := geminiUsageHTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("http status %d", response.StatusCode)
	}
	if err := json.Unmarshal(responseBody, destination); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func parseGeminiQuotaBuckets(buckets []geminiQuotaBucket) []model.UsageWindow {
	windows := make([]model.UsageWindow, 0, len(buckets))
	for _, bucket := range buckets {
		if strings.TrimSpace(bucket.ModelID) == "" {
			continue
		}
		window := model.UsageWindow{
			Kind:  geminiQuotaKind(bucket.TokenType),
			Group: strings.TrimSpace(bucket.ModelID),
			Unit:  strings.ToLower(strings.TrimSpace(bucket.TokenType)),
		}
		if window.Unit == "" {
			window.Unit = "quota"
		}
		if bucket.RemainingFraction != nil {
			remainingPercent := clampPercent(*bucket.RemainingFraction * 100)
			usedPercent := 100 - remainingPercent
			window.RemainingPercent = &remainingPercent
			window.UsedPercent = &usedPercent
		}
		if amount, ok := parseRemainingAmount(bucket.RemainingAmount); ok {
			window.Remaining = &amount
		}
		if reset, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(bucket.ResetTime)); err == nil {
			window.ResetTime = &reset
		}
		windows = append(windows, window)
	}
	return windows
}

func geminiQuotaKind(tokenType string) string {
	normalized := strings.ToLower(strings.TrimSpace(tokenType))
	switch normalized {
	case "request", "requests":
		return "requests"
	case "":
		return "quota"
	default:
		return normalized
	}
}

func parseRemainingAmount(raw json.RawMessage) (float64, bool) {
	value := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if value == "" || value == "null" {
		return 0, false
	}
	amount, err := strconv.ParseFloat(value, 64)
	return amount, err == nil
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

func resolveGeminiEmail(home, idToken string) string {
	data, err := os.ReadFile(filepath.Join(home, ".gemini", "google_accounts.json"))
	if err == nil {
		var accounts struct {
			Active string `json:"active"`
		}
		if json.Unmarshal(data, &accounts) == nil && strings.TrimSpace(accounts.Active) != "" {
			return strings.TrimSpace(accounts.Active)
		}
	}
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	return strings.TrimSpace(claims.Email)
}
