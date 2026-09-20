# Contexto atual

## Estado

- Produto: IAPro Nexus Workspace OS.
- Versão operacional instalada: `1.0.0-beta.2`.
- O launcher persiste a identidade autoritativa do `SessionHost` após o
  handshake (`PID`, `HostPID`, `HostGeneration` e fingerprints); isso evita que
  o registro final apague a geração usada pelo attach.
- A instalação local reinicia apenas o `nexus web` associado ao binário
  instalado, evitando hosts Web em memória com executável `(deleted)`.
- Branch ativa neste worktree: `feat/capacity-monitor-revival`.
- Versão-fonte: arquivo `VERSION`.
- Launcher destacado usa envelopes privados de uso único para argumentos opacos.
- Motor de Flags Canônicas e Normalização (`internal/control/flags`): mapeamento transparente de `--yolo`, `-y`, `--continue`, `-c`, `--resume`, `-r`, `--print`, `-p`, `--effort`, `--plan` e suporte a `Merged Help` (`nexus <provider> --help`).
- Quotas em tempo real para Codex: parsing dinâmico de rollouts de sessão (`rollout-*.jsonl`) com rate limits primário (5h) e secundário (semanal) associados ao grupo `claude_gpt`.
- Chaveiro seguro de sessões D-Bus: isolamento de diretório de controle temporário exclusivo (`--control-directory`), eliminando conflito com o keyring do host e timeout de 25s de `org.freedesktop.secrets`.
- Perfis Codex, AGY e OpenCode podem reaproveitar artefatos locais de conversa sem copiar credenciais.
- Concorrência supervisionada Codex: `AcquireTUILease` compartilhado por runtime;
  `TryAcquireMaintenanceLock` exclusivo para preparação/migração/app-server;
  Leads interativos são reservados atomicamente em `STARTING`.
- Promoção adaptativa: `internal/nexus/promotion` mantém sombra de linha sem
  interpretar ESC/CSI/OSC ou bracketed paste; CLI e Web aplicam `AUTO`, `ASK`
  e `OFF` sem transformar perfil, terminal ou Lead em singleton.

## Comandos

- Build: `make build` ou `go build -o /home/desenvolvedor/.local/bin/nexus ./cmd/nexus`.
- Testes focados: `go test ./internal/control/flags ./internal/control/driver ./internal/core/provider/adapters/codex ./internal/core/quota`.
- Web local: `yarn dev` ou `yarn web:dev`.

## Restrições

- Web local deve escutar em `127.0.0.1`; wildcard exige desenho remoto explícito.
- Dados de quota sem fonte/data verificável devem ser tratados como `UNKNOWN`.
- O worktree fecha a rodada surface-isolation 2026-09-20 commitada (código
  `63132df..8156836` + docs deste commit); não sobrescrever mudanças alheias.

## Consolidação arquitetural 2026-09-10

- API route registration is composed in `internal/control/web/routes.go` while
  preserving the existing standard-library router and handler contracts.
- `GET /api/v1/system/info` is the additive metadata contract for API version,
  build, providers, and driver capabilities; Web has a typed client method.
- Architecture and compatibility docs are under `docs/architecture/` and
  `docs/refactoring/`; the implementation plan is under `docs/superpowers/plans/`.
- Projects and Agents transport handlers now live in
  `internal/control/web/handlers_projects.go` and
  `internal/control/web/handlers_agents.go`; project/agent route dispatch is in
  `internal/control/web/routes_projects_agents.go`.
- Provider resource allocation handlers now live in
  `internal/control/web/handlers_resources.go`.
- Project CRUD/layout/events and resource allocation/recommendation now have
  narrow application services in `internal/nexus`, consumed by Web transport.
- Agent list/create/detail/update/delete now use
  `internal/nexus/agent_application.go`; runtime lifecycle remains on Nexus.
- WorkPlan CRUD and revision reads now use
  `internal/nexus/plan_application.go`; Composer/intelligence/compile/run
  remain on Nexus while their cross-domain dependencies are not isolated.
- Composer transport operations now use
  `internal/nexus/composer_application.go`; Composer domain rules remain on
  Nexus, and all public Composer operations honor canceled contexts.
- Mission planning operations now use `internal/nexus/mission_application.go`
  for CRUD, detail, tasks and assignments. MissionRun execution remains in
  `mission_service.go`/runner and is intentionally not duplicated in the
  planning boundary.
- `RunRepository` now honors cancellation in both its memory implementation
  and the SQLite adapter for save/read/list/lease operations; tests cover the
  transport-facing runtime contract.
- MissionRun Web operations now pass through
  `internal/nexus/run_application.go`; it is a thin boundary over the existing
  lifecycle/runner, not a second state machine.
- Raw runtime list/start/detail/delete/title, stop/respond and handoff calls
  now pass through `internal/nexus/runtime_application.go`; IPC and provider
  mechanics remain explicit control-plane dependencies.
- Event records now support optional correlation IDs persisted in
  `events_metadata`; MissionRun failover and account handoff populate them.
- Context and account handoff both use their lineage ID for event correlation;
  the stable current event taxonomy and timeline rules are documented in
  `docs/architecture/events.md`.
- `RunApplicationService` emits correlated MissionRun lifecycle/step events
  through an injectable EventBus; it uses the run ID and preserves runtime
  identity when a package has one, plus project/agent ownership metadata for
  durable timeline projection.
- HTTP errors preserve the legacy `error` string and add stable `code`; the
  filesystem mkdir policy resolves existing ancestors to block symlink escape.
- Runtime stop observation in the HTTP API now honors request cancellation
  during its bounded process-reap wait.
- The Web/Desktop HTTP transport is centralized in `web/src/api.ts`; the
  domain facade `web/src/nexus/api.ts` reuses its auth, CSRF, base URL and
  error pipeline while retaining the `NexusAPIError` public alias.
- Consumer ownership and the intentional direct-Core CLI path are documented
  in `docs/architecture/consumer-matrix.md`.
- The current requirement-by-requirement audit is
  `docs/refactoring/definition-of-done-audit.md`; it intentionally keeps the
  campaign open for runtime/event/native-platform evidence.
- Usage windows now have optional absolute limit/used/remaining values and a
  confidence field in addition to percentages; nil/UNKNOWN remains distinct
  from measured zero.

## Nexus Core consolidation — current semantic state

- `internal/nexus/intelligence.AgentSpec` is the provider-independent,
  revision-persisted specialization. Legacy `Agent.Role` is normalized into a
  minimal spec without invented instructions.
- `CompileExecutionContext` is the shared context compiler and exposes
  inspectable section provenance. `WorkPackage.role` is a task role layered on
  top of `AgentSpec.role`; the rendered context names both explicitly.
- Direct custom-specialization execution, mission package compilation, and
  headless execution use that compiler. Direct execution does not require
  Maestro; enabled Maestro guidance is an optional compiler section.
- CLI plan compile/run now call Core services. `agents PROJECT` uses the
  dispatcher-relative positional argument. The complete command-registry
  rewrite remains deferred because provider-direct parsing is intentionally
  provider-native.
