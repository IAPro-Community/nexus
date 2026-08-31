# IAPro Nexus V0 Master Implementation Plan

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


**Goal:** Refactor the existing IAPro Nexus into the approved Project-centric Workspace OS with persistent Agents, Project Shells, Composer, static DAG Flows, dependency-aware Flow Run, clean context handoffs and verified completion.

**Architecture:** Keep the current React Workspace V2 and Go Core as the foundation. Add desktop-window presentation over existing logical Views, evolve Work/Plan/Mission through compatibility façades, and add Context Capsule + Work Receipt boundaries without replacing proven runtime infrastructure.

**Tech Stack:** Go, React 19, TypeScript 5.9, Vitest, xterm.js, existing Nexus HTTP/WebSocket control plane, SQLite/store layer, Maestro integration.

---

## Plan suite

Execute these plans strictly in order. A later plan may begin only when the prior plan's exit gate passes.

1. `docs/superpowers/plans/2026-08-30-nexus-01-core-integrity.md`
2. `docs/superpowers/plans/2026-08-30-nexus-02-workspace-os-direct-work.md`
3. `docs/superpowers/plans/2026-08-30-nexus-03-maestro-context.md`
4. `docs/superpowers/plans/2026-08-30-nexus-04-composer.md`
5. `docs/superpowers/plans/2026-08-30-nexus-05-flow-editor.md`
6. `docs/superpowers/plans/2026-08-30-nexus-06-flow-run.md`
7. `docs/superpowers/plans/2026-08-30-nexus-07-production-hardening.md`

## Dependency graph

```text
01 Core Integrity
       ↓
02 Workspace OS + Direct Work
       ↓
03 Maestro + Context
       ↓
04 Composer
       ↓
05 Flow Editor
       ↓
06 Flow Run
       ↓
07 Production Hardening
```

## Repository map used by the plan

### Existing frontend anchors
- `web/src/app/NexusShell.tsx`
- `web/src/app/workspace-os.css`
- `web/src/workspace/model.ts`
- `web/src/workspace/state.ts`
- `web/src/workspace/WorkspaceProvider.tsx`
- `web/src/workspace/WorkspaceRenderer.tsx`
- `web/src/workspace/WorkspaceTaskbar.tsx`
- `web/src/features/agents/AgentsSurface.tsx`
- `web/src/features/agents/AgentConfigurationSurface.tsx`
- `web/src/features/work/DirectSessionLauncher.tsx`
- `web/src/features/work/WorkSurface.tsx`
- `web/src/features/work/PlanBuilderSurface.tsx`
- `web/src/features/work/planBuilderModel.ts`
- `web/src/nexus/AgentTerminal.tsx`
- `web/src/nexus/MaestroPage.tsx`
- `web/src/nexus/MissionsPage.tsx`
- `web/src/nexus/ResourcePicker.tsx`
- `web/src/nexus/api.ts`
- `web/src/nexus/terminalProtocol.ts`

### Existing backend anchors
- `internal/control/host/host.go`
- `internal/control/host/fanout.go`
- `internal/control/host/framing_test.go`
- `internal/control/terminal/terminal.go`
- `internal/control/terminal/terminal_unix.go`
- `internal/control/terminal/terminal_windows.go`
- `internal/control/web/broker.go`
- `internal/control/web/handler_terminal.go`
- `internal/control/web/handlers_api.go`
- `internal/control/web/handlers_fs.go`
- `internal/control/web/handlers_nexus.go`
- `internal/nexus/nexus.go`
- `internal/nexus/config.go`
- `internal/nexus/continuity.go`
- `internal/nexus/resource_discovery.go`
- `internal/nexus/resource_recommendation.go`
- `internal/nexus/scheduler.go`
- `internal/nexus/worktree.go`
- `internal/nexus/maestro.go`
- `internal/nexus/intelligence_provider.go`
- `internal/nexus/intelligence/`
- `internal/nexus/plan.go`
- `internal/nexus/plangraph/`
- `internal/nexus/mission_execution.go`
- `internal/nexus/mission_executor.go`
- `internal/nexus/mission_repository.go`
- `internal/nexus/mission_service.go`
- `internal/nexus/runner/`

## Global verification commands

```bash
go test ./...
npm run web:typecheck
npm run web:test
npm run web:build
```

## Final product proof

The suite is complete only when one commit SHA demonstrates:

```text
Project opens Workspace
→ Tabs ↔ Desktop without identity duplication
→ existing Agent Open/Ask
→ normal Project Shell
→ Composer Context READY
→ complex goal → Flow DAG
→ localized Step refinement
→ EXISTING/CREATE/AUTO assignment
→ explicit Review & Run
→ dependency-aware parallel execution
→ fresh Context Capsule per task
→ Work Receipt handoff
→ verification/remediation
→ COMPLETED_VERIFIED
```
