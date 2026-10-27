## Continuidade atual — beta.11 sincronizado e instalado — 2026-10-27

- Branch `release/v1.0.0` em `b416bb90` (= `origin/release/v1.0.0`); commits locais `5e3eff4e` (feature auth/usage/HUD), `11f3d771` (teclado Codex/PTY) e `d61b0dcc` (gates/upstream), seguidos do commit de docs.
- PASS — `GOTOOLCHAIN=go1.25.14 make quality` (exit 0): gofmt/Prettier, ESLint, Stylelint, typecheck, golangci-lint (0 issues), Go tests, Vitest 114 suítes/710 testes e version-contract.
- PASS — falhas upstream reproduzidas em worktree limpo (`/tmp/nexus-origin-verify`, `b416bb90`) e corrigidas; testes focados de `internal/nexus` e do driver passam.
- PASS — build e instalação local macOS amd64: CLI `1.0.0-beta.11`/`b416bb90`, Desktop/HUD com o mesmo build; backup em `Applications-backup-20261027-0937`.
- PASS — runtime instalado: Desktop PID 54630 + HUD vinculado PID 54635, HTTP 200 em `127.0.0.1:56992`, UI exibe `Nexus v1.0.0-beta.11`.
- BLOCKED — Developer ID/notarização, certificação arm64 e `make docs-verify` com árvore não commitada.
- SHA-256: CLI `790df820ff553119617e251c0cf7b8355b6ff150764c577592edd594081a1f3f`, Desktop `678d21ab99ce14f56828e079e8e75f26fa4a86c1173517f9f60ce847601563ec`, HUD `1109875fcbcf44ac28fead249dd5cd10b3e6d07ed996fcc37ad35d6a83209f02`.

## Continuidade atual — versão 1.0.0-beta.11 e reconciliação upstream — 2026-10-26

- Branch `release/v1.0.0`; HEAD origin `9d8d30afcefb7ec667bc4a0d3c1d1da15faf41b0`. Reconciliação dos 15 commits remotos concluída e validada.
- Versão: `1.0.0-beta.11` sincronizada em `VERSION` e `web/package.json`. Contrato verificado por `scripts/verify-version-contract.go`.
- Validações executadas com sucesso:
  - `scripts/verify-nex-visual.mjs`: PASS
  - `make version-check`: PASS
  - `make format-check`: PASS
  - `npm --prefix web run check:styles`: PASS
  - `npm --prefix web run lint:styles`: PASS
  - `npm --prefix web run lint`: PASS
  - `npm --prefix web run typecheck`: PASS
  - `npm --prefix web run test`: 110 arquivos, 684 testes PASS
  - `npm --prefix web run build`: PASS
  - `make vet`: PASS (0 warnings)
  - `go test -tags="gtk3" ./internal/...`: PASS
  - `make security`: PASS (`govulncheck` sem vulnerabilidades)
- Binários compilados: CLI `nexus`, Desktop `cmd/nexus-desktop/build/bin/nexus-desktop`, HUD `cmd/nexus-hud/build/bin/nexus-hud`.

## Continuidade atual — mascote NEX único e homologação Linux — 2026-10-26

- Branch `release/v1.0.0`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. O overlay atualmente compilado/instalado tem fingerprint `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c` (1.430 fontes; manifesto `DEV/validation/linux-100-2026-10-25/p5-current-recheck-2026-10-26-final/logs/source-provenance-preview-copy-fix-2026-10-26.txt`). Worktree compartilhada modificada, sem SHA Git congelado; não fazer commit/push automático.
- Contrato do produto: um único mascote NEX (`NexusHud → GuardianMascot → NexPixelSprite`). Barra, Mascote e Bandeja são apresentações do mesmo HUD. Companion 2D/PR #55 não faz parte do escopo; `companionModel.ts`/teste estão removidos. Preferências antigas relacionadas a `companionMotion` são somente compatibilidade/migração.
- Corrigi a prévia Composer/Flow que repetia o motivo de ausência de contas e mostrava o orçamento local em inglês. PASS: 20/20 testes focados, E2Es Composer completo e vago, e `quality:full` com 110 arquivos/681 testes/build. Sem contas elegíveis, preview segue bloqueado e não cria Mission/Agent. Chromium fixture não certifica a janela Wails nem provider externo.
- CLI/Desktop/HUD foram recompilados e instalados após backup. Hashes build/instalação coincidem: CLI `76c347d5a12697215c150aa87f6e2e52cf11e199af862c4fa16ca4e3ddc87fbe`, Desktop `4c68324423fce7e591c8e532092964375cccde2b0ccb725ee5066a65f830c6e2`, HUD `9a143bef1d9be86542dfb92da7248cadb9a3db88c18f508f38c30f78ef83e55a`. Unidade ativa; Core HTTP 200; versão reportada `1.0.0-beta.10`. Logs em `DEV/validation/linux-100-2026-10-25/p5-current-recheck-2026-10-26-final/logs/`.
- Estado atual: **NO_GO_LINUX — 169 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN (191)**. Permanecem pendentes release estável/assinatura e SHA imutável, handoff/continuidade com provedores além do smoke direto Codex, preview P5 na janela Wails, alpha/foco/múltiplos monitores e GPU/compositor Wayland.

## Verificações desta retomada — 2026-10-26

- PASS — `bun run quality:full`: 110 arquivos/681 testes, typecheck, lint, Stylelint, formatação, verificação de estilos e build de produção. Log: `DEV/validation/linux-100-2026-10-25/p5-current-recheck-2026-10-26-final/logs/web-quality-full-2026-10-26.txt`.
- PASS — regressão focada da cópia de preview: 20/20; E2E Composer completo e vago passaram em Chromium com fixtures, sem Mission/Agent quando não há provider. Relatório e capturas: `DEV/validation/linux-100-2026-10-25/P5_CURRENT_BROWSER_RECHECK.md`.
- PASS — build/instalação CLI, Desktop e HUD Linux AMD64; ELF x86-64, `ldd` sem ausências, hashes coincidentes; unidade ativa, Desktop/HUD em execução e Core HTTP 200. Evidência: `p5-current-recheck-2026-10-26-final/logs/install-latest-overlay-health-2026-10-26.txt`.
- PASS — GoReleaser v2.18.0 gerou tar.gz/DEB no fingerprint atual; checksums, metadata AMD64 e ciclo instalar/executar/remover em raiz dpkg isolada passaram. Snapshot não publicado/assinado. Logs em `p5-current-recheck-2026-10-26-final/logs/goreleaser-current-overlay-package-validation-clean-2026-10-26.txt`.
- PASS — mascote NEX único continua no caminho ativo; Companion 2D não foi integrado. Seis suítes HUD/mascote (35/35), E2E HUD e checagem da arte oficial passaram; screenshot Wails/XWayland mostra um NEX.
- Reteste do checkout atual: cinco suítes/33 testes, `bun run art:nex:check` e `bun run test:e2e:hud` PASS; evidência com os hashes das fontes em `DEV/validation/linux-100-2026-10-25/logs/mascot-current-source-recheck-2026-10-26.txt`.
- PASS — importação de sessão Gemini no perfil real do Nexus; repetição idempotente, perfil único, arquivo igual à origem e permissões `0600`. Claude não tinha credenciais tokenizadas (`NO_SESSION`). Recheck e estado sanitizados: `logs/host-session-import-real-profile-idempotency-2026-10-26.txt` e `logs/host-session-import-live-profile-state-2026-10-26.txt`. A sessão Gemini ainda requer refresh; nenhum modelo foi chamado.
- BLOCKED — fluxo Composer nativo Wails/provider autenticado real, alpha/GPU e interação completa no compositor Wayland, release estável/assinada e proveniência Git imutável.
- Resultado geral segue `NO_GO_LINUX`: 169 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN (191).
- Recheck mascote: cinco suítes Vitest, 33/33; `bun run art:nex:check`; `bun run test:e2e:hud` — todos PASS. Um NEX no caminho ativo; sem renderer/modo Companion 2D. O smoke direto real do Codex também passou em read-only/efêmero; P5 Wails/handoff ainda bloqueados.
- Wails/Xvfb no binário corrente: PASS, uma janela de 224×158, um NEX, clique abriu controles; screenshot nova preservada. A saída RGB não valida transparência nem compositor Wayland.

## Revalidação do mascote NEX único — 2026-10-26

- PASS: seis suítes Vitest HUD, 35/35 testes, `bun run test:e2e:hud` e `bun run art:nex:check` (`DEV/validation/linux-100-2026-10-25/logs/mascot-one-character-recheck-2026-10-26.txt`).
- PASS visual parcial: screenshot Wails/XWayland 224×158 mostra uma arte NEX e clique abre controles. Xvfb não valida alpha ou interação completa no compositor Wayland.
- Resultado do escopo: uma única personagem; não existe modo/renderer Companion 2D ativo. As referências `companionMotion` e chave antiga de adiamento apenas preservam preferências/dados. Sem alteração de produto nesta rechecagem. Homologação geral permanece `NO_GO_LINUX` por bloqueios independentes.

## Continuidade — DIRECT atômico até WorkPlan/MissionRunner e mascote NEX único — 2026-10-26

- HEAD/base local e remoto `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay fonte `ab5629f56e014719c0ce89c88b16b986f5bb0da8c310baac9fa079370140da5c` (1.428 arquivos; manifesto `DEV/validation/linux-100-2026-10-25/logs/source-provenance-p5-direct-atomic-mission-2026-10-26.txt`). Worktree compartilhada permanece modificada; sem commit/push.
- O produto mantém um único NEX. `companionModel`/teste e a preferência antiga de movimento como gravação canônica estão removidos; só existe o caminho `NexusHud → GuardianMascot → NexPixelSprite`. Barra/Mascote/Bandeja são apresentações do mesmo HUD. Não existe modo/renderer Companion 2D.
- PASS: cinco suítes Vitest focadas (30/30), `bun run art:nex:check`, e smoke nativo Wails prévio no mesmo overlay fonte de frontend, Xvfb 224×158, um NEX. Evidência: `DEV/validation/linux-100-2026-10-25/MASCOT_VALIDATION.md` e screenshot `screenshots/native-hud-workspace-fix-2026-10-26/hud-mascot.png`. Xvfb não comprova alpha/Wayland.
- PASS P5-009: pedido DIRECT explicitamente promovido → WorkPlan de uma etapa → admissão → MissionRunner `COMPLETED_VERIFIED`; uma execução de fixture, `go test ./...` em worktree temporária e duas entradas de evidência durável (pacote/global). Repetição `-race -count=3`, `go vet ./internal/nexus` e suíte completa `go test -count=1 ./internal/nexus` passaram. Isso não é handoff de provider real nem E2E Browser/Wails; P5 continua parcial.
- Matriz agora: **NO_GO_LINUX — 145 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (168)**. Motivos pendentes incluem release estável/proveniência congelada, Wayland alpha/GPU, handoff externo autenticado, updater oficial e soak/inspeção final.
- Relatório: `DEV/validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md`.

## Verificação atual — FIX-032 workspace idle — 2026-10-26

- Base local/remota release/v1.0.0: 7a04f1b4cf5950b97b4ea1e156dda31ac513823b; overlay fonte: eb038f52b8555c362a9129a3680746e0769ba4cbe733450b5c7a3b34a3ecbe05 (1.428 arquivos; manifesto DEV/validation/linux-100-2026-10-25/logs/source-provenance-workspace-idle-fix-2026-10-26.txt).
- Causa de CPU alta reproduzida: deduplicação recriava identidade da árvore em operação no-op e fazia o efeito de garantir superfícies repetir. Corrigido em web/src/workspace/model.ts.
- PASS: regressões workspace/Overview 22/22; bun run quality 109 arquivos/678 testes, typecheck, ESLint, Stylelint, allowlist, Prettier e build; make build CLI/Desktop/HUD Linux AMD64; ldd Desktop/HUD sem bibliotecas ausentes.
- Wails produção: janela Overview abriu em Xvfb 1280×800; WebKit principal 0,5% de um núcleo/12 s, RSS 456436→454408 kB. Inspector debug separado: 0 title writes, 1 timer inicial, 0 mutações DOM/6 s. Screenshot e logs no relatório.
- Limite: Xvfb/software, sem A/B controlado contra Wayland/GNOME; GPU, alpha e compositor real não homologados; NR-010 BLOCKED. NO_GO_LINUX permanece: 144 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN.
- Um único NEX continua sendo o mascote do produto; nenhum modo/personagem Companion 2D separado.
- HUD do overlay final também passou smoke Wails Xvfb: janela 224×158 com um NEX; clique abriu popover. Binário 12e83d1568dd79d082d2dd6640032c96e4e60ad2485d3364ff55e52cbd8a09a4, captura screenshots/native-hud-workspace-fix-2026-10-26/hud-mascot.png, log logs/current-hud-native-workspace-fix-xvfb-2026-10-26.txt. RGB não certifica o conteúdo completo do popover, alpha ou Wayland.

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

## Verificação atual — 2026-10-26 — mascote único e Desktop Wails

- HEAD local/remoto `release/v1.0.0`: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay `751cfc1a160c38d950ac5b9807e1febb9746d03981c861c722e5bd965e1ef012` (1.428/1.428 entradas do manifesto conferidas). Worktree modificada.
- Produto contém um NEX e nenhuma apresentação Companion 2D separada. Desktop/HUD Wails AMD64 correntes passaram smokes Xvfb; Desktop/Core HTTP 200, Uso/Contas exibiu resultado de importação com HOME isolado; HUD mostrou um NEX e abriu controles. RGB/Xvfb não confirma alpha/Wayland.
- **PASS:** `quality:full` (109 arquivos/675 testes/build), E2E HUD/Mission Attention, Bun 1.3.9 frozen install e auditoria Bun sem advisories em `web`/`docs-site`. OSV permanece BLOCKED por reset da API.
- **NO_GO_LINUX — 139 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (162)**. REL-001 continua FAIL; alpha Wayland, providers reais, idle acelerado, updater oficial e proveniência congelada não estão aprovados. Soak P9 segue ativo.

## Verificação anterior — FIX-028 e recursos — 2026-10-26

- HEAD/remoto `release/v1.0.0`: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay de fontes `391c5718b3f2cd494fb4fa0e6a9fae6f03a618c002155258a59b7e7e83f27459` (1.428 arquivos; `validation/linux-100-2026-10-25/logs/source-provenance-update-indicator-2026-10-26.txt`); worktree modificada.
- PASS: `bun run quality:full` (109 arquivos/675 testes/build); regressão do indicador (6/6); build Wails `gtk3,production` isolado; janela nativa Wails/Core sob Xvfb e health HTTP 200.
- Wails Xvfb: 107,8% CPU WebKit principal/20 s após remover animação contínua; baseline pré-fix 196,1%, delta direcional (três processos WebKit vs. um). Screenshot em `validation/linux-100-2026-10-25/screenshots/native-idle-xvfb-fixed-update-indicator-2026-10-26/usage-accounts-idle.png`. Sem aprovação de GPU/Wayland; NR-010 continua BLOCKED.
- Mascote: NEX único, sem modo/renderer Companion 2D; evidência e regressões vigentes em `validation/linux-100-2026-10-25/MASCOT_VALIDATION.md`.
- GitHub read-only: cinco previews e nenhuma release estável v1.0.0; #57/#58 Draft, #59/#61 conflitantes, #64 em main sem checks. REL-001 permanece FAIL. Evidência em `validation/linux-100-2026-10-25/PULL_REQUEST_RECONCILIATION_CURRENT.md` e `validation/linux-100-2026-10-25/logs/current-gh-release-and-pr-recheck-2026-10-26.txt`.
- Gate agregado: **NO_GO_LINUX — 137 PASS / 1 FAIL / 20 BLOCKED / 4 NOT_RUN (162)**. Soak P9 segue ativo; não commit/push.

## 2026-10-25 — recheck final do mascote único e homologação Linux (UTC 26/10)

- Branch/HEAD remoto/local: `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay de produto `9a66687715414fc8b42d612fd2b4a531370c7f8c4fbfa86338a81ba9728ddcd1` (1.428 arquivos). Manifesto conferido: 1.428/1.428 arquivos sem diferença.
- Direção confirmada: um único NEX; nenhum modo ou renderer Companion 2D. 32/32 testes focados e `bun run art:nex:check` passaram.
- `bun run quality:full` (109 arquivos/674 testes/build) e E2E de primeiro acesso/importação passaram no overlay atual. Wails Desktop/Core/HUD instalados foram exercitados sob Xvfb; clique em Uso/Contas mostrou a resposta esperada e a captura mostra somente um NEX. HOME vazio isolado; nenhuma sessão do perfil normal foi lida ou copiada.
- Binários instalados conferidos por SHA-256; unidade Desktop/HUD do usuário ativa e `/api/v1/health` respondeu HTTP 200 (`1.0.0-beta.10`).
- Matriz Markdown reconciliada com o JSON: 162 casos = 137 PASS, 1 FAIL, 20 BLOCKED, 4 NOT_RUN. **NO_GO_LINUX** permanece: Wayland/alpha do host, provider real, updater assinado/rollback e commit congelado seguem pendentes. A CPU alta do WebKit foi reproduzida em repouso aparente na sessão normal e sob Xvfb sem credenciais; profiling por `strace`/`perf` não é permitido pelo kernel, portanto a causa segue sem identificação. Soak P9 de 8 h está em andamento.
- Evidências: `validation/linux-100-2026-10-25/{LINUX_HOMOLOGATION_REPORT.md,TEST_MATRIX.md,LINUX_HOMOLOGATION_SUMMARY.json,MASCOT_VALIDATION.md,PROVIDER_SESSION_IMPORT_VALIDATION.md}`.

## 2026-10-25 — recheck do mascote único e gates P1 Linux

- Base local/remota `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; fingerprint corrente `f033fa678c1fdeb99373d2456347ac152638ef13f699e607ff5b9bdd12113f2b` (1.426 fontes; manifesto `validation/linux-100-2026-10-25/logs/source-provenance-current-single-mascot-2026-10-25.txt`). Worktree suja, sem SHA imutável.
- Um único NEX foi confirmado no HUD Wails instalado em XWayland, 224×158; screenshot `screenshots/current-single-mascot-installed-hud-xwayland.png` SHA-256 `df2698bd013a9c32bfe81d69d46e3c812bbb89cf82de0108371a9a39035762ed`. PR #55/Companion 2D está fechada e fora do objetivo. Novas preferências gravam `mascotMotion`; leitura legada continua.
- **PASS:** regressões do mascote 30/30, arte oficial, E2E HUD Chromium, `quality:full` (108 arquivos/674 testes/build), build/instalação CLI/Desktop/HUD AMD64 e health HTTP 200. Recheck sob `-race` também passou nos casos de replay/replan e HTTP de intervenções (`validation/linux-100-2026-10-25/logs/p1-code-review-findings-recheck-2026-10-25.txt`).
- Soak P9 de 8 h permanece em execução. Resultado **NO_GO_LINUX**: 132 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN (154). Alpha Wayland, provider/atenção real, assinatura/update oficial e proveniência imutável seguem abertos. Veja `validation/linux-100-2026-10-25/TEST_MATRIX.md`.

## 2026-10-25 — P2 nativo e mascote NEX único

- Overlay de produto: `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa`, sobre branch/remoto `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.
- PASS nativo XWayland: uma sessão Wails exibiu A/D/E, manteve os estados enquanto B/C estavam ativos, refletiu A/B/C/D/E após atualização isolada do SQLite e refez a projeção depois do restart do Core sem F5. Nenhum provedor/runner foi chamado nesse teste; ACL multiusuário não é suportada nem validada.
- Logs e checksums: `validation/linux-100-2026-10-25/logs/p2-native-live-attention-current-overlay-2026-10-25.txt` e `p2-native-live-attention-screenshots-2026-10-25.sha256`; screenshots em `screenshots/p2-native-live-attention-2026-10-25/`.
- Mascote: um único NEX, sem Companion 2D separado; testes/art check/E2E e captura nativa estão em `MASCOT_VALIDATION.md`. `node scripts/verify-nex-visual.mjs` passou estruturalmente; `make docs-verify` terminou exit 2 porque o manifesto visual não certifica a árvore HUD/NEX não commitada (`validation/linux-100-2026-10-25/logs/docs-verify-current-overlay-2026-10-25.log`).
- O veredito geral continua `NO_GO_LINUX` (126 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN).

## 2026-10-25 — P5 falha pré-dispatch e continuidade após restart

- Overlay exato: `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa`; manifesto `validation/linux-100-2026-10-25/logs/source-provenance-p5-predispatch-continuity-2026-10-25.txt`; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.
- Mascote único: 29/29 testes HUD, verificação SVG e E2E HUD Chromium passaram; o Wails do overlay exato abriu em XWayland (224×158) e três capturas (5/15/30 s) mostram um NEX. Evidências: `validation/linux-100-2026-10-25/MASCOT_VALIDATION.md` e `HUD_VALIDATION.md`. Alpha Wayland e evento de agente real permanecem pendentes.
- Integridade final: 1.425/1.425 registros de fonte do manifesto coincidem com fingerprint `2c23ebb0…`; matriz Markdown e JSON reconciliam 148 IDs (126 PASS, 1 FAIL, 16 BLOCKED, 5 NOT_RUN); JSON parse e `git diff --check` passaram.
- **PASS:** setup local sem perfil retorna como não enviado e não chama launcher; falha do launcher permanece ambígua após uma chamada.
- **PASS:** Composer OAuth multiárea injeta falha antes de dispatch, reabre SQLite e recria o MissionRunner; snapshot da revisão aprovada permanece igual, ≥4 pacotes completam e dispatches bem-sucedidos não se repetem.
- **PASS:** testes focados, race x3, `go test -count=1 -timeout=4m ./internal/nexus/...`, `go vet ./internal/nexus/...`.
- Fixture determinística, sem credenciais/requests reais. P5 permanece parcial por handoff real, cenários oficiais restantes e fluxo Wails. O updater tem PASS com fixtures HTTP locais/chaves sintéticas; release estável assinada permanece bloqueada. **NO_GO_LINUX** (126 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN, 148 casos).

## 2026-10-25 — Recheck do mascote NEX único no overlay corrente

- Overlay `0c522c885e700dc52d6d03a0feb313bc7d5e420915ca0188fe6c554b49fe9841`, 1.424 fontes; testes focados HUD: 5 suítes/29 testes **PASS**.
- `bun run --cwd web art:nex:check`: **PASS**. `NEXUS_E2E_ARTIFACTS_DIR=../DEV/validation/linux-100-2026-10-25/screenshots/mascot-single-renderer-current-overlay-0c522c88 bun run test:e2e:hud`: **PASS** com fixture Chromium; capturas e hashes registrados. Não é evidência Wails nativa.
- `companionModel` separado não existe; o renderer ativo apresenta um NEX. Alpha e atenção gerada por agente real seguem BLOCKED.

## 2026-10-25 — P5 feature OAuth multiárea até MissionRunner

