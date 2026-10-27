## Continuidade atual — beta.11 sincronizado e instalado — 2026-10-27

- Branch `release/v1.0.0` em `b416bb90` (= `origin/release/v1.0.0`) + commits locais `5e3eff4e` (feature auth/usage/HUD), `11f3d771` (teclado Codex/PTY), `d61b0dcc` (gates/upstream) e docs; push pendente.
- `GOTOOLCHAIN=go1.25.14 make quality` PASS (114 suítes/710 testes, golangci-lint 0 issues, version-contract). Falhas upstream reproduzidas em worktree limpo e corrigidas.
- Instalação local: `Applications/IAPro Nexus.app` + HUD (backup `Applications-backup-20261027-0937`); Desktop + HUD vinculados, HTTP 200 e UI em `Nexus v1.0.0-beta.11`.
- Próxima ação: push de `release/v1.0.0` e smoke do Novo Agente pelo HUD/teclado Codex no build instalado. Distribuição NO-GO (Developer ID/notarização/arm64).

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
- Importação real: sessão Gemini copiada para um perfil Nexus autenticado e padrão; repetição idempotente; Claude sem token reutilizável (`NO_SESSION`). Ainda não houve chamada de modelo Gemini. Ver `validation/linux-100-2026-10-25/PROVIDER_SESSION_IMPORT_VALIDATION.md`.
- Estado atual: **NO_GO_LINUX — 169 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN (191)**. Permanecem pendentes release estável/assinatura e SHA imutável, handoff/continuidade real e preview P5 na janela Wails, alpha/foco/múltiplos monitores e GPU/compositor Wayland.

## Evidências correntes

- [Login Claude Code no Windows — correção `auth login`](validation/CLAUDE_LOGIN_WINDOWS_20261026.md)
- [Correções Windows de agentes e terminais — YOLO, scrollback e anexação](validation/TERMINALS_AGENTS_YOLO_SCROLL_20261026.md)
- [Relatório Linux](validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md)
- [Matriz e resumo JSON](validation/linux-100-2026-10-25/TEST_MATRIX.md) · [NEX único](validation/linux-100-2026-10-25/MASCOT_VALIDATION.md)
- [P5 — reteste atual Composer/Flow](validation/linux-100-2026-10-25/P5_CURRENT_BROWSER_RECHECK.md) · [P5 — integração e MissionRunner](validation/linux-100-2026-10-25/P5_COMPOSER_MISSION_VALIDATION.md)
- [Reconciliação PRs](validation/linux-100-2026-10-25/PULL_REQUEST_RECONCILIATION_CURRENT.md) · [FIX-034](validation/linux-100-2026-10-25/FIXES_APPLIED.md)


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

# DEV Master Index

> **Documentação pública canônica:** [README](../README.md) →
> [docs/README](../docs/README.md) →
> [Como funciona](../docs/product/how-it-works.md) →
> [Arquitetura](../ARCHITECTURE.md).
>
> `DEV/` é ledger de engenharia, specs e evidência. Relatórios históricos não
> substituem a documentação pública nem o
> [status corrente de release](validation/current/RELEASE_STATUS.md).

## Homologação Linux corrente — 26/10/2026

