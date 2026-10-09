## Continuidade Windows beta.10 — reteste hosted e TEMP curto — 2026-10-09

O candidato `1.0.0-beta.10` está em `fix/windows-beta10-homologation-20261008`; source SHA local homologado `767be3b168a6d3ec7d223908e21812807486482f`, distinto da beta.7 `d14154d2`. No Windows AMD64 local, Go 1.27.2 vet/teste integral, golangci-lint v2.14.0 (zero issues), web quality (84 arquivos/504 testes), oito E2E browser, gates ConPTY/Named Pipe/runtime-web e smokes Desktop/HUD passaram.

O run hosted `37922899819` confirmou o mesmo SHA e passou Node 22.14.0, Bun frozen install, build web e vet, mas terminou FAIL em `Full Windows suite`; o log detalhado do teste não estava acessível. A falha local `'$GIT_DIR' too big` ocorreu com `TEMP` longo e não comprova causa remota; com `%LOCALAPPDATA%\Temp`, a suíte passou. O workflow agora exporta `TEMP`/`TMP` curtos antes dos testes Go. Próximo passo: commit/push somente na branch candidata e novo run Windows hosted no mesmo source SHA. Não executar Linux nem fazer push direto à release. Signing/trust root e tray físico continuam pendentes; distribuição NO-GO.
# Especificação ativa: IAPro Nexus — Implementação Completa Web + Desktop Multiplataforma

## Atualização de homologação Windows beta.10 — 2026-10-09

Candidato no Windows AMD64: branch `fix/windows-beta10-homologation-20261008`, `VERSION=1.0.0-beta.10`, homologador oficial PASS no source SHA exato `1e3f9a3be446757a3854bcb1948a4325fb271a47`. Go 1.27.2, Bun 1.3.9, Node 22.14.0, Go fmt/vet/vulncheck/lint/test, 84 arquivos/504 testes web, oito suítes E2E, Axe/visual e builds CLI/Desktop/HUD passaram. Smokes nativos Desktop/HUD também passaram no mesmo SHA.

A primeira tentativa isolada de acessibilidade teve `ERR_NETWORK_IO_SUSPENDED`; `test:a11y`, `test:visual` e o homologador integral em worktree limpo passaram depois. A release observada `a6795704` é pai de merge, e a consulta autenticada encontrou zero PRs contra `release/v1.0.0`. O usuário autorizou a execução hosted no SHA atual, com acesso potencial a `native-signing`; ela foi suspensa antes do registro do runner e do dispatch para rotação de credenciais locais. Nenhum runner foi registrado.

Tray físico, renderização visual Desktop, fluxos autenticados, assinatura/trust root e integração por PR seguem pendentes. Checksums locais são de builds não assinados; distribuição NO-GO. Evidências: `../validation/current/BETA10_WINDOWS_HOMOLOGATION_20261009.md` e `../validation/local-beta/1.0.0-beta.10/1e3f9a3be446757a3854bcb1948a4325fb271a47/windows-AMD64/`. Sem gate Linux.

A branch de homologação precisa de PR para release/v1.0.0, sem push direto à release. CI Windows hosted no SHA integrado, interação real do tray/autostart, smokes autenticados/interativos restantes, assinatura/trust root e instalador continuam pendentes. Este candidato é beta.10, não beta.7/d14154d2; distribuição segue NO-GO.

## Programa ativo — Nexus 1.0 Product Closure — 2026-09-20

Baseline canônico: `main@7605f702cce220b4111d51d06a34b37f7e0a9a8d`, merge do PR #3 após
CI #149 totalmente verde. A partir deste ponto, o fechamento 1.0 é regido por
[`DEV/SPECS/NEXUS_1_0_CLOSURE.md`](./NEXUS_1_0_CLOSURE.md).

A campanha preserva as autoridades existentes (AgentSpec, ResourceScheduler,
Mission runner, IntelligenceMemory, Capacity Monitor, SessionHost/TerminalsHost,
Attention/interventions, entitlement boundary e integração opcional com Maestro)
e proíbe subsistemas paralelos. A ordem de execução é: truth alignment →
production Capacity probes → Agent Modes + canonical AgentMatcher → Skills v1 →
Cost Intelligence/budgets → Unified Work Mode → Mission certification/Evidence →
Memory 1.0 → Maestro contract v1 → commercial + native release certification.

