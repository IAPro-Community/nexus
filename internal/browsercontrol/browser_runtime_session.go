package browsercontrol

import (
	"context"
	"errors"
	"strings"
)

// Each runtime has its own browser; browsers aren't created when a CLI is
// launched but when it first calls browser_open. The separate runtimeMu
// serializes provisioning without holding Manager.mu across Chrome startup.
func (m *Manager) FindRuntimeBrowser(runtimeID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.sessions {
		s.mu.Lock()
		belongs := !s.closed && s.info.OwnerRuntimeID == runtimeID
		s.mu.Unlock()
		if belongs {
			return s, true
		}
	}
	return nil, false
}
func (m *Manager) OpenRuntimeBrowser(ctx context.Context, cap RuntimeCapability, url string) (Info, error) {
	if cap.RuntimeID == "" || cap.ProjectID == "" {
		return Info{}, errors.New("browser runtime capability invalid")
	}
	if m.previewRegistry == nil {
		return Info{}, errors.New("runtime preview supervisor unavailable")
	}
	m.runtimeMu.Lock()
	defer m.runtimeMu.Unlock()
	registration, err := m.previewRegistry.ResolveOrRegister(cap.RuntimeID, cap.ProjectID, url)
	if err != nil {
		return Info{}, err
	}
	if s, ok := m.FindRuntimeBrowser(cap.RuntimeID); ok {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.previewOrigin != runtimePreviewOriginFromURL(registration.URL) {
			return Info{}, errors.New("runtime browser is bound to a different supervised preview")
		}
		s.previewRegistry = m.previewRegistry
		if strings.TrimSpace(url) != "" {
			if err := s.navigate(ctx, url); err != nil {
				return Info{}, err
			}
		}
		return s.infoLocked(), nil
	}
	info, err := m.create(ctx, cap.ProjectID, registration.URL, cap.RuntimeID, runtimePreviewOriginFromURL(registration.URL), m.previewRegistry)
	if err != nil {
		return Info{}, err
	}
	s, ok := m.Get(info.ID)
	if !ok {
		return Info{}, errors.New("browser disappeared during provisioning")
	}
	s.mu.Lock()
	info = s.infoLocked()
	s.mu.Unlock()
	if _, err := m.previewRegistry.Resolve(cap.RuntimeID, cap.ProjectID, registration.URL); err != nil {
		m.Close(info.ID)
		return Info{}, err
	}
	return info, nil
}

func runtimePreviewOriginFromURL(raw string) string {
	u, _, err := parseRuntimePreviewURL(raw)
	if err != nil {
		return ""
	}
	return runtimePreviewOrigin(u)
}
