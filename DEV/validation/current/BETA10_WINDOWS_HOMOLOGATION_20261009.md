# Homologação Windows beta.10 — 2026-10-09

## Revalidação mais recente — Windows AMD64 — SHA `eedf31659b0245cf7690d556c32ee6e0040ae588`

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
