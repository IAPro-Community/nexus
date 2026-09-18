---
name: Fechar experiência comercial Nexus
overview: Fechar o gap nº 1 (promoção adaptativa de prompt dentro de `nexus <provider>` para Mission Team, modo híbrido ASK/AUTO), cobrir o ciclo rejection→fix→re-review no teste canônico, migrar o module path para `IAPro-Community/nexus`, produzir prova com providers reais, regenerar evidência CURRENT no SHA congelado e fechar release engineering (CI same-SHA, branch protection, soak curto + instrução 8h).
todos:
  - id: p0-concurrency
    content: "Fase P0: permitir runtimes Codex concorrentes no mesmo perfil com lease TUI compartilhado, lock exclusivo de manutenção/app-server e reserva atômica de Lead"
    status: completed
  - id: c-gomod
    content: "Fase C: migrar module path para github.com/IAPro-Community/nexus (go.mod, imports, Makefile, goreleaser, ci ldflags, docs) + ADR; make quality"
    status: completed
  - id: a-promotion-core
    content: "Fase A: pacote internal/nexus/promotion (LineShadow zero-leak + Promoter ASK/AUTO/OFF) com testes"
    status: completed
  - id: a-promotion-cli
    content: "Fase A: wire no CLI attach (attachInteractiveLeadRuntime) com banner fanout, ClearLine por ProviderSpec, síntese ao Lead"
    status: completed
  - id: a-promotion-web
    content: "Fase A: wire no Web terminal (frame promotion + TerminalActionDialog + i18n) e docs"
    status: completed
  - id: b-e2e-test
    content: "Fase B: teste canônico CHANGES_REQUESTED→fix→re-review→APPROVED→Verifier com desired vs actual; incluir em overnight-smoke"
    status: completed
  - id: f1-ci-green
    content: "Fase F1: corrigir staticcheck unused e testes Windows; push; CI same-SHA com 13 checks verdes"
    status: in_progress
  - id: d-real-providers
    content: "Fase D: Mission real por provider autenticado, evidência em DEV/validation/current/real-provider-mission, PASS para Codex e UNVERIFIED explícito para providers sem prova"
    status: completed
  - id: e-evidence
    content: "Fase E: congelar SHA, release-proof/quality/attention-e2e/overnight-smoke/soak 45m, atualizar RELEASE_STATUS e MISSION_TEAMS_REPORT"
    status: pending
  - id: f2-protection
    content: "Fase F2: branch protection (main required checks; feat sem force-push), HANDOFF/WORKLOG, commit docs de evidência"
    status: pending
isProject: false
---

# Fechar experiência comercial + prova real (base: `cdb81f1`)

Decisões já tomadas com você: escopo = 5 itens; mecanismo = **híbrido** (intercepta e classifica; ASK confirma, AUTO promove); go.mod = **migrar** para `github.com/IAPro-Community/nexus`; soak = **curto agora (45 min) + 8h documentado**.

O worktree atual está na branch `feat/nexus-maximum-delivery` e contém alterações
locais não commitadas; preservá-las e não fazer commit/push automaticamente. Não
reescrever `PRODUCTION_NO_GO` para GO; só evidência nova.

## Ordem de execução

```mermaid
flowchart LR
  P0[Fase P0 concorrência de runtimes] --> C[Fase C go.mod rename]
  C --> A[Fase A promocao adaptativa]
  A --> B[Fase B teste canonico rejection-fix-rereview]
  B --> D[Fase D prova providers reais]
  D --> E[Fase E congelar SHA e regenerar CURRENT]
  E --> CI[Fase F1 CI hosted same-SHA (autorização)]
  CI --> F2[Fase F2 branch protection (autorização)]
```

## Fase P0 — concorrência de runtimes supervisionados

O mesmo perfil representa credenciais/configuração compartilhadas, não
propriedade exclusiva de uma execução. `nexus codex`, `nexus agy` e os demais
providers continuam supervisionados em TTY, mas cada invocação recebe runtime,
terminal, geração, sessão e Lead independentes.

- `nexus-tui.lock` usa leases compartilhados (`AcquireTUILease`) para TUIs;
  preparação, migração e `codex app-server` usam manutenção exclusiva
  (`TryAcquireMaintenanceLock`). Probes de quota não iniciam app-server enquanto
  houver qualquer lease TUI.
- Preparação sob lock exclusivo ocorre apenas quando necessária. Perfil pronto é
  reutilizado durante uma TUI ativa; perfil incompleto ou manutenção incompatível
  falha de forma transitória e explícita, sem sugerir anexar/encerrar outro runtime.
- A seleção supervisionada não filtra perfil Codex ocupado. Antes do lançamento,
  um Lead ocioso é reivindicado atomicamente por transição `STARTING`; concorrência
  cria outro Lead e cleanup devolve a reserva se o lançamento falhar.
