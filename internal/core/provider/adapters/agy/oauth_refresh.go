package agy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/security"
	"github.com/kivervinicius/ai-cli/internal/runtime"
)

const (
	agyOAuthTokenEndpoint = "https://oauth2.googleapis.com/token"
	agyOAuthSkew          = 2 * time.Minute
	agyOAuthCacheName     = "agy-oauth-client.json"
	agyOAuthRevokedSuffix = ".revoked"
)

var (
	agyClientIDPattern     = regexp.MustCompile(`[0-9]{9,14}-[a-z0-9]{32}\.apps\.googleusercontent\.com`)
	agyClientSecretPattern = regexp.MustCompile(`GOCSPX-[A-Za-z0-9_-]{28}`)

	agyOAuthHTTPClient = &http.Client{Timeout: 15 * time.Second}
	agyOAuthTokenURL   = agyOAuthTokenEndpoint

	agyOAuthMu       sync.Mutex
	agyOAuthLookPath = runtime.LookPath
	agyProfileLocks  sync.Map // home path → *sync.Mutex
)

type agyOAuthClientCache struct {
	BinPath    string         `json:"bin_path"`
	BinSize    int64          `json:"bin_size"`
	BinModUnix int64          `json:"bin_mod_unix"`
	ClientID   string         `json:"client_id"`
	ClientSec  string         `json:"client_secret"`
	Candidates []agyOAuthPair `json:"candidates,omitempty"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type agyOAuthPair struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func profileLock(home string) *sync.Mutex {
	actual, _ := agyProfileLocks.LoadOrStore(home, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

func oauthTokenPaths(home string) []string {
	return []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "antigravity-oauth-token"),
		filepath.Join(home, ".gemini", "antigravity-oauth-token"),
	}
}

func findOAuthTokenFile(home string) (string, []byte, error) {
	var lastErr error
	for _, path := range oauthTokenPaths(home) {
		data, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			continue
		}
		if len(data) == 0 {
			lastErr = fmt.Errorf("empty oauth token file")
			continue
		}
		return path, data, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("oauth token file not found")
	}
	return "", nil, lastErr
}

func extractRefreshToken(data []byte) string {
	var tok struct {
		Token struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"token"`
	}
	if json.Unmarshal(data, &tok) != nil {
		return ""
	}
	return strings.TrimSpace(tok.Token.RefreshToken)
}

func refreshTokenHash(refresh string) string {
	sum := sha256.Sum256([]byte(refresh))
	return hex.EncodeToString(sum[:])
}

func revokedSentinelPath(tokenPath string) string {
	return tokenPath + agyOAuthRevokedSuffix
}

func refreshTokenRevoked(home, refresh string) bool {
	want := refreshTokenHash(refresh)
	for _, path := range oauthTokenPaths(home) {
		data, err := os.ReadFile(revokedSentinelPath(path))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(data)) == want {
			return true
		}
	}
	return false
}

func markRefreshTokenRevoked(tokenPath, refresh string) {
	_ = os.WriteFile(revokedSentinelPath(tokenPath), []byte(refreshTokenHash(refresh)+"\n"), 0600)
}

func clearRevokedSentinel(tokenPath string) {
	_ = os.Remove(revokedSentinelPath(tokenPath))
}

func oauthCachePath() (string, error) {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".cache")
	}
	dir := filepath.Join(base, "ai-manager")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, agyOAuthCacheName), nil
}

