package claude

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/provider/oauthutil"
)

const (
	claudeOAuthTokenEndpoint = "https://platform.claude.com/v1/oauth/token"
	claudeCodeClientID       = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
)

var claudeOAuthTokenURL = claudeOAuthTokenEndpoint

type claudeOAuthCredential struct {
	AccessToken      string
	RefreshToken     string
	Email            string
	SubscriptionType string
	Scopes           []string
	ExpiresAt        time.Time
	HasExpiry        bool
	Path             string
	document         map[string]any
}

type claudeOAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	Error        any    `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func readClaudeOAuthCredentials(home string) (claudeOAuthCredential, bool, error) {
	path := filepath.Join(home, ".claude", ".credentials.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return claudeOAuthCredential{}, false, nil
	}
	if err != nil {
		return claudeOAuthCredential{}, false, err
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return claudeOAuthCredential{}, false, fmt.Errorf("decode Claude credentials: %w", err)
	}
	rawOAuth, exists := document["claudeAiOauth"]
	if !exists || rawOAuth == nil {
		return claudeOAuthCredential{}, false, nil
	}
	oauth, ok := rawOAuth.(map[string]any)
	if !ok {
		return claudeOAuthCredential{}, false, fmt.Errorf("claude credentials contain an invalid claudeAiOauth block")
	}
	credential := claudeOAuthCredential{
		AccessToken:      stringValue(oauth["accessToken"]),
		RefreshToken:     stringValue(oauth["refreshToken"]),
		Email:            stringValue(oauth["email"]),
		SubscriptionType: stringValue(oauth["subscriptionType"]),
		Scopes:           stringSliceValue(oauth["scopes"]),
		Path:             path,
		document:         document,
	}
	credential.ExpiresAt, credential.HasExpiry = parseClaudeExpiry(oauth["expiresAt"])
	return credential, true, nil
}

func refreshClaudeOAuth(ctx context.Context, home string) (claudeOAuthCredential, error) {
	lock := oauthutil.MutexFor(home)
	lock.Lock()
	defer lock.Unlock()

	credential, found, err := readClaudeOAuthCredentials(home)
	if err != nil {
		return claudeOAuthCredential{}, err
	}
	if !found {
		return claudeOAuthCredential{}, fmt.Errorf("claude OAuth credentials not found")
	}
	if !credential.HasExpiry || time.Now().Before(credential.ExpiresAt) {
		return credential, nil
	}
	if credential.RefreshToken == "" {
		return credential, fmt.Errorf("claude OAuth token expired without a refresh token")
	}
	if revoked, _ := oauthutil.IsRevoked(filepath.Dir(credential.Path)); revoked {
		return credential, fmt.Errorf("claude OAuth refresh token was previously rejected; login required")
	}

	payload, err := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     claudeCodeClientID,
		"refresh_token": credential.RefreshToken,
	})
	if err != nil {
		return credential, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeOAuthTokenURL, strings.NewReader(string(payload)))
	if err != nil {
		return credential, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "claude-code/"+claudeCodeVersion(ctx))

	resp, err := claudeUsageHTTPClient.Do(req)
	if err != nil {
		return credential, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, claudeUsageMaxBody))
	if err != nil {
		return credential, err
	}
	var token claudeOAuthTokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return credential, fmt.Errorf("claude OAuth refresh returned invalid JSON (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || strings.TrimSpace(token.AccessToken) == "" {
		code := oauthErrorCode(token.Error)
		if code == "" {
			code = fmt.Sprintf("http_%d", resp.StatusCode)
		}
		if code == "invalid_grant" {
			reason := refreshTokenFingerprint(credential.RefreshToken)
			_ = oauthutil.WriteRevokedSentinel(filepath.Dir(credential.Path), reason)
		}
		return credential, fmt.Errorf("claude OAuth refresh failed: %s", code)
	}

	oauth := credential.document["claudeAiOauth"].(map[string]any)
	oauth["accessToken"] = strings.TrimSpace(token.AccessToken)
	if replacement := strings.TrimSpace(token.RefreshToken); replacement != "" {
		oauth["refreshToken"] = replacement
	}
	expiresIn := token.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	oauth["expiresAt"] = time.Now().UTC().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	if scope := strings.TrimSpace(token.Scope); scope != "" {
		oauth["scopes"] = strings.Fields(scope)
	}
	encoded, err := json.MarshalIndent(credential.document, "", "  ")
	if err != nil {
		return credential, err
	}
	if err := oauthutil.AtomicWriteFile(credential.Path, append(encoded, '\n'), 0600); err != nil {
		return credential, err
	}
	refreshed, _, err := readClaudeOAuthCredentials(home)
	return refreshed, err
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case string:
		return strings.Fields(typed)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := stringValue(item); text != "" {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}

func oauthErrorCode(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		if code := stringValue(typed["type"]); code != "" {
			return code
		}
		return stringValue(typed["error"])
	default:
		return ""
	}
}

func refreshTokenFingerprint(refreshToken string) string {
	sum := sha256.Sum256([]byte(refreshToken))
	return "invalid_grant:" + hex.EncodeToString(sum[:])
}
