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

## Registered balance inventory

Automatic cross-provider balance never scans arbitrary CLIs or invents an
executor from a static table. The inventory is derived from `profile.List()`
through `ListResources`, which is the same registered identity set shown by
`nexus usage`. `BuildBalanceInventory` then admits only authenticated
provider/profile identities with measured quota evidence (`LIVE`, `CACHED`,
`ESTIMATED` or measured `RATE_LIMITED`). `UNKNOWN` and `UNSUPPORTED`
remain visible for diagnostics/manual setup, but are excluded from automatic
balance.

This keeps the useful routing-table idea from executor-based orchestrators while
making Nexus capacity-aware: the route candidates are the AIs the user actually
registered and for which Nexus can observe limits.

Current evidence: recommendation tests and local Web/Go gates. Full automatic
failover E2E remains pending.