- Overlay: `0c522c885e700dc52d6d03a0feb313bc7d5e420915ca0188fe6c554b49fe9841`; manifesto `validation/linux-100-2026-10-25/logs/source-provenance-p5-large-feature-mission-fixture-2026-10-25.txt`; base local/remota `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.
- **PASS:** prompt OAuth multiárea em fixture, edição semântica rejeita token antigo sem run e preview atualizado conclui ≥4 pacotes no MissionRunner; suíte `go test -count=1 -timeout=4m ./internal/nexus/...`, `go vet ./internal/nexus/...` e foco com `-race -count=3` passaram.
- Um primeiro race encontrou contador inseguro somente no executor fake do teste; mutex aplicado ao fixture, log original preservado, retry e três repetições race passaram. Não foi alteração runtime.
- E2E Chromium Flow/revisão 2 vem do overlay anterior `ee6a9dd…`; não é homologação Wails nem execução real de provider. P5 parcial e resultado global **NO_GO_LINUX**: 124 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN.

## 2026-10-25 — P5 edição semântica e aprovação stale-safe

- Overlay histórico `ee6a9dd866ae4a938d66d958fe08d3711088f41233358f2c13feee1df015d590`, 1.424 fontes; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.
- **PASS** — Chromium E2E `bun run test:e2e:composer-spine`: Flow editou Goal, persistiu revisão 2, recalculou preview/token, não criou run/Agent sem provider. Capturas em `validation/linux-100-2026-10-25/screenshots/p5-semantic-revision-e2e-2026-10-25/`.
- O E2E Chromium não é teste Wails nativo; provider real e UI nativa permanecem parciais.

## 2026-10-25 — P1 Skills com CLI fixture

- **PASS:** `go test ./internal/nexus -run '^TestSelectedSkillsReachAnIsolatedProviderSession$' -count=5`; 2 subcasos por repetição.
- **PASS:** mesmo teste com `-race -count=3`; cleanup do Agent/runtime confirmado como `STOPPED` e nenhuma credencial no perfil isolado.
- **PASS:** `go test -count=1 -timeout=20m ./internal/nexus` e `go vet ./internal/nexus`.
- CLI falso por `PATH`, sem chamadas externas; comandos e limitação externa em `validation/linux-100-2026-10-25/SKILL_SESSION_VALIDATION.md`. `BLK-008` continua BLOCKED até sessão com provedor real.
- Fingerprint testado: `d773e38c495781f52cdb3fa208c6a80bd1aae88d5e14bfabee56ac486407e0e1`; NO_GO_LINUX continua.

## 2026-10-25 — Reconfirmação do mascote NEX único

- **PASS** — 29 testes em cinco suítes do HUD; o seletor não contém opção Companion 2D e o renderer produz exatamente uma arte NEX. Log: `validation/linux-100-2026-10-25/logs/mascot-user-direction-recheck-2026-10-25.log`.
- **PASS** — `bun run art:nex:check` e `bun run test:e2e:hud` (Chromium/fixtures; não substitui validação nativa). Logs correspondentes em `validation/linux-100-2026-10-25/logs/mascot-art-check-recheck-2026-10-25.log` e `mascot-hud-e2e-user-direction-2026-10-25.log`.
- **PASS visual nativo previamente coletado** — capturas Wails mostram uma única arte, balão ligado ao mesmo NEX e preferências do HUD. Alpha/transparência e sinal de agente real seguem pendentes.
- Não havia modo ou renderer 2D no checkout; nenhuma remoção de produto foi necessária. Resultado amplo da homologação: `NO_GO_LINUX`, 120 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN.

## 2026-10-25 — P2 A/B/C/D/E com recuperação durável

- Identidade: branch `release/v1.0.0`, HEAD/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`, overlay de fontes `30223d700c7d7395705571ad72d2ea30055d191e181b03cb114a922395aef18b` (1.422 arquivos), árvore dirty.
- **PASS** — `go test -count=10 ./internal/control/web -run '^TestAttentionHTTP(MultiMissionQuotaRecoverySurvivesRestart|ProjectionSurvivesRestartWithProjectIsolation)$'`.
- **PASS** — `go test -race -count=5 ./internal/control/web -run '^TestAttentionHTTPMultiMissionQuotaRecoverySurvivesRestart$'`.
- **PASS** — `go test -count=1 ./internal/control/web` e `go vet ./internal/control/web`.
- O E2E HTTP/SQLite fixture mostra A/D/E antes do restart, omite B durante EXECUTING e C enquanto REMEDIATING por quota, e recupera B/C uma vez como COMPLETED_VERIFIED. B contou 1 alocação/execução; C contou 2. Isso não valida a UI Wails conectada com as cinco missões, nem ACL multiusuário.
- Logs: `validation/linux-100-2026-10-25/logs/p2-multi-mission-http-restart-e2e-2026-10-25.txt`, `p2-multi-mission-http-restart-race-2026-10-25.txt`, `go-test-web-p2-multi-mission-2026-10-25.txt`, `go-vet-web-p2-multi-mission-2026-10-25.txt`; manifesto: `validation/linux-100-2026-10-25/logs/source-provenance-p2-multimission-2026-10-25.txt`.
- Resultado atualizado: **NO_GO_LINUX — 117 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN (135)**.
- No overlay atual, `go test -count=1 -timeout=20m ./...`, `go vet ./...` e `go test -race -count=1 -timeout=30m ./...` passaram; 83 linhas de resultado de pacote, sem falhas/race. Logs em `validation/linux-100-2026-10-25/logs/current-go-*.txt`.
- `bun run quality:full` passou em 108 arquivos/673 testes e gerou build de produção. E2E HUD passou com capturas versionadas de desktop, atenção, expandido e mobile; checksum e logs em `validation/linux-100-2026-10-25/`.
- O reteste nativo P2-UI-MISSION-001 passou com screenshot fixture de cinco missões antes/depois do restart na mesma janela Wails. O caso não valida transições live de provedores na UI nem ACL multiusuário; detalhes e hashes em `validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md`.

## 2026-10-25 — follow-up do mascote NEX único

- **PASS** — `bun run art:nex:check` validou o SVG oficial da arte NEX; log `validation/linux-100-2026-10-25/logs/current-nex-art-check-2026-10-25.txt`.

