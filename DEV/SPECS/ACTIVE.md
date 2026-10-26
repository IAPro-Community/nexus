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
- Importação host real: Gemini foi copiado para um perfil único, autenticado e padrão; reimportação não duplica. O perfil informa `Session needs refresh`; Claude não tem tokens reutilizáveis (`NO_SESSION`). Nenhuma chamada de modelo foi feita. Evidência: `DEV/validation/linux-100-2026-10-25/PROVIDER_SESSION_IMPORT_VALIDATION.md`.
- Estado atual: **NO_GO_LINUX — 169 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN (191)**. Permanecem pendentes release estável/assinatura e SHA imutável, handoff/continuidade com provedores além do smoke direto Codex, preview P5 na janela Wails, alpha/foco/múltiplos monitores e GPU/compositor Wayland.

## Critério ativo do mascote

A apresentação `Mascote` é o NEX único; não criar modo, componente ou renderer Companion 2D. Preservar acessibilidade, movimento reduzido e estados de atenção reais. Evidência do escopo e limites nativos: `DEV/validation/linux-100-2026-10-25/MASCOT_VALIDATION.md`.

## Recheck atual — 2026-10-26

Cinco suítes HUD passaram (33/33), junto com a verificação da arte oficial e o E2E HUD. Wails/Xvfb no binário corrente abriu um NEX em 224×158 e o clique mostrou os controles; screenshots/log preservados. O smoke direto Codex pelo CLI Nexus passou em execução efêmera/read-only; sem stdout bruto preservado, como explicitado no registro. Resultado global segue NO_GO_LINUX por gates de release, provider/handoff, P5 Wails e Wayland/alpha/GPU.

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

## Estado corrente — homologação Linux e NEX único — 2026-10-26

Branch/base local/remota `release/v1.0.0@7a04f1b4cf5950b97b4ea1e156dda31ac513823b`; overlay atual `751cfc1a160c38d950ac5b9807e1febb9746d03981c861c722e5bd965e1ef012` (1.428 fontes; manifesto conferido). Worktree modificada, sem commit imutável. O produto tem um único NEX, sem modo/renderer Companion 2D. Desktop e HUD Wails atuais passaram em Xvfb; Desktop/Core HTTP 200 e importação 0/0/5/1/0 em HOME isolado; HUD mostrou um NEX e controles após clique. Alpha Wayland permanece BLOCKED. FIX-029 eliminou os advisories reportados pelo Bun após atualização do Vitest/lockfile; `quality:full` passou com 109 arquivos/675 testes/build. Resultado: **NO_GO_LINUX — 139 PASS / 1 FAIL / 18 BLOCKED / 4 NOT_RUN (162)**. REL-001, idle acelerado, evento externo real, alpha e proveniência congelada permanecem em aberto; soak P9 continua ativo. Ver `DEV/validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md`, `MASCOT_VALIDATION.md` e `SECURITY_VALIDATION.md`.

## Snapshot histórico anterior — homologação Linux e mascote único — 2026-10-25 (UTC 26/10)

Nesse snapshot histórico, o estado no overlay `391c5718b3f2cd494fb4fa0e6a9fae6f03a618c002155258a59b7e7e83f27459` (1.428 fontes; base local/remota `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`); worktree modificada e não congelada. **NO_GO_LINUX — 137 PASS / 1 FAIL / 20 BLOCKED / 4 NOT_RUN (162)**. O contrato do produto é um NEX único; PR #55/Companion 2D está fechada e fora do escopo. `quality:full` passou com 109 arquivos/675 testes/build; 32 testes focados do mascote e `art:nex:check` passaram; Wails real sob Xvfb mostrou um NEX. FIX-028 removeu `nx-spin-slow` do indicador de update; no Wails/Xvfb o WebKit principal mediu 196,1% no baseline e 107,8% pós-fix por 20 s; delta direcional, pois eram três processos WebKit no baseline e um no reteste. NR-010 continua BLOCKED porque resta CPU alta e a VM não tem aceleração 3D/perfilador permitido. O soak P9 de 8 h está ativo. REL-001 segue FAIL (sem release estável v1.0.0); alpha/interação Wayland no host, atenção de agente real, provider externo, assinatura/update oficial e proveniência imutável seguem com gates abertos. Evidência: `DEV/validation/linux-100-2026-10-25/MASCOT_VALIDATION.md`, `TEST_MATRIX.md` e `LINUX_HOMOLOGATION_REPORT.md`.

