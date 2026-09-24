package voice

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func withVoiceConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("NEXUS_CONFIG_DIR", dir)
	return dir
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func mockHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func testUpdate(base string) SettingsUpdate {
	return SettingsUpdate{
		Enabled:             true,
		Language:            "pt-BR",
		TerminalEnabled:     true,
		NexusControlEnabled: true,
		AutomaticFallback:   true,
		MaxCaptureSeconds:   60,
		Providers: []ProviderUpdate{
			{ProviderConfig: ProviderConfig{ID: "groq-primary", Kind: ProviderGroq, Label: "Groq", Enabled: true, Priority: 1, Model: "whisper-large-v3-turbo", BaseURL: base, FreeEligible: true}, APIKey: "groq-secret-test"},
			{ProviderConfig: ProviderConfig{ID: "cloudflare-fallback", Kind: ProviderCloudflare, Label: "Cloudflare", Enabled: true, Priority: 2, Model: "@cf/openai/whisper-large-v3-turbo", AccountID: "acct", BaseURL: base, FreeEligible: true}, APIKey: "cf-secret-test"},
		},
	}
}

func TestStoreKeepsVoiceSecretsOutOfSettingsViewAndSettingsFile(t *testing.T) {
	dir := withVoiceConfigDir(t)
	store := NewStore()
	if _, err := store.SaveUpdate(testUpdate("https://example.invalid")); err != nil {
		t.Fatal(err)
	}
	settingsRaw, err := os.ReadFile(filepath.Join(dir, "voice-settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(settingsRaw), "secret-test") {
		t.Fatal("voice settings file leaked an API key")
	}
	service := NewServiceWith(store, nil)
	view, err := service.Settings()
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Providers) != 2 || !view.Providers[0].HasAPIKey || !view.Providers[1].HasAPIKey {
		t.Fatalf("expected redacted configured providers, got %+v", view.Providers)
	}
	data, _ := json.Marshal(view)
	if strings.Contains(string(data), "secret-test") {
		t.Fatal("settings API view leaked an API key")
	}
}

func TestStoreCanUpdateVoiceSettingsRepeatedly(t *testing.T) {
	withVoiceConfigDir(t)
	store := NewStore()
	update := testUpdate("https://example.invalid")
	if _, err := store.SaveUpdate(update); err != nil {
		t.Fatal(err)
	}
	update.Language = "en"
	update.MaxCaptureSeconds = 90
	if _, err := store.SaveUpdate(update); err != nil {
		t.Fatalf("second voice settings save failed: %v", err)
	}
	cfg, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Language != "en" || cfg.MaxCaptureSeconds != 90 {
		t.Fatalf("voice settings update was not persisted: %+v", cfg)
	}
}

func TestVoiceSettingsPinBuiltInProviderEndpoints(t *testing.T) {
	withVoiceConfigDir(t)
	store := NewStore()
	update := testUpdate("http://127.0.0.1:9999")
	update.Providers[0].APIKeyFile = "/etc/passwd"
	update.Providers[0].APIKeyEnv = "EVIL_ENV"
	update.Providers[1].APIKeyFile = "/etc/shadow"
	update.Providers[1].APIKeyEnv = "EVIL_CF_ENV"
	if _, err := store.SaveUpdate(update); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Providers[0].BaseURL != "https://api.groq.com" || cfg.Providers[0].APIKeyFile != "" || cfg.Providers[0].APIKeyEnv != "GROQ_API_KEY" {
		t.Fatalf("Groq provider accepted unsafe endpoint/credential indirection: %+v", cfg.Providers[0])
	}
	if cfg.Providers[1].BaseURL != "https://api.cloudflare.com/client/v4" || cfg.Providers[1].APIKeyFile != "" || cfg.Providers[1].APIKeyEnv != "CLOUDFLARE_API_TOKEN" {
		t.Fatalf("Cloudflare provider accepted unsafe endpoint/credential indirection: %+v", cfg.Providers[1])
	}
}

func TestSpeechRouterFallsBackFromGroq429ToCloudflare(t *testing.T) {
	withVoiceConfigDir(t)
	var groqCalls atomic.Int32
	var cloudflareCalls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.URL.Host == "api.groq.com" && strings.Contains(r.URL.Path, "/openai/v1/audio/transcriptions"):
			groqCalls.Add(1)
			return mockHTTPResponse(http.StatusTooManyRequests, "rate limit"), nil
		case r.URL.Host == "api.cloudflare.com" && strings.Contains(r.URL.Path, "/accounts/acct/ai/run/"):
			cloudflareCalls.Add(1)
			resp := mockHTTPResponse(http.StatusOK, `{"success":true,"result":{"text":"git status"}}`)
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		default:
			return mockHTTPResponse(http.StatusNotFound, "unexpected endpoint"), nil
		}
	})}

	store := NewStore()
	if _, err := store.SaveUpdate(testUpdate("https://ignored.invalid")); err != nil {
		t.Fatal(err)
	}
	service := NewServiceWith(store, client)
	result, err := service.Transcribe(context.Background(), TranscribeRequest{
		Audio: []byte("fake-audio"), Filename: "voice.webm", MIMEType: "audio/webm", DurationMS: 1200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "git status" || result.Provider != "cloudflare" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if groqCalls.Load() != 1 || cloudflareCalls.Load() != 1 {
		t.Fatalf("unexpected calls groq=%d cloudflare=%d", groqCalls.Load(), cloudflareCalls.Load())
	}
	if len(result.Attempts) != 2 || result.Attempts[0].Status != StatusRateLimited || result.Attempts[1].Status != StatusSuccess {
		t.Fatalf("unexpected attempts: %+v", result.Attempts)
	}
}

func TestSpeechRouterDoesNotCascadeInvalidAudio(t *testing.T) {
	withVoiceConfigDir(t)
	var cloudflareCalls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "api.groq.com" {
			return mockHTTPResponse(http.StatusBadRequest, "bad request"), nil
		}
		cloudflareCalls.Add(1)
		return mockHTTPResponse(http.StatusInternalServerError, "should not be called"), nil
	})}

	store := NewStore()
	if _, err := store.SaveUpdate(testUpdate("https://ignored.invalid")); err != nil {
		t.Fatal(err)
	}
	service := NewServiceWith(store, client)
	_, err := service.Transcribe(context.Background(), TranscribeRequest{Audio: []byte("bad"), Filename: "voice.webm"})
	var txErr *TranscriptionError
	if err == nil || !strings.Contains(err.Error(), "all configured") {
		t.Fatalf("expected transcription failure, got %v", err)
	}
	if !errors.As(err, &txErr) {
		t.Fatalf("expected typed transcription error, got %T", err)
	}
	if len(txErr.Attempts) != 1 || txErr.Attempts[0].Status != StatusInvalidRequest {
		t.Fatalf("unexpected attempts: %+v", txErr.Attempts)
	}
	if cloudflareCalls.Load() != 0 {
		t.Fatalf("invalid audio cascaded to fallback provider")
	}
}