- Base Git local/remota `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay local modificado, sem commit ou fingerprint integral novo.
- **PASS** — `bun run --cwd web quality`: 108 arquivos e 673 testes; inclui format, lint, Stylelint, verificação de styles e typecheck. Log: `validation/linux-100-2026-10-25/logs/mascot-single-model-refactor-quality.log`.
- **PASS** — build Web e `bun run --cwd web test:e2e:hud`; o E2E exercitou atenção, abrir/adiar e recarga. Logs e capturas: `validation/linux-100-2026-10-25/logs/mascot-single-model-refactor-e2e.log` e `screenshots/mascot-single-model-refactor/`.
- **PASS** — CLI, Desktop Wails e HUD Wails Linux AMD64 recompilados/instalados; Desktop iniciou via XWayland, health HTTP retornou `ok`, `ldd` não encontrou dependências ausentes. Hashes: `validation/linux-100-2026-10-25/logs/local-install-mascot-refactor.sha256`.
- **PASS** — 34 testes focados, incluindo preferência `companionMotion` e chave antiga de adiamento, um único NEX e ausência de opção 2D.
- O frontend agora usa `mascotModel`/`hud.mascot`, preservando dados legados; `NO_GO_LINUX` continua pelos gates P2/P7 live/P8/P9.

## 2026-10-25 — Campaign P7 handoff, PTY, and full race

- HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay `bb575423508a0c773580804a476b0888ff2449135045a74f80aa6efaffedc6e1` (1,421 source files, manifest `logs/source-provenance-p6-agent-fallback-2026-10-25.txt`). Worktree is modified and not frozen.
- **PASS** — `go test -race -count=1 ./...`, exit 0; 82 package result lines. Slow packages: `internal/nexus` 261.558 s, store 79.035 s, capacitymonitor 25.074 s. Log `logs/go-race-p7-full-repository.log`.
- **PASS** — focused process-level PTY attach/detach/writer lease, launcher/registry stale ownership and replacement, handoff predicate, and runner recovery race tests; see `P7_HANDOFF_PTY_VALIDATION.md`.
- **PASS** — live AGY safe response from one authenticated profile; expected HTTP 429 from an authenticated exhausted profile. Raw provider output and identity were suppressed.
- **BLOCKED** — live same-provider handoff and cross-provider checkpoint continuation. Existing Codex runtimes were preserved; OpenCode did not provide a final response within 90 s. `NATIVE_RESUME_UNVERIFIED` remains.
- One NEX mascot remains the product contract; no Companion 2D mode. Counts: 97 PASS / 1 FAIL / 15 BLOCKED / 5 NOT_RUN (118), `NO_GO_LINUX`. P2/P7 live/P8/P9 remain open.

## 2026-10-25 — Campaign P6 Agent fallback

- Commit local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; product overlay `bb575423508a0c773580804a476b0888ff2449135045a74f80aa6efaffedc6e1` (1,421 source files; `logs/source-provenance-p6-agent-fallback-2026-10-25.txt`). Worktree is modified and not frozen.
- **PASS** — `go test ./internal/nexus/... ./internal/control/... -count=1`; log `logs/go-test-p6-agent-fallback.log`.
- **PASS** — focused race regressions repeated 5x; log `logs/go-race-p6-agent-fallback.log`.
- **PASS** — `go vet ./internal/nexus/... ./internal/control/...`; log `logs/go-vet-p6-agent-fallback.log`.
- **PASS** — deterministic hard gates/tie-break, provider/model fallback provenance, ephemeral specialist identity, durable assigned kind, and safe cleanup; see `AGENT_FALLBACK_VALIDATION.md`. No live provider session was used.
- One NEX mascot remains the product contract; no Companion 2D mode exists. Overall counts: 91 PASS / 1 FAIL / 13 BLOCKED / 5 NOT_RUN (110), `NO_GO_LINUX`. P2 and P7–P9 remain open.

## Follow-up P5 — conflito de revisão WorkPlan (2026-10-25)

- Compare-and-swap por `current_revision` impede overwrite concorrente; stale write retorna conflito tipado e HTTP 409.
- Regressões race passaram (store x3, serviços Nexus x2, handler x3), suíte `store/nexus/control/web` e vet passaram. Fingerprint `d3caa597ae3bc061ad0048ad4d390b1b9bb0011cc496e4d213d0ec6119400796`; manifesto e logs em `DEV/validation/linux-100-2026-10-25/`.
- P5 continua parcial; cinco cenários oficiais, runtime real e gates de distribuição continuam pendentes. Resultado geral NO_GO_LINUX.

## Snapshot histórico — Campaign P4 Project Intelligence / P5 Composer spine

- HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay `bcfbfa69c53255cf2768c1024b7932d5241e9316eff3e3d9d7ca3cf878755141` (1.421 sources; `logs/source-provenance-p4-project-intelligence-2026-10-25.txt`). Worktree is not frozen.
- **PASS** — `go test ./internal/nexus/contextsnapshot ./internal/nexus -count=1`; log `logs/go-test-p4-project-intelligence.log`.
- **PASS** — focused race tests repeated 3x and `go vet ./internal/nexus/contextsnapshot ./internal/nexus`; logs `logs/go-race-p4-project-intelligence.log` and `logs/go-vet-p4-project-intelligence.log`.
- **PASS** — `bun run --cwd web quality:full`: 108 Vitest files / 672 tests; format, lint, styles, typecheck and production build. Log `logs/web-quality-full-p4-project-intelligence.log`.
- **PASS** — isolated P4 Chromium E2E grounds the Composer with `vitest run`, source `package.json:scripts.test`, HIGH confidence and freshness; vague goal asks one material question, complete prompt has zero blocking questions. Log and artifacts in `composer-project-intelligence/`.
- **PASS** — P5 Composer→WorkPlan E2E: two clarifications, finalize HTTP 201, persisted FINALIZED, five steps, preview starts no Mission. Log `logs/composer-vague-spine-p5-repro.log`; direct store regression `logs/go-test-p5-finalization-regression.log`.
- **BLOCKED** — real authenticated `ai-cli` question: executable absent from PATH, no external account used; `logs/ai-cli-availability-p4.txt`.
- Mascot remains one NEX, with no separate Companion 2D mode. Overall counts: 91 PASS / 1 FAIL / 13 BLOCKED / 5 NOT_RUN (110), `NO_GO_LINUX`. Detailed matrix: `TEST_MATRIX.md`.

## 2026-10-25 — Campaign P3: entrega de notificações de Mission

- HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay P3 `65b209131c3ffe8101c86090ad229b3299742d4a102229db927b4e3a463c9b3d` (1.420 fontes; manifesto `DEV/validation/linux-100-2026-10-25/logs/source-provenance-p3-notifications-2026-10-25.txt`). Worktree não congelada.
- **PASS** — `go test ./internal/nexus/... ./internal/control/notify -count=1`; todos os módulos Nexus, store, runner e notifier passaram. Log `logs/go-test-p3-notifications.log`.
- **PASS** — testes focados do throttle/outbox/capability/evento de aplicação repetidos cinco vezes com `-race`; log `logs/go-race-p3-notifications.log`.
- **PASS** — `go vet ./internal/control/notify ./internal/nexus/...`, exit 0 sem diagnósticos; log `logs/go-vet-p3-notifications.log`.
- **PASS** — smoke nativo opt-in no Ubuntu GNOME 49/Wayland: `notify-send` aceitou copy genérica via D-Bus; log `logs/native-notification-smoke-p3-2026-10-25.log`. Não capturou pixels do toast.
- **PASS** — store close/reopen recuperou retry pendente; o serviço deduplicou blocker e não repetiu delivery concluída. Fixture de capability ausente adiou sem incrementar tentativas e retomou quando habilitada.
- **PASS** — quota/rate-limit/retry/remediation não geram toast; completion permanece no Attention Center; falha só notifica com escalonamento explícito. Copy e erros persistidos foram inspecionados sem vazamento de prompts, links ou credenciais.
- **Mascote:** NEX permanece único; Barra/Mascote/Bandeja são apresentações do HUD. Não há modo Companion 2D para remover.
- Resultado geral segue `NO_GO_LINUX`: 77 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN (95); P2 e P4–P9 continuam em aberto.
- Relatório P3: `DEV/validation/linux-100-2026-10-25/MISSION_NOTIFICATIONS_VALIDATION.md`.

## 2026-10-25 — Reteste no HEAD 7a04f1b4: P2 e mascote NEX único

- HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay atual `ee8a19c0cf340086f8f1cfc258d50a98ec29719952b978401be55c977a693b6c` (1.415 fontes; `DEV/validation/linux-100-2026-10-25/logs/source-provenance-p2-ui-e2e-2026-10-25.txt`). Worktree permanece sem commit imutável.
- **PASS** — `TestAttentionHTTPProjectionSurvivesRestartWithProjectIsolation`: 10 execuções normais e 5 com `-race`. Autenticação sem bearer = 401; com sessão = 200; projeções needs-user/completed/failed e filtros por projeto/mission sobreviveram a close/reopen do SQLite, serviço e endpoint HTTP de teste.
- **PASS** — `bun run test:e2e:hud`: fixture de atenção, snooze persistido e movimento reduzido passaram no overlay atual. O E2E anterior também cobriu Axe desktop/mobile em fixture.
- **PASS** — cinco suítes focadas do NEX: 27/27; `bun run art:nex:check` passou. Não há renderer/atlas/apresentação 2D separado; a arte pública SVG é exportação do mesmo componente.
- **PASS** — Desktop Wails atual iniciou em XWayland e renderizou a tela inicial em 1280×800; SIGTERM exit 0 sem `GDK-CRITICAL`. Captura `DEV/validation/linux-100-2026-10-25/screenshots/native-desktop-current-overlay-xwayland-ready.png`.
- **PASS** — HUD Wails atual foi capturado em XWayland, janela visível de 224×158, com uma única figura NEX; screenshot atual em `DEV/validation/linux-100-2026-10-25/screenshots/native-hud-current-overlay-xwayland-visible.png`. Alpha/transparência continua sem validação.
- **NO_GO_LINUX** — E2E Chromium atual passou estados vazios/loading/erro, filtros, teclado, navegação e mobile. P2 segue aberto para restart do processo real com UI conectada, cenário A/B/C/D/E integrado e ACL por usuário/projeto. Matriz: 69 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN (87). O workspace local é single-user.
- Logs: `DEV/validation/linux-100-2026-10-25/logs/attention-http-restart-p2-final-2026-10-25.log`, `attention-http-restart-p2-race-2026-10-25.log`, `mission-attention-browser-e2e-p2-2026-10-25.log`, `mascot-hud-current-overlay-final-2026-10-25.log` e `native-hud-current-overlay-xwayland-2026-10-25.log`.

## 2026-10-25 — Campaign P1: intervenção e retomada — COMPLETE

- **PASS** — `go test ./internal/nexus/runner -count=1 -v`; `go test ./internal/nexus -count=1`; `go test ./internal/nexus/...`; `go test ./internal/control/web -count=1`.
- **PASS** — `go test -race ./internal/nexus/runner -count=1`; `go vet ./internal/control/web ./internal/nexus/...`.
- **PASS** — regressões de replay inseguro, re-alocação/recompilação, espera por dependências, persistência `ReplayProhibited` no SQLite, timeout/handoff do worker e JSON/versionamento HTTP.
- **PASS** — `TestResolveInterventionHTTPDecisionSurvivesServerRestartExactlyOnce`, 10 execuções; teste com race do mesmo caso, 5 execuções. Inclui middleware real, resolução persistida, duplicata idempotente, shutdown/reopen de store/servidor, recovery e um único efeito externo. Logs `p1-http-restart-e2e-2026-10-25.log` e `p1-http-restart-e2e-race-2026-10-25.log`.
- **PASS** — suítes Web e Nexus do escopo, vet, lint Go com cache isolado, builds CLI/Desktop/HUD e instalação local/restart; evidências linkadas no relatório P1.
- **NO_GO_LINUX** — P1 fechado para o overlay, mas P2–P9 e outros gates de release seguem abertos. HEAD local/remoto `bf6adf0c80ac3531d7291844512a14a017ab0002`; worktree dirty.
- Fingerprint do overlay validado: `ddad56697609de1374cdede949e794316e3263357b052f745defb64f11a4be5b` (1.412 fontes; manifesto em `DEV/validation/linux-100-2026-10-25/logs/p1-e2e-source-provenance-current.txt`).
- Mascote: um caminho único `NexusHud → GuardianMascot → NexPixelSprite`; nenhum modo ou personagem Companion 2D separado. A apresentação Mascote continua sendo o NEX único.

## 2026-10-25 — Follow-up P2: Mission Attention e decisão nativa

- **PASS** — testes focados de rota e gate: 21/21; detalhe de uma missão continua disponível na build estável, com criação/histórico experimentais bloqueados.
- **PASS** — Wails AMD64 iniciou no GNOME Wayland/XWayland; cartão de fixture abriu e ativou o Flow Run com a decisão visível. Captura 1280×800: `DEV/validation/linux-100-2026-10-25/screenshots/mission-attention-open-detail-native-xwayland.png`.
- **PASS** — resumo de estado corrigido com SCSS Module e teste de regressão. `bun run quality:full`: 108 arquivos/672 testes; checks de estilo, formatação, lint, typecheck e build passaram.
- **PASS** — `make build-desktop-wails`; ELF AMD64, `ldd` sem bibliotecas ausentes. SHA-256 `f6e1afcf2e7b0e10ed5d75a5d716b22dfe6f6ca894e056c64938599172ee7f15`.
- **PASS** — `TestGlobalAttentionProjectionSurvivesStoreReopen`: cinco missões/2 projetos mantiveram os estados canônicos após reabrir SQLite; filtros projeto/missão não vazaram.
- **Mascote** — um único NEX (`GuardianMascot → NexPixelSprite`); nenhum Companion 2D ou modo separado. Captura: `DEV/validation/linux-100-2026-10-25/screenshots/native-nexus-hud-single-mascot-xwayland.png`.
- **NO_GO_LINUX** — base HEAD local/remota `bf6adf0c80ac3531d7291844512a14a017ab0002`, overlay dirty `7368531977b7422eccdf00e7c676f57d440e8d736d9e114bcdc071b9d5885fe6`; a fixture não valida atenção de agente real e gates globais anteriores continuam abertos. Detalhes: `DEV/validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md`.

## 2026-10-25 — FIX-012, evidência Linux atual e escopo do mascote

- **PASS** — reexecutei os cinco testes focados do mascote: 25/25. Barra/Mascote/Bandeja são apresentações do mesmo HUD; nenhum modo Companion 2D separado está conectado. A captura nativa mostra um NEX (SHA-256 `98b4320b329512b349af90284f5851b9080b20be3d97fc4bccf19520ac16cd3c`).
- **PASS** — `node scripts/verify-nex-visual.mjs`: identidade visual estrutural. O manifesto `source-provenance-gdk-fix.txt` foi rechecado arquivo a arquivo: 1.404 fontes, zero ausentes/divergentes.
- **PASS** — regressões direcionadas do popover GTK e `go test -race -count=1 ./internal/desktop ./cmd/nexus-hud`. Três ciclos nativos abrir/fechar passaram com `G_DEBUG=fatal-criticals`.
- **PASS** — builds/pacotes Linux pós-FIX-012: CLI/Desktop/HUD AMD64, tarballs CI com payload idêntico, GoReleaser tar/DEB e instalação/remoção dpkg isolada.
- **PASS** — JSON e matriz reconciliados: 63 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN (81); branch base local/remota `bf6adf0c80ac3531d7291844512a14a017ab0002`.
- **NO_GO_LINUX** — worktree dirty sem commit imutável e gates de release/runtime ainda abertos.


## 2026-10-25 — Balão de atenção nativo do mascote

- **PASS** — HUD Wails recompilado abriu no GNOME/XWayland com perfil isolado; fixture `REQUIRE_USER` apareceu junto ao NEX único. Screenshot SHA-256 `b2bff6d18ab8214d682430c3f097e93bf1e338a1924af7ccb7b25f661c7ba08a`.
- **PASS** — XTest clicou “Adiar”; HUD recolheu o balão, enquanto `/api/v1/attention` manteve a fixture (`HTTP 200`, `needs_you_count=1`). Log: `DEV/validation/linux-100-2026-10-25/logs/native-attention-fixture-click-snooze.txt`.
- **PASS** — evidência de preferências nativas e controle Barra/Mascote/Bandeja em `DEV/validation/linux-100-2026-10-25/screenshots/native-hud-expanded-single-nex-auth-fix-xwayland.png`.
- **BLOCKED** — evento real de agente ainda não alimentou o HUD; a fixture não conta como validação de runtime.
- **PASS** — contagem reconciliada em 79 casos: 61 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN. Matriz e JSON atualizados; `NO_GO_LINUX` permanece.

## 2026-10-25 — Contrato do mascote único, reteste e integridade das evidências

- **PASS** — inspeção do fluxo ativo confirma `NexusHud → GuardianMascot → NexPixelSprite`; não existe segundo renderer nem seletor Companion 2D. Barra, Mascote e Bandeja são apresentações do mesmo HUD.
- **PASS** — reteste focado atual: 5 arquivos / 25 testes, incluindo `GuardianModeSelector`; log `DEV/validation/linux-100-2026-10-25/logs/mascot-user-direction-final-retest.log`.
- **PASS** — 79 casos contabilizados igualmente no JSON e na matriz: 61 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN. Integridade de 1.403 entradas do manifesto atual: zero arquivos ausentes ou divergentes.
- **PASS** — `git diff --check`; `origin/release/v1.0.0` segue em `bf6adf0c80ac3531d7291844512a14a017ab0002`. Overlay `4e7afe7bc0e99c8d11e34dd3db9295829ab41081173bc9a7cc9b262f3c92111b`; sem commit imutável. Veredito `NO_GO_LINUX` permanece.

## 2026-10-25 — FIX-011: anexar runtime shell existente

- **PASS** — regressão unitária de roteamento: shell chama `onAttachRuntime(runtime)`; runtime de agente preserva `runtime_id` e o fluxo de terminal existente. 2/2 testes.
- **PASS** — `bun run quality:full`: 106 arquivos/664 testes; arquitetura de estilos, Prettier, ESLint, Stylelint, typecheck e build passaram.
- **PASS** — E2E de terminal dividido: sete geometrias, rótulos longos e toolbars em 520 px/360 px.
- **PASS** — Desktop Wails AMD64 atual recompilado e iniciado em XWayland. Shell iniciado com a imagem atual foi anexado; runtime permaneceu RUNNING, total de runtimes/PTYs ficou em 4 e o prompt existente ficou visível.
- **BLOCKED** — anexar host lançado por imagem anterior após troca do binário no mesmo caminho: validação de fingerprint o marcou `STALE`; a UI expôs erro genérico `TERMINAL_TRANSPORT_ERROR`. Não foi alterado o gate de compatibilidade.
- **Mascote** — confirmado NEX único; não há modo Companion 2D separado para remover. A apresentação “Mascote” continua sendo o mesmo NEX.
- **Matriz:** NO_GO_LINUX — 60 PASS / 1 FAIL / 12 BLOCKED / 6 NOT_RUN (79). Base SHA local/remota `bf6adf0c80ac3531d7291844512a14a017ab0002`; overlay `4e7afe7bc0e99c8d11e34dd3db9295829ab41081173bc9a7cc9b262f3c92111b` (1.403 fontes), dirty e não commitado.
- Evidências: `DEV/validation/linux-100-2026-10-25/logs/runtime-attach-validation.txt`, `runtime-attach-quality-full.log`, `terminal-split-toolbar-regression-final.log` e screenshots correspondentes.

## 2026-10-25 — FIX-009, revalidação do mascote e evidência corrente

- **PASS** — mascote único: `GuardianMascot → NexPixelSprite`, seletor Barra/Mascote/Bandeja, sem segundo renderer ou modo 2D. Regressão Vitest: 25/25; NEX visível em captura nativa.
- **PASS** — Cursor `InspectAuth` reconhece `✓ Login successful!`; sinais negativos e saída vazia permanecem não autenticados.
- **PASS** — `go test -count=1 -timeout=20m ./...`; `go vet ./...`; `go test -count=1 -v ./internal/core/provider/adapters/cursor`; `go test -race -count=1 ./internal/core/provider/adapters/cursor`; `bun run quality:full` (105 arquivos/660 testes); `bun run art:nex:check`. Avisos restantes são depreciações GDK/X11 upstream do Wails.
- **PASS** — snapshot GoReleaser v2.18 atual, checksums, payload tar/DEB idênticos e ciclo de instalação/consulta/execução/remoção do DEB em dpkg root temporário.
- **BLOCKED** — importação/cópia de sessão Cursor ainda não suportada; runtime orquestrado/continuidade/Skills/atenção real, interação física Wayland, alpha e updater assinado continuam pendentes.
- **Matriz:** NO_GO_LINUX — 58 PASS / 1 FAIL / 12 BLOCKED / 6 NOT_RUN (77); overlay `dac0d00b8410f70ed02c8b4c324780349655fed8f7e580d1944da1ec3eaecefa`, base `bf6adf0c80ac3531d7291844512a14a017ab0002`, worktree dirty.
- Evidências: `DEV/validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md`, `TEST_MATRIX.md`, `MASCOT_VALIDATION.md`, `logs/go-test-cursor-status-final.log`, `logs/go-vet-cursor-status-final.log`.

## 2026-10-25 — Atualização da homologação nativa e mascote NEX único

- **PASS** — busca local Docusaurus: `npm run build --prefix docs-site` gerou `search-index.json` (2.442.219 bytes); Playwright consultou `terminal`, recebeu índice HTTP 200 e encontrou `/product/terminals`, sem erros de página.
- **PASS** — `npm run check:docs --prefix docs-site`: 100 páginas; `npm audit --audit-level=high --prefix docs-site`: 0 vulnerabilidades; Prettier e `node --check` passaram.
- **PASS** — `bun run quality:full` em `web/`: 105 arquivos/660 testes, lint, Stylelint, typecheck e build. Seis suítes focadas do HUD/mascote: 30/30.
- **PASS** — contrato revisado: uma renderização `GuardianMascot → NexPixelSprite`; sem segundo renderer/atlas/modo Companion 2D. Captura XWayland mostra um NEX. PR #55 `CLOSED`; PR #61 permanece aberta/Draft.
- **PASS** — testes focados de mascote: 25/25. HUD em Wayland direto iniciou, expandiu/recolheu, alternou apresentações, restaurou Mascote pelo tray e manteve a apresentação após reinício. Desktop abriu diretamente em Wayland e expôs a página principal via AT-SPI.
- **PASS** — após FIX-006, `bun run quality:full`: 105 arquivos/660 testes, lint, Stylelint, typecheck e build. Terminais divididos: regressão de rótulo passou em 7 geometrias e na janela Wails XWayland; captura mostra dois PTYs sem sobreposição. Digitação split e persistência seguem BLOCKED.
- **PASS** — interface Wails real importou 4 sessões host, retornou resumo agregado com 0 falhas, criou cópias `0600` no perfil Nexus isolado e voltou a detectar 3 provedores após reinício.
- **BLOCKED** — captura de pixels/alpha, teclado físico, foco/movimento em Wayland, evento real no balão, roteamento/continuidade/Skills completos, Cursor, updater assinado e proveniência imutável.
- **Matriz atualizada:** 48 PASS / 1 FAIL / 11 BLOCKED / 7 NOT_RUN (67 casos). O resultado de release segue `NO_GO_LINUX`.
- Evidências: `DEV/validation/linux-100-2026-10-25/MASCOT_VALIDATION.md`, `HUD_VALIDATION.md`, `RUNTIME_VALIDATION.md` e `PR61_DOCS_SITE_VALIDATION.md`.

## 2026-10-25 — Homologação Linux AMD64 (matriz base antes do follow-up nativo)

- **NO_GO_LINUX** — HEAD local/remoto `bf6adf0c80ac3531d7291844512a14a017ab0002`; overlay `af7476a2ab2ffeabcbfcaf35051d6d24d61a0a35bd5786b5429dc2ee3a0662e4` (1.694 fontes); VERSION=1.0.0-beta.10; worktree não congelado.
- **PASS** — `make quality-full` após correções; Go/race/vet/lint/govulncheck e frontend 103 arquivos/633 testes. Vinte temas × três viewports passaram no E2E browser.
- **PASS** — builds CLI/Desktop/HUD AMD64; Core multi-instância, colisão e shutdown; snapshot GoReleaser atual e tarballs Desktop/HUD; DEB instalado/removido em dpkg root temporário.
- **PASS** — 25/25 testes focados do mascote; captura nativa XWayland do HUD atual mostra um NEX. Não existe personagem ou modo Companion 2D.
- **PASS** — Mission real DIRECT via Nexus terminou `COMPLETED_VERIFIED`; `go test ./...` exit 0 no worktree temporário.
- **BLOCKED** — atenção real no HUD; outros modos de roteamento/continuidade/skills; importação Cursor; teclado/foco/movimento Wayland; alpha; updater assinado; proveniência imutável.
- **FAIL** — versão beta.10 e ausência de release/artefatos estáveis 1.0.0.
- **NOT_RUN na matriz base, posteriormente coberto:** diálogo nativo de importação. Ainda não executados: captura nativa do balão fixture, soak/crash, persistência/multimonitor e temas GTK.
- Matriz: 41 PASS / 1 FAIL / 9 BLOCKED / 9 NOT_RUN (60 casos). Dossiê: `DEV/validation/linux-100-2026-10-25/`.

**Escopo atual do mascote:** a decisão de produto é manter um único NEX. As referências abaixo, em registros de 2026-10-23 e anteriores, que listam a ausência do Companion 2D/PR #55 como falha são históricas e foram superadas; Companion 2D não é requisito nem defeito da release atual. A PR #55 foi fechada em 2026-10-25 e não faz parte da implementação aprovada.

## 2026-10-23 — Homologação Linux e reinstalação final

- **PASS** — `make quality-full`, `make vet`, `make test-e2e`, segurança e builds CLI/Desktop/HUD em Go 1.26.10. Frontend 99 arquivos / 612 testes.
- **PASS** — instalação nas duas raízes locais; version JSON beta.10/amd64. Web, Core Desktop, HUD e quatro hosts retomados com os hashes atualmente instalados; health Web/Core/HUD HTTP 200.
- **FAIL** — Companion 2D da PR #55 ausente. E2Es composer workspace, surface, execution spine e vague spine falham; dois presets Composer também falham; UI de importação sem confirmação visível.
- **BLOCKED/NOT_RUN** — limitações GNOME Wayland/X11, pacote e updater oficial, fluxos externos de provedor, matriz interativa nativa e resiliência; detalhados na matriz.
- Veredito `NO_GO_LINUX`, 30 PASS / 7 FAIL / 6 BLOCKED / 12 NOT_RUN. SHA-base remoto `bf6adf0c80ac3531d7291844512a14a017ab0002`, worktree dirty, sem commit/push. Dossiê: `DEV/validation/linux-100-2026-10-23/LINUX_HOMOLOGATION_REPORT.md`.

## 2026-10-23 — Follow-up de sessões e comando de login

- **PASS** — `make quality`: lint Go 0 issues, suíte Go completa (69 pacotes com testes passaram; 13 sem testes), Web 99 arquivos / 610 testes e contrato de versão.
- **PASS** — `make vet`; `go test -race -tags=gtk3 ./internal/app ./internal/profile`; build `make build-desktop` e smoke final Wails em storage temporário, Core/UI HTTP 200, SIGTERM exit 0.
- **PASS** — regressão do comando do screenshot: UI gera `nexus login codex` para perfil padrão e mantém argumentos de perfil explícito; o CLI resolve o perfil e chamou o `login` do Codex fake.
- **BLOCKED** — screenshot e interação real do cartão: compositor GNOME Wayland rejeita `grim` e screenshot D-Bus. Não foi clicado **Importar sessões existentes** na janela nativa.
- Estado do teste: base `f62fdec0c571ac6a19a904164d3f580799da6bf3` + worktree dirty; nenhum commit/push. Dossiê e logs: `DEV/validation/linux-100-2026-10-22/SESSION_IMPORT_FOLLOWUP.md` e `logs/host-session-import-*`.
- **PASS** — fetch de `origin/release/v1.0.0` permaneceu em `f62fdec0c571ac6a19a904164d3f580799da6bf3`; builds `make build`, `make build-desktop` e `make build-hud` exit 0. Três saídas são ELF x86-64; `ldd` de desktop/HUD sem dependências ausentes. SHA-256: CLI `a9035c8b92538aa80b2d2fe469410ba989f457a6c04cbc2c796f0b19f2f17e44`, desktop `aa22cc5ab78644ba384ec68a9d5417fb9014787f0fdd8c5762f46c11becac206`, HUD `13982c259df5606e73a7c1d70ad12bea4b84c3384d17a172ef2599d128cc1d89`. Compilação feita no worktree dirty, não no SHA remoto puro; logs em `DEV/validation/linux-100-2026-10-22/logs/build-current-2026-10-23-*`.
- **PASS — instalação local:** recompilação seguida de hash parity em ambas as pastas `~/.local/bin` (home e perfil Codex); CLI instalado reporta `1.0.0-beta.10`, commit embutido `f62fdec0`. Desktop/HUD sem bibliotecas dinâmicas ausentes. `nexus web` reiniciado (PID 3393460), health HTTP 200/version `1.0.0-beta.10`; rollback: `~/.local/state/nexus-update-backups/20261023T131106Z`. Evidência: `logs/install-current-2026-10-23.log` e `logs/web-restart-current-2026-10-23.log`.
- **PENDENTE DE REINÍCIO:** 11 processos Nexus `__control-host` mantidos vivos; executáveis já substituídos no disco, mas os processos residentes continuam carregando a imagem antiga até o encerramento natural/reinício das sessões.

## 2026-10-22 — Homologação Linux 1.0.0 — f62fdec0

- **PASS** Linux AMD64 CLI/Desktop/HUD builds em Go 1.26.10; Wails Desktop e HUD linked iniciados em Ubuntu 25.10/GNOME 49 Wayland; Core respondeu HTTP 200.
- **PASS** frontend `quality:full` (98 arquivos/605 testes), Go `vet`, suíte completa e race detector, golangci-lint, govulncheck, `make format-check`, pacote Debian snapshot e archives temporários Desktop/HUD.
- **FAIL** Companion 2D não integrado no SHA exato; versão do source segue beta.10 e não existe release estável assinada; cinco suítes E2E falham; `docs-verify` acusa visual manifest stale.
- **BLOCKED** screenshots/controle da janela pelo compositor Wayland, GPU 3D ausente, provedores autenticados live, fluxo de instalação estável sem manifesto. **NOT_RUN** X11, segundo monitor/escala, soak/crash recovery e ShellCheck.
- **Veredito: NO_GO_LINUX**. Detalhes e matriz: `DEV/validation/linux-100-2026-10-22/`.

---


## 2026-10-21 — Click-through do HUD corrigido em 047f4348

- **PASS** `gofmt`, `go vet ./...`, `go test -count=1 ./...` (Go 1.26.10).
- **PASS** Web `npm run quality`: 97 arquivos / 598 testes.
- **PASS** `docs-verify`: 192 Markdown / 33 assets, manifesto em `047f4348`.
- **PASS** build HUD `production`, SHA-256 `19c5e299...a1e6a5`.
- **PASS** click-through real: `RESULT.json` com `verdict: PASS`, canto transparente atravessa, cápsula interativa, estado restaurado.
- **PASS (funcional)** HUD 20/20 ciclos (`PASS_FUNCTIONAL_PENDING_VISUAL`).
- **NOT_TESTED** click-through no popover destacado; confirmação visual humana.
- **BLOCKED_EXTERNAL** assinatura; **NOT_RUN** smokes Codex/multi-conta.
- Veredito: `NO_GO_WINDOWS`. Relatório: `DEV/validation/local-hud-047f4348/HOMOLOGATION.md`.


## 2026-10-21 — Homologação Windows da release em 7d92b942

- **PASS** `docs-verify`: 192 Markdown / 33 assets, manifesto em `7d92b942`.
- **PASS** `verify-version-contract`, `gofmt` (após correção), `go vet ./...`.
- **PASS** `go test -count=1 ./...` com Go `1.26.10` (via `GOTOOLCHAIN`); `internal/app` 88,9 s.
- **PASS** Web `npm run quality`: 96 arquivos / 593 testes, typecheck, ESLint, Stylelint, Prettier.
- **PASS** build Web e build nativo do HUD (`production`), binário SHA-256 `8dab24a7f616fe6fb720dc50d33cd6f6b6802328993614db2d987eafb48dcefb`.
- **PASS** certificação oficial `scripts/certify-windows-first-run.ps1` em `7d92b942`: 9 gates PASS, 4 NOT_RUN, exit 0, `automated_result: PASS`.
- **PASS (funcional)** HUD nativo: 20/20 ciclos em janela única e 20/20 em popover destacado (`PASS_FUNCTIONAL_PENDING_VISUAL`).
- **FAIL** click-through do HUD: clique em área transparente (`1696,972`) manteve `Nexus HUD` em foreground em vez da janela alvo. Evidência em `DEV/validation/local-hud-7d92b942/click-through/`.
- **INCONCLUSIVO** matriz HUD cross-platform: sem Linux X11 e macOS no mesmo SHA; agregador recusa emitir veredito.
- **BLOCKED_EXTERNAL** instalador assinado: `native-signing` com 0 secrets, sem certificado local, `signtool`/`makensis` ausentes.
- **NOT_RUN** `desktop-wails-first-run`, sessão Codex interativa e cenário multi-conta.
- Veredito: `NO_GO_WINDOWS`. Relatório: `DEV/validation/local-hud-7d92b942/HOMOLOGATION.md`.

## 2026-10-17 — Correção e Verificação do Backend Compartilhado Desktop (76587c39)

- `internal/control/web/desktop_attach.go`: PASS. Corrigido o `NewLoopbackBackendHandler` para servir arquivos estáticos embutidos (`bundle.js`, `bundle.css`, chunks) em vez de retornar incondicionalmente `index.html`.
- `internal/control/web/desktop_attach_test.go`: PASS. Novo teste TDD `TestLoopbackBackendHandlerServesStaticAssetsAndFallbackToSPA` valida que assets estáticos nunca realizam fallback indevido para HTML e que rotas SPA continuam caindo no `index.html`.
- `go test -v -run "TestLoopbackBackendHandler.*" ./internal/control/web`: PASS.
- `gofmt -w` e `go vet ./...`: PASS (código 0).
- Binários recompilados e reinstalados em `%LOCALAPPDATA%\Programs\IAPro Nexus`.
- Desktop validado em execução com PID e controlador WebView2 ativo.

## 2026-10-17 — Validação E2E Web + Desktop e Correção no NexusDemoApp (87216d26)

- `web/src/app/NexusDemoApp.tsx`: PASS. Resolvido o erro `useWorkspacePresentation must be used inside WorkspacePresentationProvider` ao envolver o `<DemoCoordinator>` no `<WorkspacePresentationProvider>`.
- `bun run --cwd web typecheck` & `bun run --cwd web lint`: PASS (código 0, zero erros de tipagem ou lint).
- `bun run --cwd web test`: PASS (89 arquivos de teste, 553 testes aprovados sem falhas).
- Suíte E2E do Navegador (Web / Playwright):
  - `test:e2e:composer`: PASS.
  - `test:e2e:surface`: PASS (isolamento de abas e keep-alive de terminais).
  - `test:e2e:capacity`: PASS.
  - `test:e2e:hud`: PASS.
  - `test:e2e:composer-spine` & `test:e2e:composer-vague-spine`: PASS.
  - `test:e2e:composer-themes`: PASS (20 temas validados).
  - `test:e2e:hardening`: PASS (WCAG 2.1 AA a11y, 7 breakpoints responsivos, deep links).
- Suíte E2E do Desktop (Wails v3 + WebView2 via CDP):
  - Todas as rotas inspecionadas e validadas: `/projects`, `/settings`, `/updates`, `/welcome`, `/demo`.
  - Todas as APIs de backend retornando 200 no contexto do shell Desktop.
  - Evidências salvas em `DEV/validation/local-beta/1.0.0-beta.10/85c5b9589ec19663023fcddb18d0539742367b40/windows-AMD64/desktop-native/`.
- Commit `87216d26` integrado por fast-forward na branch canônica `release/v1.0.0`.

## 2026-10-17 — Continuidade de homologação Windows beta.10

- `bun run --cwd web quality:full`: PASS no worktree baseado em
  `5d5aea9211b5154e96b036c519a61de9723593b1`; 89 arquivos/553 testes e build.
- `bun run --cwd web test:a11y`: PASS em três execuções após espera limitada
  de 250 ms por foco; asserção preservada. Esta alteração de teste ainda não
  está commitada e não certifica o SHA-base.
- `pwsh -File scripts/test-uninstall-windows.ps1`: PASS com TEMP/TMP em D:;
  coordenadas beta, mismatch de manifesto, digest, path segments, variável de
  ambiente, preservação de dados e diretórios desconhecidos.
- `go test -count=1 ./internal/release ./internal/update/...`: PASS. `go run
  ./scripts/verify-version-contract.go .`, `node --check` e `git diff --check`:
  PASS.
- Falhas obrigatórias mantidas: `go test -count=1 ./...` exit 1 no timeout
  padrão de 10 min em `internal/nexus`; golangci-lint exit 4 em duas expirações
  de `go/packages`; browser `test:a11y` falhou uma vez no SHA-base.
- NSIS lifecycle/signing, HUD visual matrix, Codex interativo, updater A→B,
  CI exact-SHA e Linux nativo permanecem BLOCKED/UNVERIFIED. Relatório completo
  e logs: `DEV/validation/local-beta/1.0.0-beta.10/5d5aea9211b5154e96b036c519a61de9723593b1/windows-AMD64/`.
- Veredito `NO_GO_WINDOWS`; sem commit, push ou publicação nesta atualização.

## 2026-10-15 — Homologação Windows beta.10 no SHA exato, instalação local e smoke Desktop/HUD

- PASS — cadeia local completa (mesmos scripts do CI) no SHA `c5cf076a96934f83c8dd9fffec6e957db881e637`: frontend (format/typecheck/lint/styles/test/build), Go (vet/golangci/govulncheck/test), control/runtime E2E, agent-interop, browser E2E (hardening/a11y/visual, composer, surface, capacity, hud, spine, vague-spine, composer-themes 20/20), smokes doctor/providers/usage + regressão OpenCode e builds Wails v3 Desktop/HUD; veredito `PASS automated Windows beta homologation: 1.0.0-beta.10 @ c5cf076a`. Evidência em `DEV/validation/local-beta/1.0.0-beta.10/c5cf076a96934f83c8dd9fffec6e957db881e637/windows-AMD64/` e `%TEMP%\opencode\local-gate-c5cf076a.log`.
- PASS — correção do flake do gate `composer-themes`: o × de fechar aba deixou de ser um `<button>` filho do `role=tablist` (Axe `aria-required-children` critical) e virou span decorativo `aria-hidden` nos dois strips, preservando fechamento por teclado via `Delete` e menu de contexto (mesmo diálogo de confirmação); reprodução determinística RED→GREEN no estado exato da falha.
- PASS — captura oficial de docs refeita no SHA (`source_sha=c5cf076a`); `docs-verify` 179 Markdown/33 assets; artefatos regenerados commitados em `b462e844`.
- PASS — instalação local via `install.ps1 -BuildFromSource`: CLI `1.0.0-beta.10` commit `c5cf076a` em `%LOCALAPPDATA%\Programs\IAPro Nexus` (+ alias `ai.exe`), PATH do usuário atualizado e `doctor` executado no install.
- PASS — Desktop e HUD (builds Wails v3 da própria cadeia; SHA-256 conferidos na cópia: `4a6aea46…`/`c1d26699…`) copiados para o diretório de instalação com atalhos no Desktop e Start Menu.
- PASS — smoke nativo: `nexus-desktop.exe` vivo após 9s com janela `IAPro Nexus` e 6 processos WebView2; `nexus-hud.exe` vivo após 9s com janela `Nexus HUD` e 6 WebView2; ambos encerrados sem processos residuais.
- LIMITAÇÃO PRÉ-EXISTENTE — em runs de fixture o PUT de layout nunca é disparado (GET retorna `{}`/rev 0), tornando a persistência de abas timing-dependent nesses runs; fora do escopo desta certificação e documentado no commit `b462e844`.
- NÃO EXECUTADO — `MANUAL_CHECKS.md` (smokes nativos interativos: teclado Codex, Mission/promoção, quota/certify, HUD autostart) aguardam sessão interativa do usuário.
- NO TAG / NO PUBLISH — somente commits na branch `fix/release-beta10-windows-closure-20261014`, todos com `[skip ci]`; `release/v1.0.0` intocada.

## 2026-10-07 — Continuidade Windows WebView2 e branch audit

- PASS — go fmt ./..., go vet ./..., govulncheck ./..., golangci-lint run ./... e go test -count=1 ./... no Windows AMD64 após mudanças Go.
- PASS — teste LOCALAPPDATA do perfil WebView2 e testes dos pacotes internal/desktop, cmd/nexus-desktop e cmd/nexus-hud.
- PASS — build Desktop/HUD com IDs isolados de smoke; Desktop: janela IAPro Nexus + Chrome_WidgetWin_0 por 9s e perfil criado; HUD: processo vivo por 8s e perfil criado.
- PASS — parser PowerShell do smoke Desktop e parser YAML do workflow CI.
- PASS anterior — Bun frozen install, quality:full e oito suítes Playwright nesta árvore Windows; não repetidos nesta continuação porque nenhum arquivo frontend foi alterado.
- PASS — auditoria viva: nenhuma ref Beta 7 nova. KIVER (d14154d2) é ancestral de release (52f35c04). As branches terminal/testes foram comparadas ao código e cobertura atuais; nenhuma integração pendente.
- PASS — branch antiga codex/windows-beta6-convergence revisada (5 commits exclusivos no grafo): equivalentes atuais de startup Wails, API same-origin/WebSocket, descarte de runtime inativo e resolução .cmd estão na release. A mudança para ConPTY padrão foi posteriormente revertida por cc6400db.
- PASS — go test -count=1 -run TestWindowsBackendInteractiveConPTYOptIn -v no Windows AMD64; ConPTY opt-in interativo validado. O padrão de release continua opt-in.
- PASS — parser PowerShell e smoke Desktop após restringir cleanup ao perfil Nexus: Wails/WebView2 ativos 10s, perfil isolado criado, zero processo WebView2 desse perfil após cleanup. Evidência e hash do binário em outputs/windows-desktop-smoke-cleanup-52f35c04-20261007/.
- PASS — go test -count=1 ./internal/control/host ./internal/runtime no Windows AMD64; ambas as suítes passaram.
- PASS — bun run --cwd web vitest run src/nexus/terminalInput.test.ts no Windows; 28 testes passaram.
- LIMITAÇÃO — release local e remoto estão em 52f35c04, cinco commits após d14154d2; o worktree está sujo, portanto não certifica SHA imutável. Refs remotas foram consultadas ao vivo nesta continuação.
- PASS — NSIS 3.13 compilou o nexus.nsi exato da árvore atual; instalador unsigned de 23.917.346 bytes, SHA-256 7f310bf94d318d339ccd332d4c2de6b924404f87574e3adfccdb914fa55f5cf4.
- PASS — variante com registry/atalho isolados instalou e desinstalou o payload real: setup/uninstall 0, CLI beta.7, doctor JSON válido; Path e Desktop reais inalterados.
- NÃO EXECUTADO — install/uninstall do executável NSIS exato no perfil do usuário, Authenticode, trust root, CI exact-SHA e gates Linux.
- FAIL — bun scripts/docs-verify.mjs: o manifesto visual continua em 2892f301, anterior a mudanças visuais do workspace, e visual source também está dirty. O fluxo scripts/refresh-beta-docs-windows.ps1 exige SHA limpo; recapturar e repetir o gate após congelar o candidato.
- NO TAG / NO PUBLISH — manter NO-GO até commit revisado, homologação no SHA final e gates externos de assinatura/instalador.
- Nota Git: git status/diff-check não puderam ser usados neste clone nesta execução (fatal: this operation must be run in a work tree); resultados reportados são os comandos executados e as evidências de smoke listadas em DEV/validation/current/BETA7_WINDOWS_HOMOLOGATION_20261007.md.

## 2026-10-07 — Correções Windows ProjectHub e origem Wails

- PASS — go fmt ./..., go vet ./..., govulncheck ./..., golangci-lint run ./... e go test -count=1 ./... no checkout Windows atualizado.
- PASS — bun install --cwd web --frozen-lockfile.
- PASS — bun run --cwd web quality:full com Bun 1.3.9 e Node 24.19.0; 82 arquivos e 487 testes Vitest, build incluído.
- NOTA DE AMBIENTE — a primeira execução com Node 18.12.1 falhou ao carregar import JSON do Stylelint. Repetição com Node 24.19.0 passou sem alteração de dependências.
- PASS — oito suítes Playwright E2E no worktree isolado com as mesmas alterações; captura ProjectHub 775x483 confirma mensagem em toda a largura e CTA em uma linha.
- PASS — três builds Windows AMD64 e checksums; CLI informou versão 1.0.0-beta.7 e commit 52f35c04-dirty. Authenticode NotSigned; trust root ausente.
- PASS — smoke de oito segundos do HUD; WebView2 criado.
- INCONCLUSIVO — Desktop automatizado encerrou com código 0 antes de oito segundos; binário de baseline repetiu o comportamento no mesmo launcher. Repetir em sessão Windows interativa.
- NO TAG / NO PUBLISH — código local sem commit; CI Windows exact-SHA e assinatura/trust root bloqueados.
- Escopo Windows-only; nenhum gate Linux executado nesta continuação.

## 2026-10-07 — Correção local do estado de Path no NSIS

- PASS — `go test -count=1 ./internal/release` no checkout Windows real.
- PASS — `go test -count=1 ./...`, `go vet ./...` e `golangci-lint run ./...` no worktree isolado de `52f35c04` com o mesmo patch local.
- PASS — smoke NSIS portátil 3.13, em subchave de registro isolada e payload fictício: 5/5 estados, incluindo `REG_SZ`, `REG_EXPAND_SZ`, valores vazios e Path ausente.
- PASS — `git diff --check` nos dois arquivos de código.
- NÃO EXECUTADO — build do instalador de distribuição com payload real e assinatura Authenticode após a correção.
- BLOQUEADO — CI Windows exact-SHA aguardando runner; certificado, secrets e `NEXUS_UPDATE_PUBLIC_KEY` ausentes.
- NO TAG / NO PUBLISH — a correção é local, não commitada; não existe nova certificação imutável.
- Escopo Windows-only; nenhum gate Linux executado nesta continuação.

## 2026-10-07 — Homologação técnica Windows beta.7

- PASS — branch local/remota release/v1.0.0, SHA 52f35c04181b85379219ab55fc3192c63a5440a8; VERSION=1.0.0-beta.7.
- PASS — go fmt ./..., go vet ./..., govulncheck ./..., golangci-lint run ./... e go test -count=1 ./... no Windows.
- PASS — bun install --cwd web --frozen-lockfile e bun run --cwd web quality:full com Node 22.14.0/Bun 1.3.9; 82 arquivos, 487 testes unitários e build.
- PASS — oito suítes E2E Playwright: geral, composer, surface, capacity, hud, composer-spine, composer-vague-spine e composer-themes.
- PASS — builds CLI/Desktop/HUD para Windows AMD64 com commit 52f35c04 embutido; CLI version e doctor --json; Desktop e HUD vivos por oito segundos cada.
- PASS — NSIS 3.13 compilado e smoke isolado na fonte revisada; código NSIS é idêntico no commit 52f35c04. PATH manteve o texto, mas seu tipo foi normalizado de REG_SZ para REG_EXPAND_SZ.
- PASS — checksums SHA-256 dos três executáveis registrados nos artefatos outputs/windows-revalidation-52f35c04; Authenticode NotSigned e trust root ausente, portanto não são distribuição.
- BLOCKED — run nativo Windows exact-SHA 37666126714 está queued; só existe runner Linux/offline, sem runner Windows.
- BLOCKED — assinatura: certificado privado, PFX secrets, signtool.exe e NEXUS_UPDATE_PUBLIC_KEY ausentes no host.
- NO TAG / NO PUBLISH — tag v1.0.0-beta.7 ausente e endpoint de release retornou 404. Veredito: GO técnico local Windows, NO-GO para distribuição.
- Escopo desta continuação: Windows; nenhum gate Linux foi executado. Capturas versionadas geradas incidentalmente pelos E2E foram restauradas ao baseline; logs desta rodada foram guardados fora do repositório.

## 2026-09-19 — Correção do attach por geração/PID

- PASS — `go test ./...`.
- PASS — `go test -race ./internal/control/launcher ./internal/control/protocol ./internal/control/host`.
- PASS — `make format-check` e `make vet`.
- PASS — `make build` e `make install-local`; binário instalado em
  `/home/desenvolvedor/.local/bin/nexus`, commit `50b561c`.
- PASS — Web reiniciado pelo binário novo e `GET http://127.0.0.1:13000`
  retornou HTTP 200.
