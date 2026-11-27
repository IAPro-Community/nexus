package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/desktop"
)

type fakeWindowHost struct {
	mu         sync.Mutex
	open       bool
	openErr    error
	openCalls  int
	closeCalls int
}

func (f *fakeWindowHost) IsOpen() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.open
}

func (f *fakeWindowHost) EnsureOpen() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.openCalls++
	if f.openErr != nil {
		return f.openErr
	}
	f.open = true
	return nil
}

func (f *fakeWindowHost) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closeCalls++
	f.open = false
	return nil
}

func floatPtr(v float64) *float64 { return &v }

func TestNexV2AlertWatcherWakesOnQualifyingAlert(t *testing.T) {
	host := &fakeWindowHost{}
	state := desktop.DefaultNexV2PresenceState()
	state.NotificationLevel = desktop.NexV2NotifLevelImportant
	state.HudMemoryMode = desktop.NexV2HudMemoryOnDemand

	var lastHub, lastDnd string
	now := time.Unix(1000, 0)
	watcher := newNexV2AlertWatcher(nexV2WatcherConfig{
		StateReader: func() desktop.NexV2PresenceState { return state },
		WindowHost:  host,
		FetchHUD: func(_ context.Context, hubHeader, dndHeader string) (*nexV2HudPayload, error) {
			lastHub = hubHeader
			lastDnd = dndHeader
			return &nexV2HudPayload{
				RuntimeAttention: []nexV2RuntimeAttention{{
					RuntimeID:   "rt-1",
					ProjectName: "nexus",
					Provider:    "codex",
					Reason:      "APPROVAL_REQUIRED",
					Fingerprint: "fp-1",
				}},
			}, nil
		},
		Now: func() time.Time { return now },
	})

	watcher.step(context.Background())

	if lastHub != "active" || lastDnd != "0" {
		t.Fatalf("unexpected hub headers: hub=%q dnd=%q", lastHub, lastDnd)
	}
	if !host.IsOpen() || host.openCalls != 1 {
		t.Fatalf("expected hibernated window to wake on runtime attention (open=%v calls=%d)", host.IsOpen(), host.openCalls)
	}
}

func TestNexV2AlertWatcherQuietAndCriticalKeepHubActiveWithoutOpeningWindow(t *testing.T) {
	host := &fakeWindowHost{}
	state := desktop.DefaultNexV2PresenceState()
	state.NotificationLevel = desktop.NexV2NotifLevelQuiet
	state.HudMemoryMode = desktop.NexV2HudMemoryOnDemand

	var lastHub, lastDnd string
	var fallbacks []notify.Payload
	now := time.Unix(1000, 0)
	watcher := newNexV2AlertWatcher(nexV2WatcherConfig{
		StateReader: func() desktop.NexV2PresenceState { return state },
		WindowHost:  host,
		FetchHUD: func(_ context.Context, hubHeader, dndHeader string) (*nexV2HudPayload, error) {
			lastHub = hubHeader
			lastDnd = dndHeader
			return &nexV2HudPayload{
				Alerts: []nexV2HudAlert{{
					Fingerprint:      "quota-35",
					Category:         "quota_low",
					Severity:         "WARNING",
					Provider:         "codex",
					RemainingPercent: floatPtr(35),
				}},
			}, nil
		},
		NotifyFallback: func(p notify.Payload) error {
			fallbacks = append(fallbacks, p)
			return nil
		},
		Now: func() time.Time { return now },
	})

	watcher.step(context.Background())
	if lastHub != "active" || lastDnd != "1" {
		t.Fatalf("quiet mode must keep NEX hub active and mark dnd=1 (hub=%q dnd=%q)", lastHub, lastDnd)
	}
	if host.IsOpen() || host.openCalls != 0 {
		t.Fatalf("quiet mode should not wake WebView (open=%v calls=%d)", host.IsOpen(), host.openCalls)
	}
	if len(fallbacks) != 0 {
		t.Fatalf("quiet mode must suppress OS fallback, got %d fallbacks", len(fallbacks))
	}

	state.NotificationLevel = desktop.NexV2NotifLevelCritical
	watcher.step(context.Background())
	if host.IsOpen() || host.openCalls != 0 {
		t.Fatalf("critical-only mode must not wake WebView on 35%% WARNING quota alert")
	}
}

