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
- `git ls-remote` confirmou, antes da atualização desta documentação, `origin/release/v1.0.0` em `52f35c04181b85379219ab55fc3192c63a5440a8`; a tag `v1.0.0-beta.7` não estava presente. O inventário de GitHub Releases não foi consultado.
- Não foi criada branch nova nesta rodada. A branch existente `fix/beta7-windows-KIVER-20261007` em `d14154d2` é ancestral do candidato local; nenhum merge ou cherry-pick foi necessário.

**Conclusão:** os gates automatizados locais Windows passaram no SHA `d840797b`. Não marcar a release como GO para tag ou distribuição até que os bloqueios acima sejam resolvidos. Nenhuma conclusão Linux é feita por este relatório.