- PASS — smoke real `nexus codex --yolo`: runtime iniciou com
  `host_generation` não-zero e sem “changed generation/PID”.
- PASS — smoke real `nexus agy`: runtime iniciou com `host_generation`
  não-zero e sem “changed generation/PID”.
- PASS — os hosts antigos incompatíveis foram encerrados após backup explícito
  do registro; nenhum `__control-host` órfão permaneceu.
- PASS — o host residual de captura de documentação em `/tmp` também foi
  encerrado via protocolo, sem alterar o registro principal.
- PASS — `make lint-go` após corrigir os dois avisos estáticos preexistentes
  (`ST1023` e `QF1001`) sem alteração de comportamento.
- PASS — versão `1.0.0-beta.3` consistente em `VERSION`, `web/package.json` e
  `cmd/nexus-desktop/wails.json`; binário instalado reporta a mesma versão.
- NOT EXECUTED — commit/push, conforme o contrato do Maestro.

## 2026-09-18 — Proteção permanente contra falhas de teclado

- PASS — testes Go afetados, `-race`, `make test-e2e`, `go vet ./...`,
  `make build`, `make web-verify` (10/10) e cross-compile Windows/macOS.
- PASS — pseudo-TTY executou `./nexus doctor terminal --recover` com reset
  emitido e termios legível.
- PARTIAL — `go test ./...` com dados temporários reproduz fixtures
  preexistentes de conversation/AGY/Claude/quota/profile; nenhum pacote do
  slice de TTY falhou.
- PARTIAL — `make lint-go` mantém cinco diagnósticos preexistentes fora do
  slice.
- BLOCKED — smoke autenticado `nexus codex --yolo` sem perfil Codex habilitado;
  Windows/macOS nativos continuam não verificados.

## 2026-09-18 — Beta Linux CI closure preparation

- PASS — `make quality` (format, frontend lint/styles/typecheck, 409 Vitest
  tests, Go lint, Go tests and version contract).
- PASS — release contract tests, Wails/Linux build, browser E2E/a11y/visual,
  GoReleaser check and Linux snapshot, DEB container smoke and `make docs-verify`.
- PASS — `actionlint` for CI/release workflows and `git diff --check`.
- OPERATIONAL — repository runner `nexus-ci-dev-web-kiver` is online with
  labels `self-hosted`, `Linux`, `X64`, `nexus-ci`.
- PENDING — commit/push, hosted same-SHA 7/7 run, repeated Linux run, merge,
  green `main`, distribution token and signed beta release.

## 2026-09-17 — PR #2 Windows closure patch

- PASS — `go test -v ./internal/core/provider/adapters/codex -run
  '^TestPrepareEmptyDestWithManyRolloutsRespectsBudget$' -count=20`.
- PASS — `go test -v ./internal/nexus -run
  '^TestInterventionResolutionSurvivesStoreReopen$' -count=20`.
- PASS — `go test ./internal/core/provider/adapters/codex/... ./internal/nexus/... -count=1`.
- PASS — `go test -race ./internal/core/provider/adapters/codex/... ./internal/nexus/... -count=1`.
- PASS — `git diff --check`.
- PASS — `make web-verify` (10/10 gates; relatório frontend atualizado).
- BLOCKED/EXISTING — `make docs-verify` detecta manifesto visual stale desde
  `0ec7e750`; não relacionado às alterações desta correção e requer campanha
  visual separada.
- PENDING — native Windows repeated run, hosted same-SHA CI, maintainer merge,
  `main` CI e tag beta.1 no commit entregue.

## 2026-09-16 — Reassert após interrupção Codex

- PASS — testes de regressão de `Ctrl+C`, forwarding byte-transparente e
  bracketed paste: `go test ./internal/runtime ./internal/app -run
  'Test(WriteTerminalKeyboardModeReset|TerminalKeyboardModeResetDisablesBracketedPaste|ForwardAttachedInput|ContainsTerminalInterrupt)$' -count=1`.
- PASS — `go test ./internal/runtime ./internal/app ./internal/control/host -count=1`.
- PASS — `go test -race ./internal/runtime ./internal/app -count=1`.
- PASS — `make build && make LOCAL_BIN=/home/desenvolvedor/.local/bin install`.
- PASS — binário instalado: `IAPro Nexus 1.0.0-beta.1`, commit `b3db75c`;
  `git diff --check` limpo.
- Observação operacional: PIDs 111266/111295 ainda executam a imagem anterior
  `(deleted)` em memória e precisam de reinício para carregar esta correção.

## 2026-09-16 — Guard persistente de teclado do terminal

- PASS — `go test ./internal/runtime ./internal/app ./internal/control/host -count=1`.
- PASS — `make build && make install LOCAL_BIN=/home/desenvolvedor/.local/bin`.
- PASS — `git diff --check`.
- Aplicado reset pontual de Kitty/CSI-u, modifyOtherKeys e foco estendido ao
  TTY externo da sessão antiga, preservando o processo e seu contexto.

## 2026-09-16 — Auditoria Contínua, Extração de E-mail do Cursor e Consolidação de Qualidade

- PASS — `go test -v ./internal/core/provider/adapters/cursor/...`: 2/2 testes unitários aprovados (`TestInspectAuthMemoizesStatusProbe` e `TestInspectAuthExtractsEmail`).
- PASS — `go test ./internal/core/provider/adapters/...`: 100% de aprovação em todos os adapters (`agy`, `claude`, `codex`, `cursor`, `gemini`, `opencode`).
- PASS — `go test ./internal/...`: Todos os pacotes Go internos aprovados (`internal/app`, `internal/nexus/...`, `internal/control/...`, `internal/core/...`, `internal/profile`, `internal/tui`, etc.).
- PASS — `make vet`: Go vet limpo (0 issues).
- PASS — `/home/desenvolvedor/go/bin/golangci-lint run ./...`: 0 issues.
- PASS — `make format-check`: Formatação Prettier e gofmt 100% conformes.
- PASS — `make web-verify`: 10/10 gates aprovados (`format`, `typecheck`, `lint`, `stylelint`, `null-arrays`, `test` 379/379 Vitest, `i18n`, `build`, `embed-sync`, `ui-markers`).
- PASS — `make security` (`govulncheck`): 0 vulnerabilidades detectadas.
- PASS — `make version-check`: Todas as fontes consistentes em `v1.0.0-beta.1`.
- PASS — `nexus list`: Exibe `kiver@omegasistemas.net.br` para o perfil `cursor:cursor` com status `Authenticated`.
- PASS — `make build && make install LOCAL_BIN=/home/desenvolvedor/.local/bin`: Binário atualizado instalado em `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `GET http://127.0.0.1:13000/api/v1/health`: Retorna `{"status":"ok","timestamp":...,"version":"1.0.0-beta.1"}`.
- NOT EXECUTED — `git commit` / `git push` (preservando o contrato de governança do Maestro).

## 2026-09-16 — Correção de navegação CSI embutida

- PASS — `npm --prefix web run lint` e `npm --prefix web run typecheck`.
- PASS — `npm --prefix web test -- --run src/nexus/terminalInput.test.ts` (20/20).
- PASS — `make build && make install LOCAL_BIN=/home/desenvolvedor/.local/bin`.
- PASS — `make web-verify` com 10/10 gates e `git diff --check`.

## 2026-09-15 — Revalidação completa da fundação conversacional (anti-perda)

Evidência contra o stash `stash@{0}` + gates locais:

- PASS — inventário: todos os arquivos conversation do stash no disco (feature, spike, bootstrap, handlers, 0021, ADR/docs).
- PASS — wiring: Composer → `ConversationSurface`; NDJSON create; upload attachments; CSRF; `@assistant-ui/react`; `/dev/conversation-spike`; i18n `conversation` ×3.
- PASS — DB `~/.local/share/ai-manager/nexus.db`: schema 21 + `kind`/`payload_json` + `conversation_attachments`.
- PASS — `go test ./internal/nexus/store/ ./internal/nexus/ ./internal/control/web/ -count=1`
- PASS — Vitest conversation + composerModel + i18n (19)
- PASS — `make web-verify` 10/10 (`FRONTEND_2026-09-15T13-36-13Z.md`)
- PASS — `nexus web :13000` serve UI nova; exe não deleted.
- NOT EXECUTED — commit/push (aguardando Maestro).

## 2026-09-15 — Resolução de Autenticação e UX do OpenCode (Modo Local, Multi-Provedor & Paridade CLI)

- PASS — `go test -v ./internal/core/provider/adapters/opencode/...`: 7/7 testes unitários aprovados (capacidades Login/Logout, modo local sem login, Zen provider, DeepSeek provider, cópia segura de credenciais do host no `Prepare`, probes de cota).
- PASS — `go test ./internal/profile/...`, `go test ./internal/nexus/...`, `go test ./internal/control/web/...`: 100% de aprovação sem regressões.
- PASS — `npm --prefix web run check:styles`: SCSS Modules em estrita conformidade com allowlist.
- PASS — `npm --prefix web run lint:styles`: Stylelint aprovado com 0 violações.
- PASS — `npm --prefix web run typecheck`: TypeScript compilado limpo (0 erros).
- PASS — `npm --prefix web run lint`: ESLint com `--max-warnings=0` aprovado.
- PASS — `npm --prefix web test`: Vitest executou 71 arquivos e 379 testes com 100% de aprovação.
- PASS — `make web-verify`: 10/10 gates automatizados aprovados (format, typecheck, lint, stylelint, null-arrays, test, i18n, build, embed-sync, ui-markers). Relatório em `DEV/validation/FRONTEND_LATEST.md`.
- PASS — `make build && make install LOCAL_BIN=/home/desenvolvedor/.local/bin`: Binário compilado e instalado em `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `curl -H "Authorization: Bearer <token>" http://127.0.0.1:13000/api/v1/resources`: OpenCode reportado com `authenticated: true`, `health: healthy`, `plan: OpenCode (deepseek)` e `account: deepseek`.
- NOT EXECUTED — `git commit` e `git push` (em estrita conformidade com a constituição do Orquestrador).



## 2026-09-15 — Fechamento v1.0.0-beta.1 (in-repo, worktree isolado)

- PASS — `make lint-go` (0 issues; exclusão misspell em `composer_brief.go`; SA1019 com `//nolint:staticcheck`).
- PASS — `go test -count=1 ./internal/nexus/ ./internal/app/ ./internal/control/host/ ./internal/control/web/ ./internal/core/provider/adapters/opencode/` (575 testes).
- PASS — `make version-check` (`1.0.0-beta.1` consistente).
- PASS — `make web-verify` 10/10 (`DEV/validation/FRONTEND_LATEST.md`, 2026-09-15T13:04:30Z).
- ISOLATED — spike Conversation no stash `wip: conversation spike parked outside v1.0.0-beta.1`.
- NOT EXECUTED — commit, push, tag, GitHub Release, environment `beta-signing` (exigem autorização).

## 2026-09-15 — Soak prolongado

- **PASS** — `make overnight-soak DURATION=8h`, com duração observada de
  `28800.028s` (8h) e encerramento normal do teste.
- O resultado é evidência real de execução prolongada, não um smoke test curto.

## 2026-09-15 — Modernização das Superfícies de Sessões, Agentes e Project Overview (Opções 1, 2 e 3)

- PASS — `go test -v ./internal/control/web/...`: 100% de aprovação na suite web, incluindo os novos testes unitários de `handleTelemetryStats` e `handleTelemetryHistory` (`handlers_telemetry_test.go`).
- PASS — `npm --prefix web run check:styles`: Conformidade estrita com SCSS Modules e allowlist global de estilos.
- PASS — `npm --prefix web run lint:styles`: Stylelint aprovado sem nenhuma violação (incluindo keyframes kebab-case `agent-pulse` e `runtime-pulse`).
- PASS — `npm --prefix web run typecheck`: TypeScript `tsc --noEmit` compilado limpo (0 erros).
- PASS — `npm --prefix web run lint`: ESLint com `--max-warnings=0` aprovado (0 erros, 0 warnings).
- PASS — `npm --prefix web run format:check`: Prettier 100% em conformidade.
- PASS — `npm --prefix web test`: Vitest executou 71 arquivos e 378 testes com 100% de aprovação (incluindo paridade de chaves i18n em `src/i18n/i18n.test.ts`).
- PASS — `make web-verify`: Todos os 10 gates automatizados aprovados (format, typecheck, lint, stylelint, null-arrays, test, i18n, build, embed-sync, ui-markers).
- PASS — `make build`: Binário `./nexus` v1.0.0-beta.1 compilado com novos assets embutidos em `web/dist`.
- PASS — `make install LOCAL_BIN=/home/desenvolvedor/.local/bin`: Binário instalado em `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `curl -sI http://127.0.0.1:13000/`: HTTP 200 OK com novos assets servidos pelo servidor de controle.
- PASS — `curl -s -b /tmp/nexus-cookies.txt http://127.0.0.1:13000/api/v1/telemetry/stats`: Retorno JSON estruturado de telemetria agregada (`total_sessions`, `by_provider`, `by_profile`, `rate_limits_by_profile`).
- PASS — `curl -s -b /tmp/nexus-cookies.txt http://127.0.0.1:13000/api/v1/telemetry/history`: Retorno JSON de eventos cronológicos com provider, profile, duration e timestamp.
- NOT EXECUTED — `git commit` e `git push` (em estrita conformidade com a constituição do Orquestrador).


## 2026-09-15 — Compilação e Instalação Local do Binário Nexus (v1.0.0-beta.1 @ 2b9c956)

- PASS — `make build`: Bundles frontend gerados com sucesso e binário `./nexus` compilado com commit `2b9c956`.
- PASS — `make install LOCAL_BIN=/home/desenvolvedor/.local/bin`: Binário instalado atomicamente em `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `which nexus`: Resolvido para `/home/desenvolvedor/.local/bin/nexus`.
- PASS — `nexus version`: `IAPro Nexus 1.0.0-beta.1 (linux/amd64, go1.25.0) commit 2b9c956 built 2026-09-15T03:22:18Z`.
- PASS — `nexus usage --json`: Retorno estruturado de cotas e provedores funcionando sem anomalias.
- PASS — `curl -sI http://127.0.0.1:13000/`: HTTP 200 OK com novos assets do Web OS servidos.
- NOT EXECUTED — `git commit` e `git push` (em estrita conformidade com a constituição do Orquestrador).



- PASS — `go test -v -run TestUnifiedUsage_AddProfileModal ./internal/tui/...`: 4/4 novos testes unitários aprovados (abertura por tecla `a`, abertura por `enter` em provedor sem perfil, submissão de `ActionAdd` com nome informado e cancelamento por `esc`).
- PASS — `go test ./internal/tui/...`: 100% da suite do TUI aprovada.
- PASS — `go test ./internal/app/...`: 100% da suite do App CLI aprovada, validando despacho de `tui.ActionAdd` para `addCmd`.
- PASS — `npm --prefix web run check:styles`: Conformidade estrita com SCSS Modules e allowlist global.
- PASS — `npm --prefix web run typecheck`: `tsc --noEmit` compilado limpo (0 erros).
- PASS — `npm --prefix web run lint`: ESLint com `--max-warnings=0` aprovado sem erros ou avisos.
- PASS — `npm --prefix web test`: Vitest executou 73 arquivos e 388 testes com 100% de aprovação.
- PASS — `make web-verify`: Todos os 10 gates automatizados aprovados (format, typecheck, lint, stylelint, null-arrays, test, i18n, build, embed-sync, ui-markers). Relatório em `DEV/validation/FRONTEND_LATEST.md`.
- PASS — `make build`: Binário `./nexus` v1.0.0-beta.1 gerado com sucesso.
- PASS — `install -m 755 ./nexus ~/.local/bin/nexus`: Binário instalado no ambiente de execução do usuário.
- NOT EXECUTED — `git commit` e `git push` (em conformidade com as diretrizes do Orquestrador).

## 2026-09-15 — Redesign da Superfície de Uso (UX/UI, Responsividade & Paridade CLI)

- PASS — `npm --prefix web run check:styles`: validação de CSS Modules e conformidade com allowlist de estilos globais.
- PASS — `npm --prefix web run format:check`: 100% dos arquivos formatados segundo regras do Prettier.
- PASS — `npm --prefix web run lint`: ESLint aprovado com 0 erros e 0 warnings (`--max-warnings=0`).
- PASS — `npm --prefix web run lint:styles`: Stylelint aprovado sem nenhuma violação.
- PASS — `npm --prefix web run typecheck`: TypeScript `tsc --noEmit` compilado com 0 erros.
- PASS — `npm --prefix web test`: Vitest executou 73 arquivos de teste e 388 testes com 100% de aprovação (incluindo testes de modelos de uso e paridade i18n).
- PASS — `npm --prefix web run build`: Geração completa de bundles web com Tailwind CSS v4 e esbuild em ~1.3s.
- PASS — `make web-verify`: Todos os 10 gates automatizados aprovados (format, typecheck, lint, stylelint, null-arrays, test, i18n, build, embed-sync, ui-markers). Relatório gerado em `DEV/validation/FRONTEND_LATEST.md`.
- PASS — `go test -v ./internal/control/web/...`: Todos os testes da API de controle Web e endpoints de perfis/recursos aprovados (0 falhas).
- PASS — `make build`: Binário `./nexus` (v1.0.0-beta.1) compilado com sucesso com os novos assets embutidos.
- NOT EXECUTED — `git commit` e `git push` (em estrita conformidade com a constituição do Orquestrador: commits e pushes somente sob ordem explícita do Maestro).

## 2026-09-15 — Limpeza de galhos + pick terminal (HEAD `356f1a3`)

