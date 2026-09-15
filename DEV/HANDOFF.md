# Handoff

## Resolução de Autenticação e UX do OpenCode — 2026-09-15

## Atualização — filtro de entrada do Codex

O filtro compartilhado do terminal agora remove relatórios de foco
parametrizados/corrompidos (`ESC[0[I` e equivalentes), que apareciam como
`0[I` no prompt. `Ctrl+V` continua sendo encaminhado. O binário foi
recompilado e instalado em `/home/desenvolvedor/.local/bin/nexus`.

Sessões Codex já abertas precisam ser encerradas e iniciadas novamente para
usar o bundle atualizado.

Resolvida a incompatibilidade arquitetural de autenticação do OpenCode no Nexus, adaptando tanto o backend Go quanto o Web OS para a realidade dos múltiplos modos operacionais do OpenCode:

1. **Modos de Operação Reconhecidos**:
   - **Modo Local / Self-Hosted (Ollama / Env)**: Quando nenhum provedor específico está configurado, o OpenCode é tratado como `HEALTHY` e `AUTHENTICATED`, com plano `"OpenCode Local (Ollama/Env)"`, dispensando logins ou chaves de nuvem e permitindo execução imediata.
   - **Provedores Externos (DeepSeek, Claude, OpenAI)**: O `InspectAuth` analisa o mapa real do `auth.json` (`map[string]openCodeAuthEntry`). No ambiente local do Maestro, identificou com sucesso o provedor `"deepseek"`, marcando a conta como `"OpenCode (deepseek)"` com status `healthy`.
   - **OpenCode Zen / Go**: Identificado quando a chave `"opencode"` está presente em `auth.json`, apontando para a nuvem oficial do OpenCode AI.

2. **Herança de Credenciais & Login CLI**:
   - `Prepare`: Sincroniza o `~/.local/share/opencode/auth.json` do host para o sandbox isolado do perfil caso ainda não exista.
   - `nexus login opencode <profile>`: Dispara interativamente `opencode providers login` no terminal, permitindo que o usuário gerencie e adicione provedores de IA diretamente.

3. **Web OS (UX & Modais)**:
   - `UsageAccountCard`: Eliminado o falso alerta de "Requer Login"; conta do OpenCode exibe chips contextuais e ação primária "Abrir Terminal" para lançamento imediato.
   - `AccountLoginModal`: Modal especializado explicando os 3 modos de operação, destacando o provedor ativo (`DeepSeek`), e oferecendo botões diretos para "Gerenciar Provedores no Terminal" e "Usar Modo Local".
   - Internacionalização completa em `en`, `ptBR` e `es`.

4. **Verificação**:
   - 7/7 testes unitários do adapter OpenCode PASS.
   - `make web-verify` 10/10 gates PASS.
   - Binário compilado e instalado em `/home/desenvolvedor/.local/bin/nexus`.
   - Servidor web em background atualizado e validado na porta 13000.



## Fechamento v1.0.0-beta.1 — 2026-09-15

Worktree de freeze isolado: lint Linux, attach Composer, install URLs,
changelog, OpenCode auth, filtro CSI do terminal, folga do lock Codex no
Windows. Spike de Conversation está no stash
`wip: conversation spike parked outside v1.0.0-beta.1` e **não** entra nesta
tag. **Não** houve commit, tag, push nem publicação. Próxima ação humana:
autorizar freeze commit (sem o stash), CI 14/14, `beta-signing`, repo público
(ou GH_TOKEN), retag no SHA verde, GitHub Release, depois merge em `main`.
Ver `DEV/validation/current/RELEASE_STATUS.md`.

## Soak de 8h concluído — 2026-09-15

O soak executado no processo original terminou com **PASS**: `go test`
reportou `28800.028s` (8h), sem falha. Permanecem separados os gates que
dependem de CI no mesmo SHA, plataformas nativas e credenciais reais.

## Modernização das Superfícies de Sessões, Agentes e Project Overview (Opções 1, 2 e 3) — 2026-09-15

As três superfícies selecionadas pelo Maestro foram totalmente modernizadas no Web OS com paridade CLI, novos componentes, estilização via SCSS Modules e integração completa com a telemetria do Nexus:

1. **Superfície de Sessões & Histórico (`/sessions`)**:
   - Strip superior de métricas agregadas com 4 cards estatísticos: Total de Sessões, Fallbacks Registrados, Rate Limits e Agentes Ativos.
   - Alternador de abas:
     - **Histórico de Execuções**: Timeline cronológica de eventos com timestamp relativo e absoluto, provedor, perfil, workspace, duração da execução e status.
     - **Gerações dos Agentes**: Visualização dos prompts e gerações recentes dos agentes, tokens consumidos e metadados.
   - Filtros instantâneos: Busca em tempo real e pills de seleção rápida por provedor (`agy`, `codex`, `opencode`, etc.).
   - Paridade com `nexus stats` e `nexus history`.

2. **Superfície de Agentes (`/agents`)**:
   - Barra de resumo interativa com contadores clicáveis por estado (Todos, Em Execução, Em Espera, Degradados, Parados).
   - Cards modernos com indicador pulsante animado (`agent-pulse`) quando em estado `WORKING`.
   - Badges de continuidade de contexto, tags de perfil vinculado e metadados técnicos.
   - Painel de templates e perfis recomendados com roles pré-configuradas (`developer`, `architect`, `reviewer`, `tester`) para rápido provisionamento.

3. **Superfície de Project Overview (`/overview`)**:
   - Seção de Runtimes e Processos Supervisionados ao vivo (`nexus ps`), contextualizada ao workspace do projeto.
   - Cards de runtime com pulso verde em tempo real, ID do runtime, PID, provedor/perfil, tempo de vida e ações imediatas ("Anexar Terminal" e "Parar").

4. **Backend Go & Endpoints de Telemetria**:
   - `GET /api/v1/telemetry/stats`: Retorna estatísticas de sessões, fallbacks e rate limits agregadas por provedor e perfil via `telemetry.ComputeStats`.
   - `GET /api/v1/telemetry/history`: Retorna lista cronológica de eventos de sessão e prompts via `telemetry.ReadRecentEvents`.
   - Suporte a autenticação web e proteção CSRF.

5. **Compilação e Instalação Local**:
   - 10/10 gates em `make web-verify` PASS.
   - 378/378 testes Vitest PASS.
   - Binário `./nexus` compilado com embedding atualizado e instalado em `/home/desenvolvedor/.local/bin/nexus`.
   - Servidor local rodando em background na porta 13000 (`http://127.0.0.1:13000`).
   - Sem commits e sem push conforme constituição do Orquestrador.


## Compilação e Instalação Local Concluídas — 2026-09-15

O binário unificado Nexus foi compilado com todos os novos recursos e instalado localmente com sucesso:
1. **Frontend Web & Embedding**:
   - `UsageSurface` modernizada com grid responsivo, dynamic status chips, barras de cota progressivas, menus de ação contextual e paridade completa de comandos (`login`, `logout`, `use`, `register`, `explain`).
   - Modal de adição de perfil na Web e `ModalAddProfile` no TUI interativo (`nexus usage`).
   - Assets empacotados via `node scripts/build.mjs` e embutidos via `go:embed`.
2. **Instalação**:
   - Binário instalado em `/home/desenvolvedor/.local/bin/nexus` (resolvido por `which nexus`).
   - Versão ativa: `IAPro Nexus 1.0.0-beta.1 (linux/amd64, go1.25.0) commit 2b9c956 built 2026-09-15T03:22:18Z`.
   - Servidor local `nexus web` reiniciado em background na porta 13000 (`http://127.0.0.1:13000`).
3. **Estado do Repositório**:
   - Sem commits e sem push conforme constituição do Orquestrador.



Implementada a funcionalidade para permitir adicionar perfis/provedores diretamente a partir da tela de uso, tanto no TUI (`nexus usage`) quanto no Web OS:
1. **TUI Bubbletea (`internal/tui/usage_table.go`)**:
   - Atalho `⏎` (Enter) em linhas sem perfil (como `cursor (sem perfil)`): abre o diálogo modal interativo `ModalAddProfile`, eliminando o aviso passivo anterior.
   - Atalho `a` / `A`: abre o `ModalAddProfile` para qualquer provedor selecionado na tabela, com input focado e placeholder `default`.
   - Tecla `Esc`: fecha o modal e retorna à navegação normal da tabela.
   - Confirmação com `Enter`: encerra a interface Bubbletea retornando `ActionAdd`, transferindo o controle para o CLI `internal/app/app.go` executar `addCmd(args)`. O processo cria o perfil e engaja o fluxo oficial interativo de autenticação no terminal pai sem travamentos de PTY.
   - Linha de preview atualizada para indicar `⏎/a adicionar perfil <provider>`.
   - Rodapé exibe `a add` no menu geral de atalhos e `⏎/a adicionar perfil` em linhas sem perfil.
2. **Web OS (`UsageAccountCard.tsx`, `UsageSurface.tsx`, `RegisterProviderModal.tsx`)**:
   - `UsageAccountCard`: menu `...` inclui a ação contextual "Adicionar outro perfil", disparando o modal com o provedor alvo pré-selecionado.
   - `RegisterProviderModal`: suporta `initialProvider` para pré-selecionar o provedor e focar no nome do perfil.
   - Banner de CLIs detectados na máquina pré-seleciona o provedor automaticamente quando há apenas 1 instalado.
3. **Verificação**:
   - `go test ./internal/tui/...` (incluindo 4 novos testes cobrindo o modal, atalhos, submissão e cancelamento) PASS.
   - `go test ./internal/app/...` PASS.
   - `make web-verify` 10/10 gates PASS.
   - `make build` PASS e binário instalado em `~/.local/bin/nexus`.
4. **Sem commits/push**: Preservada a árvore de trabalho conforme as regras de persistência.

## Redesign da Superfície de Uso (UX/UI, Responsividade & CLI Parity) — 2026-09-15

A tela de Uso de Provedores e Contas (`UsageSurface`) foi reformulada com foco em excelência de UX/UI, responsividade em todas as resoluções e paridade integral com os comandos do Nexus CLI:
1. **Design & Responsividade**: CSS Grid responsivo com autofill e minmax adaptativo (`UsageSurface.module.scss`), eliminando sobreposição em telas pequenas e maximizando aproveitamento do espaço em telas ultrawide.
2. **Cards de Conta (`UsageAccountCard`)**: Visual modernizado com chips semânticos compactos, indicadores visuais de integridade, barras de progresso com tons semânticos (verde, azul, âmbar, vermelho) e menu de ações `...` com todas as operações da conta.
3. **Paridade Nexus CLI**:
   - `nexus login`: Modal interativo com instruções detalhadas por CLI de IA (Codex, AGY, Claude, Gemini, OpenCode, Cursor), comando para cópia rápida com feedback, atalho de terminal e verificação imediata de autenticação.
   - `nexus logout`: Endpoint e ação de revogação de sessão e remoção segura de credenciais.
   - `nexus use`: Troca do perfil padrão da frota com recarga em tempo real.
   - `nexus providers register`: Modal dedicado para descoberta de provedores instalados e registro de novos perfis.
   - `nexus usage`: Atualização granular de cota por conta individual além do refresh global da frota.
   - `nexus explain`: Modal de explicabilidade do algoritmo de recomendação e agendamento de recursos (pesos de fallback, janelas de cota, custos de troca, continuidade de sessão).
4. **Filtros e Descoberta**: Faixa de métricas clicável (`UsageSummaryStrip`) para filtrar instantaneamente por Total, Autenticadas, Requerem Login, Rate Limit e Saudáveis; barra de busca em tempo real com debounce; pills de provedores detectados; banner de sugestão para provedores de IA instalados na máquina mas ainda não registrados.
5. **Gates de Qualidade**: 100% de aprovação em `make web-verify` (10/10 gates), `npm run quality:full` (388/388 testes Vitest, 0 erros no typecheck e lint, build limpo) e `go test ./internal/control/web/...`.
6. **Próximos passos**: Revisão visual pelo Maestro no navegador (`127.0.0.1:13000`), sem commit/push conforme a constituição de trabalho.

## Limpeza de galhos — 2026-09-15

HEAD local: `356f1a3` (feat + helper de teste terminal), ahead 1 de
`origin/feat/nexus-maximum-delivery`. Staged do usuário (5 arqs de release
`1.0.0-beta.1`) preservado. Galhos locais: só `feat/nexus-maximum-delivery` e
`main` (atrás 117 de `origin/main`); resto arquivado em
`archive/2026-09-15/*` + `/tmp/nexus-branches-backup-2026-09-15.bundle`.
Gates verdes: host tests, `make quality`, `make build`, `make web-verify` 10/10.
Próxima ação: commit/push só com autorização; fast-forward de `main` separado;
slices futuros (`capacity-monitor`, `globalSearch`) exigem spec antes de
reimplantar. Sem push/CI hosted nesta execução.

## Remediação do review pré-main — 2026-09-14

Foram corrigidos os failures locais de lock/Windows e a persistência do fluxo
`--lead`. O Registry agora recebe Agent/Project/label por mutação atômica;
falhas de preparação não degradam silenciosamente para um runtime sem Lead.
O discovery de orphan hosts valida o executável Nexus antes do cleanup.
O failover de perfil preserva o contexto do chamador ao consultar o scheduler.

Verificação atual: `go test ./...`, `go test -race ./...`, `make quality`,
`make test-e2e`, `make security`, `GOTOOLCHAIN=go1.25.1 make release-proof`,
lint frontend sem warnings e compilação cruzada Windows dos testes afetados
passaram. `make docs-verify` permanece bloqueado até o manifesto visual ser
regenerado sobre um SHA commitado.
Commit, push, CI hosted same-SHA, rulesets, branch protection, soak de 8h e
provas nativas/live continuam fora desta execução.

## Codex terminal input — 2026-09-14

Rebuilt `/home/desenvolvedor/.local/bin/nexus` from checkout `1b840ca` (`nexus version` `1b840ca`, `go version -m` `github.com/IAPro-Community/nexus`); automated gates PASS (focused web/Go race, `go test ./...`, `make quality`, `make build`, `git diff --check`). Resta apenas aceitação manual Web/Desktop + `nexus codex`. Sem push/ruleset/branch-protection.

## Plano de concorrência e prova Codex — 2026-09-14

