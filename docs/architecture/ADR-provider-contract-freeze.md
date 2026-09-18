# ADR: Provider contract freeze

- Status: Accepted
- Date: 2026-09-13
- Related: [provider-contract.md](./provider-contract.md), [usage-quota.md](./usage-quota.md), [resource-routing.md](./resource-routing.md)

## Context

Provider launch, usage probing and routing regressed repeatedly because the same
facts (binary, env, flags, quota source, eligibility) were duplicated across
adapters, drivers, mission args and schedulers. Red-team `PRODUCTION_NO_GO`
providers/quota lane failed on UNKNOWN eligibility, synthetic 0%, pass-through
`usage.json`, and substring failover.

## Decision

1. Freeze one `ProviderSpec` per provider as the only place that defines binary
   candidates, env overlay, flag aliases, kickoff/resume, mission approval and
   quota source class.
2. Treat `usage.json` as cache only; live sources must be attributable APIs/RPCs/CLIs
   with `AccountScope`. Undocumented endpoints degrade; kill switches remain.
3. Route exclusively through `quota.EvaluateEligibility`.
4. Fail over on typed `FailureKind` only.
5. Guard the freeze with Spec goldens and `guardrails_test.go`.

## Consequences

- Cursor binary order is `cursor-agent` then `agent`.
- Codex kickoff is `exec <prompt>` without a hardcoded model id.
- Claude/Gemini announce live usage only when their probes succeed; Cursor and
  OpenCode Zen stay `UNSUPPORTED` for AUTO allocation.
- Changing CLI args/env without updating goldens fails CI/tests.

## Non-goals

- Copilot provider
- Guaranteeing overnight authenticated Mission certification in this ADR
