> **HISTORICAL — NOT CURRENT RELEASE EVIDENCE**
>
> Canonical current status: `DEV/validation/current/RELEASE_STATUS.md`.

# Nexus Final Closure — Reality Audit

Date: 2026-09-11
Base SHA: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
Branch: `feat/nexus-maximum-delivery`
Worktree at audit start: clean

This is the evidence-backed characterization for the final consolidation
campaign. It is not a release claim. Local Linux checks are separated from
provider-authenticated and native-platform evidence. The audit was updated
after the implementation slices; historical “ADD/EXTEND” actions below are
the starting characterization, while the resumable result is in
`FINAL_CLOSURE_CHECKPOINTS.md`.

Continuation observed at commit HEAD `42c22137a4a57ff6b6b80df8125b5a138f32b9e1`
with additional uncommitted complete-stream evidence-projection changes.
The original base SHA above remains the campaign baseline.

## Capability matrix

| Capability | Existing | Evidence | Action |
| --- | --- | --- | --- |
| Project Intelligence | `contextsnapshot.Discover`, immutable `ProjectContextSnapshot`, identity digests, bounded static scan, SQLite persistence and Web read model; operational commands/frameworks/workspace/CI facts now extracted | `internal/nexus/contextsnapshot/*_test.go`, `project_intelligence.go`, `store/project_intelligence.go`; focused and Nexus suites PASS | REUSE/EXTEND: provider-backed relevance and E2E remain open without a second scanner |
| Native Skills | Generic `internal/nexus/skills` catalog/sources now exist; legacy Maestro catalog remains for compatibility | `internal/nexus/skills/*`, `skill_catalog.go`, focused tests | EXTEND: migrate remaining legacy consumer/storage contracts without breaking compatibility |
| Intent Router | Explicit deterministic `DIRECT | CLARIFY | PLAN` decision now sits above existing requirements | `intent_router.go`, focused tests, `internal/app/app.go` | EXTEND: persist decision as first-class mission evidence |
| Composer grounding | Composer receives a bounded identity-bound Project Intelligence envelope for intent/ambiguity analysis | `project_intelligence_context.go`, `composer.go`, focused tests | EXTEND: provider-backed E2E and relevance selection |
| Agent matching | Provider-independent `MatchAgents`, hard gates, score and deterministic ID tie-break | `agent_matching.go`, `agent_matching_test.go` | REUSE; add routing integration tests only |
| Runtime affinity | Canonical `AUTO/PREFER/PIN` desired-vs-actual resolver exists beside the existing scheduler | `runtime_routing.go`, focused tests | EXTEND: wire all live account/model inventory and engine policy |
| Resource routing | `ProviderAccount`, quota confidence, health/cooldown/capability hard gates and deterministic recommendation | `resource_recommendation.go`, `resource_discovery.go`, quota tests | REUSE; consume affinity policy |
| Model routing | Task-aware candidate selection is deterministic and cost-first with reasoning floor | `runtime_routing.go`, focused tests | EXTEND: live health/quota/escalation integration |
| Routing Decision | Typed decision is persisted as bounded `routing_decision` JSON and exposed by `/api/v1/runs/{id}/routing`; the report also projects persisted intent when the WorkPlan is available | `runtime_routing.go`, `routing_report.go`, `run_application.go`, Web route/API tests | EXTEND: full live runtime provenance and restart proof |
| Handoff | `ContextCapsule`, `WorkReceipt`, runtime generations, same-provider honest `NATIVE_RESUME_UNVERIFIED`, cross-provider context handoff | `runner/evidence.go`, `continuity.go`, handoff docs/tests | REUSE; add routing provenance to the existing capsule/receipt only where needed |
| Maestro lifecycle | Advice/status/catalog and optional compiler guidance exist; advice now carries typed `PLAN/TASK/RECOVERY/VERIFY` lifecycle metadata | `maestro.go`, `maestro_test.go`, `handlers_maestro.go`, intelligence context | EXTEND: complete lifecycle matrix and degraded-mode E2E |
| Attention | Durable MissionRunner intervention, idempotent resolution, global runner attention tests and Web runtime attention surfaces | `runner/intervention_test.go`, `runner/attention_e2e_test.go`, Web attention files | REUSE; verify current integration and do not create a second state machine |
| Evidence | Existing append-only ledger is consumed by package/global runner verification, chain-verified before completion and projected by the run application/API | `store/validation_evidence.go`, `validation_evidence.go`, `run_application.go`, Web route, tests | EXTEND: produce a real authenticated Mission stream instance; external proof remains open |

## Confirmed invariants

- `AgentSpec` is already separate from provider/model/account.
- `WorkPlan` is the semantic source; `Flow` is its projection.
- `ContextCapsule` and `WorkReceipt` are bounded evidence contracts, not raw chat history.
- `BLOCKED_NEEDS_USER` and `FAILED_NO_PROGRESS` are distinct runner states.
- Local baseline is green for the focused Core/Intelligence/Runner slice; this
  does not prove live providers, overnight autonomy or native Windows/macOS.

## Current blockers to GO

1. Legacy Maestro-named skill/storage contracts still exist in compatibility
   payloads and the Maestro catalog adapter; canonical Agent execution and
   prompt compilation are source-agnostic, but complete storage migration is
   not claimed.
2. Intent, routing and canonical validation evidence now have typed report
   projections; the evidence read model now has a same-database restart test.
   External authenticated stream proof remains open.
3. The effective Mission path still combines `RecommendResources` with the
   existing scheduler; the unified live allocation contract is not fully
   proven.
4. Model/engine/account provenance and escalation remain contract-level for
   some providers; live health/quota integration is incomplete.
5. No authenticated Mission produced a durable evidence stream ID in this
   campaign. Real provider, overnight, and same-SHA native-platform proof
   remains unavailable or partial and stays `UNVERIFIED`.

## Baseline commands

| Command | Result |
| --- | --- |
| `git fetch --all --prune` + checkout/pull/status/rev-parse/log | PASS; base SHA recorded above |
| `go test ./internal/nexus/... -count=1` | PASS |
| `go vet ./...` | PASS |
| `npm --prefix web run typecheck` | PASS |
| `git diff --check` | PASS |
