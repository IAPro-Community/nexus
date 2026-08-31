# Nexus Core Integrity Implementation Plan

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


**Goal:** Make runtime, terminal protocol, Windows PTY/host behavior, filesystem contract, resource recommendation and security trustworthy before new surfaces.

**Architecture:** Repair Core in place; use explicit protocol contracts, platform-specific Windows tests, recommendation-backed Direct selection and deny-by-default config exposure.

**Tech Stack:** Go, Windows ConPTY, HTTP/WebSocket, xterm protocol, Vitest.

---

### Task 1: Make terminal protocol exhaustive and non-leaking

**Files:**
- Modify: `web/src/nexus/terminalProtocol.ts`
- Modify: `web/src/nexus/terminalProtocol.test.ts`
- Modify: `web/src/nexus/AgentTerminal.tsx`
- Modify: `internal/control/web/handler_terminal.go`
- Modify: `internal/control/web/broker.go`
- Test: `internal/control/host/framing_test.go`

**Interfaces:**
- Consumes: current WebSocket terminal frames.
- Produces: `parseTerminalFrame(raw: string): TerminalFrame` that recognizes output/control/agent-state/continuity-state frames without writing control JSON to xterm.

- [ ] **Step 1: Write failing frontend protocol tests**

```ts
it('parses agent_state without terminal output', () => {
  expect(parseTerminalFrame(JSON.stringify({
    type: 'agent_state',
    agent_id: 'a1',
    state: 'WORKING',
  }))).toEqual(expect.objectContaining({ type: 'agent_state' }))
})

it('keeps ANSI payload byte-for-byte', () => {
  const ansi = '\u001b[32mPASS\u001b[0m\r\n'
  expect(parseTerminalFrame(JSON.stringify({ type: 'output', data: ansi })))
    .toEqual({ type: 'output', data: ansi })
})
```

- [ ] **Step 2: Run focused tests and confirm failure**

```bash
npm --prefix web test -- terminalProtocol.test.ts
```

- [ ] **Step 3: Make the TypeScript frame union exhaustive**

```ts
export type TerminalFrame =
  | { type: 'output'; data: string }
  | { type: 'agent_state'; agent_id: string; state: string }
  | { type: 'continuity_state'; agent_id: string; continuity_state: string }
  | { type: 'control'; event: string; payload?: unknown }
  | { type: 'unknown'; raw: string }
```

Only `output.data` may be written to xterm.

- [ ] **Step 4: Harden backend framing**

Emit typed envelopes for state/control. Raw PTY bytes become only `output`.

- [ ] **Step 5: Add Go framing tests**

Prove JSON-looking PTY output survives when typed as output, while real control frames never enter the output channel.

- [ ] **Step 6: Run**

```bash
go test ./internal/control/host ./internal/control/web
npm --prefix web test -- terminalProtocol.test.ts agentTerminalModel.test.ts
npm run web:typecheck
```

- [ ] **Step 7: Commit**

```bash
git add web/src/nexus internal/control/host internal/control/web
git commit -m "fix: make terminal protocol explicit and lossless"
```

---

### Task 2: Fix Windows terminal and host lifecycle

**Files:**
- Modify: `internal/control/terminal/terminal_windows.go`
- Modify: `internal/control/terminal/terminal_windows_test.go`
- Modify: `internal/control/host/host.go`
- Modify: `internal/control/host/proc_windows.go`
- Modify: `internal/control/host/host_test.go`
- Modify: `internal/control/host/qa_test.go`

**Interfaces:**
- Produces deterministic Windows interactive echo, resize, close, process cleanup and host handshake behavior.

- [ ] **Step 1: Add Windows-only tests**
Start `cmd.exe` or PowerShell, write `echo NEXUS_CONPTY_OK`, read the sentinel, close terminal, and assert process/handles exit cleanly.

- [ ] **Step 2: Add host handshake test**
Publish ready only when the pipe accepts a real request/response.

- [ ] **Step 3: Reproduce on Windows**

```powershell
go test ./internal/control/terminal -run 'Test.*Windows|Test.*ConPTY' -v
go test ./internal/control/host -run 'Test.*Host|Test.*Handshake' -v
```