## Snapshot histórico do reteste P7 — 2026-10-25

Branch `release/v1.0.0`; HEAD remoto confirmado `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`. Um único mascote NEX é o contrato: `GuardianMascot` renderiza `NexPixelSprite`; Barra/Mascote/Bandeja são apresentações do HUD. Não existe segundo mascote ou modo Companion 2D integrado para remover.

Overlay histórico P7: `bb575423508a0c773580804a476b0888ff2449135045a74f80aa6efaffedc6e1` (1.421 fontes; HEAD local/remoto `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`). Mascote permanece NEX único; nenhum modo Companion 2D separado existe. P3 implementou outbox/retry/recovery; P4 passou bounds/redaction/freshness/symlink e E2E de grounding; P5 Composer→WorkPlan passou com 2 clarificações/5 etapas; P6 passou matching/fallback/proveniência, identidade efêmera, persistência e cleanup em fixtures. Go nexus/control, race focado x5 e vet passaram; AGY real retornou READY em um perfil e HTTP 429 esperado no perfil sem quota. OpenCode não produziu resposta verificável em 90 s. `ai-cli` permanece ausente. P2 e handoffs P7 live/P8/P9 continuam abertos. Full Go race passou. `NO_GO_LINUX`: 97 PASS / 1 FAIL / 15 BLOCKED / 5 NOT_RUN (118).

**P3 — notificações:** concluído em código e testes; relatório `DEV/validation/linux-100-2026-10-25/MISSION_NOTIFICATIONS_VALIDATION.md`.
**P4 — Project Intelligence:** implementado; E2E determinística passou. Integração real `ai-cli` está BLOCKED por ausência do executável. Relatório `DEV/validation/linux-100-2026-10-25/PROJECT_INTELLIGENCE_VALIDATION.md`.
**P5 — Composer/WorkPlan:** spine E2E passou depois de aguardar resposta do finalize. **P6 — Agents/recursos:** regressões determinísticas de fallback, identidade e lifecycle passaram. **P7 — runtime:** race integral e PTY/lease passaram; handoff same/cross-provider permanece BLOCKED. P2/P8/P9 permanecem abertos.

**Follow-up P1, 2026-10-25:** P1 COMPLETE no overlay `ddad56697609de1374cdede949e794316e3263357b052f745defb64f11a4be5b` (1.412 fontes). O E2E HTTP autenticado → decisão/outbox durável → restart → exatamente uma retomada verificada passou 10 vezes e 5 vezes com race. Relatório/logs em `DEV/validation/linux-100-2026-10-25/P1_INTERVENTION_VALIDATION.md`. P2–P9 e gates de release continuam abertos; resultado global `NO_GO_LINUX`. O mascote permanece um NEX único, sem modo Companion 2D.

**Base da campanha FIX-012 (histórica):** branch `release/v1.0.0`, HEAD então consultado `bf6adf0c80ac3531d7291844512a14a017ab0002`, versão `1.0.0-beta.10`. O overlay daquele registro foi `516b706a2c3cdca1a6d76200833a785186063350fa4d41db4494fcb67bdb7f7a` (1.404 fontes); o HEAD remoto foi atualizado no follow-up acima.

**Veredito daquela campanha:** `NO_GO_LINUX` — 63 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN (81 casos). A contagem atual está no follow-up no topo e no relatório canônico `DEV/validation/linux-100-2026-10-25/LINUX_HOMOLOGATION_REPORT.md`.

**Contrato do mascote:** um NEX único, bem acabado e funcional. Não existe personagem, renderer ou modo Companion 2D no checkout atual. `NexusHud` usa `GuardianMascot`/`NexPixelSprite`; Barra, Mascote e Bandeja são apresentações do HUD. Atenção e humor são estados do mesmo mascote. A regressão focada passou 25/25, e a captura nativa mostra um NEX. A PR #55 propunha uma implementação Companion separada; foi fechada em 2026-10-25 e está fora do escopo.

