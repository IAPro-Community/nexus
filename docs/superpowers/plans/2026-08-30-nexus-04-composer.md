# Nexus Composer Implementation Plan

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


**Goal:** Evolve Work/Intelligence into a single Composer Surface that handles simple prompts or Flow creation from READY context via CLI or API.

**Architecture:** Reuse intelligence/clarification machinery, introduce bounded Composer requests and API Context Envelope, and deprecate the three-mode Direct/Assisted/Planned UX.

**Tech Stack:** React, TypeScript, Vitest, Go intelligence provider layer.

---

### Task 1: Add Composer request and Context Envelope

**Files:**
- Create: `web/src/features/work/composerModel.ts`
- Create: `web/src/features/work/composerModel.test.ts`
- Create: `internal/nexus/intelligence/context_envelope.go`
- Create: `internal/nexus/intelligence/context_envelope_test.go`
- Modify: `internal/nexus/intelligence_provider.go`

```ts
export type ComposerScope = 'PRODUCT' | 'FLOW' | 'STEP'

export interface ComposerRequest {
  projectId: string
  scope: ComposerScope
  instruction: string
  flowId?: string
  stepId?: string
  operation?: 'REFINE' | 'EXPAND' | 'COMPARE' | 'SPLIT'
}
```

- [ ] **Step 1: Add envelope tests**
STALE rejected; branch/HEAD present; unrelated content excluded; STEP scope includes selected step.

- [ ] **Step 2: Implement bounded builder**

- [ ] **Step 3: Add frontend request validation**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus/intelligence
npm --prefix web test -- composerModel.test.ts
git add internal/nexus/intelligence internal/nexus/intelligence_provider.go web/src/features/work/composerModel*
git commit -m "feat: define composer context and request contracts"
```

---

### Task 2: Evolve WorkSurface into ComposerSurface

**Files:**
- Create: `web/src/features/work/ComposerSurface.tsx`
- Create: `web/src/features/work/ComposerContextBar.tsx`
- Create: `web/src/features/work/ComposerConversation.tsx`
- Modify: `web/src/features/work/WorkSurface.tsx`
- Modify: `web/src/app/NexusShell.tsx`
- Modify: `web/src/app/workspace-os.css`

- [ ] **Step 1: Add model test**
Final user modes do not expose `direct|assisted|planned`.

- [ ] **Step 2: Implement Composer**
Context bar, CLI/API selection, conversation, context chips, Flow area.

- [ ] **Step 3: Keep WorkSurface as temporary compatibility wrapper**

- [ ] **Step 4: Register stable `composer` logical View**

- [ ] **Step 5: Run and commit**

```bash
npm run web:typecheck
npm run web:test
npm run web:build
git add web/src/features/work web/src/app
git commit -m "refactor: evolve work surface into composer"
```

---

### Task 3: Dispatch CLI and API intelligence honestly

**Files:**
- Modify: `internal/nexus/intelligence_provider.go`
- Modify concrete files under: `internal/nexus/intelligence/`
- Modify: `web/src/nexus/api.ts`
- Modify: `web/src/features/work/ComposerSurface.tsx`
- Create: `internal/nexus/intelligence/composer_test.go`

- [ ] **Step 1: Test dispatch**
CLI includes Project path; API includes Context Envelope; missing READY rejects API/plan; no silent semantic fallback.

- [ ] **Step 2: Reuse existing Analyze/GeneratePlan/ResolveClarification/CompilePrompt**

- [ ] **Step 3: Surface local-vs-envelope truth in UI**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus/intelligence ./internal/nexus
npm run web:test
git add internal/nexus/intelligence* web/src/features/work/ComposerSurface.tsx web/src/nexus/api.ts
git commit -m "feat: run composer with cli or bounded api context"
```

---

### Task 4: Support simple compiled-prompt result

**Files:**
- Create: `web/src/features/work/CompiledPromptCard.tsx`
- Modify: `web/src/features/work/ComposerSurface.tsx`
- Reuse: Ask Agent API.

Actions: existing Agent, new Agent, copy, turn into Flow.

- [ ] **Step 1: Test no implicit Flow creation**

- [ ] **Step 2: Implement card**

- [ ] **Step 3: Wire Send to existing Agent**

- [ ] **Step 4: Run and commit**

```bash
npm run web:test
npm run web:typecheck
git add web/src/features/work
git commit -m "feat: let composer finish as a direct compiled prompt"
```

## Exit gate

Composer is a Workspace View, requires READY context, supports CLI/API, exposes API Context Envelope, preserves clarification, and can finish without Flow.