func loadOAuthClientCache() (agyOAuthClientCache, bool) {
	path, err := oauthCachePath()
	if err != nil {
		return agyOAuthClientCache{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return agyOAuthClientCache{}, false
	}
	var cache agyOAuthClientCache
	if json.Unmarshal(data, &cache) != nil {
		return agyOAuthClientCache{}, false
	}
	return cache, true
}

func saveOAuthClientCache(cache agyOAuthClientCache) {
	path, err := oauthCachePath()
	if err != nil {
		return
	}
	cache.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

func extractOAuthPairsFromBinary(binPath string) ([]agyOAuthPair, error) {
	data, err := os.ReadFile(binPath)
	if err != nil {
		return nil, err
	}
	ids := uniqueStrings(agyClientIDPattern.FindAllString(string(data), -1))
	secrets := uniqueStrings(agyClientSecretPattern.FindAllString(string(data), -1))
	if len(ids) == 0 || len(secrets) == 0 {
		return nil, fmt.Errorf("agy binary has no embedded google oauth client credentials")
	}
	pairs := make([]agyOAuthPair, 0, len(ids)*len(secrets))
	for _, id := range ids {
		for _, secret := range secrets {
			pairs = append(pairs, agyOAuthPair{ClientID: id, ClientSecret: secret})
		}
	}
	return pairs, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func envOAuthPair() (agyOAuthPair, bool) {
	id := strings.TrimSpace(os.Getenv("NEXUS_AGY_OAUTH_CLIENT_ID"))
	secret := strings.TrimSpace(os.Getenv("NEXUS_AGY_OAUTH_CLIENT_SECRET"))
	if id == "" || secret == "" {
		return agyOAuthPair{}, false
	}
	return agyOAuthPair{ClientID: id, ClientSecret: secret}, true
}

// agyOAuthClientCandidates returns candidate (client_id, client_secret) pairs.
// Env overrides win; otherwise pairs are extracted from the agy binary and
// cached by binary size+mtime so the 200MB scan runs at most once per upgrade.
func agyOAuthClientCandidates() ([]agyOAuthPair, error) {
	if pair, ok := envOAuthPair(); ok {
		return []agyOAuthPair{pair}, nil
	}

	bin, err := agyOAuthLookPath("agy")
	if err != nil {
		return nil, fmt.Errorf("locate agy binary: %w", err)
	}
	info, err := os.Stat(bin)
	if err != nil {
		return nil, err
	}

	if cache, ok := loadOAuthClientCache(); ok &&
		cache.BinPath == bin &&
		cache.BinSize == info.Size() &&
		cache.BinModUnix == info.ModTime().Unix() {
		if cache.ClientID != "" && cache.ClientSec != "" {
			ordered := []agyOAuthPair{{ClientID: cache.ClientID, ClientSecret: cache.ClientSec}}
			for _, candidate := range cache.Candidates {
				if candidate.ClientID == cache.ClientID && candidate.ClientSecret == cache.ClientSec {
					continue
				}
				ordered = append(ordered, candidate)
			}
			if len(ordered) > 0 {
				return ordered, nil
			}
		}
		if len(cache.Candidates) > 0 {
			return cache.Candidates, nil
		}
	}

	pairs, err := extractOAuthPairsFromBinary(bin)
	if err != nil {
		return nil, err
	}
	saveOAuthClientCache(agyOAuthClientCache{
		BinPath:    bin,
		BinSize:    info.Size(),
		BinModUnix: info.ModTime().Unix(),
		Candidates: pairs,
	})
	return pairs, nil
}

func rememberWorkingOAuthPair(pair agyOAuthPair, all []agyOAuthPair) {
	bin, err := agyOAuthLookPath("agy")
	if err != nil {
		return
	}
	info, err := os.Stat(bin)
	if err != nil {
		return
	}
	saveOAuthClientCache(agyOAuthClientCache{
		BinPath:    bin,
		BinSize:    info.Size(),
		BinModUnix: info.ModTime().Unix(),
		ClientID:   pair.ClientID,
		ClientSec:  pair.ClientSecret,
		Candidates: all,
	})
}

type googleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func refreshAccessTokenHTTP(ctx context.Context, pair agyOAuthPair, refreshToken string) (googleTokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", pair.ClientID)
	form.Set("client_secret", pair.ClientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, agyOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return googleTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := agyOAuthHTTPClient.Do(req)
	if err != nil {
		return googleTokenResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return googleTokenResponse{}, err
	}
	var parsed googleTokenResponse
	if json.Unmarshal(body, &parsed) != nil {
		return googleTokenResponse{}, fmt.Errorf("oauth token response is not JSON (status %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || parsed.AccessToken == "" {
		errCode := strings.TrimSpace(parsed.Error)
		if errCode == "" {
			errCode = fmt.Sprintf("http_%d", resp.StatusCode)
		}
		return parsed, fmt.Errorf("oauth refresh failed: %s", errCode)
	}
	return parsed, nil
}

func tokenNeedsRefresh(data []byte, now time.Time) bool {
	var tok struct {
		Token struct {
			Expiry string `json:"expiry"`
		} `json:"token"`
	}
	if json.Unmarshal(data, &tok) != nil || strings.TrimSpace(tok.Token.Expiry) == "" {
		return true
	}
	expTime, err := time.Parse(time.RFC3339Nano, tok.Token.Expiry)
	if err != nil {
		expTime, err = time.Parse(time.RFC3339, tok.Token.Expiry)
		if err != nil {
			return true
		}
	}
	return !now.Before(expTime.Add(-agyOAuthSkew))
}

func writeOAuthTokenAtomic(path string, data []byte, accessToken, refreshToken string, expiry time.Time) error {
	var blob map[string]any
	if err := json.Unmarshal(data, &blob); err != nil {
		return fmt.Errorf("decode oauth token blob: %w", err)
	}
	tokenObj, _ := blob["token"].(map[string]any)
	if tokenObj == nil {
		tokenObj = map[string]any{}
		blob["token"] = tokenObj
	}
	tokenObj["access_token"] = accessToken
	tokenObj["expiry"] = expiry.UTC().Format(time.RFC3339Nano)
	if strings.TrimSpace(refreshToken) != "" {
		tokenObj["refresh_token"] = refreshToken
	}

	encoded, err := json.MarshalIndent(blob, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "antigravity-oauth-token-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(encoded); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	bak := path + ".bak"
	_ = os.Remove(bak)
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, bak); err != nil {
			return err
		}
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Rename(bak, path) // best-effort restore
		return err
	}
	return nil
}

// ensureFreshAccessToken refreshes the AGY access_token via a pure HTTPS
// refresh_token grant when it is expired or within the skew window. It never
// spawns the AGY CLI and never opens a browser.
func ensureFreshAccessToken(ctx context.Context, home string) error {
	agyOAuthMu.Lock()
	defer agyOAuthMu.Unlock()

	lock := profileLock(home)
	lock.Lock()
	defer lock.Unlock()

	path, data, err := findOAuthTokenFile(home)
	if err != nil {
		return err
	}
	if !tokenNeedsRefresh(data, time.Now()) {
		return nil
	}
	refresh := extractRefreshToken(data)
	if refresh == "" {
		return fmt.Errorf("access token expired and no refresh_token is available")
	}
	if refreshTokenRevoked(home, refresh) {
		return fmt.Errorf("refresh_token previously rejected as invalid_grant; re-login required")
	}

	pairs, err := agyOAuthClientCandidates()
	if err != nil {
		return err
	}

	var lastErr error
	allInvalidGrant := true
	tried := 0
	for _, pair := range pairs {
		tried++
		resp, refreshErr := refreshAccessTokenHTTP(ctx, pair, refresh)
		if refreshErr != nil {
			lastErr = refreshErr
			if !strings.Contains(refreshErr.Error(), "invalid_grant") {
				allInvalidGrant = false
			}
			if debug := os.Getenv("NEXUS_AGY_DEBUG") == "1" || os.Getenv("NEXUS_DEBUG") == "1"; debug {
				slog.Debug("AGY oauth refresh candidate failed", "err", security.Redact(refreshErr.Error()))
			}
			continue
		}
		expiresIn := resp.ExpiresIn
		if expiresIn <= 0 {
			expiresIn = 3600
		}
		expiry := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
		newRefresh := strings.TrimSpace(resp.RefreshToken)
		if newRefresh == "" {
			newRefresh = refresh
		}
		if err := writeOAuthTokenAtomic(path, data, resp.AccessToken, newRefresh, expiry); err != nil {
			return err
		}
		clearRevokedSentinel(path)
		rememberWorkingOAuthPair(pair, pairs)
		return nil
	}

	if tried > 0 && allInvalidGrant {
		markRefreshTokenRevoked(path, refresh)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("oauth refresh failed for all client credential candidates")
	}
	return lastErr
}
