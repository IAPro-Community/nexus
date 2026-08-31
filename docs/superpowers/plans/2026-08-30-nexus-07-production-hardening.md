# Nexus Production Hardening Implementation Plan

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


**Goal:** Prove the complete V0 on Linux, macOS and Windows from one SHA with browser, recovery, security and release evidence.

**Architecture:** Add evidence-producing acceptance tests and mandatory platform gates; failed behavior may not be converted to skips or exclusions.

**Tech Stack:** Go tests, Vitest, existing build/release/CI tooling.

---

### Task 1: Complete frontend regression matrix

**Files:**
- Add/modify focused tests beside `web/src/workspace/`, `web/src/features/agents/`, `web/src/features/work/`, `web/src/nexus/`.

**Required assertions:**
Tabs↔Desktop identity, close-view-no-stop, Ask-no-create, Shell-not-Agent, Context gate, API envelope, localized Flow refinement, no false completed state.

- [ ] **Step 1: Add tests**
- [ ] **Step 2: `npm run web:test`**
- [ ] **Step 3: `npm run web:typecheck`**
- [ ] **Step 4: `npm run web:build`**
- [ ] **Step 5: Commit**

```bash
git add web
git commit -m "test: cover nexus v0 workspace and flow contracts"
```

---

### Task 2: Add Go end-to-end product proof

**Files:**
- Modify: `internal/control/web/e2e_test.go`
- Create: `internal/nexus/flow_e2e_test.go`
- Reuse: `internal/testutil/`

Scenario: Project → Agent → detach/reopen → Ask → Shell → READY context → Flow A→B||C→D → receipts → verify final.

- [ ] **Step 1: Add deterministic fake-provider E2E**
- [ ] **Step 2: Run focused suite**
- [ ] **Step 3: Fix integration defects without weakening assertions**
- [ ] **Step 4: `go test ./...`**
- [ ] **Step 5: Commit**

```bash
git add internal/control/web/e2e_test.go internal/nexus/flow_e2e_test.go internal/testutil
git commit -m "test: prove nexus v0 end to end"
```

---

### Task 3: Prove browser terminal and Workspace interactions

**Files:**
- Modify: `internal/control/web/e2e_test.go`
- Modify: `web/src/nexus/terminalProtocol.test.ts`
- Add browser-harness files only under the repository's existing browser-test mechanism discovered before editing.

Evidence: ANSI unchanged, control frames hidden, presentation toggle preserves logical View, minimize/maximize/close affect View state only.

- [ ] **Step 1: Add evidence assertions**
- [ ] **Step 2: Run Linux**
- [ ] **Step 3: Run macOS**
- [ ] **Step 4: Run Windows**
- [ ] **Step 5: Commit**

```bash
git add internal/control/web web/src/nexus web/src/workspace
git commit -m "test: prove browser terminal and workspace interactions"
```

---

### Task 4: Add crash/restart no-double-dispatch proof

**Files:**
- Create: `internal/nexus/flow_recovery_test.go`
- Modify: `internal/nexus/mission_repository.go`
- Modify: `internal/nexus/mission_service.go`
- Modify: `internal/nexus/mission_executor.go`

- [ ] **Step 1: Add fake-clock/fake-runner restart test**
Completed Step not rerun; valid lease not doubled; expired lease recovers by policy; receipts survive.

- [ ] **Step 2: Run**
- [ ] **Step 3: Repair fencing/state transition defects**
- [ ] **Step 4: Stress**

```bash
go test ./internal/nexus -count=10
```

- [ ] **Step 5: Commit**

```bash
git add internal/nexus
git commit -m "fix: recover flow runs without duplicate execution"
```

---

### Task 5: Add same-SHA platform release gate

**Files:**
- Modify exact existing workflow files under `.github/workflows/` after listing them.
- Create: `scripts/validate-nexus-v0.sh`
- Create: `scripts/validate-nexus-v0.ps1`
- Modify: `Makefile` only if one shared deterministic validation target reduces duplication.

Both scripts run equivalent Go/web/integration/release smoke validation.

- [ ] **Step 1: Implement Unix validation script**
- [ ] **Step 2: Implement PowerShell validation script**
- [ ] **Step 3: Add Linux/macOS/Windows CI jobs**
- [ ] **Step 4: Windows E2E is mandatory**
- [ ] **Step 5: Preserve evidence artifacts/job output**
- [ ] **Step 6: Commit**

```bash
git add .github/workflows scripts Makefile
git commit -m "ci: gate nexus v0 on linux macos and windows"
```

---

### Task 6: Final product-language cleanup

**Files:**
- Modify visible copy under:
  - `web/src/features/work/`
  - `web/src/nexus/MissionsPage.tsx`
  - `web/src/nexus/MaestroPage.tsx`
  - localization resources under `web/src/`
  - `README.md` and user docs where needed.

Final user nouns: Project, Workspace, Agent, Terminal, Composer, Flow, Flow Run, Maestro, Resource.

- [ ] **Step 1: Search visible strings for Mission / Direct / Assisted / Planned / Nexus Intelligence / Ask Maestro**
- [ ] **Step 2: Keep internal compatibility symbols intact where required**
- [ ] **Step 3: Change only user-facing vocabulary**
- [ ] **Step 4: Run tests/typecheck/build**
- [ ] **Step 5: Commit**

```bash
git add web README.md docs
git commit -m "refactor: align nexus product language with composer and flow"
```

## Final GO gate

One commit SHA must have Go + web + Windows ConPTY/SessionHost + browser ANSI + complete Project/Agent/Shell/Composer/Flow E2E + recovery + security + release artifacts green. Otherwise verdict is explicitly `NO_GO` or `PARTIAL`.
