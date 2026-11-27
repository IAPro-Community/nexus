package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/control/web"
	"github.com/IAPro-Community/nexus/internal/desktop"
)

const (
	defaultNexV2PollInterval    = 4 * time.Second
	defaultNexV2IdleHibernate   = 5 * time.Minute
	minNexV2IdleHibernateEnvSec = 5
)

type nexV2WindowHostControl interface {
	IsOpen() bool
	EnsureOpen() error
	Close() error
}

type nexV2HudPayload struct {
	Alerts           []nexV2HudAlert         `json:"alerts"`
	RuntimeAttention []nexV2RuntimeAttention `json:"runtime_attention"`
	Missions         []nexV2MissionAttention `json:"missions"`
}

type nexV2HudAlert struct {
	Fingerprint      string   `json:"fingerprint"`
	Category         string   `json:"category"`
	Severity         string   `json:"severity"`
	Provider         string   `json:"provider"`
	Profile          string   `json:"profile"`
	Reason           string   `json:"reason"`
	RemainingPercent *float64 `json:"remaining_percent,omitempty"`
	Acked            bool     `json:"acked"`
}

type nexV2RuntimeAttention struct {
	RuntimeID   string `json:"runtime_id"`
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	Provider    string `json:"provider"`
	Reason      string `json:"reason"`
	PromptKind  string `json:"prompt_kind"`
	Fingerprint string `json:"fingerprint"`
}

type nexV2MissionAttention struct {
	RunID              string `json:"run_id"`
	CurrentStage       string `json:"current_stage"`
	CurrentTaskTitle   string `json:"current_task_title"`
	Provider           string `json:"provider"`
	NeedsUser          bool   `json:"needs_user"`
	InterventionReason string `json:"intervention_reason"`
}

type nexV2WatcherConfig struct {
	StateReader          func() desktop.NexV2PresenceState
	InteractionOpen      func() bool
	WindowHost           nexV2WindowHostControl
	FetchHUD             func(ctx context.Context, hubHeader, dndHeader string) (*nexV2HudPayload, error)
	ReportHub            func(ctx context.Context, canNotify, doNotDisturb bool, reason string) error
	NotifyFallback       func(payload notify.Payload) error
	PollInterval         time.Duration
	IdleHibernateTimeout time.Duration
	Now                  func() time.Time
}

type nexV2AlertWatcher struct {
	cfg            nexV2WatcherConfig
	mu             sync.Mutex
	lastActivityAt time.Time
	cooldownUntil  time.Time
}

func resolveIdleHibernateTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("NEXUS_HUD_IDLE_HIBERNATE_SEC"))
	if raw == "" {
		return defaultNexV2IdleHibernate
	}
	secs, err := strconv.Atoi(raw)
	if err != nil || secs < minNexV2IdleHibernateEnvSec {
		return defaultNexV2IdleHibernate
	}
	return time.Duration(secs) * time.Second
}

func newNexV2AlertWatcher(cfg nexV2WatcherConfig) *nexV2AlertWatcher {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultNexV2PollInterval
	}
	if cfg.IdleHibernateTimeout <= 0 {
		cfg.IdleHibernateTimeout = resolveIdleHibernateTimeout()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.NotifyFallback == nil {
		cfg.NotifyFallback = func(payload notify.Payload) error {
			return notify.Default().Notify(payload)
		}
	}
	return &nexV2AlertWatcher{cfg: cfg}
}

func (w *nexV2AlertWatcher) now() time.Time {
	return w.cfg.Now()
}

func (w *nexV2AlertWatcher) Run(ctx context.Context) {
	w.step(ctx)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if w.cfg.ReportHub != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
				_ = w.cfg.ReportHub(shutdownCtx, false, false, "nex_offline")
				cancel()
			}
			return
		case <-ticker.C:
			w.step(ctx)
		}
	}
}

