# Architecture: IAPro Nexus Workspace OS

```text
                                 nexus CLI / TUI / Web
                                          │
                            Provider Registry & Dispatcher
                                          │
         ┌──────────────┬────────────┼────────────┬──────────────┐
         │              │            │            │              │
       Codex           AGY         Claude      OpenCode        Gemini
      Adapter        Adapter      Adapter      Adapter        Adapter
         │              │            │            │              │
     CODEX_HOME    profile HOME   CLAUDE_HOME  OPENCODE_HOME  GEMINI_HOME
    per profile   XDG per profile per profile  per profile    per profile
         │        D-Bus keyring      │            │              │
         │              │            │            │              │
   official codex  official agy official claude official opencode official gemini
         │              │            │            │              │
         └──────────────┴────────────┼────────────┴──────────────┘
                                          │
                                  current directory
                                 same Linux UID/GID
```

---

## 1. Core Principles

- **No Central OAuth Proxy**: Authentication and refresh tokens are owned by official CLIs. `nexus` manages process isolation, environment variables, and credential storage paths.
- **Honest Quota Engine**: Never presents missing data as 100%. Quotas explicitly report `LIVE`, `CACHED`, `UNKNOWN`, `RATE_LIMITED`, or `UNSUPPORTED`.
- **Smart Account Selection**: Scores candidate accounts based on capacity, health, project bindings, and priorities.
- **Automatic Fallback**: Recovers from 429 rate limits by switching to the next best account seamlessly.

---

## 2. Component Diagram

```mermaid
flowchart TB
    CLI["nexus CLI / TUI / Web Workspace OS"] --> Dispatcher["Command Dispatcher & Controller"]
    Dispatcher --> Scheduler["Smart Account Selector"]
    Dispatcher --> Registry["Provider Registry"]
    Dispatcher --> SessionIndex["Universal Session Index"]
    Dispatcher --> SecurityEngine["Isolation & Security Engine"]

    Scheduler --> QuotaEngine["Usage & Quota Engine (TTL / Cache)"]
    Scheduler --> Cooldown["Cooldown & Rate Limit Tracker"]
    Scheduler --> Config["Config & Project Bindings Store"]

    Registry --> CodexAdapter["Codex Provider Adapter"]
    Registry --> AGYAdapter["AGY Provider Adapter"]
    Registry --> ClaudeAdapter["Claude Code Adapter"]
    Registry --> OpenCodeAdapter["OpenCode Adapter"]
    Registry --> GeminiAdapter["Gemini CLI Adapter"]

    CodexAdapter --> IsolatedRuntime["Isolated Process Runtime (PTY / Signals)"]
    AGYAdapter --> IsolatedRuntime
    ClaudeAdapter --> IsolatedRuntime
    OpenCodeAdapter --> IsolatedRuntime
    GeminiAdapter --> IsolatedRuntime
```

---

## 3. Run & Fallback Sequence

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant CLI as nexus CLI
    participant Selector as AccountSelector
    participant Runtime as IsolatedRuntime
    participant Adapter as ProviderAdapter
    participant Cooldown as CooldownTracker

    User->>CLI: nexus codex
    CLI->>Selector: SelectBestProfile(codex)
    Selector-->>CLI: Selected: profile-work (Capacity: 85%)
    CLI->>Runtime: RunInteractive(profile-work, args)
    Runtime->>Adapter: Execute Official CLI
    Adapter-->>Runtime: Process Exits with 429 RateLimit
    Runtime-->>CLI: Execution Result: Failure(Kind=RateLimitFailure)
    CLI->>Cooldown: RecordRateLimit(codex, profile-work, 30m)
    CLI->>Selector: SelectBestProfile(codex, exclude=[profile-work])
    Selector-->>CLI: Selected: profile-personal (Capacity: 70%)
    CLI->>User: ⚠ profile-work rate-limited. Falling back to profile-personal...
    CLI->>Runtime: RunInteractive(profile-personal, args)
    Runtime->>Adapter: Execute Official CLI
    Adapter-->>User: Interactive Session Connected
```

---

## 4. Provider Capabilities Matrix

| Provider | Profiles | Isolation | Usage | Resume Syntax | Cross-Account Resume | Auto Selection |
|---|---|---|---|---|---|---|
| **Codex** | Unlimited | `CODEX_HOME` | `LIVE` / `CACHED` | `codex resume <id>` | Yes (shared sessions) | Yes |
| **AGY** | Unlimited | D-Bus Keyring + `HOME` | `LIVE` / `CACHED` | `agy --conversation=<id>` | Yes (shared brain) | Yes |
| **Claude Code** | Unlimited | `CLAUDE_CONFIG_DIR` + `HOME` | `LIVE` / `UNKNOWN` | `claude --resume <id>` | No | Yes |
| **OpenCode** | Unlimited | `OPENCODE_CONFIG_DIR` + `XDG` | `LIVE` / `UNKNOWN` | `opencode -s <id>` | Yes | Yes |
| **Gemini CLI** | Unlimited | `GEMINI_CLI_HOME` + `HOME` | `LIVE` / `UNKNOWN` | `gemini -r <id>` | No | Yes |

---

## 5. Nexus Supervised Runtimes Architecture

The Nexus Control Plane introduces a supervised runtime model where AI developer tools execute inside a managed host wrapper. This wrapper provides cross-provider interoperability, resilient handoffs, and capability discovery.

### Core Concepts

- **Session Host (`internal/control/host`)**: The background daemon wrapper for a single provider process. It manages a persistent Unix Domain Socket (or Windows Named Pipe) for RPC commands and interactive terminal data streaming.
- **Control Drivers (`internal/control/driver`)**: Provider-specific adapters that declare runtime capabilities (e.g., `Process`, `Terminal`, `Sessions`, `Resume`, `SlashControl`) and construct exact execution binaries and environments.
- **Session Registry (`internal/control/registry`)**: A unified, cross-process persistent registry tracking all active runtime sessions, PIDs, host PIDs, and active states (`STARTING`, `RUNNING`, `STOPPED`, `HANDOFF`).
- **Handoff Engine (`internal/control/handoff`)**: Orchestrates safe state transitions, capturing bounded `WorkCheckpoints` (git branches, diff stats, goal) and persisting `LineageRecords` to safely move work between profiles or providers.

### Inter-Process Control Flow

1. The `nexus start` (or `nexus control start`) command provisions a new `registry.RuntimeSession`, spawns `nexus __control-host` in the background, and immediately attaches to the new socket.
2. The background `SessionHost` runs the actual provider CLI (via PTY) and broadcasts stdout to all attached clients via a ring buffer.
3. Attached user terminals send input which the `SessionHost` intercepts using a `SlashRouter`.
4. Commands like `/nexus handoff` trigger asynchronous transactional handoff routines that quiesce the current process, launch the target process, and safely link session IDs before stopping the source process.

---

## 6. Canonical Flags Normalization & Merged Help

To provide seamless developer ergonomics across disparate AI coding tools, Nexus provides a unified flag normalization and help engine (`internal/control/flags`):

- **Canonical Flag Translation**: Common universal options (`--yolo`, `-y`, `--continue`, `-c`, `--resume`, `-r`, `--print`, `-p`, `--effort`, `--plan`) are intercepted and translated to each CLI's native syntax before execution. Native options are never shadowed and remain 100% supported.
- **Merged Help Pipeline**: Invoking `nexus <provider> --help` or `nexus help <provider>` inspects provider capabilities, generates an informative comparison banner of canonical aliases, and transparently passes through the complete native CLI help output.