- PASS — `go test ./internal/control/host/ -count=1` (pós-pick e pós-recuperação).
- PASS — `go vet ./internal/control/host/` + `gofmt -l` limpo nos 3 arquivos do pick.
- PASS — `make quality` (frontend 73 arqs/385 testes, version contract `1.0.0-beta.1`).
- PASS — `make build` (binário `./nexus` 20,5M) + `git diff --check`.
- PASS — `make web-verify` 10/10 gates (format, typecheck, lint, stylelint,
  null-arrays, test, i18n, build, embed-sync, ui-markers).
- PASS — loss audit: 12/12 galhos apagados recuperáveis via
  `archive/2026-09-15/*`; `git bundle verify` OK.
- NOT EXECUTED — commit/push dos 5 arquivos de release staged, CI hosted
  same-SHA, `release-proof`/`overnight-soak`/`docs-verify` (exigem SHA
  congelado + Go 1.25.1/covdata).

## 2026-09-14 — Remediação do review pré-main

- PASS — `GOTOOLCHAIN=go1.25.1 make release-proof`: gates locais completos,
  cobertura Go agregada 52,4%, segurança e version contract aprovados;
  `OVERNIGHT_SOAK_EVIDENCE=MISSING` foi mantido como pendência explícita.
- PASS — `go test ./...` e `go test -race ./...` sem falhas ou data races.
- PASS — `make quality`, `make test-e2e` e `make security`.
- PASS — lint frontend com `--max-warnings=0`.
- PASS — `GOOS=windows GOARCH=amd64 go test -c` para `scripts`, `internal/control/web`
  e `internal/control/host`.
- PASS — regressão de persistência do binding `--lead` em Registry reloadado.
- BLOCKED — `make docs-verify`: manifesto visual stale para o worktree não
  commitado; exige novo SHA e evidência visual após congelamento.
- NOT EXECUTED — commit, push, CI hosted same-SHA, rulesets e branch protection.

## 2026-09-14 — CLI supervised sem Lead no teclado

- PASS — `wantInteractiveLead` default false; `--lead` true (case-insensitive); `StripControlFlags` remove `--lead` junto de `--direct`/`--supervised`.
- PASS — `go test ./internal/app/` (flag/strip + suite existente).
- PASS — smoke `nexus codex --yolo` (PTY): runtime `nexus.interactive_lead=false`; **0** agentes novos; `/status` chega ao Codex (autocomplete de slash) sem hang de promoter.
- PASS — smoke `nexus codex --yolo --lead`: label `nexus.interactive_lead=true` + bind Lead; caminho LineShadow preservado.
- PASS — `nexus control list` lista o runtime do atalho; help/completions expõem `--lead`.
- PASS — `make build` + install `~/.local/bin/nexus` (`0.5.0-beta.33`).

## 2026-09-14 — Enter preso em /status (qualquer CLI supervised)

- PASS — `IsImmediateSubmitLine` cobre `/status`, `/model`, `/nexus`, `:help`, vazio; prompts normais não bypassam.
- PASS — `decideInteractiveSubmit` não chama DecideIntent para `/status`; Decide lento estoura 250ms → KeepWithLead; Promote ainda funciona em goal multi-stream.
- PASS — `go test` promotion+app focados (11); binário reinstalado em `~/.local/bin/nexus`.
- UNVERIFIED — smoke manual `nexus codex` → `/status` responde na TUI (requer sessão interativa).

## 2026-09-14 — CLI shortcuts: busy failover + orphan host reap

- PASS — `go test ./internal/app ./internal/control/launcher ./internal/nexus ./internal/control/tui` (350): failover auto cai no próximo perfil; explícito não faz fallback; `ReapOrphanHostsFrom` mata host sem registry e preserva RUNNING.
- PASS — lock exclusivo em `kivergmail` + `nexus codex --yolo`: stderr `Profile codex:kivergmail is busy. Automatically falling back...` e attach em `codex:kiver.omegasistemas` (~8s, TUI YOLO). `nexus codex:kivergmail --yolo` falhou em 159ms sem fallback.
- PASS — `nexus control cleanup` reaped 1 orphan host de teste; 3 RUNNING reais intactos (`agy-06G9ZZ…`, `runtime_06GA0Z…`, `codex-06G9YT…`).
- UNVERIFIED — Windows named-pipe scan of `__control-host` (Linux `/proc` only).

## 2026-09-14 — Rail flicker + terminal recover

- PASS — `useNexusData` só atualiza estado quando o snapshot muda; `resolveProjectSelection` não bounceia para `projects[0]` em poll vazio; outline não re-entra em loading com cache.
- PASS — overlay de Recover só em ERROR/DISCONNECTED/recovering; `CONNECTING` usa indicador leve; WS effect sem dep de `t`.
- PASS — attach prefere `surface.data.runtimeId`; `runtime_changed` faz rebind em `new_runtime_id`; sticky `nextBoundRuntimeId` ignora poll stale.
- PASS — `make web-verify` (FRONTEND_LATEST PASS), testes focados rail/terminal, `make build`, smoke browser: expand estável + Recover Agy conectou Antigravity CLI.

## 2026-09-14 — Codex terminal input

- PASS — `web/src/nexus/terminalInput.ts` shared filter drops focus/CPR/`[i`; `AgentTerminal` and `TerminalPane` use it; mouse/CSI-u/UTF-8/`0x03`/literals forwarded.
- PASS — `internal/control/host` forwards `CmdInput` byte-for-byte; `/nexus stop` never intercepted; `LineShadow` keeps control/paste/OSC/CPR out of prompt.
- PASS — `CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT=1` in direct adapter and supervised spec/driver.
- PASS — rebuilt `/home/desenvolvedor/.local/bin/nexus` from `1b840ca`; `nexus version` `1b840ca`, `go version -m` module `github.com/IAPro-Community/nexus`.
- PASS — focused web `terminalInput.test.ts` (10), Go race (`host`, `promotion`, `driver`, `spec`), `go test ./...`, `make quality`, `make build`, `git diff --check`.
- UNVERIFIED — manual Web/Desktop + `nexus codex` acceptance (no mouse `[i`/`[I`/CSI residue; common keys no interrupt; explicit `0x03` interrupts; literals intact).

## 2026-09-14 — Codex autonomous launch compatibility

- PASS — Codex autonomous argument regression: approval/sandbox flags are
  placed before `exec` so Codex 0.154.0 accepts the command.
- PASS — Codex reviewer argument contract: read-only sandbox and no-approval
  flags are placed before `exec`; provider contract golden updated.
- PASS — targeted race tests and `make quality`; alternate Go 1.25.1
  `make release-proof` completed all local gates.
- PASS — authenticated Mission `run_06G9VJV1K6J649PN2EMEHR1EMG` routed to
  `codex:kivergmail`, executed the implementer in a dedicated worktree,
  received an approved verdict from a distinct read-only Codex reviewer, and
  completed package plus global verification as `COMPLETED_VERIFIED`.

## 2026-09-13 — P0/concurrency and release closure refresh

- PASS — `make attention-e2e`, `make overnight-smoke` and
  `make overnight-soak DURATION=45m` on the current worktree.
- PASS — `GOTOOLCHAIN=go1.25.1 make release-proof` completes all local gates
  with aggregate coverage at 52.4%; the default Go 1.25.0 installation still
  lacks `covdata` and is recorded as an environment-only block.
- PASS — authenticated Codex Mission `run_06G9VJV1K6J649PN2EMEHR1EMG`
  completed with a distinct read-only reviewer and durable package/global
  verification evidence (artifacts in
  [`validation/current/real-provider-mission/`](validation/current/real-provider-mission/)).
- BLOCKED — `make docs-verify` still sees a stale visual manifest for
  uncommitted frontend changes.
- NOT EXECUTED — push, hosted same-SHA CI, rulesets and branch protection;
  these require explicit maestro authorization.

## 2026-09-13 — Fase A promoção adaptativa

- PASS — `internal/nexus/promotion` cobre Enter retido, UTF-8/backspace,
  ESC/CSI/OSC, CPR, bracketed paste, controles e OFF passthrough.
- PASS — CLI attach e Web terminal integram `KEEP_WITH_LEAD`, `ASK` e
  `PROMOTE`; Web expõe confirmação `promotion_answer`.
- PASS — `go test ./... -count=1`, `go test -race` nos pacotes tocados,
  `go vet ./...`, `make quality` e `git diff --check`.
- UNVERIFIED — Mission com providers autenticados, CI hosted same-SHA e
  soak de oito horas; nenhum push ou branch protection foi executado.

## 2026-09-13 — Fase 0 pós PRODUCTION_NO_GO

- Scope: código apenas (falso sucesso / Maestro / Mission PATCH / quota /
  launcher / installers / CI key gate). **Não** overnight, **não** CI hosted,
  **não** Mission autenticada.
- PASS — `make quality`; `go test -race` em store/nexus/runner/quota/launcher/
  release/sign-update-manifest; smoke isolado (`maestro_mode=OFF`,
  `nexus run` → strict admission, sem wording “quota/rate limit”).
- PASS (contrato) — CI desktop Linux/Windows/macOS falham se
  `NEXUS_UPDATE_PUBLIC_KEY` vazia; installers desktop recusam
  `desktop-checksums.txt` unsigned.
- BLOCKED / NEXT — congelar SHA (commit); humano provisiona Ed25519 em
  `beta-signing` / `vars.NEXUS_UPDATE_PUBLIC_KEY`; só então push e CI same-SHA.
- UNVERIFIED — overnight 8h, Windows/macOS nativos same-SHA, Mission
  `COMPLETED_VERIFIED` autenticada, red team no SHA novo.

## 2026-09-13 — Current local closure audit

- Candidate observed: `eed65e5a7e46adc896165c94accb7fc73fc6750f`; worktree was
  not frozen, so no release certification is asserted.
- PASS — `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet
  ./...`, `make lint-go`, `make quality`, `make security`, `make build`,
  `make build-desktop-wails`, `make web-verify`, and
  `bash scripts/verify-update-gate.sh`.
- PASS — Go/browser regressions preserve PTY bytes including ESC/OSC/kitty,
  UTF-8 and literal `/e[...]` or `\\e[...]` provider text.
- BLOCKED — `make docs-verify` because the visual manifest is stale for
  uncommitted frontend changes.
- UNVERIFIED — authenticated Mission evidence, production updater key
  provisioning, native Windows/macOS/NSIS, hosted same-SHA browser/release
  matrix, and eight-hour overnight certification.

## 2026-09-13 — Fase B revisão/remediação

- PASS — ciclo canônico Planner → Builder → Reviewer (`CHANGES_REQUESTED`)
  → remediação com finding novo → `APPROVED` → Verifier sem write turn.
- PASS — ledger distingue quatro identidades e termina sem findings abertos;
  `make overnight-smoke` executa o teste.
- BLOCKED — `make docs-verify` exige manifesto visual renovado para as
  alterações frontend não commitadas; nenhuma regeneração foi feita.
- BLOCKED — `make docs-verify` still requires a refreshed visual manifest for
  uncommitted frontend changes; no manifest regeneration was performed.

## 2026-09-13 — Mission Teams validation

- PASS — full Go tests, full race suite, `go vet`, security, build, desktop
  build, E2E, Attention E2E and bounded overnight smoke.
- PASS — canonical deterministic Planner → Builder → Reviewer → remediation →
  Verifier test and direct-task anti-overprocessing coverage.
- PASS — structured finding identity/deduplication/resolution, durable phase
  state, Flow/WorkPlan round-trip and desired-vs-actual routing preference.
- FAIL / ENVIRONMENT — `make coverage` and `make release-proof` stop because
  the active Go 1.25 toolchain lacks `covdata`.
- FAIL / CONCURRENT WORKTREE — latest `make web-verify` reports formatting in
  unrelated `web/src/app/NexusSplashScreen.tsx`; the Mission Teams surface
  itself passed typecheck, lint, styles, tests and build.
- UNVERIFIED — authenticated real-provider Mission, provider failover proof,
  native Windows/macOS and eight-hour overnight certification.
- Report: [`validation/current/MISSION_TEAMS_REPORT.md`](validation/current/MISSION_TEAMS_REPORT.md).

## 2026-09-13 — Production GO/NO-GO red team

- Verdict: `PRODUCTION_NO_GO` / `CERTIFICATION_INVALID_FOR_CURRENT_HEAD`.
- Candidate executable SHA: `e53f8352e4c3647632ebac7877cf0a6a3bd62647`.
- PASS (local, this session): `make quality`, `go vet`, `make security`,
  `make docs-verify`, `make build`, `go test -race ./...` (1079/65).
- PASS-but-not-soak: `make overnight-smoke` 0.243s.
- FAIL: overnight 0h abort; no same-SHA CI; ancestor CI E2E failures;
  Windows/macOS native; updater trust root; authenticated Mission;
  Maestro default ASSIST; UNKNOWN quota allocation; Mission PATCH complete.
- Evidence: `DEV/validation/production-go-no-go/FINAL_REPORT.md`.

## 2026-09-13 — Overnight production certification

- FAIL / ABORT — precondition `production-finalization/FINAL_REPORT.md` missing.
- FAIL / ABORT — verdict `READY_FOR_OVERNIGHT_CERTIFICATION` not present in repo.
- FAIL / ABORT — HEAD `e53f8352e4c3647632ebac7877cf0a6a3bd62647` is not a certified overnight candidate.
- NOT RUN — campaigns A–F, 8h soak, recovery, provider failover, red team.
- Evidence: `DEV/validation/overnight-production-certification/FINAL_REPORT.md`.

## 2026-09-12 — Capability review intensive

- PASS — CLI JSON: doctor, security, control doctor, providers, profiles, version.
- PASS — `make security` (govulncheck clean).
- PASS — focused tests after fixes: web, codex, nexus, flags, update; `make lint-go` clean.
- PASS — TunnelHost ⇒ `IsTunnelActive` regression test.
- PASS — providers advertise CrossAccountResume only for Codex after honesty fix.
- PASS — follow-up: official cloudflared 2026.8.2 checksums, Darwin `.tgz` extract,
  tunnel `OnExit` disarms auth, `--effort` preserves native `-c`, AGY `-p`→DIRECT,
  supervised Codex bootstrap calls `Prepare` (CrossAccountResume seed).
  Retest: `go test ./internal/control/web ./internal/control/flags ./internal/app ./internal/control/driver ./internal/core/provider/adapters/codex -count=1`.
- SKIPPED — Claude CLI OAuth expired; Gemini CLI client unsupported.
- UNVERIFIED — interactive `/resume` cross-account and live mid-turn Codex TUI.
- DEFERRED — Claude/OpenCode Usage file-stub honesty; developer-preset credential exposures;
  TLS on private `--remote`.

## 2026-09-12 — Auth follow-up (desktop + WS query tokens)

- PASS — `/api/v1/desktop/bootstrap` requires pre-provisioned desktop bearer (Origin alone → 401).
- PASS — WS `?token`/`?session` accepted only on loopback listen; rejected on private `--remote`
  and while tunnel active (`TestWebSocketQueryTokenRejectedOnPrivateRemote`).
- PASS — frontend omits WS query token outside loopback (`shouldAttachWebSocketQueryToken`).
- PASS — `go test ./internal/control/web -count=1`; focused web vitest for agent terminal + networkHost.
- PASS — `TestVerifyCloudflaredChecksumUsesPlatformPin` (Darwin sidecar path covered).

## 2026-09-12 — Codex CrossAccountResume

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/conversation ./internal/control/flags -count=1`.
- PASS — `TestAdoptSessionFromSiblingProfile` copies sibling rollout + index
  into destination canonical sessions and marks cross-account.
- PASS — `TestPrepareSeedsSiblingSessionsForPicker` seeds sibling IDs into
  destination `session_index.jsonl` during Prepare.
- PASS — `TestForeignAccountRolloutIgnoredForQuota` and
  `TestRolloutBelongsRejectsMismatchedAccountID` keep GetUsage from inheriting
  foreign resume artifacts.
- UNVERIFIED — interactive `nexus codex:kivergmail` `/resume` listing of
  omegasistemas threads and live `nexus usage` attribution on account
  `9cba8371-...` (requires local authenticated Codex TUI).

## 2026-09-12 — Mission evidence isolation

- PASS — two persisted Missions in the same project receive only their own
  validation entries from the shared canonical stream.
- PASS — the complete hash chain is verified before the per-Mission filter;
  invalid or missing `run_id` metadata fails closed (`TestRunApplicationRejectsMissionEvidenceWithoutRunID`).
- PASS — `TestRunApplicationProjectsCompleteMissionEvidenceBeyondDefaultPage`
  projects 101 entries without inheriting the bounded 100-entry list page.

## 2026-09-12 — Nexus Codex TUI lock / flag compatibility

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/control/flags ./internal/control/driver ./internal/control/host ./internal/profile -count=1`.
- PASS — `go test ./internal/core/provider/adapters/codex -race -count=1`.
- PASS — `go vet` on the touched packages; focused `Quota|Monitor` Nexus tests.
- PASS — TUI lock blocks `runAppServerRateLimits` / `GetUsage` app-server spawn;
  rollout windows still returned; `thread_history` symlink untouched on usage path.
- PASS — Codex `-c`/`-p` pass through; `--continue` → `resume --last`;
  `--print` → `exec`; `--effort` → `-c model_reasoning_effort=...`.
- PASS — Prepare creates canonical `sessions` + `config.toml` only (no second
  `.codex/sessions` write tree).
- PASS — legacy `.codex/config.toml` is migrated before canonical defaults are
  appended; the regression test preserves the legacy model setting.
- PASS — profile preparation waits for an active per-home TUI lock before
  touching configuration/session artifacts.
- PASS — E2E profile import resolves the real host home when `HOME` points at an
  isolated provider home; explicit source override precedence is covered.
- UNVERIFIED — live authenticated `nexus codex` mid-turn interrupt / `model:
  loading` reproduction after the fix (requires interactive provider session).

## 2026-09-12 — Canonical validation evidence projection

- PASS — `go test ./internal/nexus ./internal/control/web -count=1`.
- PASS — `RunApplicationService.ValidationEvidence` reads the existing
  append-only Mission stream, verifies its hash chain before projection and
  returns explicit empty/no-chain state when no stream exists.
- PASS — the projection test closes and reopens the same SQLite store, then
  reloads the run, stream, entry and verified chain successfully.
- PASS — Codex TUI lock tests cover non-blocking ownership, app-server rate
  limit suppression, session migration safety and canonical home preparation.
- PASS — Web route `GET /api/v1/runs/{id}/validation-evidence` exposes that same
  projection; no second ledger or report store was introduced.
- PASS — Web client typed method `getRunValidationEvidence` and its transport
  test are green; frontend typecheck and build pass.
- UNVERIFIED — no authenticated Mission produced a durable production stream
  in this campaign, so this verifies the projection contract only.

## 2026-09-12 — Final closure continuation after routing/guidance integration

- PASS — `go test ./... -count=1`.
- PASS — `go test -race ./... -count=1`.
- PASS — `go vet ./...` and `git diff --check`.
- PASS — `make security`, `make build`, `make build-desktop`.
- PASS — `make quality` (aggregated formatting, lint, staticcheck, Go tests,
  frontend tests and version consistency). One existing ESLint warning at
  `web/src/nexus/AgentTerminal.tsx:218` is non-blocking.
- PASS — `make web-verify` (format, typecheck, lint, stylelint, null-array,
  tests, i18n, build, embed-sync and UI markers).
- PASS — `./nexus doctor --json` at `2026-09-12T03:02:22Z`; desktop shell is
  correctly reported as `SKIPPED`, not PASS.
- PASS — focused AGY/runtime hardening tests: expired-token quota probes fail
  closed and no-browser helper paths do not open external OAuth routes.
- PASS — task requirements now carry bounded generic `ExecutionGuidance` into
  prompt compilation; persisted routing decisions project canonical Skill refs
  and Maestro guidance refs only when an actual source/reference is present,
  including selected Skill source/version/hash/reason provenance.
- PASS — the local autopilot contract now exercises package/global recorder
  integration into `ValidationEvidenceStream`, chain verification and
  Git-SHA-bound `VERIFIED` entries. Latest ephemeral test stream was sequence 2;
  it is not a production Mission stream.
- PASS — Mission evidence captures bounded Go/OS/arch metadata and Node plus
  package-manager versions when applicable, using fixed no-shell probes with
  a two-second timeout.
- PASS — E2E harness now supports an explicit empty stable root for providers
  that reject temporary credential homes; focused root-safety tests pass.
- UNVERIFIED — Codex remained at `model: loading` even with stable root;
  AGY reached the runtime but reported `not signed in`. No real provider
  marker or authenticated Mission evidence was promoted.
- UNVERIFIED — authenticated Nexus provider Mission, durable production
  evidence stream, live failover/escalation/handoff, Windows/macOS and
  overnight acceptance remain unavailable.

## 2026-09-12 — Final closure local autopilot continuation

- PASS — `go test ./... -count=1` after the local autopilot acceptance test.
- PASS — `go test -race ./... -count=1`.
- PASS — `go vet ./...` and `git diff --check`.
- PASS — `TestLocalAutopilotContractTraversesDiscoveryRoutingSkillsAndVerification`:
  bounded discovery → `DIRECT` → Builtin Skill → task-aware model → Mission
  Runner → `COMPLETED_VERIFIED`.
- PASS — SQLite restart reload of `RoutingDecisionJSON` and durable routing
  report projection.
- PASS — persisted routing decisions now include Agent score/confidence/reason
  and selected account scope in addition to provider/profile/model.
- Release remains `NO-GO`: authenticated provider Mission, live failover/
  handoff, native Windows/macOS and overnight evidence are unavailable.

## 2026-09-11 — Codex frescor + TUI Contas & Quotas

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/core/quota ./internal/profile ./internal/tui ./internal/control/host -count=1`.
- PASS — host same-account ingest + adopt; parser aceita qualquer evento com `rate_limits.primary`; evento velho → `ESTIMATED`.
- PASS — `AvailabilityLabel` UNKNOWN → `SEM DADOS` (nunca DISPONIVEL); 5h=0 → `QUOTA ESGOTADA`.
- PASS — last-known Codex aceita bump de `IdentityVersion` (email → `chatgpt_account_id`) com mesmo `AccountID` durável.
- PASS — TUI: chrome compacto (`usageChromeLines=6`), colunas sem MODELO, STATUS `OK`/`ESGOTADA`/`SEM DADOS`/`OFF` cabem em width 80/132/160.
- PASS — `SessionHost.Stop`/`Terminate` disparam `RefreshUsageSnapshot` async para codex/agy.
- PASS — `nexus usage --json` pós-rebuild: `codex:kiver.omegasistemas` LIVE; `codex:kivergmail` ESTIMATED com last-known (rollout isolado tinha `primary:null` do Codex — sem inventar LIVE).
- NOTE — cota LIVE da gmail exige sessão isolada que grave `rate_limits.primary` (não só splash com primary null).

## 2026-09-11 — Native Skill contract transport continuation

- PASS — `go test ./internal/nexus/... -count=1` after adding canonical
  `SkillIDs` to WorkPlan, Flow, runner packages and ContextCapsule.
- PASS — builtin `coding`/`testing` resolve with Maestro unavailable; unknown
  IDs reject deterministically and legacy/generic packages can coexist.
- PASS — the actual execution freeze path accepts a real project plan with
  builtin `SkillIDs` while Maestro is offline and preserves compatibility aliases.
- PASS — `go test ./... -count=1` and `go test -race ./... -count=1` after
  correcting handoff receipts and reconciling latest Codex/TUI contracts.
- PASS — `go vet ./...`, `gofmt` cleanliness and `git diff --check`.
- PASS — `make web-verify`, `make build`, `make build-desktop`,
  `git diff --check` and repository-wide `gofmt` cleanliness.
- Release remains NO-GO for missing authenticated Mission/provider/platform
  evidence.

## 2026-09-11 — Codex quota attribution by chatgpt_account_id + isolated sessions

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/core/provider/adapters/agy ./internal/core/scheduler ./internal/profile ./internal/core/quota ./internal/tui ./internal/app`.
- PASS — e-mails no texto do rollout **não** roubam atribuição; no máximo um perfil reivindica cada rollout host.
- PASS — `FetchedAt` vem do evento `token_count`; `resets_at` no passado vira `window rolled over` e não conta como esgotado.
- PASS — AGY com `refresh_token` permanece autenticado (`Token refresh pending`) e o probe de quota pode rodar com `BROWSER=false`.
- PASS — `nexus usage --json` após rebuild: `codex:kiver.omegasistemas` LIVE com 5h/weekly alinhados ao `/status` Codex (reset weekly `05:35 on 15 Sep`); `codex:kivergmail` UNKNOWN com erro `aguardando primeira sessão isolada desta conta` (esperado até a primeira sessão isolada).
- PASS — symlinks `sessions` → `~/.codex/sessions` migrados para diretórios reais por perfil no `GetUsage`/`Prepare`.
- PASS — scheduler prefere conta Codex com capacidade sobre 5h esgotado (`TestCodexExhaustedFiveHourLosesToHealthyAccount`).

