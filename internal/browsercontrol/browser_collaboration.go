package browsercontrol

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const agentGrantDuration = time.Hour

// AgentGrant describes an explicitly authorized browser capability.
// The secret token is returned once and never appears in lists.
type AgentGrant struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Actor     string    `json:"actor"`
	Label     string    `json:"label"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Handoff struct {
	Status      string    `json:"status"`
	Reason      string    `json:"reason,omitempty"`
	RequestedBy string    `json:"requested_by,omitempty"`
	RequestedAt time.Time `json:"requested_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type grantEntry struct {
	AgentGrant
	digest [32]byte
}

// IssueAgentGrant creates an hour-limited browser-only token. It does not
// inherit Nexus administration, filesystem, other browser or project rights.
func (m *Manager) IssueAgentGrant(sessionID, label string) (AgentGrant, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closing {
		return AgentGrant{}, "", errors.New("browser manager shutting down")
	}
	s, ok := m.sessions[sessionID]
	if !ok {
		return AgentGrant{}, "", errors.New("browser session not found")
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return AgentGrant{}, "", errors.New("browser session closed")
	}
	id, err := randomID()
	if err != nil {
		return AgentGrant{}, "", err
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return AgentGrant{}, "", err
	}
	token := "nb_" + hex.EncodeToString(secret)
	grant := AgentGrant{
		ID: id, SessionID: sessionID, Actor: "agent:" + id,
		Label: strings.TrimSpace(label), ExpiresAt: time.Now().Add(agentGrantDuration),
	}
	if grant.Label == "" {
		grant.Label = "Agent browser connection"
	}
	if m.grants == nil {
		m.grants = make(map[[32]byte]grantEntry)
	}
	digest := sha256.Sum256([]byte(token))
	m.grants[digest] = grantEntry{AgentGrant: grant, digest: digest}
	return grant, token, nil
}

func (m *Manager) ResolveAgentGrant(token string) (AgentGrant, *Session, bool) {
	if len(token) != 67 || !strings.HasPrefix(token, "nb_") {
		return AgentGrant{}, nil, false
	}
	digest := sha256.Sum256([]byte(token))
	m.mu.RLock()
	entry, exists := m.grants[digest]
	s := m.sessions[entry.SessionID]
	m.mu.RUnlock()
	if !exists || s == nil || time.Now().After(entry.ExpiresAt) {
		return AgentGrant{}, nil, false
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return AgentGrant{}, nil, false
	}
	return entry.AgentGrant, s, true
}

func (m *Manager) ListAgentGrants(sessionID string) []AgentGrant {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]AgentGrant, 0)
	for _, entry := range m.grants {
		if entry.SessionID == sessionID && time.Now().Before(entry.ExpiresAt) {
			out = append(out, entry.AgentGrant)
		}
	}
	return out
}

func (m *Manager) RevokeAgentGrant(sessionID, id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for digest, entry := range m.grants {
		if entry.SessionID == sessionID && entry.ID == id {
			delete(m.grants, digest)
			if s := m.sessions[sessionID]; s != nil {
				s.ReleaseControl(entry.Actor)
			}
			return true
		}
	}
	return false
}

// Handoff is held separately from the ephemeral input lease: a sleeping phone
// must not cause an agent to type into a password form merely because the
// 25-second lease expired.
func (s *Session) HandoffState() Handoff {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.handoff
}

func (s *Session) RequestHuman(actor, reason string) (Handoff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Handoff{}, errors.New("browser session closed")
	}
	if !strings.HasPrefix(actor, "agent:") {
		return Handoff{}, errors.New("agent identity required")
	}
	if s.controller != actor || time.Now().After(s.controllerUntil) {
		return Handoff{}, errors.New("agent must control browser before requesting help")
	}
	if s.handoff.Status == "requested" || s.handoff.Status == "human_active" {
		return Handoff{}, errors.New("human intervention already pending")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 500 {
		return Handoff{}, errors.New("reason must be 1 to 500 characters")
	}
	now := time.Now()
	s.handoff = Handoff{Status: "requested", Reason: reason, RequestedBy: actor, RequestedAt: now, UpdatedAt: now}
	s.controller = ""
	s.controllerUntil = time.Time{}
	return s.handoff, nil
}

func (s *Session) BeginHuman(actor string) (Handoff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Handoff{}, errors.New("browser session closed")
	}
	if actor == "" || strings.HasPrefix(actor, "agent:") {
		return Handoff{}, errors.New("human Nexus session required")
	}
	if s.handoff.Status != "requested" && s.handoff.Status != "human_active" {
		// A human may proactively pause an agent. The session lock ensures
		// that any in-flight CDP action finishes before this transfer.
		if !strings.HasPrefix(s.controller, "agent:") {
			return Handoff{}, errors.New("no agent or intervention to take over")
		}
		now := time.Now()
		s.handoff = Handoff{
			Status: "requested", Reason: "User requested browser control",
			RequestedBy: s.controller, RequestedAt: now, UpdatedAt: now,
		}
	}
	if s.handoff.Status == "human_active" && s.controller != actor &&
		s.controller != "" && time.Now().Before(s.controllerUntil) {
		return Handoff{}, errors.New("another user is handling this intervention")
	}
	s.controller = actor
	s.controllerUntil = time.Now().Add(25 * time.Second)
	s.handoff.Status = "human_active"
	s.handoff.UpdatedAt = time.Now()
	return s.handoff, nil
}

