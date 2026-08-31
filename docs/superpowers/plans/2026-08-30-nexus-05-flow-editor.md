# Nexus Flow Editor Implementation Plan

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


**Goal:** Expose WorkPlan/WorkPackage as an editable static DAG Flow inside Composer without duplicating persistence.

**Architecture:** Use compatibility adapters and existing dependency/parallel/Agent allocation fields as graph truth; add deterministic graph model, canvas and Step inspector.

**Tech Stack:** React, TypeScript, Vitest, Go plan/plangraph.

---

### Task 1: Add Flow compatibility façade

**Files:**
- Create: `web/src/features/work/flow/flowAdapter.ts`
- Create: `web/src/features/work/flow/flowAdapter.test.ts`
- Create: `internal/nexus/flow.go`
- Create: `internal/nexus/flow_test.go`
- Reuse: `internal/nexus/plan.go`

```ts
export interface FlowDefinition {
  id: string
  title: string
  revision: number
  steps: FlowStep[]
}

export interface FlowStep {
  id: string
  title: string
  goal: string
  dependencies: string[]
  parallelGroup?: string
  agentAssignment?: string
  acceptanceCriteria: string[]
}
```

- [ ] **Step 1: Round-trip tests**
Preserve IDs, order, deps, parallel group, agent allocation, provider lock, acceptance criteria.

- [ ] **Step 2: Go façade test**
Existing WorkPlan returns Flow JSON without DB migration.

- [ ] **Step 3: Implement adapters only**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus -run TestFlow -v
npm --prefix web test -- flowAdapter.test.ts
git add internal/nexus/flow* web/src/features/work/flow
git commit -m "feat: expose work plans as flow definitions"
```

---

### Task 2: Add deterministic DAG model

**Files:**
- Create: `web/src/features/work/flow/flowGraphModel.ts`
- Create: `web/src/features/work/flow/flowGraphModel.test.ts`
- Modify/add tests under: `internal/nexus/plangraph/`

Functions: `validateFlowDAG`, `topologicalOrder`, `executionWaves`, `dependentsOf`.

- [ ] **Step 1: Test sequence, parallel, convergence, missing dependency and cycle**

```ts
expect(executionWaves(flow)).toEqual([
  ['context'],
  ['frontend', 'tests'],
  ['verify'],
])
```

- [ ] **Step 2: Implement deterministic graph algorithms**

- [ ] **Step 3: Enforce backend DAG validity**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus/plangraph
npm --prefix web test -- flowGraphModel.test.ts
git add internal/nexus/plangraph web/src/features/work/flow
git commit -m "feat: validate and schedule static flow dag"
```

---

### Task 3: Build Flow Canvas inside Composer

**Files:**
- Create: `web/src/features/work/flow/FlowCanvas.tsx`
- Create: `web/src/features/work/flow/FlowStepNode.tsx`
- Create: `web/src/features/work/flow/flowLayout.ts`
- Create: `web/src/features/work/flow/flowLayout.test.ts`
- Create: `web/src/features/work/flow/flow-canvas.css`
- Modify: `web/src/features/work/ComposerSurface.tsx`

- [ ] **Step 1: Add deterministic layout tests**

- [ ] **Step 2: Implement nodes/arrows/parallel groups/selection/fit/zoom**

- [ ] **Step 3: Use current Nexus CSS variables**

- [ ] **Step 4: Run and commit**

```bash
npm run web:test
npm run web:typecheck
npm run web:build
git add web/src/features/work/flow web/src/features/work/ComposerSurface.tsx
git commit -m "feat: render approved flow dag in composer"
```

---

### Task 4: Add Step inspector and localized refinement

**Files:**
- Create: `web/src/features/work/flow/FlowStepInspector.tsx`
- Create: `web/src/features/work/flow/stepRefinementModel.ts`
- Create: `web/src/features/work/flow/stepRefinementModel.test.ts`
- Modify: `web/src/features/work/ComposerSurface.tsx`
- Modify: `internal/nexus/intelligence/context_envelope.go`
- Modify concrete refinement code under `internal/nexus/intelligence/`

- [ ] **Step 1: Add scope-isolation test**
Refining frontend leaves unrelated Step hashes unchanged.

- [ ] **Step 2: Implement inspector**
Goal, executor, context, dependencies, skills, acceptance, verify, prompt preview.

- [ ] **Step 3: Reuse existing split/merge/revision mutation logic**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus/intelligence
npm run web:test
npm run web:typecheck
git add web/src/features/work/flow internal/nexus/intelligence
git commit -m "feat: refine individual flow steps contextually"
```

---

### Task 5: Add EXISTING / CREATE / AUTO assignment

**Files:**
- Create: `web/src/features/work/flow/agentAssignment.ts`
- Create: `web/src/features/work/flow/agentAssignment.test.ts`
- Modify: `web/src/features/work/flow/FlowStepInspector.tsx`
- Modify: `internal/nexus/plan.go`
- Modify: `internal/nexus/mission_execution.go`

```ts
type AgentAssignment =
  | { strategy: 'EXISTING'; agentId: string }
  | { strategy: 'CREATE'; role: string }
  | { strategy: 'AUTO'; role?: string }
```

- [ ] **Step 1: Test validation/serialization**

- [ ] **Step 2: Map reversibly to current WorkPackage fields**

- [ ] **Step 3: Show assignment in pre-run review**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus
npm run web:test
git add web/src/features/work/flow internal/nexus/plan.go internal/nexus/mission_execution.go
git commit -m "feat: assign flow steps to existing new or automatic agents"
```

## Exit gate

A Flow is a persisted, revisioned static DAG inside Composer, supports localized refinement and assignment, and still executes nothing while DRAFT.
