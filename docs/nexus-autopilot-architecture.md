# Nexus Autopilot architecture

Nexus is the visual and operational control plane around the existing Maestro
ecosystem. It does not reimplement Maestro's planner or methodology.

```text
Composer intent
  -> structured WorkPlan / revision
  -> Flow projection (editable façade)
  -> strict admission + immutable execution snapshot (GO)
  -> durable MissionRun / leased workers
  -> Agent execution in isolated workspaces
  -> package evidence + review
  -> global Definition of Done verification
  -> DONE only after evidence, or BLOCKED_NEEDS_USER
```

## Ownership

Maestro owns semantic planning, decomposition, workflow/task schemas, methodology,
skills, gates and shared context conventions. Nexus owns project/agent identities,
workspace and process lifecycle, scheduling, durable run state, Flow presentation,
dispatch, evidence collection and recovery. The mapping is documented in
`docs/nexus-maestro-orchestration-gap-analysis.md`.

## Composer and readiness

Composer persists a living brief and prompt artifact. It refines intent and reports
unknowns/readiness; it does not silently turn an incomplete conversation into a
successful run. Plan generation creates WorkPlan revisions with package goals,
dependencies, acceptance criteria and verification requirements. The backend DAG and
execution contract are authoritative at admission.

## Plan and Flow

`WorkPlan` plus its immutable `PlanRevision` is canonical. `FlowDefinition` is a
lossless façade used by the canvas and inspector. `FlowFromWorkPlan` projects the
plan; `WorkPlanFromFlow` converts accepted edits back before persistence. Node layout
is not dependency semantics. Dependencies, parallel groups, assignment and gates
are explicit plan fields and are validated before a revision can run.

## GO and lifecycle

Approved GO rechecks the plan revision, runs strict preflight, validates Maestro
requirements, freezes a snapshot, creates a durable MissionRun and starts at most one
worker for that run. The worker uses leases and heartbeats. Pause, resume, cancel and
take-control actions coordinate with the worker before changing state.

Operational states include `EXECUTING`, `VERIFYING`, `PAUSED`,
`BLOCKED_NEEDS_USER`, bounded failure states and `COMPLETED_VERIFIED`. The latter is
reserved for the global gate.

## Convergence

Each package follows allocate → compile → execute → evidence verification → independent
review. Provider text is never proof. Failed verification/review enters bounded
remediation; repeated identical failures become `FAILED_NO_PROGRESS` or a human
blocker. When all packages are verified, Nexus runs the persisted global verification
commands. A global failure reopens the most recently verified package and records the
failure evidence; only a subsequent global pass can complete the run.

## Recovery and evidence

MissionRun payload, snapshot ID/revision, package states, dispatch IDs, leases,
capsules, receipts, verifications, reviews and global verification results are
persisted. An unresolved dispatch intent is never automatically duplicated after a
restart. Startup enumerates non-terminal runs and resumes workers; unavailable
providers, missing credentials and unknown outcomes remain explicit blockers.

The UI consumes typed run/evidence endpoints and can show progress, attempts,
assignments, blockers and receipts without requiring raw provider logs.
