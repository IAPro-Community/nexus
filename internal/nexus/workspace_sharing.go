package nexus

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Workspace sharing (capability workspace-sharing).
//
// Observation shares are read-only grants scoped to one project (and
// optionally one agent). The registry denies by default with explicit
// reasons: unknown grants, revoked grants (immediate, no grace period),
// cross-project access, out-of-scope agents and any non-read action are
// all rejected. Viewer is the only role in this phase.

// ShareRoleViewer is the only role supported in phase 1.
const ShareRoleViewer = "viewer"

// ShareGrantMaxTTL bounds any issued grant lifetime. External links must not
// outlive a week even when a caller asks for more (defense against a grant
// being forgotten open).
const ShareGrantMaxTTL = 7 * 24 * time.Hour

// Share actions the registry can authorize.
const (
	ShareActionRead   = "read"
	ShareActionMutate = "mutate"
)

// ObservationShareGrant is one viewer grant bound to a project.
type ObservationShareGrant struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	// AgentID optionally narrows the grant to a single agent. Empty means
	// every agent of the project.
	AgentID   string     `json:"agent_id,omitempty"`
	Role      string     `json:"role"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// ObservationShareRegistry holds grants and answers authorization checks.
// It is safe for concurrent use; every check consults the current state, so
// a revoke takes effect on the next request with no grace period.
type ObservationShareRegistry struct {
	mu     sync.RWMutex
	grants map[string]*ObservationShareGrant
}

// NewObservationShareRegistry returns an empty registry.
func NewObservationShareRegistry() *ObservationShareRegistry {
	return &ObservationShareRegistry{grants: make(map[string]*ObservationShareGrant)}
}

// Grant issues a viewer grant for one project, optionally scoped to an
// agent. Grant IDs are 128-bit random hex values, not sequential. Phase-2
// callers that want a bounded lifetime use GrantWithTTL instead.
func (r *ObservationShareRegistry) Grant(projectID, agentID string, now time.Time) *ObservationShareGrant {
	return r.grantInternal(projectID, agentID, now, nil)
}

// GrantWithTTL issues a viewer grant that stops authorizing after ttl.
// Non-positive TTL is refused (an unbounded link must be an explicit Grant
// call, never an accident); TTLs above ShareGrantMaxTTL are refused rather
// than silently clamped.
func (r *ObservationShareRegistry) GrantWithTTL(projectID, agentID string, now time.Time, ttl time.Duration) (*ObservationShareGrant, error) {
	if ttl <= 0 {
		return nil, fmt.Errorf("workspace sharing: ttl must be positive; use Grant for an explicit unbounded link")
	}
	if ttl > ShareGrantMaxTTL {
		return nil, fmt.Errorf("workspace sharing: ttl %s exceeds the maximum %s", ttl, ShareGrantMaxTTL)
	}
	expires := now.UTC().Add(ttl)
	return r.grantInternal(projectID, agentID, now, &expires), nil
}

func (r *ObservationShareRegistry) grantInternal(projectID, agentID string, now time.Time, expiresAt *time.Time) *ObservationShareGrant {
	grant := &ObservationShareGrant{
		ID:        newShareGrantID(),
		ProjectID: strings.TrimSpace(projectID),
		AgentID:   strings.TrimSpace(agentID),
		Role:      ShareRoleViewer,
		CreatedAt: now.UTC(),
		ExpiresAt: expiresAt,
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.grants[grant.ID] = grant
	return grant
}

// Restore re-registers a persisted grant (workspace-sharing phase 2
// hydration). Identity and timestamps come from the durable record; the
// in-memory registry remains the runtime authority. An expired or revoked
// grant is restored as-is: Authorize denies it honestly instead of hiding it.
func (r *ObservationShareRegistry) Restore(grant ObservationShareGrant) {
	r.mu.Lock()
	defer r.mu.Unlock()
	restored := grant
	r.grants[restored.ID] = &restored
}

// Revoke revokes a grant immediately. Returns false when the grant does not
// exist; revoking twice keeps the first timestamp (first revoke wins).
func (r *ObservationShareRegistry) Revoke(grantID string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	grant, ok := r.grants[strings.TrimSpace(grantID)]
	if !ok || grant.RevokedAt != nil {
		return false
	}
	stamped := now.UTC()
	grant.RevokedAt = &stamped
	return true
}

// Get returns a copy of the grant when it exists.
func (r *ObservationShareRegistry) Get(grantID string) (ObservationShareGrant, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	grant, ok := r.grants[strings.TrimSpace(grantID)]
	if !ok {
		return ObservationShareGrant{}, false
	}
	return *grant, true
}

// List returns project-scoped grants ordered by creation time.
func (r *ObservationShareRegistry) List(projectID string) []ObservationShareGrant {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ObservationShareGrant, 0)
	for _, grant := range r.grants {
		if grant.ProjectID == strings.TrimSpace(projectID) {
			out = append(out, *grant)
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].CreatedAt.Before(out[j-1].CreatedAt); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// Authorize answers one access attempt. It never panics on empty inputs and
// always explains a denial.
func (r *ObservationShareRegistry) Authorize(grantID, projectID, agentID, action string) (string, bool) {
	r.mu.RLock()
	grant, ok := r.grants[strings.TrimSpace(grantID)]
	r.mu.RUnlock()
	if !ok {
		return "unknown grant", false
	}
	if grant.RevokedAt != nil {
		return "grant revoked", false
	}
	if grant.ExpiresAt != nil && time.Now().UTC().After(*grant.ExpiresAt) {
		return "grant expired", false
	}
	if grant.Role != ShareRoleViewer {
		return "unsupported role", false
	}
	if grant.ProjectID != strings.TrimSpace(projectID) {
		return "cross-project access denied", false
	}
	if grant.AgentID != "" && strings.TrimSpace(agentID) != "" && grant.AgentID != strings.TrimSpace(agentID) {
		return "agent outside grant scope", false
	}
	if action != ShareActionRead {
		return "viewer grants are read-only", false
	}
	return "viewer", true
}

func newShareGrantID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand failure is exceptional; fail closed instead of
		// issuing a predictable identifier.
		panic(fmt.Sprintf("workspace sharing: entropy unavailable: %v", err))
	}
	return hex.EncodeToString(buf)
}
