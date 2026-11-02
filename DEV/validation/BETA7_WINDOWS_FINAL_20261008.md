# Nexus 1.0.0-beta.7 — homologação Windows final

Data: 8 de outubro de 2026  
Plataforma: Windows AMD64 nativo  
Branch: `release/v1.0.0`  
SHA do código validado: `d840797bbd975a5c7edc54ebe23f3dfef56ff947`  
Versão: `1.0.0-beta.7`  
Go: `1.25.14` · Bun: `1.3.9` · Node para Playwright: `22.14.0`

## Veredito

**PASS nos gates automatizados locais Windows no SHA registrado.** O veredito de distribuição permanece **NO-GO**: o host não dispõe de CI nativo exact-SHA autenticado, assinatura Authenticode nem NSIS para gerar o instalador final; os smokes manuais de provider/Mission/Voice também não foram executados neste SHA.

Esta continuação foi Windows-only. Nenhum gate Linux foi executado.

## Integridade e identidade

- `git rev-parse HEAD`: `d840797bbd975a5c7edc54ebe23f3dfef56ff947`.
- `VERSION` e o binário CLI: `1.0.0-beta.7`.
- O arquivo versionado `UPDATE_PUBLIC_KEY` passou validação de formato; o build CLI/Desktop/HUD usou essa chave pública como trust root.
- `nexus version --json`: Windows AMD64, Go 1.25.14, commit `d840797b`.
- O worktree estava limpo ao início do homologador; a tabela e os relatórios guardam a identidade do código testado mesmo que a documentação de status receba um commit posterior.
- O candidato é descendente do SHA KIVER `d14154d2490dfdd3966c7fddc990a1d9127edd39`.

## Gates Windows

| Gate | Resultado | Evidência |
| --- | --- | --- |
| Sintaxe PowerShell | PASS | `powershell-syntax.log` — install, uninstall, homologador e refresh |
| Verificação documental | PASS | 177 Markdown e 33 assets/arquivos requeridos |
| Bun frozen install | PASS | 537 instalações verificadas, sem mudanças no lockfile |
| `bun run --cwd web quality:full` | PASS | 82 arquivos Vitest, 487 testes, build frontend |
| Go formatting | PASS | `go fmt ./...` e checagem `gofmt` dos arquivos Go rastreados; sem alterações |
| `go vet ./...` | PASS | `go-vet.log` |
| `golangci-lint run ./...` | PASS | versão pinned v2.12.2, zero issues |
| `govulncheck ./...` | PASS | versão pinned v1.7.0, nenhuma vulnerabilidade reportada |
| `go test -count=1 ./...` | PASS | execução verbose equivalente: 65 resultados de pacote PASS, zero linhas FAIL |
| Preflight Mission/terminal | PASS | `internal/app`, `internal/nexus/promotion`, `internal/runtime` |
| Control/runtime e Agent Interop | PASS | terminal/protocol/host/web, OpenCode projection, launcher e `internal/nexus`/store |
| Contrato da versão | PASS | `VERSION`, versão web e runtime Wails consistentes |
| Browser E2E | PASS | 8 suítes: hardening, composer, surface, capacity, HUD, composer spine, vague spine e themes |
| Acessibilidade e visual | PASS | Axe e verificação visual/headless passaram |
| Builds CLI/Desktop/HUD | PASS | executáveis Windows AMD64 compilados no candidato |
| Smoke nativo Desktop | PASS | janela Wails `IAPro Nexus`, filho `Chrome_WidgetWin_0`, perfil próprio; 9 s |
| Smoke nativo HUD | PASS | processo ativo por 8 s e perfil WebView2 próprio criado |

Logs e saídas: `DEV/validation/local-beta/1.0.0-beta.7/d840797bbd975a5c7edc54ebe23f3dfef56ff947/windows-AMD64/`.

## Binários e checksums

Os executáveis e ZIPs foram copiados para a pasta de evidências. Os executáveis são `NotSigned`; a assinatura Authenticode dos ZIPs não se aplica.

