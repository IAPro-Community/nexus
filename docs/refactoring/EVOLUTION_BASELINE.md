# Evolution Baseline

**Date:** 2026-09-11
**Branch:** feat/nexus-maximum-delivery
**HEAD:** ab056f558fb2886b21db60ca7951cfb92deb46c7
**Platform:** Linux x86_64
**Go:** 1.25.0 | **Node:** v22.17.0 | **Bun:** 1.3.9
**Go Tests:** 753 passed / 63 packages
**Frontend Tests:** 329 passed / 64 files

---

## Surface Inventory

### CLI (40+ commands)

| Area | Commands | Status |
|------|----------|--------|
| Provider dispatch | `codex`, `agy`, `claude`, `opencode`, `gemini`, `cursor` | VERIFIED |
| Provider profiles | `providers`, `providers status`, `providers register` | VERIFIED |
| Account mgmt | `add`, `remove`, `rename`, `login`, `logout`, `profiles`, `switch` | VERIFIED |
| Session mgmt | `sessions`, `workspaces`, `bind`, `unbind`, `bindings` | VERIFIED |
| Control plane | `start`, `stop`, `ps`, `attach`, `handoff`, `continue`, `resume` | VERIFIED |
| Information | `status`, `usage`, `current`, `explain`, `history`, `stats` | VERIFIED |
| Project mgmt | `projects`, `plan`, `agents`, `run` | VERIFIED |
| System | `doctor`, `security`, `config`, `update`, `maestro`, `export`, `completion` | VERIFIED |
| Desktop | `web`, `control`, `tunnel` | VERIFIED |
| Flags | `--yolo`, `--continue`, `--resume`, `--print`, `--supervised`, `--direct` | VERIFIED |
| Colon syntax | `provider:profile`, `provider:auto` | VERIFIED |
| JSON output | `--json` on 21 commands | VERIFIED |
| Completion | bash, zsh, fish, powershell | VERIFIED |

### Web Frontend

| Area | Components | Status |
|------|-----------|--------|
| Workspace OS Shell | NexusShell, NexusWorkspaceApp, WorkspaceRenderer | VERIFIED |
| Routes | `/`, `/projects`, `/settings`, `/updates`, `/welcome`, `/p/:id/*` | VERIFIED |
| Project System | ProjectRail, ProjectHub, ProjectManager, AddProject, DirectoryBrowser | VERIFIED |
| Agent System | AgentsSurface, AgentConfiguration, NewAgentModal, AgentTerminal | VERIFIED |
| Terminal | xterm.js, WebSocket PTY, lease management, reconnect | VERIFIED |
| Composer | WorkSurface, ComposerSurface, conversation mode | VERIFIED |
| Flow | FlowCanvas (ReactFlow), FlowRunsHistory, FlowStepInspector | VERIFIED |
| Missions | MissionsPage, MissionAutonomyCard | VERIFIED |
| Resources | ResourcePicker, UsageSurface, UsageAccountCard | VERIFIED |
| Settings | SettingsSurface (5 tabs), IntelligenceProvider, RemoteAccess | VERIFIED |
| Notifications | InAppNotificationCenter, PushNotificationManager, AttentionRadar | VERIFIED |
| Maestro | MaestroSurface (skills catalog) | VERIFIED |
| Sessions | SessionsSurface (lineage) | VERIFIED |
| i18n | pt-BR, en, es via react-i18next | VERIFIED |
| Design System | 10 themes, 4 accents, 2 densities, primitives library | VERIFIED |
| Desktop | Wails v2, PlatformBridge, native file picker, deep links | VERIFIED |

### Desktop (Wails)

| Area | Status |
|------|--------|
| Core attachment | VERIFIED |
| Single instance lock | VERIFIED |
| Auth bootstrap | VERIFIED |
| Deep links | VERIFIED |
| WebSocket | VERIFIED |
| System tray | VERIFIED |
| Native menus | VERIFIED |

### Control Plane

| Area | Status |
|------|--------|
| SessionHost | VERIFIED |
| PTY (creack/pty) | VERIFIED |
| ConPTY (Windows) | UNVERIFIED (no Windows runner) |
| Protocol (19 cmd types) | VERIFIED |
| Registry (runtimes.json) | VERIFIED |
| Launcher | VERIFIED |
| Event Bus | VERIFIED |
| Handoff Service | VERIFIED |
| Slash Router | VERIFIED |
| SlashPrefixRouter | VERIFIED |
| Driver Registry (8 drivers) | VERIFIED |

### Provider System

| Area | Status |
|------|--------|
| Provider Registry (6 adapters) | VERIFIED |
| ControlDriver (8 implementations) | VERIFIED |
| Capability Detection | VERIFIED |
| Profile Isolation | VERIFIED |
| AccountScope | VERIFIED |
| Quota Engine | VERIFIED |
| Cooldown Tracker | VERIFIED |
| Flag Normalizer | VERIFIED |