**Estado de verificação:** Os gates anteriores de Go quality-full com race/security e frontend 106 arquivos/664 testes passaram; CLI/Desktop/HUD AMD64, snapshot GoReleaser e instalação DEB isolada foram validados. Após FIX-012, os testes do popover e race passaram, o HUD nativo passou três ciclos sem critical GDK, os testes do mascote passaram 25/25 e a captura mostra apenas o NEX único. Uma Mission real DIRECT terminou `COMPLETED_VERIFIED`. O Desktop final importou quatro sessões e as reconheceu após reinício; a regressão do timeout lento passou. FIX-006 corrigiu e validou visualmente as abas PTY divididas; FIX-011 anexa o terminal existente pelo `runtime_id`. Input nativo nos dois shells segue BLOCKED. HUD e Desktop iniciaram em Wayland direto; controles funcionais do HUD passaram. Balão/adiamento nativos passaram com fixture local, mas a ligação de atenção real a agente, roteamento completo, continuidade, skills, teclado/foco/movimento Wayland, transparência alpha e updater assinado continuam bloqueados/não executados.

**Follow-up PR #57/#58 — 2026-10-25:** integração local automatizada passou `make quality-full` (frontend 105/660, Go/race, `govulncheck`), 7/7 geometrias Chromium, build CLI/Desktop/HUD AMD64 e inicialização real Desktop+HUD em XWayland e Wayland direto. A regressão nativa posterior corrigiu a sobreposição dos nomes PTY em duas sessões; input split e navegação nativa de worktree continuam pendentes. Fingerprint do follow-up #57/#58: `a6be83416651b6cdb4e0cb77a8c0448c17708d19e4c3b8d6f7e41bb8a5ad0b0a`; overlay posterior FIX-009: `dac0d00b8410f70ed02c8b4c324780349655fed8f7e580d1944da1ec3eaecefa`. Detalhes em `DEV/validation/linux-100-2026-10-25/PR578_FOLLOWUP.md`. PRs #57/#58 continuam Draft. Overlay sem commit, nenhum SHA imutável; `NO_GO_LINUX` permanece.

**Follow-up PR #61 — 2026-10-25:** a busca do portal falhava porque o plugin indexava a rota padrão `/docs`, enquanto o preset publica documentos em `/`. `docsRouteBasePath: ["/"]` gerou o índice; build estático, busca Playwright por Terminais, catálogo de 100 páginas e auditoria npm (0 vulnerabilidades) passaram. A PR permanece Draft. Evidências: `DEV/validation/linux-100-2026-10-25/PR61_DOCS_SITE_VALIDATION.md`.

**Direção confirmada do mascote:** não desenvolver Companion 2D nem um segundo personagem. A apresentação `MASCOT` mostra o único NEX; estado/atenção pertence ao mesmo modelo visual. PR #55 está `CLOSED`.

---

## Continuidade — compilar/instalar HEAD remoto mais recente — 2026-10-23 13:56 UTC

`release/v1.0.0` avançou por fast-forward até `9929bef2070a95672a3af8381c09cffbfbe36711` (commits `689fa3e2` e `9929bef2`). `make build build-desktop build-hud` compilou CLI, Desktop e HUD sobre a árvore com alterações locais preservadas. Instalados atomicamente em `~/.local/bin` e no `~/.local/share/ai-manager/profiles/codex/kivergmail/home/.local/bin`; `nexus version --json` confirma `1.0.0-beta.10`, commit `9929bef2`, Linux AMD64. Hashes: CLI `fb24d4d05769032621e0cb926fa52152016c0f88183dcc5513d34a6d14f78310`, Desktop `a12d8ad2a576a0f99c7013bbe463729d2d4fe37f6827fe3b864b159616412320`, HUD `597a5101c4cf0a85edbff0e6259a020d824f2045a542f1ee25445e511f8acad1`. Backups em `~/.local/state/ai-manager/nexus-update-backups/20261023T135607Z`. A atualização compilou; reinício dos seis runtimes e do serviço web com executáveis novos ainda está pendente nesta passagem.

