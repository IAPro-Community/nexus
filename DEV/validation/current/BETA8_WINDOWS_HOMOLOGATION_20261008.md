# Homologação Windows — Nexus 1.0.0-beta.8 — 2026-10-08

STATUS: PASS (gates automatizados locais Windows AMD64 no commit de código ab397c54)
RELEASE_STATUS: NO-GO para tag/distribuição até fechar os gates externos e manuais
BRANCH: `fix/windows-beta8-homologation-20261008`
CODE_SHA: `ab397c5449cc76dfd07b44e335f45de6762a94f4`
GATE_SHA: `ab397c5449cc76dfd07b44e335f45de6762a94f4`
VERSION: `1.0.0-beta.8`
BASE_RELEASE: `origin/release/v1.0.0` em `2abbf31a770162e845e708695fbe397d02a5374e`
ENVIRONMENT: Windows AMD64, Go 1.25.14, Bun 1.3.9, Node 24.19.0

## Revalidação Windows no commit ab397c54 — 2026-10-08

O commit `ab397c5449cc76dfd07b44e335f45de6762a94f4` passou pelo gate completo com `scripts/homologate-beta.ps1 -ExpectedSha ab397c54` em uma árvore limpa e no mesmo SHA. `VERSION` contém exatamente `1.0.0-beta.8`. Nenhum gate Linux foi executado.

- PASS — parser PowerShell e `docs-verify` (178 arquivos Markdown e 33 arquivos/ativos requeridos).
- PASS — `bun install --cwd web --frozen-lockfile` e `bun run --cwd web quality:full`; 83 arquivos Vitest e 499 testes passaram.
- PASS — `go fmt`, `go vet ./...`, `golangci-lint run ./...` (0 issues), `govulncheck ./...` (nenhuma vulnerabilidade) e `go test -count=1 ./...`.
- PASS — oito suítes Playwright browser, tarefas Axe/acessibilidade e visual; builds Windows AMD64 de CLI, Desktop e HUD.
- PASS — teste de integração Windows `TestCodexSessionHostDefaultsToConPTYAndForwardsArrowInput`: Codex recebe console interativo sem redirecionamento e a seta chega ao shell como `DownArrow`.
- PASS — smoke interativo com perfil isolado e sem `AI_CLI_ENABLE_CONPTY`: Codex v0.160.1 abriu a tela de boas-vindas/login via Nexus; a sessão permaneceu ativa e encerrou limpa com Ctrl+C, sem prompt ou login.
- PASS — smoke nativo Desktop no código `ab397c54`: build Windows com ID Wails de teste, janela `IAPro Nexus` e controller `Chrome_WidgetWin_0` ativos por 13 s; perfil WebView2 isolado e curto.
- PASS — smoke nativo HUD no código `ab397c54`: build Windows com ID Wails de teste, processo ativo por 8 s e perfil WebView2 isolado criado.
- PASS — regressão da mensagem `invalid origin` na criação do primeiro projeto: `TestWindowsWailsVirtualHostSupportsAuthenticatedProjectCreation` passou no HEAD `7cc03499`; bootstrap Wails retornou 200 e criação do projeto retornou 201 com autenticação/CSRF, enquanto hosts lookalike e origens externas retornaram 403.
- PASS — `go test -count=1 ./internal/control/originpolicy ./internal/control/web ./internal/desktop ./cmd/nexus-desktop ./cmd/nexus-hud` no Windows/HEAD `7cc03499`.
- PASS — no commit Windows `e7e021879cb3e948290eefaeabc3ec7dc51e04ae`, o teste de integração ConPTY repetido 10 vezes após endurecer o harness para manter o processo PowerShell aberto até o encerramento da sessão e preservar o resultado da tecla em arquivo temporário isolado: 10/10.
- PASS — `go test -count=1 ./...` no Windows/`e7e021879cb3e948290eefaeabc3ec7dc51e04ae`; todos os pacotes passaram (`GO_TEST_EXIT=0`), inclusive `internal/control/host`. `go vet ./...`, `golangci-lint run ./...` e `govulncheck ./...` também passaram nesse commit.
- O harness `scripts/smoke-desktop-windows.ps1` foi ajustado para compilar no PowerShell nativo do host: substituídas declarações C# inline `out` por variáveis explícitas. O smoke aprovado usou diretório de perfil curto, conforme o runner Windows.

