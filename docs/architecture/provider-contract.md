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
| Autonomous containment | `AutonomousIsolation` |
| Review containment | `ReviewIsolation` |
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

## Autonomous isolation

The ProviderSpec is also the source of truth for whether a provider can enforce
Nexus' restrictive autonomous contract. `WORKSPACE_WRITE_NO_NETWORK` means the
provider has a verified provider-native sandbox that confines writes to the
workspace and denies network access by default. `READ_ONLY_NO_NETWORK` is the
corresponding review guarantee.

The default Mission contract requires verified containment. Providers marked
`UNVERIFIED` are excluded by the existing ResourceScheduler and fail closed if
they reach launch. They can only use coding auto-approval when the caller
explicitly opts into an unrestricted contract. Review always requires verified
read-only containment.

Codex uses `workspace-write` / `read-only`; autonomous coding also forces
`sandbox_workspace_write.network_access=false` so a profile-level override
cannot silently reopen network access. Gemini may still be launched with
`--sandbox` when an unrestricted contract explicitly permits it, but its
cross-platform no-network/read-only guarantees are marked `UNVERIFIED` until
Nexus can prove them per platform. Command shims remain defense-in-depth; they
are not treated as the sandbox boundary.

## Quota honesty

- `usage.json` is a **cache written by Nexus**, never a primary source.
- Claude → `GET /api/oauth/usage` (kill switch `NEXUS_CLAUDE_USAGE_API=0`).
- Gemini → Cloud Code `retrieveUserQuota` (kill switch `NEXUS_GEMINI_USAGE_API=0`).
- AGY → CLI `--print=/quota` + OAuth refresh.
- Codex → app-server rate limits RPC.
- OpenCode Zen → health probe only → `UNSUPPORTED` (no balance API).
- Cursor → `UNSUPPORTED` (auth via `agent status` only).

Semantics:

- `UNKNOWN` → missing/untrusted evidence; ineligible for automatic allocation. Explicit/manual may proceed with a warning when the caller intentionally overrides the uncertainty.
- `UNSUPPORTED` → the provider does not expose a trustworthy quota metric. If authentication, health, cooldown and rate-limit gates pass, the resource remains operationally eligible with `capacity_evidence=OPAQUE`; it receives no synthetic capacity percentage and ranks below measured positive capacity.
- Missing percent never becomes synthetic `0%` or `100%`.

## Eligibility

All routing layers call `quota.EvaluateEligibility`:

- CLI scheduler (`internal/core/scheduler`)
- `RecommendResources` / `eligibleAccounts` / `validateResourceAccount`
- Flow preflight

Hard gates (auth, network, health unavailable, cooldown, rate limit, UNKNOWN and
known exhaustion) are not reimplemented per caller. `UNSUPPORTED` quota is an
opaque-capacity state, not an operational failure.

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
