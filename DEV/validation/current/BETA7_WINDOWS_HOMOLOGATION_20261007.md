# Homologação Windows AMD64 — Nexus 1.0.0-beta.7

LAST_FROZEN_GIT_SHA: 52f35c04181b85379219ab55fc3192c63a5440a8
BRANCH: release/v1.0.0
LIVE_REMOTE_SHA: CONFIRMED (origin/release/v1.0.0 = 52f35c04181b85379219ab55fc3192c63a5440a8 via git ls-remote)
VERSION: 1.0.0-beta.7
REQUESTED_WINDOWS_SHA: d14154d2490dfdd3966c7fddc990a1d9127edd39
CURRENT_WORKTREE_BASE_SHA: 52f35c04181b85379219ab55fc3192c63a5440a8
CURRENT_WORKTREE_STATE: DIRTY
PLATFORM: Windows AMD64
WINDOWS_LOCAL_GATES: PASS_WITH_LIMITATION
OFFICIAL_RELEASE: NO-GO
WINDOWS_DESKTOP_RETEST: PASS_RUNTIME_VISUAL_UNVERIFIED
WINDOWS_NATIVE_CI: NOT_REVALIDATED (unauthenticated Actions API returned 404 for run 37666126714)
BETA7_TAG: ABSENT (live origin has no refs/tags/v1.0.0-beta.7; local tag absent)
GITHUB_RELEASE: NOT_REVALIDATED
UPDATED_AT: 2026-10-07

## Veredito

Os gates Go/Web/E2E/build passaram em worktree Windows baseado no HEAD 52f35c04181b85379219ab55fc3192c63a5440a8, com alterações locais sem commit; isto não é certificação exact-SHA das alterações. VERSION contém exatamente 1.0.0-beta.7. O smoke local atualizado também passou na criação e estabilidade da janela Wails/WebView2 por oito segundos com perfil isolado; renderização visível não foi verificada.

A publicação oficial permanece NO-GO. O run nativo Windows 37666126714 foi observado como queued e o último inventário conhecido não tinha runner Windows compatível; o estado remoto atual não pôde ser revalidado porque gh não está instalado e github.com falha na resolução DNS. Não há certificado/private key de code signing, signtool.exe no PATH, variáveis de assinatura ou NEXUS_UPDATE_PUBLIC_KEY. Os executáveis locais são NotSigned e foram construídos sem trust root de atualização. A tag local v1.0.0-beta.7 está ausente; tag e GitHub Release remotas não foram revalidadas.

## Correções Windows enviadas

- 9a23e9ef: funções de remoção do PATH e busca de substring passaram a usar o escopo uninstaller do NSIS, conforme exigido pelo compilador.
- 52f35c04: o teste do contrato de release foi atualizado para exigir os nomes un.RemoveFromUserPath e Call un.RemoveFromUserPath.
- A primeira suíte completa no commit 9a23e9ef encontrou somente a expectativa antiga no teste estático; o teste do pacote passou após o ajuste, e a suíte completa passou no SHA 52f35c04.

## Gates locais Windows — SHA 52f35c04

- PASS — go fmt ./...
- PASS — go vet ./...
- PASS — govulncheck ./... (nenhuma vulnerabilidade encontrada)
- PASS — golangci-lint run ./...
- PASS — go test -count=1 ./...
- PASS — bun install --cwd web --frozen-lockfile
- PASS — bun run --cwd web quality:full com Node 22.14.0 e Bun 1.3.9; 82 arquivos e 487 testes unitários, mais build de produção.
- PASS — oito suítes browser E2E: test:e2e, test:e2e:composer, test:e2e:surface, test:e2e:capacity, test:e2e:hud, test:e2e:composer-spine, test:e2e:composer-vague-spine e test:e2e:composer-themes.
- PASS — builds Windows AMD64 de CLI, Desktop e HUD com commit 52f35c04 embutido.
- PASS — CLI version informa 1.0.0-beta.7/windows/amd64 e doctor --json gerou JSON válido; o resumo de evidência não contém dados de credenciais.
- PASS — Desktop e HUD permaneceram ativos por oito segundos cada em smokes com diretórios de dados isolados.

## Instalador NSIS

A fonte NSIS revisada foi compilada com NSIS 3.13 durante a validação do ajuste do uninstaller. O smoke local isolado instalou CLI/Desktop/HUD, executou version e doctor --json, removeu os arquivos e o registro de uninstall. O shortcut foi redirecionado para o diretório de instalação para evitar efeito no Desktop real. O SHA 52f35c04 alterou apenas uma expectativa de teste Go depois dessa validação; o arquivo nexus.nsi permaneceu idêntico. O compilador portátil não está mais disponível nesta sessão, portanto a compilação não foi repetida após o commit de teste.

