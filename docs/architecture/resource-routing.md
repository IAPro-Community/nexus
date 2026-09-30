# Resource routing

`ResourceScheduler`/`RecommendResources` is the canonical provider-profile
allocator. Hard operational gates come from `quota.EvaluateEligibility` (the
same function used by the CLI scheduler, runtime routing and explicit account
validation): authentication, network, unavailable health, cooldown, rate-limit,
known exhaustion and `UNKNOWN` evidence.

Quota truth and runtime readiness are deliberately separate:

- `LIVE`/`CACHED`/`ESTIMATED` can contribute measured capacity evidence;
- `UNKNOWN` means Nexus lacks sufficient trustworthy evidence and remains
  fail-closed for automatic allocation;
- `UNSUPPORTED` means the provider does not expose a trustworthy quota metric.
  A healthy authenticated resource remains operationally eligible as
  `OPAQUE`, but receives no fabricated percentage and ranks below measured
  positive capacity;
- explicit `PIN` never silently falls back; `PREFER` may fall back through
  the same scheduler authority.

When every eligible candidate has opaque capacity, account selection uses a
bounded deterministic/LRU policy instead of inventing a quota score.

Agent matching is intentionally separate: an Agent specializes the work, while
the ResourceScheduler chooses an executable provider/profile/model resource.

Current evidence: recommendation/scheduler tests and local Web/Go contracts.
Authenticated cross-provider failover and exact-SHA native release validation
remain separate certification gates.
