## Continuidade atual — versão 1.0.0-beta.11 e reconciliação upstream — 2026-10-26

- Branch `release/v1.0.0`; base `9d8d30afcefb7ec667bc4a0d3c1d1da15faf41b0`. 15 commits do remoto reconciliados e validados.
- Versão: `1.0.0-beta.11` sincronizada em `VERSION` e `web/package.json`. Contrato verificado por `scripts/verify-version-contract.go`.
- Suíte completa de testes passou: TypeScript, ESLint, Stylelint, Prettier, Vitest (110 suítes, 684 testes), Go vet, Go test (internal), e govulncheck.
- Binários locais compilados com sucesso.

## Continuidade atual — mascote NEX único e homologação Linux — 2026-10-26

- Branch `release/v1.0.0`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. O overlay atualmente compilado/instalado tem fingerprint `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c` (1.430 fontes; manifesto `DEV/validation/linux-100-2026-10-25/p5-current-recheck-2026-10-26-final/logs/source-provenance-preview-copy-fix-2026-10-26.txt`). Worktree compartilhada modificada, sem SHA Git congelado; não fazer commit/push automático.
- Contrato do produto: um único mascote NEX (`NexusHud → GuardianMascot → NexPixelSprite`). Barra, Mascote e Bandeja são apresentações do mesmo HUD. Companion 2D/PR #55 não faz parte do escopo; `companionModel.ts`/teste estão removidos. Preferências antigas relacionadas a `companionMotion` são somente compatibilidade/migração.
- Corrigi a prévia Composer/Flow que repetia o motivo de ausência de contas e mostrava o orçamento local em inglês. PASS: 20/20 testes focados, E2Es Composer completo e vago, e `quality:full` com 110 arquivos/681 testes/build. Sem contas elegíveis, preview segue bloqueado e não cria Mission/Agent. Chromium fixture não certifica a janela Wails nem provider externo.
- CLI/Desktop/HUD foram recompilados e instalados após backup. Hashes build/instalação coincidem: CLI `76c347d5a12697215c150aa87f6e2e52cf11e199af862c4fa16ca4e3ddc87fbe`, Desktop `4c68324423fce7e591c8e532092964375cccde2b0ccb725ee5066a65f830c6e2`, HUD `9a143bef1d9be86542dfb92da7248cadb9a3db88c18f508f38c30f78ef83e55a`. Unidade ativa; Core HTTP 200; versão reportada `1.0.0-beta.10`. Logs em `DEV/validation/linux-100-2026-10-25/p5-current-recheck-2026-10-26-final/logs/`.
- Importação real do host: Gemini copiado e reconhecido como autenticado no perfil Nexus; repetição idempotente, sem duplicatas; a sessão requer refresh e não foi chamada. Claude não tem tokens disponíveis e retornou `NO_SESSION`. Perfis Codex/AGY/OpenCode/Cursor já permanecem autenticados.
- Smoke real separado pelo CLI Nexus: Codex respondeu `NEXUS_RUNTIME_SMOKE_OK` (exit 0), em execução efêmera/read-only; nenhum runtime ficou ativo. Isso não valida handoff, continuidade, Skills ou a janela Wails. O stdout original não foi salvo; procedimento/registro sanitizado em `RUNTIME_VALIDATION.md`.
- Reexecução nativa no overlay atual: HUD Wails/Xvfb abriu em 224×158, mostrou um NEX e o clique abriu controles. Seletor contém apenas Barra/Mascote/Bandeja. Capturas e log em `MASCOT_VALIDATION.md`; alpha/Wayland/GPU continuam pendentes.
- Estado atual: **NO_GO_LINUX — 169 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN (191)**. Permanecem pendentes release estável/assinatura e SHA imutável, handoff/continuidade com provedores e preview P5 na janela Wails, alpha/foco/múltiplos monitores e GPU/compositor Wayland.

## Continuidade — DIRECT atômico até WorkPlan/MissionRunner e mascote NEX único — 2026-10-26