O smoke observou que o texto do valor HKCU\Environment\Path foi restaurado integralmente, mas seu tipo foi normalizado de REG_SZ para REG_EXPAND_SZ. Essa diferença deve ser avaliada antes de declarar o ciclo de instalação totalmente sem efeitos residuais.

## Artefatos locais

CLI, Desktop e HUD foram compilados para windows/amd64 no SHA 52f35c04. Os executáveis retornaram Authenticode NotSigned e foram compilados sem update trust root; servem apenas para validação.

| Artefato | Bytes | SHA-256 |
| --- | ---: | --- |
| nexus-cli_1.0.0-beta.7_windows_amd64.exe | 24460288 | 4c12471b66891a8d9af281d4cbbfc4bd9d05e4391893c37cccf9be5d2b2f9f87 |
| nexus-desktop_1.0.0-beta.7_windows_amd64.exe | 27064832 | 090522e3861c9a951b1f88c6fc07ee675b45365ce6f4fa9b98ffe049bc9a22c6 |
| nexus-hud_1.0.0-beta.7_windows_amd64.exe | 27088896 | eff6751859bf19a2cf60442bf90319f42ccc6b494a8234f18766b425755ba17a |

A tabela é espelhada em CHECKSUMS.md e SHA256SUMS.txt nos artefatos externos desta sessão.

## Bloqueios externos de fechamento

1. Disponibilizar runner self-hosted Windows AMD64 com labels Windows, X64 e nexus-windows; o run 37666126714 está aguardando esse runner.
2. Configurar certificado e secrets de assinatura no ambiente native-signing, além da chave pública de atualização exigida pelo build.
3. Reexecutar a certificação nativa exact-SHA, incluindo assinatura Authenticode e smoke do instalador assinado.
4. Revisar a normalização do tipo do valor Path observada no uninstaller.

Escopo desta continuação: Windows. Nenhum gate Linux foi executado nesta rodada.


## Aditamento — correção local de preservação do Path — 2026-10-07

O smoke inicial havia encontrado normalização do tipo do registro de `Path`. A correção foi aplicada localmente em `packaging/windows/nexus.nsi` e `internal/release/installer_test.go`, sem commit. O HEAD e `origin/release/v1.0.0` continuam em `52f35c04181b85379219ab55fc3192c63a5440a8`; portanto esta alteração ainda não pertence a um SHA homologado e congelado.

**Evidência Windows:** NSIS portátil 3.13 compilou uma fixture isolada. Em 5/5 casos, instalação e desinstalação retornaram código 0 e preservaram a existência, o conteúdo e o tipo do Path. Os casos cobrem `REG_SZ` e `REG_EXPAND_SZ`, valores não vazios e vazios, além de valor ausente. Foi usado payload fictício; o Path real do usuário e o Desktop real não foram alterados.

- PASS — `go test -count=1 ./internal/release` no checkout real.
- PASS — suíte Go completa, `go vet` e `golangci-lint` no worktree isolado de `52f35c04` com o mesmo patch.
- PASS — `git diff --check`.
- NÃO EXECUTADO — rebuild do instalador final com os binários reais depois desta correção.
- BLOQUEADO — workflow Windows exact-SHA sem runner; assinatura Authenticode e chave pública de atualização não configuradas.

A homologação local do SHA-base permanece registrada acima. O patch local não autoriza tag ou distribuição; ele precisa ser integrado e validado em um novo SHA imutável. Nenhum gate Linux foi executado.

## Aditamento — correção de origem Wails e ProjectHub — 2026-10-07

A imagem anexada reproduziu dois defeitos no Project Hub Windows: invalid origin ficava em uma coluna estreita e o botão Adicionar Projeto quebrava em duas linhas. A causa funcional era a política não permitir o Host exato wails.localhost usado pelo WebView Wails v3 com o Core embutido. A correção permite somente esse Host reservado quando a origem desktop reconhecida está presente; hosts lookalike e origens externas continuam rejeitados.

O ProjectHub passou a usar ProjectHub.module.scss. O erro ocupa as colunas do formulário, é anunciado como alerta e o botão não quebra o rótulo. A captura E2E em 775x483 confirma o resultado e está nos artefatos Windows da sessão.