## 2026-09-11 — Codex/CSI-u no terminal Nexus

- PASS — `go test ./internal/control/driver ./internal/core/provider/adapters/codex ./internal/runtime`.
- PASS — `gofmt` nos arquivos Go alterados e `git diff --check`.
- PASS — o driver supervisionado injeta
  `CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT=1`; o adapter direto usa a mesma
  proteção.

## 2026-09-11 — Evolution corrective closure (current working tree)

- PASS — focused Go regression: `internal/nexus`, `internal/core/scheduler`,
  `internal/control/web`, `internal/control/handoff`.
- PASS — natural-language classification covers React, Go REST, Kubernetes,
  security, review and complex ecommerce; classified requirements persist on
  generated Flow steps.
- PASS — AgentMatcher no longer treats provider resource gates as Agent
  personality; generic `implementer` labels no longer erase specialized task
  roles.
- PASS — same-provider handoff preserves Agent/Project identity and reports
  `NATIVE_RESUME_UNVERIFIED` without provider-level confirmation; cross-provider
  context handoff is labeled `CONTEXT_HANDOFF`.
- PASS — `nexus run "<goal>"` dispatches through Flow, AgentMatcher,
  ResourceScheduler and MissionRunner while provider-native `nexus run <provider>`
  remains compatible.
- PASS — scheduler tie-breaking is deterministic by profile name.
- PASS — `make quality`, `make security`, `make build`, `make web-verify`,
  `go vet ./...` and `make lint-go` on the corrected tree.
- PASS — `make test-e2e` (terminal, protocol, host and web race scenarios).
- UNVERIFIED — `make docs-verify` rejects the stale visual manifest because the
  Web change is intentionally uncommitted; refreshing that manifest requires a
  committed source SHA or an explicit release commit.
- UNVERIFIED — full Go race completion, authenticated PTY/failover E2E, live
  providers, and native Windows/macOS execution; see final validation report.

## 2026-09-10 — Gates de format/lint alinhados

- PASS — `make format-check` usa o Prettier local e verifica TypeScript, CSS,
  SCSS e JSON.
- PASS — `make lint-styles` verifica CSS e SCSS.
- PASS — `make lint-frontend` usa o ESLint local; permanece apenas o warning
  preexistente de diretiva não utilizada em `NexusWorkspaceApp.tsx`.
- PASS — `.lintstagedrc.json` usa binários locais e inclui SCSS.

## 2026-09-10 — Fechamento dos últimos caminhos de quota

- PASS — AGY só persiste usage live com `AccountScope` verificável; arquivos
  legados sem identidade permanecem não atribuídos até refresh autenticado.
- PASS — `BatchFetch` e seleção de modelo AGY não leem/escrevem cache de quota
  sem escopo; runtimes carregam o escopo quando o perfil registrado o fornece.
- PASS — monitor de quota associa runtime afetado por escopo exato quando
  disponível, evitando colisões entre contas do mesmo provider/perfil.
- PASS — perfis pendentes/sem identidade mantêm apenas o identificador local
  reservado; `AccountScope.Verifiable()` só fica verdadeiro após identidade
  autenticada, impedindo cache e routing atribuíveis antes do login.
- PASS — `go test ./...` após o fechamento.

## 2026-09-10 — Review corrective pass

- PASS — `go test ./...` após as correções.
- PASS — `go test -race ./...` (processo concluído sem detector de corrida).
- PASS — `go vet ./...`.
- PASS — `npm --prefix web run typecheck`, testes Web (320/320), build e
  Prettier nos arquivos Web alterados.
- PASS — `git diff HEAD --check`.
- NOT VERIFIED — execução nativa Windows/macOS e Cloudflare real; o tunnel
  recebeu correção de lifecycle e readiness, mas não foi iniciado contra uma
  instalação real de `cloudflared` nesta sessão.

## 2026-09-10 — AccountScope e isolamento de estado

- PASS — `go test ./internal/core/model ./internal/core/quota
  ./internal/core/cooldown ./internal/profile ./internal/control/events
  ./internal/nexus`.
- PASS — regressões A/B: cache de quota e cooldown da conta A não são visíveis
  para a conta B; troca de credencial preserva `account_id` e incrementa
  `identity_version`.
- PASS — `go test -race ./...`.
- PASS — `git diff --check`.
- NOT VERIFIED — registro TUI/CLI de instalação não registrada e execução nativa
  Windows/macOS permanecem fora deste slice.

## 2026-09-10 — Registry progressivo de CLIs

- PASS — `InstallationRegistry` separa descoberta de binário do registro de
  conta e persiste `installations.json` com permissões restritas.
- PASS — `providers status --json` expõe estado de instalação/registro.
- PASS — `providers register` cria perfil isolado e permite adiar autenticação.
- PASS — testes de contrato e `go test -race` nos pacotes alterados.
- PENDENTE — integração equivalente na TUI/Web e modo `UNMANAGED_EPHEMERAL`.

## 2026-09-10 — Integração Web/TUI

- PASS — API de providers retorna estado de instalação/registro e aceita
  registro de perfil via POST.
- PASS — TUI exibe Providers separadamente e registra o item selecionado como
  `PENDING_AUTH` sem copiar credenciais.
- PASS — cliente Web tipado expõe listagem e registro.
- PASS — `go test -race ./...` e `npm --prefix web run typecheck`.
- PASS — `SaveUsageForExecution(..., managed=false)` garante que execuções
  efêmeras não persistam usage; o wiring de lançamento com HOME temporário
  permanece pendente.

## 2026-09-10 — Correção de uso compartilhado entre contas Codex

- PASS — `go test ./internal/core/provider/adapters/codex ./internal/profile
  ./internal/core/quota ./internal/nexus -count=1`.
- PASS — `go vet` nos mesmos pacotes.
- PASS — regressão cobre perfis distintos apontando por symlink para o mesmo
  `~/.codex/sessions`; apenas a conta do host recebe o rollout LIVE.
- PASS — `nexus usage --json` foi executado localmente; a resposta manteve
  `profile_id`, `account`, `status` e `source` por perfil, com fallback
  `ESTIMATED` quando não há rollout atribuível.
- PASS — `git diff --check`.
- NOT VERIFIED — execução nativa Windows/macOS; esta correção foi verificada
  no Linux.
- CONDITIONAL — `go test ./...` foi tentado, mas o sandbox bloqueou sockets
  Unix/TCP e escrita em `/home/.../checkpoints`; falhas ocorreram em testes
  de host/protocol/web/update/core lifecycle fora da área alterada. Os testes
  focados do domínio passaram.

## 2026-09-10 — Composer application boundary and context propagation

- PASS — TDD RED confirmou que Composer ignorava contexto cancelado; após a
  correção, `TestComposerOperationsRejectCanceledContext` passou cobrindo
  criação, listagem, leitura, turno, skill, finalização e resolução.
- PASS — `go test ./internal/nexus ./internal/control/web`.
- PASS — `make quality-full`: Web 62 arquivos / 315 testes; Go completo com
  race/vet; lint; build; security (`No vulnerabilities found`).
- WARNING PRE-EXISTING — ESLint mantém um warning de dependência ausente em
  `web/src/app/NexusWorkspaceApp.tsx`; nenhum erro novo.
- NOT VERIFIED — execução nativa em Windows/macOS; esta fatia foi verificada
  no Linux.
- PASS — smoke não destrutivo: `go run ./cmd/nexus --help` e `go run
  ./cmd/nexus doctor` retornaram exit 0; o diagnóstico listou os providers
  instalados e manteve estados não verificáveis como WARN/SKIPPED.

## 2026-09-10 — WorkPlan application boundary

- PASS — `go test ./internal/nexus ./internal/control/web` após extrair CRUD
  e revisões para `internal/nexus/plan_application.go`.
- PASS — `make quality-full`: suíte Web com 62 arquivos / 315 testes; testes
  Go completos incluindo race; `go vet`; lint; build; security (`No
  vulnerabilities found`).
- WARNING PRE-EXISTING — ESLint mantém um warning de dependência ausente em
  `web/src/app/NexusWorkspaceApp.tsx`; sem erros novos.
- NOT VERIFIED — execução nativa em Windows/macOS; esta etapa foi verificada
  no Linux.

## 2026-09-08 — Project Shell IPC timeout

- PASS — teste de fallback do ShellDriver com `SHELL` inválido.
- PASS — `go test ./internal/control/driver ./internal/control/web ./internal/nexus`.
- Causa confirmada nos logs: `/usr/bin/zsh` não pôde ser executado pelo
  SessionHost e o handler anterior convertia o erro em `409`.
- PENDENTE — smoke real após reiniciar o processo `nexus web`; o processo atual
  foi iniciado antes da recompilação.

## 2026-09-08 — Rail inteligente e gestão de projetos

- PASS — `cd web && bun run typecheck`.
- PASS — `cd web && bun run lint:styles` e `bun run check:styles`.
- PASS — `cd web && bun run test`: 62 arquivos / 313 testes.
- PASS — `cd web && bun run build`.
- PASS — `make quality` (inclui lint Go, testes Go e gates frontend); ESLint
  manteve somente o warning preexistente de hook em `NexusWorkspaceApp.tsx`.
- PENDENTE — screenshot/browser visual específico do rail nos cinco breakpoints;
  a implementação foi revisada estaticamente, mas nenhum harness autenticado foi
  executado nesta sessão.

## 2026-09-07 — Deep review fixes (commit 059bb5c)

- `go test -count=1 ./...` — PASS (all packages)
- `go vet ./...` — PASS
- `cd web && bun run typecheck` — PASS
- `cd web && bun run lint` — PASS (1 pre-existing warning)
- `cd web && bun run lint:styles` — PASS
- `cd web && bun run check:styles` — PASS
- `cd web && bun run test -- --run` — PASS (62 files, 313 tests)
- `make quality` — PASS
- `make security` — PASS (no vulnerabilities)
- `make build` — PASS (nexus v0.5.0-beta.33)
- `git diff --check` — PASS
- Commit: `059bb5c` (29 files, +1062/-271)

## 2026-09-07 — Preparar tarefa

- PASS — `npm run verify` (10/10 gates), `npm run build` e 311 testes Vitest.
- PASS — `go test ./...`, `go vet` afetado, typecheck, lint, format, stylelint,
  allowlist e `git diff --check`.
- PASS — validação visual/Axe específica do novo diálogo nos cinco breakpoints
  registrada na seção seguinte.

## 2026-09-07 — Visual/Axe do diálogo de preparação

- PASS — `node web/scripts/task-preparation-visual-verify.mjs`.
- PASS — screenshots em 320×568, 390×844, 768×1024, 1024×768 e 1440×900.
- PASS — ausência de overflow horizontal, navegação por teclado e Axe sem
  violações `serious`/`critical`.

## 2026-09-07 — Browser hardening e bootstrap persistido

- PASS — `timeout 180 node web/scripts/e2e-hardening-verify.mjs`; seis
  breakpoints, deep-links, Axe, Settings/ARIA e density.
- PASS — harness atualizado para resolver o bootstrap por `nexus web url`, sem
  exigir token impresso pelo servidor.

`make docs-verify` permanece **CONDITIONAL** neste worktree: o verificador
rejeita corretamente o manifesto visual ancorado em `2358618` enquanto há
alterações visuais não commitadas. A âncora só deve ser atualizada junto com um
novo SHA e screenshots correspondentes.

## 2026-09-07 — Verificação same-SHA de evidência visual

- `scripts/docs-verify.mjs` também inspeciona alterações staged e unstaged nos
  caminhos visuais; não é permitido certificar screenshots de um candidato
  imutável enquanto o worktree contém mudanças locais.
- `make docs-verify` falha de forma esperada no checkout atual, identificando
  o manifesto em `23586183...` e os paths dirty. Isso é um bloqueio honesto,
  não uma falha de teste a ser mascarada.

## 2026-09-07 — Resume-first local slice

- PASS — `overviewResumeModel.test.ts`: prioridade Needs You → ativo →
  recuperável → recente, filtragem por projeto e associação ao agente.
- PASS — `bun run test` (62 arquivos / 313 testes), `bun run typecheck`,
  `bun run lint:styles`, Prettier e build Web.
- PASS — `node web/scripts/e2e-hardening-verify.mjs` após a mudança; os seis
  breakpoints e Axe continuam verdes.
- CONDITIONAL — o painel usa runtimes/agentes já carregados; runs históricos de
  missão ainda não são propagados ao Overview e precisam de E2E de retorno.
- PASS — o hardening E2E passou a verificar explicitamente
  `[aria-labelledby="overview-resume-title"]` no Overview.

## 2026-09-07 — Composer permissões por destino

- `cd web && bun run typecheck` — PASS.
- `cd web && bun run test -- --run src/features/work` — PASS (10 arquivos,
  40 testes), incluindo Composer `MISSING` editável e destinos bloqueados.
- `cd web && bun run lint && bun run lint:styles && bun run check:styles` — PASS
  (apenas warning preexistente de dependência de hook em `NexusWorkspaceApp`).
- `cd web && bun run build` — PASS.
- `node web/scripts/build.mjs` sincronizou `web/dist` com o bundle embutido;
  `go test ./...` — PASS.
- `go test ./internal/nexus/runner -run 'Test(OvernightAcceptanceSandbox|RemediationPersistsStrategyChangeBeforeRetry|NeedsHumanContractIsStructuredAndDurable)'` — PASS; sandbox determinístico cobre plano paralelo, receipt de dependência, falha injetada, remediação e DoD global.
- `make quality` — PASS; frontend 61 arquivos/311 testes e `golangci-lint` sem issues após corrigir a mensagem de erro capitalizada em `internal/nexus/maestro.go`.

## 2026-09-07 — Correção do bootstrap de autenticação Web

- `nexus web` + `curl` em processo real — PASS: fragmento de bootstrap trocado
  por cookie; `/api/v1/session` retornou `authenticated=true` e CSRF válido.
- `go test ./internal/control/web -run 'TestServerBootstrapURLKeepsTokenInFragment|TestProvidersDocsCaptureUsesSyntheticInventory|TestServer_BootstrapAndAuth'` — PASS.
- `npm --prefix web run test -- --run src/api.test.ts` — PASS, incluindo
  consumo do fragmento e limpeza via `history.replaceState`.
- `npm --prefix web run typecheck` — PASS; `make build` — PASS.

## 2026-09-07 — Higiene de mídia documental

- `npm --prefix web run docs:capture` — PASS em fixture sintética isolada; shell
  real validado por round-trip do marcador `__NEXUS_DOCS_TERMINAL_OK__`.
- `go test ./internal/control/web -run 'TestProvidersDocsCaptureUsesSyntheticInventory|TestServer_BootstrapAndAuth'` — PASS.
- `node scripts/docs-verify.mjs` — PASS; manifesto com VIS-001…VIS-011,
  classificação `SYNTHETIC` e rejeição do cenário `real-local-bootstrap`.
- Inspeção visual — PASS para terminal, providers, workspace e Direct; sem
  projetos, paths de usuário, credenciais ou versões reais de providers.
- `git diff --check` continua apontando whitespace em bindings Wails já
  modificados por outra alteração do worktree; não foi alterado nesta tarefa.

## 2026-09-07 — Review closure local

- `npm run verify` — PASS, 10/10 gates; 61 arquivos Vitest / 310 testes.
- Browser E2E — PASS: bootstrap/API readiness, Overview → Terminais, Axe,
  deep-links, 320/390/768/1024/1280/1440px, accordion e density delta.
- `go test ./...` — PASS; `go vet ./...` — PASS.
- `make docs-verify` — PASS com manifest VIS-001…VIS-011.
- `make build-desktop-wails` — PASS com Wails Linux production + `webkit2_41`.
- `VIS-011` — PASS visual em janela Wails real, com Core Web isolado e captura
  da tela de boas-vindas funcional; captura não é mock.
- `git diff --check` — PASS.
- `go run github.com/goreleaser/goreleaser/v2@v2.18.0 release --snapshot --clean`
  — PASS; archives, DEB/RPM packages and checksums generated locally.
- Pendentes obrigatórios para GO: execução nativa Windows/macOS, snapshot
  GoReleaser, CI em SHA único após integração com `origin/main` e regeneração
  do publication report nesse SHA. Nenhum commit/push foi feito.

## 2026-09-07 — Correções iniciais de desempenho Web/Desktop

- `npm --prefix web run verify` — PASS, 10/10 gates.
- `go test ./...` — PASS.
- `go test -race` dos pacotes Desktop/Web/Terminal/Host — PASS.
- `go vet ./...` — PASS.
- `make build-desktop-wails` — PASS em produção com `webkit2_41`.
- `git diff --check` — PASS após restaurar bindings gerados automaticamente.
- Benchmark comparativo e smoke nativo de performance — PENDENTES.

## 2026-09-07 — Plano de paridade de desempenho Web/Desktop

- Artefato `.omx/plans/desktop-web-performance-parity.md` criado e inspecionado.
- Referências principais conferidas em `cmd/nexus-desktop/main.go`,
  `web/src/platform/desktopBridge.ts`, `web/src/api.ts`,
  `web/src/app/NexusWorkspaceApp.tsx`, `web/src/nexus/AgentTerminal.tsx` e
  `web/src/features/work/FlowCanvas.tsx`.
- `git diff --check -- .omx/plans/desktop-web-performance-parity.md` — PASS.
- Nenhum benchmark foi executado e nenhuma causa foi declarada confirmada.

## 2026-09-07 — Execução Luna P0/P1 e fundação de continuidade

- `go test -count=1 ./...` — PASS.
- `go vet ./...` — PASS.
- `cd web && bun run verify` — PASS, 10/10 gates; a execução anterior foi
  corrigida formatando `web/src/nexus/api.ts`.
- `go test ./internal/app ./internal/control/handoff ./internal/nexus -count=1`
  — PASS após adicionar launch supervisionado, checkpoint Maestro schema 4 e
  lease do quota monitor.
- `git diff --check` — PASS antes das alterações atuais; repetir no gate final.
- Cross-platform native runtime — PENDENTE; cross-build não substitui Windows
  ConPTY/Named Pipe nem macOS PTY/race.

## Evidência de cobertura de quota — 2026-09-07

- `go run ./cmd/nexus providers --json` — PASS; binários e capabilities foram
  enumerados sem tratar capability `Usage` como dado atual.
- `go run ./cmd/nexus profiles --json` — PASS; AGY/Codex autenticados, porém
  todos `UNKNOWN`/`NONE` sem janelas/freshness. OpenCode não autenticado e Cursor
  sem Usage. O monitor corretamente não emite alerta de percentual nesses casos.
- Teste de prioridade cross-provider — PASS; `TestResourceRecommendation_ProviderPriorityControlsFallbackOrder`.

## Terminal/continuidade — 2026-09-07

- `go test -race -count=1 ./internal/control/host ./internal/control/handoff
  ./internal/nexus ./internal/app ./internal/core/config` — PASS.
- `go vet` dos pacotes afetados e `git diff --check` — PASS.
- Cross-build de testes dos pacotes host, handoff e nexus para Windows amd64 e
  macOS arm64 — PASS como compilação; execução nativa permanece PENDENTE.
- `--supervised` e `/nexus<TAB>` ainda não têm PASS nativo Windows/macOS.

## 2026-09-06 — Plano Luna (somente documentação)

- Código de lançamento, attach, quota monitor, checkpoint/handoff e workflow
  inspecionado para fundamentar DEV/SPECS/NEXUS_TERMINAL_CONTINUITY_LUNA.md.
- Nenhuma implementação ou suíte runtime executada nesta etapa; não há novo
  resultado de CI remoto. Verificações de execução estão definidas em P0–P12.

## 2026-09-06 — CI closure pass after remote failure reproduction

- Remote run `34060911997` was inspected by job/step metadata. Frontend failed
  on format; Windows and macOS failed on their full Go test/race step. Logs are
  unavailable through the repository API without admin log permission.
- Frontend: **PASS**, `cd web && bun run verify` (10/10 gates), 306 tests,
  including build and embed synchronization.
- Go: **PASS**, `go test -count=1 ./...`, `go test -race -count=1 ./...`, and
  `go vet ./...`.
- Lint: **PASS**, `golangci-lint v2.12.2 run --timeout=5m ./...` (`0 issues`).
- Platforms: **PASS (compile evidence)**, Go test binaries and `cmd/nexus`
  compile for Windows amd64 and Darwin arm64; native execution still belongs
  to the GitHub Windows/macOS runners.
- Desktop/release: **PASS**, Wails Linux package and GoReleaser v2.18.0
  snapshot artifacts/checksums.
- Repository hygiene: **PASS**, `gofmt` and `git diff --check`.
- Generated Wails bindings: **PASS**, Prettier normalized the generated model
  file and the aggregate `make quality` gate passed afterward.
- Remote CI must be rerun on the resulting commit; no commit or push was made.

## 2026-09-06 — Monitor residente de quotas

- Go: `go test ./...` PASS; `go vet ./...` PASS; testes focados de Nexus,
  eventos, Web e App PASS.
- Web: `npm run typecheck` PASS; Vitest PASS (61 arquivos / 306 testes);
  ESLint PASS com warnings preexistentes, Stylelint PASS, `check:styles` PASS,
  build PASS.
- `git diff --check` PASS.
- Cobertura adicionada para Gemini 28% independente de Claude/GPT 100% e para
  histórico global sem filtro de runtime.

## 2026-09-06 — Workspace tabs and terminal viewport regression

- Route synchronization now keys off the actual pathname/search and does not
  reopen `/overview` on every workspace state update.
- Product tab clicks route through the application opener, preserving the URL
  and active surface together.
- Hidden xterm panels defer output, focus, and resize until `data-active=true`;
  activation flushes output and performs a guarded fit.
- Frontend verification: Prettier, TypeScript, and 292 Vitest tests passed.
- `bun run build`, `make build-desktop-wails`, installation of the Linux Wails
  artifact, and `git diff --check` passed.

## 2026-09-06 — Desktop input hardening

- Deep links now enforce bounded length, one resource ID, no fragments/users,
  and reject path/control/whitespace ambiguity.
- Linux autostart quotes the executable path using Desktop Entry `Exec` rules.
- `go test ./internal/desktop -run 'Test(AutoStart|DeepLink)'` passed.

## 2026-09-06 — Desktop capability evidence

- `DesktopBridge` now consumes the backend `GetCapabilities()` result instead
  of equating generated Wails methods with available native functionality.
- Pre-bootstrap state is conservative; capability evidence is cached after the
  authenticated desktop bootstrap.
- `cd web && bun run verify` passed all checks, including the regression test.

## 2026-09-06 — Native notification input safety

- AppleScript and PowerShell notification paths now pass title/body as
  arguments to static scripts instead of interpolating user-controlled text.
- `go test -race ./internal/control/notify -count=20` passed; Windows package
  cross-compilation also passed.

## 2026-09-06 — External URL fallback safety

- Added shared structural URL validation to Web/Desktop bridges and all
  browser-opening fallbacks.
- Unsafe schemes (`javascript`, `data`, `file`, malformed URLs) are rejected;
  the frontend regression and full `bun run verify` passed.

## 2026-09-06 — Handoff secret hygiene

- Redacted two persisted loopback bootstrap token URLs from `DEV/HANDOFF.md`
  and removed machine-local installation paths.
- Current-tree scan found no long hexadecimal bootstrap token URLs outside Git
  metadata; history was not rewritten because local user work must be preserved.
