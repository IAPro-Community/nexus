# Final hardening report

STATUS: CURRENT  
GIT_SHA: `3760df2d5b3c6656c9c32d9cfc5d922d352c2644` (BASE) + uncommitted hardening on the worktree  
GENERATED_AT: 2026-09-12  
ENVIRONMENT: Linux x86_64, go1.25.0, Node v22.17.0, npm 10.9.2 / pnpm 11.18.0  
EVIDENCE_STREAM_ID: local tests `./internal/nexus` + `./internal/nexus/runner` (337 tests PASS, race PASS on those packages)

BASE SHA: `3760df2d5b3c6656c9c32d9cfc5d922d352c2644`  
Reviewed historical SHA: `1185de89ce91423b10d5913e7f80b418ab1062d6`  
FINAL SHA: uncommitted (no commit requested)

## ARCHITECTURE DESTINATION

**PARTIAL**

Nexus owns operational intelligence. Maestro remains an optional subprocess. WorkPlan is execution truth. Composer/Flow are optional (`ComposerIsRequired` only for CLARIFY; `FlowIsRequired` always false). Attention is a projection. Evidence stream is canonical; Markdown `FINAL_*` reports are now bannered historical.

Remaining architectural debt is compatibility, not a second brain:

- `MaestroSkills` JSON transport (KEEP until SkillIDs-only persisted plans)
- `SourceExternal` not wired in production catalog
- `AgentConfig.Model` still exists as config, but AUTO no longer mints it as the sole WorkUnit pool

## TEST COVERAGE

before (this campaign freeze): `./internal/nexus/...` **59.6%** statements (`DEV/validation/current/coverage-nexus.out`)  
after: same profile plus new watchdog/intent/composer-flow/fault tests; critical funcs improved (watchdog 91.3%, configuredModelCandidates 92.9%, DecideIntent 85.7%, ClassifyTaskRequirements 100%, ResolveTaskModel 100%)

Frontend coverage plugin: **UNVERIFIED** (not installed; not added)

## COMPOSER NORMALIZATION

C1 DIRECT `"corrija este teste quebrado"` → Composer not required.  
C2 `"quero melhorar os testes"` → Project Intelligence facts recorded; no framework/build ritual questions.  
C3 complete specification → PLAN, 0 blocking questions.  
C4 conflicting deve/não deve → CLARIFY, one question, Composer required.

## FLOW NORMALIZATION

WorkPlan → Flow → WorkPlan preserves identity. Semantic goal edit changes WorkPlan payload. Visual-only layout is not a second plan (no layout fields in FlowDefinition).

## ATTENTION PRECISION

Policy unchanged and PASS: running work IGNORE; FAILED_NO_PROGRESS is not NEEDS_YOU; quota/allocation silent. Dedup key survives JSON reopen. `make attention-e2e` exists and PASS locally.

## MAKE TARGETS

Canonical: `make test`, `coverage`, `e2e` (alias of `test-e2e`), `attention-e2e`, `overnight-smoke`, `overnight-soak DURATION=…`, `release-proof`.  
`make quality` unchanged as the local lint/test gate.

## OVERNIGHT SMOKE

`make overnight-smoke` PASS (sandbox + watchdog + fault injection). Duration ~seconds, not 8h.

## OVERNIGHT SOAK

Routine exists: `make overnight-soak DURATION=8h`.  
This host did **not** run 8h. **OVERNIGHT_SOAK = UNVERIFIED**  
Bounded harness `TestOvernightSoakHarnessBounded` PASS (15s default).

## SKILLS / ROUTING / HUMAN INTERVENTION / HANDOFF

Skills catalog tests remain green. AUTO no longer inherits leftover `AgentConfig.Model` as the only candidate. PIN still fail-closed. Intervention duplicate resolve still idempotent. Handoff package tests unchanged (PARTIAL vs live providers).

## PROVIDER PROOF

Codex / OpenCode / AGY authenticated Mission: **UNVERIFIED** (no credentials requested). Adapter fail-closed diffs exist in the worktree.

## PLATFORM PROOF

This run: **Linux amd64 only**. Windows/macOS native same-SHA: **UNVERIFIED**.

## DOCUMENTATION MIGRATION

- Inventory: `DEV/DOCUMENTATION_INVENTORY.md` (1080 markdown files categorized)
- Current status: `DEV/validation/current/RELEASE_STATUS.md`
- Historical `DEV/validation/FINAL_*.md` bannered
- `docs/INDEX.md` + `docs/future/README.md`
- Incremental: files not bulk-moved (link blast radius)

`make docs-verify` still fails on **pre-existing stale visual manifest**, not on the new INDEX. Do not fake visual PASS.

## QUALITY GATES (fresh this session)

| Gate | Result |
|------|--------|
| `make quality` | PASS (exit 0, ~123s; includes `go test ./...` + frontend) |
| `go test -race ./internal/nexus ./internal/nexus/runner ./internal/nexus/skills` | PASS |
| `make attention-e2e` | PASS |
| `make overnight-smoke` | PASS |
| `make coverage` (nexus profile) | PASS 59.6% freeze |
| `make e2e` / `make release-proof` | not re-run this pass (`release-proof` duplicates quality+race+coverage+attention+overnight-smoke+security) |
| `make docs-verify` | FAIL visual manifest stale (pre-existing; not a core behavioral P0) |

## P0

None identified in local core after watchdog + model-pool + intent contract fixes.

## P1 (local/core)

None remaining that violate destination behavior. Leftover `MaestroSkills` is **LEGACY_COMPATIBILITY** with an explicit removal condition (SkillIDs-only persisted plans + web consumer type rename).

## P2

- `mission_executor.Allocate/Execute` 0% unit coverage (I/O; runner uses fakes)
- SourceExternal unwired
- `/api/v1/maestro/*` compatibility API
- ClassifyAttention some summary branches
- Visual manifest / docs-verify stale
- Branch protection cannot be configured from this agent

## UNVERIFIED

- 8h soak
- Authenticated provider Missions and live failover
- Native Windows and macOS evidence for this SHA
- External Maestro repo does not import Nexus
- `make release-proof` as a single umbrella (would re-run quality+race+coverage+security; components already gated)
- External Maestro repo does not import Nexus
- Frontend statement coverage %

## EVIDENCE STREAM IDS

- Conformance: `local-hardening-conformance-3760df2`
- Coverage: `DEV/validation/current/coverage-nexus.out`
- Behavior matrix: `local-hardening-behavior-matrix`

## FINAL VERDICT

**GO** for continuing on `feat/nexus-maximum-delivery` as destination-hardening (local core).  
**NO-GO** for production-certification claims until provider, native Win/macOS, and 8h soak evidence exist.

The product contract is: the user defines work; Nexus must not require supervising AI infrastructure for recoverable quota/model/retry paths. Local tests now lock that contract.