O contrato P0 de runtimes concorrentes, a migração de module path, a promoção
adaptativa e o ciclo canônico de revisão estão implementados neste worktree.
A prova real Codex `run_06G9VJV1K6J649PN2EMEHR1EMG` alcançou
`COMPLETED_VERIFIED`: routing autenticado `codex:kivergmail`, worktree
dedicado, reviewer Agent distinto em sandbox `read-only` e verificações de
pacote e globais aprovadas. O prompt de review usa JSON válido e o parser lida
com o transcript duplicado emitido pelo CLI.

Continuam condicionados à autorização explícita: commit/SHA congelado, push,
CI hosted same-SHA, rulesets/branch protection e provisionamento da chave de
assinatura. `make docs-verify` segue bloqueado pelo manifesto visual stale;
Windows/macOS nativos e soak de 8h permanecem não verificados. Não houve
commit, push ou alteração remota.

## Fase A — promoção adaptativa — 2026-09-13

Implementado o pacote `internal/nexus/promotion` com sombra de linha sem
vazamento de sequências de terminal e política `AUTO`/`ASK`/`OFF`. O attach do
CLI e o terminal Web retêm Enter, preservam o stream bruto e, quando elegível,
criam Mission autônoma no mesmo Lead; o Web usa frames `promotion` e resposta
explícita internacionalizada. Verificação local completa passou. Push e
proteções remotas continuam fora desta execução.

## Fase B — revisão com remediação — 2026-09-13

O teste canônico agora cobre `CHANGES_REQUESTED` com três findings, nova rodada
com um finding aberto e um novo, aprovação e Verifier read-only. O ledger de
findings e o estado final `COMPLETED_VERIFIED` passaram; `make overnight-smoke`
inclui o fluxo.

## Fase C — module path IAPro-Community — 2026-09-13

Module path migrado para `github.com/IAPro-Community/nexus`, incluindo imports,
ldflags, Makefile, GoReleaser, CI e documentação; ADR criada. `go mod tidy`,
`go test ./...`, `make quality` e `git diff --check` passaram. Sem push/commit.

## P0 concorrência de runtimes — 2026-09-13

Implementado no worktree (sem commit): Codex agora aceita leases TUI
compartilhados no mesmo `CODEX_HOME`; manutenção/app-server usam lock exclusivo,
e `Prepare` reaproveita perfis prontos sem migrar durante uma TUI ativa. O CLI
supervisionado não filtra mais perfil ocupado nem retorna singleton; Leads são
reservados atomicamente via `STARTING`/`RowsAffected`, com cleanup em falha.

Verificação: testes focados e `go test ./...` PASS; `go test -race` nos pacotes
tocados PASS; compilação Windows dos pacotes alterados PASS; `git diff --check`
PASS; `make quality` PASS. O CI remoto/push não foi executado. Próxima ação:
revisão do contrato e só então executar C (module path) e A (promoção adaptativa),
sem push/branch protection sem autorização explícita.

## Provider contract freeze — 2026-09-13

Blindagem de CLIs / `nexus usage` / roteamento está no worktree (ainda sem
commit pedido):

- `internal/core/provider/spec` — Spec único + goldens por provider
- `quota.EvaluateEligibility` consumido por scheduler, recommend, routing e validação
- Quota live Claude/Gemini; OpenCode Zen health→UNSUPPORTED; Cursor UNSUPPORTED
- Failover tipado (`FailureKind`); `EventQuotaFailoverCompleted` pós-sucesso
- `nexus usage --json` inclui `quota_view` + `eligibility`
- Guardrails anti-substring / anti-LookPath literal; `make smoke-providers`
- Docs: `docs/architecture/provider-contract.md` + ADR freeze

Validação local focada: `go test` em `provider/...`, `quota`, `nexus`, `app`,
`profile`, drivers/flags PASS nesta máquina. Commit/push só sob pedido.

## Pós PRODUCTION_NO_GO — Fase 0 código — 2026-09-13

Fase 0 (código) fechou os P0 de falso sucesso do red team
`PRODUCTION_NO_GO` / `CERTIFICATION_INVALID_FOR_CURRENT_HEAD`.

Feito no worktree (ainda **sem** SHA congelado até o commit desta sessão):

- Maestro default `OFF`; ASSIST degrada; branding Web sem Maestro obrigatório
- PATCH de status terminal de Mission → 409; DoD global na árvore integrada
- `nexus run` / `nexus plan run` autônomos (worker); UNKNOWN inelegível; erros tipados
- Kill pós-handshake; Stop/StopAgent force-kill; watchdog cancela `opCtx`
- Installers desktop exigem manifesto Ed25519; CI desktop Linux/Windows/macOS
  fail-closed sem `NEXUS_UPDATE_PUBLIC_KEY`

Gates locais: `make quality` + `go test -race` nos pacotes tocados + smoke isolado
(`maestro_mode=OFF`; `nexus run` sem “quota/rate limit”) PASS nesta máquina.

**Ainda não é GO.** Não inventar chave Ed25519. Sem push até
`vars.NEXUS_UPDATE_PUBLIC_KEY` existir. Overnight/CI same-SHA/Mission autenticada/
soak 8h = Frentes B/C em pasta de evidência **nova** (não reusar `e53f835`).

Relatório NO-GO do SHA antigo:
[`validation/production-go-no-go/FINAL_REPORT.md`](validation/production-go-no-go/FINAL_REPORT.md).

## Current closure handoff — 2026-09-13

Observed SHA: `eed65e5a7e46adc896165c94accb7fc73fc6750f`. The worktree is not
frozen and no commit/push was performed. Local implementation and quality
gates are green, including Mission classification/failover safeguards,
runner evidence gating, failed-launch cleanup, updater trust-root checks,
Linux Wails packaging, and `/e[...]` byte-preservation regressions.

Do not promote this to release readiness: `make docs-verify` is blocked by a
stale visual manifest, and authenticated provider Missions, production signing
keys, native Windows/macOS/NSIS, hosted same-SHA evidence and 8-hour soak are
still unverified. Re-run the complete matrix after freezing one clean SHA.

## Mission Teams / autonomous delivery pipeline — 2026-09-13

The current worktree contains the Mission Team product layer as an additive
projection over the existing Flow → WorkPlan → MissionRunner path. It selects
DIRECT for small work and persists role-based phases for substantial work,
including planning, implementation fan-out, review, automatic remediation and
verification. Agent identity remains separate from desired/actual provider
routing. Review findings are structured, stable, deduplicated and bounded;
phase state, handoffs, receipts and routing are restart-safe projections.

The canonical fixture `TestMissionTeamPlannerBuilderReviewerVerifierPipeline`
and `make overnight-smoke` are green. Full Go tests, race, vet, security,
build, desktop build, E2E and Attention E2E are green. `make coverage` and
`make release-proof` remain blocked by the installed Go 1.25 toolchain's
missing `covdata`; latest `make web-verify` is blocked by formatting in an
unrelated concurrent `NexusSplashScreen.tsx` edit. Real authenticated provider,
native Windows/macOS and eight-hour overnight evidence remain UNVERIFIED.

Canonical report: [`validation/current/MISSION_TEAMS_REPORT.md`](validation/current/MISSION_TEAMS_REPORT.md).
No commit or push was made; preserve unrelated concurrent worktree edits.

## Documentation product guide — 2026-09-13

Public documentation now runs from README through Quick Start, feature
discovery, personas and focused CLI/API/configuration references.
`docs/internal/FEATURE_INVENTORY.md`, `DOCUMENTATION_EVIDENCE.md` and
`DOCUMENTATION_COVERAGE.md` anchor claims to evidence; `DOCUMENTATION_COMPLETE.md`
is the delivery report.

Fresh proof: synthetic capture, docs verifier, isolated Quick Start project
flow, release tests, `git diff --check` and `make quality` PASS. This does not
change production NO-GO: authenticated provider/Mission, native platform and
hosted same-SHA evidence remain governed by the release reports below. No
commit/push was made.

## Production GO/NO-GO red team — 2026-09-13

Verdict: `PRODUCTION_NO_GO`. Also `CERTIFICATION_INVALID_FOR_CURRENT_HEAD`.

Executable candidate `e53f8352e4c3647632ebac7877cf0a6a3bd62647` was never frozen:
HEAD moved to `019dc585…` during the audit. Overnight is
`ABORTED_PRECONDITION_FAILED` (0h). Production finalization is
`NOT_READY_FOR_OVERNIGHT_CERTIFICATION`. No same-SHA hosted CI. Last completed
branch CI failed Linux/Windows/macOS E2E and Browser. Native Windows/macOS,
signed updater, and authenticated Mission `COMPLETED_VERIFIED` are unproven.
Public `nexus run` aborted in 0.2s. Maestro default remains ASSIST.

Evidence: `DEV/validation/production-go-no-go/FINAL_REPORT.md`.
Do not merge, release, or treat local `make quality` as production GO.

## Overnight production certification — 2026-09-13

Aborted before any Mission/soak work (`ABORTED_PRECONDITION_FAILED`, 0h).
Production finalization later published
`NOT_READY_FOR_OVERNIGHT_CERTIFICATION` for executable SHA `e53f835…`.
That still does not authorize overnight or production GO.

Evidence: `DEV/validation/overnight-production-certification/FINAL_REPORT.md`.

## Capability review intensive — 2026-09-12

Multi-reviewer pass (CLI JSON, make gates, Bugbot, security-review, cavecrew, GSD,
Codex) found tunnel auth not armed for CLI tunnels, CrossAccountResume hardlink/marker
fail-open, and capability lies on OpenCode/AGY/Gemini. Safe CRITICAL/HIGH/MEDIUM fixes
are in the worktree; canvas report at
`canvases/nexus-capability-review.canvas.tsx`. Claude/Gemini CLIs were SKIPPED.
Do not claim live CrossAccountResume PASS without TUI evidence.

## Handoff — 2026-09-12 automatic delegation interactive Lead closure

The dedicated branch now connects the real supervised provider entrypoint to
the existing delegation policy. `nexus agy`/`nexus codex` keep the foreground
runtime as a persistent Lead Agent; complete input lines are evaluated by the
deterministic AUTO/ASK/OFF policy, compound AUTO requests create the existing
Flow → WorkPlan → MissionRunner topology, and the Lead receives the final
integration prompt. No second scheduler or anonymous worker pool was added.

The Lead is resolved by the canonical current workspace path. If the workspace
has not been registered yet, the supervised interactive path creates the
normal Nexus Project record, then reuses or creates the persistent Lead in that
project. Provider/profile remain the Lead's current resource preference;
delegated Agent matching and downstream provider/model routing remain separate.

Local verification for this slice: focused app/control/launcher/Nexus tests,
full `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`,
`make build`, `make build-desktop`, and `git diff --check` pass.
The supervised Lead attachment uses canonical line submission so Nexus can
route complete prompts while preserving the existing raw PTY path for ordinary
`control attach`; real authenticated provider E2E and provider-level Mission
evidence remain `UNVERIFIED` in this environment.

The attempted real Codex smoke with the reported authenticated profile ended
at the supervised IPC handshake timeout; `nexus ps --json` was empty afterward,
so no orphan runtime remained. This is recorded as `UNVERIFIED`, not provider
success or Mission evidence.

## Codex CrossAccountResume — 2026-09-12

Cross-account Codex resume is restored without re-symlinking `sessions` to the
host. On Prepare/Resume, Nexus hardlinks or copies recent rollouts from sibling
Codex profiles (and a capped host set) into the active profile's canonical
`CODEX_HOME/sessions`, merges `session_index.jsonl`, and marks imports so
GetUsage never attributes foreign quota. Unit coverage is green; live omegaedu
picker verification against omegasistemas threads is still UNVERIFIED.

## Mission evidence isolation — 2026-09-12

The canonical `RunApplicationService.ValidationEvidence` projection now
verifies the complete append-only stream and then filters entries by the
persisted `run_id` envelope. Two Missions sharing a project therefore cannot
receive each other's validation claims. Missing/invalid mission identity in an
entry fails closed instead of silently broadening the report. The projection
uses the explicit complete-stream store read, so it does not truncate after
the normal 100-entry list page. Focused isolation, negative-metadata and
101-entry tests are GREEN; external authenticated Mission evidence remains
unavailable, so release is still NO-GO.

## Nexus Codex TUI lock — 2026-09-12

`nexus codex` no longer lets the quota app-server probe compete with a live
interactive TUI on the same isolated `CODEX_HOME`. Direct and supervised
launches acquire `$CODEX_HOME/nexus-tui.lock`; probes back off to last-known
official CACHED or isolated rollout evidence. Codex short flags `-c`/`-p` are
native again; use `--continue` / `--print` for Nexus aliases. Prepare writes a
single CODEX_HOME. Focused unit/race tests pass; live mid-turn interrupt with a
real Codex session is still UNVERIFIED. No additional commit/push was made by
this continuation; the existing campaign-autopilot commit is preserved.

## Canonical evidence projection — 2026-09-12

The existing append-only `ValidationEvidenceStream` now has a single
read-model boundary in `RunApplicationService.ValidationEvidence` and the Web
endpoint `GET /api/v1/runs/{id}/validation-evidence`. The projection verifies
the hash chain before returning entries and reports missing evidence explicitly
instead of inferring PASS. Focused Nexus/Web tests pass. No authenticated
production Mission stream was available; release remains NO-GO.

The Web client now exposes typed `getRunValidationEvidence` against this route;
its RED → GREEN transport test, typecheck and production build pass.

Codex runtime ownership was hardened with a per-profile OS lock. Interactive
TUI sessions hold the lock; quota/app-server probes fail closed or use bounded
last-known data while it is held, and the canonical profile home no longer
creates a competing `.codex/sessions` tree. Focused, full and race tests pass.
The stable-root real Codex retry still stopped at `model: loading`, so this is
not provider Mission evidence.

## Final closure continuation — 2026-09-12

Current commit HEAD is `42c22137a4a57ff6b6b80df8125b5a138f32b9e1`, matching
`origin/feat/nexus-maximum-delivery`; the complete-stream evidence projection
follow-up and final documentation remain uncommitted in the worktree. The
preceding finalization commit was created externally by the campaign
autopilot; it is preserved and was revalidated before continuation.

The existing routing contract now carries generic persisted `ExecutionGuidance`
through task requirements into prompt compilation. Durable routing decisions
also project deduplicated canonical `skill_refs` and `maestro_guidance_ref`
only for a real Maestro source/reference, plus candidate/selected Skill
source/version/hash/reason provenance. Agent identity remains separate from
provider, profile, account and model. Web types were extended to preserve the
same projection.

