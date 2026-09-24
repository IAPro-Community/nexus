package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/security"
)

const maxAudioBytes = 25 << 20

const developerPrompt = "Nexus, Codex, Claude, Gemini, Antigravity, OpenCode, Git, GitHub, npm, pnpm, yarn, bun, Nx, Docker, kubectl, TypeScript, JavaScript, Go, React"

type Service struct {
	store  *Store
	client *http.Client
}

func NewService() *Service {
	return NewServiceWith(NewStore(), &http.Client{Timeout: 60 * time.Second})
}

func NewServiceWith(store *Store, client *http.Client) *Service {
	if store == nil {
		store = NewStore()
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &Service{store: store, client: client}
}

func (s *Service) Settings() (SettingsView, error) {
	cfg, err := s.store.LoadSettings()
	if err != nil {
		return SettingsView{}, err
	}
	usage, err := s.store.LoadUsage()
	if err != nil {
		return SettingsView{}, err
	}
	view := SettingsView{
		Version:             cfg.Version,
		Enabled:             cfg.Enabled,
		Language:            cfg.Language,
		TerminalEnabled:     cfg.TerminalEnabled,
		NexusControlEnabled: cfg.NexusControlEnabled,
		AutomaticFallback:   cfg.AutomaticFallback,
		MaxCaptureSeconds:   cfg.MaxCaptureSeconds,
		Providers:           make([]ProviderView, 0, len(cfg.Providers)),
	}
	for _, p := range cfg.Providers {
		key, _ := s.store.ResolveAPIKey(p)
		account := s.store.ResolveAccountID(p)
		configured := strings.TrimSpace(key) != ""
		if p.Kind == ProviderCloudflare {
			configured = configured && account != ""
		}
		view.Providers = append(view.Providers, ProviderView{
			ProviderConfig: p,
			HasAPIKey:      strings.TrimSpace(key) != "",
			Configured:     configured,
			Usage:          usage[p.ID],
		})
	}
	return view, nil
}

func (s *Service) UpdateSettings(update SettingsUpdate) (SettingsView, error) {
	if _, err := s.store.SaveUpdate(update); err != nil {
		return SettingsView{}, err
	}
	return s.Settings()
}

func (s *Service) Transcribe(ctx context.Context, input TranscribeRequest) (TranscribeResult, error) {
	if len(input.Audio) == 0 {
		return TranscribeResult{}, &TranscriptionError{Message: "audio is empty"}
	}
	if len(input.Audio) > maxAudioBytes {
		return TranscribeResult{}, &TranscriptionError{Message: "audio exceeds 25 MiB safe transcription limit"}
	}
	cfg, err := s.store.LoadSettings()
	if err != nil {
		return TranscribeResult{}, err
	}
	if !cfg.Enabled {
		return TranscribeResult{}, &TranscriptionError{Message: "voice accessibility is disabled"}
	}
	if input.Language == "" {
		input.Language = cfg.Language
	}
	input.Language = normalizeLanguage(input.Language)
	if input.DurationMS > int64(cfg.MaxCaptureSeconds+3)*1000 {
		return TranscribeResult{}, &TranscriptionError{Message: "audio exceeds configured maximum capture duration"}
	}

	providers := append([]ProviderConfig(nil), cfg.Providers...)
	attempts := make([]ProviderAttempt, 0, len(providers))
	for _, p := range providers {
		if !p.Enabled {
			continue
		}
		key, keyErr := s.store.ResolveAPIKey(p)
		if keyErr != nil {
			attempt := ProviderAttempt{ProviderID: p.ID, Provider: string(p.Kind), Status: StatusMisconfigured, Message: "credential could not be loaded"}
			attempts = append(attempts, attempt)
			s.store.RecordUsage(p.ID, attempt.Status, 0, false, false)
			if !cfg.AutomaticFallback {
				break
			}
			continue
		}
		if strings.TrimSpace(key) == "" {
			attempt := ProviderAttempt{ProviderID: p.ID, Provider: string(p.Kind), Status: StatusMisconfigured, Message: "API key is not configured"}
			attempts = append(attempts, attempt)
			s.store.RecordUsage(p.ID, attempt.Status, 0, false, false)
			if !cfg.AutomaticFallback {
				break
			}
			continue
		}

		var text string
		var attempt ProviderAttempt
		var sent bool
		switch p.Kind {
		case ProviderGroq:
			text, attempt, sent = s.transcribeGroq(ctx, p, key, input)
		case ProviderCloudflare:
			account := s.store.ResolveAccountID(p)
			if account == "" {
				attempt = ProviderAttempt{ProviderID: p.ID, Provider: string(p.Kind), Status: StatusMisconfigured, Message: "Cloudflare account ID is not configured"}
				sent = false
			} else {
				text, attempt, sent = s.transcribeCloudflare(ctx, p, key, account, input)
			}
		default:
			attempt = ProviderAttempt{ProviderID: p.ID, Provider: string(p.Kind), Status: StatusMisconfigured, Message: "unsupported provider"}
		}
		attempts = append(attempts, attempt)
		success := attempt.Status == StatusSuccess
		s.store.RecordUsage(p.ID, attempt.Status, input.DurationMS, success, sent)
		if success {
			return TranscribeResult{
				Text:       strings.TrimSpace(text),
				ProviderID: p.ID,
				Provider:   string(p.Kind),
				Model:      p.Model,
				Attempts:   attempts,
				DurationMS: input.DurationMS,
			}, nil
		}
		if attempt.Status == StatusInvalidRequest || !cfg.AutomaticFallback {
			break
		}
	}
	if len(attempts) == 0 {
		return TranscribeResult{}, &TranscriptionError{Message: "no enabled speech provider is configured"}
	}
	return TranscribeResult{}, &TranscriptionError{Message: "all configured speech providers failed", Attempts: attempts}
}

func (s *Service) transcribeGroq(ctx context.Context, p ProviderConfig, apiKey string, input TranscribeRequest) (string, ProviderAttempt, bool) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, safeFilename(input.Filename)))
	if input.MIMEType != "" {
		header.Set("Content-Type", input.MIMEType)
	} else {
		header.Set("Content-Type", "application/octet-stream")
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		return "", attempt(p, StatusUnknown, 0, "failed to prepare audio"), false
	}
	if _, err := part.Write(input.Audio); err != nil {
		return "", attempt(p, StatusUnknown, 0, "failed to prepare audio"), false
	}
	_ = writer.WriteField("model", p.Model)
	if input.Language != "" {
		_ = writer.WriteField("language", input.Language)
	}
	_ = writer.WriteField("prompt", developerPrompt)
	if err := writer.Close(); err != nil {
		return "", attempt(p, StatusUnknown, 0, "failed to prepare request"), false
	}

	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = "https://api.groq.com"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/openai/v1/audio/transcriptions", &body)
	if err != nil {
		return "", attempt(p, StatusUnknown, 0, "failed to create request"), false
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.Do(req)
	if err != nil {
		return "", attempt(p, StatusTemporarilyUnavailable, 0, "network request failed"), true
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", classifyHTTP(p, resp.StatusCode), true
	}
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil || strings.TrimSpace(payload.Text) == "" {
		return "", attempt(p, StatusUnknown, resp.StatusCode, "provider returned no transcript"), true
	}
	return payload.Text, attempt(p, StatusSuccess, resp.StatusCode, ""), true
}

