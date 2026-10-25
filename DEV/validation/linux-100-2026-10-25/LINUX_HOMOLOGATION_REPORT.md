# Homologação Linux AMD64 — Nexus 1.0.0

> **Atualização:** overlay corrente `516b706a2c3cdca1a6d76200833a785186063350fa4d41db4494fcb67bdb7f7a` (1.404 fontes), derivado do commit base abaixo. FIX-012 corrigiu crash GDK ao abrir o popover lazy do HUD; três ciclos nativos abrir/fechar passaram com `G_DEBUG=fatal-criticals`. Desktop e HUD atuais foram recompilados/abertos em XWayland. Os tarballs Desktop/HUD do workflow Linux e o snapshot CLI/DEB foram reconstruídos em diretórios isolados e tiveram payloads/checksums verificados. O NEX continua único; não há modo Companion 2D separado. A release permanece `NO_GO_LINUX`: a árvore está dirty, sem commit imutável, e persistem gates críticos de release/runtime.

Data: 2026-10-25

Resultado: **NO_GO_LINUX**

Branch: `release/v1.0.0`

Commit Git base local/remoto: `bf6adf0c80ac3531d7291844512a14a017ab0002`

Overlay atual testado: `516b706a2c3cdca1a6d76200833a785186063350fa4d41db4494fcb67bdb7f7a`

Arquivos no manifesto: 1.404 (`logs/source-provenance-gdk-fix.txt`)

Versão embutida: `1.0.0-beta.10`

Árvore: modificada, sem commit; o overlay está identificado por fingerprint e manifesto.

## Decisão sobre o mascote

O produto tem **um único mascote NEX**, com arte, estados e ações integrados ao HUD. A implementação ativa monta uma `GuardianMascot` e renderiza uma `NexPixelSprite`. `companionModel` contém o modelo de atenção e humor do mesmo NEX. O seletor apresenta Barra, Mascote e Bandeja; não há uma segunda personagem, renderer, atlas ou modo Companion 2D no checkout. A PR #55 não integra esta release.

## PRs de release inspecionadas em 2026-10-25