# Especificação ativa: IAPro Nexus — Implementação Completa Web + Desktop Multiplataforma

## Contrato de mascote reafirmado — 2026-10-25

O produto tem um único NEX, sem um personagem ou modo Companion 2D separado. Barra/Mascote/Bandeja são apresentações do mesmo HUD; preservar a apresentação Mascote, que renderiza o NEX. No overlay `2c23ebb0…`, 29 testes em cinco suítes, verificação da arte, E2E HUD Chromium e três capturas nativas Wails/XWayland confirmam o renderer único. Alpha Wayland e atenção de agente real seguem pendentes. Ver `DEV/validation/linux-100-2026-10-25/MASCOT_VALIDATION.md`.

## Registro da matriz base — 2026-10-25 (superado pelo follow-up nativo acima)

O HEAD local e remoto consultado é bf6adf0c80ac3531d7291844512a14a017ab0002; VERSION permanece 1.0.0-beta.10. A árvore está modificada. O Desktop da última revalidação foi compilado com overlay fingerprint 825800ba46bab26ac692a8cc5f8cfc82f64ed474cf36459a672d09943c85b62b (1.399 arquivos), mas ainda não é um SHA Git limpo e imutável.

**Resultado da matriz base, atualizado nos follow-ups:** 41 PASS / 1 FAIL / 9 BLOCKED / 9 NOT_RUN (60 casos). O resultado corrente é 48 PASS / 1 FAIL / 11 BLOCKED / 7 NOT_RUN (67 casos). O produto tem um mascote NEX, sem personagem/modo Companion 2D.

**Contrato do mascote:** o produto tem um único mascote NEX. Personagem e modo Companion 2D não fazem parte do escopo. Barra e Bandeja são apresentações do HUD. O modelo de atenção pertence ao NEX único.

Evidências: relatório, matriz e resumo em DEV/validation/linux-100-2026-10-25/.

### Registro histórico substituído — homologação de 2026-10-23

## Estado atualizado — homologação Linux, build e instalação — 2026-10-23

O `release/v1.0.0` consultado está em `bf6adf0c80ac3531d7291844512a14a017ab0002`; `VERSION` permanece `1.0.0-beta.10`. CLI, Desktop e HUD foram compilados com Go 1.26.10, instalados nas duas raízes do usuário e os processos ativos foram reiniciados. Hosts, Web, Desktop e HUD carregam binários com os hashes instalados; health do Web e Core Desktop = HTTP 200. Gates Go/Web passam (99 arquivos / 612 testes), mas há E2Es Composer falhos e cobertura nativa incompleta.

**Registro histórico, substituído em 2026-10-25:** a avaliação daquela data chamava o NEX de Guardian e tratava a PR #55 como requisito. A direção vigente é um mascote NEX único, sem Companion 2D. Matriz histórica: 30 PASS / 7 FAIL / 6 BLOCKED / 12 NOT_RUN. Relatório daquela execução: `DEV/validation/linux-100-2026-10-23/LINUX_HOMOLOGATION_REPORT.md`; para o estado atual, use a homologação de 2026-10-25 no início deste documento.

## Follow-up ativo — configuração de provedores no primeiro acesso — 2026-10-22

O cartão **Ambiente de execução** deve tentar reaproveitar sessões autenticadas locais reconhecidas antes de encaminhar o usuário para novo login. Aceite implementado: ação explícita na tela; importação somente de formatos file-backed suportados para perfis Nexus isolados; nenhum overwrite; origem intacta; resumo de resultado; fluxo manual como fallback; API protegida por CSRF; regressões Go/Web. No diálogo de conta, o comando padrão não deve repetir o nome do provedor: `nexus login codex` resolve ao perfil Codex padrão; perfis nomeados continuam explícitos.