- HEAD/base local e remoto `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay fonte `ab5629f56e014719c0ce89c88b16b986f5bb0da8c310baac9fa079370140da5c` (1.428 arquivos; manifesto `DEV/validation/linux-100-2026-10-25/logs/source-provenance-p5-direct-atomic-mission-2026-10-26.txt`). Worktree compartilhada permanece modificada; sem commit/push.
- O produto mantém um único NEX. `companionModel`/teste e a preferência antiga de movimento como gravação canônica estão removidos; só existe o caminho `NexusHud → GuardianMascot → NexPixelSprite`. Barra/Mascote/Bandeja são apresentações do mesmo HUD. Não existe modo/renderer Companion 2D.
- PASS: cinco suítes Vitest focadas (33/33), `bun run art:nex:check` e `bun run test:e2e:hud` (repetido com TMPDIR/GOCACHE isolados após `/tmp` cheio); smoke nativo Wails/Xvfb 224×158 mostra um NEX e clique abre controles. Evidências em `DEV/validation/linux-100-2026-10-25/MASCOT_VALIDATION.md` e nos logs `mascot-scope-*` da pasta `logs/`. Xvfb não comprova alpha/Wayland.
- PASS P5-009: pedido DIRECT explicitamente promovido → WorkPlan de uma etapa → admissão → MissionRunner `COMPLETED_VERIFIED`; uma execução de fixture, `go test ./...` em worktree temporária e duas entradas de evidência durável (pacote/global). Repetição `-race -count=3`, `go vet ./internal/nexus` e suíte completa `go test -count=1 ./internal/nexus` passaram. Isso não é handoff de provider real nem E2E Browser/Wails; P5 continua parcial.
- Matriz agora: **NO_GO_LINUX — 145 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (168)**. Motivos pendentes incluem release estável/proveniência congelada, Wayland alpha/GPU, handoff externo autenticado, updater oficial e soak/inspeção final.
- Relatório: `DEV/validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md`.

## Continuidade — build/install e perfil Cursor autenticado em fixture — 2026-10-26

- HEAD/base local/remoto `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay testado/instalado `0df26de4a35641d1e5d02ded7757b257fa7f4f15747220416574c083b4127208` (1.428 fontes; manifesto `DEV/validation/linux-100-2026-10-25/p5-recheck-2026-10-26T0347Z/logs/source-provenance-after-auth-maestro-fixes.txt`). Worktree modificada, sem commit/push.
- `make build build-desktop-wails build-hud-wails` passou; CLI/Desktop/HUD ELF AMD64 e `ldd` sem ausências. Hashes: `fb45b0a42af24aa3427958983fb9315cbab338c88713b0290c4cb3f708222a8f`, `6bbbafdab42309d2657ceed4ccf90f1d87c4381aee5d7b6d4528903a136dc61f`, `76aabefd32c3eb88b5693546b3402e46f6006006250e2eff47822fa598e33e15`.
- Desktop e HUD instalados atomicamente com backup e unidade reiniciada; PID do Desktop/HUD vinculado carregam os hashes, Core HTTP 200. Backup: `/home/desenvolvedor/.local/state/ai-manager/nexus-update-backups/20261026T041144Z-import-auth-maestro-fix`.
- Perfil Cursor fixture já autenticado na UI Wails resultou em `1 já autenticada`, `0 sem suporte`; nenhum login ou token real foi copiado. HUD nativo mostra um único NEX e controles. Companion 2D segue fora do produto.
- Matriz atual: **NO_GO_LINUX — 144 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (167)**. Xvfb não certifica alfa/Wayland; provider externo, release estável, updater oficial e árvore imutável continuam sem aprovação. Relatório: `DEV/validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md`.

## Continuidade — regressões de sessão autenticada e perfil Maestro — 2026-10-26