Nenhum item histórico abaixo substitui o contrato atual desta seção.

## Limite do registro histórico

Tudo abaixo deste marcador é preservado como histórico datado. Termos como
“ativo”, “current”, “candidate” ou versões beta usados nos títulos/textos abaixo
descrevem o estado observado naquela data e **não** substituem a verdade atual.
Para estado de release use exclusivamente
[`docs/release-status.md`](../../docs/release-status.md). O arquivo
[`DEV/validation/current/RELEASE_STATUS.md`](../validation/current/RELEASE_STATUS.md)
é ledger de evidências históricas e aponta de volta para essa fonte atual.
Para arquitetura/escopo histórico do fechamento 1.0 use
[`DEV/SPECS/NEXUS_1_0_CLOSURE.md`](./NEXUS_1_0_CLOSURE.md); políticas de
versão/plataforma/publicação nele registradas não substituem os workflows e o
status de release atuais.

## Slice ativo — proteção permanente contra falhas de teclado — 2026-09-18

O terminal supervisionado/direct usa um controlador único de TTY que reaplica
reset seguro na entrada, após interrupção/perda de conexão e antes da
restauração. O attach valida geração, PIDs e identidade dos binários; sessões
antigas são marcadas `STALE` e recebem ação explícita de recriação, sem matar o
processo silenciosamente. `nexus doctor terminal [--json|--recover]` expõe o
estado e a recuperação. Filtros Web removem apenas metadados conhecidos,
preservando setas, Ctrl+C, colagem, UTF-8 e comandos literais.

## Beta Linux closure scope — 2026-09-18

The current closure candidate is locally certified for Linux only. GitHub-hosted
jobs are blocked before start by billing state and no Windows/macOS machines
are available. The executable beta CI contract is seven same-SHA Linux jobs:
frontend, security, browser, Linux Go, Linux Wails, GoReleaser/DEB snapshot
and DEB install smoke. Native Windows, macOS, NSIS and RPM evidence remain
pending and are not release claims.

## Closure update — 2026-09-17 — PR #2

Os blockers determinísticos identificados no Windows foram corrigidos no
worktree: seed cross-account em lote e sincronização determinística do worker
de recuperação de Mission. O CI agora evita a duplicação de `push`/`pull_request`
em branches de feature. Permanecem gates hosted/native: novo SHA commitado,
14 jobs verdes, Windows repetido verde, merge commit, CI da `main` e retag
`v1.0.0-beta.1`. Branch protection nativa permanece indisponível no plano
privado e será mitigada operacionalmente.

## Closure state — 2026-09-15 (v1.0.0-beta.1)

Alvo desta spec: Community Preview `v1.0.0-beta.1`, não 1.0.0 estável.
Slice de fechamento in-repo: lint, attach Composer, docs de install, changelog.
Gates humanos restantes: freeze commit, CI same-SHA, `beta-signing`, visibilidade
do repositório, tag no SHA verde, GitHub Release. Composer → Flow é o caminho Web
de planejamento; Conductor permanece CLI.

## Closure state — 2026-09-13

### P0 ativo — concorrência de runtimes supervisionados

O contrato de continuidade agora permite múltiplas TUIs Codex no mesmo perfil:
leases compartilhados protegem cada runtime, enquanto preparação, migração e
quota app-server exigem manutenção exclusiva. A seleção não trata perfil ativo
como singleton e Leads interativos são reivindicados atomicamente antes do
lançamento. O slice foi implementado e verificado localmente; a migração do
module path e a promoção adaptativa permanecem bloqueadas até revisão deste P0.

Após a revisão local, a migração do module path foi concluída e validada; a
promoção adaptativa (Fase A) continua pendente.

Observed candidate SHA: `eed65e5a7e46adc896165c94accb7fc73fc6750f`. The local
closure slice is implemented and its deterministic/Linux gates pass, but the
candidate is not frozen. Required release blockers remain: stale visual docs
manifest, production updater key provisioning, authenticated provider Mission
evidence, native Windows/macOS/installer execution, hosted same-SHA evidence
and overnight soak. Verdict remains `NOT_READY_FOR_OVERNIGHT_CERTIFICATION`.

