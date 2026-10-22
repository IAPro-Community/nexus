package main

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestHUDAttentionMonitorTracksNewAndResolvedRequests(t *testing.T) {
	wait := registry.RuntimeSession{
		RuntimeID: "rt-agy", ProviderID: "agy", ProjectID: "project-b",
		Transport: "mock", State: registry.StateWaiting,
		AttentionKind: "needs_user", AttentionReason: "QUESTION",
		PromptKind: "choice", AttentionContext: "private prompt",
		AttentionFingerprint: "fingerprint-v1", UpdatedAt: time.Now(),
	}
	sessions := []registry.RuntimeSession{wait}
	var emitted []string
	monitor := &hudAttentionMonitor{
		source: func() []registry.RuntimeSession { return sessions },
		deliver: func(runtimeID, fp string) error {
			emitted = append(emitted, runtimeID+":"+fp)
			return nil
		},
		seen: map[string]struct{}{},
	}
	monitor.poll()
	monitor.poll()
	if len(emitted) != 1 || emitted[0] != "rt-agy:fingerprint-v1" {
		t.Fatalf("same AGY wait should produce one OS notification, got %v", emitted)
	}
	// Answered wait clears the active set, even with the HUD fully hidden.
	sessions[0].State = registry.StateRunning
	sessions[0].AttentionKind = "working"
	monitor.poll()
	if len(monitor.seen) != 0 {
		t.Fatalf("resolved attention remained active: %v", monitor.seen)
	}
	// The same question in another turn is a new actionable transition.
	sessions[0] = wait
	monitor.poll()
	if len(emitted) != 2 {
		t.Fatalf("new wait transition should be noticed: %v", emitted)
	}
}

func TestHUDAttentionMonitorIgnoresNoiseAndStaleSessions(t *testing.T) {
	var emitted int
	monitor := &hudAttentionMonitor{
		source: func() []registry.RuntimeSession {
			return []registry.RuntimeSession{
				{RuntimeID: "tty-shortcut", Transport: "mock", State: registry.StateRunning,
					AttentionKind: "working", AttentionContext: "? for shortcuts"},
				{RuntimeID: "dead", State: registry.StateWaiting, ProviderID: "codex",
					AttentionKind: "needs_user", PromptKind: "yn", AttentionContext: "prompt",
					HostPID: 0},
			}
		},
		deliver: func(_, _ string) error { emitted++; return nil },
		seen:    map[string]struct{}{},
	}
	monitor.poll()
	if emitted != 0 {
		t.Fatalf("non-actionable or dead sessions must never generate OS alerts: %d", emitted)
	}
}