### Correção homologada

No Windows, sessões supervisionadas do provider Codex selecionam ConPTY automaticamente. Os demais providers mantêm standard pipes por padrão; `AI_CLI_ENABLE_CONPTY=1` continua como override global e `AI_CLI_DISABLE_CONPTY=1` desativa o ConPTY. O capability report agora descreve o mecanismo escolhido por provider.

### Evidência e limites

Logs e `MANUAL_CHECKS.md` locais: `DEV/validation/local-beta/1.0.0-beta.8/ab397c5449cc76dfd07b44e335f45de6762a94f4/windows-AMD64/` (ignorados pelo Git). Evidências dos smokes Desktop/HUD ficam em `native-smokes/desktop-ci-contract/` e `native-smokes/hud-ci-contract/`. Os builds CLI/Desktop/HUD foram copiados para `outputs/windows-beta8-distribution-ab397c54-20261008/`; a CLI empacotada informa beta.8 e commit `ab397c54`. SHA-256:

| Artefato | SHA-256 | Authenticode |
| --- | --- | --- |
| `nexus-cli_1.0.0-beta.8_windows_amd64.exe` | `4af475402c3bc99dbb59f06316e849f4974a0c056b24230e01d5ffa12330e678` | NotSigned |
| `nexus-desktop_1.0.0-beta.8_windows_amd64.exe` | `2b49e2fae848698b4ffc1c4e56e281d31c071e38d7706286d931254587147a65` | NotSigned |
| `nexus-hud_1.0.0-beta.8_windows_amd64.exe` | `fec6b2fcc7d43dc423a1a116732cecc6ea9815150b316b904e8b2105748d5997` | NotSigned |

Os manifestos `MANIFEST.md` e `SHA256SUMS.txt` acompanham os binários. O instalador NSIS e seu smoke registrados em `outputs/windows-beta8-distribution-7d2ef8aa-20261008/` pertencem ao candidato anterior; não foi produzido nem testado um instalador no SHA atual.

CI Windows exact-SHA, assinatura Authenticode, trust root de atualização e smokes manuais restantes de provider/Mission/Voice continuam pendentes. O smoke Codex sem login valida TUI/ConPTY e não substitui autenticação nem esses fluxos. A PR draft #45 segue sem merge/revisão.

`doctor.json` detectou Gemini CLI instalado. A shim local usa Node 18.12.1 e falha nessa versão; o CLI reportou `0.38.2` quando executado com o Node portátil 22.14 em perfil isolado. Codex `0.160.1` e AGY `1.3.0` também responderam a `--version` em perfil isolado. Esses comandos só validam inicialização/versão, não autenticação nem prompts reais, portanto provider/Mission/Voice continuam pendentes. Os smokes Desktop/HUD verificam inicialização e criação do WebView2; não substituem inspeção visual da interface nem os fluxos manuais de show/hide, autostart e projeção.

### Auditoria de branches — 2026-10-08

Auditoria posterior ao push: `fix/windows-beta8-homologation-20261008@6fc85644` está publicada. `origin/release/v1.0.0` avançou para `95c7121d` (`VERSION=1.0.0-beta.10`), com 12 commits da release desde a base beta.8 `2abbf31a`; a candidata beta.8 não está alinhada para merge direto. A release atual tem commits posteriores de ConPTY e de correção dos avisos de lint, terminal tracing e teste localizado, mas os gates Windows exact-SHA em `95c7121d` não foram executados aqui. A PR #45 era registrada como draft; a consulta não autenticada atual retornou 404 e não confirma seu estado.

`origin/release/v1.0.0` permanece em `2abbf31a770162e845e708695fbe397d02a5374e`; a branch Windows beta.8 contém o código homologado em `ab397c54`, seguido de atualizações somente documentais, e continua ligada à release pela PR draft #45. A branch recém-criada `fix/beta7-windows-homologation-20261008` termina em `4a0bd992`, está baseada no ancestral `705d5b0c`, contém três commits exclusivos e ainda define `VERSION=1.0.0-beta.7`; a release avançou 12 commits desde essa base. Não é seguro vinculá-la diretamente à release beta.8. A branch Beta 7 KIVER em `d14154d2` não tem commits exclusivos. `fix/beta8-manual-opaque-quota` segue sem integração até alinhar contrato e testes.