## Slice ativo — Mission Teams / Autonomous Delivery Pipeline — 2026-09-13

Esta campanha implementa uma projeção data-driven de Mission Team sobre os
contratos canônicos existentes de Intent, Delegation, WorkPlan, MatchAgents,
ResourceScheduler, MissionRunner, ContextCapsule, WorkReceipt, ValidationEvidence
e Attention. O objetivo é selecionar DIRECT para trabalho pequeno e uma pipeline
explicável PLAN → BUILD → REVIEW → FIX → VERIFY para trabalho substancial, sem
criar outro scheduler, Mission engine, WorkPlan ou Agent model.

Contrato executável e critérios estão em
`../../.omx/plans/2026-09-13-mission-teams.md`; o relatório final ficará em
`DEV/validation/current/MISSION_TEAMS_REPORT.md`. Provider autenticado, native
platform e overnight evidence continuam sujeitos às limitações reais do
ambiente e serão classificados separadamente, nunca inferidos de fake fixtures.

## Slice de documentação orientada ao produto — 2026-09-13

README como landing page técnica, Quick Start, mapa de funcionalidades,
referência de CLI/API local/configuração, personas, troubleshooting, evidência
visual sintética e ledger/coverage. Não muda o status de release: provider
autenticado, Desktop nativo e Mission real continuam condicionados à evidência
dos relatórios de validação.

## Slice da branch `feat/nexus-auto-delegation` — 2026-09-12

O contrato específico desta branch está em
[`docs/superpowers/specs/2026-09-12-nexus-auto-delegation.md`](../../docs/superpowers/specs/2026-09-12-nexus-auto-delegation.md).
O objetivo é adicionar a política AUTO/ASK/OFF para o Lead Agent usar os
contratos existentes de Flow, WorkPlan, MatchAgents, scheduler e MissionRunner.
Não altera o veredito de release/hardening desta especificação.

## Slice ativo — registry progressivo de CLIs (2026-09-10)

O Core agora separa descoberta de instalação (`InstallationRegistry`) do
registro de contas. `nexus providers status [--json]` expõe os estados e
`nexus providers register` cria perfil isolado, com autenticação imediata ou
adiada via `--no-login`.

## Complemento de execução planejado — 2026-09-06

O plano solicitado para execução posterior pelo Luna está em
[`NEXUS_TERMINAL_CONTINUITY_LUNA.md`](NEXUS_TERMINAL_CONTINUITY_LUNA.md).
Cobre CI, quotas, pools, supervisão CLI e continuidade multiplataforma; adiciona
pacotes P0–P12 sem substituir o contrato Web/Desktop abaixo. Status: planejado,
não implementado nesta etapa. Maestro permanece instalação opcional.

## Objetivo

Tornar oficialmente o IAPro Nexus uma aplicação com duas superfícies de execução equivalentes:
1. Web (acessada pelo navegador via `nexus web` / `nexus serve`)
2. Desktop (aplicação nativa Wails v2 estável para Windows, macOS e Linux)

Ambas utilizando:
- O mesmo frontend React (`web/src`)
- O mesmo Nexus Core Go (`internal/app`, `internal/control`, `internal/core`, `internal/nexus`)
- Os mesmos contratos REST e WebSocket
- A mesma arquitetura de terminal (PTY / ConPTY -> WebSocket -> xterm.js)
- O mesmo Update Service unificado (`internal/update`) com suporte a InstallationMethod
- A mesma versão Nexus, canal e Git SHA
- Integração Maestro desacoplada e 100% opcional (sem instalação automática silenciosa)

## Complemento ativo — separação terminal/controle (2026-09-11)

O canal de entrada do terminal é byte-transparente e não reconhece comandos
Nexus. Ações de status, eventos, stop, handoff, continue e detach usam RPCs
tipados; `nexus control monitor [runtime-id]` é uma superfície TUI separada,
somente leitura. Reconnect não relança runtimes; recuperação exige ação
explícita.

## Aceitação

### Finalization autopilot status — 2026-09-07