- PASS — go fmt ./..., go vet ./..., govulncheck ./..., golangci-lint run ./... e go test -count=1 ./... no checkout Windows atualizado.
- PASS — bun install --cwd web --frozen-lockfile.
- PASS — quality:full com Bun 1.3.9 e Node 24.19.0, incluindo Stylelint, 82 arquivos/487 testes Vitest e build. A primeira execução com Node 18.12.1 falhou ao carregar a sintaxe JSON import do Stylelint; com Node 24 o gate passou sem alteração de dependências.
- PASS — oito suítes browser E2E no worktree Windows isolado com as mesmas alterações; após o mock 403, criação válida também passou.
- PASS — builds CLI/Desktop/HUD locais. CLI reporta 1.0.0-beta.7 e 52f35c04-dirty. SHA-256: CLI fa5912eb40f7889fda0bb7f4a1a2cc2c935585c1f8c513878de79ad1ce55a1d0; Desktop 2991a5d92440b2c931cc19b5e9565e0139ecf74c898a108d1030394e57b1a1b3; HUD d0698a8fe1447347b694cca40cf9b815874227c19514994990a5d131314a86cc. Todos NotSigned e sem trust root.
- PASS — smoke de oito segundos do HUD; ambiente WebView2 criado.
- INCONCLUSIVO — o Desktop encerrou com código 0 antes de oito segundos também no launcher atual; os binários de baseline repetiram o comportamento sem erro nos logs. Falta smoke em sessão Windows interativa.
- PASS — git diff --check nos arquivos alterados da correção.
- BLOQUEADO — workflow Windows exact-SHA 37666126714 sem agente Windows; certificado/secrets Authenticode e NEXUS_UPDATE_PUBLIC_KEY ausentes.
- NO TAG / NO PUBLISH — HEAD é o SHA-base 52f35c04181b85379219ab55fc3192c63a5440a8; mudanças locais não commitadas e sem certificação imutável. Artefatos não são distribuição.
- Escopo Windows-only; nenhum gate Linux executado.

## Aditamento — limite de desktop interativo no host Codex — 2026-10-07

A investigação do smoke Desktop confirmou a limitação do ambiente de execução: a sessão PowerShell atual está no SessionId 1, em `WinSta0` / desktop de thread `Default`, mas a chamada Windows `OpenInputDesktop` não retornou um handle para o desktop de entrada (GetLastError retornado: 0). Portanto, este processo não tem evidência de acesso ao desktop interativo necessário para observar a janela Wails. O encerramento do binário com código 0 neste launcher permanece INCONCLUSIVO; o teste deve ser repetido na sessão interativa Windows do host KIVER.

Essa observação descreve o ambiente Codex e não demonstra defeito funcional no Desktop. O veredito de release continua NO-GO até a captura nativa confirmar a janela aberta e estável no mesmo SHA congelado. Nenhum gate Linux foi executado.
## Aditamento — smoke nativo Desktop e controller WebView2 — 2026-10-07

O binário local nexus-desktop.exe (SHA-256 2991a5d92440b2c931cc19b5e9565e0139ecf74c898a108d1030394e57b1a1b3, Authenticode NotSigned) foi executado com APPDATA, NEXUS_DATA_DIR e AI_MANAGER_DATA_DIR isolados nos artefatos da sessão. Em 8 segundos contínuos, o processo permaneceu ativo e a enumeração Win32 encontrou a janela Wails IAPro Nexus com o filho WebView2 Chrome_WidgetWin_0. O smoke terminou sem processos WebView2 residuais.

Com o APPDATA padrão deste host, uma execução anterior falhou na criação do controller com 0x800700AA (The requested resource is in use). O isolamento do perfil removeu essa interferência no smoke; não alterei o perfil padrão do usuário. O processo foi iniciado oculto, e OpenInputDesktop não forneceu handle, portanto renderização visível e interação humana continuam sem comprovação neste host.

Foi adicionado scripts/smoke-desktop-windows.ps1 ao workflow .github/workflows/ci.yml. A etapa usa APPDATA e dados Nexus isolados, exige a janela Wails e Chrome_WidgetWin_0 durante 8 segundos e grava evidência. O script passou no host Windows e ci.yml foi analisado como YAML válido. Essa alteração permanece local e suja; ainda não pertence a um SHA exact-SHA homologado.

O checkout segue em release/v1.0.0 no HEAD/base 52f35c04181b85379219ab55fc3192c63a5440a8, com VERSION 1.0.0-beta.7 e sem tag local v1.0.0-beta.7. O rastreamento local de origin também aponta para 52f35c04, mas não consegui confirmar o estado vivo do remoto: gh não está instalado e git ls-remote falhou por DNS de github.com. O estado atual do run 37666126714 e da publicação remota permanece não revalidado. Nenhum gate Linux foi executado.