- Branch `release/v1.0.0`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. Overlay atual dos testes Go: `0df26de4a35641d1e5d02ded7757b257fa7f4f15747220416574c083b4127208` (1.428 fontes; manifesto `DEV/validation/linux-100-2026-10-25/p5-recheck-2026-10-26T0347Z/logs/source-provenance-after-auth-maestro-fixes.txt`). Worktree compartilhada modificada, sem commit/push.
- FIX-030 reconhece perfil Nexus já autenticado antes de exigir importador do host; FIX-031 prioriza raiz isolada e não percorre ancestrais do `HOME`. O Nexus mantém um único mascote NEX, sem Companion 2D.
- PASS: `go test -count=1 -timeout=4m ./internal/nexus` (103,474 s), `go test -count=1 ./internal/profile`, `go vet ./internal/nexus ./internal/profile`. E2Es Composer passaram no overlay intermediário `1c3aa2c6ca9043f4a43dc3cd7a5ade451930d21c6fe3ff2e07de4cc0ed068693`; saída não pronta é esperada sem provider elegível.
- Matriz: **NO_GO_LINUX — 141 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (164)**. Esta correção não fecha release, compositor/alpha, provider real, idle acelerado, updater oficial ou proveniência imutável. O binário Wails instalado não inclui este último overlay ainda. Evidências em `DEV/validation/linux-100-2026-10-25/p5-recheck-2026-10-26T0347Z/`.

# Contexto atual

## Homologação Linux — atualização de 2026-10-26

- HEAD/base local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`, overlay de fontes `751cfc1a160c38d950ac5b9807e1febb9746d03981c861c722e5bd965e1ef012` (1.428 fontes, manifesto conferido sem divergências). Worktree compartilhada modificada, sem SHA próprio.
- Produto: um mascote NEX único, sem modo/renderer Companion 2D; Desktop e HUD Wails correntes passaram em Xvfb; Desktop/Core HTTP 200 e importação 0/0/5/1/0 sob HOME vazio. Alpha Wayland real continua BLOCKED.
- FIX-028 removeu o giro contínuo do indicador de atualização. FIX-029 corrigiu 23 advisories Bun; Bun 1.3.9 frozen install/audit Web/docs-site passaram sem advisories. `quality:full` 109/675/build e E2E HUD/Mission Attention passaram.
- CPU WebKit principal no Wails/Xvfb: 196,1% no baseline → 107,8% pós-fix em amostras de 20 s; delta direcional, pois o baseline tinha três WebKit e o reteste um (HUD spawn desativado). NR-010 continua BLOCKED por CPU remanescente e falta de GPU/profiler permitidos.
- REL-001: GitHub lista cinco previews e nenhuma release estável v1.0.0. A consulta OSV complementar resetou; não foi tratada como scan limpo. PRs abertas não foram integradas.
- **NO_GO_LINUX — 139 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (162)**. Soak P9 ativo; instalação do usuário preservada.

## Homologação Linux — recheck anterior, 2026-10-25 (UTC 26/10)

- Branch/HEAD local/remoto: `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. Fingerprint do overlay compilado: `9a66687715414fc8b42d612fd2b4a531370c7f8c4fbfa86338a81ba9728ddcd1` (1.428 fontes; manifesto `DEV/validation/linux-100-2026-10-25/logs/source-provenance-usage-host-session-import-2026-10-25.txt`; 1.428/1.428 entradas conferidas). Worktree modificada, sem commit imutável. Resultado **NO_GO_LINUX — 137 PASS / 1 FAIL / 20 BLOCKED / 4 NOT_RUN (162)**.
- Produto mascote: um único NEX (`NexusHud → GuardianMascot → NexPixelSprite`); não existe modo ou renderer Companion 2D. Preferências antigas são lidas para migração; gravações atuais usam `mascotMotion`.
- `quality:full` passou (109 arquivos/674 testes/build), E2E de importação passou e os testes focados do mascote passaram 32/32; `art:nex:check` passou. Desktop Wails real sob Xvfb abriu com Core embutido e HUD linkado, respondeu ao clique em Uso/Contas e renderizou um NEX. Captura: `DEV/validation/linux-100-2026-10-25/screenshots/native-usage-import-xvfb/usage-after-import.png`.
- CLI/Desktop/HUD Linux AMD64 estão compilados e instalados. O perfil normal do usuário continua aberto; Desktop PID 22800, HUD PID 22810, unidade `nexus-homologation-final-restored-20261026.service`, `/api/v1/health` HTTP 200 (`1.0.0-beta.10`). Nenhuma sessão do perfil real foi lida no teste Xvfb, executado com HOME isolado vazio.
- Wayland/XWayland do host não oferece saída capturável; alpha/interação nesse compositor e benchmark idle acelerado seguem bloqueados. Recheck posterior mediu 111,5% CPU no WebKit principal em 20 s no perfil normal; Xvfb isolado, tela Uso estável e ponteiro fora da janela, reproduziu 196,1% e 72,4% em dois WebKit após 30 s. Uma amostra por thread atribuiu 96,8% do núcleo à thread principal do WebKit. `strace`/`perf` são bloqueados pelo kernel, então a causa/callsite não foi localizada; rendering por software impede concluir sobre hardware acelerado. Não se afirma vazamento. Soak P9 de 8 h segue ativo; NR-007 composto ainda não passou.
- Artefato estável assinado, provider/agente real, ciclo oficial do updater e proveniência congelada continuam pendentes. Ver `DEV/HANDOFF.md`, relatório e matriz em `DEV/validation/linux-100-2026-10-25/`.