Também permanecem branches de terminal com mudanças exclusivas frente à release: `fix/beta7-terminal-review-trace-lifecycle` (10 commits, 6 arquivos), `fix/beta7-terminal-trace-diagnostics` (24 commits, 13 arquivos), `fix/beta7-codex-terminal-protocol-p0` (4 commits, 4 arquivos) e `test/beta7-terminal-final-boundary-regression` (3 commits de teste). Elas alteram caminhos de host/runtime/Web que evoluíram na linha beta.8; exigem revisão e port seletivo com nova validação Windows, não merge em bloco. O inventário público de PRs não pôde ser confirmado neste host (`gh` ausente; API sem autenticação respondeu 404); o vínculo confirmado nesta auditoria é a PR #45 da branch beta.8.

## Histórico — gate anterior em 7d2ef8aa

O bloco abaixo registra a validação anterior e não substitui o estado atual em `ab397c54`.

### Resultado

`scripts/homologate-beta.ps1 -ExpectedSha 7d2ef8aa` terminou com código 0 no SHA exato `7d2ef8aa`. O código validado é `474dfcec`; o commit posterior `7d2ef8aa` contém somente evidência visual gerada no código validado. Não foram executados gates Linux.

- PASS — parser PowerShell e `docs-verify` (178 arquivos Markdown e 33 ativos/arquivos requeridos).
- PASS — `bun install --cwd web --frozen-lockfile` e quality web completa: format, typecheck, ESLint, Stylelint, allowlist, build e verify; 83 arquivos Vitest e 499 testes passaram.
- PASS — `go fmt`, `go vet`, `golangci-lint` (0 issues), `govulncheck` (nenhuma vulnerabilidade) e `go test -count=1 ./...`; regressões de control/runtime, terminal e projeção OpenCode também passaram.
- PASS — oito suítes browser E2E, mais verificações de acessibilidade Axe e visual; asserções de layout do Project Hub passaram no viewport mínimo do Windows.
- PASS — builds CLI, Desktop e HUD e smokes nativos isolados incluídos no gate.
- PASS — capturador documental em checkout limpo no `CODE_SHA`; 10 superfícies sintéticas e manifesto associado ao mesmo SHA. Screenshot e quadros revisados mostram caminhos sintéticos, sem o perfil do host.

Os testes Go que exigem permissões POSIX ou criação de symlink sem privilégios foram marcados como skip pelo próprio teste no Windows NTFS. Não houve falha de teste.

### Pendências de distribuição do candidato anterior

O PASS certifica os gates automatizados locais Windows e o ciclo de instalação/desinstalação NSIS sem assinatura; não é GO de publicação. `MANUAL_CHECKS.md` continua listando smokes interativos de provider/Mission/Voice pendentes. CI Windows exact-SHA, Authenticode, trust root de atualização, revisão/merge da PR e validação manual ainda impedem a distribuição. Nenhuma tag foi criada.

Evidência bruta local do gate: `DEV/validation/local-beta/1.0.0-beta.8/7d2ef8aab669f88f22d09a86ec6442f761d0f7b5/windows-AMD64/` (ignorada pelo Git).

## Continuidade Windows — build e smoke do instalador beta.8 — 2026-10-08

O checkout está em `fix/windows-beta8-homologation-20261008`. O código em `474dfcec` e o gate exact-SHA `7d2ef8aa` são idênticos quanto a código-fonte; as mudanças posteriores até `2506c067` são documentação e mídia documental. `VERSION` contém exatamente `1.0.0-beta.8`.

