# Homologação Windows beta.10 — 2026-10-08/09

## Resultado

**PASS nos gates automatizados Windows AMD64** para o código no SHA `226d7b57a279a5c91421fb7dc790e778220417d4`, branch `fix/windows-beta10-homologation-20261008`, `VERSION=1.0.0-beta.10`.

Evidências locais: `DEV/validation/local-beta/1.0.0-beta.10/226d7b57a279a5c91421fb7dc790e778220417d4/windows-AMD64/`.

O comando oficial `scripts/homologate-beta.ps1 -ExpectedSha 226d7b57a279a5c91421fb7dc790e778220417d4` terminou com `PASS automated Windows beta homologation`. Ambiente: Windows AMD64, Go 1.27.2, Bun 1.3.9 e Node 22.14.0.

## Gates aprovados

- PowerShell syntax e verificação da documentação: 178 arquivos Markdown e 33 assets requeridos.
- `bun install --cwd web --frozen-lockfile` e `bun run --cwd web quality:full`; Vitest: 84 arquivos e 503 testes.
- Go format, `go vet ./...`, `golangci-lint run ./...` (zero findings), `govulncheck ./...` e `go test -count=1 ./...`.
- Regressões de control/runtime, ConPTY e Agent Interop.
- Oito suítes browser E2E, acessibilidade/axe e verificação visual.
- Builds CLI, Wails Desktop e HUD para Windows.

## Correção adicional validada

O diagnóstico de providers agora trata um CLI instalado cujo `--version` falha como `WARN`, com versão `unknown` e orientação para correção. Gemini não aparece mais como `PASS` a partir de uma saída de erro do wrapper Node do host. Os testes do detector, doctor e seis adapters passaram; nenhum prompt de modelo foi enviado.

## Limites e pendências

A primeira tentativa suplementar do Desktop usou o identificador de instância de produção e encerrou com código 0. A repetição no padrão do CI, com executável compilado do mesmo SHA e ID de instância de teste isolado, passou: a janela Wails IAPro Nexus e o filho WebView2 permaneceram ativos por 11 segundos e o perfil isolado foi criado. O smoke nativo do HUD também passou por oito segundos e criou o perfil WebView2 isolado. Os dois executáveis de smoke usam IDs de teste, não são binários de distribuição; os smokes confirmam inicialização e perfil, sem certificar renderização visível nem interação show/hide.

Evidências dos smokes nativos: `DEV/validation/local-beta/1.0.0-beta.10/226d7b57a279a5c91421fb7dc790e778220417d4/windows-AMD64/native-smoke-summary.txt`. A tentativa de navegação por setas abriu o TUI do Codex `0.160.1` com perfil temporário sem autenticação, mas o processo encerrou antes de aceitar teclas interativas; o resultado é inconclusivo e nenhum prompt foi enviado. Evidência: `DEV/validation/local-beta/1.0.0-beta.10/226d7b57a279a5c91421fb7dc790e778220417d4/windows-AMD64/codex-arrow-smoke.txt`. Continuam pendentes a navegação interativa ConPTY/Codex, prompts controlados de provider/Mission/custos/isolamento, projeção OpenCode, interação HUD show/hide/autostart e Voice; CI nativo no SHA final; Authenticode, chave de atualização, instalador e smoke do instalador. O veredito é **PASS automatizado local Windows / NO-GO para tag e distribuição**. Nenhum gate Linux foi executado.

## Branches e vínculo com a release

No fetch de 2026-10-09, `origin/release/v1.0.0` estava em `deca290b8ad2479a1cfdcc7f7b6aebf8fc944b6d`; `origin/integration/v1.0.0-beta10-pr45-pr46-20261008` estava em `9aad7444afb7581fd1238541ee0c6909211a3e78`. A release contém o merge beta10 PR45/PR46 em `f10fb19b`; a ponta da branch de integração ainda não é ancestral da release e difere da árvore remota em `DEV/validation/current/RELEASE_STATUS.md` e `web/scripts/e2e-composer-vague-execution-spine.mjs`. Não mesclar a branch inteira sem revisar essas duas diferenças.

A branch de homologação Windows também diverge da release: o merge-base é `2ac2dc69`, com três commits exclusivos da release e 109 exclusivos do candidato no fetch. O SHA homologado acima ainda não é o HEAD da release. A branch deve ser atualizada/reconciliada em PR e os gates Windows repetidos no SHA resultante antes de qualquer certificação canônica.