| Artefato | Bytes | SHA-256 | Authenticode |
| --- | ---: | --- | --- |
| `nexus.exe` | 24465408 | `5b8d640fdb3c502b6644a534617e1c18045fe2056e2d3d1d605c97745947de9c` | NotSigned |
| `nexus_Windows_x86_64.zip` | 9781727 | `a9b34d7194ee06c06c5ef209f509923c66544c75f21905f1f0a65f9242c342f6` | N/A |
| `nexus-desktop.exe` | 27071488 | `37324cb3bfb3f5658803383c35196dc394f0b7cf352207a49e2c773ae2f7750c` | NotSigned |
| `nexus-desktop_Windows_x86_64.zip` | 10757562 | `9bc1d0eb227fa249fcbb1440f242bb68e682912cbddb4cd0c2fb24c2baf4fb06` | N/A |
| `nexus-hud.exe` | 27095040 | `87796fe9cea5e88f65e06cee7f95f54d207961a6a59ff32c479b2f3ce34d00b2` | NotSigned |
| `nexus-hud_Windows_x86_64.zip` | 10767067 | `1286ea6741c4bf5652fa8835a311619fd7092a0e7486692cae61a539fd583e92` | N/A |

Tabela independente: `DEV/validation/local-beta/1.0.0-beta.7/d840797bbd975a5c7edc54ebe23f3dfef56ff947/windows-AMD64/distribution-artifacts/CHECKSUMS.md`.

## Ajustes para a homologação Windows

O teste completo havia travado em `TestResolveComposerUnknown` durante uma chamada Git. A inspeção da identidade de código agora limita cada subprocesso Git a cinco segundos e retorna identidade incompleta em timeout, sem afirmar estado limpo. O homologador também remove variáveis Git herdadas e mantém `TEMP` isolado num diretório curto; isso evita que fixtures longas ultrapassem o limite de caminho do Windows.

Uma execução inicial após o primeiro ajuste encontrou `Filename too long` porque `TEMP` estava abaixo da pasta profunda de evidências. Após mover `TEMP` para uma pasta curta no temp nativo do Windows, toda a suíte `go test -count=1 ./...` passou no SHA final.

## Limitações e bloqueios de distribuição

- `gh`, `signtool.exe` e `makensis.exe` não estão disponíveis; a execução do workflow nativo GitHub Actions, a assinatura Authenticode e o instalador exato não foram validados.
- Não há certificado/secrets Authenticode nem `NEXUS_UPDATE_PRIVATE_KEY`/`NEXUS_UPDATE_KEY_ID` neste host. Os executáveis e ZIPs são candidatos locais não assinados, não pacotes autorizados para distribuição.
- Os smokes interativos de Codex/AGY/OpenCode, Direct/Lead/Mission e Voice não foram executados neste SHA.
- O smoke Desktop verificou janela e controlador WebView2 em modo oculto; não houve captura visual do render numa sessão desktop interativa.
- `nexus doctor --json` gerou diagnóstico com alertas específicos deste host. O Gemini CLI instalado invocou Node 18.12.1 e falhou no parse; não se contabilizou provider smoke como aprovado.
- `git ls-remote` confirmou inicialmente `origin/release/v1.0.0` em `52f35c04181b85379219ab55fc3192c63a5440a8`; em seguida código e relatórios foram enviados por fast-forward. O primeiro HEAD remoto confirmado foi `f1928d0f75cf49a0968f248bd7dc9b8e54655dcb`, que contém o SHA validado como ancestral. A tag `v1.0.0-beta.7` continuou ausente. O inventário de GitHub Releases não foi consultado.
- Não foi criada branch nova nesta rodada. A branch existente `fix/beta7-windows-KIVER-20261007` em `d14154d2` é ancestral do candidato local; nenhum merge ou cherry-pick foi necessário.

**Conclusão:** os gates automatizados locais Windows passaram no SHA `d840797b`. O Actions API sem autenticação retornou 404 após o push, então não há confirmação de CI. Não marcar a release como GO para tag ou distribuição até que os bloqueios acima sejam resolvidos. Nenhuma conclusão Linux é feita por este relatório.
## Retestes Windows e auditoria de branches — 2026-10-08