- `internal/core/security` and script tests passed.

## 2026-09-06 — Same-SHA security gate

- Added a dedicated CI `Security` job running pinned `govulncheck` with Go
  1.25.14.
- Added `Security` to the release workflow's required same-SHA job list.
- `actionlint` and release package tests passed; remote CI execution remains
  pending because no push was performed.

## 2026-09-06 — Detached update signature consumption

- Fixed the shared Update Service to consume `update-manifest.sig` beside a
  static `update-manifest.json` when no signature header is present.
- Missing, malformed, unavailable, or invalid signatures remain fail-closed.
- RED/GREEN evidence: detached-signature unit test passes normally and under
  `go test -race ./internal/update -count=5`.

## 2026-09-06 — Explicit Maestro update boundary

- Added the explicit CLI namespace `nexus maestro status|doctor|update`.
- `nexus update` remains owned by the shared Nexus Update Service and no longer
  claims to update Maestro in help/completion text.
- Added a degraded-status test for an unavailable Maestro installation and
  aligned Web settings copy with the explicit Maestro operation.

## Current closure pass — 2026-09-06

- Frontend: **PASS**, `make web-verify` 10/10 at `2026-09-06T05:20:55Z`.
- Go: **PASS**, `gofmt`, `go vet ./...`, `go test ./...`, and
  `go test -race ./...`.
- Go lint: **PASS**, `golangci-lint v2.12.2 run ./...` (`0 issues`).
- Aggregate quality: **PASS**, `PATH=/tmp/nexus-tools:$PATH make quality`.
- Stress: **PASS**, Desktop/Update/SessionHost/Workspace at `-count=20`.
- macOS fallback input handling: **PASS** by static review and Go lint; native
  execution remains pending on a macOS runner.
- Installers: **PARTIAL**, pinned release + SHA256 checks pass; signed
  manifest/keyring publication is still unavailable.
- Release workflow: **STATIC PASS**, YAML parses and publication is gated on
  named same-SHA CI jobs; runtime execution remains unverified until a new
  branch/tag CI run exists.
- Update signer byte binding: **PASS**, published manifest bytes are covered by
  the detached signature and regression-tested.
- Wails Linux packaging: **PASS**, `make build-desktop-wails` generated and
  packaged the native artifact; Windows/macOS execution remains pending.
- Native release artifact promotion: **STATIC PASS**, CI workflow uploads and
  release workflow promotes artifacts from the selected same-SHA run; runtime
  evidence remains pending.
- GoReleaser snapshot: **PASS locally** with v2.18.0; all six CLI archives,
  checksums, and Linux DEB/RPM artifacts were generated. Deprecation warnings
  remain non-blocking follow-up items.
- Desktop Linux build: **PASS**, `make build-desktop`.
- Security: **PASS**, `make security` reported `No vulnerabilities found`.
- Native Windows/macOS: **NO-GO**, CI run `34012236345` failed those jobs;
  browser/Desktop downstream jobs were skipped. See
  [`FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md`](validation/FINAL_PLATFORM_RELEASE_DESKTOP_REPORT.md).

## 2026-09-05 — Implementação Completa Web + Desktop Multiplataforma

- `go test ./...` — PASS (100% de sucesso nos pacotes `internal/app`, `internal/desktop`, `internal/update`, `internal/control/web`, `internal/doctor`, `internal/release`, etc.).
- `go vet ./...` — PASS.
- Frontend Quality (`bun run quality`):
  - `check:styles` — PASS (allowlist SCSS Modules estrita).
  - `format:check` — PASS.
  - `lint` — PASS (0 erros).
  - `lint:styles` — PASS.
  - `typecheck` — PASS (`tsc --noEmit` 0 erros).
  - `vitest run` — PASS (58 arquivos de teste, 280 testes unitários).
- Frontend Build (`bun run build`):
  - Geração de chunks ESM sob demanda em `web/dist` — PASS.
  - Sincronização e verificação de integridade do bundle embutido no Core e no Desktop (`web.EmbeddedDistFS()`) — PASS.
- Desktop Shell & Capabilities:
  - Wails v2 estável (`v2.15.0`) — build e testes unitários do bridge nativo (`internal/desktop`) — PASS.
  - Deep links (`nexus://`), window management e PlatformBridge — PASS.
- Maestro Desacoplado:
  - Scripts de instalação (`install.sh` e `install.ps1`) com opt-in explícito (`--with-maestro` / `-WithMaestro`) verificados — PASS.
  - Modo degradado `MAESTRO_DEGRADED` sem falhar o produto — PASS.

- `bun install --frozen-lockfile` — PASS com Bun 1.3.9 após regeneração do lockfile.
- Frontend format/typecheck/lint/stylelint — PASS; lint com 0 erros e 36 warnings preexistentes.
- Frontend Vitest — PASS: 51 arquivos, 253 testes.
- golangci-lint `v2.12.2 config verify` — PASS; execução completa reproduz 59 achados existentes, sem supressão.
- `go test ./...` — PASS após ExecutableResolver.
- `go test -race ./internal/control/terminal ./internal/control/protocol ./internal/control/host` — PASS.
- Windows cross-compile `go test -c` para terminal, launcher, host e protocol — PASS; execução nativa ainda pendente.

Estado: runtime/release continuam NO-GO até lint debt, build/embed gate e evidência nativa Windows/macOS serem fechados.

## 2026-09-05 — Path identity and credential capability increment

- `go test ./internal/core/config ./internal/nexus/store` — PASS após migração `0012_path_identity.sql`.
- Cross-compile de config/runtime para Windows e Darwin — PASS.
- `go test ./...` — PASS.
- `go vet ./...` — PASS.
- `git diff --check` — PASS.

Limitações mantidas: identidade nativa por handle no Windows, execução nativa Windows/macOS e bundle do `nexus doctor` ainda não foram validados.

## 2026-09-05 — Doctor CLI and diagnostic bundle

- `nexus doctor --json` com diretórios temporários isolados — PASS.
- `nexus doctor --bundle` — PASS; ZIP contém somente `report.json` e `MANIFEST.txt`.
- `go test ./internal/doctor ./internal/app` — PASS.
- `go test ./...` — PASS.

O bundle foi validado por allowlist; execução nativa Windows/macOS e reuso na Web permanecem pendentes.

## 2026-09-04 — Topbar Overlap Refactor, Responsive Hardening & Accordion Theme Quality Gates

- `make web-verify` — PASS (8/8 gates verdes: typecheck, lint, null-arrays, vitest [51 files, 249 tests], i18n, build, embed-sync, ui-markers).
- `make build` — PASS (binário `nexus v0.5.0-beta.33` gerado e instalado com bundle web embutido).
- `npm --prefix web run test:e2e-hardening` — PASS (100% de sucesso nos 6 viewports):
  - `320x568` (mobile-se): botão Terminal desobstruído (`elementFromPoint` verificado sem overlap).
  - `390x844` (mobile-iphone): botão Terminal desobstruído.
  - `768x1024` (tablet): botão Terminal desobstruído.
  - `1024x768` (laptop-compact): layout estável, ações secundárias colapsadas.
  - `1280x800` (laptop-critical): terminal acessível em 1º lugar, sem clipping.
  - `1440x900` (desktop): renderização completa sem colisão.
  - Validação WAI-ARIA do Accordion de Temas com paletas visuais e medição real do delta de densidade (Comfortable 497px vs Compact 481px).


- `make web-verify` — PASS (typecheck, lint, null-arrays, vitest (244 tests), i18n, build, embed-sync e ui-markers).
- `make build` — PASS (binário `nexus v0.5.0-beta.33` compilado com assets embutidos atualizados).
- Auditoria Playwright multi-viewport (1440x900, 1024x768, 768x800, 480x800) em `/usr/bin/google-chrome`:
  - Visualização de screenshots reais renderizados:
    - `verify_1440x900_desktop.png`: layout limpo com rail lateral, topbar sem colisão, janelas em cascata e radar visível.
    - `verify_1024x768_compact.png`: botões redundantes de criação escondidos, hierarquia de janelas com z-index correto.
    - `verify_768x800_tablet.png`: rail colapsado em drawer/overlay, topbar compacto com badges e menus sem quebra.
    - `verify_480x800_mobile.png`: zero sobreposição/encavalamento de controles; comandos e labels longos suprimidos graciosamente, radar e switcher de idiomas alinhados no topo sem overflow horizontal.
- Z-Index verificado: radar e popovers operando em camada superior (`--nx-z-popover: 4000`) a janelas focadas (`500`).
- Primitivas `<Select>` e `<Input>` do design system adotadas em 100% dos modais e superfícies auditadas, eliminando selects HTML crus não estilizados.

## 2026-09-04 — Terminal recover após reboot/serviço

- `go test -count=1 ./internal/control/registry ./internal/control/web` — PASS.
- `make web-verify` — PASS (typecheck, lint, null-arrays, Vitest, i18n,
  build, embed-sync e ui-markers). Relatório:
  `DEV/validation/FRONTEND_LATEST.md`.
- Reprodução manual restante: reiniciar `nexus web` (ou a máquina) com uma
  janela de terminal de agente aberta e confirmar overlay **Iniciar /
  Recuperar Agente** em vez de xterm preto.

## 2026-09-04 — Baseline atual e Composer skills

- `make web-verify` — PASS (typecheck, lint, null-arrays, Vitest, i18n,
  build, embed-sync e UI markers).
- `go test -count=1 ./...` — PASS.
- `go vet ./...` — PASS.
- `git diff --check` — PASS.
- CI frontend passou a fixar Bun `1.3.9` e o `packageManager` do workspace foi
  declarado como `bun@1.3.9`.
- Composer agora expõe e persiste aceitar/dispensar de Maestro skills e envia
  as skills aceitas na finalização; Readiness, Unknowns e Assumptions são
  visíveis.

## Identidade visual e assets de marca — 2026-09-04

- `make web-verify` — PASS (typecheck, lint, null-arrays, vitest, i18n, build, embed-sync e ui-markers).
- `go test ./...` — PASS (todos os pacotes Go testados e passando).
- `make build` — PASS; binário `nexus v0.5.0-beta.33` gerado e instalado com novos assets embutidos em `internal/control/web/dist`.
- Validação visual: extração do master em alta resolução com desmatting transparente, verificação de contraste em fundo escuro (`#080a0f`) e fundo claro, geração do pacote favicon e renderização no rail e hero.

## Controles de terminais e resumo de Agentes — 2026-09-03

- `make web-verify` — PASS (typecheck, lint, null-arrays, testes, i18n, build,
  embed-sync e ui-markers).
- `make build` — PASS; instalou `nexus v0.5.0-beta.33`.
- Cobertura funcional implementada: troca de modo com erro visível e rebinding,
  feedback de lease, fechamento com escolha aba/runtime, confirmação de Project
  Shell e badge operacional clicável com foco/recuperação.

## Bootstrap de contexto durável — 2026-09-03

- `go test ./internal/nexus ./internal/control/web` — PASS.
- `npm --prefix web run typecheck` — PASS.
- `npm --prefix web test -- --run` — PASS (42 arquivos, 180 testes).
- `make build` — PASS; instalou `nexus v0.5.0-beta.33`.
- A criação é explícita, gera `AGENTS.md` base e nunca sobrescreve contexto
  existente.

## Recuperação de terminal desconectado — 2026-09-03

- Web: `npm --prefix web run typecheck` — PASS.
- Web: `npm --prefix web test -- --run` — PASS (42 arquivos, 175 testes).
- Build: `make build` — PASS; instalou `nexus v0.5.0-beta.33`.
- Recover/Start agora vincula diretamente a geração retornada; o terminal também
  oferece fechamento explícito da aba.

## Nova sessão de IA — 2026-09-03

- Web: `npm --prefix web test -- --run` — PASS (40 arquivos, 157 testes).
- Web: `npm --prefix web run typecheck` — PASS.
- Build: `make build` — PASS; instalou `nexus v0.5.0-beta.33`.
- O launcher agora é disparado depois que a superfície Composer está montada.

## TERM=dumb em CLIs interativos — 2026-09-03

- `go test ./internal/runtime ./internal/control/terminal` — valida substituição
  de `TERM=dumb` sem mutar o ambiente original e preservação de terminais normais.
- A correção cobre execução direta e runtimes supervisionados com PTY/ConPTY.

## AGY quota por grupo — 2026-09-03

- `go test ./internal/core/quota ./internal/app ./internal/core/scheduler ./internal/nexus` — PASS
- Conta AGY com Gemini disponível e Claude/GPT esgotado agora fica `INDISPONIVEL`/
  `QUOTA ESGOTADA` no nível da conta.
- Conta AGY só fica `DISPONIVEL` quando os dois grupos têm quota.

## AGY keyring e modelo — 2026-09-03

O processo AGY recebe `GNOME_KEYRING_CONTROL` do Secret Service privado, e o
arquivo `antigravity-cli/settings.json` não é mais sobrescrito pelo host a cada
execução. Testes Go focados passaram e `make build` instalou `nexus v0.5.0-beta.15`.

## Radar falso + terminal “connecting” — 2026-09-03

- Go: `go test ./internal/control/host` — PASS
  (lista numerada sozinha não vira pergunta; lixo `\uFFFD`/box-drawing rejeitado;
  `Select an option` + `1. …` continua `choice`)
- Web: `make web-verify` → [`DEV/validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md) — PASS
  (radar colapsa mensagens iguais; honesty usa `sanitizeAttentionText`;
  WS `?runtime_id=`; erro fatal para reconnect; taskbar “agentes degradados”)
- Runtime: `make install` + reinício de `nexus web` + `pkill` dos `__control-host` velhos
- Browser smoke: Radar **ok** / “Nenhum terminal ativo” sem hosts;
  AgentTerminal mostra `error` + “Use Recover/Start…” em vez de `connecting` eterno

## Organizador de terminais + radar — 2026-09-03

Cobertura automatizada do P0:

- Go: `go test ./internal/control/host ./internal/control/notify` — PASS
  (shortcuts/`[y/n]` isolado silenciam; yn real + free_text; clear on `thinking...`;
  OS notify só com evidência + dedupe de fingerprint)
- Web: `make web-verify` → [`DEV/validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md) — PASS
  (título sem emoji/`(N ❓)`; fingerprints de mensagem iguais colapsam; Sim/Não só se `yn`;
  radar/focus model; i18n parity)
- Embed sync: `web/dist` ≡ `internal/control/web/dist` após o gate

Caminho manual ainda necessário no ambiente do operador (depende de `make build` +
reinício de `nexus web`): dois Projetos, pergunta yn real vs chrome de help, e
notificação de SO com a aba fechada.

## `null.forEach` no Flow — 2026-09-03

O bundle servido foi regenerado a partir de `flowModel.ts`, que trata
`plan.phases` e `phase.packages` não-array como coleções vazias. A regressão
para payloads nulos passou em `yarn test src/features/work/flowModel.test.ts`
(9/9), e `yarn build` atualizou `internal/control/web/dist`.

Nesta execução, `yarn typecheck` também passou.

## Normalização de WorkPlan — 2026-09-03

`web/src/nexus/workPlan.ts` passou a ser a fronteira de normalização para
todos os endpoints que retornam WorkPlan, cobrindo planos e fases nulos antes
do acesso da UI. Verificação atual: testes focados (19/19), `yarn lint`,
`yarn typecheck` e `yarn build` — PASS.

## Reanexação de terminal — 2026-09-03

Após Recover/Start, o terminal não reutiliza o `runtime_id` histórico da
superfície. Ele reanexa pelo Agente e acompanha a nova geração. Testes focados
(14/14), typecheck e build passaram; lint passou com 7 warnings preexistentes.

## Tabela TUI de uso — 2026-09-02

`nexus usage` usa a tabela navegável do Charm Bubbles. O filtro cobre
identidade e grupo de modelo; há teste unitário para ambas as buscas.

## Banner de atenção — 2026-09-02

O banner de intermediação não participa mais da linha expansível da grade do
Workspace OS. Ele é exibido como uma faixa compacta sob o cabeçalho, com
truncamento, controles padronizados e quebra responsiva. A heurística ignora
`? for shortcuts`, evitando o aviso falso visto no terminal.

Validação: `go test ./...`, `go vet ./...`, `yarn --cwd web typecheck`,
`yarn --cwd web lint`, `yarn --cwd web test` (31 arquivos/111 testes),
`git diff --check` e `make build` — todos PASS.

## Notificações transitórias — 2026-09-02

`TASK_COMPLETED` e `ERROR` são convertidos em toasts próprios, enquanto
`QUESTION` e `APPROVAL` continuam no componente de resposta. O modelo possui
cobertura para conclusão e exclusão de eventos interativos.

## Uso/quota — 2026-09-02

O fluxo de sessão Web mantém rotação automática antes da expiração absoluta,
propaga novo CSRF e orienta a recuperação via novo Bootstrap quando a sessão
expira. Testes Go e frontend passaram.

CLI e dashboard de recursos foram verificados com tabela/grid por modelo,
layout responsivo e status independente por grupo. Typecheck, lint, 31 arquivos
de testes (111 testes), `make build` e `git diff --check` passaram.

`nexus usage` agora preserva a capacidade por grupo de modelo e por janela,
exibindo percentual restante e reset 5h/semanal. A validação passou em
`go test ./internal/app ./internal/core/quota ./internal/profile`, `go vet ./...`
e `make build` (`v0.5.0-beta.9`). Cache vencido e snapshot de outra identidade
agora são rejeitados antes da exibição/seleção.
Grupos independentes também têm disponibilidade independente; no estado atual
`kiveromegasistemas` aparece como `DISPONIVEL` para Gemini e `INDISPONIVEL` para
Claude/GPT.

## Review update — 2026-09-02

The complete current diff was reviewed against `HEAD` (`c968852`). The current
worktree is **PASS AFTER CORRECTIONS**: `go build ./cmd/nexus`, `go test ./...`,
`go vet ./...`, frontend lint, tests and typecheck pass. The detailed findings
and exact file locations are recorded in
[`DEV/CODE_REVIEW.md`](CODE_REVIEW.md).

The corrections and latest evidence supersede the historical validation
snapshots below.

## AGY quota semantics — 2026-09-02

Legacy AGY quota files expose `percent_left` as consumed percentage. The
adapter and cache reader now normalize it to remaining percentage before
selection or display. Coverage includes boundary and fractional values.

## Status: ALL 7 PENDING ISSUES RESOLVED

### Build & Test
- `go build ./...` — clean
- `go test -race ./...` — ALL PASS
- `go vet ./...` — clean
- `npx tsc --noEmit` — clean
- `npx vitest run` — 44/44 tests PASS
- `make build` — PASS (v0.4.6)

### Pending Issues Resolution
| # | Issue | Status | Key Changes |
|---|-------|--------|-------------|
| 1 | Resources facade | ✅ | `ListResources()` real discovery, `AllocateResource()` persistence, `ResourcePicker` UI |
| 2 | Maestro synthetic state | ✅ | Honest degraded fallback, no hardcoded capabilities/recommendations |
| 3 | Update simulated | ✅ | Returns `501 Not Implemented` |
| 4 | Agent start without provider | ✅ | `ResolveStartParams()` + `REQUIRED_RESOURCE_SELECTION` flow |
| 5 | Config not reaching runtime | ✅ | `LaunchOptions` extended, full `AgentConfig` propagation |
| 6 | Terminal continuity | ✅ | `AgentTerminalBroker` integration, `runtime_changed` frames |
| 7 | Missions scaffold | ✅ | Kept as-is, documented as future work |

### Business Rules Location
All business logic lives in `internal/nexus/` (service layer). Web and TUI consume the same API endpoints. No resource selection, health checking, config propagation, or Maestro degradation logic exists in the frontend.

### Files Modified (this session)
- `internal/nexus/resource_discovery.go` (new)
- `internal/nexus/maestro.go`
- `internal/nexus/nexus.go`
- `internal/nexus/scheduler.go`
- `internal/nexus/store/agents.go`
- `internal/control/launcher/launcher.go`
- `internal/control/web/handlers_nexus.go`
- `internal/control/web/handler_terminal.go`
- `internal/control/web/broker.go`
- `internal/control/web/server.go`
- `web/src/features/agents/AgentsSurface.tsx`
- `web/src/nexus/ResourcePicker.tsx`
- `web/src/nexus/api.ts`

### Documentation Updated
- `DEV/NEXUS_V1_ARCHITECTURE.md` — resource discovery + terminal broker
- `DEV/NEXUS_V1_RESOURCE_SCHEDULER.md` — marked implemented with flow
- `DEV/NEXUS_V1_MAESTRO_INTEGRATION.md` — honest degraded fallback
- `DEV/NEXUS_V1_AGENT_MODEL.md` — updated start flow + config propagation
- `DEV/WORKLOG.md` — session entry with all changes

### Not Yet Started
- Cross-provider scoring with continuity, cooldown, and rate-limit risk
- Project-level and global policy hierarchy
- Account Handoff and Context Handoff recommendations
- Full Maestro CLI contract (`feat/nexus-contracts-v1`)
- Mission execution/orchestration (scaffold only)
- Full E2E verification with live providers
- Performance benchmarks

## Review final — 2026-09-04

- `go test -count=1 ./...`: PASS
- `go vet ./...`: PASS
- `go test -race -count=1 ./internal/control/... ./internal/nexus/... ./internal/core/security/... ./internal/core/session/...`: PASS
- `make web-verify`: PASS (typecheck, lint, null-arrays, 188 testes, build, embed-sync, ui-markers)
- `npm audit --offline --omit=dev --audit-level=high`: 0 vulnerabilidades
- Cross-build `linux/amd64`, `darwin/amd64`, `windows/amd64`: PASS
- `git diff --check`: PASS
- Smoke final `/api/v1/health`: HTTP 200; bootstrap: HTTP 302; servidor ativo em `127.0.0.1:3000`

Parecer e limitações: [`DEV/validation/CURRENT_CODE_REVIEW.md`](validation/CURRENT_CODE_REVIEW.md).

## TUI Usage selection — 2026-09-06

- `go test ./internal/tui -run 'TestUnifiedUsage(RestoresSelectionAfterEmptyFilter|ModeTogglingAndFlags|EscAndQQuit)$' -count=20` — PASS.
- `go test -race ./internal/tui` — PASS.
- `Enter` após filtro seleciona a linha destacada; cursor é restaurado após
  resultado vazio; `Esc` mantém comportamento de apenas fechar o filtro.

## Frontend embedded identity — 2026-09-06

- `cd web && bun run verify` — PASS (10/10 gates, incluindo igualdade entre
  `web/dist` e `internal/control/web/embedded`).
- `.github/workflows/ci.yml` agora usa esse verificador no job Frontend, não um
  teste de mera existência de arquivos.
- `go test ./internal/release -run 'TestInstaller' -count=1` — PASS; a política
  de caminho `IAPro Nexus` com preservação do diretório legado foi verificada.
- Actionlint/YAML dos workflows — PASS; jobs Windows/macOS preservam logs de
  falha em artifacts sem mascarar o resultado.
- O gate Windows inclui parsing explícito do `install.ps1`; execução local não é
  possível porque `pwsh` não está instalado neste runner Linux.
- Cross-build CLI Linux amd64/Windows amd64/macOS arm64 e Desktop Windows
  amd64 — PASS como compilação cruzada apenas; não substitui execução nativa.
- Listener failure: teste Linux 20x PASS; pacote `internal/control/host`
  compilou para Windows amd64 PASS; execução Windows ainda requer runner nativo.
- ConPTY interactive fixture: pacote Linux e compilação Windows amd64 PASS;
  execução nativa ainda pendente.
- FSMkdir: testes FS focados 20x PASS e compilação Windows amd64 do pacote Web
  PASS; execução nativa ainda pendente.
- IPC readiness: Host/Protocol 20x PASS e race PASS sem sleeps de prontidão;
  execução nativa Unix socket/Named Pipe continua dependente dos runners.
- HTTP readiness: bootstrap/session/restart focados 3x PASS e race PASS sem
  sleeps fixos.
- Web E2E/API/túnel completos: PASS normal e com race após remover waits de
  startup artificiais.