## Continuidade Windows — perfil WebView2 e fechamento de smokes — 2026-10-07

Base do checkout: release/v1.0.0 em 52f35c04181b85379219ab55fc3192c63a5440a8; VERSION=1.0.0-beta.7. A árvore de trabalho contém alterações locais e não é um SHA imutável homologado.

PASS — após os ajustes de perfil e smoke, go fmt ./..., go vet ./..., govulncheck ./..., golangci-lint run ./... e go test -count=1 ./... no Windows AMD64. O teste desktop também confirmou que WindowsWebviewUserDataPath usa LOCALAPPDATA.

PASS — workflow ci.yml validado como YAML e scripts/smoke-desktop-windows.ps1 validado pelo parser PowerShell. O workflow constrói binários de smoke com IDs Wails separados; os IDs de produção permanecem com seus valores originais. Os smokes exigem janela/controlador WebView2 e diretório de perfil efetivamente criado.

PASS — Desktop Wails IAPro Nexus e filho Chrome_WidgetWin_0 permaneceram ativos por 9 segundos; perfil isolado criado em LOCALAPPDATA\IAPro\Nexus\desktop\WebView2. HUD permaneceu ativo por 8 segundos e criou LOCALAPPDATA\IAPro\Nexus\hud\WebView2. Artefatos de smoke foram construídos com IDs exclusivos de teste, não são binários de distribuição.

A execução local com o ID de instância de produção terminou com código 0. O caminho Wails para uma segunda instância encerra com o código configurado, mas o proprietário do mutex não pôde ser inspecionado nesta sessão; nenhum processo do usuário foi encerrado. O binário de smoke com ID temporário isolado confirmou o caminho nativo.

Auditoria de branches locais/refspecs em cache: fix/beta7-windows-KIVER-20261007 está em d14154d2490dfdd3966c7fddc990a1d9127edd39 e é ancestral de release/v1.0.0 em 52f35c04181b85379219ab55fc3192c63a5440a8. A release contém cinco commits posteriores; a branch fix tem zero commits exclusivos. Não há merge pendente. Os refs origin locais não foram atualizados ao vivo.

A release continua NO-GO: HEAD não é o SHA d14154d2 informado no pedido original, mudanças não foram commitadas, estado remoto/CI não foi revalidado, assinatura/trust root e rebuild final do instalador seguem pendentes. Nenhum gate Linux foi executado nesta continuação.
## Auditoria ao vivo de branches beta7 — 2026-10-07

Consulta ao vivo confirmou origin/release/v1.0.0 em 52f35c04181b85379219ab55fc3192c63a5440a8 e ausência da tag v1.0.0-beta.7 no remoto. A branch fix/beta7-windows-KIVER-20261007 (d14154d2490dfdd3966c7fddc990a1d9127edd39) é ancestral da release e não precisa ser mesclada.

Auditoria de branches encontrou commits Beta 7 ainda fora da release: fix/beta7-codex-terminal-protocol-p0 (4 commits de protocolo terminal/web); fix/beta7-terminal-trace-diagnostics (24 commits de coalescência, privacidade de traces, testes e documentação); test/beta7-terminal-final-boundary-regression (3 commits de testes). git cherry não encontrou patches equivalentes. fix/beta7-windows-autopilot-20261001 tem um commit apenas documental e deve ser comparado com a evidência KIVER atual. Esses candidatos requerem triagem de escopo antes do fechamento; nenhum foi mesclado.

A branch fix/v1.0.0-gap-closure-20260930 tem 38 commits próprios de escopo amplo e requer decisão separada; fix/beta8-manual-opaque-quota é Beta 8 e está fora do escopo. O worktree tem 31 caminhos rastreados com diff de conteúdo e 10 arquivos não rastreados, medidos por git diff e git ls-files (git status contém ruído de stat), portanto integração e nova homologação devem ocorrer em checkout limpo. O veredito permanece NO-GO. Nenhum gate Linux foi executado nesta continuação.

## Continuidade final Windows — Project Hub 775×483 e artefatos beta.7 — 2026-10-07

A captura do usuário foi reproduzida no Chromium Windows. autoFocus escondia o título ao rolar o painel e o cartão de criação excedia o viewport. A correção remove o foco automático, mantém rolagem interna e compacta o layout somente em alturas até 600px. As descrições dos modelos agora são localizadas em inglês, português e espanhol. O E2E verifica que título, formulário com invalid origin e cabeçalho dos modelos cabem em 775×483; os três cartões permanecem alcançáveis pela rolagem.