## Snapshot histórico — homologação Linux, 2026-10-25 — P7

- Branch `release/v1.0.0`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; `VERSION=1.0.0-beta.10`. Fingerprint `bb575423508a0c773580804a476b0888ff2449135045a74f80aa6efaffedc6e1` identifica o overlay P7 anterior, não inclui o follow-up posterior do mascote. Worktree modificada e sem fingerprint integral atual.
- Resultado **NO_GO_LINUX**: 97 PASS / 1 FAIL / 15 BLOCKED / 5 NOT_RUN (118). Relatório em `DEV/validation/linux-100-2026-10-25/`.
- Produto tem um mascote NEX. Não há personagem, renderer ou modo Companion 2D separado; `GuardianMascot` e `NexPixelSprite` compõem o único caminho visual.
- Follow-up local: frontend usa agora `mascotModel`/`hud.mascot`, com compatibilidade para preferências e adiamentos legados. Web quality (108/673), build e E2E HUD passaram; ver `DEV/validation/linux-100-2026-10-25/MASCOT_VALIDATION.md`.
- CLI/Desktop/HUD recompilados e instalados; Desktop em XWayland e health HTTP 200 confirmados. O runtime AGY preexistente e seu Web Core foram preservados durante a instalação.
- P2 permanece pendente para restart real da UI, isolamento multi-missão e ACL. P3–P5 passaram os escopos registrados; `ai-cli` real permanece bloqueado por ausência do executável.
- P6 passou matching/fallback/proveniência, identidade efêmera, persistência e limpeza por fixtures. P7 full Go race (82 resultados) e regressões processuais passaram; AGY real retornou READY em um perfil e 429 em perfil sem quota. Handoffs same-provider/cross-provider continuam bloqueados.
- Próxima campanha: P8 — segurança/plataforma/distribuição; P7 permanece parcialmente bloqueado para handoff live. Atenção real, continuidade externa, Wayland físico/alpha, updater assinado e distribuição continuam pendentes.

## Estado

- Produto: IAPro Nexus Workspace OS.
- Versão-fonte neste checkout: `1.0.0-beta.10` (`VERSION`).
- O launcher persiste a identidade autoritativa do `SessionHost` após o
  handshake (`PID`, `HostPID`, `HostGeneration` e fingerprints); isso evita que
  o registro final apague a geração usada pelo attach.
- A instalação local reinicia apenas o `nexus web` associado ao binário
  instalado, evitando hosts Web em memória com executável `(deleted)`.
- Registro histórico de homologação Windows beta.7: branch `release/v1.0.0`, base `52f35c04181b85379219ab55fc3192c63a5440a8`; não descreve o checkout Linux atual. Exact-SHA, Authenticode e trust root permaneciam pendentes naquela campanha.
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

## Estado atual — correções de terminais/agentes — 2026-10-26

- Revisão solicitada para o modo YOLO do Codex e rolagem de OpenCode/Oh My Pi. Correções de código foram feitas em `web/` e no normalizador/launcher Go.
- Ver `DEV/validation/TERMINALS_AGENTS_YOLO_SCROLL_20261026.md` para evidências e status. A instalação beta.11 permanece servindo a interface; ainda não foi reiniciada com o código novo porque mantém um runtime Codex ativo.
