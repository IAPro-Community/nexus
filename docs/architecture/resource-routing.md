# Resource routing

`ResourceScheduler`/`RecommendResources` is the canonical provider-profile
allocator. Hard gates come from `quota.EvaluateEligibility` (the same function
used by the CLI scheduler, runtime routing and explicit account validation):
authentication, network, health, cooldown, rate-limit, exhausted windows,
UNKNOWN and UNSUPPORTED. UNKNOWN quota is never treated as full capacity and is
ineligible for automatic allocation; UNSUPPORTED and UNKNOWN available accounts
may proceed only with explicit selection.

Agent matching is intentionally separate: an Agent specializes the work, while
the ResourceScheduler chooses an executable provider resource.

Current evidence: recommendation tests and local Web/Go gates. Full automatic
failover E2E remains pending.
