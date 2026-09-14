# AI CLI — Control Baseline Audit

**Audit Date:** 2026-08-27
**Target Project:** `github.com/IAPro-Community/nexus`

---

## 1. Git Baseline Identity

- **Base Branch:** `feat/control-plane-evolution`
- **Base SHA:** `ced1e31191544a49fe05634d0b134d1b848c783e`
- **Main SHA:** `ced1e31191544a49fe05634d0b134d1b848c783e`
- **Merge Base with Main:** `ced1e31191544a49fe05634d0b134d1b848c783e`
- **Current Active Branch:** `feat/ai-control-runtime`
- **Working Tree State:** CLEAN

---

## 2. Architecture Overview

```text
                               ai CLI (Classic & TUI)
                                         │
                             Dispatcher / Controller
                                         │
                    ┌────────────────────┼────────────────────┐
                    │                    │                    │
              Provider Registry    Quota Engine         Smart Scheduler
                    │                    │                    │
        ┌───────────┼───────────┐        │            (Multi-factor scoring,
      Codex        AGY        Claude     │             Workspace bindings,
     Adapter     Adapter     Adapter     │             Cooldown tracker)
        │           │           │        │                    │
     OpenCode    Gemini         └────────┼────────────────────┘
     Adapter     Adapter                 │
        │           │                    │
        └───────────┴────────────────────┘
```

The system operates as a local Control Plane coordinating CLI authentication, sandbox isolation (`0600`), honest quota resolution (`LIVE`, `CACHED`, `UNKNOWN`, `RATE_LIMITED`), and automatic fallback without acting as a central OAuth proxy.

---

## 3. Capability Audit & Classification

| Capability / Subsystem | Location | Status | Notes |
| :--- | :--- | :--- | :--- |
| **CLI Dispatcher & Classic Mode** | `internal/app/app.go` | `IMPLEMENTED_AND_TESTED` | `ai codex`, `ai agy`, `ai claude`, `ai gemini`, `ai opencode` |
| **Profile & Sandbox Isolation** | `internal/profile`, `internal/core/security` | `IMPLEMENTED_AND_TESTED` | Keyrings, D-Bus session, presets (`developer`, `strict`, `compat`) |
| **Secret Redaction Engine** | `internal/core/security/redact.go` | `IMPLEMENTED_AND_TESTED` | Masks OpenAI keys, Anthropic keys, Google OAuth, private keys, JWTs |
| **Honest Quota Engine** | `internal/core/quota/quota.go` | `IMPLEMENTED_AND_TESTED` | Statuses: `LIVE`, `CACHED`, `UNKNOWN`, `RATE_LIMITED`, `UNSUPPORTED` |
| **Smart Account Selector** | `internal/core/scheduler/scheduler.go` | `IMPLEMENTED_AND_TESTED` | Multi-factor scoring, `ai explain <provider>` |
| **Cooldown & 429 Tracker** | `internal/core/cooldown/cooldown.go` | `IMPLEMENTED_AND_TESTED` | Tracks HTTP 429 timeouts and rate limits |
| **Automatic Fallback Loop** | `internal/core/fallback/fallback.go` | `IMPLEMENTED_AND_TESTED` | Safe fallback to next available profile without retry loops |
| **Universal Session Index** | `internal/core/session/session.go` | `IMPLEMENTED_AND_TESTED` | Multi-provider session aggregation and searching |
| **Workspace / Project Bindings** | `internal/core/config/config.go` | `IMPLEMENTED_AND_TESTED` | `ai bind`, `ai unbind`, `ai workspaces` |
| **Interactive TUI (Classic)** | `internal/tui/tui.go` | `IMPLEMENTED_AND_TESTED` | 3-panel Bubble Tea interface (Providers, Accounts, Sessions) |
| **Linux Support & PTY** | `internal/runtime/runtime.go` | `IMPLEMENTED_AND_TESTED` | TTY passthrough, signal handling |
| **Windows / PowerShell Support** | `install.ps1`, `internal/core/config` | `IMPLEMENTED_AND_TESTED` | Path normalization, `%USERPROFILE%`, `%LOCALAPPDATA%`, completions |
| **Shell Completions** | `internal/app/app.go` | `IMPLEMENTED_AND_TESTED` | Bash, Zsh, Fish, PowerShell (`pwsh`) |
| **1-Line Zero-Clone Installers** | `install.sh`, `install.ps1` | `IMPLEMENTED_AND_TESTED` | GitHub release downloads + Go build fallback |
| **Automated Test Suite** | `*_test.go` | `IMPLEMENTED_AND_TESTED` | 35+ tests passing with race detector (`go test -race ./...`) |
| **Documentation & Guides** | `README.md`, `README.en.md`, `docs/` | `IMPLEMENTED_AND_TESTED` | Full bilingual documentation with generic examples |
| **AI Control Runtime Subsystem** | `internal/control` | `MISSING` | To be implemented in `feat/ai-control-runtime` |
| **SessionHost & IPC Protocol** | `internal/control/host` | `MISSING` | To be implemented in `feat/ai-control-runtime` |
| **Universal Slash Channel (`/ai`)** | `internal/control/slash` | `MISSING` | To be implemented in `feat/ai-control-runtime` |
| **Account Handoff (`/ai handoff`)** | `internal/control/handoff` | `MISSING` | To be implemented in `feat/ai-control-runtime` |
| **Context Handoff (`/ai continue`)** | `internal/control/context` | `MISSING` | To be implemented in `feat/ai-control-runtime` |
| **Runtime Control TUI (`ai control`)** | `internal/control/tui` | `MISSING` | To be implemented in `feat/ai-control-runtime` |

---

## 4. Existing Provider Adapters Audit

1. **OpenAI Codex (`codex`)**:
   - Capabilities: `Login`, `Logout`, `InspectAuth`, `GetUsage` (5h/weekly), `ListConversations`, `Resume` (`codex resume <id>`), `ClassifyError`.
   - Control Level: `TERMINAL` / `PROCESS` (with app-server detection guard).
2. **Google AGY / Antigravity (`agy`)**:
   - Capabilities: `Login`, `Logout`, `InspectAuth` (OAuth keyring / token inspection), `GetUsage` (5h/weekly), `ListConversations`, `Resume` (`agy --conversation=<id>`), `ClassifyError`.
   - Control Level: `TERMINAL` (D-Bus keyring sandbox).
3. **Anthropic Claude Code (`claude`)**:
   - Capabilities: `InspectAuth`, `GetUsage`, `ListConversations`, `Resume` (`claude --resume <id>`), `ClassifyError`.
   - Control Level: `TERMINAL` (`CLAUDE_CONFIG_DIR` isolation).
4. **OpenCode (`opencode`)**:
   - Capabilities: `Detect`, `Run`, `ClassifyError`.
   - Control Level: `TERMINAL` (with `opencode serve` detection guard).
5. **Google Gemini CLI (`gemini`)**:
   - Capabilities: `Detect`, `Run`, `ClassifyError`.
   - Control Level: `TERMINAL` (`GEMINI_CLI_HOME` isolation).

---

## 5. Verification Baseline

All 35 unit & integration tests pass with 0 data races:
```bash
go test -v ./...
go test -race ./...
```