Implementado para Codex, Claude, Gemini, AGY e OpenCode. Cursor continua `UNSUPPORTED` no adaptador atual. O comando de login duplicado foi corrigido e `make quality`, vet, race focado, build Wails e startup nativo passaram. **Validação pendente:** abrir o cartão e clicar na janela Wails nativa, porque GNOME Wayland bloqueou screenshot e controle nesta sessão. Dossiê: `DEV/validation/linux-100-2026-10-22/SESSION_IMPORT_FOLLOWUP.md`. O veredito exact-SHA da release permanece `NO_GO_LINUX`.

## Homologação Linux 1.0.0 — estado observado 2026-10-22

`release/v1.0.0` está em `f62fdec0c571ac6a19a904164d3f580799da6bf3` no remoto e na cópia de homologação. `VERSION` ainda reporta `1.0.0-beta.10`. Decisão independente: **NO_GO_LINUX**. A branch não integra o Nexus Companion 2D da PR #55 (só Guardian antigo); release estável assinada não existe; cinco suítes browser E2E falham; Wayland bloqueou screenshots e input da janela nativa. Build/execução real Linux passaram, mas não completam esses gates. Correções permanecem locais, não commitadas, em `DEV/validation/linux-100-2026-10-22/FIXES.patch`; SHA-256 `5557c3bccf606baffcdc8e3db1e44c7cd423d0da570baf69f9acb8744d749b10`. Matriz e evidências estão no diretório de validação. Este estado substitui os snapshots históricos subsequentes deste arquivo; a fonte de verdade da campanha é o SHA e o relatório ligados acima. O documento de reconciliação anterior em `DEV/validation/NEXUS_100_BRANCH_COMPANION_RECONCILIATION_20261022.md` é uma fotografia do SHA `9ff1889`, enquanto a confirmação atual foi refeita no HEAD `f62fdec0`.

---


## Estado atualizado — sincronização 1.0/1.1 e branches — 2026-10-22

`release/v1.1.0` = `c01cd04a`, com o hardening da 1.0 incorporado via PR #60. O sync também restaurou a buildabilidade da 1.1 (bloco `var` corrompido em `project_manifest.go`, `strings.ValidUTF8` inexistente, IPv6 entre colchetes no SSH bridge, literais quebrados em testes e `registeredAccountScope` faltando). `release/v1.0.0` = `0ac9d8ed` (docs-only sobre `f789db63`).

Registro histórico de 2026-10-22: PRs #59 (fix HUD, mergeable, aguardando review/homologação), #57 e #58 (sincronizados com a 1.0, ainda draft; o check "Sanitize self-hosted workspace" fica `queued`), #55 (Companion 1.1, retargetado para 1.1; merge bloqueado por conflito de arquitetura com o Guardian Mascot da 1.0). O estado atual da PR #55 é `CLOSED`, após a decisão do produto de manter apenas o NEX. `fix/beta8-manual-opaque-quota` não está na 1.0 e o PR #52 foi fechado sem merge; destino pendente. Veredito de release permanece `NO_GO_WINDOWS` naquele snapshot.

## Estado atualizado — popup independente/Mascote controlável @ f789db63 — 2026-10-22

`release/v1.0.0` em `f789db63` (código). O modo Mascote deixou de ser inalcançável: o popup independente virou preferência persistida (`HUDWindowState.DetachedPopover`) com toggle no painel do HUD (`Popup independente: Sim/Não`), aplicado em runtime sem reiniciar (segunda janela criada sob demanda; ao habilitar a cápsula volta ao tamanho compacto; ao desabilitar restaura BAR e o painel inline). Evidência real em `DEV/validation/mascot-toggle-f789db63/homologation/RESULT.json` (`PASS`, duas sessões com reinício).

Gates: `gofmt`, `go vet`, `go test -count=1 ./...` (Go 1.26.10), Web quality 97/599, build Web e build HUD. HUD SHA-256 `1f3ced07...7929de`.

Pendências inalteradas: click-through do popover destacado, confirmação visual humana, instalador assinado, 4 gates `NOT_RUN`, Codex interativo e matriz cross-platform. Veredito de release: `NO_GO_WINDOWS`. Relatório: `DEV/validation/mascot-toggle-f789db63/HOMOLOGATION.md`.

