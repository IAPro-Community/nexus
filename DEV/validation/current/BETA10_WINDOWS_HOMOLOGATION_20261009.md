# Homologação Windows beta.10 — 2026-10-09

## CI Windows hosted — run `37898823857` — 2026-10-09

O run [37898823857](https://github.com/IAPro-Community/nexus/actions/runs/37898823857) executou no runner Windows efêmero e terminou **FAIL** no build web. A validação conferiu o source SHA exato `767be3b168a6d3ec7d223908e21812807486482f` e arquitetura Windows AMD64. O run usou a definição do workflow no commit `add42efdcb65d45ddfd7e5ab1f444346297305fd`.

- **PASS** — checkout no SHA-alvo, arquitetura AMD64, setup Go 1.27.2 e Bun 1.3.9; `bun install --frozen-lockfile` instalou 522 pacotes.
- **FAIL** — `bun run build` chama `node scripts/build.mjs`, mas o runner resolveu Node.js 18.12.1. O script falhou em `collectedStyles.toSorted is not a function` (`web/scripts/build.mjs:96`). O gate local do mesmo SHA usa Node 22.14.0 e passou; a causa é o runtime Node implícito do runner, não um delta de código entre plataformas.
- **SKIPPED** — Go vet/testes, regressões ConPTY/Named Pipe, testes Runtime/Web, signing/package, smokes Desktop/HUD, install/uninstall NSIS e upload de artefatos. O job encerrou antes da etapa que injeta os secrets de assinatura.
- **Correção preparada no workflow** — instalar `actions/setup-node@v7` com Node 22.14.0, desativar cache automático do package manager e validar `node --version` antes do build. O novo run no mesmo source SHA ainda é necessário para confirmar a correção.
- O runner v2.338.0 foi baixado com o SHA-256 publicado, registrado sem serviço e configurado como efêmero. Após um job, removeu automaticamente suas credenciais e o registro; nenhum gate Linux foi executado.

**Status da homologação hosted Windows: FAIL no pré-requisito de Node; correção aguardando novo run. Distribuição permanece NO-GO.**

## Continuidade Windows no SHA `767be3b168a6d3ec7d223908e21812807486482f` — 2026-10-09

O candidato permanece `1.0.0-beta.10`, distinto da solicitação beta.7 no SHA `d14154d2490dfdd3966c7fddc990a1d9127edd39`. A branch `fix/windows-beta10-homologation-20261008` passou novamente por `scripts/homologate-beta.ps1 -ExpectedSha 767be3b168a6d3ec7d223908e21812807486482f` em Windows AMD64.

- **PASS — homologação automatizada Windows no SHA exato.** Go 1.27.2, Bun 1.3.9 e Node 22.14.0. Formatação Go, `go vet ./...`, golangci-lint sem findings, govulncheck sem vulnerabilidades e `go test -count=1 -v ./...` passaram; skips condicionados a Windows/permissões foram os esperados. `bun install --cwd web --frozen-lockfile` e `bun run --cwd web quality:full` passaram sem mudança no lockfile, com 84 arquivos e 504 testes. As oito suítes browser E2E, Axe/visual e builds Windows AMD64 de CLI, Desktop e HUD passaram. `nexus version --json` confirmou beta.10, commit `767be3b1` e `windows/amd64`.
- **PASS — smoke nativo Desktop com ID singleton isolado.** O build diagnóstico usou `com.iapro.nexus.desktop.windows-smoke`; janela Wails `IAPro Nexus`, filho `Chrome_WidgetWin_0`, perfil WebView2 isolado (115 caracteres) e cleanup foram confirmados. SHA-256 do executável: `C8294C6CA4DAD84F77E16FF909D539065029A34FA6034E5C6DB2776FDF62374D`. Evidência local ignorada pelo Git em `DEV/validation/local-beta/1.0.0-beta.10/767be3b168a6d3ec7d223908e21812807486482f/windows-AMD64/desktop-native-767be3b1-unique-id-smoke.txt` e logs adjacentes.
- **PASS — smoke do instalador NSIS no payload real do SHA.** NSIS 3.13, `makensis` SHA-256 `4EC390D0F96F9728507FBC2F4E91311DD90BFC6C8DDFE2159357B3651AE151C0`; instalação silenciosa e desinstalação terminaram com código 0. CLI/Desktop/HUD instalados, execução `version --json`/`doctor --json`, atalho de teste e registro isolado passaram. O PATH e Desktop reais do usuário permaneceram idênticos; PATH de teste e diretório de instalação foram restaurados/removidos. SHA-256 do instalador diagnóstico não assinado: `46DF989E44C588682DA221128570160E4BFBA401922731646A981A030F6B1A9F`. Evidência local ignorada pelo Git: `DEV/validation/local-beta/1.0.0-beta.10/767be3b168a6d3ec7d223908e21812807486482f/windows-AMD64/nexus-beta10-767be3b1-nsis-install-smoke.json`.
- **PASS — autostart do Windows isolado.** `AutoStartManager` criou e removeu um valor com AppID GUID exclusivo em `HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run`; o comando e argumentos escapados foram conferidos, e o valor ficou ausente ao final. Nenhuma entrada real do Nexus foi usada. Evidência local ignorada pelo Git: `DEV/validation/local-beta/1.0.0-beta.10/767be3b168a6d3ec7d223908e21812807486482f/windows-AMD64/autostart-registry-smoke-767be3b1.json`.
- **PENDENTE — bandeja nativa.** O smoke de autostart não interagiu com o ícone/menu do tray; os testes de bandeja desta continuidade seguem inconclusivos.

Auditoria remota confirmada após o push documental: `origin/release/v1.0.0=deca290b8ad2479a1cfdcc7f7b6aebf8fc944b6d` e `origin/fix/windows-beta10-homologation-20261008=874a15707b1c951bdfb2c2fcf940b264593d1b03`; a release é ancestral da candidata, com 139 commits exclusivos no grafo. A primeira atualização remota falhou por DNS, mas `git ls-remote` posterior confirmou ambas as refs. A branch beta.10 é a candidata a PR contra `release/v1.0.0`; [comparar branches](https://github.com/IAPro-Community/nexus/compare/release/v1.0.0...fix/windows-beta10-homologation-20261008?expand=1). Não fazer push direto à release. A consulta não autenticada à API de PR retornou 404, então a existência/estado de PR não foi confirmada.

Branches beta.7 que não constavam no conjunto de refs local anterior foram comparadas com a candidata:

| Branch remota | Commits exclusivos no grafo vs. release | Avaliação para a candidata beta.10 |
| --- | ---: | --- |
| `fix/beta7-codex-terminal-protocol-p0` (`8d763ce0`) | 4 | O coalescer de CSI/SS3 e testes de atraso já estão na candidata. A heurística de descartar rajadas ambíguas `[A`/`[B` do patch web não deve ser transplantada sem reprodução, pois pode apagar texto legítimo digitado. Sem PR em bloco. |
| `fix/beta7-terminal-review-trace-lifecycle` (`d0bec9a4`) | 10 | Cancelamento de timers no fechamento, limite/validação de trace e testes correspondentes já aparecem na candidata. Sem PR em bloco. |
| `fix/beta7-terminal-trace-diagnostics` (`25e41d21`) | 24 | Diagnóstico mascarado, validação do frame não confiável, limite de arquivo e testes de privacidade já aparecem na candidata. Sem PR em bloco. |
| `test/beta7-terminal-final-boundary-regression` (`1c12f07a`) | 3 | O teste de raw attach CSI/SS3 e os testes de não vazamento ANSI-like já aparecem na candidata. Sem PR de teste separado. |
| `fix/beta7-windows-autopilot-20261001` (`ffe3571f`) | 1 | Apenas documentação antiga de beta.7. Sem código a integrar. |
| `fix/beta7-windows-homologation-20261008` (`4a0bd992`) | 3 | O patch de código `9105853a` já está representado na árvore beta.10; os dois commits restantes são status/capturas beta.7. Não mesclar em bloco. |
| `fix/beta7-windows-KIVER-20261007` (`d14154d2`) | 0 | Está contida na release remota; não requer PR. |

#### Outras branches ativas revisadas

| Branch | Diferença relevante para a candidata | Decisão de vinculação |
| --- | --- | --- |
| `fix/beta8-manual-opaque-quota` (`edcd2ab4`) | Um commit exclusivo altera `UNSUPPORTED` para exigir seleção explícita; a candidata ainda considera esse estado elegível para roteamento automático e tem teste que fixa essa regra. | Precisa de decisão funcional e revisão seletiva antes de PR; não misturar ao fechamento Windows, pois muda comportamento de roteamento e não é correção específica de Windows. |
| `fix/windows-beta8-homologation-20261008` (`e057ff2e`) | 16 commits à frente no grafo, mas nenhum patch exclusivo em relação à candidata (`git cherry`: zero). | Não precisa ser vinculada separadamente. |
| `audit/v1.0.0-beta9-maintainability-20261008` (`3a2a689b`) | Nove commits no grafo; nenhuma alteração de patch exclusiva em relação à candidata. | Não precisa de PR separado. |
| `integration/v1.0.0-beta10-pr45-pr46-20261008` (`9aad7444`) | Dois commits documentais não presentes por patch; branch antiga e divergente. | Não integrar em paralelo; usar a branch beta.10 atual como candidata única de PR. |
| `fix/v1.0.0-gap-closure-20260930` (`80738385`) | 38 commits exclusivos no grafo e 199 atrás da release; conjunto amplo de mudanças anteriores. | Não vincular em bloco; requer triagem seletiva fora deste fechamento Windows. |
| `release/v1.1.0` (`4a42f437`) | Linha de versão 1.1, divergente da 1.0.0. | Manter na sua própria linha de release. |

O teste de bandeja usou um HUD diagnóstico com identidade/perfil exclusivos. Core e WebView2 iniciaram. Uma captura confirmou o flyout de ícones ocultos aberto (`hud-tray-flyout-visible-767be3b1.png`), mas há dois ícones genéricos do Wails visualmente iguais e a UI Automation não expõe botões/tooltip que os distingam. `Shell_NotifyIconGetRect` apontou para o chevron quando o flyout estava fechado; mensagens Wails sintéticas demonstraram apenas que o callback registrado pode mostrar a janela, não que um clique físico funcione. Nesta sessão, `SendInput` também não abriu o flyout. Nenhum clique/menu físico foi provado. As capturas e JSONs locais estão em `DEV/validation/local-beta/1.0.0-beta.10/767be3b168a6d3ec7d223908e21812807486482f/windows-AMD64/` (ignorados pelo Git); o processo de teste foi encerrado pelo PID exato. O tray segue **não verificado**, sem conclusão de falha do produto.

**Veredito: PASS para os gates automatizados locais Windows e smokes isolados descritos; NO-GO para tag/distribuição.** Permanecem pendentes interação nativa do tray, CI Windows hosted no SHA integrado, smokes autenticados de provider/Mission/Voice e assinatura/update trust root. O smoke NSIS foi executado, mas o instalador não tem assinatura verificável. Esta continuidade não executou gates Linux e não altera a solicitação original beta.7/d14154d2.

## Continuidade no HEAD documental `45c12be785f3cfa9b503969c61a7fe98f8b220d3` — 2026-10-09

O candidato continua sendo `1.0.0-beta.10`, separado da solicitação beta.7 (`d14154d2490dfdd3966c7fddc990a1d9127edd39`). A branch `fix/windows-beta10-homologation-20261008` foi revalidada em Windows AMD64 no SHA exato `45c12be785f3cfa9b503969c61a7fe98f8b220d3` com `scripts/homologate-beta.ps1 -ExpectedSha 45c12be785f3cfa9b503969c61a7fe98f8b220d3`.

- **PASS — gates automatizados Windows no SHA exato.** Go 1.27.2, Bun 1.3.9 e Node 22.14.0. Formatação Go, `go vet ./...`, golangci-lint sem findings, govulncheck sem vulnerabilidades e `go test -count=1 ./...` passaram. A instalação Bun congelada e `quality:full` passaram com 84 arquivos e 504 testes. As oito suítes browser E2E, Axe/visual e builds Windows AMD64 de CLI, Desktop e HUD também passaram.
- **PASS — smoke nativo do Desktop com ID de teste isolado.** Um build diagnóstico do mesmo código-fonte substituiu somente `desktopSingleInstanceID` pelo ID exclusivo `com.iapro.nexus.desktop.windows-smoke`. A janela Wails `IAPro Nexus` e o filho WebView2 `Chrome_WidgetWin_0` foram detectados, o perfil isolado foi criado (171 caracteres) e o cleanup encerrou sem processos remanescentes. SHA-256 do executável diagnóstico: `1F2F70D2EEBEAEC85ADD2BAD0B3CBFB48B198267CF1CE628461A34CE0E987676`. Evidência local ignorada pelo Git: `DEV/validation/local-beta/1.0.0-beta.10/45c12be785f3cfa9b503969c61a7fe98f8b220d3/windows-AMD64/desktop-native-45c12be-unique-id-smoke.txt` e logs adjacentes `.stdout.log`/`.stderr.log`.
- **Causa do falso negativo com o ID de produção.** O host mantém uma instância beta.7 ativa (`nexus-desktop_1.0.0-beta.7_windows_amd64.exe`, PID 2572), que já possui `com.iapro.nexus.desktop`. O comportamento Wails de segunda instância encerra com código 0 antes de abrir outra janela; o processo existente foi preservado. O workflow Windows já cria um executável de smoke com ID separado; agora também passa esse ID explicitamente ao script. `scripts/smoke-desktop-windows.ps1` verifica o HWND singleton antes de iniciar e explica a colisão, sem encerrar o app ativo.
- O harness agora rejeita previamente perfis WebView2 com caminho completo acima de 240 caracteres. Um perfil de 287 caracteres havia reproduzido erro 0x8000ffff na criação do controller; com 171 caracteres, o smoke passou. Os preflights de ID singleton ocupado e caminho longo passaram sem iniciar outro app.
- O gate automatizado e este smoke não alteraram fontes do produto; o ajuste foi restrito ao harness Windows/CI e à documentação. Os logs integrais do gate permanecem no diretório local ignorado `DEV/validation/local-beta/1.0.0-beta.10/45c12be785f3cfa9b503969c61a7fe98f8b220d3/windows-AMD64/`.

**Veredito atual: NO-GO para tag, instalador e distribuição.** CI Windows hosted, tray/autostart, smokes autenticados, instalador e assinatura/trust root continuam pendentes. Nenhum gate Linux foi executado nesta continuidade.

## Continuidade Windows beta.10 — SHA `665103a06d94eda7011bb7d7abd299fc1ce2a330` — 2026-10-09

Esta rodada valida o candidato `1.0.0-beta.10` no Windows AMD64. É um candidato separado da beta.7/`d14154d2490dfdd3966c7fddc990a1d9127edd39`.

- **PASS — gate automatizado no SHA exato.** `scripts/homologate-beta.ps1 -ExpectedSha 665103a06d94eda7011bb7d7abd299fc1ce2a330` fechou entre `2026-10-09T05:06:48Z` e `2026-10-09T05:16:22Z`. `VERSION=1.0.0-beta.10`; Go 1.27.2, Bun 1.3.9 e Node 22.14.0.
- **PASS — Go e segurança.** Formatação, `go vet ./...`, golangci-lint sem findings, govulncheck sem vulnerabilidades e `go test -count=1 -v ./...` passaram. Casos condicionados a permissões/plataforma Windows foram registrados como skips esperados.
- **PASS — Web e E2E.** Bun frozen install sem alterações no lockfile; `quality:full` aprovou 84 arquivos e 504 testes. As oito suítes browser E2E, Axe/acessibilidade e validação visual passaram.
- **PASS — builds Windows AMD64.** CLI, Desktop Wails v3 e HUD compilaram. A CLI usada no smoke tem SHA-256 `4707F14000438D7EB89148FA3B34106B6618F6CFF10FBD3DD4B3EC5E56DCEB94`; `version --json` confirmou beta.10, commit `665103a0`, Windows/amd64 e Go 1.27.2.
- **PASS delimitado — AGY via ConPTY no mesmo SHA.** Comando `nexus.exe agy:windows-smoke`, perfil `PENDING_AUTH` isolado em APPDATA/LOCALAPPDATA e PTY nativo. A tela de onboarding do AGY CLI 1.3.2 recebeu três ciclos Up/Down/Left/Right; as teclas foram interpretadas pela TUI, sem sequências literais `[A`–`[D`. Ctrl+C encerrou com exit code 0. Runtime `agy-06GHY6TDP6A5HMYW3A3NDBQ6JR` ficou `STOPPED`; PIDs 12756 e 15676 não estavam ativos após o encerramento.
- Nenhum prompt foi submetido e nenhum Enter foi enviado. A TUI exibiu conteúdo de onboarding/exemplo; conversa autenticada, consumo de quota e execução de Mission não foram testados. A tentativa inicial dentro da sandbox expirou na confirmação IPC; o reteste nativo fora da sandbox passou com o mesmo binário e configuração isolada.
- Evidência local ignorada pelo Git: `DEV/validation/local-beta/1.0.0-beta.10/665103a06d94eda7011bb7d7abd299fc1ce2a330/windows-AMD64/agy-conpty-arrow-smoke-665103a0.txt`. Evidências completas do gate: a mesma pasta.
- Os commits de código `2f7a39ca` (ConPTY automático para AGY supervisionado) e `665103a0` (telemetria Go no perfil isolado de testes) foram enviados para `fix/windows-beta10-homologation-20261008`.

## Auditoria de branches remotas — 2026-10-09

Consulta de refs via Git: `origin/release/v1.0.0=deca290b8ad2479a1cfdcc7f7b6aebf8fc944b6d` (`VERSION=1.0.0-beta.10`) e `origin/fix/windows-beta10-homologation-20261008=3458826d1183951ffa0c75c9acaf80436a6bca0c (HEAD documental; fonte validada=665103a06d94eda7011bb7d7abd299fc1ce2a330)`. O SHA beta.10 parte diretamente da ponta atual da release e é o candidato a vincular por PR, após revisão.

- `fix/beta7-windows-KIVER-20261007=d14154d2490dfdd3966c7fddc990a1d9127edd39`: totalmente contida na release atual; zero commits exclusivos da branch. Não requer PR.
- `fix/beta7-windows-homologation-20261008=4a0bd992740bf3cc8d3d24d56c3f5f5afc6e6494`: permanece em `VERSION=1.0.0-beta.7`, diverge da release beta.10 e tem três commits exclusivos (`9105853a` de código e dois commits documentais). Não mesclar a branch inteira; qualquer aproveitamento do patch de código deve ser avaliado seletivamente no candidato beta.10.
- `integration/v1.0.0-beta10-pr45-pr46-20261008=9aad7444afb7581fd1238541ee0c6909211a3e78`: branch de integração anterior, baseada em histórico divergente; a auditoria por patch deixa dois commits documentais exclusivos em relação ao candidato beta.10 atual. Usar a branch de homologação beta.10 como candidata de PR, sem abrir uma segunda integração em bloco.

O estado de PR não pôde ser confirmado pela API sem autenticação; nenhum PR foi criado nesta continuação. Não houve push direto para `release/v1.0.0`.

**Limites e veredito:** PASS dos gates automatizados locais e do transporte interativo AGY/ConPTY no Windows para o SHA exato. CI Windows hosted no SHA integrado, smokes autenticados de provider/Mission/Voice, interação com tray e autostart do HUD, assinatura/trust root e instalador final ainda pendem. **NO-GO para tag/distribuição.** Nenhum gate Linux foi executado. A integração com `release/v1.0.0` segue por PR; não fazer push direto para release.

## Smoke nativo do controlador HUD show/hide — fonte 665103a0 — 2026-10-09

O smoke foi compilado com Go 1.27.2 para Windows AMD64, tags production e linker metadata beta.10. Um Go overlay em scratch acrescentou apenas chamadas temporizadas ao entrypoint diagnóstico; as fontes rastreadas do repositório não foram alteradas. O produto-fonte entre 665103a0 e o HEAD documental 3458826d é idêntico.

- PASS — controlador nativo: a janela iniciou visível; HUDController.HideHUD() resultou em IsVisible=false; HUDController.ShowHUD() resultou em IsVisible=true; o ciclo final de cleanup deixou a janela oculta. O processo diagnóstico terminou com exit code 0.
- CLEANUP — WebView2: três processos do perfil isolado ainda apareciam na amostra imediatamente após a saída; após dois segundos, não havia processo WebView2 usando o perfil exclusivo nem processo raiz do HUD ativo.
- ISOLAMENTO: APPDATA, LOCALAPPDATA, NEXUS_DATA_DIR e AI_MANAGER_DATA_DIR apontaram para um perfil temporário exclusivo. Nenhum getter/setter de autostart foi chamado; o Registro não foi objeto deste teste.
- LIMITE: este teste invocou os métodos nativos do controlador; não clicou no ícone/menu da bandeja. Interação do tray e inicialização automática continuam não verificadas. O stderr registrou um aviso Chromium de unregister da classe da janela (1412); a aplicação saiu com código 0 e os processos do perfil encerraram.
- Evidência local ignorada pelo Git: DEV/validation/local-beta/1.0.0-beta.10/665103a06d94eda7011bb7d7abd299fc1ce2a330/windows-AMD64/hud-show-hide-665103a0.txt e logs .stdout.log/.stderr.log. SHA-256 do binário diagnóstico: FE57D52859753C8ADEC61AA5D2F76B08ECE283A46A22D06F82F977EE7725B21C.

## Reteste físico do HUD — continuação no mesmo SHA — 2026-10-09

O reteste usa o mesmo código-fonte homologado `0782f87696ad3f28058d6532cb4e44dd25b08a7f`; somente a compilação diagnóstica adicionou instrumentação local. A captura anterior não provava invisibilidade do produto: o harness iniciou o processo com `-WindowStyle Hidden`, estado que também fazia o HWND reportar `IsWindowVisible=False`. Repeti a captura com inicialização normal e foco explícito da janela.

- **PASS — HUD transparente visível no desktop:** `window_visible=True`, `window_iconic=False`, `window_cloaked=0`, foreground PID igual ao PID do HUD e filho WebView2 presente. O frame capturado mostra o painel sobre a área de trabalho.
- **PASS — controle de expansão:** o clique no controle do HUD alterou o tamanho nativo de `280×48` para `280×520`; a captura expandida mostra o painel renderizado.
- **Evidências locais ignoradas pelo Git:** `DEV/validation/local-beta/1.0.0-beta.10/0782f87696ad3f28058d6532cb4e44dd25b08a7f/windows-AMD64/hud-native-visible-0782.png` (SHA-256 `6540df738dbbb20a310342252ebfd1fb95ac50f752f8bec0aa9a0c2ffe9cbc0e`), `hud-native-expanded-0782.png` (SHA-256 `00da95e8d09131eeb483595507778b2c55d6fc33ddd1ac9bb0c0882a5928325e`) e `hud-native-window-0782.txt`.
- **Limite mantido:** autostart não foi validado. O teste comprova apresentação e expansão do HUD nesta sessão interativa; não fecha CI hosted, smokes autenticados de providers/Mission/Voice, assinatura/trust root nem instalador final.

O reteste substitui a conclusão anterior de composição física inconclusiva para renderização visível do HUD no host; não altera o SHA do código homologado nem o NO-GO atual para distribuição.

## Smoke ConPTY de setas — continuação no mesmo SHA — 2026-10-09

CLI recompilada do worktree detached no SHA `0782f87696ad3f28058d6532cb4e44dd25b08a7f` com Go 1.27.2; `nexus --version` confirmou beta.10 e o SHA. SHA-256 do executável de smoke: `FA114CFD471F21EB0066D597F316D613E20738F764C02A3D28F7DC6F90CC0F99`. Nenhum fonte do produto foi alterado.

- **PASS delimitado — transporte de setas:** usei um perfil `codex:windows-smoke` temporário, isolado em `APPDATA`/`LOCALAPPDATA`, registrado com `--no-login` e estado `PENDING_AUTH`. O comando `nexus codex:windows-smoke --no-daemon` abriu o TUI Codex 0.160.1 via pseudo-console nativo. Enviei três ciclos de Up/Down/Left/Right; nenhuma sequência literal `[A`, `[B`, `[C` ou `[D` apareceu. Ctrl+C encerrou com exit code 0 e o runtime ficou `STOPPED`, sem processo Nexus/Codex remanescente.
- O TUI permaneceu na tela de entrada do Codex. Nenhum prompt foi digitado ou enviado; o resultado valida somente o transporte de teclas. Conversa autenticada, Mission, limites de custo e smokes AGY seguem pendentes até existir perfil autenticado/autorização para consumir quota.
- Evidência local ignorada pelo Git: `DEV/validation/local-beta/1.0.0-beta.10/0782f87696ad3f28058d6532cb4e44dd25b08a7f/windows-AMD64/conpty-arrow-smoke-0782.txt`.

## Revalidação mais recente — Windows AMD64 — SHA `0782f87696ad3f28058d6532cb4e44dd25b08a7f`

Esta rodada valida o candidato `1.0.0-beta.10` no Windows nativo. Ela não substitui a homologação beta.7 no SHA solicitado `d14154d2490dfdd3966c7fddc990a1d9127edd39`.

- **Gate oficial automatizado: PASS no SHA exato.** `.\scripts\homologate-beta.ps1 -ExpectedSha 0782f87696ad3f28058d6532cb4e44dd25b08a7f`, executado com árvore limpa no início, de `2026-10-09T03:52:13Z` a `2026-10-09T04:02:42Z`. `VERSION=1.0.0-beta.10`; Go 1.27.2, Bun 1.3.9 e Node 22.14.0.
- **Go e segurança: PASS.** Formatação, `go vet ./...`, golangci-lint sem findings, govulncheck sem vulnerabilidades e `go test -count=1 -v ./...`: 66 pacotes PASS, zero falhas e 18 skips condicionados à plataforma/permissões Windows.
- **Web e browser E2E: PASS.** Bun frozen install sem alterações no lockfile; `quality:full` com 84 arquivos e 504 testes PASS. O gate aprovou as oito suítes browser E2E, Axe/acessibilidade, validação visual e builds Windows AMD64 de CLI, Desktop e HUD.
- **Desktop nativo: inicialização e cleanup PASS.** O smoke repetido no SHA validado detectou a janela Wails `IAPro Nexus` e o filho `Chrome_WidgetWin_0` por 10 segundos e encerrou com zero processos WebView2 do app. Evidência: `desktop-native-0782f876.txt`.
- **Project Hub nativo: criação real PASS.** No WebView2 Desktop em `http://wails.localhost/`, o formulário submeteu o diretório temporário no viewport Windows de 775×483; o Core embutido respondeu HTTP 201, a aplicação abriu o overview do projeto e a captura `project-hub-native-submit-0782.png` mostra o workspace carregado. O processo diagnóstico usou DevTools somente em loopback e perfil isolado; terminou com `WEBVIEW2_LEFT=0`. A mensagem `invalid origin` da captura anterior não se reproduziu nessa origem Wails. Isso valida este fluxo nativo, sem afirmar que a captura original sob outra origem foi corrigida ou que outras origens devem ser aceitas.
- **Frontend gerado: PASS.** O relatório atualizado foi produzido pelo gate em `2026-10-09T03:52:52Z`: 10 gates PASS, zero falhas, 84 arquivos/504 testes. Os oito arquivos rastreados alterados pelo gate (relatórios, JSON e capturas visuais Composer) foram revisados e permanecem como evidência dessa execução.
- **Evidências completas:** `DEV/validation/local-beta/1.0.0-beta.10/0782f87696ad3f28058d6532cb4e44dd25b08a7f/windows-AMD64/` (logs locais ignorados pelo Git). O verificador registrou 178 Markdown e 33 assets/arquivos requeridos.
- **Escopo:** nenhum gate Linux foi executado. Os arquivos de evidência/documentação desta atualização não alteram o código validado no SHA acima.

**Veredito da execução original:** PASS dos gates automatizados Windows e do envio real pelo Project Hub Desktop no código `0782f876`. Na captura original, a composição física do HUD ficou inconclusiva; o reteste acima comprovou a apresentação visível e a expansão nativa. Permanecem pendentes CI Windows hosted no SHA integrado, autostart do HUD, smokes autenticados/interativos restantes, assinatura/update trust root e instalador final. **NO-GO para tag/distribuição.** Beta.10 é diferente da beta.7/d14154d2. A branch Windows deve ser vinculada à `release/v1.0.0` por PR, sem push direto à release.

## Revalidação anterior — Windows AMD64 — SHA `eedf31659b0245cf7690d556c32ee6e0040ae588`

Esta rodada é beta.10 e não substitui a homologação beta.7 no SHA solicitado `d14154d2490dfdd3966c7fddc990a1d9127edd39`.

- **Gate oficial automatizado: PASS no SHA exato.** `scripts/homologate-beta.ps1 -ExpectedSha eedf3165`, executado com árvore limpa no início, de `2026-10-09T03:18:53Z` a `03:29:02Z`. `VERSION=1.0.0-beta.10`; Go 1.27.2, Bun 1.3.9 e Node 22.23.3.
- **Qualidade Go e segurança: PASS.** `gofmt`, `go vet ./...`, `golangci-lint` (0 issues), `govulncheck` (sem vulnerabilidades) e `go test -count=1 -v ./...` (66 pacotes PASS; skips condicionais de plataforma/permissão registrados no log). Preflights de provider/Mission/terminal, control/runtime e Agent Interop/OpenCode também passaram.
- **Web: PASS.** Instalação Bun congelada; `quality:full` com 84 arquivos e 504 testes PASS; build/embed e relatórios passaram.
- **Browser E2E: PASS.** O gate concluiu as suítes browser, Axe/acessibilidade e visual, incluindo HUD, Composer, isolamento de superfícies, Capacity e temas.
- **Distribuíveis Windows: PASS de compilação.** CLI, Desktop Wails v3 e HUD compilados para Windows AMD64. Isto não substitui assinatura, instalador final ou CI Windows hosted.
- **Desktop nativo no SHA homologado: PASS de inicialização e cleanup.** O binário Desktop do gate (`FD1DFB3B…411E0486`) abriu a janela Wails `IAPro Nexus` e o filho `Chrome_WidgetWin_0` por quatro segundos em perfil isolado. Corrigi `scripts/smoke-desktop-windows.ps1` para enumerar somente descendentes WebView2 do processo do smoke via Toolhelp, sem depender de CIM; o reteste em Windows PowerShell 5.1 terminou com exit code 0. A verificação posterior encontrou zero processos `nexus-desktop`/`msedgewebview2`. Evidência: `DEV/validation/local-beta/1.0.0-beta.10/eedf31659b0245cf7690d556c32ee6e0040ae588/windows-AMD64/desktop-native-eedf3165-cleanup-ps51-final.txt`. A janela foi iniciada oculta, então isto não comprova apresentação visual nem submissão do Project Hub.
- **HUD nativo: PASS parcial delimitado.** O binário de homologação `002afb4a…685761dc` abriu uma janela Wails de 280×48 com filho WebView2; a API autenticada retornou HTTP 200, `enabled=true` e estado presente. Um build diagnóstico do mesmo SHA, com DevTools apenas em loopback, confirmou `http://wails.localhost/?nativeHud=1`, documento completo, tema dark e painel DOM 280×48. Captura renderizada pelo WebView2: `hud-webview2-eedf3165.png`, SHA-256 `b8a6a3ad4bce719ce875a05ebb8246f470266f7eaf414d75fb8d187cba169de9` (evidência local ignorada pelo Git). As tentativas de captura física por GDI `CopyFromScreen` com `CAPTUREBLT` e `PrintWindow` não mostraram o painel; o HWND estava visível/topmost, mas outro PID permaneceu em foreground. Como essas capturas não verificam de forma confiável a superfície DirectComposition/WebView2 neste host, a composição DWM final permanece **inconclusiva**; não declarar show/hide/autostart ou visual físico aprovados.
- **Evidências completas:** `DEV/validation/local-beta/1.0.0-beta.10/eedf31659b0245cf7690d556c32ee6e0040ae588/windows-AMD64/` (diretório local ignorado pelo Git). O gate registrou 178 Markdown e 33 assets/arquivos de documentação.
- **Escopo de plataforma:** nenhuma validação Linux foi executada.

**Veredito:** PASS dos gates automatizados locais Windows no SHA exato; renderização do DOM pelo WebView2 confirmada; composição visual física ainda inconclusiva. **NO-GO para tag/distribuição** até fechar a composição visual nativa, os smokes interativos/autenticados, CI Windows hosted no SHA integrado, assinatura/update trust root e instalador final. O candidato é beta.10, distinto da solicitação beta.7/d14154d2.

## Revalidação Windows anterior — SHA `3a09b5f0035cde09931b8f8eef1c783f49de47e0`

Esta evidência antecede o gate acima. `VERSION=1.0.0-beta.10`; não substitui a homologação beta.7 pedida originalmente.

- **Gate local Windows AMD64: PASS no SHA exato.** `scripts/homologate-beta.ps1 -ExpectedSha 3a09b5f0` fechou: PowerShell 5.1, docs-verify, Bun frozen + `quality:full` (84 arquivos/503 testes), Go fmt/vet/lint/security/testes (66 pacotes PASS), preflights, oito suítes browser E2E, Axe/visual e builds CLI/Desktop/HUD. Go 1.27.2, Bun 1.3.9, Node 22.14.0. Evidências: `DEV/validation/local-beta/1.0.0-beta.10/3a09b5f0035cde09931b8f8eef1c783f49de47e0/windows-AMD64/`.
- **Origin Wails e criação pelo servidor: PASS.** `TestWindowsWailsVirtualHostSupportsAuthenticatedProjectCreation` retornou HTTP 201 para `Host: wails.localhost` / `Origin: http://wails.localhost` e HTTP 403 para origins maliciosos. A consulta foi repetida fora da sandbox porque ela bloqueava localhost. A janela Desktop nativa carregou o Project Hub; a submissão do formulário pela UI não foi concluída nesta automação.
- **Desktop visível: PASS de carregamento.** A janela `IAPro Nexus` e `Chrome_WidgetWin_0` abriram em perfil limpo; a captura `desktop-hub-loaded.png` mostra o Project Hub renderizado. Evidência local ignorada pelo Git: `DEV/validation/local-beta/1.0.0-beta.10/3a09b5f0035cde09931b8f8eef1c783f49de47e0/windows-AMD64/`.
- **HUD nativo: inicialização/API PASS; projeção visual inconclusiva.** A janela `Nexus HUD` e WebView2 abriram; com `NEXUS_ENABLE_HUD=1`, a API autenticada no host Wails retornou HTTP 200, `enabled=true` e estado presente. A captura após 12 s ainda mostra uma superfície transparente sem o painel. Não contar show/hide, autostart ou projeção nativa como aprovados. O E2E browser do HUD passou.
- Os dois ajustes de homologação desta branch já foram commitados e enviados: `2a78056c` (BOM para parser Windows PowerShell 5.1) e `3a09b5f0` (preservar stdout/stderr e exit code real de Bun no PowerShell 5.1).

Refs remotas confirmadas em `git ls-remote`: `release/v1.0.0=deca290b8ad2479a1cfdcc7f7b6aebf8fc944b6d`, `fix/windows-beta10-homologation-20261008=3a09b5f0035cde09931b8f8eef1c783f49de47e0` e `integration/v1.0.0-beta10-pr45-pr46-20261008=9aad7444afb7581fd1238541ee0c6909211a3e78`. Nenhuma branch nova foi criada nesta continuação. A branch beta.10 de homologação precisa de PR para `release/v1.0.0`; não houve push direto à release. A tag remota `v1.0.0-beta.7` não apareceu na consulta.

**Distribuição: NO-GO.** Pendem confirmação visual/interativa do HUD nativo, submissão do Project Hub pela UI, smokes autenticados/interativos de providers/Mission/Voice, workflow hosted Windows no SHA integrado, assinatura/update trust root e instalador final. `gh` e ferramentas/segredos de assinatura não estão disponíveis. O SHA solicitado originalmente para beta.7 (`d14154d2`) e o checkout atual beta.10 são candidatos distintos. Nenhum gate Linux foi executado.

## Registro inicial — gate Windows no SHA `638cfb3b`

| Campo | Valor |
| --- | --- |
| GIT_SHA | `638cfb3bbb7aa9a647d6802964e7e4ab794cc786` |
| EVIDENCE_COMMIT | `4ab1457f5d4554cde2998f32b697f45196349654` |
| BRANCH | `fix/windows-beta10-homologation-20261008` |
| VERSION | `1.0.0-beta.10` |
| PLATFORM | Windows AMD64 |
| WORKTREE_FROZEN | YES — o gate iniciou com a árvore de código limpa neste SHA |

## Resultado

**PASS — homologação automatizada local Windows no SHA exato `638cfb3b`.** O comando `scripts/homologate-beta.ps1 -ExpectedSha 638cfb3bbb7aa9a647d6802964e7e4ab794cc786` terminou com `PASS automated Windows beta homologation`.

O remoto `origin/release/v1.0.0` estava em `deca290b8ad2479a1cfdcc7f7b6aebf8fc944b6d`, e esse commit é ancestral do SHA validado. A release remota ainda não aponta para o candidato homologado. Nenhum gate Linux foi executado nesta rodada.

Evidências locais: `DEV/validation/local-beta/1.0.0-beta.10/638cfb3bbb7aa9a647d6802964e7e4ab794cc786/windows-AMD64/` (diretório local ignorado pelo Git).

Ambiente do gate: Go 1.27.2, Bun 1.3.9 e Node 22.14.0 para Playwright. A execução fechou entre `2026-10-09T01:39:06Z` e `2026-10-09T01:51:48Z`.

## Gates aprovados

- Sintaxe PowerShell e documentação: 178 arquivos Markdown e 33 assets/arquivos requeridos.
- `bun install --cwd web --frozen-lockfile` e `bun run --cwd web quality:full`: 84 arquivos Vitest, 503 testes PASS.
- Formatação Go verificada com `gofmt -l` nos arquivos rastreados, `go vet ./...`, `golangci-lint` v2.14.0 (0 findings), `govulncheck` v1.7.0 (sem vulnerabilidades) e `go test -count=1 -v ./...`: 66 pacotes PASS, zero falhas e 18 skips de casos condicionados a plataforma ou permissão.
- Preflight de regressões provider/Mission/terminal, E2E de control/runtime e regressões Agent Interop/OpenCode: PASS.
- Oito suítes browser E2E, além dos gates Axe/acessibilidade e visual: PASS. A captura de erro “invalid origin” no Project Hub passou no viewport mínimo Windows; o placeholder do caminho segue o formato do sistema operacional.
- Builds Windows AMD64 da CLI, Desktop Wails v3 e HUD: PASS.

## Smokes nativos Desktop e HUD

Os smokes foram recompilados do mesmo SHA com IDs de instância de teste e diretórios temporários próprios.

- Desktop: PASS. Janela Wails `IAPro Nexus`, filho WebView2 `Chrome_WidgetWin_0` e perfil isolado permaneceram presentes por 11 segundos. SHA-256 do binário de smoke: `9e6be1b05452111fb1954250c305e6d52c6bba2ffacd9ce0dec05dda8cb6f032`.
- HUD: PASS. Processo permaneceu ativo por oito segundos e criou seu perfil WebView2 isolado. SHA-256 do binário de smoke: `41e1f0d0522a685c876b2530548ab907073c9d320d057378cf284e79226c28e1`.

Esses smokes confirmam inicialização e isolamento de perfil. Não certificam renderização visual da janela nem interação HUD show/hide/autostart. Os executáveis de smoke não são binários de distribuição.

## Providers e pendências Windows

`nexus doctor --json` classificou Gemini como `WARN` porque o wrapper local falha em `--version`; o detector não apresenta uma instalação sem versão verificável como `PASS`. O CLI Gemini do host está associado ao Node 18.12.1 do PATH padrão. Para o gate browser, Node 22.14.0 foi colocado à frente no PATH. Nenhum prompt de modelo foi enviado.

O teste manual anterior do TUI Codex com perfil sem autenticação terminou antes da navegação por setas; permanece **inconclusivo** e nenhum prompt foi enviado. As regressões automatizadas ConPTY/setas passaram no SHA validado.

Permanecem pendentes: navegação interativa ConPTY/Codex em sessão real, Mission com provider autenticado e validação de custo/isolamento, projeção OpenCode na UI, controles interativos HUD, Voice, captura visual Desktop, workflow Windows hosted no mesmo SHA, Authenticode, chave privada de atualização, build e smoke do instalador final. Este host não tem `gh`, `signtool.exe`, `makensis.exe` nem credenciais de assinatura/update configuradas.

**Veredito:** PASS para homologação automatizada local Windows do candidato beta.10; **NO-GO para tag, instalador assinado e distribuição** até a integração/revisão em `release/v1.0.0` e o fechamento dos gates operacionais pendentes.

## Auditoria de branches e vínculo com release

Consulta remota de 2026-10-09: `origin/release/v1.0.0=deca290b8ad2479a1cfdcc7f7b6aebf8fc944b6d`, `origin/fix/windows-beta10-homologation-20261008=e40791c01332cc6d0bc1ea39745e7064f581cf8d` e `origin/integration/v1.0.0-beta10-pr45-pr46-20261008=9aad7444afb7581fd1238541ee0c6909211a3e78`.

O candidato local inclui a ponta atual da release como ancestral (`0` commits exclusivos da release; `114` commits exclusivos do candidato no grafo). A branch beta.10 de homologação é a que deve ser vinculada à release por PR após este push: [comparar com release/v1.0.0](https://github.com/IAPro-Community/nexus/compare/release/v1.0.0...fix/windows-beta10-homologation-20261008?expand=1). Não fazer push direto para `release/v1.0.0`.

A branch `fix/beta7-windows-homologation-20261008` tem três commits exclusivos de histórico, mas o conteúdo de código do commit `9105853a` já está presente em forma equivalente na árvore beta.10 atual; o cherry-pick ficou vazio após preservar os testes mais recentes. Os outros dois commits são capturas/status beta.7 e não devem ser transplantados para os documentos beta.10. A branch beta.7 não precisa de PR de código separado.

A branch de integração beta.10 difere da árvore atual da release apenas em `DEV/validation/current/RELEASE_STATUS.md` e no harness `web/scripts/e2e-composer-vague-execution-spine.mjs`; não mesclar a branch inteira. A branch Windows beta.8 mantém `VERSION=1.0.0-beta.8` e é histórica. A branch `audit/v1.0.0-beta9-maintainability-20261008` tem commits exclusivos de auditoria/versionamento e uma árvore divergente da release, incluindo uma alteração de CI para separar Axe/visual. Não mesclar em bloco; qualquer cherry-pick deve ser avaliado em uma revisão de pipeline separada, fora deste candidato Windows.