The fresh release-candidate ledger is maintained in
[`DEV/NEXUS_1_FINAL_ACCEPTANCE.md`](../NEXUS_1_FINAL_ACCEPTANCE.md). Local
control-plane gates are green; native Windows/macOS, authenticated crash/provider
recovery, overnight acceptance and real dogfooding remain explicit blockers.

1. **Baseline Zero-Red**:
   - `make quality`, `web-verify`, `go test ./...` 100% verdes.
2. **Core Lifecycle & Discovery**:
   - Ciclo de vida desacoplado e reutilizável (`Core` / `ControlCenter`) inicializável tanto por CLI/Web quanto pela shell Desktop.
   - Descoberta e autenticação loopback segura via token/descritor de runtime.
3. **PlatformBridge Frontend**:
   - `PlatformBridge` abstrato em `web/src/platform/` consumido pela UI React.
   - `WebBridge` (browser) e `DesktopBridge` (Wails) sem `window.wails` solto no código.
   - Suporte nativo a seletores de arquivo/pasta, notificações nativas, deep links (`nexus://`), autostart, window management e system theme.
4. **Wails Desktop Shell**:
   - Entrypoint em `cmd/nexus-desktop/main.go` e pacote `internal/desktop/`.
   - Sem duplicação de React, runtime, PTY ou updater na shell Wails.
5. **Update Service Unificado**:
   - `internal/update/` gerencia manifestos assinados Ed25519, SHA256, canais e `InstallationMethod` (STANDALONE, NSIS, DEB, RPM, HOMEBREW, etc.).
   - Instalações gerenciadas por pacotes não sobrescrevem binários indevidamente.
   - CLI e Desktop UI utilizam exatamente o mesmo serviço.
6. **Maestro Isolado & Opcional**:
   - Removida instalação silenciosa do Maestro em `install.sh` e `install.ps1`.
   - Flag explícita `--with-maestro` / `-WithMaestro` nos instaladores.
   - Modo degradado gracioso (`MAESTRO_DEGRADED`) quando ausente, sem falhar o Nexus.
   - Separação clara de status e atualização entre Nexus e Maestro em Settings > Integrations e CLI.
7. **Suporte Multiplataforma**:
   - Windows (ConPTY, WebView2, NSIS), macOS (PTY, WKWebView, DMG/app), Linux (PTY, WebKitGTK, DEB/RPM).
   - Matriz de compatibilidade documentada e `nexus doctor` expandido com diagnósticos de desktop e SO.
8. **Documentação & ADRs**:
   - ADRs de Desktop (Wails v2), Update Architecture e Maestro Integration.
   - Relatórios em `DEV/validation/` e `docs/superpowers/reports/`.
Decisões executivas e pendências de promoção: `DEV/DECISIONS/NEXUS_TERMINAL_CONTINUITY.md`.

## Nexus Core consolidation status — 2026-09-10

## Final closure campaign — 2026-09-11

The current campaign audit and executable plan are tracked in
`DEV/validation/FINAL_CLOSURE_REALITY_AUDIT.md`,
`DEV/validation/FINAL_CLOSURE_CHECKPOINTS.md`, and
`docs/superpowers/plans/2026-09-11-nexus-final-closure.md`. The first native
Skill Catalog, bounded intelligence grounding, explicit intent router, runtime
affinity/routing decision slices, and runner integration with the canonical
ValidationEvidenceStream are implemented and locally tested. The global
release verdict remains NO-GO while an authenticated Mission stream instance,
complete live provider execution, overnight acceptance, and native
Windows/macOS validation are unavailable.

## Evolution corrective closure — 2026-09-11

The independent audit and corrective work are tracked in
[`docs/validation/EVOLUTION_FINAL_AUDIT.md`](../../docs/validation/EVOLUTION_FINAL_AUDIT.md)
and [`docs/validation/EVOLUTION_FINAL_VALIDATION.md`](../../docs/validation/EVOLUTION_FINAL_VALIDATION.md).
Persistent Agents now carry typed `AgentSpec` personality metadata from the
Web presets into revision config and compiled execution context. The canonical
`MatchAgents` path is used by Flow/Mission AUTO allocation, while provider
resource capabilities remain owned by `ResourceScheduler`. Natural-language
`nexus run "<goal>"` creates a classified Flow/WorkPlan and enters the same
pipeline. A same-provider handoff reports `NATIVE_RESUME_UNVERIFIED` unless a
provider-level confirmation exists; cross-provider handoff remains
`CONTEXT_HANDOFF`.