- Stop/detach atua somente sobre o runtime e a geração reivindicados. A sintaxe
  pública da CLI e os contratos JSON permanecem inalterados; AGY, Claude, Gemini,
  OpenCode e Cursor não recebem essa trava de perfil.

### Aceitação P0

- Dois leases TUI no mesmo `CODEX_HOME` são aceitos; manutenção exclusiva falha
  enquanto qualquer um estiver ativo e novos leases falham durante manutenção.
- Duas inicializações concorrentes do mesmo workspace/perfil recebem runtime IDs,
  endpoints e Leads distintos; liberar uma não altera a outra.
- Testes focados/race e compilação Unix/Windows cobrem locks, preparação,
  reserva atômica e ausência de mensagem singleton “já está em execução”.

Esta fase foi concluída localmente e libera a migração do module path, já validada
no mesmo worktree. Push, rulesets e branch protection continuam condicionados à
autorização explícita do maestro.



Rename primeiro para que todo código novo já nasça com o import path final.

---

## Fase C — Module path `github.com/IAPro-Community/nexus`

- `go mod edit -module github.com/IAPro-Community/nexus` + substituição mecânica nos 279 arquivos `.go` (722 imports).
- Atualizar `MODULE=` em [Makefile](Makefile) (linha 2 e ldflags), [.goreleaser.yaml](.goreleaser.yaml) ldflags, [.github/workflows/ci.yml](.github/workflows/ci.yml) ldflags desktop (linhas ~480 e ~535 usam `-X github.com/kivervinicius/ai-cli/internal/update.ProductionTrustRootHex`), [BUILD.md](BUILD.md), `git clone` em [docs/getting-started/installation.md](docs/getting-started/installation.md).
- ADR `docs/architecture/ADR-module-path-iapro-community.md` (contexto: repo transferido; sem consumidores Go externos; breaking aceito antes do 1.0).
- Gate: `go build ./... && make quality`. Commit isolado `refactor(go): migrate module path to IAPro-Community/nexus`.

## Fase A — Promoção adaptativa em `nexus <provider>` (gap nº 1)

Restaurar a capacidade removida em `95f23e3` (`interactive_delegation.go`), agora com line-shadow zero-leak e política ASK/AUTO/OFF.

**Onde:** o `SessionHost` não tem `*nexus.Nexus`; a classificação roda nos dois processos que têm: CLI attach ([internal/app/control_cmd.go](internal/app/control_cmd.go) `attachInteractiveLeadRuntime`) e Web terminal ([internal/control/web/handler_terminal.go](internal/control/web/handler_terminal.go), `h.nexus`).

**Novo pacote `internal/nexus/promotion`:**

- `LineShadow` — máquina de estados byte-a-byte: forward transparente de tudo (TUI continua ecoando), cópia-sombra da linha atual; ESC/CSI/OSC e bracketed paste (`\x1b[200~…\x1b[201~`) passam sem tocar a sombra; Ctrl-keys passam e resetam; backspace edita a sombra; **Enter é retido** até a decisão.
- `Promoter.Decide(ctx, projectID, line, mode)` → reusa `DecideIntentForProject` ([internal/nexus/intent_router.go](internal/nexus/intent_router.go)), `DecideDelegation` ([internal/nexus/delegation.go](internal/nexus/delegation.go)) e `SelectMissionTeam` ([internal/nexus/mission_team.go](internal/nexus/mission_team.go)). Resultado: `KEEP_WITH_LEAD | PROMOTE | ASK`.
- Ações: `KEEP` → forward `\r`; `PROMOTE` → envia `ClearLine` ao PTY (default Ctrl+U, campo novo em `ProviderSpec` para override), banner via fanout `\r\n[nexus] …\r\n`, `DecomposePromptIntoFlowProposal` → `CreateWorkPlan` → `StartMissionRun(planID, leadAgentID, contract, autonomous=true)`; ao terminar, síntese ao Lead via `AskAgent`/`SubmitPrompt` (padrão antigo `runInteractiveDelegatedMission`). `ASK` → banner "promover? [y/n]"; próxima tecla consumida pela sombra: `y` promove, `n` forward `\r`.
- Modo vem de `session.Labels["nexus.delegation_mode"]` com override `NEXUS_DELEGATION_MODE`; `OFF` = passthrough puro (sem sombra).

**Web:** mesmo `LineShadow` no lado servidor do WS (`case "input"`); novo frame `promotion` (padrão do frame `attention`) e resposta `promotion_answer`; UI reusa `TerminalActionDialog` + i18n `terminal.promotion.`*.

**Testes obrigatórios:** zero-leak (ESC/CPR/UTF-8/JSON RPC nunca ao PTY — espelhar `host_test.go` `CmdInputForwardsTerminalBytesVerbatim`), bracketed paste não dispara, OFF passthrough, ASK y/n, AUTO promove uma única vez (sem duplo envio), integração CLI: linha "implemente backend e frontend…" vira WorkPlan + MissionRun.

