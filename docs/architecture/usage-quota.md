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


## Usage/Capacity V2: evidence and resource channels

Nexus distinguishes source evidence from the projections used by routing.

- `UsageObservation` records the provider's original semantics
  (`USED_PERCENT`, `REMAINING_PERCENT`, absolute values, etc.).
- `SUBSCRIPTION` and `API` are independent resource channels. A ChatGPT/Codex
  subscription session never inherits OpenAI API budget merely because the
  identities belong to the same person.
- `quota_remaining` is a literal 0..1 projection of the best usable quota pool.
  Pool-count bonuses and other routing preferences live in
  `routing_capacity_score` and must never be rendered as percentages.
- The legacy `UsageSnapshot` remains the compatibility projection for current
  TUI/Web/scheduler consumers while collectors migrate to typed evidence.
- API consumption/cost may be attached only through explicit credentials and
  account/project binding. Nexus must not reuse a product credential as an API
  credential or infer that one grants access to the other.

For multi-account providers, every observation stays bound to `AccountScope`.
For multi-pool providers such as AGY, model groups remain independent rather
than being averaged into one fabricated account percentage.


### Optional OpenAI API telemetry

OpenAI API telemetry is disabled unless a managed profile has an explicit
binding. The binding stores only attribution metadata and the **name** of the
environment variable that contains an OpenAI Admin API key:

```bash
export OPENAI_ADMIN_KEY_WORK='<admin-key>'
nexus config api-usage bind codex work \
  --project proj_123 \
  --key-env OPENAI_ADMIN_KEY_WORK
```

Nexus then reads month-to-date completion usage from
`/v1/organization/usage/completions`, attributable project costs from
`/v1/organization/costs`, and the project spend limit when exactly one
project is bound. The result is cached under the exact profile/account scope;
ordinary resource listing is cache-only and performs no network I/O.

A binding must contain at least one project, user, or API-key identifier.
Organization-wide unfiltered usage is never assigned to one CLI profile. API
consumption/cost does not populate subscription `quota_remaining`.