O checkout estava em release/v1.0.0, HEAD e56c091f5038a8c972c339f47a960167077e45d8, VERSION=1.0.0-beta.7 e worktree limpo antes desta atualização documental. O SHA de código homologado d840797bbd975a5c7edc54ebe23f3dfef56ff947 é ancestral; a comparação do intervalo d840797b..e56c091f mostrou nove caminhos documentais e nenhum arquivo de código.

O teste direcionado web/scripts/e2e-hardening-verify.mjs passou no Windows com Node 24.19.0. No viewport 775x483, o alerta de 403 invalid origin ficou abaixo dos campos e ocupou a largura do formulário; “Adicionar Projeto” ficou numa linha; rolagem vertical e Axe passaram; após remover a interceptação, a criação legítima respondeu com sucesso. Todas as asserções da suíte hardening passaram. Evidência local: .tempmediaStorage/e2e_project_hub_error_775x483.png.

A primeira chamada via Bun parou antes da suíte porque o Node prioritário no PATH é 18.12.1 e Playwright requer Node 20+. A chamada seguinte usou diretamente Node 24.19.0 e passou, sem alterar o PATH global. A imagem enviada pelo usuário continua diferente da captura atual; isso sugere build ou ambiente distinto, mas não identifica a causa. O frame visual da janela Desktop em sessão interativa permanece sem verificação.

Tentativas sem prompt de nexus codex e nexus agy terminaram antes do provider com no enabled profiles configured. Nenhum prompt foi enviado. OpenCode não está instalado. Portanto os smokes autenticados de provider/Mission/Voice permanecem pendentes. O CLI do artefato continua reportando beta.7, Windows AMD64, commit d840797b; Authenticode está NotSigned.

As refs de branches foram comparadas localmente, sem integração. KIVER em d14154d2 é ancestral da release. A release contém a linha de correções de terminal pelos PRs #41–#43. fix/beta7-windows-autopilot tem apenas um commit documental exclusivo; fix/beta8-manual-opaque-quota e release/v1.1.0 são de outras linhas. fix/v1.0.0-gap-closure-20260930 tem 38 commits exclusivos e divergência ampla, sem gate beta.7 Windows que justifique merge em bloco. O refresh das refs remotas falhou porque github.com não resolveu por DNS; portanto esta auditoria cobre apenas as refs cacheadas.

O checkout ficou sem alterações de código. Atualizações documentais desta rodada são locais e não commitadas. CI exact-SHA, assinatura Authenticode, instalador final, smokes interativos e captura visual Desktop continuam bloqueios; manter NO-GO e sem tag. Nenhum gate Linux executado.

## Retomada Windows — correção de placeholder e captura nativa — 2026-10-08

### Estado do checkout

`release/v1.0.0` está em `e56c091f5038a8c972c339f47a960167077e45d8` local e remotamente; `VERSION` é exatamente `1.0.0-beta.7`. O HEAD contém nove caminhos documentais após o SHA de código homologado `d840797bbd975a5c7edc54ebe23f3dfef56ff947`. Esta retomada acrescentou cinco caminhos frontend ainda não commitados, então `WORKTREE_FROZEN: NO` e não se declara validação exact-SHA da árvore modificada.

### Correção de UX Windows

O Desktop nativo exibia `/home/user/my-repository` nos dois formulários de cadastro, inclusive no Windows. Os campos agora selecionam exemplos localizados para Windows, macOS e POSIX usando o user agent, sem alterar componentes visuais ou estilos. No Windows o Project Hub exibe `C:\Projetos\meu-repositorio`.

### Verificação no Windows