## Estado atualizado — click-through do HUD corrigido @ 047f4348 — 2026-10-21

`release/v1.0.0` em `047f4348` (código). O click-through do HUD foi corrigido: o renderer reporta as regiões interativas (`data-hud-panel`) e um watcher nativo de 30 ms alterna `WS_EX_TRANSPARENT` só fora delas, com fail-open. Evidência real: `CLICK_THROUGH=PASS` (canto transparente atravessa para `Nexus ClickTarget`, cápsula continua interativa) em `DEV/validation/local-hud-047f4348/click-through-official/RESULT.json`.

Gates: `gofmt`, `go vet`, `go test ./...` (Go 1.26.10), Web quality 97/598, `docs-verify` 192/33, HUD 20/20 (`PASS_FUNCTIONAL_PENDING_VISUAL`). HUD SHA-256 `19c5e299...a1e6a5`.

Pendências inalteradas: instalador assinado, 4 gates `NOT_RUN`, Codex interativo, matriz cross-platform, click-through do popover destacado e confirmação visual humana. Veredito de release: `NO_GO_WINDOWS`. Relatório: `DEV/validation/local-hud-047f4348/HOMOLOGATION.md`.


## Estado atualizado — homologação Windows 7d92b942 — 2026-10-21

`release/v1.0.0` em `42a6971b` (docs-only); código homologado `7d92b942`. PR #54 integrado (`c3c518ce`, head `3e140c52`). Gates verdes: `docs-verify` 192/33, contrato de versão, `gofmt`, `go vet`, `go test ./...` (Go 1.26.10), Web quality 96/593, builds Web/HUD e certificação oficial 9 PASS.

HUD nativo: 20/20 ciclos em janela única e 20/20 em popover destacado; visual carregado sem clipping. **Click-through FAIL**: a área transparente consome o clique em vez de repassá-lo à janela atrás — `docs/product/hud-native-visual-homologation.md` classifica isso como NO_GO.

Pendências externas: certificado Authenticode (`native-signing` com 0 secrets), NSIS/`signtool`, sessão Codex real, 4–5 contas autenticadas e desktop first-run manual. Veredito permanece `NO_GO_WINDOWS`. Relatório: `DEV/validation/local-hud-7d92b942/HOMOLOGATION.md`.

## Estado atualizado — homologação beta.10 — 2026-10-17

HEAD local `984c2b0fdbd7b4c4b4917c9de7f1a4d2a60a4b17` na `release/v1.0.0`; remoto `6552f3d0b834`. Três commits de remediação locais, sem push. Temp curto corrigiu os erros de worktree Git; suíte Go integral passou em 414,8s, exit 0. Golangci-lint pinned v2.12.2 passou em 5 min. Ainda faltam refresh oficial no SHA final, harness clean exact-SHA, NSIS, assinatura, Codex interativo, HUD e demais evidências externas. Veredito continua `NO_GO_WINDOWS`.
## Slice concluído — HUD nativo arraste/tamanhos — 2026-10-21

Worktree `C:\Dev\nexus-hud-validation`, base `f6098b8c684a97bf7e8e56804109ca491c78cfa2`, sem commit/push. Entregue: alça de arraste nativa de 24 px (clique do corpo preservado), movimento por setas e presets persistentes `compact`/`comfortable`/`large`, com primeiro paint corrigido.

Aceite final no rebuild `nexus-hud-diag-v4.exe` (SHA-256 `ae64668c…`): `npm run quality` PASS (93 arquivos/578 testes), `go vet`/`go test ./internal/desktop/...` PASS, presets e arraste medidos por UI Automation em Windows 11 `10.0.26200` (`-132,-88` px), reinício em `220×60` na posição persistida. Relatório: `DEV/validation/local-hud-f6098b8c/native-hud-drag-size/REPORT.md`. Linux/macOS `NOT_TESTED`; `docs-verify` `BLOCKED` por alterações visuais não commitadas. O slice **não** altera o veredito `NO_GO_WINDOWS` nem autoriza tag/release.

