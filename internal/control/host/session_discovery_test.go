package host

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func newDiscoveryTestHost(t *testing.T, discoverer SessionIDDiscovererFunc) (*SessionHost, *registry.Registry) {
	t.Helper()
	reg := registry.NewRegistry(filepath.Join(t.TempDir(), "registry.json"))
	sh, err := NewSessionHost(Config{
		Session: registry.RuntimeSession{
			RuntimeID:    "rt-discovery",
			ProviderID:   "codex",
			ProfileID:    "work",
			Workspace:    t.TempDir(),
			State:        registry.StateStarting,
			ControlLevel: registry.ControlLevelTerminal,
		},
		Registry:            reg,
		Binary:              "/bin/true",
		SessionIDDiscoverer: discoverer,
	})
	if err != nil {
		t.Fatalf("NewSessionHost: %v", err)
	}
	return sh, reg
}

func withFastDiscoveryPolling(t *testing.T) {
	t.Helper()
	previousInterval, previousTimeout := sessionDiscoveryPollInterval, sessionDiscoveryProbeTimeout
	sessionDiscoveryPollInterval = 5 * time.Millisecond
	sessionDiscoveryProbeTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		sessionDiscoveryPollInterval = previousInterval
		sessionDiscoveryProbeTimeout = previousTimeout
	})
}

func TestDiscoverProviderSessionID_AdoptsDiscoveredID(t *testing.T) {
	withFastDiscoveryPolling(t)
	calls := 0
	sh, reg := newDiscoveryTestHost(t, func(ctx context.Context, session registry.RuntimeSession, startedAt time.Time) (string, error) {
		calls++
		if calls < 2 {
			return "", nil
		}
		return "provider-session-1", nil
	})
	// The launcher registers the session before Start(); mirror that here so
	// the registry write has a row to update.
	_ = reg.Register(sh.session)

	done := make(chan struct{})
	go func() {
		sh.discoverProviderSessionID(time.Now())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		close(sh.stopChan)
		t.Fatalf("discovery did not converge")
	}
	close(sh.stopChan)

	if sh.session.ProviderSessionID != "provider-session-1" {
		t.Fatalf("session ProviderSessionID = %q", sh.session.ProviderSessionID)
	}
	persisted, ok := reg.Get("rt-discovery")
	if !ok || persisted.ProviderSessionID != "provider-session-1" {
		t.Fatalf("registry ProviderSessionID = %q (ok=%v)", persisted.ProviderSessionID, ok)
	}
}

func TestDiscoverProviderSessionID_NeverOverwritesStructuredID(t *testing.T) {
	withFastDiscoveryPolling(t)
	sh, _ := newDiscoveryTestHost(t, func(ctx context.Context, session registry.RuntimeSession, startedAt time.Time) (string, error) {
		return "discovered-late", nil
	})
	sh.session.ProviderSessionID = "structured-id"

	done := make(chan struct{})
	go func() {
		sh.discoverProviderSessionID(time.Now())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("discovery did not exit when id was already known")
	}
	close(sh.stopChan)

	if sh.session.ProviderSessionID != "structured-id" {
		t.Fatalf("structured id was overwritten: %q", sh.session.ProviderSessionID)
	}
}

func TestDiscoverProviderSessionID_StopsOnHostLifecycle(t *testing.T) {
	withFastDiscoveryPolling(t)
	blocked := make(chan struct{})
	sh, _ := newDiscoveryTestHost(t, func(ctx context.Context, session registry.RuntimeSession, startedAt time.Time) (string, error) {
		<-blocked
		return "never", nil
	})

	done := make(chan struct{})
	go func() {
		sh.discoverProviderSessionID(time.Now())
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)
	close(sh.stopChan)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("discovery ignored host stop")
	}
	close(blocked)
}