- PASS — teste unitário do seletor por user agent: 3/3.
- PASS — `bun run quality:full` com Node 24.19.0: allowlist, formatação, ESLint, Stylelint, TypeScript, 83 arquivos/490 testes e build.
- PASS — `bun run test:e2e`: viewport 775×483, 403 `invalid origin`, alerta na largura do formulário, botão em uma linha, rolagem, Axe e POST válido após remover a interceptação.
- PASS — build e smoke Desktop no estado de trabalho atual com `UniqueID` temporário e diretórios APPDATA/LOCALAPPDATA/dados isolados. Janela Wails e filho `Chrome_WidgetWin_0` foram vistos; o cleanup deixou zero processo do smoke e zero WebView2 ligado ao perfil temporário. SHA-256 do executável de teste: `49B629C43A58C5CCAFC5E4C93F3D50939F193382E9EF788E6F9C396F8C0F8D7A`.
- PASS — captura visual nativa em tema escuro confirmou o Project Hub e o placeholder Windows. O tamanho cliente mínimo da janela é 784×561; 775×483 segue coberto pelo Playwright.
- BLOCKED — `node scripts/docs-verify.mjs` não certifica o manifesto visual enquanto `ProjectHub.tsx`, `AddProjectModal.tsx` e `resources.ts` estiverem modificados sem commit; o gate exige SHA imutável.
- PASS — consulta viva de refs confirmou `origin/release/v1.0.0=e56c091f`; não apareceu branch Beta 7 nova que precise ser integrada. KIVER é ancestral. `protocol-p0`, `trace-diagnostics` e `final-boundary` têm respectivamente 4, 24 e 3 commits exclusivos no grafo, porém as alterações de código e cobertura pertinentes já estão na release. `windows-autopilot` tem um commit documental; beta8/v1.1 são linhas separadas; `gap-closure` tem 38 commits exclusivos e não deve ser mesclada em bloco. A tag beta.7 não foi encontrada no remoto.

A saída antecipada com código 0 observada ao testar o binário normal era colisão com uma instância interativa do usuário que mantém o ID singleton de produção. Ela foi identificada e não foi encerrada. O smoke isolado com ID temporário contornou a colisão e passou.

**Veredito:** validação local Windows passou nos escopos registrados, mas a distribuição continua **NO-GO**. Faltam CI Windows exact-SHA, assinatura Authenticode/trust root, instalador final e smokes manuais autenticados de provider/Mission/Voice. As mudanças desta retomada permanecem locais e sem commit/push. Nenhum gate Linux foi executado.

## Continuidade Windows no snapshot remoto 81dea07 — 2026-10-08

O ref de rastreamento local `origin/release/v1.0.0` aponta para `81dea07b3fba6d08fbf80c71c17c09518435aa64`; o checkout principal permanece em `e56c091f5038a8c972c339f47a960167077e45d8`, um commit atrás e com alterações locais preexistentes. `VERSION` contém exatamente `1.0.0-beta.7`. A validação foi executada em worktree temporário Windows AMD64 no mesmo SHA remoto, sem mudança em arquivos de produção.

**Gates Windows no SHA `81dea07b`:** `gofmt` (725 arquivos), `go vet ./...`, `golangci-lint run ./...`, `govulncheck ./...` e `go test -count=1 ./...` passaram. `bun install --cwd web --frozen-lockfile` passou. A execução bruta de `bun run quality:full` falhou no Prettier em `src/workspace/ptyLiveChrome.test.ts`; após formatar esse teste no worktree temporário, falhou uma asserção desatualizada em `src/workspace/taskbarHonesty.test.ts`. `quality:full` passou (82 arquivos, 488 testes e build) somente depois de duas alterações locais e não commitadas nesses arquivos de teste. Isso é diagnóstico e não converte o gate bruto do SHA em PASS.

As oito suítes Playwright, builds CLI/Desktop/HUD e smokes nativos Desktop/HUD passaram no worktree temporário; as suítes rodaram com as duas alterações locais de teste e geraram diferenças em evidências versionadas. Os executáveis de smoke usam IDs temporários e não são artefatos de distribuição.

**Auditoria de branches:** nas refs cacheadas, `fix/beta7-windows-KIVER-20261007` em `d14154d2` está 18 commits atrás e 0 à frente de `origin/release/v1.0.0`; não há integração pendente dessa branch. Nenhuma branch Beta 7 nova que exija vínculo apareceu no snapshot disponível. A consulta ao vivo falhou por resolução DNS de `github.com`, então esta conclusão não atesta branches criadas após a última atualização das refs.

**Veredito: NO-GO para tag ou distribuição.** O gate bruto `quality:full` ainda falha no SHA publicado; também permanecem os bloqueios previamente registrados de CI exact-SHA, assinatura/trust root e instalador Windows final. A tag local não existe; a tag remota e o inventário de GitHub Releases não puderam ser reconsultados nesta rodada. Nenhum gate Linux foi executado, conforme o foco Windows.
