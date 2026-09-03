package workspace

import (
	"context"
	"sync"
	"time"
)

// ProviderSession represents a discovered session from an external provider.
type ProviderSession struct {
	ID                string    `json:"id"`
	ProviderID        string    `json:"providerId"`
	ProjectPath       string    `json:"projectPath,omitempty"`
	Title             string    `json:"title,omitempty"`
	LastActivityAt    time.Time `json:"lastActivityAt,omitempty"`
	ResumeCapability  string    `json:"resumeCapability"` // SUPPORTED, PARTIAL, UNSUPPORTED, UNKNOWN
	ForkCapability    string    `json:"forkCapability"`   // SUPPORTED, PARTIAL, UNSUPPORTED, UNKNOWN
	ContinuityNotes   string    `json:"continuityNotes,omitempty"`
}

// ProviderSessionDiscovery defines the interface for discovering provider sessions.
type ProviderSessionDiscovery interface {
	// ProviderID returns the provider identifier.
	ProviderID() string

	// DiscoverSessions finds active sessions for a project path.
	DiscoverSessions(ctx context.Context, projectPath string) ([]ProviderSession, error)

	// CanResume checks if a session can be resumed.
	CanResume(ctx context.Context, sessionID string) (bool, string, error)

	// CanFork checks if a session can be forked.
	CanFork(ctx context.Context, sessionID string) (bool, string, error)
}

// SessionDiscoveryManager coordinates session discovery across all providers.
type SessionDiscoveryManager struct {
	mu          sync.RWMutex
	discoveries map[string]ProviderSessionDiscovery // providerID -> discovery
}

// NewSessionDiscoveryManager creates a new session discovery manager.
func NewSessionDiscoveryManager() *SessionDiscoveryManager {
	return &SessionDiscoveryManager{
		discoveries: make(map[string]ProviderSessionDiscovery),
	}
}

// Register registers a provider's session discovery implementation.
func (m *SessionDiscoveryManager) Register(discovery ProviderSessionDiscovery) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.discoveries[discovery.ProviderID()] = discovery
}

// DiscoverAll discovers sessions from all registered providers.
func (m *SessionDiscoveryManager) DiscoverAll(ctx context.Context, projectPath string) ([]ProviderSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var allSessions []ProviderSession

	for _, discovery := range m.discoveries {
		sessions, err := discovery.DiscoverSessions(ctx, projectPath)
		if err != nil {
			continue // Skip failed providers
		}
		allSessions = append(allSessions, sessions...)
	}

	return allSessions, nil
}

// DiscoverByProvider discovers sessions from a specific provider.
func (m *SessionDiscoveryManager) DiscoverByProvider(ctx context.Context, providerID, projectPath string) ([]ProviderSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	discovery, ok := m.discoveries[providerID]
	if !ok {
		return nil, nil
	}
	return discovery.DiscoverSessions(ctx, projectPath)
}