- [ ] **Step 4: Repair ConPTY ownership/order**
Make Close idempotent, terminate reader/writer goroutines, release handles once, and prevent wait/close races.

- [ ] **Step 5: Repair named-pipe readiness lifecycle**

- [ ] **Step 6: Stress focused tests**

```powershell
go test ./internal/control/terminal ./internal/control/host -count=10
```

- [ ] **Step 7: Commit**

```bash
git add internal/control/terminal internal/control/host
git commit -m "fix: stabilize Windows terminal and session host lifecycle"
```

---

### Task 3: Fix filesystem mkdir contract and DB cleanup

**Files:**
- Modify: `internal/control/web/handlers_fs.go`
- Modify: `internal/control/web/handlers_fs_test.go`
- Modify: `internal/control/web/e2e_test.go`
- Modify: concrete store shutdown file under `internal/nexus/store/` identified by current constructor/Close ownership.

- [ ] **Step 1: Add failing mkdir request test**
Use the client request shape and assert `201 Created`.

- [ ] **Step 2: Reproduce**

```bash
go test ./internal/control/web -run TestFSMkdir -v
```

- [ ] **Step 3: Normalize typed request decoding**
Use one canonical field name across client/server.

- [ ] **Step 4: Add shutdown test**
Close server/brokers/host/store before deleting `nexus.db`. File-delete retries are not the correctness mechanism.

- [ ] **Step 5: Run**

```bash
go test ./internal/control/web -count=3
```

- [ ] **Step 6: Commit**

```bash
git add internal/control/web internal/nexus/store
git commit -m "fix: align filesystem contract and release runtime state cleanly"
```

---

### Task 4: Make recommendation the default Direct resource path

**Files:**
- Modify: `internal/nexus/resource_recommendation.go`
- Modify: `internal/nexus/resource_recommendation_test.go`
- Modify: `internal/nexus/scheduler.go`
- Modify: `internal/nexus/scheduler_test.go`
- Modify: `web/src/features/work/directSessionModel.ts`
- Modify: `web/src/features/work/directSessionModel.test.ts`
- Modify: `web/src/features/work/DirectSessionLauncher.tsx`
- Modify: `web/src/nexus/ResourcePicker.tsx`

- [ ] **Step 1: Add failing frontend model test**

```ts
it('defaults new direct work to recommendation', () => {
  expect(defaultDirectResourceSelection()).toEqual({ mode: 'AUTO' })
})
```

- [ ] **Step 2: Add backend tests**
Unhealthy/cooldown resources are excluded; equal inputs produce deterministic selection.

- [ ] **Step 3: Run focused tests**

```bash
go test ./internal/nexus -run 'Test.*Recommend|Test.*Scheduler' -v
npm --prefix web test -- directSessionModel.test.ts
```

- [ ] **Step 4: Route launcher through recommendation**
Persist `MANUAL` only after explicit provider/profile selection.

- [ ] **Step 5: Run and commit**

```bash
go test ./internal/nexus
npm run web:test
git add internal/nexus/resource_recommendation* internal/nexus/scheduler* web/src/features/work web/src/nexus/ResourcePicker.tsx
git commit -m "fix: use resource recommendation as the direct-work default"
```

---

### Task 5: Replace broad host-config sharing with allowlisting

**Files:**
- Modify: `internal/nexus/config.go`
- Modify: `internal/nexus/config_test.go`
- Modify concrete files under `internal/control/launchenv/` after listing that package before the edit.

- [ ] **Step 1: Add failing security test**
Create allowed provider config plus unrelated `~/.config/example-private/secret.txt`; generated launch env must expose only allowlisted paths.

- [ ] **Step 2: Run**

```bash
go test ./internal/nexus -run 'Test.*Config' -v
```

- [ ] **Step 3: Implement provider-specific allowlists**

- [ ] **Step 4: Add deny-by-default assertion**
Unknown provider receives no host-config exposure.

- [ ] **Step 5: Run and commit**

```bash
go test ./...
git add internal/nexus/config.go internal/nexus/config_test.go internal/control/launchenv
git commit -m "security: allowlist provider configuration exposure"
```

## Exit gate

Do not start Plan 02 until Windows terminal/host, terminal protocol, filesystem E2E, resource AUTO recommendation and config security tests are green.
