// Package entitlement models what an account may do without scattering
// plan-name checks across the codebase. The Core consumes Capabilities,
// never plan labels, so Free/Pro/Team can evolve without touching callers.
//
// Beta scope: a static local service grants the beta capability set. Future
// server-issued signed entitlement documents plug in behind Service without
// changing callers. No billing lives here (and none is implemented).
package entitlement

import "context"

// Capability is a single gated product ability.
type Capability string

const (
	// CapMissionsRun allows starting missions.
	CapMissionsRun Capability = "missions.run"
	// CapMissionsUnlimited lifts monthly mission counting.
	CapMissionsUnlimited Capability = "missions.unlimited"
	// CapAgentsConcurrent gates extra concurrent agents (value = limit).
	CapAgentsConcurrent Capability = "agents.concurrent"
	// CapProvidersMultiAccount allows several accounts per provider.
	CapProvidersMultiAccount Capability = "providers.multi_account"
	// CapRoutingAdvanced gates advanced model routing.
	CapRoutingAdvanced Capability = "routing.advanced"
	// CapRemoteAccess gates tunnel/remote-access surfaces.
	CapRemoteAccess Capability = "remote_access"
	// CapAnalyticsAdvanced gates advanced usage analytics.
	CapAnalyticsAdvanced Capability = "analytics.advanced"
	// CapTeamSharedContext gates shared team context.
	CapTeamSharedContext Capability = "team.shared_context"
)

// Grants is the resolved entitlement snapshot for an account.
type Grants struct {
	Capabilities map[Capability]bool `json:"capabilities"`
	// MaxConcurrentAgents bounds parallel agent execution.
	MaxConcurrentAgents int `json:"max_concurrent_agents"`
	// MaxMonthlyMissions bounds counted missions; <0 means unlimited.
	MaxMonthlyMissions int `json:"max_monthly_missions"`
	// RetentionDays bounds server-side history retention displays.
	RetentionDays int `json:"retention_days"`
	// PlanLabel is display-only. Never branch on it; use Capabilities.
	PlanLabel string `json:"plan_label"`
}

// Has reports whether a capability is granted.
func (g Grants) Has(c Capability) bool { return g.Capabilities[c] }

// Service resolves entitlements for accounts.
type Service interface {
	GrantsFor(ctx context.Context, accountID string) (Grants, error)
}

// StaticService grants a fixed beta set. It is the beta implementation;
// a server-backed service replaces it without caller changes.
type StaticService struct {
	Grants Grants
}

// BetaGrants is the v1.0.0-beta capability set: everything free while beta.
func BetaGrants() Grants {
	return Grants{
		Capabilities: map[Capability]bool{
			CapMissionsRun:           true,
			CapMissionsUnlimited:     true,
			CapAgentsConcurrent:      true,
			CapProvidersMultiAccount: true,
			CapRoutingAdvanced:       true,
			CapRemoteAccess:          true,
			CapAnalyticsAdvanced:     true,
			CapTeamSharedContext:     false,
		},
		MaxConcurrentAgents: 4,
		MaxMonthlyMissions:  -1,
		RetentionDays:       90,
		PlanLabel:           "beta",
	}
}

// GrantsFor implements Service.
func (s StaticService) GrantsFor(_ context.Context, _ string) (Grants, error) {
	g := s.Grants
	if g.Capabilities == nil {
		g = BetaGrants()
	}
	return g, nil
}

// Require is a tiny gate helper for call sites: it reports whether a
// capability is present in already-resolved grants.
func Require(g Grants, c Capability) bool { return g.Has(c) }