func TestNexV2AlertWatcherCooldownSuppressionAndCriticalBypass(t *testing.T) {
	host := &fakeWindowHost{}
	state := desktop.DefaultNexV2PresenceState()
	state.NotificationLevel = desktop.NexV2NotifLevelImportant
	state.NotificationCooldownSec = 120

	now := time.Unix(1000, 0)
	alerts := &nexV2HudPayload{
		RuntimeAttention: []nexV2RuntimeAttention{{
			RuntimeID: "rt-1", Provider: "claude", Reason: "APPROVAL", Fingerprint: "fp-1",
		}},
	}
	watcher := newNexV2AlertWatcher(nexV2WatcherConfig{
		StateReader: func() desktop.NexV2PresenceState { return state },
		WindowHost:  host,
		FetchHUD:    func(context.Context, string, string) (*nexV2HudPayload, error) { return alerts, nil },
		Now:         func() time.Time { return now },
	})

	watcher.step(context.Background())
	if host.openCalls != 1 {
		t.Fatalf("expected initial open, got %d", host.openCalls)
	}

	// Simulate user closing/hibernating within cooldown, then another non-critical alert arrives.
	_ = host.Close()
	now = now.Add(30 * time.Second)
	watcher.step(context.Background())
	if host.IsOpen() || host.openCalls != 1 {
		t.Fatalf("non-critical alert within cooldown should not reopen window (open=%v calls=%d)", host.IsOpen(), host.openCalls)
	}

	// Critical alert bypasses cooldown immediately.
	alerts = &nexV2HudPayload{
		Alerts: []nexV2HudAlert{{
			Fingerprint: "crit-1", Category: "critical_failure", Severity: "CRITICAL", Provider: "codex",
		}},
	}
	watcher.step(context.Background())
	if !host.IsOpen() || host.openCalls != 2 {
		t.Fatalf("critical alert must bypass cooldown and wake HUD (open=%v calls=%d)", host.IsOpen(), host.openCalls)
	}
}

func TestNexV2AlertWatcherFailSafeFallbackWhenOpenFails(t *testing.T) {
	host := &fakeWindowHost{openErr: errors.New("wayland display unavailable")}
	state := desktop.DefaultNexV2PresenceState()
	state.NotificationLevel = desktop.NexV2NotifLevelImportant

	var hubCanNotify = true
	var hubReason string
	var fallbacks []notify.Payload

	watcher := newNexV2AlertWatcher(nexV2WatcherConfig{
		StateReader: func() desktop.NexV2PresenceState { return state },
		WindowHost:  host,
		FetchHUD: func(context.Context, string, string) (*nexV2HudPayload, error) {
			return &nexV2HudPayload{
				RuntimeAttention: []nexV2RuntimeAttention{{
					RuntimeID:   "rt-9",
					ProjectName: "nexus",
					Provider:    "codex",
					Reason:      "AUTH_REQUIRED",
					Fingerprint: "fp-9",
				}},
			}, nil
		},
		ReportHub: func(_ context.Context, canNotify, _ bool, reason string) error {
			hubCanNotify = canNotify
			hubReason = reason
			return nil
		},
		NotifyFallback: func(p notify.Payload) error {
			fallbacks = append(fallbacks, p)
			return nil
		},
	})

	watcher.step(context.Background())

	if hubCanNotify || hubReason != "webview_open_failed" {
		t.Fatalf("expected hub lease released on EnsureOpen failure (canNotify=%v reason=%q)", hubCanNotify, hubReason)
	}
	if len(fallbacks) != 1 || !fallbacks[0].ForceFallback {
		t.Fatalf("expected 1 forced OS fallback notification, got %+v", fallbacks)
	}
}

func TestNexV2AlertWatcherIdleHibernateRespectsInteraction(t *testing.T) {
	host := &fakeWindowHost{open: true}
	state := desktop.DefaultNexV2PresenceState()
	state.HudMemoryMode = desktop.NexV2HudMemoryOnDemand
	interactionOpen := true
	now := time.Unix(2000, 0)

	watcher := newNexV2AlertWatcher(nexV2WatcherConfig{
		StateReader:          func() desktop.NexV2PresenceState { return state },
		InteractionOpen:      func() bool { return interactionOpen },
		WindowHost:           host,
		IdleHibernateTimeout: 60 * time.Second,
		FetchHUD:             func(context.Context, string, string) (*nexV2HudPayload, error) { return &nexV2HudPayload{}, nil },
		Now:                  func() time.Time { return now },
	})

	watcher.step(context.Background())
	now = now.Add(120 * time.Second)
	watcher.step(context.Background())
	if !host.IsOpen() || host.closeCalls != 0 {
		t.Fatalf("must not hibernate while panel/settings open")
	}

	// User closes panel; idle timer starts counting from now.
	interactionOpen = false
	watcher.step(context.Background())
	if !host.IsOpen() || host.closeCalls != 0 {
		t.Fatalf("must not hibernate before timeout elapses")
	}

	now = now.Add(61 * time.Second)
	watcher.step(context.Background())
	if host.IsOpen() || host.closeCalls != 1 {
		t.Fatalf("expected clean hibernation after idle timeout (open=%v closes=%d)", host.IsOpen(), host.closeCalls)
	}
}
