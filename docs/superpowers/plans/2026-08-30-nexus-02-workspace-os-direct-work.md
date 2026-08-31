# Nexus Workspace OS and Direct Work Implementation Plan

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


**Goal:** Make each Project a real multi-view Workspace with Tabs/Desktop, persistent Agents, Ask Agent and normal Project Shells.

**Architecture:** Extend Workspace V2 with presentation state rather than replacing its tree; Desktop windows reference the same logical Views as tabs and splits.

**Tech Stack:** React 19, TypeScript, Vitest, xterm.js, Go terminal/control packages.

---

### Task 1: Add Workspace presentation state

**Files:**
- Create: `web/src/workspace/presentation.ts`
- Create: `web/src/workspace/presentation.test.ts`
- Modify: `web/src/workspace/WorkspaceProvider.tsx`
- Modify: `web/src/workspace/state.ts`
- Modify: `web/src/workspace/state.test.ts`

**Interfaces:**

```ts
export type WorkspacePresentationMode = 'TABS' | 'DESKTOP'

export interface DesktopWindowState {
  viewId: string
  x: number
  y: number
  width: number
  height: number
  zIndex: number
  minimized: boolean
  maximized: boolean
}

export interface WorkspacePresentationState {
  mode: WorkspacePresentationMode
  windows: Record<string, DesktopWindowState>
}
```

- [ ] **Step 1: Write tests**
Switching TABS→DESKTOP→TABS preserves `viewId`, `logicalKey`, Workspace tree and Agent identity.

- [ ] **Step 2: Run**

```bash
npm --prefix web test -- presentation.test.ts state.test.ts model.test.ts
```

- [ ] **Step 3: Implement pure reducers**
`setPresentationMode`, `ensureDesktopWindow`, `moveDesktopWindow`, `resizeDesktopWindow`, `minimizeDesktopWindow`, `toggleDesktopMaximize`, `focusDesktopWindow`.

- [ ] **Step 4: Persist presentation with backward-compatible hydration**

- [ ] **Step 5: Run and commit**

```bash
npm run web:typecheck
npm run web:test
git add web/src/workspace
git commit -m "feat: add workspace desktop presentation state"
```

---

### Task 2: Render Desktop windows over existing Surfaces

**Files:**
- Create: `web/src/workspace/DesktopWorkspace.tsx`
- Create: `web/src/workspace/DesktopWindow.tsx`
- Modify: `web/src/workspace/WorkspaceRenderer.tsx`
- Modify: `web/src/app/NexusShell.tsx`
- Modify: `web/src/app/workspace-os.css`

- [ ] **Step 1: Add behavior tests to presentation model**
One window per `viewId`; close affects View only; maximize does not mutate Workspace tree.

- [ ] **Step 2: Implement window chrome**
Move, resize, focus, minimize, maximize, close; use current Nexus tokens.

- [ ] **Step 3: Reuse existing Surface renderer**
No duplicate AgentTerminal/Composer implementation.

- [ ] **Step 4: Add `Abas | Janelas` switch**

- [ ] **Step 5: Run and commit**

```bash
npm run web:typecheck
npm run web:test
npm run web:build
git add web/src/workspace web/src/app
git commit -m "feat: render workspace surfaces as desktop windows"
```

---

### Task 3: Add Workspace launcher

**Files:**
- Create: `web/src/workspace/WorkspaceLauncher.tsx`
- Create: `web/src/workspace/launcherModel.ts`
- Create: `web/src/workspace/launcherModel.test.ts`
- Modify: `web/src/app/NexusShell.tsx`

**Actions:** `NEW_AGENT`, `NEW_TERMINAL`, `OPEN_COMPOSER`, `OPEN_ACTIVE_FLOW_RUN`.

- [ ] **Step 1: Test exact action mapping**
No Mission/Assisted/Planned items.

- [ ] **Step 2: Implement launcher and keyboard behavior**

- [ ] **Step 3: Wire to stable logical keys**

- [ ] **Step 4: Run and commit**

```bash
npm run web:test
npm run web:typecheck
git add web/src/workspace web/src/app/NexusShell.tsx
git commit -m "feat: add workspace launcher"
```

---

### Task 4: Add Ask existing Agent

**Files:**
- Create: `web/src/features/agents/AskAgentDialog.tsx`
- Create: `web/src/features/agents/askAgentModel.ts`
- Create: `web/src/features/agents/askAgentModel.test.ts`
- Modify: `web/src/features/agents/AgentsSurface.tsx`
- Modify: `web/src/features/work/DirectSessionLauncher.tsx`
- Modify: `web/src/features/work/WorkSurface.tsx`
- Modify: `web/src/nexus/api.ts`
- Modify: `internal/control/web/handlers_nexus.go`
- Modify: `internal/nexus/nexus.go`
- Test: `internal/nexus/nexus_test.go`

**API concept:**

```ts
askAgent(agentId: string, instruction: string): Promise<Agent>
```

- [ ] **Step 1: Add backend test**
Ask leaves Agent count unchanged and returns same AgentID.

- [ ] **Step 2: Add frontend state test**
Working→ASK; stopped→START_AND_ASK; never CREATE_AGENT.

- [ ] **Step 3: Implement backend lifecycle/submission path**

- [ ] **Step 4: Implement Agent card Open/Ask/Configure/Stop**

- [ ] **Step 5: Change Direct launcher**
Primary choice is Existing Agent or New Agent.

- [ ] **Step 6: Run and commit**

```bash
go test ./internal/nexus ./internal/control/web
npm run web:test
npm run web:typecheck
git add web/src/features/agents web/src/features/work web/src/nexus/api.ts internal/nexus/nexus.go internal/control/web/handlers_nexus.go
git commit -m "feat: work directly with persistent project agents"
```

---

### Task 5: Add normal Project Shell

**Files:**
- Create: `web/src/nexus/ProjectShell.tsx`
- Create: `web/src/nexus/projectShellModel.ts`
- Create: `web/src/nexus/projectShellModel.test.ts`
- Modify: `web/src/nexus/api.ts`
- Create: `internal/control/web/handler_shell.go`
- Create: `internal/control/web/handler_shell_test.go`
- Reuse: `internal/control/terminal/terminal.go`
- Reuse: `internal/control/terminal/terminal_unix.go`
- Reuse: `internal/control/terminal/terminal_windows.go`
- Modify: `web/src/app/NexusShell.tsx`

**Request:**

```ts
interface CreateProjectShellRequest {
  project_id: string
  name: string
}
```

Server resolves canonical cwd.

- [ ] **Step 1: Add handler test**
Shell cwd equals Project canonical path.

- [ ] **Step 2: Add frontend model test**
Surface type is `project-shell`, never `agent-terminal`.

- [ ] **Step 3: Implement shell PTY session**

- [ ] **Step 4: Implement xterm Surface and launcher action**

- [ ] **Step 5: Run and commit**

```bash
go test ./internal/control/web ./internal/control/terminal
npm run web:test
npm run web:typecheck
npm run web:build
git add web/src/nexus web/src/app/NexusShell.tsx internal/control/web/handler_shell*
git commit -m "feat: add normal project shell surfaces"
```

## Exit gate

Project Workspace supports Tabs/Desktop, live Agents, normal Shell, Ask existing Agent, and presentation switching without identity duplication.