### Persistence

| Area | Status |
|------|--------|
| SQLite Store | VERIFIED |
| Agent/Revision/Generation models | VERIFIED |
| Event correlation | VERIFIED |
| Migration system | VERIFIED |

### Security

| Area | Status |
|------|--------|
| Auth bootstrap | VERIFIED |
| CSRF protection | VERIFIED |
| Origin policy | VERIFIED |
| Path traversal protection | VERIFIED |
| Secret redaction | VERIFIED |
| Cookie Secure (tunnel-aware) | VERIFIED |
| Cloudflared pinned + SHA-256 verified | VERIFIED |
| Installer signed manifest | PARTIAL — implementação e testes locais verificados; chave pública de produção e execução do release nativo ainda pendentes |

---

## Reused Architecture (NOT to replace)

| Subsystem | Location | Notes |
|-----------|----------|-------|
| SessionHost | `internal/control/host/host.go` | Core runtime supervisor |
| PTY/ConPTY | `internal/control/terminal/` | Platform-specific terminal backend |
| ControlDriver | `internal/control/driver/` | 8 provider runtime adapters |
| RuntimeLauncher | `internal/control/launcher/` | Launch orchestrator |
| RuntimeSession Registry | `internal/control/registry/` | Cross-process state |
| handoff.Service | `internal/control/handoff/` | Account + context handoff |
| PerformAccountHandoff | `internal/control/handoff/account.go` | Same-provider switch |
| PerformContextHandoff | `internal/control/handoff/context.go` | Cross-provider switch |
| RuntimeGeneration | `internal/nexus/store/models.go` | Generation lineage |
| AgentRevision | `internal/nexus/store/models.go` | Agent config versions |
| AgentSpec | `internal/nexus/intelligence/types.go` | Agent specialization |
| SlashPrefixRouter | `internal/control/host/slash_prefix.go` | Byte-level state machine |
| RouteSlashCommand | `internal/control/host/slash_router.go` | Line-level command dispatch |
| ResourceScheduler | `internal/nexus/scheduler.go` | Policy-based provider selection |
| RecommendResources | `internal/nexus/resource_recommendation.go` | Functional scoring |
| MissionRunner | `internal/nexus/runner/` | Durable state machine |
| Composer | `internal/nexus/composer.go` | Goal elaboration engine |
| Event Bus | `internal/control/events/` | Pub/sub with history |
| Quota Engine | `internal/core/quota/` | Usage tracking |
| Flag Normalizer | `internal/control/flags/normalizer.go` | Canonical alias translation |
| Control Protocol | `internal/control/protocol/` | JSON IPC over Unix/NamedPipe |

---

## Definition of Done — Milestone A (Interactive Continuity)

- [ ] `nexus codex --yolo` enters supervised in TTY
- [ ] `--direct` preserves old path
- [ ] Headless doesn't break
- [ ] `:nexus` canonical works
- [ ] Short aliases work
- [ ] Provider slash commands remain intact
- [ ] Legacy aliases continue working
- [ ] Dynamic completion works
- [ ] Agent/Project survive account handoff
- [ ] RuntimeGeneration N+1 registered
- [ ] No double writer
- [ ] Rollback proven
- [ ] Continuity states honest
- [ ] Terminal follows runtime replacement
- [ ] User doesn't involuntarily return to shell
- [ ] Fake provider E2E complete

## Definition of Done — Milestone B (Agent Routing Truth)

- [ ] TaskRequirements evolved without breaking compatibility
- [ ] AgentSpec has domains/strengths/tags when needed
- [ ] Legacy Agents remain readable
- [ ] Centralized taxonomy
- [ ] Single AgentMatcher
- [ ] Hard gates tested
- [ ] Deterministic ranking
- [ ] Confidence scoring
- [ ] Explainability
- [ ] Flow uses AgentMatcher
- [ ] Mission uses AgentMatcher
- [ ] Direct run uses AgentMatcher
- [ ] Agent specialization reaches execution context
- [ ] Complex task requires decomposition

## Definition of Done — Milestone C (Resource Continuity)

- [ ] ResourceScheduler evolved (not replaced)
- [ ] No duplicate scheduler
- [ ] Provider/profile/model selection explainable
- [ ] Quota truth preserved
- [ ] No hardcoded provider-per-role
- [ ] `:switch auto` uses ResourceScheduler
- [ ] Manual switch green
- [ ] Auto failover only by policy
- [ ] Safe point protected
- [ ] Same-provider prefers continuity
- [ ] Cross-provider uses context handoff
- [ ] Native resume never falsely claimed