- PASS — builds Windows AMD64 de CLI, Desktop e HUD com Go 1.25.14, mais compilação de `packaging/windows/nexus.nsi` usando NSIS 3.13. O único aviso do compilador é `ai.exe` ausente; o script declara esse payload como opcional (`/nonfatal`).
- PASS — ciclo do instalador NSIS real: instalação silenciosa e desinstalação silenciosa retornaram 0; CLI instalado informou beta.8/commit `7d2ef8aa`; os três binários, atalho e registro foram criados; `doctor --json` foi válido.
- PASS — remoção do payload, diretório e registro; o instalador restaurou exatamente a existência, valor e tipo originais de `HKCU\Environment\Path`. O smoke também confirmou o estado original do host após cleanup.
- Os artefatos locais e unsigned, checksums e manifesto estão em `outputs/windows-beta8-distribution-7d2ef8aa-20261008/`. SHA-256: CLI `c2f8de7c730b898ec6d03112a00bbfd4d6f385342397a8b475646964cd473e2f`; Desktop `8a51672a1b3caecc5c3d0fd4aeb6a4f7811528c8dbebbcba1dde5914e4d906c5`; HUD `0b02d13e01bb7f5d3a1e5016bdc6721633d7a74ccc552b34985f20d6cbc192bc`; instalador `5eeb69e08e985dbdca64c27c5750de2b7fcb2429d3cc58198b3db1527ef25e4e`.
- LIMITAÇÃO — os quatro artefatos estão `NotSigned`; a chave trust root de atualização não está configurada. CI Windows exact-SHA não foi certificado: a PR #45 mostra o check Windows como skipped e o repositório não tem runner Windows online com os labels requeridos. O smoke interativo Codex alcançou a tela de boas-vindas com ConPTY opt-in, mas não autenticou provider nem executou Mission/Voice; esses smokes continuam pendentes.
- RELEASE NO-GO — não criar tag nem distribuir até fechar assinatura/trust root, certificação Windows nativa, smokes manuais e revisão/merge da PR #45.

### Auditoria de branches do candidato anterior

Auditoria remota em 2026-10-08: `origin/release/v1.0.0`=`2abbf31a770162e845e708695fbe397d02a5374e`; `origin/fix/windows-beta8-homologation-20261008`=`41a3ec0e42c36e3effda6139b98ad56f62fed925`. A branch Windows está vinculada à release pela PR draft #45, aberta e ainda sem merge/revisão: [PR #45](https://github.com/IAPro-Community/nexus/pull/45). `fix/beta8-manual-opaque-quota` (`edcd2ab`) não tem PR aberta e mantém alteração exclusiva de contrato de quota `UNSUPPORTED` com cobertura desalinhada; não está pronta para integrar. A branch Beta 7 KIVER (`d14154d2`) não tem trabalho novo a portar. O único runner Actions do repositório estava offline; nenhum runner online tem os labels Windows/X64/nexus-windows. Na PR, o check de certificação Windows está skipped.

Vínculo de revisão: [PR #45 — fix/windows-beta8-homologation-20261008 → release/v1.0.0](https://github.com/IAPro-Community/nexus/pull/45) (draft; sem merge).

### Reteste interativo Windows anterior — ConPTY opt-in — 2026-10-08

Smoke isolado com o binário `nexus.exe` beta.8, commit embutido `7d2ef8aa`, perfil Codex `windows-smoke` sem login e diretórios de dados temporários. `AI_REAL_HOME`, `USERPROFILE` e `HOME` apontaram para uma pasta vazia de teste, sem links para o perfil real do host.

- Com o padrão de release (sem `AI_CLI_ENABLE_CONPTY=1`), o runtime usou standard pipes e o Codex terminou com `stdin is not a terminal`. Isso confirma que sessões Codex supervisionadas precisam do ConPTY opt-in neste fluxo.
- Com `AI_CLI_ENABLE_CONPTY=1`, `TERM=xterm-256color` e `nexus codex:windows-smoke --no-daemon`, o Nexus anexou à sessão supervisionada e o Codex v0.160.1 exibiu a tela de boas-vindas/login. O runtime ficou ativo, recebeu uma tecla de seta e foi encerrado com Ctrl+C; o registro passou a `STOPPED` sem erro.
- Nenhum prompt foi enviado, login iniciado ou servidor compartilhado reiniciado. A saída capturada não confirmou visualmente a mudança de seleção após a seta, então a navegação semântica do TUI permanece parcialmente verificada.

O padrão ConPTY permanece opt-in por decisão anterior (`cc6400db`) para evitar regressões de teclado, mouse e CSI. Este reteste confirma o caminho de inicialização/attach com a opção habilitada; não substitui o smoke autenticado do provider nem os testes manuais de Mission e Voice. Nenhum código foi alterado nesta continuação e nenhum gate Linux foi executado.