## Contexto de ambiente — PR #54 instalado no host — 2026-10-21

`%LOCALAPPDATA%\Programs\IAPro Nexus\nexus.exe` e `ai.exe` correspondem ao head do PR #54 (`3e140c52`, SHA-256 `4cb4a311…`), aprovado no GitHub e sem merge local. O `nexus-hud.exe` instalado não foi trocado; o slice de HUD permanece isolado em `.tmp/homolog/bin/nexus-hud-diag-v4.exe`.

## Contrato ativo — remediação/homologação Windows beta.10 — 2026-10-17

Escopo corrente é o plano `DEV/validation/current/WINDOWS_BETA10_CODEX_LOCAL_REMEDIATION_20261016.md`.
Checkout de homologação somente em D:, branch `release/v1.0.0`, HEAD-base
`5d5aea9211b5154e96b036c519a61de9723593b1`; remoto observado em `6552f3d0`.
Não usar `-SkipDesktop`/`-AllowDirty`, não alterar dados/perfis reais, não
consumir APIs pagas, não publicar nem criar assinatura/tag.

Estado verificável atual: frontend quality completo passou no worktree posterior
(89 arquivos/553 testes), testes direcionados terminal/control, interop,
version contract, fixtures offline release/update/uninstaller passaram. Gate
principal Go falhou pelo timeout padrão de 10 min da package `internal/nexus`;
golangci-lint falhou pelo timeout de loader em duas tentativas. Um teste a11y
falhou no SHA-base e passou três vezes após sincronização limitada de foco,
mantendo a asserção. NSIS, assinatura, updater real A→B, HUD visual completo,
Codex interativo, CI do SHA final e Linux nativo seguem sem evidência ou
bloqueados. Ver relatório e JSON no diretório de evidências desse SHA; decisão
atual `NO_GO_WINDOWS`.

## Atualização de homologação Windows beta.7 — 2026-10-07

Base local: branch release/v1.0.0, SHA-base 52f35c04181b85379219ab55fc3192c63a5440a8; VERSION=1.0.0-beta.7. Gates Go/Web/E2E, HUD/Desktop e instalador isolado passaram no Windows. O NSIS exato da árvore compilou com NSIS 3.13; a variante com HKCU/atalho isolados instalou/removeu o payload real. O smoke Desktop Wails/WebView2 passou por 10s após restringir cleanup ao perfil Nexus. A árvore segue dirty; docs-verify falha porque o manifesto visual está preso a 2892f301 e o gate exige recaptura em SHA limpo. Exact-SHA, execução do instalador exato, CI Windows remoto, assinatura e trust root permanecem pendentes. O host não tem runner Windows, Windows SDK/signtool ou certificado Authenticode. Nenhum gate Linux foi executado. A auditoria viva não encontrou branch Beta 7 nova nem merge pendente. Consulte DEV/validation/current/BETA7_WINDOWS_HOMOLOGATION_20261007.md e DEV/validation/current/RELEASE_STATUS.md.

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

### Implantação — 2026-10-22

955b197f implantado localmente: servidor PID 24184 (HTTP 200, commit 955b197f) e HUD PID 21168 --linked com SHA-256 8364a942...4065, o mesmo artefato que passou no harness do Mascote. O estado do usuário estava em TRAY com o popup desligado; o toggle agora fica disponível no painel. Rollback documentado em DEV/validation/mascot-toggle-f789db63/DEPLOYMENT.md.

### Verificação real — 2026-10-22

Instalação exercitada no perfil do usuário: popup habilitado pelo painel, Mascote aplicado e Bandeja restaurada (real-profile/RESULT.json PASS); app desktop abriu e anexou ao Core existente. Estado final TRAY + detachedPopover: true.

### Entrega atual — 2026-10-22

Toggle do popup independente no topo do painel (inline e popup), validado por harness no build `9b78fb10` (commit `8d100e96`) e no perfil real; implantado localmente. Pendências inalteradas: assinatura/instalador, smokes interativos de Codex/multi-conta, confirmação visual humana e matriz Linux/macOS.
