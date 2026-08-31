# Nexus Maestro and Project Context Implementation Plan

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


**Goal:** Correct Maestro integration and add reproducible Project Context Readiness without blocking Direct work.

**Architecture:** Maestro remains independent; Nexus detects/indexes/invokes it and stores readiness/fingerprint metadata only. Semantic memory remains AGENTS.md + DEV/.

**Tech Stack:** Go, React, TypeScript, existing Maestro integration.

---

### Task 1: Correct Maestro detection and status

**Files:**
- Modify: `internal/nexus/maestro.go`
- Modify: `internal/nexus/maestro_test.go`
- Modify: `web/src/nexus/MaestroPage.tsx`
- Modify: `web/src/nexus/api.ts`
- Modify: `internal/control/web/handlers_nexus.go`

**Interface:**

```go
type MaestroStatus struct {
    Installed bool
    Version string
    Root string
    SkillsManifest string
    ProjectMemoryReady bool
}
```

- [ ] **Step 1: Add fixture detection tests**
Real-style install, missing install, missing manifest, legacy binary-only case.

- [ ] **Step 2: Run**

```bash
go test ./internal/nexus -run TestMaestro -v
```

- [ ] **Step 3: Implement installation/layout detection**
Command advice remains optional legacy behavior.

- [ ] **Step 4: Update Maestro UI**
Installed/version/verify/update/skills/context.

- [ ] **Step 5: Run and commit**

```bash
go test ./internal/nexus ./internal/control/web
npm run web:typecheck
npm run web:test
git add internal/nexus/maestro* internal/control/web/handlers_nexus.go web/src/nexus/MaestroPage.tsx web/src/nexus/api.ts
git commit -m "refactor: manage Maestro as an independent integration"
```

---

### Task 2: Add real Maestro skills catalog

**Files:**
- Modify: `internal/nexus/maestro.go`
- Create: `internal/nexus/maestro_skills_test.go`
- Modify: `web/src/nexus/MaestroPage.tsx`
- Create: `web/src/nexus/maestroSkillsModel.ts`
- Create: `web/src/nexus/maestroSkillsModel.test.ts`

```go
type MaestroSkill struct {
    ID string
    Name string
    Description string
    SourcePath string
}
```

- [ ] **Step 1: Test manifest/folder discovery**
Malformed entries are rejected; no synthesized skills.

- [ ] **Step 2: Implement catalog**

- [ ] **Step 3: Implement frontend search/filter; selection returns references only**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus -run 'TestMaestro.*Skill' -v
npm --prefix web test -- maestroSkillsModel.test.ts
git add internal/nexus web/src/nexus
git commit -m "feat: expose real Maestro skills catalog"
```

---

### Task 3: Add Project Context Readiness

**Files:**
- Create: `internal/nexus/context_readiness.go`
- Create: `internal/nexus/context_readiness_test.go`
- Create: `internal/control/web/handlers_context.go`
- Create: `internal/control/web/handlers_context_test.go`
- Create: `web/src/nexus/contextReadiness.ts`
- Create: `web/src/nexus/contextReadiness.test.ts`
- Modify: `web/src/nexus/api.ts`

```go
type ContextState string

const (
    ContextMissing ContextState = "MISSING"
    ContextHydrating ContextState = "HYDRATING"
    ContextReady ContextState = "READY"
    ContextStale ContextState = "STALE"
    ContextFailed ContextState = "FAILED"
)

type ContextFingerprint struct {
    ProjectID string
    Branch string
    Head string
    DirtyFingerprint string
    MaestroVersion string
}
```

- [ ] **Step 1: Add state tests**
No memory→MISSING; matching fingerprint→READY; changed branch/HEAD→STALE; running hydration→HYDRATING; failure→FAILED.

- [ ] **Step 2: Implement deterministic fingerprint**

- [ ] **Step 3: Implement readiness/hydration handlers**

- [ ] **Step 4: Frontend model**
`canCompose` only for READY; `canDirectWork` always true.

- [ ] **Step 5: Run and commit**

```bash
go test ./internal/nexus ./internal/control/web
npm --prefix web test -- contextReadiness.test.ts
git add internal/nexus/context_readiness* internal/control/web/handlers_context* web/src/nexus
git commit -m "feat: gate composition on project context readiness"
```

---

### Task 4: Define Maestro-owned hydration adapter

**Files:**
- Create: `docs/maestro/NEXUS_HYDRATION_CONTRACT.md`
- Modify: `internal/nexus/maestro.go`
- Create: `internal/nexus/maestro_hydration_test.go`

**Expected machine-readable result:**

```json
{
  "status": "READY",
  "project_path": "/repo",
  "head": "abc123",
  "updated_files": ["DEV/CONTEXT.md"],
  "warnings": []
}
```

- [ ] **Step 1: Fake-runner adapter test**
Nexus passes canonical path, parses result, maps nonzero exit to FAILED, and never writes semantic DEV content itself.

- [ ] **Step 2: Implement process adapter behind an interface**

- [ ] **Step 3: Document companion Maestro implementation requirement**

- [ ] **Step 4: Run and commit**

```bash
go test ./internal/nexus -run 'TestMaestro.*Hydrat' -v
git add internal/nexus docs/maestro
git commit -m "feat: define Maestro project hydration contract"
```

## Exit gate

Nexus detects Maestro, lists real skills, requests Maestro-owned hydration, reaches READY reproducibly, marks drift STALE, and never blocks Direct work.
