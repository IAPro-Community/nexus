# Usage, quota and capacity evidence

`model.UsageSnapshot` remains the backwards-compatible canonical quota projection used by CLI, Web, TUI and the scheduler. Beta 6 adds a provider-boundary evidence contract without forcing existing surfaces to migrate at once.

## Resource channels

Subscription quota and API consumption are independent resource channels:

- `SUBSCRIPTION` — product/account quota such as Codex 5h/weekly or provider subscription pools;
- `API` — metered API requests, tokens, cost, budget and throughput limits.

Nexus must never add these channels together or infer that API health implies subscription capacity (or the reverse).

## Provider-native observations

`model.UsageObservation` records the provider-native meaning before normalization. Percentage evidence declares its semantics explicitly:

- `USED_PERCENT`
- `REMAINING_PERCENT`
- `USED_ABSOLUTE`
- `REMAINING_ABSOLUTE`
- `BINARY_AVAILABILITY`

The quota layer owns normalization. Adapters must not duplicate provider-specific inversion rules throughout the codebase.

Examples:

- Codex reports `usedPercent=0` → canonical `remaining=100`, `used=0`;
- AGY reports `100% Limit Remaining` → canonical `remaining=100`, `used=0`.

`quota.NormalizePercentEvidence` is the single percentage arithmetic boundary for these semantics.

## Canonical snapshot

`model.UsageSnapshot` represents provider, profile/account, status, source, observation time, optional expiry, model/plan, confidence, and one or more windows with optional absolute limit/used/remaining values, percentage values, units, and reset descriptions.

Absolute fields are nullable because providers may expose only ratios. Nil is unknown and is never converted into zero or unlimited.

Statuses distinguish `LIVE`, `CACHED`, `ESTIMATED`, `UNSUPPORTED`, `RATE_LIMITED`, and `UNKNOWN`. `UNKNOWN` means Nexus has no trustworthy current observation. It is not zero, unlimited, or a fabricated 100% value.

## Capacity semantics

Public `quota_remaining` has exactly one meaning: literal remaining capacity normalized to 0..1.

- single-pool providers such as Codex use the tightest active window in that pool;
- multi-pool providers such as AGY expose the best independently usable pool for generic account capacity;
- model-family breadth and aggregate scheduling preference remain internal ranking signals and are never encoded into `quota_remaining`.

A model-specific scheduler must still validate the requested pool.

## Routing freshness

Automatic selection uses evidence tiers:

1. `LIVE` / trustworthy `CACHED`;
2. `ESTIMATED` attributable last-known evidence;
3. no measured evidence.

Fresh evidence always outranks an estimate. When every eligible profile only has `ESTIMATED` evidence, Nexus ranks those measured estimates instead of discarding them and rotating by unrelated LRU. LRU is reserved for the absence of measured quota evidence.

## Live sources

Primary subscription sources remain:

- AGY: CLI `--print=/quota`;
- Codex: app-server `account/rateLimits/read`;
- Claude: `GET /api/oauth/usage`;
- Gemini: existing Cloud Code quota integration (kept for compatibility);
- OpenCode Zen: health only → `UNSUPPORTED`;
- Cursor: `UNSUPPORTED`.

`usage.json` is a cache written by Nexus, never a primary source.

Future API consumption collectors should populate the `API` channel from provider-authoritative usage/cost endpoints and/or per-request telemetry, then reconcile that against local execution attribution. They must not mutate subscription quota.

## Eligibility

Routing eligibility remains centralized in `quota.EvaluateEligibility`. See [provider-contract.md](provider-contract.md). Canonical parser fixtures live in `internal/core/provider/spec/testdata/quota/`.

The quota engine enforces a trust window for cached observations and preserves last-known evidence only with explicit degraded/estimated status. Missing source, observation time, account attribution or usable windows never become invented capacity.

## Adding providers

A new quota integration must declare:

1. resource channel;
2. source;
3. native semantics;
4. model/pool identity;
5. window/reset semantics;
6. confidence/freshness;
7. degradation behavior.

No provider-specific percentage inversion should be implemented outside the evidence normalization boundary.
