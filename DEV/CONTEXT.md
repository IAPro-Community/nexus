# Contexto atual

## Atualização de estado — PR #54, primeiro uso Codex/OpenCode no Windows — 2026-10-21

- Branch fix/issue-53-windows-agent-startup, commit d42db81b; PR #54 contra release/v1.0.0 e relacionado ao issue #53.
- Escopo: somente o startup e o terminal Codex/OpenCode após a instalação limpa beta.10 relatada em Windows amd64. As mudanças cobrem filtragem dos metadados do launcher, resolução PATHEXT, fallback --no-daemon do TUI Codex, preservação da configuração OpenCode e ConPTY no Windows.
- Testes dos seis pacotes relacionados, go vet, build Windows amd64 e smoke instalado passaram; Codex e OpenCode ficaram RUNNING e OpenCode exibiu a tela inicial interativa.
- A suíte Go completa passou antes da última mudança ConPTY e não foi repetida para o conjunto final. Sanitize self-hosted workspace estava pendente na última consulta do CI.
- A primeira execução de teste leu credencial OpenCode Zen por USERPROFILE ainda apontar ao perfil real; os seis pacotes passaram após isolar USERPROFILE temporariamente. A validação visual do frontend ficou fora da PR.

## Registro pré-PR — TUI do OpenCode no Windows — 2026-10-21

- Causa da tela vazia: no Windows, provedores genéricos usavam pipes comuns; o OpenCode iniciava e conectava, mas sua TUI não tinha uma console real para renderizar. `NewBackendForProvider` agora escolhe ConPTY automaticamente para OpenCode, mantendo pipes para provedores desconhecidos e os overrides existentes por variável de ambiente. A capacidade do driver também informa ConPTY corretamente.
- Testes direcionados `go test -count=1 -timeout=5m ./internal/control/terminal ./internal/control/driver`, `go vet ./internal/control/terminal ./internal/control/driver ./internal/control/host` e build Windows amd64 passaram. Build instalado: `%LOCALAPPDATA%\Programs\IAPro Nexus\nexus.exe`, beta.10, commit local `f6098b8c-dirty`, SHA-256 `97BBF81F6BD405008AA3D3138B89B929CD47E71D056903E74C5A8AEF3CB70FB5`.
- Após reiniciar o servidor instalado e recuperar os agentes, `nexus ps --json` confirmou Codex e OpenCode em `RUNNING`; a interface exibiu dois PTYs, a tela inicial do OpenCode (1.19.0) e o TUI Codex. Nenhum prompt foi enviado nesta verificação; o Codex retomou a sessão anterior “Inspect Codex executable path | GreenHub”.
- Cópia anterior preservada em `%TEMP%\nexus-before-opencode-conpty-20261021.exe` (SHA-256 `BE53D2D8EB898C8459909CE8CBC8864309F90E6593DB5569A402C3EFB9D754EC`). O host não permitiu cadastrar `nexus.dev`, então o app segue em `http://127.0.0.1:13000`. Sem commit ou push.

## Registro anterior à correção ConPTY — Windows — 2026-10-21

- Correções locais em `release/v1.0.0`, HEAD `f6098b8c`, sem commit: metadados do launcher, precedência de `PATHEXT`, fallback `--no-daemon` do TUI Codex para caminhos longos e isolamento OpenCode que preserva a configuração já provisionada no perfil.
- Causa OpenCode confirmada: a inicialização copiava novamente cerca de 3.676 arquivos/55 MB de `%USERPROFILE%\.config\opencode` antes de abrir o endpoint IPC; a cópia com flush por arquivo excedia o timeout de handshake. Agora só a configuração OpenCode é preservada depois de provisionada no perfil; outros diretórios mantêm a sincronização existente.
- `go test -count=1 -timeout 20m ./...`, `go vet ./...`, `golangci-lint v2.12.2` (0 issues), 577 testes web, typecheck, lint, validações de estilo, build e `verify` passaram. Os testes Go foram executados com perfil de usuário isolado para não ler credenciais reais.
- Build Windows amd64 instalado naquele smoke em `%LOCALAPPDATA%\Programs\IAPro Nexus\nexus.exe`, SHA-256 `BE53D2D8EB898C8459909CE8CBC8864309F90E6593DB5569A402C3EFB9D754EC` (depois preservado como backup da atualização ConPTY). `version --json` confirmou beta.10 / `f6098b8c-dirty`; `GET http://127.0.0.1:13000/` retornou HTTP 200.
- Smoke real após fechar a aba e reiniciar o servidor instalado: a interface carregou `1.0.0-beta.10`; Codex e OpenCode foram recuperados pelo botão Iniciar/Recuperar e aparecem `WORKING`. `nexus ps --json` confirma os provedores `codex` e `opencode` em `RUNNING`, com dois PTYs na interface. Nenhum prompt foi enviado; o Codex retomou apenas a sessão anterior já concluída. `nexus web` não conseguiu registrar `nexus.dev` no hosts sem elevação, mas serviu normalmente em `127.0.0.1:13000`.
- O executável anterior foi preservado em `%TEMP%\nexus-before-final-fix-20261021.exe`. Não houve commit nem push.

## Estado

- Produto: IAPro Nexus Workspace OS.
- Versão-fonte em homologação: `1.0.0-beta.7` (`VERSION`).
- O launcher persiste a identidade autoritativa do `SessionHost` após o
  handshake (`PID`, `HostPID`, `HostGeneration` e fingerprints); isso evita que
  o registro final apague a geração usada pelo attach.
- A instalação local reinicia apenas o `nexus web` associado ao binário
  instalado, evitando hosts Web em memória com executável `(deleted)`.
- Branch ativa neste checkout Windows: release/v1.0.0, HEAD-base 52f35c04181b85379219ab55fc3192c63a5440a8; a árvore suja contém ajustes locais de homologação. Não há merge Beta 7 pendente. Gates Windows locais passaram; NSIS exato compilado e variante isolada com payload real passaram; Desktop Wails/WebView2 validado por 10s com cleanup limitado ao perfil Nexus. Exact-SHA, gate docs-verify (manifesto visual stale em 2892f301), renderização visual, CI nativo, Authenticode e update trust root continuam NO-GO; host sem runner Windows, Windows SDK/signtool e certificado.
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

## Estado Windows atual — Nexus 1.0.0-beta.7 — 2026-10-07

Project Hub corrigido e validado em 775×483; gates web, oito suítes browser, builds CLI/Desktop/HUD e smokes isolados passaram no worktree 52f35c04-dirty. O SHA d14154d2 solicitado é ancestral, mas não é o HEAD. A release segue NO-GO até homologação exact-SHA, assinatura/trust root e fechamento de CI/instalador. Ver DEV/validation/current/RELEASE_STATUS.md e DEV/validation/current/BETA7_WINDOWS_HOMOLOGATION_20261007.md.
