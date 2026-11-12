package nexus

import (
	"testing"
	"time"
)

func TestGrantAuthorizesViewerRead(t *testing.T) {
	registry := NewObservationShareRegistry()
	grant := registry.Grant("proj-1", "", time.Now())
	reason, ok := registry.Authorize(grant.ID, "proj-1", "", ShareActionRead)
	if !ok || reason != "viewer" {
		t.Fatalf("viewer read must be authorized: %s ok=%v", reason, ok)
	}
	if grant.Role != ShareRoleViewer {
		t.Fatalf("only viewer role is issued: %+v", grant)
	}
}

func TestCrossProjectAccessDenied(t *testing.T) {
	registry := NewObservationShareRegistry()
	grant := registry.Grant("proj-1", "", time.Now())
	reason, ok := registry.Authorize(grant.ID, "proj-2", "", ShareActionRead)
	if ok || reason != "cross-project access denied" {
		t.Fatalf("cross-project must be denied explicitly: %s ok=%v", reason, ok)
	}
}

func TestRevokeIsImmediate(t *testing.T) {
	registry := NewObservationShareRegistry()
	grant := registry.Grant("proj-1", "", time.Now())
	if _, ok := registry.Authorize(grant.ID, "proj-1", "", ShareActionRead); !ok {
		t.Fatal("precondition: grant must work before revoke")
	}
	if !registry.Revoke(grant.ID, time.Now()) {
		t.Fatal("first revoke must succeed")
	}
	reason, ok := registry.Authorize(grant.ID, "proj-1", "", ShareActionRead)
	if ok || reason != "grant revoked" {
		t.Fatalf("revoked grant must be denied immediately: %s ok=%v", reason, ok)
	}
	// Double revoke: first timestamp wins, second call is a no-op false.
	if registry.Revoke(grant.ID, time.Now().Add(time.Hour)) {
		t.Fatal("second revoke must not report success")
	}
}

func TestUnknownGrantDenied(t *testing.T) {
	registry := NewObservationShareRegistry()
	reason, ok := registry.Authorize("nope", "proj-1", "", ShareActionRead)
	if ok || reason != "unknown grant" {
		t.Fatalf("unknown grant must be denied: %s ok=%v", reason, ok)
	}
}

func TestViewerCannotMutate(t *testing.T) {
	registry := NewObservationShareRegistry()
	grant := registry.Grant("proj-1", "", time.Now())
	reason, ok := registry.Authorize(grant.ID, "proj-1", "", ShareActionMutate)
	if ok || reason != "viewer grants are read-only" {
		t.Fatalf("viewer must never mutate: %s ok=%v", reason, ok)
	}
}

func TestAgentScopeRespected(t *testing.T) {
	registry := NewObservationShareRegistry()
	grant := registry.Grant("proj-1", "agent-7", time.Now())
	if _, ok := registry.Authorize(grant.ID, "proj-1", "agent-7", ShareActionRead); !ok {
		t.Fatal("in-scope agent must pass")
	}
	reason, ok := registry.Authorize(grant.ID, "proj-1", "agent-8", ShareActionRead)
	if ok || reason != "agent outside grant scope" {
		t.Fatalf("out-of-scope agent must be denied: %s ok=%v", reason, ok)
	}
}

func TestGrantIDsAreRandomNotSequential(t *testing.T) {
	registry := NewObservationShareRegistry()
	first := registry.Grant("proj-1", "", time.Now())
	second := registry.Grant("proj-1", "", time.Now())
	if first.ID == second.ID || len(first.ID) != 32 {
		t.Fatalf("grant ids must be unique 128-bit hex: %s vs %s", first.ID, second.ID)
	}
}

func TestListIsProjectScopedAndOrdered(t *testing.T) {
	registry := NewObservationShareRegistry()
	early := registry.Grant("proj-1", "", time.Now().Add(-time.Minute))
	registry.Grant("proj-2", "", time.Now())
	late := registry.Grant("proj-1", "agent-1", time.Now())
	grants := registry.List("proj-1")
	if len(grants) != 2 || grants[0].ID != early.ID || grants[1].ID != late.ID {
		t.Fatalf("list must scope to project and sort by creation: %+v", grants)
	}
}

func TestGrantWithTTLAuthorizesUntilExpiry(t *testing.T) {
	registry := NewObservationShareRegistry()
	now := time.Now()
	grant, err := registry.GrantWithTTL("proj-1", "", now, time.Hour)
	if err != nil {
		t.Fatalf("GrantWithTTL: %v", err)
	}
	if grant.ExpiresAt == nil || !grant.ExpiresAt.After(now) {
		t.Fatalf("grant must carry a future expiry: %+v", grant)
	}
	if _, ok := registry.Authorize(grant.ID, "proj-1", "", ShareActionRead); !ok {
		t.Fatal("fresh TTL grant must authorize")
	}
	// Expire the grant the way hydration would surface a persisted expired
	// row (Authorize reads the wall clock; the registry has no fake clock).
	expired := now.UTC().Add(-time.Second)
	registry.Restore(ObservationShareGrant{
		ID: grant.ID, ProjectID: "proj-1", Role: ShareRoleViewer,
		CreatedAt: grant.CreatedAt, ExpiresAt: &expired,
	})
	reason, ok := registry.Authorize(grant.ID, "proj-1", "", ShareActionRead)
	if ok || reason != "grant expired" {
		t.Fatalf("expired grant: ok=%t reason=%q", ok, reason)
	}
}

func TestGrantWithTTLRefusesInvalidTTLs(t *testing.T) {
	registry := NewObservationShareRegistry()
	if _, err := registry.GrantWithTTL("proj-1", "", time.Now(), 0); err == nil {
		t.Fatal("zero TTL must be refused")
	}
	if _, err := registry.GrantWithTTL("proj-1", "", time.Now(), -time.Minute); err == nil {
		t.Fatal("negative TTL must be refused")
	}
	if _, err := registry.GrantWithTTL("proj-1", "", time.Now(), ShareGrantMaxTTL+time.Minute); err == nil {
		t.Fatal("TTL above the maximum must be refused, not clamped")
	}
}

func TestRestoreRehydratesPersistedGrant(t *testing.T) {
	registry := NewObservationShareRegistry()
	expires := time.Now().UTC().Add(time.Hour)
	revoked := time.Now().UTC().Add(-time.Minute)
	registry.Restore(ObservationShareGrant{
		ID: "restored-1", ProjectID: "proj-1", Role: ShareRoleViewer,
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour), ExpiresAt: &expires,
	})
	if _, ok := registry.Authorize("restored-1", "proj-1", "", ShareActionRead); !ok {
		t.Fatal("restored live grant must authorize")
	}
	registry.Restore(ObservationShareGrant{
		ID: "restored-2", ProjectID: "proj-1", Role: ShareRoleViewer,
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour), RevokedAt: &revoked,
	})
	reason, ok := registry.Authorize("restored-2", "proj-1", "", ShareActionRead)
	if ok || reason != "grant revoked" {
		t.Fatalf("restored revoked grant: ok=%t reason=%q", ok, reason)
	}
}