- PTY Unix: 20x PASS e race PASS com leitura baseada em output observado, sem
  sleep de readiness.
- Pacotes terminal/host/protocol/web: testes normais, race e golangci-lint PASS;
  `make quality PATH=/tmp/nexus-tools:$PATH` PASS.
- Revalidação final: `go test -race ./...`, `make security`, GoReleaser v2.18.0
  snapshot e `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7`
  passaram.
- Browser E2E real: PASS após corrigir locator case-sensitive de Settings;
  Playwright, Axe, deep-links, breakpoints e screenshots passaram em três
  execuções consecutivas; o script passou pelo Prettier.
- Fixtures Host/QA Windows não usam mais `cat`; usam `cmd.exe /D /Q /C more`.
  Linux normal/race e cross-compilação Windows/macOS dos pacotes afetados
  passaram. Wine/ConPTY permanece explicitamente não nativo.
- Update Service: testes 20x e race PASS; `HOME` de teste isolado e nenhum
  receipt gerado no checkout.
- Doctor: probes read-only e estados `WARN`/`SKIPPED` para capacidades nativas
  sem evidência; pacote passou 20x e race 5x.
- Registry: invalidação cross-process agora usa `ModTime` + tamanho + fingerprint
  SHA-256; teste de concorrência passou 50x, race 10x e `go test ./...` passou.

## Maestro catalog and skill context — 2026-09-07

- `go test ./internal/nexus -run 'TestMaestro|TestFindOrquestrador' -count=1` —
  PASS after global/profile catalog merge and prompt provenance changes.
- Frontend typecheck, lint, stylelint, focused tests and build — PASS after
  replacing command copy with complete skill context.
- Real browser smoke against the rebuilt server — PASS: `53 de 53 disponíveis`,
  53 cards and 53 usage-context panels; `skill-database-migrations` exposed
  its `SKILL.md` prompt.
- This does not replace native macOS/Windows execution; those release gates
  remain pending where recorded above.

## Open Design typography and visual QA — 2026-09-07

- Open Design audit identified undersized semantic typography and one dark-theme
  contrast failure in the shell.
- `node web/scripts/maestro-visual-verify.mjs` — PASS after correction:
  screenshots at five required viewports, no horizontal overflow, search
  interaction, Axe with zero serious/critical violations, and no page errors.
- `bun run format:check`, `bun run typecheck`, `bun run lint:styles`, focused
  theme/Maestro tests (14/14), frontend build, focused Go tests, `go vet` and
  `git diff --check` — PASS. Existing ESLint warnings remain warnings only.
- Typography tokens use `rem`; visual verification was repeated after the
  migration and the rebuilt server health endpoint returned `status: ok`.
- `FontScalePicker` no toolbar foi validado por typecheck/build e usa o mesmo
  `ThemeProvider` da tela Configurações, sem estado paralelo.
- Tipografia completa: nenhuma declaração `font-size`, `font` ou `fontSize` em
  px permanece em `web/src`; o servidor recompilado respondeu `status: ok`.

## Terminal helper do Project Shell — 2026-09-07

- `make build` — PASS; frontend recompilado e binário Nexus gerado.
- Verificação do bundle — PASS; `xterm-helper-textarea` presente em
  `web/dist/bundle.css` e CSS embutido sincronizado.
- Smoke Playwright contra o servidor real — PASS; os helpers do xterm ficaram
  invisíveis, não houve sequência de `W` sobre o terminal e não houve erro de
  página.
- O problema era visual do CSS ausente, não saída indevida do PTY.

## Tokens de espaçamento — 2026-09-07

- `bun run build` — PASS.
- `bun run typecheck` — PASS.
- `bun run lint:styles` — PASS.
- Bundle final contém `--nx-spacing-1`, `--nx-spacing-8`,
  `--nx-spacing-16` e aliases `--nx-space-4` — PASS.
- `make build` — PASS; bundle embutido regenerado.
- Instalação/execução final — PASS; `LOCAL_BIN=/home/desenvolvedor/.local/bin
  make install-local`, CSS HTTP contém `--nx-spacing-4: 16px` e o navegador
  calcula `16px` no `:root`.
- `make -n install` — PASS; fluxo local padrão e fluxo `DESTDIR` ficam explícitos
  no Makefile.

## Cota oficial do Codex via app-server — 2026-09-11

- Fonte primária nova: `codex app-server --stdio` + `account/rateLimits/read`
  (`internal/core/provider/adapters/codex/app_server_usage.go`). Rollouts e
  last-known continuam como fallback honesto, nessa ordem.
- `go test ./... -count=1` — PASS (suíte completa).
- `go test -race -count=1` em `internal/profile`, `internal/core/quota`,
  `internal/core/provider/adapters/codex`, `internal/tui`, `internal/nexus` — PASS.
- `go vet ./...`, `gofmt -l internal/` e `git diff --check` — PASS.
- Validação real com as duas contas, sem abrir sessão e sem enviar prompt:
  - `codex:kivergmail` — `RATE_LIMITED` / `OFFICIAL_API`; 5h em `0%` restante
    (reset 21:55) e semanal em `56%` restante (reset 08:37 de 15 Sep). Confere
    com o splash oficial do Codex CLI.
  - `codex:kiver.omegasistemas` — `LIVE` / `OFFICIAL_API`; 5h `98%` e semanal
    `58%`, com `accountId` próprio. As duas contas não se contaminam.
- `RATE_LIMITED` passou a ser leitura confiável: `quota.Engine.Trustworthy`,
  as barras e o monitor tratam o bloqueio informado pelo provedor como dado
  exato, em vez de descartá-lo para `SEM DADOS`.
- TTL de 60s vale também entre processos para leituras `OFFICIAL_API`; uma
  segunda chamada de `nexus usage` dentro da janela reaproveita o cache e não
  sobe outro app-server. Cache derivado de rollout nunca é reaproveitado.
- Identidade: a sonda exige `chatgpt_account_id` verificável, confere o
  `accountId` da resposta e o `codexHome` ecoado pelo `initialize`. Qualquer
  divergência descarta o payload. Desligável por `NEXUS_CODEX_APP_SERVER=0`.
- Controle estruturado (aprovações/eventos) segue diferido — ver
  [`DEV/AI_CONTROL_DEFERRED.md`](AI_CONTROL_DEFERRED.md), item 6.

## 2026-09-12 — automatic delegation interactive Lead closure

- PASS — focused `go test ./internal/app ./internal/control/launcher ./internal/nexus -count=1`.
- PASS — `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`
  and `git diff --check`.
- PASS — `make build` and `make build-desktop`; both retain explicit Nexus
  build metadata while disabling only redundant Go VCS stamping.
- PASS — canonical workspace lookup/registration, persistent Lead binding and
  runtime generation lifecycle are covered locally; launcher labels preserve
  the interactive policy metadata.
- UNVERIFIED — authenticated `nexus agy`/Codex provider session, real provider
  prompts and a durable authenticated Mission evidence stream remain unavailable.
- UNVERIFIED — the attempted supervised Codex smoke timed out during IPC
  handshake; no runtime remained registered afterward.

## Fechamento beta.10 Windows — 2026-10-16

Ciclo de correção e certificação da instalação local (branch
`fix/release-beta10-windows-closure-20261014`), fechado sem alterar
`release/v1.0.0` e sem merge/tag/publicação.

- Picker de diretório (`6c14b57e`): `GET /api/v1/fs/browse` sem path devolve
  200 com default seguro (`defaultBrowsePath`: cwd só quando dentro de uma
  raiz permitida, senão home) e bookmarks expõem apenas raízes permitidas —
  sem bookmark apontando para `/`. Coberto por e2e RED→GREEN e 15/15 unit.
- HUD vinculado ao core (`baf91839` + manifesto `37434c6d`): spawn com
  `--linked` quando o core sobe; opt-out `NEXUS_HUD_SPAWN=0`; hard-kill do
  core mata o HUD (watchdog). Verificado ao vivo: HUD parent = PID do core.
- Fechamento de terminal ConPTY com bound de 2s no `Wait()` do rPipe
  (`aa854590`).
- PS 5.1 + stderr nativo: banner do `bun run` vira NativeCommandError
  terminante sob `ErrorActionPreference=Stop`. Corrigido no script da cadeia
  (`36445f10`, `Invoke-NativeLogged`) e depois no próprio instalador
  (`996659f4`, `Invoke-NativeBuild`): `bun install`, `bun run build`,
  `go build` (cache frio imprime `go: downloading` em stderr) e
  `git fetch/checkout`. `$LASTEXITCODE` permanece a única fonte de verdade
  dos checks (provado empiricamente em PS 5.1).
- Flake da cadeia #5 (`36445f10`, fase Go tests):
  `TestEvidenceBundleRequiresAndProjectsCanonicalProof` degradou identidade
  sob saturação. Causa-raiz provada por digest: `IdentityDigest:4eab7424…` =
  `sha256("git\0INCOMPLETE\0git identity command timed out")` =
  `incompleteGitIdentity()` (deadline de 5s do child git, comportamento
  conservador intencional do produto). Correção apenas no teste (`30e2e6f3`):
  retry tolerante exclusivamente ao digest constante de deadline-loss, teto
  5 tentativas + backoff, qualquer outra incompletude falha imediato.
  Repro controlada com git falso (`degrade-once` → PASS com retry;
  `degrade-always` → FAIL estrito com digest idêntico).
- Cadeia local de homologação #6 (`38968218`) e #7 (`996659f4`): **PASS
  18/18** — veredito `PASS automated Windows beta homologation:
  1.0.0-beta.10 @ 996659f4134f2f3f26007c69afd387b2d6905f77` (~18 min cada).
  Evidência: `DEV/validation/local-beta/1.0.0-beta.10/996659f4134f2f3f26007c69afd387b2d6905f77/windows-AMD64/`.
- Reinstalação ao vivo pós-cadeia: `install.ps1 -BuildFromSource -Yes` com
  EAP=Stop completa (exit 0); Desktop e HUD copiados dos builds da cadeia;
  core relançado (`web --no-open`) spawnou o HUD `--linked` (parent = core).
- Live check da API contra o core instalado: bootstrap 200 com cookies
  `ai_control_session`/`ai_control_device`; `GET /api/v1/fs/browse` sem path
  200; `current_path` = diretório de lançamento (cwd dentro de raiz
  permitida, conforme contrato de `defaultBrowsePath`); bookmarks
  `Home → C:\Users\kiver`, `Documents → C:\Users\kiver\Documents`, nenhum
  apontando para raiz do sistema.
- Passagem headless somente leitura sobre o `MANUAL_CHECKS.md` executada
  contra o core vivo (2026-10-16, nenhuma mutação): `GET /api/v1/providers`,
  `/api/v1/execution-roles`, `/api/v1/hud`, `/api/v1/voice/settings`,
  `/api/v1/capacity/status`, `/api/v1/cost/summary` e `/api/v1/projects`
  todos 200. Item 16 confirmado ao vivo (opencode `INSTALLED_UNREGISTERED`
  visível sem perfil); item 15 com OpenCode listado como provider próprio
  independente do AGY; roles build/investigate/review presentes (item 17);
  voz N/A (`enabled=false`, sem chave Groq — item 22). Itens 1 e 10-12
  cobertos pelos testes da cadeia #7 (CSI/setas byte-level, delegation
  fail-closed, budget gate, isolamento). Detalhe:
  `%TEMP%\opencode\MANUAL_CHECKS_HEADLESS_RESULT.md`.
- Ícones do tray fixados na área de notificação (fora do overflow):
  `IsPromoted=1` em `HKCU:\Control Panel\NotifyIconSettings` para os dois
  binários de produção (`nexus-desktop.exe` e `nexus-hud.exe`), mecanismo
  real do Windows; a promoção se concretiza quando o ícone volta a se
  registrar (relançamento do HUD/Desktop ou novo login).
- Autostart do HUD ativado no formato exato do toggle do produto
  (Wails v3 `Autostart.Enable()`): valor `nexus-hud` sob
  `HKCU:\Software\Microsoft\Windows\CurrentVersion\Run` apontando para
  `nexus-hud.exe` entre aspas, sem argumentos (igual ao `Enable()`
  padrão). O toggle nativo do HUD reconhece a entrada e pode desligá-la.
- UNVERIFIED (por design, interativo) — itens do `MANUAL_CHECKS.md` que
  exigem sessão nativa real (prompt em provider, admissão de Mission com
  Agent pinado, Certify/enforcement) permanecem a cargo do operador.

## HUD nativo — arraste e presets de tamanho — 2026-10-21

Worktree `C:\Dev\nexus-hud-validation`, base `f6098b8c684a97bf7e8e56804109ca491c78cfa2`, sem commit/push.

- **PASS** — `tsc --noEmit`, ESLint, Prettier e Stylelint nos arquivos do HUD; Vitest direcionado 37/37; `go test -count=1 ./internal/desktop -run HUD`; build Web e build Go do `nexus-hud`.
- **PASS** — UI Automation em Windows 11 `10.0.26200`, binário isolado `nexus-hud-diag-v3` (execução anterior): presets medidos `220×60 → 334×496 → 380×560 → 440×640 → 334×496 → 220×60`; nomes acessíveis `Compacto`, `Confortável`, `Grande` e `Arrastar HUD. Use as setas para mover.`; arraste real `-144,-108` px; seta direita `+16` px; `Shift`+seta `+64` px.
- **PASS** — persistência: após escolher `comfortable` e recolher, o estado ficou `260×70`; após reiniciar o HUD no mesmo diretório de dados, abriu em `260×70` na posição persistida (`1516,866`).
- **PASS** — primeiro paint: com estado limpo, o HUD abriu em `220×60` no canto inferior direito, sem o antigo mínimo `190×48`.
- **PASS** — rebuild final (`nexus-hud-diag-v4.exe`, SHA-256 `ae64668c16eedbacabc9f91eb0b05a8a3cb6f28f37800ed59a961e4aa5d49674`): `npm run quality` no `web/` passou com 93 arquivos/578 testes após reflow de Prettier em `NexusHud.tsx`; `go vet ./internal/desktop ./cmd/nexus-hud` e `go test -count=1 ./internal/desktop/...` PASS; build Web e build Go do HUD PASS.
- **PASS** — UI Automation no build final: presets medidos `260×70 → 380×560 → 334×496 → 380×560 → 440×640 → 334×496 → 220×60`; alça acessível `24×42` com nome `Arrastar HUD. Use as setas para mover.`; arraste real `-132,-88` px (pedido `-144,-96`); reinício abriu em `220×60` na posição persistida `1524,768`.
- **PASS** — teclado (setas `+16` px, `Shift`+seta `+64` px) validado no build `v3`; não repetido no rebuild `v4` porque o caminho de teclado não mudou.
- **NOT_TESTED** — Linux e macOS do HUD nativo neste slice.
- **BLOCKED** — `node scripts/docs-verify.mjs` falha por alterações visuais não commitadas em `web/src`/`internal/desktop`; o gate exige SHA imutável. Nenhum GO, tag ou release.

<!-- frontend-verify:latest -->
## Frontend gate — 2026-10-25T17:07:00Z

Verdict: **PASS**. Relatório completo: [`DEV/validation/FRONTEND_LATEST.md`](validation/FRONTEND_LATEST.md).

## P8 Linux security/distribution — 2026-10-25

- `make security`: PASS (`No vulnerabilities found.`), govulncheck v1.7.0 / Go 1.26.10; escopo do scanner limitado, revisão arquitetural pendente.
- `make release-dry-run`: PASS; build do frontend/CLI e contratos de versão.
- GoReleaser v2.18.0 snapshot: PASS; CLI tar e DEB AMD64 com checksum válido e payload idêntico. DEB instalado/consultado/executado/removido em raiz dpkg temporária; instalação host intacta.
- Source: HEAD/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + fingerprint local `268a9e9a0036e89e39bcff07eca880869534c9c75cc13765384bb3fa3fc30b32` (1.421 fontes); worktree não congelada.
- Evidências: [matriz](validation/linux-100-2026-10-25/TEST_MATRIX.md), [segurança](validation/linux-100-2026-10-25/SECURITY_VALIDATION.md), [distribuição](validation/linux-100-2026-10-25/DISTRIBUTION_VALIDATION.md).
- Resultado global: `NO_GO_LINUX`; beta.10 sem assinatura/update oficial e campanhas P2/P9 permanecem abertas.

## P9 fixture smoke e soak — 2026-10-25

- `GOCACHE=/tmp/nexus-linux-100-2026-10-25/gocache-p9 make overnight-smoke`: PASS, exit 0 (15,864 s).
- `GOCACHE=/tmp/nexus-linux-100-2026-10-25/gocache-p9 make overnight-soak DURATION=5m`: PASS, exit 0 (300,012 s); o teste exige ciclo não vazio e terminal permitido.
- Cobertura é determinística e isolada. Persistência/reabertura para inspeção no dia seguinte e continuidade com provider real seguem pendentes. Ver [runtime](validation/linux-100-2026-10-25/RUNTIME_VALIDATION.md) e [matriz](validation/linux-100-2026-10-25/TEST_MATRIX.md).


## Follow-up Linux — mascote único, CLI e Web lifecycle — 2026-10-25

- **PASS** — direção de mascote: renderer único `NexusHud → GuardianMascot → NexPixelSprite`; 20/20 testes focados e verificação de arte. Evidência em `validation/linux-100-2026-10-25/MASCOT_VALIDATION.md`.
- **PASS** — CLI: help e erros de argumentos ocorrem antes de inicializar o servidor; pacote `internal/app` e vet passaram; build atual instalado com backup. SHA-256 `51b58fe5972f27fcdb55c8284388b3f37998dc57db384b11223719ad6a132905`.
- **PASS/BLOCKED P2** — servidor Web isolado passou dois ciclos health 200/shutdown/restart. A UI conectada fica BLOCKED: o XID observado pertenceu ao Desktop pré-existente PID 1235319; o ensaio não provou continuidade de um cliente isolado. A/B/C/D/E integrado e isolamento WebKit continuam pendentes. Capturas com nomes do host foram removidas. Um aviso opcional `/etc/hosts` foi registrado; arquivo não mudou.
- **PASS** — GoReleaser snapshot CLI tar/DEB e ciclo dpkg temporário. Nenhuma publicação.
- Global: **NO_GO_LINUX**, 104 PASS / 1 FAIL / 16 BLOCKED / 5 NOT_RUN. Fingerprint do overlay: `a2ed858e013e8e43391cec7c1bb6d6e00324a3170ea9804d0d068bf231e73dfd`; remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. Próxima ação: isolar o cliente WebKit e repetir UI conectada/restart do servidor, além de A/B/C/D/E; preservar processos existentes.
- Reteste corrente do overlay: mascote 20/20 + arte oficial, `go test -count=1 ./internal/nexus/runner ./internal/control/web`, `git diff --check` e consistência matrix/JSON passaram. Log: `validation/linux-100-2026-10-25/logs/mascot-and-go-regression-recheck-2026-10-25.log`.
## 2026-10-25 — reteste do mascote único e do Desktop nativo isolado

- **PASS** — fonte ativa `NexusHud → GuardianMascot → NexPixelSprite`; seletor com Barra/Mascote/Bandeja do mesmo HUD, sem Companion 2D. Quatro suítes focadas passaram 20/20 e `bun run art:nex:check` passou. Log: `DEV/validation/linux-100-2026-10-25/logs/mascot-single-renderer-recheck-2026-10-25.txt`; captura XWayland mostra uma figura NEX.
- **PASS** — P2-UI-001 retestado: janela Wails de teste PID 2154281/XID `0x140000e` continuou aberta durante restart do Core e renderizou a tela inicial após F5; HTTP 200 antes/depois. Build temporário alterou apenas IDs de instância para isolar o cliente. Log: `DEV/validation/linux-100-2026-10-25/logs/p2-native-ui-xwayland-isolated-instance-2026-10-25.txt`.
- **FOLLOW-UP** — captura em 5 s estava preta; medir o primeiro paint. E2E integrado A/B/C/D/E e ACL multiusuário seguem NOT_RUN; não confundir o teste isolado com validação de isolamento cross-project.
- Matriz/JSON: **105 PASS / 1 FAIL / 15 BLOCKED / 5 NOT_RUN (126)**; resultado `NO_GO_LINUX`. SHA base `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay `a2ed858e013e8e43391cec7c1bb6d6e00324a3170ea9804d0d068bf231e73dfd`, worktree modificada.

## 2026-10-25 — first paint e tela inicial Wails medidos

- **PASS** — sob o D-Bus real do GNOME, XWayland e perfil/home isolados, o fundo escuro persistiu até 5,87 s; splash visível aos 9,54 s; tela inicial/workspace aos 15,78 s e sem mudança até 30,66 s. Processo terminou com exit 0. A VM não tem aceleração 3D utilizável; não há SLA definido. Logs: `DEV/validation/linux-100-2026-10-25/logs/p2-native-first-paint-sessionbus-2026-10-25.txt` e `p2-native-startup-state-2026-10-25.txt`.
- Capturas sem nomes de projeto do host: splash `p2-native-first-paint-sessionbus-visible-2026-10-25.png`; workspace `p2-native-startup-state-15s-2026-10-25.png`. As capturas aos 15 s e 30 s têm o mesmo SHA-256. O ensaio privado de D-Bus fica inconclusivo por erros de portal e detecção de janela; não conta como falha de produto.
- `NO_GO_LINUX` permanece. P2 A/B/C/D/E integrado e ACL multiusuário continuam sem validação.

## 2026-10-25 — Composer → MissionRunner (fixture)

- **PASS:** `go test ./internal/nexus/... -count=1 -timeout=4m`.
- **PASS:** `go vet ./internal/nexus/...`.
- **PASS:** `go test -race ./internal/nexus -run '^(TestComposerApprovedFlowReachesMissionRunner|TestProviderResourceRequirementsKeepTaskSpecialtiesOutOfControlGate)$' -count=3`.
- O fixture percorre Composer, WorkPlan com proveniência, preview pronto, token de aprovação, snapshot imutável e MissionRunner até `COMPLETED_VERIFIED`. Perfil/CLI Codex e auth/quota são sintéticos; nenhum provider externo foi chamado.
- Evidências e limites: `validation/linux-100-2026-10-25/P5_COMPOSER_MISSION_VALIDATION.md`. Overlay `5e39faee13bd663dc56798cffe435e52e2bca081f5332f6d0619c0a61ddce011`, base `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. P5 parcial; NO_GO_LINUX.

## 2026-10-26 — Mission Codex real em fixture isolada

- `make build` passou. Binário CLI testado: SHA-256 `437ccf9a14b04a3bd463ce994cc36ba6fcf947f98d8e452ba990fcd4b5fad51c`; HEAD `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`, fingerprint de produto `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c`.
- Uma execução real com sessão Codex descartável retornou `COMPLETED_VERIFIED` e package `VERIFIED`, porém o harness inicial não guardou o worktree/recibo. A repetição com inspeção explícita excedeu 180 s; o runtime foi encerrado, contagem final zero, Core exit 0 e raiz temporária removida.
- RT-002 ficou **BLOCKED** até que worktree, recibo e DoD global sejam inspecionados juntos. Matriz/JSON: **NO_GO_LINUX — 169 PASS / 1 FAIL / 22 BLOCKED / 0 NOT_RUN (192)**.
- O produto mantém um único mascote NEX, sem Companion 2D separado.

## Terminais e agentes — YOLO e scrollback — 2026-10-26

- Validação de implementação e testes para `--yolo` persistido, atualização do modo Safe/YOLO no terminal e preservação da rolagem manual em mouse/PageUp/PageDown/Home/End: ver `DEV/validation/TERMINALS_AGENTS_YOLO_SCROLL_20261026.md`.
- Escopo coberto: Codex (alias para a flag nativa), normalização compartilhada do launcher e ambos os componentes PTY, usados por OpenCode e Oh My Pi. Não foi feita chamada a modelo.
- O binário beta.11 instalado continua aberto e não contém este overlay; há runtime Codex ativo. A atualização/reinício do aplicativo fica pendente para não interromper essa sessão.
