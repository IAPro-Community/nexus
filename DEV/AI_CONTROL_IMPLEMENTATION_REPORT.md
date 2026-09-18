# AI Control Implementation Report

**Date:** 2026-08-27
**Repository:** `github.com/IAPro-Community/nexus`
**Continuation Branch:** `feat/ai-control-runtime`
**Base Branch:** `feat/control-plane-evolution` (SHA: `ced1e31191544a49fe05634d0b134d1b848c783e`)

> **SUPERSEDED NOTICE (2026-08-28):**
> This report reflects the preliminary runtime milestone and has been audited and superseded by the Truth Audit and Hardening Phase report: [`DEV/AI_CONTROL_HARDENING_REPORT.md`](DEV/AI_CONTROL_HARDENING_REPORT.md).

## 1. Executive Verdict: SUPERSEDED (See Hardening Report)

The AI Control Runtime and Universal Agent Management layer has been successfully implemented on top of the latest development baseline without modifying or regressing the classic execution plane.

All 40+ unit and integration tests pass with zero data races (`go test -race ./...`).

---

## 2. Baseline & Lineage

- **Base Branch:** `feat/control-plane-evolution`
- **Base Commit SHA:** `ced1e31191544a49fe05634d0b134d1b848c783e`
- **Merge Base with Main:** `ced1e31191544a49fe05634d0b134d1b848c783e`
- **Active Working Branch:** `feat/ai-control-runtime`
- **Working Tree:** CLEAN

---

## 3. What Was Preserved (Classic Mode)

- **Classic Direct Invocation:** `ai <provider>` (e.g. `ai codex`, `ai agy`, `ai claude`, `ai opencode`, `ai gemini`) continues to run directly via provider adapters without background SessionHost overhead.
- **Smart Account Selector:** Multi-factor scoring, workspace bindings (`ai bind`), priorities, and `ai explain <provider>`.
- **Honest Quota Engine:** `LIVE`, `CACHED`, `UNKNOWN`, `RATE_LIMITED`, `UNSUPPORTED` statuses.
- **429 Cooldown & Automatic Fallback:** Loop-safe failover.
- **Sandbox Isolation:** D-Bus keyring daemons, per-profile HOME directories, `0600` permissions.
- **Installation Scripts:** 1-line zero-clone installers (`install.sh`, `install.ps1`).

---

## 4. What Was Implemented (AI Control Runtime)

### A. IPC & Control Protocol (`internal/control/protocol`)
- Versioned JSON protocol supporting `ping`, `status`, `attach`, `detach`, `resize`, `input`, `stop`, `terminate`, `handoff`, `continue`, `events`.
- Cross-platform local endpoints: Unix Domain Sockets on Linux/macOS (`/tmp/ai-control-<uid>/<id>.sock`) and Named Pipes on Windows (`\\.\pipe\ai-control-<id>`).
- RPC client (`protocol.Client`) for synchronous inspection and interactive I/O streaming.

### B. Runtime Registry (`internal/control/registry`)
- Centralized tracking of managed runtime sessions with atomic persistence to disk.
- Lifecycle states: `STARTING`, `RUNNING`, `WAITING`, `APPROVAL`, `DETACHED`, `HANDOFF`, `STOPPING`, `STOPPED`, `FAILED`, `STALE`.
- **Zero Secrets in State:** Authentication tokens and private keys are never stored.
- Automatic stale detection and cleanup (`ai control cleanup`).

### C. SessionHost & PTY Engine (`internal/control/host`)
- One lightweight SessionHost per supervised runtime process.
- PTY allocation via `creack/pty` on Linux/macOS with standard pipe fallbacks.
- Bounded 128 KB ring buffer for terminal history reattachment.
- Multi-observer broadcast with single-writer terminal input.

### D. Universal Slash Command Channel (`/ai`)
- In-session command interception before forwarding input to the AI model:
  - `/ai status`: Live runtime status, PID, session ID, and quota.
  - `/ai accounts`: Configured accounts and remaining quotas for active provider.
  - `/ai usage`: Point-in-time usage metrics.
  - `/ai handoff <profile>`: Same-provider account handoff.
  - `/ai continue <provider>`: Cross-provider context handoff.
  - `/ai detach`: Disconnect from terminal without killing process.
  - `/ai stop`: Graceful shutdown.
  - `//ai <text>`: Escape prefix to send literal `/ai` commands to the model.

### E. Provider Control Drivers (`internal/control/driver`)
- Dedicated drivers for OpenAI Codex, Google AGY, Anthropic Claude Code, OpenCode, and Gemini CLI.
- Granular capability declaration (`Process`, `Terminal`, `Attach`, `StructuredEvents`, `Sessions`, `Resume`, `SlashControl`).

### F. Continuity & Handoff Engine (`internal/control/handoff`)
- **Account Handoff:** Migrates active work to another profile of the same provider while preserving the underlying provider session ID.
- **Context Handoff:** Extracts workspace context (git branch, git status, changed files) with automatic secret redaction (`security.Redact`) and kicks off a new session on another provider.

### G. Event Bus (`internal/control/events`)
- In-memory pub/sub routing and historical buffer for runtime lifecycle events.

### H. Control Center TUI (`internal/control/tui`)
- Bubble Tea interface with tabs for Runtimes, Events, and Live Details.
- Full keyboard (`↑/↓`, `Tab`, `a`, `s`, `r`, `q`) and mouse support.

---

## 5. Provider Capability Matrix

| Provider | OS | Level | Process | Terminal | Events | Control API | Resume | Account Handoff | Context Handoff |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Codex** | Linux / Win / macOS | `CONTROL_API` / `TERMINAL` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` |
| **AGY** | Linux / Win / macOS | `TERMINAL` | `SUPPORTED` | `SUPPORTED` | `UNSUPPORTED` | `UNSUPPORTED` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` |
| **Claude Code** | Linux / Win / macOS | `TERMINAL` | `SUPPORTED` | `SUPPORTED` | `UNSUPPORTED` | `UNSUPPORTED` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` |
| **OpenCode** | Linux / Win / macOS | `CONTROL_API` / `TERMINAL` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` | `SUPPORTED` |
| **Gemini CLI** | Linux / Win / macOS | `TERMINAL` | `SUPPORTED` | `SUPPORTED` | `UNSUPPORTED` | `UNSUPPORTED` | `UNSUPPORTED` | `UNSUPPORTED` | `SUPPORTED` |

---

## 6. Verification Results

- **Unit & Concurrency Tests:** `go test -race ./...` (40+ passing tests, 0 failures, 0 races).
- **Classic Regression:** `TestControlPlaneCLICommands` (100% passing).
- **CLI Smoke Tests:** `ai --help`, `ai control --help`, `ai control doctor`, `ai control running --json`.
- **Driver Detection:** 5/5 installed drivers detected on system.

---

## 7. Production Recommendation

The `feat/ai-control-runtime` branch is stable, thoroughly tested, and ready for production use and code review.