The aggregate `make quality` gate passes after small lint-only corrections;
the only remaining output is a non-blocking existing ESLint unused-variable
warning at `web/src/nexus/AgentTerminal.tsx:218`.

The E2E harness now supports `NEXUS_E2E_ROOT` for an empty, stable diagnostic
directory. A real AGY run reached the provider but reported `not signed in`;
Codex remained at `model: loading` even after the stable-root retry. Neither
run produced a provider marker or authenticated Mission evidence.

Canonical Mission evidence now also records bounded toolchain metadata
(Go/OS/arch and Node/package-manager versions when applicable) via fixed,
timeout-bounded commands with no shell.

The local E2E harness was corrected and regression-tested for bootstrap token
exchange, PTY Enter submission and symlink-safe profile isolation. Fresh full
Go/race/vet/security/build/desktop/frontend gates all pass. This does not turn
provider availability into Mission evidence.

Latest provider classification is conservative: direct host Codex marker PASS
only for host availability; Nexus Codex isolated runtime UNVERIFIED at model
loading (reproduced with the live-quota profile); AGY runtime UNVERIFIED
because it reported not signed in; OpenCode UNVERIFIED/pending auth. No durable authenticated Mission
`ValidationEvidenceStream` ID exists.

The follow-up RED → GREEN test also fixed legacy Codex configuration migration:
`Prepare` now copies a non-empty `home/.codex/config.toml` before adding the
canonical credentials-store default, preserving existing model/settings data.
Profile preparation itself waits on the same per-home lock, so setup cannot
mutate session/config artifacts while the TUI is active.
The local E2E harness also resolves its default imported-profile source from
`security.FindHostHome`, avoiding a provider-rewritten `HOME`; explicit source
configuration remains supported.

Release remains `NO-GO` with P1 evidence blockers for authenticated Mission,
live failover/escalation/handoff, Windows/macOS same-SHA and overnight proof.

## Final closure consolidation — 2026-09-11

Current working tree contains the final-closure audit/plan and a verified
backend slice. Native Skills are now source-agnostic at the catalog boundary;
Maestro is optional. Composer receives bounded Project Intelligence and uses
explicit intent routing. Runtime affinity and task-aware model candidate
resolution preserve desired preferences separately from actual allocation, and
mission package state carries an explainable `routing_decision` JSON record.
Canonical `SkillIDs` now survives WorkPlan → Flow → runner → ContextCapsule;
`MaestroSkills`/`MaestroGates` remain compatibility aliases, and builtin
skills resolve deterministically without Maestro. Handoff dependency receipts
remain intact after this transport change.
Mission verification now records concrete package/global results in the existing
append-only `ValidationEvidenceStream` and verifies its chain before promotion.

Focused and full local verification are green for the backend, frontend and
Codex adapter. No commit/push was made. The campaign remains `NO-GO`: no
authenticated Mission produced a durable stream instance ID in this run, and
real provider/native-platform/overnight evidence is incomplete. See
`DEV/validation/FINAL_CLOSURE_CHECKPOINTS.md` for resumable state and blockers.

## Atualização 2026-09-12 — Model routing e remoção de acoplamento interno

O roteamento de modelo passou a compartilhar uma única resolução task-aware nos
caminhos puro e integrado. Candidatos configuráveis são filtrados pela conta
live selecionada; `PREFER` pode fazer fallback, `PIN` bloqueia, e escalonamento
por falha de verificação reentra em `ALLOCATING` sem mutar a preferência.
Também foi adicionado tie-break determinístico e teste de raiz Skill opcional
com symlink cíclico.

O PromptCompiler canônico não referencia mais `CatalogSkill`,
`MaestroSkillDesc`, `MaestroClient` ou `maestrogates`; a compatibilidade legada
foi isolada em `maestro_prompt_compat.go`. `ExecutionContextRequest` recebe
somente `ExecutionGuidance` genérico, e a Intelligence não faz fallback para
campo Maestro.

Verificação fresca: `go test ./... -count=1`, `go test -race ./... -count=1`,
`go vet ./...`, `make security`, `make build`, `make build-desktop`,
`make web-verify` e `git diff --check` passaram. O ambiente ainda não fornece
Mission autenticada com `ValidationEvidenceStream` real, Windows/macOS nativos
ou overnight; o verdict global permanece `NO-GO`.

O teste composto `TestLocalAutopilotContractTraversesDiscoveryRoutingSkillsAndVerification`
agora comprova a cadeia local bounded discovery → `DIRECT` → Skill → model
task-aware → Mission Runner → `COMPLETED_VERIFIED`. A decisão de routing também
é recarregada após reopen SQLite e projetada sem recomputação.

## Atualização 2026-09-11 — Entrada PTY separada do controle Nexus

O caminho `CmdInput` do `SessionHost` está transparente byte a byte: não usa
mais `SlashPrefixRouter`, não descarta ESC/CPR e não intercepta `/nexus`. O
roteador permanece apenas como compatibilidade para `CmdSlash` explícito.

O protocolo agora expõe respostas tipadas e `CmdEvents` com limite e resumo
redigido. Foi adicionado `nexus control monitor [runtime-id]`, um sidecar TUI
somente leitura que consulta status/eventos sem `Attach` nem writer lease.

Reconnect e falha de attach não iniciam Recover/Start automaticamente; somente
Recover/Start, Stop e fechamento confirmado alteram o lifecycle. Testes focados
e race dos pacotes de controle passaram. O gate repetido Web ainda reproduz
falhas ambientais preexistentes de sudo/fixtures; não declarar E2E nativo ou
release multiplataforma a partir desta sessão.

## Atualização 2026-09-11 — Controle canônico Nexus

O protocolo de comandos dentro de runtimes supervisionados foi padronizado em
`/nexus` (ou `:nexus`). Os aliases `/ai`/`:ai` não são mais interceptados e
devem ser tratados como texto do provider. O binário `ai` continua apenas como
compatibilidade externa; o uso e a documentação novos devem usar `nexus`.

Verificação: `go test ./internal/control/host` passou.

## Atualização 2026-09-11 — Compatibilidade do terminal Codex

Foi corrigida a exibição de sequências Kitty/CSI-u (`/e[47;1:3u` e similares)
ao abrir `nexus codex`. O adapter direto e o driver supervisionado agora
definem `CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT=1`, compatível com o xterm.js
5.3 usado pelo Nexus. Próxima ação: recompilar/reinstalar o binário Nexus e
abrir uma nova sessão Codex; sessões já iniciadas precisam ser encerradas e
reiniciadas para receber o ambiente novo.

## Atualização 2026-09-11 — Correção final dos gaps de roteamento

O caminho de execução foi corrigido para não confundir capabilities de recurso
com personalidade de Agent: `MatchAgents` recebe somente especializações, e o
rótulo legado `implementer` não sobrescreve a classificação detalhada de uma
tarefa. `TaskRequirements` agora persiste a classificação natural nos Flow
steps, e `nexus run "<objetivo>"` entra no pipeline Flow → AgentMatcher →
ResourceScheduler → MissionRunner, mantendo `nexus run <provider>` compatível.

O handoff same-provider conserva `AgentID`, `ProjectID` e `ProjectName` no
runtime alvo. Como processo vivo e argumento `resume` não comprovam a sessão
no provedor, o estado/evento agora é `NATIVE_RESUME_UNVERIFIED`; somente uma
confirmação provider-level poderá elevar a `VERIFIED`. Cross-provider é
explicitamente `CONTEXT_HANDOFF`.

Validação focada passou. O relatório independente permanece `NO-GO` enquanto
não houver E2E autenticado de PTY/failover, live providers, runner nativo
Windows/macOS e conclusão do race completo.

## Atualização 2026-09-10 — Troca de projeto e efeitos visuais

A troca de projeto não desmonta mais a tela inteira para exibir o splash. O
shell permanece montado, preservando o fundo, chrome e efeitos visuais; um
indicador discreto aparece no canvas até o layout autenticado chegar. A
persistência fica bloqueada durante o carregamento para não gravar o fallback
temporário no projeto remoto.

Verificação: typecheck, ESLint, Stylelint, 25 testes Web focados, build Web,
`make build-desktop-wails` e HTTP 200 em `127.0.0.1:13000`. O binário desktop
está rodando localmente. O ambiente VMware/Mesa não oferece aceleração 3D, então
blur/efeitos GPU podem aparecer reduzidos durante o teste local.

## Atualização 2026-09-11 — Auditoria independente e AgentSpec persistente

A auditoria independente está registrada em
`docs/validation/EVOLUTION_FINAL_AUDIT.md` e o fechamento final em
`docs/validation/EVOLUTION_FINAL_VALIDATION.md`. O veredito permanece NO-GO:
Interactive Continuity, Agent Routing Truth e Autonomous Resource Continuity
não possuem toda a evidência E2E exigida; Windows/macOS e live providers estão
UNVERIFIED.

Os presets persistentes de Agent agora carregam `AgentSpec` completo no JSON de
revisão, incluindo instruções, responsabilidades, capabilities, domains,
strengths, tags e política de verificação. O 409 observado ao iniciar um Agent
novo é compatível com `REQUIRED_RESOURCE_SELECTION`/provider indisponível; o
servidor atualmente em execução é anterior ao último rebuild e precisa ser
reiniciado antes do smoke definitivo.

Próxima ação exata: reiniciar Nexus usando o binário deste worktree, capturar o
body/status do POST de criação/início do Agent, selecionar um recurso disponível
e executar o cenário E2E; depois repetir o race gate e os cenários de failover.

## Atualização 2026-09-10 — Investigação de janelas duplicadas no desktop

O desktop Wails foi compilado e iniciado de fato. Ele anexou ao Web Core em
`127.0.0.1:13000`. Uma segunda execução foi tentada e terminou sem criar outro
processo persistente, confirmando que `SingleInstanceLock` está funcionando.

A interação visual não pôde ser acompanhada neste ambiente porque WebKit/Mesa
reportou falhas de aceleração VMware/EGL/DRI e não expôs uma janela ao inspector
X11. O código possui somente dois caminhos explícitos de janela externa
(`target="_blank"` no link externo e fallback `window.open`); as janelas do
Workspace são painéis internos, não novas janelas nativas. Próximo teste deve
ser feito em uma sessão gráfica nativa funcional, clicando especificamente na
ação que dispara a duplicação.

## Atualização 2026-09-10 — Configuração de qualidade corrigida

O `Makefile` foi alinhado aos scripts do frontend: os gates agora usam os
binários locais instalados, formatam/verificam SCSS e o Stylelint cobre CSS e
SCSS. O lint-staged também foi corrigido para usar os binários locais e incluir
SCSS. Verificados com `make format-check`, `make lint-styles` e
`make lint-frontend`; todos passaram, com apenas o warning ESLint preexistente.

## Atualização 2026-09-10 — Fechamento dos caminhos restantes

AGY deixou de gravar snapshots live por chave de provider/perfil sem identidade;
`BatchFetch` segue a mesma regra e o launcher só usa quota AGY com escopo
verificável. `RuntimeSession` agora transporta `AccountScope`, e o monitor só
associa runtime afetado quando o escopo coincide. Snapshots legados continuam
em quarentena lógica, sem atribuição automática. Perfis `PENDING_AUTH` também
não são considerados verificáveis: o ID local é reservado, mas o escopo só é
ativado após identidade autenticada.

Verificação: `go test ./...`, `make quality-full`, `go vet ./...`, `make
lint-go` e `git diff HEAD --check` passaram após este último patch. O gate
reportou apenas o warning preexistente de diretiva ESLint não utilizada; não
houve commit/push.

## Atualização 2026-09-10 — Correções pós-review

Foram corrigidos os bloqueadores do review local: lifecycle do tunnel sem
cancelamento prematuro, readiness com tratamento de erro, manager por Server,
backfill seguro de AccountScope, cache last-known scoped, uso de scope em
scheduler/cooldown/telemetria/eventos, persistência de escopo em eventos,
registry de runtime injetado, estado de instalações preservado e discovery da
TUI desacelerado. A tela Web de remote access agora usa i18n e contrato de
perfil tipado.

Verificação atual: `go test ./...`, `go test -race ./...`, `go vet ./...`,
typecheck/build/test Web, Prettier e `git diff HEAD --check` passaram. Ainda
é necessário repetir os gates completos do Make e validar o tunnel real; a
execução nativa Windows/macOS continua não verificada. Não houve commit/push.

## Atualização 2026-09-10 — AccountScope / próximo passo

Foi implementada a base de isolamento por conta: `AccountScope` tipado,
persistência e versionamento de identidade, APIs scoped de quota, cooldown e
monitor com chave canônica, além de eventos/notificações com escopo. Snapshots
legados sem identidade não são migrados nem atribuídos nas APIs novas.

Verificação: testes focados, `go test -race ./...` e `git diff --check` PASS.
O registry de instalação e os comandos `providers status/register` foram
implementados, conectados ao fluxo de perfil isolado e ao estado `PENDING_AUTH`.
TUI, API e cliente Web agora exibem/operam o lifecycle de instalação e registro;
o registro sob demanda fica em `PENDING_AUTH` e usa perfil isolado.
O boundary de quota para execução efêmera já impede persistência quando
`managed=false`. Próxima ação exata: conectar um comando de uso único a HOME
temporário, marcando `UNMANAGED_EPHEMERAL` e evitando também eventos/status de
conta.

## Atualização 2026-09-10 — Verificação final da consolidação

O último ciclo eliminou o warning conhecido de `NexusWorkspaceApp` usando a
referência estável `refreshAgents`, removeu `fileModTime` órfão do adapter Codex
e corrigiu o contrato de formatação do help CLI/tunnel (53 placeholders e
argumentos alinhados, spelling corrigido). Nenhum comportamento público foi
redesenhado.

Verificação atual: `make quality-full` PASS (exit 0; Web 62/320; Go tests,
race, vet, lint, build e security); `make web-verify` PASS 10/10; `make
lint-go` PASS 0 issues; testes focados de `internal/app` e
`internal/control/web` PASS; smoke `nexus --help` e `nexus doctor --json` PASS;
`git diff --check` PASS. O worktree continua deliberadamente dirty e sem
commit/push automático.

A Definition of Done local está coberta para Linux e para o escopo tocado. A
matriz nativa Windows/macOS continua `NOT VERIFIED` (há cross-build, mas não
execução nativa/same-SHA CI nesta sessão); permanecem P1 de isolamento mais
profundo de process supervision e taxonomia provider/quota, e P2 de registry
CLI gerado/DTOs restantes. Próxima ação recomendada: promover o worktree por
CI same-SHA nativo antes de qualquer claim multiplataforma ou release.