Local Linux gates are green in isolation, but the final verdict remains
`NO-GO` until required PTY/failover E2E, authenticated live-provider evidence,
same-SHA native Windows/macOS evidence and the full race gate are available.

### Final closure implementation continuation — 2026-09-11

The active contract now includes the source-agnostic SkillCatalog in the Agent
prompt path, generic `ExecutionGuidance` for Intelligence, canonical frontend
`skill_ids`, separation of legacy process gates from Skills, and persisted
desired-vs-actual Mission routing explainability. These are locally verified;
they do not waive the release evidence gates below.

Release remains `NO-GO` until an authenticated Mission produces a durable
ValidationEvidenceStream instance, live provider failover/PIN/model escalation
and semantic handoff are proven, native Windows/macOS runs exist for the same
SHA, and the overnight scenario reaches a verified terminal state.

The current continuation also records operational Project Intelligence facts
(commands, frameworks, workspace topology and CI run commands) through the
existing bounded scanner, and the run routing report projects the persisted
intent decision alongside durable allocation decisions. These local contracts
are tested, but do not replace authenticated/runtime/native evidence.

## Account isolation slice — 2026-09-10

Introduzido `model.AccountScope` com chave canônica
`provider/account_id/identity_version`. Perfis persistem o identificador
opaco e versionam mudanças de identidade; APIs scoped de quota recusam cache
sem correspondência. Cooldown, monitor de quota e eventos aceitam o mesmo
escopo. Registro progressivo de instalações permanece como próximo slice.

- Characterization baseline and capability matrix are versioned under
  `docs/refactoring/`.
- Agent identity remains separate from provider. Typed `AgentSpec` is persisted
  in `AgentRevision.Config` and normalized for legacy role-only Agents.
- The existing intelligence compiler now exposes provenance sections and is
  reused by custom direct, headless, and WorkPackage/Mission execution.
- CLI `plan compile`, `plan run`, and positional `agents PROJECT` behavior are
  wired/tested. Full CLI registry generation remains a documented follow-up
  because provider-direct commands retain native parsing.

## Consolidação arquitetural incremental — 2026-09-10

Esta campanha não adiciona uma leva de funcionalidades. A primeira fatia
concluída modulariza a composição das rotas HTTP e formaliza metadata aditiva
da API, preservando handlers, paths, CLI aliases e o contrato Web existente.
O plano completo e a baseline estão em `docs/superpowers/plans/` e
`docs/refactoring/`; os principais domínios HTTP já possuem arquivos próprios
de transporte, e extrações futuras devem ser guiadas por domínio e testes.

Projects, Resources, Agents, WorkPlan CRUD, Missions planning, Composer e o
boundary de transporte de MissionRun já possuem services de aplicação no Core,
com testes de cancelamento/CRUD/delegação e preservação dos contratos HTTP.
Correlation IDs opcionais já atravessam eventos duráveis nos fluxos de failover
e handoff. A campanha continua ativa: não declarar a Definition of Done global
enquanto a matriz completa de contratos API/CLI, a taxonomia integral de
eventos e as evidências nativas de Windows/macOS permanecerem pendentes.

Atualização: `RuntimeApplicationService` agora concentra também os controles
de runtime usados pela API (stop/respond/handoff), cleanup e espera cancelável;
`RunApplicationService` produz eventos correlacionados de MissionRun com
ownership de projeto/agente quando disponível. O lifecycle/process state ainda
é implementado pelo control plane e runner, conforme documentado na auditoria.

## Validação Windows Beta 7 — estado 2026-10-07

Project Hub passou na regressão 775×483, e gates web/E2E e smokes Windows passaram no worktree 52f35c04-dirty. Certificação de release aberta: exact-SHA d14154d2 não é HEAD; worktree sujo; signing/update trust root ausentes; a auditoria viva de branches não encontrou ramo Beta 7 novo nem delta de correção/teste pendente para integrar. Ver DEV/validation/current/BETA7_WINDOWS_HOMOLOGATION_20261007.md.
