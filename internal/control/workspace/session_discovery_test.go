package workspace

import (
	"context"
	"testing"
)

func TestSessionDiscoveryManagerRegisterAndDiscover(t *testing.T) {
	manager := NewSessionDiscoveryManager()

	mock := &mockDiscovery{
		providerID: "claude",
		sessions: []ProviderSession{
			{ID: "sess-1", ProviderID: "claude", Title: "Auth refactor"},
		},
	}

	manager.Register(mock)

	sessions, err := manager.DiscoverAll(context.Background(), "/path/to/project")
	if err != nil {
		t.Fatalf("DiscoverAll error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != "sess-1" {
		t.Fatalf("expected sess-1, got %s", sessions[0].ID)
	}
}

func TestSessionDiscoveryManagerDiscoverByProvider(t *testing.T) {
	manager := NewSessionDiscoveryManager()

	manager.Register(&mockDiscovery{
		providerID: "claude",
		sessions: []ProviderSession{
			{ID: "sess-1", ProviderID: "claude"},
		},
	})
	manager.Register(&mockDiscovery{
		providerID: "codex",
		sessions: []ProviderSession{
			{ID: "sess-2", ProviderID: "codex"},
		},
	})

	sessions, err := manager.DiscoverByProvider(context.Background(), "claude", "/path")
	if err != nil {
		t.Fatalf("DiscoverByProvider error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ProviderID != "claude" {
		t.Fatalf("expected claude, got %s", sessions[0].ProviderID)
	}
}

type mockDiscovery struct {
	providerID string
	sessions   []ProviderSession
}

func (m *mockDiscovery) ProviderID() string {
	return m.providerID
}

func (m *mockDiscovery) DiscoverSessions(_ context.Context, _ string) ([]ProviderSession, error) {
	return m.sessions, nil
}

func (m *mockDiscovery) CanResume(_ context.Context, _ string) (bool, string, error) {
	return true, "SUPPORTED", nil
}

func (m *mockDiscovery) CanFork(_ context.Context, _ string) (bool, string, error) {
	return false, "UNSUPPORTED", nil
}