func (s *Service) transcribeCloudflare(ctx context.Context, p ProviderConfig, apiKey, accountID string, input TranscribeRequest) (string, ProviderAttempt, bool) {
	payload := struct {
		Audio         []byte `json:"audio"`
		Language      string `json:"language,omitempty"`
		InitialPrompt string `json:"initial_prompt,omitempty"`
		VADFilter     bool   `json:"vad_filter"`
	}{
		Audio: input.Audio, Language: input.Language, InitialPrompt: developerPrompt, VADFilter: true,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", attempt(p, StatusUnknown, 0, "failed to encode audio"), false
	}
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = "https://api.cloudflare.com/client/v4"
	}
	modelPath := escapePathPreserveSlash(p.Model)
	endpoint := fmt.Sprintf("%s/accounts/%s/ai/run/%s", base, escapePathSegment(accountID), modelPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return "", attempt(p, StatusUnknown, 0, "failed to create request"), false
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", attempt(p, StatusTemporarilyUnavailable, 0, "network request failed"), true
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", classifyHTTP(p, resp.StatusCode), true
	}
	var result struct {
		Success bool `json:"success"`
		Result  struct {
			Text string `json:"text"`
		} `json:"result"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&result); err != nil || !result.Success || strings.TrimSpace(result.Result.Text) == "" {
		return "", attempt(p, StatusUnknown, resp.StatusCode, "provider returned no transcript"), true
	}
	return result.Result.Text, attempt(p, StatusSuccess, resp.StatusCode, ""), true
}

func classifyHTTP(p ProviderConfig, status int) ProviderAttempt {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return attempt(p, StatusAuthError, status, "provider rejected credentials")
	case status == http.StatusTooManyRequests:
		return attempt(p, StatusRateLimited, status, "provider rate limit or quota reached")
	case status == http.StatusPaymentRequired:
		return attempt(p, StatusQuotaExhausted, status, "provider quota or billing limit reached")
	case status == http.StatusBadRequest || status == http.StatusRequestEntityTooLarge || status == http.StatusUnsupportedMediaType:
		return attempt(p, StatusInvalidRequest, status, "provider rejected the audio request")
	case status >= 500:
		return attempt(p, StatusTemporarilyUnavailable, status, "provider is temporarily unavailable")
	default:
		return attempt(p, StatusUnknown, status, "provider request failed")
	}
}

func attempt(p ProviderConfig, status string, httpStatus int, message string) ProviderAttempt {
	return ProviderAttempt{
		ProviderID: p.ID,
		Provider:   string(p.Kind),
		Status:     status,
		HTTPStatus: httpStatus,
		Message:    security.Redact(message),
	}
}

func safeFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "voice.webm"
	}
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, "/", "_")
	if len(value) > 100 {
		value = value[len(value)-100:]
	}
	return value
}

func escapePathSegment(value string) string {
	replacer := strings.NewReplacer("%", "%25", "/", "%2F", "?", "%3F", "#", "%23", " ", "%20")
	return replacer.Replace(strings.TrimSpace(value))
}

func escapePathPreserveSlash(value string) string {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(value), "/"), "/")
	for i := range parts {
		parts[i] = escapePathSegment(parts[i])
	}
	return strings.Join(parts, "/")
}