Atualizar comentário em `control_cmd.go:564-567` e [docs/internal/FEATURE_INVENTORY.md](docs/internal/FEATURE_INVENTORY.md); nova seção em `docs/product/autonomous-delivery.md`.

## Fase B — Teste canônico com rejection→fix→re-review

- Estender [internal/nexus/runner/review_test.go](internal/nexus/runner/review_test.go) `TestMissionTeamPlannerBuilderReviewerVerifierPipeline` (ou novo `…WithRemediationCycle`): Reviewer `CHANGES_REQUESTED` com 3 findings → Builder corrige 2 → Reviewer: 1 aberto + 1 novo (ledger distingue) → corrige → `APPROVED` → Verifier sem write turn → `COMPLETED_VERIFIED`.
- Assertar `ResourcePreference`/desired vs actual por fase (Planner PREFER codex, Builder PREFER agy, Reviewer PREFER claude, Verifier AUTO) via `RoutingDecisionJSON`/`HandoffJSON` com executor fake.
- Incluir no regex de `overnight-smoke` no [Makefile](Makefile).

## Fase F1 — CI same-SHA verde

O lint, typecheck, testes, `go vet`, build e `make quality` locais estão verdes
neste worktree. Push e a confirmação dos 13 checks hosted permanecem
condicionados à autorização explícita do maestro; portanto este item segue
`in_progress`.

- Remover wrappers unused `toUsageTableRows`/`toSessionTableRows` em [internal/tui/usage_table.go](internal/tui/usage_table.go) (staticcheck U1000).
- Tornar platform-aware: `TestIsWithinAllowedRoots` ([internal/control/web/handlers_fs_test.go](internal/control/web/handlers_fs_test.go)), `TestNoBrowserBinDirInstallsInertHelpers` ([internal/runtime/credentials_test.go](internal/runtime/credentials_test.go)), `TestResolveE2ERootUsesExplicitNonTemporaryDirectory` ([scripts/nexus-e2e-local_test.go](scripts/nexus-e2e-local_test.go)) — skip/ajuste em `windows`.
- Push; `gh run watch`; iterar até os 13 checks do gate de [release.yml](.github/workflows/release.yml) ficarem verdes (Desktop com trust root já configurada).

## Fase D — Prova com providers reais

- `nexus providers status --json` → lista autenticados nesta máquina.
- Para cada autenticado: projeto git temporário + `NEXUS_DATA_DIR` isolado, `nexus run "<goal pequeno>"` autônomo; capturar `run_id`, estado final, `RoutingDecisionJSON` (desired vs actual, fallback), handoffs, quota status (UNKNOWN não participa — registrar).
- Salvar em `DEV/validation/current/real-provider-mission/<provider>.json|md`; o
  Codex autenticado foi validado em `COMPLETED_VERIFIED` com review read-only
  distinto; providers não autenticados = `UNVERIFIED` explícito. Nada de
  `PASS` sem run real.

## Fase E — Congelar SHA e regenerar CURRENT

- Convenção (registrar em `DEV/INDEX.md`): `GIT_SHA` = SHA do código validado; commit de evidência é docs-only (`EVIDENCE_COMMIT`), verificável por `git diff <GIT_SHA>..HEAD --stat` só em `DEV/validation/`.
- No SHA congelado: `make quality`, `make release-proof` (verificar se o erro `covdata` ainda ocorre; Makefile atual só usa `go tool cover`), `make attention-e2e`, `make overnight-smoke`, `make overnight-soak DURATION=45m` (background).
- Atualizar [DEV/validation/current/RELEASE_STATUS.md](DEV/validation/current/RELEASE_STATUS.md) e [MISSION_TEAMS_REPORT.md](DEV/validation/current/MISSION_TEAMS_REPORT.md): `GIT_SHA`/`FINAL_SHA` = novo SHA, `WORKTREE_FROZEN: YES`, matriz de gates real, 8h soak = `NOT_RUN (instruction: make overnight-soak DURATION=8h)`, Windows/macOS = resultado do CI same-SHA. Veredito continua honesto (sem GO se algo faltar).

## Fase F2 — Branch protection + ADRs

- `main`: ruleset via `gh api` com os 13 required checks (nomes exatos de `release.yml:37-51`), strict up-to-date, bloquear force-push/deletion.
- `feat/nexus-maximum-delivery`: bloquear force-push/deletion apenas (required checks aqui bloqueariam seu push direto — decisão registrada).
- Commit final `docs(release): …` e `HANDOFF.md`/`WORKLOG.md` atualizados.

## Fora de escopo

- Executar o soak de 8h nesta sessão (fica instrução + target).
- Reescrever relatórios históricos NO-GO.
- Frente de UI além do banner/dialog de promoção.