| PR | Estado remoto atual | Resultado para esta homologação |
| --- | --- | --- |
| [#55 Companion 2D](https://github.com/IAPro-Community/nexus/pull/55) | CLOSED, sem merge | Está fora do escopo confirmado pelo usuário. O checkout tem um NEX e nenhuma segunda personagem ou apresentação Companion 2D. |
| [#57 Worktrees](https://github.com/IAPro-Community/nexus/pull/57) | OPEN, Draft, MERGEABLE | A integração local e os testes estão descritos em [PR578_FOLLOWUP.md](PR578_FOLLOWUP.md). Navegação nativa de worktrees segue pendente. |
| [#58 Workspace contextual](https://github.com/IAPro-Community/nexus/pull/58) | OPEN, Draft, MERGEABLE | As partes exercitadas localmente estão cobertas em [PR578_FOLLOWUP.md](PR578_FOLLOWUP.md); a entrada simultânea nos dois PTYs e a recuperação de sessões não estão completamente homologadas. |
| [#59 Restauração do HUD](https://github.com/IAPro-Community/nexus/pull/59) | OPEN, não Draft, CONFLICTING | O overlay deste checkout contém `sync.OnceFunc` no primeiro `WindowShow`, `CommandNoConsole` e o uso em `hudlink.Spawn`. Build, testes Go e movimento/restauração XWayland passaram. A PR remota conflita e ainda requer smoke nativo Windows; nenhum merge foi feito. |
| [#61 Portal de documentação](https://github.com/IAPro-Community/nexus/pull/61) | OPEN, Draft, MERGEABLE | Build e busca do portal foram validados em [PR61_DOCS_SITE_VALIDATION.md](PR61_DOCS_SITE_VALIDATION.md); não altera a aplicação desktop. |
| [#64 Quality gates/hosts](https://github.com/IAPro-Community/nexus/pull/64) | OPEN, não Draft, base `main` | Não é direcionado à branch de release; não foi aplicado à homologação Linux da `release/v1.0.0`. |

O HEAD remoto permanece `bf6adf0c80ac3531d7291844512a14a017ab0002`. As verificações desta campanha correspondem a esse commit base mais o fingerprint local informado; nenhum PR foi mesclado ou publicado durante a homologação.

## Resultado

| Status | Casos |
| --- | ---: |
| PASS | 63 |
| FAIL | 1 |
| BLOCKED | 12 |
| NOT_RUN | 5 |
| Total | 81 |

Contagens atuais: **63 PASS / 1 FAIL / 12 BLOCKED / 5 NOT_RUN (81 casos)**. Foram acrescentados HUD-004 (restauração do popover GTK sob `G_DEBUG=fatal-criticals`) e NATIVE-008 (startup do Desktop no build pós-FIX-012). A matriz e o resumo JSON mantêm os fingerprints históricos por caso e apontam para 516b… como overlay final compilado.

- **Go:** no overlay atual, `go test -count=1 -timeout=20m ./...`, `go vet ./...`, `go test -count=1 -v ./internal/core/provider/adapters/cursor` e `go test -race -count=1 ./internal/core/provider/adapters/cursor` passaram. FIX-009 faz `InspectAuth` reconhecer `✓ Login successful!` sem tratar mensagens negativas como autenticação; copiar a sessão Cursor continua BLOCKED. Restam avisos de depreciação GDK/X11 upstream do Wails.
- **Frontend:** `bun run quality:full` após FIX-011 passou com 106 arquivos e 664 testes, formato, lint, Stylelint, typecheck, estilos e build; [log](logs/runtime-attach-quality-full.log). O E2E dividido passou em sete geometrias e com toolbars de 520 px/360 px; [log](logs/terminal-split-toolbar-regression-final.log).
- **Mascote:** FIX-010 removeu o estado/animação `success` sem telemetria própria verificada. `COMPLETED` e `COMPLETED_VERIFIED` mantêm o NEX neutro; a regressão RED/GREEN passou. Logs: [quality-full](logs/mascot-single-nex-quality-full.txt) e [source provenance](logs/source-provenance-mascot-fix.txt).
- **HUD/atenção do mascote:** fixture controlada `REQUIRE_USER` foi exibida no balão do HUD Wails real; a ação “Adiar” recolheu a bolha sem apagar a tarefa canônica. Capturas e API estão em [MASCOT_VALIDATION.md](MASCOT_VALIDATION.md). O caminho com evento real de agente continua BLOCKED.
- **Build nativo após FIX-010:** `make build-desktop-wails build-hud-wails` passou. Desktop SHA-256 `c32ca11e29a2add3a8d398a8bc63ed36c5f87d5b6d25409d569e6de90fcddf98`; HUD SHA-256 `f9acd2de7827f7ce4529d99189f0fa4116e6ca1a4ab05a9c3da030a30a4c4348`. O HUD Wails recém-compilado abriu em XWayland (224×158) e mostrou uma arte NEX; [captura e log](MASCOT_VALIDATION.md).
- **Build Linux AMD64:** frontend, CLI/Desktop/HUD e snapshots suportados foram compilados. Os binários raw são ELF x86-64; `ldd` não encontrou bibliotecas ausentes. Build após FIX-009: CLI `bcea3cbe3bff455c2b49c161f0c1b428e0a568395f23edbba05b9190d977fe47`, Desktop `e648e1273e0654f3ea1ff7ab6e438a72f8b019881a480806964689267dcc57aa`, HUD `cf00cee5b82c6f856625c87bee8aef1891e39ad975e3fbbe1a491b76457e1dfd`; [log](logs/build-cursor-status-fix.txt). Smoke nativo de Desktop/HUD passou por 7 s e encerrou sem processos residuais: [log](logs/native-smoke-cursor-status-fix.txt).
- **Build e pacotes pós-FIX-012:** CLI `632f6e00620e79e19794b55df3fdb00f6f39ed1263f11f114b59deb7ad13926e`, Desktop `b500303ec1e4654ec267cb171097b65c707b7de67b571b8650837a30cd8df492`, HUD `a70789f8c5e6afa45515c84abab8a0a311070821719712227360d1ed6137a0d6`; os três são ELF x86-64. `ldd` não encontrou bibliotecas ausentes no Desktop/HUD. Os tarballs do workflow CI têm hashes `8ed42fe048300f4069a4cbafb61378080c0aae9d2c7e950053fe7da67cf8f` (Desktop) e `425e58cf811dc6b7245211bd289e4ab3f90fe908f922197f4e64f6b401f9eb2f` (HUD); cada payload bate com o binário testado. Snapshot GoReleaser: tar `8fc86f6a435a28259e1d7029870f0669858ced731235b38680ea42c76e159598`, DEB `f5c4932defd2745d6a256ee66478eb8eda2214089b16ffec1d2484a7ded1a83f`; checksums passaram e a instalação/remoção isolada do DEB passou. Evidências: [build](logs/gdk-popup-fix-build-artifacts.txt), [pacotes nativos](logs/native-packages-gdk-fix-validation.txt), [snapshot e instalação](logs/distribution-gdk-fix-install-remove-rerun.txt).
- **HUD GDK FIX-012:** causa reproduzida em `WebviewWindow.SetSize → IsFullscreen → gdk_window_get_state(NULL)` ao dimensionar o popover lazy antes da primeira exposição. O HUD agora exibe a janela para realizá-la antes de aplicar geometria/foco e a oculta se a geometria falhar. Regressões unitárias e race passaram; três interações nativas abrir/fechar passaram sem critical GDK. Evidências: [causa](logs/gdk-critical-root-cause-before-fix.txt), [log nativo](logs/native-hud-gdk-fix-open-close-xwayland.txt), [captura de um NEX](screenshots/native-hud-gdk-fix-mascot-xwayland.png).
- **Desktop nativo:** após FIX-010, o Desktop recompilado abriu em XWayland com janela Wails visível em 1280×800 e a tela inicial renderizada; [captura real](screenshots/native-desktop-mascot-fix-ready-xwayland.png), [log](logs/native-desktop-mascot-fix-xwayland-ready-final.txt). O cartão de ambiente ainda estava em “Identificando seu ambiente...” após 8 s, portanto esse estado não foi aprovado como pronto. O build anterior e os tarballs Desktop/HUD do workflow CI também iniciaram no GNOME 49/XWayland. Revalidação do tar do overlay `dac0d00b8410f70ed02c8b4c324780349655fed8f7e580d1944da1ec3eaecefa`: o HUD foi localizado pelo PID/título X11 real `nexus-hud`, aguardado em 224×158 e capturado com um único NEX; o encerramento retornou 0. A primeira tentativa tinha detector por título incorreto e geometria prematura, e foi repetida corretamente. O Desktop desse overlay abriu em 1280×800 e detectou seis provedores; o HUD empacotado abriu em 224×158 com um único NEX. No HUD atual, ArrowRight moveu a janela e a posição persistiu após reinício. Em Wayland direto, a navegação AT-SPI do seletor passou; entrada de teclado física, captura de pixels/alpha e transparência continuam sem evidência suficiente. A VM não tem aceleração 3D utilizável.
- **Terminal dividido:** corrigida a sobreposição de nomes PTY longos; E2E passou em sete geometrias e toolbars compactas. Em sessão nativa, comandos separados, Unicode, cancelamento e `stty size` passaram; ANSI ficou inconclusivo e persistência split após reinício sem troca de imagem segue pendente. A substituição do binário no mesmo caminho marcou runtimes antigos `STALE` por divergência de fingerprint; essa reconexão de atualização continua BLOCKED.
- **Anexação de runtime:** FIX-011 separou “Anexar Terminal” de criação de shell. No Desktop atual, o mesmo runtime permaneceu RUNNING, o total ficou em quatro runtimes/PTYs e o prompt existente apareceu após anexar; [relatório e capturas](logs/runtime-attach-validation.txt). Desktop binário SHA-256 `e4f46da9af49adb51bbdce91b5c78b91de0d852e89446dc6d2a9f8cf088b41d4`; [build](logs/runtime-attach-desktop-build.log).
- **Importação de sessões:** no Desktop do overlay final, a UI importou 4 sessões (Codex, Gemini, AGY e OpenCode), com 0 já autenticadas, 1 sem sessão, 1 sem suporte e 0 falhas. Após reinício do mesmo binário/perfil isolado, a UI mostrou 4 provedores com conta. O perfil temporário foi removido. Capturas e hashes em [smoke nativo](logs/native-final-overlay-smoke.txt).
- **Runtime:** uma Mission DIRECT real com Codex terminou `COMPLETED_VERIFIED`; AGY e OpenCode responderam `READY` via Nexus. Gemini chegou ao serviço, que bloqueou a conta por elegibilidade individual (`IneligibleTierError`). O HUD empacotado passou por amostra ociosa de 60 s e crash/recovery sem órfãos; soak prolongado e crash durante tarefa ativa seguem pendentes. Roteamento completo, continuidade, failover e Skills ainda não estão homologados.
- **Distribuição:** GoReleaser v2.18 criou tar.gz/DEB beta.10 do overlay atual; checksums passaram e os payloads coincidem. O DEB foi instalado, consultado, executado e removido com `dpkg --force-not-root` em raiz temporária; a instalação do host não foi tocada. Os tarballs Desktop/HUD iniciaram em XWayland. Snapshot local não é assinado nem publicado; a versão oficial continua beta.10, sem release estável. Evidências: [snapshot](logs/goreleaser-cursor-status-snapshot.log) e [instalação isolada](logs/distribution-current-overlay-install-remove.log).
- **Logs:** a busca heurística anterior examinou 65 arquivos sem encontrar padrões comuns de segredo. Ela não substitui uma auditoria completa.

## Correção final da detecção de provedores

O endpoint de prontidão usava um timeout de 2 s para sondar CLIs. Uma sondagem local de Gemini levou cerca de 2,2 s enquanto a tela inicial e o diagnóstico do sistema carregavam em paralelo, e a UI ocultava uma conta importada. O limite foi ampliado para 5 s e a regressão de 2,5 s passou. Com o binário Desktop final (`043ed2ba9fbd60ff2587c495e1db33070a893ed788fbd8a0360f32af74116ba4`), importação e reinício nativos mostraram quatro contas. O [manifesto](logs/source-provenance-linux-provider-readiness-final.txt) identifica as fontes testadas.

## Por que a release continua NO-GO

A branch ainda declara beta.10 e não há release estável `v1.0.0` nem assets Linux oficiais assinados. A árvore segue sem commit congelado. Permanecem sem homologação o roteamento completo, continuidade/failover, Skills em sessão, atenção real no HUD, importação Cursor, teclado/foco/movimento global e captura alpha em Wayland direto, updater/rollback assinado e reanexação de runtimes antigos após substituir o binário. O runtime atual pode ser anexado sem criar outro shell; a incompatibilidade entre fingerprints produz estado `STALE` e uma mensagem genérica na interface. A instalação isolada do DEB passou; a movimentação por teclado em XWayland também. Gemini está bloqueado por elegibilidade do serviço.

Não foi observado defeito P0 nos gates executados. O gate de release estável falha em P1. Há validações P1 e P2 bloqueadas ou não executadas. Esta decisão cobre Linux AMD64 apenas.

## Evidências

- [Matriz completa atualizada](TEST_MATRIX.md)
- [Resumo JSON](LINUX_HOMOLOGATION_SUMMARY.json)
- [Ambiente](ENVIRONMENT.md)
- [Mascote NEX único](MASCOT_VALIDATION.md)
- [Snapshot GoReleaser após FIX-007](logs/goreleaser-final-gemini-overlay.txt)
- [HUD Linux](HUD_VALIDATION.md)
- [Runtime de agentes](RUNTIME_VALIDATION.md)
- [Distribuição](DISTRIBUTION_VALIDATION.md)
- [Correções](FIXES_APPLIED.md)
- [Validação do terminal dividido após a correção](logs/terminal-split-native-after-fix-validation.txt) e [regressão do rótulo](logs/terminal-split-overflow-browser-regression.log)
- [Correção e validação do botão Anexar Terminal](logs/runtime-attach-validation.txt), [captura do terminal reaberto](screenshots/terminal-runtime-attach-current-build-xwayland.png) e [contagem de runtimes após anexar](screenshots/terminal-runtime-count-after-attach-xwayland.png)
- [Proveniência atual após FIX-012](logs/source-provenance-gdk-fix.txt) (`516b706a2c3cdca1a6d76200833a785186063350fa4d41db4494fcb67bdb7f7a`, 1.404 fontes; rechecagem sem divergências)
- [Portal de documentação e contrato do mascote](PR61_DOCS_SITE_VALIDATION.md)
- [Manifesto atual das fontes FIX-009](logs/source-provenance-cursor-status-fix.txt) (`dac0d00b8410f70ed02c8b4c324780349655fed8f7e580d1944da1ec3eaecefa`)
- [Manifesto final das fontes FIX-008](logs/source-provenance-linux-xwayland-position-fix.txt)
- [Build e dependências do overlay FIX-008](logs/linux-xwayland-position-fix-build.txt), [movimentação e persistência nativas](logs/native-xwayland-keyboard-position-fix.txt), [Desktop final](logs/native-desktop-xwayland-position-fix.txt), [tarballs Desktop/HUD](logs/linux-native-packages-xwayland-position-fix.txt), [snapshot CLI/DEB](logs/goreleaser-xwayland-position-fix.txt) e [smokes reais de provedores](logs/live-provider-smokes-final.txt)
- [Hashes dos binários de base](logs/current-binary-sha256.txt), [build final Linux AMD64](logs/provider-readiness-final-linux-build.log) e [versão CLI final](logs/provider-readiness-final-version.json)
- [Hashes das capturas atuais](logs/current-native-screenshots-sha256.txt)
- [Desktop nativo](screenshots/native-nexus-desktop-final-xwayland.png)
- [HUD com o mascote NEX](screenshots/native-nexus-hud-single-mascot-xwayland.png)
- [Importação de sessões pela interface nativa](RUNTIME_VALIDATION.md#importação-pela-interface-nativa) e [revalidação final de provedores](logs/provider-readiness-recheck-xwayland.txt)
- [Build e versão embutida](logs/build-meta.txt) e [versão JSON](logs/nexus-version.json)
- [Quality completo](logs/quality-full.log)
- [Mission Direct sanitizada](logs/runtime-direct-mission-summary.json)
- [Pacotes anteriores](logs/package-verification.json) e [pacotes do overlay final](logs/goreleaser-final-package-validation.log)

## Reprodução

Na raiz do repositório:

```bash
GOCACHE=/tmp/nexus-linux-100-2026-10-25/gocache-native make quality-full
cd web
bun run vitest run src/features/hud/GuardianMascot.test.tsx src/features/hud/GuardianModeSelector.test.tsx src/features/hud/hudPresentation.test.ts src/features/hud/companionModel.test.ts src/features/hud/guardianModel.test.ts
```

O snapshot do overlay final está em `/tmp/nexus-linux-100-2026-10-25-goreleaser-final/dist`; DEB instalado e removido em `/tmp/nexus-linux-100-2026-10-25-goreleaser-final/dist/dpkg-root`. O snapshot histórico anterior está em `/tmp/nexus-linux-100-2026-10-25/goreleaser-current`. Nenhuma tag, release ou publicação foi feita.