- PASS — bun install --cwd web --frozen-lockfile, Bun 1.3.9.
- PASS — bun run --cwd web quality:full, Node 22.14.0; 82 arquivos/487 testes Vitest, lint, typecheck e build.
- PASS — oito suítes Playwright na árvore final, incluindo o viewport 775×483.
- PASS — builds Windows AMD64 CLI/Desktop/HUD; CLI version e doctor --json válidos.
- PASS — smoke Desktop: janela Wails IAPro Nexus e filho Chrome_WidgetWin_0 durante 9s; HUD durante 8s; IDs e perfis WebView2 isolados.
- PASS — checksums em outputs/windows-final-artifacts-52f35c04-dirty-20261007/CHECKSUMS.md. Todos os artefatos estão Authenticode NotSigned.
- EVIDÊNCIA VISUAL — outputs/windows-e2e-layout-52f35c04-dirty-20261007/e2e_project_hub_error_775x483.png.

O checkout continua em release/v1.0.0 no SHA 52f35c04181b85379219ab55fc3192c63a5440a8, não no d14154d2490dfdd3966c7fddc990a1d9127edd39 solicitado. O worktree está sujo, portanto os gates acima não certificam SHA imutável. VERSION segue 1.0.0-beta.7. O remoto vivo permanece em 52f35c04; a tag Beta 7 está ausente. Authenticode, update trust root, instalador final e CI exact-SHA continuam sem fechamento. A consulta não autenticada do run 37666126714 retornou 404; seu status atual não está confirmado.

A auditoria ao vivo não encontrou branch Beta 7 nova. A branch KIVER está integrada como ancestral. Quatro branches existentes de terminal/testes mantêm commits exclusivos fora da release para triagem; nenhuma foi mesclada. Nenhum gate Linux foi executado.
## Aditamento — branches Beta 7 e reteste direcionado — 2026-10-07

A consulta ao vivo via git ls-remote confirmou que as refs Beta 7 permanecem as mesmas da rodada anterior; nenhum ramo Beta 7 novo foi criado. A branch KIVER d14154d2 está integrada como ancestral de release/v1.0.0 em 52f35c04. As correções de transporte ConPTY, filtro de rajadas de navegação, trace redigido e as regressões raw-attach CSI/SS3 e privacidade constam da árvore atual. As três branches de terminal/testes não têm conteúdo relevante pendente; a branch Windows autopilot é apenas documental e mais antiga. Não há merge/cherry-pick necessário.

- PASS — go test -count=1 ./internal/control/host ./internal/runtime em Windows AMD64.
- PASS — bun run --cwd web vitest run src/nexus/terminalInput.test.ts; 28 testes.
- O primeiro intento sob o sandbox bloqueou IPC/alteração do diretório de trabalho; a reexecução autorizada no checkout passou. Isso não exigiu alteração de código.
- NO-GO mantido para distribuição: checkout dirty baseado em 52f35c04, sem SHA congelado e com pendências externas de CI/instalador/assinatura.

## Aditamento — build do instalador e smoke isolado com payload real — 2026-10-07

- PASS — NSIS 3.13 compilou a cópia exata do nexus.nsi da árvore atual. O instalador unsigned tem 23.917.346 bytes e SHA-256 7f310bf94d318d339ccd332d4c2de6b924404f87574e3adfccdb914fa55f5cf4. SHA-256 do script: d581f0f86513b5d64568ce0bc1f54c65e41d4ab68b9b9a527db36562680dc789.
- PASS — variante de smoke com namespace de registry e destino de shortcut isolados, usando CLI/Desktop/HUD beta.7 reais: instalação e desinstalação retornaram 0; arquivos de payload presentes; CLI version beta.7; doctor --json válido.
- PASS — Path de teste REG_SZ foi restaurado por valor e tipo; Path e Desktop reais permaneceram inalterados; diretório TEMP e chave de teste foram limpos.
- LIMITAÇÃO — a variante redireciona os alvos HKCU e shortcut para a fixture. O instalador exato compilado não foi executado no perfil do usuário. O instalador e os executáveis seguem NotSigned, sem trust root.
- EVIDÊNCIAS — outputs/windows-installer-validation-52f35c04-dirty-20261007/ e outputs/windows-installer-smoke-isolated-52f35c04-dirty-20261007/.
- NO-GO mantido até SHA imutável, CI Windows exact-SHA, assinatura/trust root e smoke do instalador exato em perfil descartável.
