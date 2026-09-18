# Nexus × Maestro orchestration gap analysis

Audit date: 2026-09-06. Sources: Nexus `internal/nexus`, `web/src/features/work`,
and Maestro `<orchestrator-repository>` (`TASK_SCHEMA.json`,
`WORKFLOW_SCHEMAS.json`, `runtime/`, `tests/`). Nexus remains the execution/control
plane; Maestro remains the owner of methodology, task/workflow contracts and planning
advice.

| Capability | Nexus today | Maestro today | Reuse? | Gap | Target owner | Implementation decision | Evidence/path |
|---|---|---|---|---|---|---|---|
| Task lifecycle | `WorkPackage`, `PackageRun`, durable states | `TASK_SCHEMA` lifecycle and artifacts | EXTEND_MAESTRO | Mapping is adapter-level | Shared contract + Nexus runtime | Keep WorkPlan/PackageRun canonical locally; map statuses without copying planner | `internal/nexus/runner/types.go`, `orquestrador/TASK_SCHEMA.json` |
| Plan representation | WorkPlan revisions/snapshots | planner/plan artifacts and revisions | REUSE_AS_IS | external plan import is optional | Maestro | Nexus consumes validated plan data and freezes revision | `internal/nexus/plan.go`, `runtime/planner/plan-*` |
| Dependencies | DAG validation and execution waves | graph validator/DAG utilities | REUSE_AS_IS | no duplicate semantic planner | Maestro plan + Nexus scheduler | Nexus validates execution graph; Maestro owns semantic generation | `internal/nexus/flow.go`, `runtime/planner/graph-validator.js` |
| Artifacts | typed capsules/receipts and prompt artifacts | artifact store/renderer | ADAPT_IN_NEXUS | common transport is not yet machine API | Nexus adapter | Persist bounded execution evidence; do not copy memory | `internal/nexus/runner/evidence.go` |
| Approvals / GO | revision check and approved run endpoint | plan approval and human gates | EXTEND_MAESTRO | GO must bind Nexus snapshot | Nexus control plane | GO is admission + durable snapshot + worker start | `internal/nexus/mission_service.go`, `runtime/planner/plan-approval-gate.js` |
| Verification / DoD | package verification/review plus global verification | workflow verify phase and gates | REUSE_AS_IS | global result needed before DONE | Nexus execution + Maestro gate definitions | Added explicit persisted global verification gate | `internal/nexus/runner/runner.go` |
| Agent assignment | AUTO/EXISTING/CREATE, provider/resource policy | execution agent/executor fields | ADAPT_IN_NEXUS | assignment is a runtime concern | Nexus | Reuse assignment concepts; Nexus scheduler enforces them | `internal/nexus/flow.go`, `TASK_SCHEMA.json` |
| Parallel execution | `ParallelGroup`, leases, bounded group dispatch | descriptive workflow only | ADAPT_IN_NEXUS | no execution engine in Maestro | Nexus | Keep parallel dispatch in MissionRunner | `internal/nexus/runner/runner.go` |
| Flow visualization | Flow façade, canvas, inspector, revisions | no Web canvas authority | ADAPT_IN_NEXUS | layout is presentation only | Nexus | Plan → Flow projection; mutations round-trip to WorkPlan | `internal/nexus/flow.go`, `web/src/features/work/flowModel.ts` |
| Plan mutation | revisioned WorkPlan updates, client DAG edits | plan revision/diff services | EXTEND_MAESTRO | Nexus UI mutation endpoint is narrower | Nexus adapter | Store every accepted edit as WorkPlan revision and reject invalid DAG | `internal/nexus/store/plans.go`, `web/src/features/work/planBuilderModel.ts` |
| Recovery | persisted runs, leases, startup recovery | workflow state/lock and run store | ADAPT_IN_NEXUS | provider outcome uncertainty remains human-blocked | Nexus | Reconcile leases; never duplicate unresolved dispatch | `internal/nexus/mission_service.go`, `runner/repository.go` |
| Provider failover | quota/resource scheduler and assignment policy | model router/recommendations | ADAPT_IN_NEXUS | live policy bridge is optional | Nexus | AUTO may reallocate; fixed assignment surfaces conflict | `internal/nexus/scheduling.go`, `runtime/planner/model-router.js` |
| Stagnation | bounded identical-failure detection | retry/manual workflow semantics | EXTEND_MAESTRO | strategy change is runtime-specific | Nexus | bounded remediation and `FAILED_NO_PROGRESS`; no infinite retry | `internal/nexus/runner/runner.go`, `runner/retry.go` |
| Communication routing | no free-form group chat; typed evidence context | shared context/memory and handoffs | REUSE_AS_IS | directed ASK/ANSWER transport not yet exposed | Shared adapter | Prefer capsules, receipts and artifacts; add message transport only with a real owner | `internal/nexus/runner/evidence.go`, `runtime/context/` |
| Runtime persistence | SQLite MissionRun payload + lease columns | private RunStore JSON | ADAPT_IN_NEXUS | cross-process recovery needs store tests | Nexus | SQLite remains authoritative for Nexus Run state | `internal/nexus/mission_repository.go`, `internal/nexus/store/autonomy.go` |

## CREATE_NEW decisions

No new semantic Task, Workflow, Plan, Gate, Approval, Memory or Agent model was
created. `FlowDefinition` is an explicitly Nexus-owned projection/presentation
facade, not a second source of truth. `GlobalVerificationCommands` is a runtime
execution field because Nexus owns the process/workspace and must persist evidence;
it does not define Maestro methodology or replace Maestro gates.

## Current honest boundary

Maestro's schemas and planner are descriptive/local-runtime contracts and do not
provide a stable Nexus execution API. Nexus therefore uses its existing bridge and
fail-closed capability validation. A real provider-authenticated overnight run and
native platform process recovery remain environment-dependent evidence, not claims
that can be inferred from compilation.