func (w *nexV2AlertWatcher) step(ctx context.Context) {
	if w.cfg.StateReader == nil || w.cfg.FetchHUD == nil {
		return
	}

	now := w.now()
	presence := w.cfg.StateReader()
	hubHeader := "active"
	effectiveDnd := presence.DoNotDisturb || strings.EqualFold(presence.NotificationLevel, desktop.NexV2NotifLevelQuiet)
	if presence.Paused {
		hubHeader = "paused"
	}
	dndHeader := "0"
	if effectiveDnd {
		dndHeader = "1"
	}

	payload, err := w.cfg.FetchHUD(ctx, hubHeader, dndHeader)
	if err != nil || payload == nil {
		return
	}
	if presence.Paused {
		return
	}

	qualifying := extractQualifyingNexV2Alerts(payload, presence.NotificationLevel)
	w.mu.Lock()
	if len(qualifying) > 0 {
		w.lastActivityAt = now
		hasCritical := false
		for _, alert := range qualifying {
			if strings.EqualFold(alert.Severity, "CRITICAL") {
				hasCritical = true
				break
			}
		}
		cooldownBlocking := presence.NotificationCooldownSec > 0 && !w.cooldownUntil.IsZero() && now.Before(w.cooldownUntil)
		shouldWake := !effectiveDnd && (hasCritical || !cooldownBlocking)
		cooldownSec := presence.NotificationCooldownSec
		w.mu.Unlock()

		if shouldWake && w.cfg.WindowHost != nil && !w.cfg.WindowHost.IsOpen() {
			if openErr := w.cfg.WindowHost.EnsureOpen(); openErr != nil {
				if w.cfg.ReportHub != nil {
					_ = w.cfg.ReportHub(ctx, false, false, "webview_open_failed")
				}
				if w.cfg.NotifyFallback != nil {
					top := qualifying[0]
					_ = w.cfg.NotifyFallback(buildFallbackPayload(top))
				}
				return
			}
			if !hasCritical && cooldownSec > 0 {
				w.mu.Lock()
				w.cooldownUntil = now.Add(time.Duration(cooldownSec) * time.Second)
				w.mu.Unlock()
			}
		}
		return
	}

	memMode := desktop.NormalizeHudMemoryMode(presence.HudMemoryMode)
	isOpen := w.cfg.WindowHost != nil && w.cfg.WindowHost.IsOpen()
	if memMode == desktop.NexV2HudMemoryOnDemand && isOpen {
		if w.cfg.InteractionOpen != nil && w.cfg.InteractionOpen() {
			w.lastActivityAt = now
			w.mu.Unlock()
			return
		}
		if w.lastActivityAt.IsZero() {
			w.lastActivityAt = now
			w.mu.Unlock()
			return
		}
		if now.Sub(w.lastActivityAt) >= w.cfg.IdleHibernateTimeout {
			w.lastActivityAt = time.Time{}
			w.mu.Unlock()
			_ = w.cfg.WindowHost.Close()
			return
		}
	}
	w.mu.Unlock()
}

func nexV2AlertQualifies(alert nexV2HudAlert, level string) bool {
	normalized := desktop.NormalizeNotificationLevel(level)
	if normalized == desktop.NexV2NotifLevelQuiet {
		return false
	}
	if strings.EqualFold(alert.Severity, "CRITICAL") {
		return true
	}
	if normalized == desktop.NexV2NotifLevelCritical {
		return false
	}
	if normalized == desktop.NexV2NotifLevelAll {
		return true
	}
	if alert.Category == "runtime_attention" || alert.Category == "mission_attention" {
		return true
	}
	return alert.RemainingPercent != nil && *alert.RemainingPercent < 15
}