- [Relatório Linux AMD64](validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md), [matriz](validation/linux-100-2026-10-25/TEST_MATRIX.md) e [resumo JSON](validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_SUMMARY.json): branch `release/v1.0.0`, HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`, overlay compilado/instalado `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c` (1.430 fontes, worktree modificada), **NO_GO_LINUX** (167 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN; 189 casos). A versão continua beta.10; sem release estável `v1.0.0` ou SHA imutável do overlay.
- Contrato confirmado: um único mascote NEX; sem modo ou renderer Companion 2D. O HUD Wails/XWayland do overlay corrente mostra um NEX e abre controles; cinco suítes focadas (33/33), E2E HUD e arte oficial passaram. Alpha, foco físico e interação completa no compositor Wayland permanecem pendentes. O P9 determinístico de 8 h foi concluído; provider real/continuidade não foram exercitados.
- `quality:full` passou com 110 arquivos/681 testes/build; E2Es Composer completo/vago e regressão de cópia passaram. GoReleaser gerou tar.gz/DEB no overlay corrente, com checksums e ciclo dpkg isolado aprovados. Desktop/HUD instalados estão ativos com Core HTTP 200.
- [P5 preview atual](validation/linux-100-2026-10-25/P5_CURRENT_BROWSER_RECHECK.md), [distribuição](validation/linux-100-2026-10-25/DISTRIBUTION_VALIDATION.md), [mascote único](validation/linux-100-2026-10-25/MASCOT_VALIDATION.md), [reconciliação das PRs](validation/linux-100-2026-10-25/PULL_REQUEST_RECONCILIATION_CURRENT.md) e [handoff](HANDOFF.md).

## Fechamento beta.10 — 22/10/2026

- [Reconciliação de código, documentação e evidências](validation/current/BETA10_GAP_CLOSURE_20261022.md):
  delta de 27 commits, contratos públicos, seis recursos adicionados ao catálogo,
  CI/Windows exact-SHA, falha de formatação e lista de gates pendentes.
- [Changelog público](../CHANGELOG.md) e [status de release](../docs/release-status.md)
  não significam aprovação de plataforma; a certificação depende de um HEAD congelado.

## Contratos públicos v1

- [Como o Nexus funciona](../docs/product/how-it-works.md)
- [Coordenação entre Agents](../docs/product/agent-coordination.md)
- [Runtime da Mission](../docs/architecture/mission-runtime.md)
- [Guardrails](../docs/architecture/guardrails.md)
- [Modelo mental](../docs/product/mental-model.md)
- [Matriz de funcionalidades](../docs/reference/feature-matrix.md)


Guia e índice de navegação estruturada da documentação de engenharia e governança do IAPro Nexus.

**Current release status for HEAD:**

**Atualização corrente de homologação Linux (2026-10-26):** o overlay `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c` passa `quality:full` (110 arquivos/681 testes/build), E2Es Composer completo/vago e build/instalação CLI/Desktop/HUD AMD64 com health HTTP 200. A importação local de sessão Gemini passou e é idempotente, mas ainda requer refresh; um smoke real direto do Codex via CLI Nexus passou em modo efêmero/read-only; o runtime dentro da janela Wails e handoff/continuidade ainda não foram validados. A release segue beta.10, sem publicação estável/assinatura; alpha/GPU/Wayland e proveniência imutável também bloqueiam GO. Ver [relatório](validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md), [P5 atual](validation/linux-100-2026-10-25/P5_CURRENT_BROWSER_RECHECK.md), [importação de sessão](validation/linux-100-2026-10-25/PROVIDER_SESSION_IMPORT_VALIDATION.md), [runtime Codex](validation/linux-100-2026-10-25/RUNTIME_VALIDATION.md), [updater](validation/linux-100-2026-10-25/UPDATER_VALIDATION.md), [correções](validation/linux-100-2026-10-25/FIXES_APPLIED.md) e [matriz](validation/linux-100-2026-10-25/TEST_MATRIX.md). Resultado: **NO_GO_LINUX** (167 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN). Produto: um NEX único; Companion 2D está fora do escopo.

**Homologação Linux AMD64 (atualizada em 2026-10-26):** [relatório](validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md) · [matriz](validation/linux-100-2026-10-25/TEST_MATRIX.md) · [resumo JSON](validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_SUMMARY.json) · [importação de sessão](validation/linux-100-2026-10-25/PROVIDER_SESSION_IMPORT_VALIDATION.md) · [P5 preview](validation/linux-100-2026-10-25/P5_CURRENT_BROWSER_RECHECK.md) · [NEX único](validation/linux-100-2026-10-25/MASCOT_VALIDATION.md) · [Attention P2](validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md) · [P3 notificações](validation/linux-100-2026-10-25/MISSION_NOTIFICATIONS_VALIDATION.md) · [P4 Project Intelligence](validation/linux-100-2026-10-25/PROJECT_INTELLIGENCE_VALIDATION.md) · [P6 fallback de Agents](validation/linux-100-2026-10-25/AGENT_FALLBACK_VALIDATION.md) · [P7 handoff/PTY/race](validation/linux-100-2026-10-25/P7_HANDOFF_PTY_VALIDATION.md) · [atualizador](validation/linux-100-2026-10-25/UPDATER_VALIDATION.md) · [Skills](validation/linux-100-2026-10-25/SKILL_SESSION_VALIDATION.md). Resultado `NO_GO_LINUX` (167 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN; base `7a04f1b4`, overlay `101e6e62…`). Produto: um NEX único; a PR #55/Companion 2D não é gate. Runtime/handoff externo além do smoke direto Codex, fluxo Composer Wails, alpha/GPU/Wayland e assinatura de produção seguem pendentes. [`validation/current/RELEASE_STATUS.md`](validation/current/RELEASE_STATUS.md)

Snapshot histórico CLI/P2 e direção do mascote: help/flags da CLI validados antes do Core; dois ciclos Web isolados passaram; Companion 2D permanece ausente. Resultado naquele overlay: 105/1/15/5.

Follow-up nativo do Attention Center (2026-10-25): [validação de filtros, navegação para decisão e layout](validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md). O cartão agora abre a missão no workspace; a evidência vem de fixture isolada e inclui teste SQLite após reopen. Fingerprint do overlay: `7368531977b7422eccdf00e7c676f57d440e8d736d9e114bcdc071b9d5885fe6`; o veredito geral continua NO_GO_LINUX.

Follow-ups da mesma campanha: [PRs #57/#58 e testes Linux](validation/linux-100-2026-10-25/PR578_FOLLOWUP.md) · [PR #61, busca do portal e contrato do mascote](validation/linux-100-2026-10-25/PR61_DOCS_SITE_VALIDATION.md).

Latest Windows beta.7 evidence: [BETA7_WINDOWS_HOMOLOGATION_20261007.md](validation/current/BETA7_WINDOWS_HOMOLOGATION_20261007.md). NSIS Path-state smoke: [BETA7_NSIS_PATH_STATE_SMOKE_20261007.md](validation/current/BETA7_NSIS_PATH_STATE_SMOKE_20261007.md).

Reports under `validation/FINAL_*.md` are historical unless they live in `validation/current/` and carry matching `GIT_SHA`.

Release evidence uses `GIT_SHA` for the validated code revision and, when a
documentation-only evidence commit is authorized, `EVIDENCE_COMMIT` for that
commit. A dirty worktree is recorded as `WORKTREE_FROZEN: NO`; it must not be
represented as a frozen release candidate. The current real-provider attempt
and its honest status live in [`validation/current/real-provider-mission/`](validation/current/real-provider-mission/).

O código canônico utiliza o binário `nexus`, mantendo `ai` como alias transparente de compatibilidade.

---

## Agent Interop incorporado à linha v1 beta

- [`SPECS/NEXUS_V1_AGENT_INTEROP.md`](SPECS/NEXUS_V1_AGENT_INTEROP.md): spec canônica da V1 para ExecutionRole, Agent Projection, capability truth e interoperabilidade de agents.
- [`docs/architecture/ADR-agent-interop-projection.md`](../docs/architecture/ADR-agent-interop-projection.md): decisão de projetar o modelo canônico de Agents para hosts externos sem criar um segundo runtime/orquestrador.

A decisão de release de 24/09/2026 incorporou esse escopo ao candidato
`v1.0.0-beta.3`. Os nomes de arquivo “1_1” são preservados por rastreabilidade,
não representam mais uma linha de release separada. O status operacional segue
`docs/release-status.md`.

---

## 1. Governança e Estado Ativo

- [`SPECS/NEXUS_CONDUCTOR.md`](SPECS/NEXUS_CONDUCTOR.md): Spec técnica do Nexus Conductor — orquestração nativa de Goals → Missions.
- [`SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md`](SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md): Plano executável Luna para CI, quota, pools, terminal supervisionado e continuidade nos três SOs.
- [`HANDOFF.md`](HANDOFF.md): Estado atual de execução e próxima ação recomendada.
- [`CONTEXT.md`](CONTEXT.md): Arquitetura corrente, runtime, drivers e restrições operacionais.
- [`SPECS/ACTIVE.md`](SPECS/ACTIVE.md): Contrato ativo de trabalho e critérios de aceitação vigentes.
- [`VERIFY.md`](VERIFY.md): Evidências e histórico consolidado de validações e gates.
- [`NEXUS_1_FINAL_ACCEPTANCE.md`](NEXUS_1_FINAL_ACCEPTANCE.md): ledger executável da aceitação RC do Nexus 1.0.
- [`WORKLOG.md`](WORKLOG.md): Diário de bordo detalhado de todas as implementações e mudanças.
- [`../DOCUMENTATION_COMPLETE.md`](../DOCUMENTATION_COMPLETE.md): entrega de documentação pública; inventário e ledger estão em `docs/internal/`.

---

## 2. Arquitetura, Interoperabilidade e Especificações Canônicas

- [`docs/architecture/quota/account-isolation.md`](../docs/architecture/quota/account-isolation.md): fronteira canônica de estado por conta.
- [`docs/architecture/quota/identity-scoping.md`](../docs/architecture/quota/identity-scoping.md): identidade persistida e versionamento de credencial.
- [`docs/architecture/providers/registration-lifecycle.md`](../docs/architecture/providers/registration-lifecycle.md): separação entre instalação e registro.
- [`docs/architecture/agent-routing.md`](../docs/architecture/agent-routing.md): matching canônico de Agents persistentes.
- [`docs/architecture/resource-routing.md`](../docs/architecture/resource-routing.md): seleção de recursos e políticas do scheduler.
- [`docs/architecture/mission-teams.md`](../docs/architecture/mission-teams.md): projeção de Mission Teams sobre Flow, WorkPlan e MissionRunner.
- [`docs/architecture/session-continuity.md`](../docs/architecture/session-continuity.md): estados e limites de continuidade.
- [`docs/architecture/ADR-module-path-iapro-community.md`](../docs/architecture/ADR-module-path-iapro-community.md): decisão do module path canônico.
- [`docs/architecture/quota/legacy-cache-migration.md`](../docs/architecture/quota/legacy-cache-migration.md): tratamento não destrutivo de cache legado.

- [`NEXUS_V1_ARCHITECTURE.md`](NEXUS_V1_ARCHITECTURE.md): Arquitetura mestre do Workspace OS, runtimes e store.
- [`NEXUS_CANONICAL_ALIGNMENT.md`](NEXUS_CANONICAL_ALIGNMENT.md): Alinhamento canônico entre CLI, Web e drivers de provedores.
- [`NEXUS_CAPABILITY_PRESERVATION.md`](NEXUS_CAPABILITY_PRESERVATION.md): Matriz de preservação honesta de capacidades.
- [`NEXUS_V1_AGENT_MODEL.md`](NEXUS_V1_AGENT_MODEL.md): Modelo de agentes duráveis e persistentes.
- [`NEXUS_V1_MAESTRO_INTEGRATION.md`](NEXUS_V1_MAESTRO_INTEGRATION.md): Integração profunda e governança com o Orquestrador Maestro.
- [`BRANDING_AND_ASSETS.md`](BRANDING_AND_ASSETS.md): Identidade visual, catálogo de assets, favicon, script de extração e propostas de design.

---

## 3. Matrizes de Plataforma, Provedores e Segurança

- [`FINAL_PROVIDER_MATRIX.md`](FINAL_PROVIDER_MATRIX.md): Tabela de suporte e comportamento de Codex, AGY, Claude, OpenCode, Gemini e Cursor.
- [`FINAL_PLATFORM_MATRIX.md`](FINAL_PLATFORM_MATRIX.md): Compatibilidade entre Linux, macOS e Windows.
- [`FINAL_SECURITY_REPORT.md`](FINAL_SECURITY_REPORT.md) & [`NEXUS_MAXIMUM_DELIVERY_SECURITY.md`](NEXUS_MAXIMUM_DELIVERY_SECURITY.md): Isolamento de processos, sandbox, redação de credenciais e integridade de D-Bus/keyrings.

---

## 4. Frontend e Validação Canônica

- [`validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md): Último gate do frontend (`make web-verify`).
- [`validation/FRONTEND_HISTORY.md`](validation/FRONTEND_HISTORY.md): Histórico compacto dos gates de frontend.
- [`validation/CURRENT_STATE_AUDIT.md`](validation/CURRENT_STATE_AUDIT.md): Inventário evidence-backed da campanha atual.
- [`validation/FINAL_CLOSURE_REALITY_AUDIT.md`](validation/FINAL_CLOSURE_REALITY_AUDIT.md) e [`validation/FINAL_CLOSURE_CHECKPOINTS.md`](validation/FINAL_CLOSURE_CHECKPOINTS.md): auditoria e checkpoints resumíveis da consolidação final.
- [`validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md`](validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md): Relatório final de plataforma, Desktop, Update e release.
- [`../docs/product/autonomous-delivery.md`](../docs/product/autonomous-delivery.md): experiência de entrega autônoma para usuários.
- [`../docs/operations/mission-teams.md`](../docs/operations/mission-teams.md): operação, recuperação e evidências de Mission Teams.
- [`../docs/testing/mission-teams.md`](../docs/testing/mission-teams.md): cenários e gates de teste do pipeline.
- Após alterações em `web/`, execute `make web-verify` (ou `npm --prefix web run verify`).
- Em caso de inconsistência visual na Web, faça o rebuild completo do binário com `make build` e reinicie o processo.

