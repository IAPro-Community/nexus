# Provider Contract (frozen)

This document freezes how Nexus talks to external AI CLIs (`codex`, `agy`,
`claude`, `gemini`, `cursor`, `opencode`), how usage is measured, and how
routing decides eligibility.

## Single source of truth

Every provider has one [`ProviderSpec`](../../internal/core/provider/spec/) under
`internal/core/provider/spec/`. Specs own:

| Concern | Spec field |
| --- | --- |
| Binary discovery order | `BinaryCandidates` / `ResolveBinary()` |
| Isolated env overlay | `EnvKeys` / `EnvDelete` / `OverlayEnv` |
| Canonical flag aliases | `Normalize` / `BuiltinAliases` |
| Kickoff / resume args | `Kickoff` / `Resume` |
| Mission auto-approval | `AutoApproval` |
| Read-only review mode | `ReviewMode` |
| Quota source class | `Quota` (`LIVE_HTTP` / `LIVE_RPC` / `LIVE_CLI` / `UNSUPPORTED`) |
| Fixture-tested CLI versions | `KnownVersions` |

Adapters (`internal/core/provider/adapters/*`), supervised drivers
(`internal/control/driver/*`), `flags.Normalize`, and
`missionProviderArgs` **derive** from Spec. Do not reintroduce parallel tables.

Contract goldens live in `internal/core/provider/spec/testdata/<provider>/contract.golden`.
Update with `go test ./internal/core/provider/spec -update` or `UPDATE_GOLDENS=1`.
Canonical quota parser fixtures live in
`internal/core/provider/spec/testdata/quota/<provider>/`. Changing a live
response shape requires updating those fixtures and the adapter testdata copies
together.

## Quota honesty

- `usage.json` is a **cache written by Nexus**, never a primary source.
- Claude → `GET /api/oauth/usage` (kill switch `NEXUS_CLAUDE_USAGE_API=0`).
- Gemini → Cloud Code `retrieveUserQuota` (kill switch `NEXUS_GEMINI_USAGE_API=0`).
- AGY → CLI `--print=/quota` + OAuth refresh.
- Codex → app-server rate limits RPC.
- OpenCode Zen → health probe only → `UNSUPPORTED` (no balance API).
- Cursor → `UNSUPPORTED` (auth via `agent status` only).

Semantics:

- `UNKNOWN` → ineligible for automatic Mission allocation; explicit/manual may proceed with a warning.
- `UNSUPPORTED` → eligible only with explicit selection (PIN/PREFER/MANUAL).
- Missing percent never becomes synthetic `0%` exhausted.

## Eligibility

All routing layers call `quota.EvaluateEligibility`:

- CLI scheduler (`internal/core/scheduler`)
- `RecommendResources` / `eligibleAccounts` / `validateResourceAccount`
- Flow preflight

Hard gates (auth, network, health unavailable, cooldown, rate limit, UNKNOWN,
UNSUPPORTED, exhausted) are not reimplemented per caller.

## Failures

Mission/runner failover uses `model.FailureKind` from `classifier.Classify`.
Substring matching for `"quota"` / `"429"` outside the classifier is forbidden
(`internal/core/provider/guardrails_test.go`).
`EventQuotaFailoverCompleted` fires only after a successful destination run.

## Surfaces

| Surface | Payload |
| --- | --- |
| `nexus usage` TUI | `QuotaView` via `profile.GetQuotaView` |
| `nexus usage --json` | snapshot + `quota_view` + `eligibility` |
| Web `/api/v1/resources` | `ProviderAccount` including `eligibility` |

Parity test: `TestUsageParityAcrossSurfaces`.

## Local smoke

```bash
make smoke-providers
```

Not part of CI. Requires authenticated local profiles.

## Changing the contract

1. Update Spec + goldens.
2. Keep adapters/drivers as thin wrappers.
3. Add/adjust fixtures under adapter `testdata/`.
4. Run `go test ./internal/core/provider/... ./internal/core/quota/... ./internal/nexus/...`.
5. Record the decision in `docs/architecture/ADR-provider-contract-freeze.md`.
