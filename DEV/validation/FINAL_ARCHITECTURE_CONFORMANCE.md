# Final Architecture Conformance

STATUS: CURRENT  
GIT_SHA: `3760df2d5b3c6656c9c32d9cfc5d922d352c2644`  
GENERATED_AT: `2026-09-12T04:30:00Z`  
ENVIRONMENT: `linux/amd64 go1.25.0 node v22.17.0`  
EVIDENCE_STREAM_ID: `local-hardening-conformance-3760df2`

This document is the pre-fix audit. Code after this SHA may close PARTIAL
findings; those closures belong in `DEV/validation/current/FINAL_HARDENING_REPORT.md`.

Reviewed HEAD reference: `1185de89ce91423b10d5913e7f80b418ab1062d6`  
Authoritative HEAD at audit: `3760df2d5b3c6656c9c32d9cfc5d922d352c2644` (1 commit ahead: Codex CrossAccountResume).

Dirty tree at audit (kept as in-progress hardening, not discarded):

- `internal/control/web/auth.go` — Origin-only desktop auth removed (fail-closed)
- provider adapter session/auth tightenings
- `internal/nexus/plan_application.go` — intent decision error facts

## Verdict

Destination architecture is **implemented with remaining PARTIAL debt**. No
invariant is a hard FAIL of ownership. No redesign is warranted.

| Invariant | Classification |
|-----------|----------------|
| 4.1 Nexus owns operational intelligence | PARTIAL |
| 4.2 Maestro owns methodology only; Nexus→Maestro | PARTIAL |
| 4.3 Skills catalog | PARTIAL + LEGACY_COMPATIBILITY |
| 4.4 Agent ≠ resource identity | PARTIAL |
| 4.5 Model decision WorkUnit-scoped | PARTIAL |
| 4.6 WorkPlan canonical; Composer/Flow optional | PARTIAL |
| 4.7 Attention is a projection | PASS |
| 4.8 ValidationEvidenceStream canonical | PARTIAL |

---

## 4.1 Nexus owns operational intelligence — PARTIAL

**Sources:** `internal/nexus/project_intelligence.go`, `intent_router.go`, `skills/`, `agent_matching.go`, `runtime_routing.go`, `scheduler.go`, `quota_monitor.go`, `mission_executor.go`, `runner/`, `runtime_application.go`, `runner/attention_center.go`, `validation_evidence.go`

**Tests:** `intent_router_test.go`, `agent_matching_test.go`, `runtime_routing_test.go`, `runner/*_test.go`, `validation_evidence_test.go`, `autopilot_contract_test.go`

**Debt:** parallel Maestro catalog types in `maestro.go`; Personas not a first-class type (mapped to `Agent.Role`); live provider E2E UNVERIFIED.

---

## 4.2 Maestro standalone — PARTIAL

**Sources:** `internal/nexus/maestro.go` (subprocess client), `docs/architecture/ADR-maestro-integration.md`

**Forbidden import Maestro→Nexus:** none in this repo (UNVERIFIED for the external Maestro binary).

**Debt:** `/api/v1/maestro/*` compatibility surface; `docs/nexus-maestro-orchestration-gap-analysis.md` still reads as competing ownership.

**Migration:** keep Maestro as optional advice + `SourceMaestro`. Do not import Nexus from Maestro.

---

## 4.3 Skills — PARTIAL + LEGACY_COMPATIBILITY

**Canonical:** `internal/nexus/skills/types.go` SourceBuiltin/Project/User/External/Maestro. Consumers receive `nexusskills.Skill`.

**Legacy transport:** `MaestroSkills` JSON on WorkPlan/Flow/Package (`skill_contract.go`). Removal condition: all persisted plans have `SkillIDs` AND web types no longer import `MaestroSkill` as the consumer type. Target: deprecate in next minor after migration tests stay green for one RC.

**Debt:** `SourceExternal` unwired in production `skillCatalogForProject`; web `MaestroSkill` alias.

---

## 4.4 Agent identity — PARTIAL

**Sources:** `internal/nexus/store/models.go` Agent vs RuntimeGeneration.

**Debt:** `AgentConfig` still embeds provider/profile/model used as allocation hints. Not a second identity, but easy to confuse. Keep as config, do not split types in this campaign unless tests prove mutation of Agent.ID.

---

## 4.5 Model routing WorkUnit-scoped — PARTIAL (fix in this campaign)

**Sources:** `ResolveTaskModel` in `runtime_routing.go` is WorkUnit-scoped.

**Gap:** `configuredModelCandidates` mints a singleton pool from `AgentConfig.Model` when the WorkUnit has no candidates, inheriting the previous step’s model.

**Tests existing:** PIN/PREFER/AUTO in `runtime_routing_test.go`. Missing: leftover AgentConfig.Model must not become the only AUTO candidate.

---

## 4.6 WorkPlan / Composer / Flow — PARTIAL (contract tests in this campaign)

**Sources:** `store/plans.go`, `flow.go`, `composer.go`. CLI Direct skips Composer (`internal/app/app.go` Simple=DIRECT).

**Gap:** no single suite proving IntentRouter → Composer optional → WorkPlan → Flow optional → Mission. Composer API can still be created for DIRECT goals if a client calls it (optional capability, not auto-required).

---

## 4.7 Attention — PASS

**Sources:** `runner/attention_policy.go`, `attention_center.go`. Notification is `web/src/notifications/attentionDelivery.ts`.

Quota failover, retry, remediation → IGNORE. NEEDS_USER only for `BLOCKED_NEEDS_USER`. FAILED_NO_PROGRESS → IN_APP/NOTIFY, not REQUIRE_USER.

**Debt:** no named Make target; restart matrix incomplete (fix in this campaign).

---

## 4.8 Evidence — PARTIAL

**Sources:** `store/validation_evidence.go`, hash chain tests.

**Gap:** Markdown reports under `DEV/validation/FINAL_*.md` look like current PASS claims. Canonical current status must live under `DEV/validation/current/`.

---

## Watchdog / overnight — PARTIAL

`MaxNoProgress` + identical-failure remediation exist. No stall watchdog on ALLOCATING/EXECUTING/VERIFYING/REMEDIATING without timestamp progress. No `make overnight-smoke` / `overnight-soak`.

---

## Make surface at audit

Present: `quality`, `quality-full`, `test-go`, `test-frontend`, `test-e2e`, `race`, `security`.  
Absent: `test`, `coverage`, `e2e`, `attention-e2e`, `overnight-smoke`, `overnight-soak`, `release-proof`.

---

## Remaining non-local UNVERIFIED

- Authenticated Codex/OpenCode/AGY Mission closure
- Native Windows / macOS same-SHA evidence on this host
- 8h overnight soak
- External Maestro repository does not import Nexus
