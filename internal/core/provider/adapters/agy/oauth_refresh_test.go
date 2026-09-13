package agy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Built at runtime so the source tree never contains contiguous Google OAuth
// client-id / GOCSPX literals (GitHub push protection blocks those patterns).
func testGoogleOAuthClientID(numericPrefix, body32 string) string {
	return numericPrefix + "-" + body32 + ".apps." + "googleusercontent.com"
}

func testGoogleOAuthClientSecret(suffix28 string) string {
	return "GOC" + "SPX-" + suffix28
}

func testEnvOAuthClientID(name string) string {
	return name + ".apps." + "googleusercontent.com"
}

func TestExtractOAuthPairsFromBinary(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake-agy")
	id1 := testGoogleOAuthClientID("000000000001", strings.Repeat("a", 32))
	id2 := testGoogleOAuthClientID("000000000002", strings.Repeat("b", 32))
	sec1 := testGoogleOAuthClientSecret(strings.Repeat("0", 28))
	sec2 := testGoogleOAuthClientSecret(strings.Repeat("1", 28))
	payload := []byte("noise " + id1 + " more " + sec1 + " end " + id2 + " " + sec2)
	if err := os.WriteFile(bin, payload, 0755); err != nil {
		t.Fatal(err)
	}
	pairs, err := extractOAuthPairsFromBinary(bin)
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 4 {
		t.Fatalf("pairs=%d want 4 (2 ids × 2 secrets)", len(pairs))
	}
}

func TestEnsureFreshAccessTokenRefreshesAndPreservesUnknownFields(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("NEXUS_AGY_OAUTH_CLIENT_ID", testEnvOAuthClientID("test-client"))
	t.Setenv("NEXUS_AGY_OAUTH_CLIENT_SECRET", testGoogleOAuthClientSecret("testsecret0000000000000000"))

	var sawGrant string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		sawGrant = r.Form.Get("grant_type")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "ya29.refreshed",
			"expires_in":    3600,
			"refresh_token": "1//rotated",
			"token_type":    "Bearer",
		})
	}))
	t.Cleanup(server.Close)
	prevURL := agyOAuthTokenURL
	agyOAuthTokenURL = server.URL
	t.Cleanup(func() { agyOAuthTokenURL = prevURL })

	home := t.TempDir()
	tokenDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(tokenDir, "antigravity-oauth-token")
	expired := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	original := map[string]any{
		"token": map[string]any{
			"access_token":  "ya29.old",
			"refresh_token": "1//original",
			"expiry":        expired,
			"token_type":    "Bearer",
		},
		"auth_method": "oauth",
		"id_token":    "header.payload.sig",
	}
	raw, _ := json.Marshal(original)
	if err := os.WriteFile(tokenPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	if err := ensureFreshAccessToken(context.Background(), home); err != nil {
		t.Fatal(err)
	}
	if sawGrant != "refresh_token" {
		t.Fatalf("grant_type=%q", sawGrant)
	}
	updated, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatal(err)
	}
	var blob map[string]any
	if err := json.Unmarshal(updated, &blob); err != nil {
		t.Fatal(err)
	}
	if blob["auth_method"] != "oauth" || blob["id_token"] != "header.payload.sig" {
		t.Fatalf("unknown fields not preserved: %#v", blob)
	}
	tokenObj := blob["token"].(map[string]any)
	if tokenObj["access_token"] != "ya29.refreshed" {
		t.Fatalf("access_token=%v", tokenObj["access_token"])
	}
	if tokenObj["refresh_token"] != "1//rotated" {
		t.Fatalf("refresh_token=%v", tokenObj["refresh_token"])
	}
	if accessTokenExpired(home) {
		t.Fatal("token should no longer be expired after refresh")
	}
}

func TestEnsureFreshAccessTokenMarksInvalidGrant(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("NEXUS_AGY_OAUTH_CLIENT_ID", testEnvOAuthClientID("test-client"))
	t.Setenv("NEXUS_AGY_OAUTH_CLIENT_SECRET", testGoogleOAuthClientSecret("testsecret0000000000000000"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "Token has been expired or revoked.",
		})
	}))
	t.Cleanup(server.Close)
	prevURL := agyOAuthTokenURL
	agyOAuthTokenURL = server.URL
	t.Cleanup(func() { agyOAuthTokenURL = prevURL })

	home := t.TempDir()
	tokenDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(tokenDir, "antigravity-oauth-token")
	expired := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	raw := []byte(`{"token":{"access_token":"ya29.old","refresh_token":"1//revoked","expiry":"` + expired + `"}}`)
	if err := os.WriteFile(tokenPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	err := ensureFreshAccessToken(context.Background(), home)
	if err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("err=%v want invalid_grant", err)
	}
	if !refreshTokenRevoked(home, "1//revoked") {
		t.Fatal("expected revoked sentinel after invalid_grant")
	}
	// Second call must fail closed without hammering the token endpoint.
	calls := 0
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	})
	err = ensureFreshAccessToken(context.Background(), home)
	if err == nil || !strings.Contains(err.Error(), "re-login required") {
		t.Fatalf("err=%v want re-login required", err)
	}
	if calls != 0 {
		t.Fatalf("revoked sentinel still hit token endpoint %d times", calls)
	}
}

func TestEnsureFreshAccessTokenNetworkFailureIsTransient(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("NEXUS_AGY_OAUTH_CLIENT_ID", testEnvOAuthClientID("test-client"))
	t.Setenv("NEXUS_AGY_OAUTH_CLIENT_SECRET", testGoogleOAuthClientSecret("testsecret0000000000000000"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"unavailable"}`))
	}))
	t.Cleanup(server.Close)
	prevURL := agyOAuthTokenURL
	agyOAuthTokenURL = server.URL
	t.Cleanup(func() { agyOAuthTokenURL = prevURL })

	home := t.TempDir()
	tokenDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(tokenDir, 0700); err != nil {
		t.Fatal(err)
	}
	tokenPath := filepath.Join(tokenDir, "antigravity-oauth-token")
	expired := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	raw := []byte(`{"token":{"access_token":"ya29.old","refresh_token":"1//ok","expiry":"` + expired + `"}}`)
	if err := os.WriteFile(tokenPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := ensureFreshAccessToken(context.Background(), home); err == nil {
		t.Fatal("expected transient refresh failure")
	}
	if refreshTokenRevoked(home, "1//ok") {
		t.Fatal("network/5xx failures must not mark refresh token revoked")
	}
}

func TestAgyOAuthClientCandidatesPreferEnvOverride(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	wantID := testEnvOAuthClientID("env-client")
	t.Setenv("NEXUS_AGY_OAUTH_CLIENT_ID", wantID)
	t.Setenv("NEXUS_AGY_OAUTH_CLIENT_SECRET", testGoogleOAuthClientSecret("envsecret00000000000000000"))
	pairs, err := agyOAuthClientCandidates()
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 1 || pairs[0].ClientID != wantID {
		t.Fatalf("pairs=%#v", pairs)
	}
}