func (s *Session) FinishHuman(actor string) (Handoff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handoff.Status != "human_active" || s.controller != actor || actor == "" {
		return Handoff{}, errors.New("only the human controller can complete intervention")
	}
	s.handoff.Status = "completed"
	s.handoff.UpdatedAt = time.Now()
	s.controller = ""
	s.controllerUntil = time.Time{}
	return s.handoff, nil
}

func (s *Session) CancelHuman(actor string) (Handoff, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if actor == "" || strings.HasPrefix(actor, "agent:") {
		return Handoff{}, errors.New("human Nexus session required")
	}
	if s.handoff.Status != "requested" && s.handoff.Status != "human_active" {
		return Handoff{}, errors.New("no human intervention pending")
	}
	if s.handoff.Status == "human_active" && s.controller != actor {
		return Handoff{}, errors.New("only current human controller can cancel")
	}
	s.handoff.Status = "cancelled"
	s.handoff.UpdatedAt = time.Now()
	s.controller = ""
	s.controllerUntil = time.Time{}
	return s.handoff, nil
}

// AcquireAgentControl honors an intervention regardless of lease expiry.
func (s *Session) AcquireAgentControl(actor string) (bool, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !strings.HasPrefix(actor, "agent:") || s.closed {
		return false, "invalid agent"
	}
	if s.handoff.Status == "requested" || s.handoff.Status == "human_active" {
		return false, "waiting for human intervention"
	}
	now := time.Now()
	if s.controller != "" && s.controller != actor && now.Before(s.controllerUntil) {
		return false, "another actor controls this browser"
	}
	s.controller = actor
	s.controllerUntil = now.Add(25 * time.Second)
	return true, ""
}

// DoAs checks ownership and handoff under the same mutex as CDP execution.
// Checking CanControl separately would allow a stale command to run after a
// different actor acquired the lease.
func (s *Session) DoAs(ctx context.Context, actor string, action Action) (Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Info{}, errors.New("browser session closed")
	}
	if actor == "" || s.controller != actor || time.Now().After(s.controllerUntil) {
		return Info{}, errors.New("browser input lease unavailable")
	}
	if strings.HasPrefix(actor, "agent:") && (s.handoff.Status == "requested" || s.handoff.Status == "human_active") {
		return Info{}, errors.New("agent paused for human intervention")
	}
	return s.doLocked(ctx, action)
}

func (s *Session) SnapshotAs(ctx context.Context, actor string) ([]byte, error) {
	s.mu.Lock()
	humanControl := strings.HasPrefix(s.controller, "human:") && time.Now().Before(s.controllerUntil)
	if strings.HasPrefix(actor, "agent:") && (humanControl || s.handoff.Status == "requested" || s.handoff.Status == "human_active") {
		s.mu.Unlock()
		return nil, errors.New("human intervention active; page contents private")
	}
	defer s.mu.Unlock()
	raw, err := s.snapshotLocked(ctx)
	return []byte(raw), err
}

func (s *Session) ScreenshotAs(ctx context.Context, actor string) ([]byte, error) {
	s.mu.Lock()
	humanControl := strings.HasPrefix(s.controller, "human:") && time.Now().Before(s.controllerUntil)
	if strings.HasPrefix(actor, "agent:") && (humanControl || s.handoff.Status == "requested" || s.handoff.Status == "human_active") {
		s.mu.Unlock()
		return nil, errors.New("human intervention active; screenshots private")
	}
	defer s.mu.Unlock()
	return s.screenshotLocked(ctx)
}

func (s *Session) ControlInfo() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Human control identities contain Nexus authentication session IDs.
	// Never expose those credentials through MCP or the browser status API.
	controllerKind := "none"
	if strings.HasPrefix(s.controller, "human:") {
		controllerKind = "human"
	} else if strings.HasPrefix(s.controller, "agent:") {
		controllerKind = "agent"
	} else if s.controller != "" {
		controllerKind = "other"
	}
	return map[string]any{
		"controller":         controllerKind,
		"occupied":           s.controller != "" && time.Now().Before(s.controllerUntil),
		"handoff":            s.handoff,
		"project_id":         s.info.ProjectID,
		"browser_session_id": s.info.ID,
	}
}

func (s *Session) String() string { return fmt.Sprintf("BrowserSession(%s)", s.Info().ID) }
