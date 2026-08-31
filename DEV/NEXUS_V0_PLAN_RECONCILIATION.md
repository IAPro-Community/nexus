# Nexus V0 Plan Reconciliation

Baseline: `c96885218b3cff319f481aed4423dc5fd7b52a30`  
Canonical implementation: `feat/nexus-v0-implementation`

| Plan | Task | Required capability | Code evidence | Test evidence | Status |
|---|---|---|---|---|---|
| 01 Core Integrity | terminal protocol | typed control/output separation | `terminalProtocol.ts`, consumers | frontend protocol tests, full suite | PARTIAL |
| 01 Core Integrity | Windows/host/DB/filesystem/resources/config | lifecycle, cleanup, selection, allowlists | existing baseline packages | full Go/race and focused tests; native Windows unavailable | PARTIAL |
| 02 Workspace OS | presentation state | Tabs/Desktop, semantic view identity, persistence | `workspace/presentation.ts`, provider/renderer | presentation/model/state tests | PARTIAL |
| 02 Workspace OS | launcher | open supported core surfaces | `WorkspaceLauncher.tsx`, `launcherModel.ts` | launcher model tests | PARTIAL |
| 02 Direct Work | existing Agent Ask | ask without implicit create; start-and-ask | `AskAgentDialog.tsx`, API/handler/Nexus | model + API validation tests | PARTIAL |
| 02 Direct Work | Project Shell | normal shell lifecycle/cwd | no implementation found | none | NOT_STARTED |
| 03 Maestro + Context | Maestro integration | detect/status/skills/hydration | existing `maestro.go` integration | Maestro tests | PARTIAL |
| 03 Maestro + Context | Context Readiness | MISSING/HYDRATING/READY/STALE/FAILED | no dedicated V0 readiness model found | none | NOT_STARTED |
| 04 Composer | Composer Surface | context-gated composition/direct fallback | no dedicated Composer surface found | none | NOT_STARTED |
| 05 Flow Editor | Flow DAG editor | sequence/dependency/parallel, revisions | existing WorkPlan/PlanBuilder infrastructure | plan/plangraph tests | PARTIAL |
| 06 Flow Run | durable Flow execution | approve, scheduling, receipts, verification | existing Mission runner infrastructure | runner durability/verifier tests | PARTIAL |
| 07 Production Hardening | cross-platform/release/browser/security | same-SHA platform evidence | CI/scripts/docs only | Linux local pass; Windows/macOS/CI unverified | BLOCKED_EXTERNAL |

## Ruling

Existing WorkPlan/Mission infrastructure is retained as the compatibility implementation. V0-specific missing capabilities are not claimed as complete; no duplicate engine was created.
