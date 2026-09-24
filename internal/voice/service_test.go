package voice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestSpeechRouterFallsBackFromGroq429ToCloudflare(t *testing.T) {
	withVoiceConfigDir(t)
	var groqCalls atomic.Int32
	var cloudflareCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/openai/v1/audio/transcriptions") {
			groqCalls.Add(1)
			http.Error(w, "rate limit", http.StatusTooManyRequests)
			return
		}
		if strings.Contains(r.URL.Path, "/accounts/acct/ai/run/") {
			cloudflareCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"result":{"text":"git status"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	store := NewStore()
	if _, err := store.SaveUpdate(testUpdate(server.URL)); err != nil {
		t.Fatal(err)
	}
	service := NewServiceWith(store, server.Client())
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/openai/v1/audio/transcriptions") {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		cloudflareCalls.Add(1)
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	defer server.Close()

	store := NewStore()
	if _, err := store.SaveUpdate(testUpdate(server.URL)); err != nil {
		t.Fatal(err)
	}
	service := NewServiceWith(store, server.Client())
	_, err := service.Transcribe(context.Background(), TranscribeRequest{Audio: []byte("bad"), Filename: "voice.webm"})
	var txErr *TranscriptionError
	if err == nil || !strings.Contains(err.Error(), "all configured") {
		t.Fatalf("expected transcription failure, got %v", err)
	}
	if !AsTranscriptionError(err, &txErr) {
		t.Fatalf("expected typed transcription error, got %T", err)
	}
	if len(txErr.Attempts) != 1 || txErr.Attempts[0].Status != StatusInvalidRequest {
		t.Fatalf("unexpected attempts: %+v", txErr.Attempts)
	}
	if cloudflareCalls.Load() != 0 {
		t.Fatalf("invalid audio cascaded to fallback provider")
	}
}

func AsTranscriptionError(err error, target **TranscriptionError) bool {
	if err == nil {
		return false
	}
	value, ok := err.(*TranscriptionError)
	if ok {
		*target = value
	}
	return ok
}