O SHA publicado `bfc90fc98acd700c165c8e7df26ac3d3670dce54` possui o run remoto
[34472251335](https://github.com/IAPro-Community/nexus/actions/runs/34472251335).
Esse run falhou nos asserts remotos de Windows/macOS/browser e Desktop macOS;
os logs detalhados não estão acessíveis com a autenticação local atual. O
rerun de jobs falhos foi tentado e retornou HTTP 401 (`gh` token inválido).
Reautenticar `gh` é requisito externo antes de repetir o CI same-SHA; não houve
commit, push ou alteração de workflow nesta tentativa.

Após reautenticação, o rerun foi executado no mesmo SHA (attempt 2) e terminou
com falhas reproduzidas. Os artefatos mostram: quota monitor/lease e testes de
lease do host no Windows; allowed-roots/filesystem em Windows/macOS; race/Web
no macOS; assert do build Wails macOS após aviso de ampersand-escape; e browser
sem `.nx-os-shell` dentro de 10s. Linux/frontend/security/Desktop Windows
passaram. O próximo trabalho técnico deve ser um ciclo de debugging por
cluster, começando pelo contrato de allowed-roots compartilhado por Windows e
macOS; não declarar suporte nativo nem fazer release antes de novo same-SHA.

Esse primeiro cluster já foi corrigido localmente: `allowed-roots` usa a raiz
temporária do SO e canonicaliza a raiz comparada; quota leases usam sequência
atômica para evitar colisão de `UnixNano` no Windows. Testes Web/Nexus completos,
race focado, lint Go e diff check passaram. Ainda não há novo CI same-SHA porque
essas mudanças não foram commitadas/publicadas.

O gate global local após o slice foi interrompido por alteração staged alheia
ao slice em `install.sh`/`install.ps1`: o teste de release rejeita
`@iapro/orquestrador-maestro-cli@latest`. Essa falha permanece classificada como
pre-existing e não foi mascarada nem modificada.

## Atualização 2026-09-10 — Isolamento de quota Codex

O diagnóstico de quota repetida encontrou duas falhas no caminho Codex:
rollouts compartilhados por symlink eram candidatos para qualquer conta, e o
adapter lia `usage.json` diretamente como `LIVE`, bypassando a validação do
quota engine. Agora rollouts compartilhados exigem correspondência com o
login do host; snapshots locais são tratados pelo cache central, que valida
provider/perfil/e-mail e frescor.

Regressão adicionada em `internal/core/provider/adapters/codex/usage_test.go`.
Testes dos pacotes Codex/profile/quota/Nexus e vet passaram. O próximo passo é
reconstruir/reiniciar o processo Web e conferir a tela Usage/ResourcePicker;
arquivos `usage.json` já contaminados historicamente podem permanecer com a
mesma janela até uma observação autêntica substituir o cache. Windows/macOS
continuam sem execução nativa nesta sessão.

## Atualização 2026-09-10 — Contratos tipados do Resource Scheduler

`ResourcePicker` agora consome `ProviderAccount` e `SchedulerDecision` de
`web/src/types.ts`; o facade usa `ResourceAllocation` para a resposta de
seleção. A quota continua parcial/defensiva: a UI cria defaults de exibição,
mas não converte ausência em quota conhecida.

Verificação: `npm --prefix web run test -- --run src/nexus/api.test.ts` 12/12,
typecheck PASS e `make quality-full` PASS (Web 62/320; Go/race/vet/lint/build/
security PASS). O warning ESLint de `NexusWorkspaceApp.tsx:228` continua
preexistente. Próxima ação: auditoria final requisito-a-requisito, mantendo
pendentes os itens P1/P2 já documentados.

## Atualização 2026-09-10 — Contratos Web legados tipados

Os últimos `any` de produção no facade foram removidos dos contratos de Agent
Config revisions, Maestro status/advice e input de WorkPlan. Extensões abertas
continuam modeladas como `unknown`/records, sem fabricar estrutura inexistente.

Verificação final: typecheck PASS, testes focados 14/14 e `make quality-full`
PASS (Web 62/320; Go/race/vet/lint/build/security PASS). O warning ESLint de
`NexusWorkspaceApp.tsx:228` permanece preexistente. A campanha ainda mantém
P1/P2 e a matriz nativa como pendências documentadas em `docs/refactoring`.

## Atualização 2026-09-10 — Ownership do registry de runtime

O `SessionHost` agora recebe e usa o registry proprietário do Launcher/control
host para persistência de estado, startup e atenção. O singleton só é usado
como fallback de compatibilidade quando o construtor legado não fornece um
registry.

Verificação: `TestSessionHostUsesConfiguredRegistryOwnership` e os pacotes
host/launcher/app passaram. O próximo corte P1 continua sendo a fronteira
mais profunda entre process supervision, provider discovery e eventos; não há
claim nativo Windows/macOS nesta sessão.

## Atualização 2026-09-10 — Cancelamento de resource discovery

O caminho Web de resources agora mantém o `context.Context` até as operações
de detecção/capabilities dos providers; o método sem contexto continua como
fachada compatível para callers antigos. `AllocateResource` também usa o
caminho cancelável.

## Atualização 2026-09-10 — Registry de providers injetável

O Core deixou de depender diretamente do registry global de drivers nos
caminhos de discovery, continuidade, execução autônoma e intelligence. O
`Nexus` default injeta o registry de produção; construções legadas usam
fallback compatível.

Verificação: teste dedicado de isolamento e `go test ./internal/nexus/...`
passaram. O gate integrado precisa ser repetido após esta alteração.

Verificação: testes de contexto e `go test` dos pacotes control/app/nexus
passaram. O próximo gate deve ser `make quality-full` antes de qualquer nova
decisão arquitetural.

## Atualização 2026-09-10 — Transporte HTTP compartilhado dos clientes Web/Desktop

A auditoria dos consumidores encontrou duas implementações de `fetch` com
duplicação de autenticação desktop, base URL, CSRF, expiração de sessão e
erros. `web/src/api.ts` agora expõe o transporte tipado; `web/src/nexus/api.ts`
mantém a fachada de domínio e reutiliza esse transporte. `NexusAPIError`
continua exportado como alias compatível.

Testes focados: Web API + Nexus API 17/17 e typecheck PASS. Gate Web completo:
`npm --prefix web run quality:full` PASS, com 62/317 e o warning ESLint já
conhecido. Próxima ação: repetir `make quality-full` e então auditar a matriz
CLI/Web/Desktop sem criar outro transporte paralelo.

A auditoria CLI também corrigiu o help principal: `paths`, `rename`, `export` e
`issue-report` eram despachados mas não documentados. O teste de contrato passou
no ciclo red/green. `make quality-full` passou novamente com Web 62/317,
Go/race/vet, lint, build e security. Próxima ação: revisar os contratos de
provider/quota contra o registry existente, sem introduzir uma interface
monolítica ou novos scrapers. A auditoria encontrou e corrigiu a assimetria de
casing no `internal/core/provider.Registry`; `go test ./internal/core/provider/...`
passou. Próxima ação: repetir o gate integrado e auditar a evidência de Usage/
Quota (`UNKNOWN`, source e observedAt) contra os adapters atuais.

Essa auditoria foi concluída: o modelo agora representa valores absolutos
opcionais e confiança, sem scraping novo. `make quality-full` passou novamente;
smoke `go run ./cmd/nexus --help` e `go run ./cmd/nexus doctor` também passou.
O próximo corte deve ser a auditoria final requisito-a-requisito, não uma nova
abstração especulativa.

Essa auditoria foi registrada em `docs/refactoring/definition-of-done-audit.md`.
Ela mantém a campanha aberta: runtime/MissionRun, timeline de eventos e
evidência nativa permanecem P1/NOT VERIFIED apesar dos gates Linux verdes.

Nesta continuação, os caminhos HTTP de MissionRun passaram por
`internal/nexus/run_application.go`, e ContextCapsule/WorkReceipt passaram a
respeitar cancelamento em memória e SQLite. Focados de Nexus/Web passaram.
Também foi adicionada correlation ID opcional ao EventBus e à persistência de
atividade (`0015_event_correlation.sql`), com propagação para o mapper Web e
uso concreto em failover de MissionRun e handoff. O gate integrado seguinte
passou com exit 0. A auditoria encontrou e corrigiu uma assimetria: o handoff
de contexto agora também correlaciona o evento pelo `lineage_id`; a taxonomia
compatível está documentada em `docs/architecture/events.md`. Próxima ação
exata: isolar o próximo contrato de runtime/processo apenas se houver uma
interface concreta e testável; evidência nativa Windows/macOS continua
pendente. O gate integrado posterior (`make quality-full`) passou com exit 0;
o stop HTTP também passou a respeitar cancelamento durante o wait de reap.

O próximo boundary extraído foi `RuntimeApplicationService`: list/start,
detail/capabilities, cleanup/delete/title, stop/respond/handoff e `WaitForExit`
já não são montados diretamente pelo `APIHandler`. IPC, launcher e provider
mechanics continuam dependências explícitas do serviço, sem duplicar lifecycle.
O gate integrado após essa extração passou com exit 0.

O `RunApplicationService` também publica eventos correlacionados de lifecycle
MissionRun e redige erros antes de persistir/emitir. O gate integrado posterior
passou com exit 0; a próxima revisão deve auditar produtores de runtime/host
que ainda não têm correlation IDs de operação maior. A projeção agora inclui
ownership de projeto/agente quando o run fornece esses dados.

## Atualização 2026-09-10 — Contrato de cancelamento do MissionRun

A auditoria do fluxo de runs encontrou uma lacuna concreta: os métodos de
`RunRepository` recebiam `context.Context`, mas o repositório em memória e o
adapter SQLite ignoravam cancelamentos antes de leituras, gravações e leases.
Ambas as implementações agora verificam `ctx.Err()`; testes cobrem o contrato
com contexto cancelado sem alterar estado.

Verificação focada: `go test ./internal/nexus/runner ./internal/nexus` PASS.
Próxima ação exata: executar novamente `make quality-full` após esta correção.

## Atualização 2026-09-10 — Boundary de aplicação para Missions

`MissionApplicationService` foi adicionado para o CRUD de Missions, detalhe
com tarefas/assignments, criação de tarefas e atribuição de agentes. Os
handlers HTTP de Missions agora usam esse contrato; a execução viva de
`MissionRun` continua no runner existente, sem duplicar lifecycle de runtime.

Durante os testes foi corrigida uma falha real no store: timestamps de Missions
eram persistidos como texto SQLite e lidos diretamente como `time.Time`. A
leitura agora normaliza timestamps nullable com RFC3339Nano, preservando o
contrato JSON e cobrindo o fluxo com testes de aplicação e HTTP.

Verificação: `go test ./internal/nexus ./internal/control/web` PASS;
`make quality-full` PASS, com Web 62/315, Go/race/vet, lint, build e security.
Linux verificado; Windows/macOS continuam `NOT VERIFIED`.

Próxima ação exata: auditar o contrato HTTP de `MissionRun`/runs e só extrair
um boundary de aplicação se houver duplicação concreta de regra no transporte.

## Atualização 2026-09-10 — Composer application boundary

`ComposerApplicationService` foi adicionado e conectado aos handlers Web de
sessões Composer. O serviço centraliza o contrato de transporte para criar,
listar, obter, adicionar turnos, alterar skills, finalizar, refinar e resolver
unknowns. As regras permanecem no Core/Nexus por enquanto, evitando duplicação
enquanto inteligência e Maestro ainda são dependências cruzadas.

Também foi corrigido o descarte de `context.Context` nas operações públicas de
Composer. O teste `TestComposerOperationsRejectCanceledContext` cobre criação,
listagem, leitura, turno, skill, finalização e resolução.

Verificação focada: `go test ./internal/nexus ./internal/control/web` PASS;
`make quality-full` PASS, com Web 62/315, Go/race/vet, lint, build e security.

Próxima ação exata: executar `make quality-full`, depois auditar contratos Web
de Composer e decidir se a lógica de persistência pode ser extraída sem copiar
as regras de inteligência/Composer.

## Atualização 2026-09-10 — Correções do deep review

Foram corrigidos os achados prioritários: `nexus plan run` agora processa o
run; mudanças de `AgentSpec` exigem nova sessão; workspace/isolation aparecem
na compilação de contexto; e o tipo Web aceita `agent_spec`. Gates Go e Web
passaram. Não houve commit ou push. A pendência arquitetural restante é
congelar a especialização do Agent no snapshot imutável de cada MissionRun.

## Atualização 2026-09-10 — Higiene do índice Git

A branch foi auditada para separar fonte/documentação de estado local e artefatos
gerados. `.omx/`, `.superpowers/`, `.orquestrador/runtime/` e o binário gerado
`loadtest` foram removidos do índice, permanecendo no checkout local quando
aplicável. O `.gitignore` passou a cobrir esses caminhos e deixou de ignorar
falsamente os arquivos-fonte `web/src/styles/_tokens.scss` e
`internal/update/keyring.go`.

Validação concluída com `git ls-files -ci --exclude-standard`,
`git diff --check` e `git diff --cached --check`. As alterações de código já
existentes (`internal/control/web/*`) e documentos novos do worktree foram
preservados. Próxima ação: revisar o diff e criar o commit manualmente se a
limpeza estiver aprovada; não houve commit ou push automático.

## Atualização 2026-09-08 — Correção do Project Shell

O `409` observado ao abrir Project Shell foi rastreado aos logs do runtime:
`fork/exec /usr/bin/zsh: no such file or directory`, seguido de timeout IPC.
O status HTTP era enganoso porque o handler classificava toda falha de launch
como conflito.

O `ShellDriver` agora valida a execução do shell configurado e usa fallback para
um shell disponível. O handler retorna `503` em falhas de boot, mantendo o erro
real. Testes focados de driver, web e Nexus passaram.

`VM108...reportAllChanges` é de um script/instrumentação do navegador (não está
no bundle fonte do Nexus) e não foi alterado. Próxima ação: reiniciar o processo
`nexus web`/binário em execução e abrir novamente o Project Shell; se falhar,
usar a mensagem 503 e o log em `~/.local/share/ai-manager/logs/shell-*.log`.

## Atualização 2026-09-08 — Rail inteligente e gestão simplificada de projetos

### Estado atual
O rail de projetos foi ajustado para uma composição adaptativa estilo VS Code:
260px em desktop, Projects/Agents dividindo o espaço e Tools com scroll próprio.
O menu contextual de projetos inclui Abrir, Renomear e Remover; renomeação e
remoção usam os endpoints Nexus existentes. Exclusão do projeto atual seleciona
o próximo item ou abre o Project Hub, e `409` por agentes ativos vira toast
localizado.

### Validação
Typecheck, lint de estilos, allowlist, testes Vitest (313/313), build Web e
`make quality` passaram. Existe apenas o warning React hook já conhecido em
`NexusWorkspaceApp.tsx`.

### Próxima ação exata
Rodar o harness visual/browser autenticado nos viewports 320x568, 390x844,
768x1024, 1024x768 e 1440x900 para confirmar ausência de overflow, truncamento,
scroll duplicado e foco quebrado. Não criar commit/push automaticamente.

## Atualização 2026-09-07 — Deep review fixes + commit

### Alterações desta sessão
Commit `059bb5c`: fix(web,go): close review findings and migrate inline styles to SCSS Modules.

**Backend (Go):**
- `internal/nexus/maestro.go`: catálogo com cache TTL 30s, allowlist de diretório para execução de shell, captura de stderr, hash de preview incluindo mtime.
- `internal/nexus/runner/runner_test.go`: asserção de timestamp reforçada.

**Frontend (React/TypeScript):**
- `web/src/features/work/ComposerSurface.tsx` + `.module.scss`: ~20 inline styles migrados para SCSS Module.
- Outros findings do review (H3, M1, M2, M4, L1) já estavam resolvidos no codebase.

### Validação
- `go test ./...` PASS, `go vet ./...` PASS
- `bun run typecheck` PASS, `bun run lint` PASS (1 warning preexistente)
- `bun run lint:styles` PASS, `bun run check:styles` PASS
- `bun run test` 62/62 arquivos, 313/313 testes PASS
- `make quality` PASS, `make security` PASS, `make build` PASS (v0.5.0-beta.23)

### Próximo passo
CI same-SHA: fazer push do commit `059bb5c` e aguardar resultados do GitHub Actions.
Se CI green → preparar release candidate com changelog e branch protection.

---

## Atualização 2026-09-07 — Auth inválida ao abrir `nexus web`

A causa era o comando abrir apenas a URL base depois que o token deixou de ser
colocado na URL. A SPA recebia `/api/v1/session` sem cookie e entrava como não
autenticada.

Agora `Server.BootstrapURL()` entrega o token em fragmento
`#nexus_bootstrap=...`; `web/src/api.ts` troca esse token por POST, remove o
fragmento com `history.replaceState` e então valida a sessão normalmente.
`nexus web open` e `nexus web url` usam o BootstrapURL persistido, inclusive
para estados antigos que ainda armazenam apenas o token separado.

Validação real concluída com `nexus web --port 0 --no-open`: bootstrap, cookie,
sessão autenticada e CSRF foram confirmados. O token não aparece em query string
nem em URL enviada ao servidor.

## Atualização 2026-09-07 — Mídia documental sem dados reais

O capturador documental agora trabalha somente com dados temporários e
sintéticos. Ele cria `Nexus Demo Workspace`, inicia um Project Shell real com
prompt neutro, exige retorno do marcador `__NEXUS_DOCS_TERMINAL_OK__` e falha
quando encontra erro, recuperação, desconexão ou dados conhecidos do host.

O modo `NEXUS_DOCS_CAPTURE=1` também substitui o inventário de providers por
`demo-provider`/`synthetic-fixture`, evitando que versões de Codex, Claude,
AGY ou outros binários instalados apareçam na documentação. O manifesto e o
verificador exigem classificação `SYNTHETIC`; VIS-005 contém evidência de
terminal e VIS-011 permanece explicitamente como tela inicial Desktop sem
projeto/runtime.

Validação concluída: captura Web isolada PASS, teste Go de inventário sintético
PASS e `node scripts/docs-verify.mjs` PASS. Não considerar o screenshot Desktop
como prova de terminal nativo; ele documenta apenas o estado inicial seguro.

## Atualização 2026-09-07 — Memoização do Flow

`FlowTaskNode` foi memoizado após a revisão final do caminho de renderização.
O frontend foi revalidado com `verify` 10/10. O próximo passo continua sendo
benchmark nativo; não há evidência suficiente para afirmar FPS ou GPU.

## Atualização 2026-09-07 — Primeiras correções de desempenho

Foram aplicadas três correções: Flow Canvas não reconstrói o grafo ao mudar
somente a seleção e usa `applyNodeChanges`; DesktopBridge coalesce bootstrap
concorrente; AgentTerminal agrupa saída WebSocket por frame antes de escrever
no xterm. Frontend, Go, race/vet e build Wails de produção passaram.

Próxima ação exata: implementar/executar o benchmark comparativo do plano
`.omx/plans/desktop-web-performance-parity.md` e confirmar se WebView/GPU,
startup ou outra superfície ainda excede a meta de 1,20x da Web.

## Atualização 2026-09-07 — Plano de paridade Web/Desktop

O plano executável para diagnosticar e corrigir a lentidão percebida do Desktop
está em `.omx/plans/desktop-web-performance-parity.md`. Ele exige baseline na
mesma máquina/SHA, instrumentação do caminho crítico e benchmark comparativo
antes de otimizar. A primeira execução deve realizar as Fases 0–2; somente
depois escolher as duas trilhas de causa com maior contribuição ao p95.

Não houve alteração de código nem confirmação de causa nesta etapa. A meta
proposta é Desktop <= 1,20x Web nos fluxos críticos; Linux requer smoke nativo,
e Windows/macOS continuam sem afirmação de correção até execução nativa.

## Atualização 2026-09-07 — AGY sem keyring no monitor de quota

O probe de quota do AGY não inicia mais Secret Service nem keyring privado. O
keyring isolado continua reservado ao fluxo interativo/login (`Run`). Quando o
probe não consegue usar o armazenamento em arquivo, a quota permanece
`UNKNOWN`/última observação e não é usada como capacidade atual. O binário em
`/home/desenvolvedor/.local/bin/nexus` foi recompilado e instalado; a Web foi
reiniciada em `127.0.0.1:3000` e o health check passou.

Também foi removido o menu duplicado `Novo` do cabeçalho de Terminais. O único
ponto global de criação é o menu `Criar`.

Próxima validação manual: abrir o Nexus, observar por pelo menos um ciclo de
quota e confirmar que não aparece solicitação de senha; abrir uma sessão AGY
interativa e confirmar que login/execução continuam funcionando.

O Desktop Linux também foi recompilado e iniciado como
`/home/desenvolvedor/.local/bin/nexus-desktop`. A Web foi iniciada novamente em
`127.0.0.1:3000`. Os runtimes AGY, Codex e shells já existentes permaneceram
ativos; não houve encerramento em massa.

## Execução Luna em andamento — 2026-09-07

O plano `DEV/SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md` está sendo executado.
P0/P1 local passaram: Go completo/vet e frontend 10/10. A implementação já
inclui `nexus <provider> --supervised` opt-in, completion de flags, checkpoint
Maestro schema 4 com digest dos arquivos DEV e lease interprocesso para eleger
um coletor de quota. Testes focados passaram.

Pacotes confirmados nesta sessão: P0/P1 local; P3 lease do monitor; P4 prioridade
cross-provider; P6 `--supervised` opt-in; P7 sugestões `/nexus<TAB>` e completion
shell; P8 checkpoint Maestro. Host, handoff e nexus também compilam cruzado para
Windows amd64 e macOS arm64; isso não substitui execução nativa.

Ainda não declarar conclusão: pools entre providers, fallback automático,
autocomplete interno, transação/reconexão completa e execução nativa Windows/
macOS continuam pendentes. Na próxima sessão, reidratar este handoff e o plano;
rodar o gate completo após cada pacote. A decisão de manter supervisão opt-in
até evidência nativa deve ser revista somente após P6/P12.

Quota atual: a auditoria de 2026-09-07 encontrou contas AGY/Codex autenticadas,
porém sem fonte/janela observada (`UNKNOWN`/`NONE`); OpenCode não autenticado e
Cursor sem Usage. Não usar esses números para anunciar capacidade ou fallback.

## Plano para Luna — 2026-09-06

Leia [`SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md`](SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md)
para executar a próxima campanha. Contém P0–P12, evidência de código, contratos,
matriz nativa e prompt. Esta etapa alterou somente documentação; nenhum novo PASS
de runtime/CI foi produzido. Começar pelo inventário do worktree e baseline.
Correções às promessas anteriores: quota não exposta é desconhecida; checkpoint
não prova reidratação; handoff exige reconexão e recuperação real; singleton por
processo não evita alerta duplicado entre instâncias. Flags novas são propostas.

## Atualização 2026-09-06 — Fechamento local dos reds do CI

O run remoto `34060911997` falhou em Frontend (Prettier), Windows (teste Go
completo) e macOS (race test). O acesso aos logs detalhados exige permissão de
admin, mas os jobs/steps falhos foram confirmados pela API.

As causas reproduzíveis foram corrigidas no worktree: `QuotaDropMonitor` aceita
os estados legados `OK`/`RATE_LIMITED` sem reabrir estados desconhecidos e
restaura a recomendação de failover; os quatro arquivos reportados pelo
Prettier foram formatados; e `agyQuotaGroupAvailable` não tem mais retorno
nomeado inútil para o golangci-lint v2.

Evidência fresca: `bun run verify` 10/10 e 306 testes; `go test ./...`; `go
test -race ./...`; `go vet ./...`; golangci-lint v2.12.2 com 0 issues; teste e
binário compilados para Windows amd64 e Darwin arm64; Wails Linux; e
GoReleaser v2.18.0 snapshot. Próxima ação exata: revisar o diff, criar o commit
`fix(ci): close cross-platform runtime failures` e fazer push manual para
disparar uma nova matriz CI. Não houve commit/push automático.

## Atualização 2026-09-06 — Monitor residente de quotas

O monitor de quotas agora é residente no processo Nexus: Core Web/Desktop o
inicia e encerra com o contexto; execuções diretas `nexus agy|codex|cursor` e
terminais/Agentes também o ativam de forma idempotente. A primeira coleta é
imediata e as seguintes usam ticker de 60s e cache de cinco minutos.

Alertas são independentes por provedor, perfil, grupo, janela e ciclo de reset.
Somente `LIVE`/`CACHED` alertam consumo; leituras degradadas emitem no máximo
um evento de monitoramento degradado e a recuperação rearma o estado. O estado
de supressão fica em `StateDir/quota-monitor-state.json` com escrita atômica.

O histórico sem `runtime_id` agora agrega eventos de todos os runtimes, e a UI
exibe alertas globais, grupo/janela e idade da leitura. A aba Recursos não é
mais responsável por iniciar o monitor.

Verificação concluída: `go test ./...`, `go vet ./...`, typecheck, 306 testes
Vitest, lint/style gates e build Web passaram. Não houve commit ou push.

## Atualização 2026-09-06 — Quota AGY e seleção de conta

O cache de quota agora respeita TTL de cinco minutos para todos os providers;
cache AGY expirado vira `UNKNOWN` e não pode ser usado como capacidade atual.
O adaptador AGY tenta leitura ao vivo quando a cache expirou, e o scheduler usa
a fotografia por perfil capturada durante a seleção.

Contas sem quota conhecida não vencem contas com quota positiva conhecida. Com
os dados atuais, `kiver.omegasistemas@gmail.com` está `UNKNOWN`, enquanto
`kivervinicius@gmail.com` reporta 28% Gemini e 32% Claude/GPT; `nexus explain agy`
seleciona `kivervinicius-gmail`.

Web e Desktop foram recompilados; a Web em `127.0.0.1:3000` foi reiniciada com
o binário atualizado.

## Atualização 2026-09-06 — Correção reforçada de abas e xterm

O sincronizador de rota agora só reabre uma superfície quando a rota realmente
mudou; isso impede que a rota antiga `/overview` sobrescreva o clique em
`Terminal`. As abas também usam o mesmo callback de abertura da aplicação para
atualizar estado e URL de forma atômica.

`AgentTerminal` e `TerminalPane` não escrevem, focam ou fazem `fit` enquanto o
painel está oculto. A saída fica pendente e é descarregada após `data-active` ser
ativado, evitando o erro de `Viewport._innerRefresh` com `dimensions` indefinido.

O frontend foi recompilado e o `nexus-desktop` local foi reinstalado com essa
correção. Próximo teste manual: abrir o Desktop atualizado, clicar em
`Visão geral`, depois `Terminal`, e alternar as abas algumas vezes observando o
console.

## Atualização 2026-09-06 — Auditoria de fechamento atual

O baseline autoritativo continua em `feat/nexus-maximum-delivery`, HEAD
`1899ca6334576d859056d48a394e51d03758f313`, sem commits locais à frente do
remoto e com alterações não commitadas preservadas. A auditoria atual está em
[`validation/CURRENT_STATE_AUDIT.md`](validation/CURRENT_STATE_AUDIT.md) e o
relatório final em
[`validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md`](validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md).

Estado verificável: frontend 10/10, Go test/vet/race, Desktop Linux build e
diff check verdes. O Update Service agora falha fechado para manifesto sem
assinatura e o Desktop não anuncia capabilities não implementadas. O veredito
continua **NO-GO** porque o tree remoto commitado no SHA base falhou nos jobs
nativos Windows e macOS do CI `34012236345`; o worktree atual contém fixes
adicionais não publicados. Logs detalhados estão bloqueados por permissão de admin,
e `make security` reportou `No vulnerabilities found` usando o fallback
versionado quando o binário standalone não está no `PATH`.

Próxima ação exata: obter logs completos ou executar os runners Windows/macOS,
reproduzir os testes vermelhos por nome, corrigir causa raiz e repetir o mesmo
SHA; depois integrar a trust chain dos instaladores e executar browser,
Desktop e GoReleaser somente após os gates upstream verdes.

## Atualização 2026-09-05 — Implementação Completa Web + Desktop Multiplataforma

A implementação da arquitetura multiplataforma equivalente Web + Desktop foi concluída com sucesso no branch `feat/nexus-desktop-multiplatform`:
- **Dual Surface Equivalence**: Web (`nexus web`) e Desktop (`nexus-desktop` via Wails v2 estável) consomem exatamente o mesmo Core em Go (`internal/app/core.go`), o mesmo frontend React (`web.EmbeddedDistFS()`) e a mesma API REST / WebSocket.
- **PlatformBridge Abstrato**: Contrato unificado em `web/src/platform/` eliminando condicionais de plataforma espalhadas e integrando seletores de pastas, notificações, tema do SO e deep links.
- **Update Service Único**: Centralizado em `internal/update/service.go` com verificação estrita de assinatura Ed25519, SHA256 e respeito ao `InstallationMethod`.
- **Maestro Opcional**: Remoção de instalações silenciosas em `install.sh` / `install.ps1`, suporte à flag explícita `--with-maestro` e degradação graciosa para `MAESTRO_DEGRADED`.
- **Verificação**: `bun run quality` (8/8 gates, 280 testes unitários), `go test ./...` e `go vet ./...` 100% verdes. Sem auto-commit ou auto-push.

## Atualização 2026-09-05 — Autopilot Platform Stabilization

A campanha `01a07224-1c17-7863-881a-6b1963a6ce43` está em execução no branch
`feat/nexus-maximum-delivery`, HEAD `ab88fcbfddb5d99cf77c5e6f651aa2e5aa281770`.
T0 foi concluído. T1 corrigiu lockfile e pinos da CI, mas o lint reproduz 59
achados existentes; isso mantém o release bloqueado. T2 criou o
`ResolvedCommand`; T3/T4 corrigiram o teste ConPTY bloqueante, a semântica de
`ClosePseudoConsole`, a ordem IPC-before-provider e os metadados de startup.

Próxima ação: executar/verificar Job Objects no runner Windows, depois fechar
path identity, credenciais e doctor. Não anunciar suporte Windows/macOS nem
fazer release enquanto a matriz nativa não estiver verde. Não houve commit ou
push automático.

O incremento seguinte adicionou `PathRef`/`FilesystemIdentity`, migração SQLite
aditiva `0012_path_identity.sql` e capacidades explícitas de isolamento de
credenciais por SO. Os testes Go e vet continuam verdes. A próxima ação é
consolidar `nexus doctor` e produzir bundle diagnóstico redigido; a evidência
nativa Windows/macOS continua obrigatória.

`nexus doctor` agora usa um relatório read-only compartilhado para texto/JSON e
ZIP allowlisted (`report.json` + `MANIFEST.txt`). `nexus control doctor` só faz
limpeza com `--repair`. Próximo passo: expor o mesmo relatório na Web e obter
execução nativa Windows/macOS antes do checkpoint de runtime.

## Atualização 2026-09-04

Baseline do working tree revalidado após correções pequenas de P0. `make
web-verify`, `go test -count=1 ./...`, `go vet ./...` e `git diff --check`
estão verdes. O CI agora fixa Bun `1.3.9`; Composer possui rota/UI para
aceitar ou dispensar Maestro skills, envia skills aceitas ao PromptArtifact e
exibe Prompt Readiness, Unknowns e Assumptions.

As alterações locais anteriores foram preservadas; não houve commit/push.
PromptArtifact → Flow e o DAG visual editável já foram implementados neste
ciclo. Próxima ação: validar manualmente drag/zoom/conexão em flows de 20, 50 e
100 nodes e formalizar `FlowMaterializationRequest` caso o contrato externo
exija esse tipo nomeado.

## Current state

Nexus was rebuilt from the current source for local validation. Machine-local
installation paths and binary locations are intentionally omitted from this
durable handoff.
O servidor `nexus web --port 3000` está ativo com o bundle mais recente incorporando o seletor de temas em Accordion com paleta visual, densidade dinâmica real (`compact` vs `comfortable`), Topbar com proteção de largura e Terminal prioritário sem clipping.
Bootstrap ativo: `[REDACTED — ephemeral local secret; do not persist]`.

## Verification

- `make web-verify`: 8/8 gates aprovados (typecheck, lint, null-arrays, vitest, i18n, build, embed-sync, ui-markers).
- `npm --prefix web run test:e2e-hardening`: 100% aprovado cobrindo 6 viewports (320px até 1440px), verificação de não-obstrução do botão Terminal e delta de densidade.
- `make build`: Binário compilado com assets embarcados sincronizados e instalado em `/home/desenvolvedor/.local/bin/nexus`.

## Revisão atual — 2026-09-04

Revisão técnica registrada em [`DEV/validation/CURRENT_CODE_REVIEW.md`](validation/CURRENT_CODE_REVIEW.md).
O estado é **aprovado para uso local individual em loopback**. O frontend evita
polling concorrente e polling em background; o handler de terminal encerra
recursos de forma idempotente e limita resize concorrente. Build final instalado:
`nexus v0.5.0-beta.23`; Web ativo em `http://127.0.0.1:3000`.

Riscos que continuam explícitos: `--remote` usa HTTP sem TLS (risco aceito para
LAN privada), verificações de missão executam shell confiado sem sandbox hostil,
e macOS/Windows/Safari só têm evidência de compilação/configuração, não execução
local nesta máquina. Não há app nativo iOS.

Os controles dos terminais agora exibem estados de aplicação/reinício/pronto/erro,
feedback de lease e confirmação explícita para fechar mantendo ou parando o
runtime. Project Shells confirmam o encerramento do processo. O badge superior
resume trabalho, espera, desconexão e degradação; clique em runtime vivo foca o
terminal e clique em desconectado abre Agentes para recuperação.

Última validação: `make web-verify`, `go test ./...` e `make build` passaram em 2026-09-04 com a aplicação da identidade visual oficial e geração do pacote de assets de marca.

O Codex agora aparece como recurso compatível no modo Coding CLI quando autenticado,
usando `codex exec` para prompts não interativos. O servidor foi reiniciado após o
build e está ativo em `http://127.0.0.1:3000`.

A confirmação de modo ocorre antes da persistência/reinício; o fechamento de abas,
Project Shells e runtimes usa confirmação Nexus centralizada. Bootstrap atual:
`[REDACTED — ephemeral local secret; do not persist]`.

AGY agora mantém o modelo por perfil e recebe o diretório privado do keyring
explicitamente. O binário instalado é `nexus v0.5.0-beta.17`; reinicie sessões
AGY antigas para usar o wrapper corrigido.

CLIs interativos não recebem mais `TERM=dumb` quando o Nexus possui PTY real;
isso evita a confirmação bloqueada “Continue anyway?” do Codex e cobre também
agentes supervisionados.

O atalho `Sessão IA` agora abre o Composer e o launcher de sessão depois da
montagem da superfície, evitando perder o evento no primeiro clique. Na shell,
`Controle` indica quem pode digitar; `Liberar` devolve essa permissão e
`Assumir controle` solicita a permissão quando a sessão está em somente leitura.

Terminais de Agentes desconectados usam imediatamente o `runtime_id` retornado
por Recover/Start, sem esperar o polling global. O botão `Fechar terminal` remove
a aba visual; o Agente persistente continua separado até ser parado em Agentes.

Quando um projeto não tem contexto durável, o Composer informa que criará um
`AGENTS.md` base. A criação só ocorre ao clicar no botão, é atômica e não
substitui `AGENTS.md`, `DEV/INDEX.md` ou `DEV/CONTEXT.md` existentes.

Para AGY, a disponibilidade da conta exige quota nos dois grupos de modelos
(Claude/GPT e Gemini). Uma conta com somente um grupo disponível aparece como
indisponível e não deve ser escolhida pelo scheduler; os badges dos grupos
continuam exibindo o detalhe de cada pool.

O erro de runtime `Cannot read properties of null (reading 'forEach')` foi
corrigido no Flow: payloads com `phases` ou `packages` nulos são tratados como
vazios, e o bundle embutido foi recompilado. O teste de regressão e `yarn build`
passaram; o typecheck também passou na validação final.

A proteção foi centralizada em `web/src/nexus/workPlan.ts`; os endpoints de
planos normalizam as respostas na entrada, evitando que novas telas repitam a
mesma vulnerabilidade.

Terminais de Agentes também foram ajustados para não fixarem uma geração
antiga: após Recover/Start, a superfície resolve novamente pelo `agent_id` e o
bundle embutido foi recompilado.

`nexus usage` foi recompilado para mostrar a capacidade completa por modelo e
os resets de cada janela (5h/semanal), sem reduzir a saída ao bottleneck.

Sessões Web rotacionam automaticamente antes do vencimento. Se o servidor for
reiniciado ou a sessão ficar ociosa além do limite, a interface explica como
abrir o novo Bootstrap exibido por `nexus web`.

O uso de recursos é apresentado em tabela no CLI e em grid responsivo no
dashboard, com capacidade/reset por grupo e status sem sobreposição visual.
Para localizar contas e grupos no terminal, use `nexus usage` e
pressione `/` para filtrar.

O banner de atenção do agente também foi corrigido: ele não ocupa mais a área
central do dashboard e a mensagem padrão `? for shortcuts` não abre aviso
falso. O binário instalado contém esses ajustes.

Conclusões de tarefa e erros usam agora um componente de notificações
transitórias próprio; perguntas e aprovações continuam visíveis no banner por
exigirem resposta.

O motor de Flags Canônicas e Merged Help (`internal/control/flags`) foi incorporado
com sucesso: comandos universais (`--yolo`, `-y`, `--continue`, `-c`, `--resume`, `-r`,
`--print`, `-p`, `--effort`, `--plan`, `--accept-edits`) são traduzidos em tempo de
execução para cada CLI nativo e exibidos com destaque em `nexus <provider> --help`.

## Finalization status — 2026-09-06

The current dirty worktree preserves the user changes and the validation
artifacts. Local Linux/frontend/browser/security/release-static gates are green.
Public support claims now distinguish verified Linux evidence from pending
native Windows/macOS evidence; public community files and the MIT license
metadata are aligned. Installer documentation explicitly records the remaining
checksum-only trust-chain limitation.

The last remote run (`34012236345`) belongs to the committed tree at the same
base SHA, not to these uncommitted fixes. It failed in Frontend Format, Windows
full tests, and macOS race; downstream jobs were skipped. No native PASS is
claimed from that run, cross-compilation, or Wine.

## Community Preview publication preparation — 2026-09-06

The public launch documentation foundation is now in
[`docs/README.md`](../docs/README.md), with product, technical and release
playbooks under [`docs/community-preview`](../docs/community-preview). The
planned destination is `IAPro-Community/nexus`, but that public repository does
not exist yet and the local GitHub token is invalid. Do not rewrite installer or
update URLs until the destination is created or a transfer is explicitly
authorized. Next external action: re-authenticate `gh`, confirm repository
ownership/slug, then push the preserved candidate and inspect the exact-SHA CI.

Required next external gate: publish the preserved candidate through the
authorized branch workflow, then inspect the same-SHA native Windows/macOS,
Desktop, Browser, Axe, Visual and GoReleaser results. Do not declare GO before
those checks and the installer trust-root decision are complete.

## Next action

O cartão de diagnóstico em Configurações já usa o relatório read-only compartilhado
com `nexus doctor`; o próximo gate é evidência nativa Windows para ConPTY, startup,
Named Pipe e Job Objects, além do race/runtime nativo macOS. O build/embed Web
local já passou; o bloqueio restante é obter o mesmo resultado no CI do candidato
publicado.

Execute `nexus agy --help`, `nexus codex --help` ou `nexus claude --help` para conferir
a ajuda fusionada com aliases canônicos; execute `nexus usage` para quotas por janela
e `nexus web` para o Web Workspace OS local.

Correção mais recente: o Dialog de fechamento não aparece mais ao abrir a tela;
ele só é renderizado após uma ação explícita de fechar terminal ou Project Shell.
O servidor foi reiniciado com o bundle corrigido em `http://127.0.0.1:3000`.

O modo Plan foi removido dos controles de terminal e do cadastro de Agentes;
planejamento deve ocorrer no Composer/Flow ou ser enviado como instrução ao
Agente. O fluxo Safe/YOLO não encaminha mais `--plan` ao Codex. Os controles
“Assumir digitação”/“Liberar digitação” continuam ativos porque correspondem
ao lease de escrita do WebSocket do PTY compartilhado.

Build atual instalado em `/home/desenvolvedor/.local/bin/nexus` e servidor ativo
em `http://127.0.0.1:3000` (Bootstrap gerado no processo atual).

Os botões manuais de lease foram removidos do terminal. O WebSocket solicita o
lease automaticamente ao conectar; “Somente leitura” aparece apenas quando
outro acesso já possui a escrita. A barra lateral agora sempre oferece
“Visão geral”, além de Composer, Flow Runs, Uso, Sessões, Projetos e
Configurações, mesmo quando o layout salvo abre em terminais.

## Project picker and 1366×768 handoff — 2026-09-06

O cadastro de projetos usa o seletor nativo do sistema quando executado no Desktop
Wails. No navegador, ou quando a capacidade não é confirmada pelo backend, o modal
HTML continua disponível. A listagem HTML agora é leve por padrão e reutiliza
diretórios visitados durante a sessão; erros podem ser repetidos sem perder o contexto.

O modal foi ampliado e recebeu layout responsivo; o shell ganhou compactação para
1366×768 sem reduzir controles abaixo de 32px. O próximo passo recomendado é validar
visualmente no notebook alvo e, se ainda houver pressão vertical, experimentar o preset
de densidade `compact` antes de reduzir mais a barra lateral.

## Auth and installer follow-up — 2026-09-06

O problema de tela “Session Expired or Unauthorized” no `nexus-desktop` foi rastreado
à chamada de API cross-origin criada pelo uso da URL loopback a partir do WebView Wails.
As chamadas do bootstrap agora permanecem same-origin e usam o token Bearer no handler
embutido; há teste de regressão cobrindo esse contrato.

O instalador agora instala o `nexus-desktop` e cria lançador no Desktop quando a release
publica o artefato nativo com checksum. Para releases antigas sem esse artefato, o CLI
é instalado normalmente e o aviso explica a ausência; publique uma nova release para
ativar a instalação nativa automática.

## AGY model routing — 2026-09-06

O launcher agora resolve o modelo AGY antes de iniciar o processo: Gemini é a primeira
família quando sua cota local confiável está disponível; Claude Sonnet é fallback quando
Gemini está esgotado. A seleção é explícita (`--model`), evitando que o default interno
do AGY escolha Claude/GPT antes da hora. Configurações com modelo explícito continuam
com prioridade. Binários recompilados e instalados em `/home/desenvolvedor/.local/bin`.

## CI and Windows installer — 2026-09-06

O CI remoto está sendo acionado, mas a execução mais recente falhou antes dos jobs
dependentes: formato frontend, testes Go Windows e race test macOS. O workflow agora
permite execução manual e preserva logs dos testes de plataforma em artefatos. As
alterações locais de workflow só serão exercitadas no GitHub depois de commit/push ou
atualização da PR; este agente não cria commits automaticamente.

No Windows, o caminho explícito `-BuildFromSource` tenta instalar Go via WinGet quando
o compilador não existe. O caminho normal de release continua usando apenas o binário
assinado/checksum e não instala Go sem necessidade.

## Workspace tabs and xterm — 2026-09-06

O clique nas abas de produto agora navega pela URL do projeto e ativa a superfície no
mesmo evento. Isso impede o retorno de Terminal para Visão geral causado pelo sincronizador
de rota. Os terminais PTY também não escrevem nem fazem `fit` enquanto o painel está oculto;
a saída pendente é aplicada quando a aba fica ativa, evitando `Viewport.dimensions` indefinido.

## Quota fallback genérico — 2026-09-06

O pipeline de uso agora preserva a última observação válida quando a atualização de
qualquer CLI falha. O snapshot é marcado como `ESTIMATED`, recebe uma mensagem de
diagnóstico e não é tratado como cota atual. A seleção automática só considera
`LIVE`/`CACHED` como evidência fresca e prefere essas leituras às estimadas.
O launcher instalado em `/home/desenvolvedor/.local/bin/nexus` foi recompilado e
validado com as duas contas AGY.
## Autopilot orchestration checkpoint — 2026-09-06

The existing Nexus Composer/Flow/Mission Runner foundation was preserved. The concrete
false-success gap found in audit was that all package reviews could complete a Run
without a persisted global Definition of Done gate. `GlobalVerificationCommands`,
`VERIFYING`, persisted global results and fail→reopen→repair→global-pass behavior are
now implemented and tested. Read `docs/nexus-autopilot-architecture.md`,
`docs/nexus-maestro-orchestration-gap-analysis.md` and
`docs/nexus-autopilot-validation.md`.

Do not claim total product completion yet: provider-authenticated overnight execution
and native Windows/macOS process recovery require their respective environments. Next
action is to execute those scenarios and attach evidence; no commit/push was made.

## CI hardening — 2026-09-07

O checkout contém a correção local para os failures do SHA `1899ca6`: identidade
de filesystem, recência determinística, Codex shared-host, SessionHost/ConPTY
Windows, environment UTF-16, `processAlive`, JSON/PATH portáveis e frontend.

Go normal/race/vet/lint, frontend `bun run verify`, compilação cruzada de todos os
pacotes para Windows amd64/macOS arm64, GoReleaser snapshot e Wails Linux passaram.
Após publicar as alterações, reexecute o workflow e confirme a matriz nativa; não
há commit ou push automático nesta sessão.

## Quota alert deduplication — 2026-09-07

O monitor não usa mais o texto variável de contagem regressiva do reset na
identidade da janela. Isso elimina alertas repetidos para a mesma quota Gemini
abaixo do limiar quando apenas o countdown muda. Regressão normal e race passaram.
## Composer best-prompt handoff — 2026-09-07

Composer-only slice completed without touching Flow internals. New domain
contracts and migration are in `internal/nexus/composer*.go`,
`internal/nexus/store/composer.go`, `internal/nexus/store/migrations/0014_composer_v2.sql`
and `internal/nexus/prompt_compiler.go`. Contract details and known gaps are
in `DEV/SPECS/COMPOSER_FLOW_IMPLEMENTATION.md` and
`docs/nexus-composer-validation.md`.

Do not claim the full approved plan complete yet: remaining work includes
provider/local federated skill adapters, all-mutation revision enforcement,
context redaction/budgets, Composer UI cleanup and dedicated E2E scenarios.
## Sessão de decisões Luna

As decisões de quota sem fabricação, agrupamento visual com credenciais
separadas, prioridade configurável, supervisão opt-in e limites de continuidade
estão registradas em
[`DEV/DECISIONS/NEXUS_TERMINAL_CONTINUITY.md`](DECISIONS/NEXUS_TERMINAL_CONTINUITY.md).
O status atual continua parcial: gates locais passaram, mas a promoção oficial
aguarda matriz nativa macOS/Windows e CI remoto no mesmo SHA.

## Maestro catalog follow-up — 2026-09-07

The Maestro surface now reports the merged global + active-profile catalog.
Local runtime smoke confirmed 53/53 skills (the previous profile-only view was
48). Cards expose and copy the complete `SKILL.md` usage context, with a
generated fallback only when the source file is unavailable. Preserve merge by
stable skill ID and prompt provenance when changing discovery.

## Desktop/Web terminal continuity — 2026-09-07

O Desktop agora tenta anexar ao Core Web loopback já em execução antes de criar
um Core próprio. O fluxo reutiliza o bootstrap autenticado persistido pela Web,
cria uma sessão Desktop válida e usa um reverse proxy no AssetServer Wails para
REST e WebSocket. Isso mantém o mesmo registry, runtime e host PTY usados pelo
navegador. Se a Web não estiver ativa ou o estado não puder ser validado, o
fallback continua sendo o Core embutido do Desktop.

Evidência local: testes de anexação/proxy passaram; `go test ./...`, `go vet
./...`, race focado e `npm run verify` passaram; `make build-desktop-wails`
gerou `cmd/nexus-desktop/build/bin/nexus-desktop`. Para validação operacional,
reinicie a Web, abra primeiro um terminal no navegador, abra o binário Desktop e
confirme que o mesmo terminal aparece e continua emitindo saída. Windows/macOS
seguem sem smoke nativo neste ambiente.

## Desktop navigation continuity — 2026-09-07

O handler do Desktop foi refinado: rotas SPA como `/p/<project>/terminals`
agora recebem o `index.html` embutido localmente; o proxy para o Core Web fica
restrito às rotas `/api/*` e WebSocket. Isso evita que uma troca de tela faça a
WebView carregar um documento remoto e perca o estado do workspace.

O novo binário foi gerado em `cmd/nexus-desktop/build/bin/nexus-desktop`, a
instância antiga foi encerrada e a nova está ativa junto da Web em `127.0.0.1:3000`.
O teste E2E confirmou a troca Overview → Terminais; a execução inicial ainda
registrava uma falha responsiva no menu de criação em viewport 320px, corrigida
no fechamento local abaixo.

## Review closure — 2026-09-07

O bloqueio responsivo foi corrigido: o status do topbar não invade mais o
contexto em 320px e o botão Criar permanece fisicamente acionável. Também foram
corrigidos o deadlock de captura stdout, a comparação cross-platform de paths,
o readiness determinístico das fixtures de terminal, o diagnóstico do Browser
E2E, o bug de loop de loading por dependência de objetos em
`NexusWorkspaceApp`, o gate do bundle macOS e os gates documentais.

Estado local comprovado: frontend 10/10, 310 testes, Browser E2E/Axe PASS,
Go test/vet PASS, docs-verify PASS, Wails Linux build PASS e VIS-011 capturado
com Wails/Core reais. O relatório publicado permanece deliberadamente NO-GO
até o CI nativo Windows/macOS, GoReleaser snapshot e a integração final com
`origin/main` ocorrerem no mesmo SHA. Não fazer commit/push automático.

## Project Shell terminal overlay — 2026-09-07

O terminal do Project Shell apresentava uma linha de `W` porque o textarea
auxiliar do xterm estava visível: o build final não continha o stylesheet
oficial de `xterm`. O PTY e o WebSocket estavam corretos. `web/scripts/build.mjs`
agora incorpora o CSS após o esbuild, e o relatório de verificação exige o
marcador de isolamento do input.

Após recompilar e reiniciar a Web, o smoke real confirmou helper invisível,
terminal sem `W` espúrio e sem erros de console. A Web ativa continua em
`127.0.0.1:3000`; depois de mudanças no frontend, use `make build` e reinicie
somente a instância Nexus dessa porta para validar a tela de Terminais.

Na validação seguinte, a porta estava servindo o executável global antigo em
`/home/desenvolvedor/.local/bin/nexus`, apesar de o build correto existir na
worktree. Essa instância foi encerrada e substituída por `./nexus web --no-open`.
O smoke novo contra `127.0.0.1:3000` confirmou o helper invisível (`0x0`,
`opacity: 0`) e um terminal montado sem o overlay de `W`.

## Espaçamento semântico — 2026-09-07

A escala `--nx-spacing-*` foi restaurada em `workspace-os.css`, com aliases
`--nx-space-*` para compatibilidade. O binário ativo em `127.0.0.1:3000` foi
regenerado após `make build`; para visualizar a alteração no navegador, faça
hard refresh (`Ctrl+Shift+R`).

Nota operacional: neste ambiente `HOME` pode apontar para um perfil do Codex.
Para atualizar a instância usada pelo serviço, instalar explicitamente com
`LOCAL_BIN=/home/desenvolvedor/.local/bin make install-local`; `make build`
sozinho não substitui o executável global.
# Atualização 2026-09-07 — Composer com destinos separados

O Composer agora permanece utilizável quando o contexto está `MISSING`, `STALE`,
`HYDRATING` ou `FAILED`: elaboração, finalização e Copy seguem disponíveis.
Materialização Flow e envio ao Agent ficam explicitamente desabilitados até
`Context Readiness = READY`, com a razão e a ação de preparação visíveis.

Verificação: typecheck, testes focados de Work (40), lint/style/check-styles,
build Web e `go test ./...` passaram após sincronizar o bundle embutido. O gate
agregado `node web/scripts/verify-report.mjs` também passou (10/10 gates; o
relatório está em `DEV/validation/FRONTEND_LATEST.md`).
# Atualização 2026-09-07 — Catálogo honesto e preparação de tarefa

Implementada a primeira fatia do fluxo: catálogo Maestro separado em
operacional/biblioteca, deduplicação por ID com origem e disponibilidade,
contratos completos de skills no prompt compilado, e endpoints autenticados de
catálogo e sync com prévia/ID de confirmação e timeout.

Agentes agora exibem “Preparar tarefa”, com Objetivo, Contexto e Skills/revisão;
Agentes ativos reutilizam o runtime e Agentes parados continuam usando
“Iniciar e enviar tarefa”.

Verificação: `go test ./...`, typecheck, format check e Vitest focado passaram;
lint sem erros, com um warning preexistente. Próximo passo: completar a
integração do mesmo diálogo no Terminal/Composer e executar `make quality` mais
validação visual/Axe nos cinco breakpoints.
# Atualização 2026-09-07 — Finalization autopilot audit

Auditoria RC fresca registrada em `DEV/NEXUS_1_FINAL_ACCEPTANCE.md`. O runner
agora persiste mudança de estratégia por remediação e um contrato estruturado
`NeedsHuman`; o novo `TestOvernightAcceptanceSandbox` cobre o fluxo unattended
determinístico com falha injetada e DoD global. Gates locais Linux/Web/Go passam;
Frontend está em 61 arquivos/311 testes, Go/race/vet/quality/security passam;
o veredito permanece `NO_GO` por ausência de dogfooding real, crash/provider E2E
autenticado, matriz same-SHA e execução nativa Windows/macOS.

# Atualização 2026-09-07 — Correções de E2E e macOS CI

O Browser E2E foi reproduzido localmente e corrigido para resolver o bootstrap
via `nexus web url`, além de aceitar hit-test no ancestral posicionado sem
permitir overlays irmãos. O cenário passou nos seis breakpoints, Axe, Settings
ARIA e density. O harness Go Direct Work também aceita o novo bootstrap em
fragmento e mantém a rejeição de URLs sem token.

O workflow Desktop macOS deixou de usar `mapfile`, incompatível com Bash 3.2;
YAML foi validado localmente. Essas correções ainda precisam de um novo commit
e de uma execução CI same-SHA antes de alterar o veredito de release.
# Atualização 2026-09-07 — Diálogo único de preparação

O fluxo “Preparar tarefa” agora usa `web/src/components/TaskPreparationDialog.tsx`
nos Agentes, Terminal e Composer. Ele consulta contexto durável, exige estado
`READY` para envio, permite preparar/atualizar contexto e deixa a criação de
`AGENTS.md` explícita. Skills ficam limitadas a três e a prévia mostra o
envelope completo sem diff, credenciais, `.env`, sessões ou transcrições.

O endpoint de envio valida `project_id` e `context_fingerprint_id`; mudanças de
branch/worktree/estado invalidam o envio. Verificação completa de testes Go/Web,
typecheck, lint, estilos e vet passou. Próxima ação: `npm run build` e executar
validação visual/Axe do diálogo nos breakpoints definidos no plano.

Validação visual/Axe concluída: `node web/scripts/task-preparation-visual-verify.mjs`
passou nos cinco breakpoints, sem overflow horizontal, sem violações Axe
serious/critical e com foco de teclado na prévia rolável. Evidências estão em
`.tempmediaStorage/task-preparation/`.

## Atualização 2026-09-07 — Gate documental same-SHA

`scripts/docs-verify.mjs` passou a rejeitar também mudanças visuais staged e
unstaged. Portanto, `make docs-verify` só pode passar depois de capturas e
manifesto serem regenerados no candidate SHA imutável; no worktree atual o
resultado CONDITIONAL é intencional. Não atualizar `source_sha` manualmente.

## Atualização 2026-09-07 — Resume-first

O Overview agora apresenta retomada operacional baseada nos runtimes/agentes do
projeto, com prioridade Needs You, Continue/Recover, Terminal e Recent Work.
O modelo é puro e coberto por testes. A integração de histórico de missões e o
E2E de reabertura após reinício continuam pendentes e mantêm o P0-01 como
CONDITIONAL.

## Premium shell visual pass — 2026-09-08

O shell autenticado recebeu o primeiro vertical slice visual do
`skill-premium-web-experience`: topbar com contraste e estados de foco mais
claros, ambiente de canvas com profundidade sutil, rail com largura de 260px e
transição adaptativa, além de taskbar alinhada ao mesmo sistema visual. A
implementação usa módulos SCSS e tokens `--nx-*`, sem texto novo e sem mudanças
de API.

Validações concluídas: format check, ESLint (somente warning preexistente),
Stylelint, allowlist, typecheck, 62/313 testes Web, build e `make quality`.

Próximo passo operacional: reiniciar a instância Nexus local após instalar o
build atualizado e repetir o smoke visual autenticado nos breakpoints
`320x568`, `390x844`, `768x1024`, `1024x768` e `1440x900`, incluindo Axe e
checagem de overflow/foco.

Nota de validação: o runner visual existente precisa ser alinhado ao contrato
atual do comando `nexus web` (ele espera uma linha `Bootstrap` com token, mas o
comando imprime `URL`); portanto a captura autenticada desta fatia ficou
pendente por incompatibilidade do harness, não por falha observada no bundle.

## Correção de identidade Codex no widget de quota — 2026-09-08

O widget/TUI não deve interpretar a chave técnica `claude_gpt` como identidade
do provedor. Para contas `codex`, a camada `QuotaView` agora apresenta `Codex`;
as janelas de 5 horas e semanal continuam calculadas no mesmo pool. A
apresentação compartilhada por AGY permanece detalhada como Gemini e Claude/GPT
quando há grupos distintos.

O binário foi recompilado e instalado em `/home/desenvolvedor/.local/bin/nexus`.
Feche e reabra o widget/terminal para carregar a versão nova.

## Handoff — consolidação Nexus Core — 2026-09-10

Adicionados route composition em `internal/control/web/routes.go`, metadata
contratual em `GET /api/v1/system/info`, cliente/tipos Web correspondentes e
documentação em `docs/architecture/` e `docs/refactoring/`. O handler
monolítico ainda existe; sua extração deve continuar somente por domínio, com
testes de contrato antes de cada movimento.

Verificação fresca: `go test ./...`, `go vet ./...`, `go test -race ./...`,
`make lint-go`, `make security`, `bun run quality:full` e `bun run verify`
passaram. O único aviso é o hook React já conhecido. Linux foi verificado;
Windows/macOS permanecem `NOT VERIFIED`. O worktree tinha deleções/staging
pré-existentes em `.omx`, `.superpowers`, `loadtest` e mudanças em `DEV/`; não
reverter nem incluir esses artefatos.

Projects/Agents, Resources, Planning, Intelligence, Missions, Git e Maestro
foram extraídos para arquivos próprios do transporte, e o dispatch de
Projects/Agents foi removido de `server.go`, sem alterar receivers, rotas ou
payloads. `handlers_nexus.go` ficou restrito a wiring comum e doctor.
O isolamento de `nexus.Default()` em testes também foi corrigido para permitir
race detector confiável quando o diretório SQLite é temporário.
Verificação final desta etapa: `make quality-full` e `go test -race ./...`
passaram; frontend 62/314 e segurança passaram.

Projects e Resources agora possuem boundaries de aplicação no Core. Erros HTTP
preservam o campo `error` e acrescentam `code`; `mkdir` resolve ancestrais reais
para impedir escape por symlink. O help do CLI tem teste de cobertura dos
comandos despachados.

Agents CRUD/detail também foi movido para `agent_application.go`; start/stop/
recover/config continuam explicitamente no agregado por atravessarem o runtime.

O transporte Web agora usa contratos Go nomeados para erro estável, metadata do
servidor e detalhe de runtime (`internal/control/web/api_contracts.go`), sem
alterar os campos JSON consumidos por Web/Desktop. A correlação de processo com
MissionRun ficou explicitamente deferred porque o launcher não recebe run ID;
não foi criado um campo especulativo em RuntimeSession/Host.

O lifecycle do runtime, porém, já é observável sem essa correlação: o
`SessionHost` publica fatos de processo e eventos Nexus distintos
(`RUNTIME_STARTED`, `RUNTIME_STOPPED`, `RUNTIME_FAILED`), preservando lineage e
ownership quando o launcher os fornece.

Também foi corrigido um erro de compilação multiplataforma no fallback Windows
de `internal/control/web/hosts.go`; a variável de erro agora permanece válida
fora do escopo do `if` e o gate Linux completo passou depois da correção.

O consumidor Web também não usa mais `any` nos contratos de runtime detail e
event data: capabilities são `EffectiveCapabilities | null` e payloads de
evento são `Record<string, unknown>`.

Mission CRUD agora também usa DTOs nomeados compartilhados (`Mission`,
`MissionDetail`, `MissionTask`, `MissionAssignment` e inputs correspondentes),
reduzindo a divergência entre o Core e o cliente Web.

WorkPlan CRUD/revisions foram movidos para `plan_application.go`; Composer,
geração inteligente, compilação e execução continuam no agregado até que suas
dependências cruzadas permitam um boundary próprio.

Verificação mais recente: `make quality-full` passou após essas mudanças;
frontend 62/314, race Go, vet, lint, segurança e smoke `nexus --help`/`doctor`
passaram. Linux está verificado; Windows/macOS permanecem `NOT VERIFIED`.

Próximo contexto recomendado: completar a cobertura de DTOs dos endpoints v1
restantes ou formalizar a interface de cliente para consumidores não-Web, mas
somente após escolher um contrato consumidor real. O isolamento completo de
runtime/processo e a taxonomia mais ampla de eventos permanecem P1. Não criar
commit/push automaticamente.

## Handoff — semantic consolidation — 2026-09-10

The characterization report and executable audit are in
`docs/refactoring/CHARACTERIZATION-REPORT.md` and
`scripts/characterize-nexus.sh`. The typed AgentSpec/compiler work is in
`internal/nexus/intelligence`, `internal/nexus/config.go`, and the Nexus
execution paths. `AgentSpec` is stored in the revision JSON config; runtime
generations already retain the effective revision ID.

CLI `plan compile`, `plan run`, and `agents PROJECT` are now tested against
real Core paths. No commits were made. The worktree still contains broad
pre-existing HTTP extraction and frontend changes; preserve them. Before any
new architectural change, rerun the relevant focused tests and inspect
`git diff --check`. Remaining debt is deeper application-service extraction,
public API authoring of custom AgentSpec, and a generated registry for the
provider-native CLI surface.

Latest verification after injected provider registry ownership and the
continuity wrapper cleanup: `make quality-full` passed with exit 0 (Web 62/320;
Go tests, race, vet, lint, build and security passed). `git diff --check` also
passed. Linux is verified; Windows/macOS remain `NOT VERIFIED`. The only lint
warning is the pre-existing React exhaustive-deps warning at
`web/src/app/NexusWorkspaceApp.tsx:228`.

The latest native-CI remediation slice fixed OS temporary-root canonicalization,
quota lease token collisions, and the Wails macOS About plist ampersand. The
installer `@latest` regression was removed and `make quality-full` plus
`make build-desktop-wails` passed locally. Native Windows/macOS execution is
still `NOT VERIFIED` until these changes are published and CI is rerun; no
commit or push was created.

The subsequent handoff cancellation slice also passed `make quality-full`
(exit 0; Web 62/320; Go tests/race/vet/lint/build/security). The goal remains
active while native platform evidence and deeper runtime/process ownership are
still outstanding.

WorkPlan DTO typing was then completed for the stable Web facade responses;
the subsequent `make quality-full` also passed with exit 0 (Web 62/320; Go
tests/race/vet/lint/build/security). No commit was created.

The handoff dependency boundary was then exercised through the full quality
gate: `make quality-full` passed with exit 0 (Web 62/320; Go
tests/race/vet/lint/build/security). Native platform evidence remains absent.

The TUI registry ownership slice was then verified by the full gate:
`make quality-full` passed with exit 0 (Web 62/320; Go
tests/race/vet/lint/build/security). No commit was created.

Local cross-build evidence was also collected for the CLI: Windows amd64
PE32+ and macOS amd64/arm64 Mach-O binaries, plus `go test -c` for
`internal/control/registry` on those targets. This does not replace native
execution; native smoke/test and same-SHA CI evidence remain pending.

The cancellable IPC slice was then verified by `make quality-full`, which
passed with exit 0 (Web 62/320; Go tests/race/vet/lint/build/security).

## Handoff — Human Intervention / Resume P0 — 2026-09-11

The P0 duplicate-external-side-effect path is closed locally. The durable
runner contract is documented in
`docs/validation/NEXUS_HUMAN_INTERVENTION_SAFETY_REPORT.md`, with the execution
plan in `docs/superpowers/plans/2026-09-11-human-intervention-safety.md`.

Key safety facts: unresolved dispatches remain
`UNKNOWN_EXTERNAL_OUTCOME`; only an explicit typed reconciliation option can
mark that package verified; safe retry requires a proven pre-dispatch failure;
resolution and resume intent commit atomically in SQLite; duplicate semantic
decisions do not create a second resume; stale/conflicting decisions have no
business side effect; and worker generations cannot delete newer ownership.

All scoped Go/Web gates passed. A later concurrent AGY worktree change makes
three unrelated AGY fixtures fail in the global suite; that area was not
modified here. The branch remains uncommitted and retains the pre-existing
dirty worktree changes. This closes only the first promotion gate;
Attention Center final UX, Project Intelligence, provider UX/failover and
native platform evidence remain outside this campaign.

## Handoff — Final closure continuation — 2026-09-11

The current working tree remains based on `2925ca746c198334f20d1e0cef7feb51e4f4e3`; no commit or push was created. Preserve the broad pre-existing staged and unstaged changes.

The native SkillCatalog slice is now used by Agent prompt execution and
Composer selection. Generic `ExecutionGuidance` is available to Intelligence,
while the Maestro field/type is compatibility-only. Legacy `MaestroGates` are
kept separate from catalog Skills. Mission allocation serializes a durable
explainability projection with desired affinity and actual provider/profile/model
selection. Codex usage snapshots accept only prior identities in the same
persisted account scope.

Fresh local verification passed: `go test ./... -count=1`,
`go test -race ./... -count=1`, `go vet ./...`, `make build`,
`make build-desktop`, `make web-verify`, and `git diff --check`.

The campaign is still `NO-GO` for release. Remaining blockers are external or
runtime evidence: authenticated Mission execution with a real
`ValidationEvidenceStream` ID, complete Codex/OpenCode/AGY provider and
failover/PIN/model-escalation/handoff scenarios, native Windows/macOS same-SHA
validation, and overnight acceptance. Model inventory/escalation remains a
contract-level slice in the live Mission executor; do not claim those scenarios
as PASS without fresh evidence.

## Handoff — 2026-09-12 scanner and report continuation

Current HEAD is `a58cca4d73bdfd55678649b30c0ca64f73b7d410`, with uncommitted
scanner/report changes preserved in the worktree. The existing Project
Intelligence scanner now emits bounded observed facts for operational commands,
frameworks, Go workspace uses, nested packages and CI workflow commands. The
existing run routing report now includes the persisted intent decision when the
WorkPlan is available and rejects corrupt durable JSON.

Focused contextsnapshot, report/intent and full Nexus tests passed. A real local
AGY bootstrap attempt remained `UNVERIFIED` because sudo authentication failed
while configuring `/etc/hosts`; no authenticated Mission stream ID exists.
Continue with model/escalation integration, evidence/native/overnight scenarios,
then rerun all quality gates. Do not report GO from local unit/package tests.

## Handoff — 2026-09-12 final local verification

The persisted routing contract is now also represented in `web/src/types.ts`:
Agent score/confidence/reason and the identity-scoped selected account are
available to the existing routing report client. No new UI or store was added.

Fresh gates all passed: `make web-verify`, `go test ./... -count=1`,
`go test -race ./... -count=1`, `go vet ./...`, `make security`, `make build`,
`make build-desktop`, and `git diff --check`. `./nexus doctor --json` was
recorded on Linux/amd64 at `2026-09-12T01:36:44Z`.

Release remains `NO-GO`: no authenticated Mission produced a durable
ValidationEvidenceStream ID; live provider/account/model failover, native
Windows/macOS, overnight and native desktop shell evidence remain
`UNVERIFIED`/`SKIPPED`. No commit or push was created.
# Atualização 2026-09-11 — Alinhamento do destino de instalação

`install.sh` não fixa mais o destino em `~/.local/bin` quando já existe um
`nexus` anterior no `PATH`. Ele instala no primeiro diretório resolvido pelo
shell, preservando `~/.local/bin` como fallback de instalação nova.

Verificação: `bash -n install.sh`, `go test ./internal/release` e
`git diff --check` passaram. Não houve commit/push.

O instalador cria somente `nexus`; o alias `ai` não é mais preservado nem
recriado durante a instalação.
# 2026-09-11 — Auditoria independente final

Candidate `c8747481fc04c65614b38b5c9d9da106f56858c3` terminou em
`NO_GO_FOR_MERGE`: trust root placeholder, updater sem instalação segura de
archive/health-check, browser visual não independente e ausência de evidência
nativa/release same-SHA. Próxima ação: corrigir blockers e repetir CI nativo e
esta auditoria no mesmo SHA.

## Handoff — 2026-09-12 automatic delegation

Worktree dedicado: `/projetos/tools/IAPro-Nexus-Workspace-OS-Handoff-2026-08-29/nexus-auto-delegation`.
Base: `37b9ddec49ce72d869bbeb2ad402a1d4dd2bc674`; branch
`feat/nexus-auto-delegation`; sem commit/push.

Implementado o policy slice em `internal/nexus/delegation.go` e integrado aos
boundaries existentes: WorkPlan facts, Flow decomposition, MissionRunner,
`MatchAgents`, rich `AgentSpec` para Agents criados e routing report. A suíte
Nexus e os gates completos passaram. `make build` e `make build-desktop`
continuam limitados pelo stamping VCS do ambiente; os binários compilam com
`go build -buildvcs=false`. O passthrough PTY provider-interativo de
`nexus agy` permanece fora do E2E de delegação; o caminho Flow → WorkPlan →
Mission está verificado. Próximo passo: commitar esta branch e aguardar o
fim do hardening antes de qualquer rebase/merge.

Base SHA: `37b9ddec49ce72d869bbeb2ad402a1d4dd2bc674`.

## Handoff — production finalization candidate

Candidate code SHA: `e53f8352e4c3647632ebac7877cf0a6a3bd62647` on
`feat/nexus-maximum-delivery`. The release hardening commit covers updater
signatures/trust-root injection, installer fail-closed behavior, stricter CI
gates, native installer smoke definitions, browser isolation, Mission project
authorization, cloudflared pin verification and frontend contract fixes.

Local Linux quality/build/security/race/browser/update-gate evidence is fresh.
Do not describe the branch as release-ready: production trust-root
configuration, native Windows/macOS and hosted same-SHA CI, PowerShell/native
terminal gates, and authenticated Mission evidence still require execution in
the appropriate environments. No push, merge, deploy or production secret was
performed by this campaign. See `DEV/validation/production-finalization/FINAL_REPORT.md`.