func extractQualifyingNexV2Alerts(payload *nexV2HudPayload, level string) []nexV2HudAlert {
	if payload == nil {
		return nil
	}
	var critical []nexV2HudAlert
	var standard []nexV2HudAlert
	for _, a := range payload.Alerts {
		if a.Acked {
			continue
		}
		if strings.EqualFold(a.Severity, "CRITICAL") {
			critical = append(critical, a)
		} else {
			standard = append(standard, a)
		}
	}

	runtimeAlerts := make([]nexV2HudAlert, 0, len(payload.RuntimeAttention))
	for _, r := range payload.RuntimeAttention {
		severity := "WARNING"
		if strings.EqualFold(strings.TrimSpace(r.Reason), "ERROR") {
			severity = "CRITICAL"
		}
		profile := strings.TrimSpace(r.ProjectName)
		if profile == "" {
			profile = strings.TrimSpace(r.ProjectID)
		}
		reasonParts := make([]string, 0, 2)
		if strings.TrimSpace(r.Reason) != "" {
			reasonParts = append(reasonParts, strings.TrimSpace(r.Reason))
		}
		if strings.TrimSpace(r.PromptKind) != "" {
			reasonParts = append(reasonParts, strings.TrimSpace(r.PromptKind))
		}
		runtimeAlerts = append(runtimeAlerts, nexV2HudAlert{
			Fingerprint: fmt.Sprintf("runtime:%s:%s", r.RuntimeID, r.Fingerprint),
			Category:    "runtime_attention",
			Severity:    severity,
			Provider:    r.Provider,
			Profile:     profile,
			Reason:      strings.Join(reasonParts, " · "),
		})
	}

	var missionAlerts []nexV2HudAlert
	for _, m := range payload.Missions {
		if !m.NeedsUser {
			continue
		}
		provider := strings.TrimSpace(m.Provider)
		if provider == "" {
			provider = "Mission"
		}
		profile := strings.TrimSpace(m.CurrentTaskTitle)
		if profile == "" {
			profile = strings.TrimSpace(m.RunID)
		}
		reason := strings.TrimSpace(m.InterventionReason)
		if reason == "" {
			reason = "Intervenção necessária"
		}
		missionAlerts = append(missionAlerts, nexV2HudAlert{
			Fingerprint: fmt.Sprintf("mission:%s:%s", m.RunID, m.CurrentStage),
			Category:    "mission_attention",
			Severity:    "WARNING",
			Provider:    provider,
			Profile:     profile,
			Reason:      reason,
		})
	}

	queue := make([]nexV2HudAlert, 0, len(critical)+len(runtimeAlerts)+len(missionAlerts)+len(standard))
	queue = append(queue, critical...)
	queue = append(queue, runtimeAlerts...)
	queue = append(queue, missionAlerts...)
	queue = append(queue, standard...)

	out := make([]nexV2HudAlert, 0, len(queue))
	for _, a := range queue {
		if nexV2AlertQualifies(a, level) {
			out = append(out, a)
		}
	}
	return out
}

func mapNexAlertToNotifyCategory(alert nexV2HudAlert) notify.Category {
	switch alert.Category {
	case "runtime_attention", "mission_attention":
		if strings.EqualFold(alert.Severity, "CRITICAL") {
			return notify.CategoryCriticalFailure
		}
		return notify.CategoryActionRequired
	default:
		if strings.EqualFold(alert.Severity, "CRITICAL") {
			return notify.CategoryCriticalFailure
		}
		return notify.CategoryQuotaLow
	}
}

func buildFallbackPayload(alert nexV2HudAlert) notify.Payload {
	subject := strings.TrimSpace(strings.Join([]string{alert.Provider, alert.Profile}, " · "))
	if subject == "" || subject == "·" {
		subject = "Nexus HUD"
	}
	body := strings.TrimSpace(alert.Reason)
	if body == "" {
		body = "Atenção necessária no Nexus"
	}
	return notify.Payload{
		Title:         "Nexus · " + subject,
		Body:          body,
		Tag:           alert.Fingerprint,
		Category:      mapNexAlertToNotifyCategory(alert),
		EpisodeKey:    "nex-fallback:" + alert.Fingerprint,
		ForceFallback: true,
	}
}

func newCoreHUDClient(serverURL string, session *web.Session) (
	func(ctx context.Context, hubHeader, dndHeader string) (*nexV2HudPayload, error),
	func(ctx context.Context, canNotify, doNotDisturb bool, reason string) error,
) {
	base := strings.TrimRight(strings.TrimSpace(serverURL), "/")
	client := &http.Client{Timeout: 3 * time.Second}
	var token, csrf string
	if session != nil {
		token = strings.TrimSpace(session.ID)
		csrf = strings.TrimSpace(session.CSRFToken)
	}

	applyAuth := func(req *http.Request) {
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
	}

	fetchHUD := func(ctx context.Context, hubHeader, dndHeader string) (*nexV2HudPayload, error) {
		if base == "" {
			return nil, fmt.Errorf("server URL unavailable")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/hud", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Nexus-Nex-Hub", hubHeader)
		req.Header.Set("X-Nexus-Nex-Dnd", dndHeader)
		applyAuth(req)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("hud status %d", resp.StatusCode)
		}
		var payload nexV2HudPayload
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
			return nil, err
		}
		return &payload, nil
	}

	reportHub := func(ctx context.Context, canNotify, doNotDisturb bool, reason string) error {
		if base == "" {
			return fmt.Errorf("server URL unavailable")
		}
		body, err := json.Marshal(map[string]any{
			"can_notify":     canNotify,
			"do_not_disturb": doNotDisturb,
			"reason":         reason,
		})
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/nex/hub-status", bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		applyAuth(req)

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("hub-status http %d", resp.StatusCode)
		}
		return nil
	}

	return fetchHUD, reportHub
}
