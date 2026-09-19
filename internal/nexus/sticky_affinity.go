package nexus

// Sticky affinity reuses the existing seams (RuntimeAffinityPolicy,
// RuntimeRoutingDecision and the mission allocate path). A live session stays
// on its current resource while that account remains healthy, authenticated,
// eligible and free of exhaustion/rate-limit/cooldown pressure. No persistent
// RouteLease is introduced: stickiness is evaluated per allocation from live
// eligibility, never from durable lease state.

// ShouldRetainCurrentResource reports whether the current Provider/Profile
// must be retained. It delegates to the existing selectCurrentResource +
// quota eligibility seam so health, auth, UNKNOWN, exhaustion, rate-limit and
// cooldown stay canonical in one place.
func ShouldRetainCurrentResource(accounts []ProviderAccount, current AgentConfig, req TaskRequirements, policy SchedulerPolicy) bool {
	_, keep := selectCurrentResource(accounts, current, req, policy)
	return keep
}