---

## 5. Histórico e Relatórios de Release

- [`FINAL_RELEASE_REPORT.md`](FINAL_RELEASE_REPORT.md): Relatório de fechamento de releases estáveis.
- [`NEXUS_V0_FINAL_AUDIT.md`](NEXUS_V0_FINAL_AUDIT.md) & [`NEXUS_V1_FINAL_ENGINEERING_REPORT.md`](NEXUS_V1_FINAL_ENGINEERING_REPORT.md): Auditorias técnicas de consolidação.
- [`validation/production-go-no-go/FINAL_REPORT.md`](validation/production-go-no-go/FINAL_REPORT.md): red team independente — `PRODUCTION_NO_GO`.
- [`validation/overnight-production-certification/FINAL_REPORT.md`](validation/overnight-production-certification/FINAL_REPORT.md): aborto `ABORTED_PRECONDITION_FAILED` da certificação overnight.
- [`validation/production-finalization/FINAL_REPORT.md`](validation/production-finalization/FINAL_REPORT.md): evidência da campanha de fechamento do release candidate.
- [`DEV/validation/`](validation/): Logs brutos, capturas de tela e artefatos de testes automatizados e manuais.
Decision record:
- [ADR-20261007-WAILS-LOCALHOST-ORIGIN.md](DECISIONS/ADR-20261007-WAILS-LOCALHOST-ORIGIN.md): allowlist exata do host virtual Wails no Desktop Windows.
- [Release Notes beta.7 Windows](NEXUS_V1_RELEASE_NOTES.md): destaques e status local de publicação. [`DEV/DECISIONS/NEXUS_TERMINAL_CONTINUITY.md`](DECISIONS/NEXUS_TERMINAL_CONTINUITY.md)
- [Nexus 1.0 delivery meta](NEXUS_1_DELIVERY_META.md)
