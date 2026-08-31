# Nexus Flow Run Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-08-30-nexus-workspace-os-composer-flow-design.md`

## Global Constraints

- Target only `feat/nexus-maximum-delivery`.
- Baseline design freeze: `c96885218b3cff319f481aed4423dc5fd7b52a30`.
- Do not rewrite the application from scratch.
- Preserve Project Rail, Topbar, Workspace OS visual language, Taskbar, persistent Agent identity, terminal broker, Resource Scheduler, worktrees, plan persistence, Mission durability, and Maestro independence.
- Closing a Workspace View must never implicitly stop an Agent.
- Direct Agent and Project Shell work must remain usable without Composer.
- Composer planning/refinement is blocked unless Project Context is `READY`.
- User-facing Flow is a compatibility façade over proven WorkPlan/Mission infrastructure until a safe data migration is justified.
- Flow V0 supports only sequence, dependency, and parallel DAG semantics.
- Flow task context defaults to a fresh scoped Context Capsule.
- No production claim without same-SHA Linux, macOS, Windows, browser, security and release evidence.
- Use TDD: failing test → minimal implementation → passing test → focused commit.


**Goal:** Execute approved static Flows to verified completion with Mission durability, safe parallelism, fresh contexts and Work Receipts.

**Architecture:** Keep Mission runner persistence/recovery; add Flow façade, ready-Step scheduling, Context Capsules, receipts and Flow Run Surface.

**Tech Stack:** Go mission/runner/scheduler, React, TypeScript, Agent/worktree infrastructure.

---

### Task 1: Lock Mission runner contracts

**Files:**
- Create: `internal/nexus/mission_runner_contract_test.go`
- Read/modify only if tests expose defects:
  - `internal/nexus/mission_service.go`
  - `internal/nexus/mission_executor.go`
  - `internal/nexus/mission_execution.go`
  - `internal/nexus/mission_repository.go`

- [ ] **Step 1: Add tests**
Persistence/reload, lease/fencing, pause, cancel, verification terminal states, Take Control behavior already supported.

- [ ] **Step 2: Run**

```bash
go test ./internal/nexus -run 'TestMission.*Contract' -v
```

- [ ] **Step 3: Fix any exposed current defects before new Flow behavior**

- [ ] **Step 4: Commit contract tests/fixes**

```bash
git add internal/nexus
git commit -m "test: lock mission runner durability contracts"
```

---

### Task 2: Add Flow Run façade

**Files:**
- Create: `internal/nexus/flow_run.go`
- Create: `internal/nexus/flow_run_test.go`
- Modify: `internal/control/web/handlers_nexus.go`
- Modify: `web/src/nexus/api.ts`
- Create: `web/src/features/work/flowRunModel.ts`
- Create: `web/src/features/work/flowRunModel.test.ts`

```go
type FlowRunState string

const (
    FlowQueued FlowRunState = "QUEUED"
    FlowReady FlowRunState = "READY"
    FlowRunning FlowRunState = "RUNNING"
    FlowVerifying FlowRunState = "VERIFYING"
    FlowCompleted FlowRunState = "COMPLETED"
    FlowBlocked FlowRunState = "BLOCKED"
    FlowFailed FlowRunState = "FAILED"
    FlowCanceled FlowRunState = "CANCELED"
)
```

- [ ] **Step 1: Test every Mission→Flow state mapping**
Unknown is never completed.

- [ ] **Step 2: Implement façade over Mission service**

- [ ] **Step 3: Add frontend normalized model**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus -run TestFlowRun -v
npm --prefix web test -- flowRunModel.test.ts
git add internal/nexus/flow_run* internal/control/web/handlers_nexus.go web/src/nexus/api.ts web/src/features/work/flowRunModel*
git commit -m "feat: expose mission execution as flow run"
```

---

### Task 3: Add ready-Step scheduler and bounded parallel waves

**Files:**
- Modify: `internal/nexus/scheduler.go`
- Modify: `internal/nexus/scheduler_test.go`
- Modify: `internal/nexus/mission_executor.go`
- Modify: `internal/nexus/mission_execution.go`
- Reuse: `internal/nexus/plangraph/`

- [ ] **Step 1: Test A → B||C → D**
A initially; B+C after A; D after both; failed B blocks D; concurrency=1 safely serializes.

- [ ] **Step 2: Implement scheduler from graph truth**

- [ ] **Step 3: Persist dispatch state before worker launch**

- [ ] **Step 4: Stress test**

```bash
go test ./internal/nexus -run 'Test.*Scheduler|Test.*Mission' -count=10
```

- [ ] **Step 5: Commit**

```bash
git add internal/nexus/scheduler* internal/nexus/mission_executor.go internal/nexus/mission_execution.go
git commit -m "feat: execute ready flow steps with bounded parallelism"
```

---

### Task 4: Add Context Capsule

**Files:**
- Create: `internal/nexus/context_capsule.go`
- Create: `internal/nexus/context_capsule_test.go`
- Modify: `internal/nexus/mission_execution.go`
- Modify: `internal/nexus/intelligence_provider.go`

```go
type ContextCapsule struct {
    ProjectID string
    Branch string
    Head string
    FlowID string
    FlowRevision int
    StepID string
    Goal string
    RelevantMemory []ContextRef
    RelevantPaths []string
    DependencyReceipts []string
    MaestroSkills []string
    AcceptanceCriteria []string
    Constraints []string
}
```

Use existing context reference types if present.

- [ ] **Step 1: Test scope**
Required dependency receipt included; unrelated transcript/source excluded.

- [ ] **Step 2: Implement bounded capsule**

- [ ] **Step 3: Default Flow tasks to fresh task context**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus -run 'TestContextCapsule|TestMission' -v
git add internal/nexus/context_capsule* internal/nexus/mission_execution.go internal/nexus/intelligence_provider.go
git commit -m "feat: isolate flow task context capsules"
```

