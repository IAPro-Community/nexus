# Usage and quota evidence

`model.UsageSnapshot` is the canonical usage value. It represents provider,
profile/account, status, source, observation time, optional expiry, model/plan,
confidence, and one or more windows with optional absolute limit/used/remaining
values, percentage values, units, and reset descriptions. Absolute fields are
nullable because providers may expose only ratios; nil is unknown and is never
converted into zero or unlimited.

The statuses distinguish `LIVE`, `CACHED`, `ESTIMATED`, `UNSUPPORTED`,
`RATE_LIMITED`, and `UNKNOWN` (plus provider-specific degraded states where
applicable). `UNKNOWN` means Nexus has no trustworthy current observation. It is
not zero, unlimited, or a fabricated 100% value. A missing remaining percent
never becomes synthetic `0%` exhausted.

Live sources (never `usage.json` as primary evidence):

- AGY: CLI `--print=/quota`
- Codex: app-server `account/rateLimits/read`
- Claude: `GET /api/oauth/usage`
- Gemini: Cloud Code `retrieveUserQuota`
- OpenCode Zen: health only → `UNSUPPORTED`
- Cursor: `UNSUPPORTED`

Routing eligibility is a single function, `quota.EvaluateEligibility`. See
[provider-contract.md](provider-contract.md). Canonical parser fixtures live in
`internal/core/provider/spec/testdata/quota/`.

The quota engine enforces a trust window for cached observations and preserves
legacy files only when their timestamps and fields provide attributable
evidence. A missing source, observation time, or usable window remains
unknown. The source and timestamp are retained for UI/CLI explanation and
future timeline work.

New quota integrations must report evidence, reset/window semantics, and
degradation explicitly. This cycle does not add new scraping.