---

### Task 5: Add Work Receipt persistence and handoff

**Files:**
- Create: `internal/nexus/work_receipt.go`
- Create: `internal/nexus/work_receipt_test.go`
- Modify: `internal/nexus/mission_repository.go`
- Modify concrete Mission store migration/schema file under `internal/nexus/store/`
- Modify: `internal/nexus/mission_executor.go`

**Receipt fields:**
step/status/summary, changed files, commands, tests, decisions, artifacts, remaining issues, verification, agent, base/result revision, timestamps.

- [ ] **Step 1: Add persistence round-trip test**

- [ ] **Step 2: Add B+C receipts → D Context Capsule handoff test**

- [ ] **Step 3: Implement structured receipt**
Empty evidence stays empty; never fabricate.

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus -run 'TestWorkReceipt|TestMission' -v
git add internal/nexus/work_receipt* internal/nexus/mission_repository.go internal/nexus/mission_executor.go internal/nexus/store
git commit -m "feat: persist work receipts for clean flow handoffs"
```

---

### Task 6: Add Guided / Autonomous policy façade

**Files:**
- Create: `internal/nexus/flow_policy.go`
- Create: `internal/nexus/flow_policy_test.go`
- Modify: `internal/nexus/mission_execution.go`
- Modify: `internal/nexus/mission_service.go`
- Modify: `web/src/features/work/missionAutonomyModel.ts`
- Modify: `web/src/features/work/missionAutonomyModel.test.ts`

```go
type FlowAutonomyPolicy string

const (
    FlowGuided FlowAutonomyPolicy = "GUIDED"
    FlowAutonomous FlowAutonomyPolicy = "AUTONOMOUS"
)
```

- [ ] **Step 1: Test Guided blocking cases**
Critical missing fact, destructive out-of-policy action, unremediated verification failure, scope expansion.

- [ ] **Step 2: Map existing advanced AutonomyContract behind façade**

- [ ] **Step 3: Assert no completion without required verification**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus
npm --prefix web test -- missionAutonomyModel.test.ts
git add internal/nexus/flow_policy* internal/nexus/mission_* web/src/features/work/missionAutonomyModel*
git commit -m "feat: bound flow autonomy with verification gates"
```

---

### Task 7: Build Flow Run Workspace Surface

**Files:**
- Create: `web/src/features/work/FlowRunSurface.tsx`
- Create: `web/src/features/work/FlowRunStepTable.tsx`
- Create: `web/src/features/work/FlowRunActivity.tsx`
- Modify: `web/src/nexus/MissionsPage.tsx`
- Modify: `web/src/app/NexusShell.tsx`
- Modify: `web/src/app/workspace-os.css`

Surface logical key: `flow-run:<runId>`.

- [ ] **Step 1: Add model tests**
Waiting dependency, status display, Agent terminal target.

- [ ] **Step 2: Implement Flow Run**
Revision, policy, Step table, live activity, Open Agent, Pause, Take Control, Cancel.

- [ ] **Step 3: Reduce MissionsPage to compatibility list/redirect**

- [ ] **Step 4: Run and commit**

```bash
npm run web:test
npm run web:typecheck
npm run web:build
git add web/src/features/work/FlowRun* web/src/nexus/MissionsPage.tsx web/src/app
git commit -m "feat: supervise approved work through flow run"
```

## Exit gate

Approved static Flow executes dependencies correctly, supports bounded parallelism, fresh contexts, receipts, verification, restart-safe durable state and Flow Run supervision.
