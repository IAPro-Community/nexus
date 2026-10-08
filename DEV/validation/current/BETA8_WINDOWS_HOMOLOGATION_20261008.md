# Homologação Windows — Nexus 1.0.0-beta.8 — 2026-10-08

STATUS: PASS (gates automatizados locais Windows AMD64)
RELEASE_STATUS: NO-GO para tag/distribuição até fechar os gates externos e manuais
BRANCH: `fix/windows-beta8-homologation-20261008`
CODE_SHA: `474dfcecdd82aab0d0a2d6d1f36fd26d0d1b1f8b`
GATE_SHA: `7d2ef8aab669f88f22d09a86ec6442f761d0f7b5`
VERSION: `1.0.0-beta.8`
BASE_RELEASE: `origin/release/v1.0.0` em `2abbf31a770162e845e708695fbe397d02a5374e`
ENVIRONMENT: Windows AMD64, Go 1.25.14, Bun 1.3.9, Node 22.14.0
GATE_STARTED_AT: `2026-10-08T18:12:39Z`
GATE_COMPLETED_AT: `2026-10-08T18:23:20Z`

## Resultado

`scripts/homologate-beta.ps1 -ExpectedSha 7d2ef8aa` terminou com código 0 no SHA exato `7d2ef8aa`. O código validado é `474dfcec`; o commit posterior `7d2ef8aa` contém somente evidência visual gerada no código validado. Não foram executados gates Linux.

- PASS — parser PowerShell e `docs-verify` (178 arquivos Markdown e 33 ativos/arquivos requeridos).
- PASS — `bun install --cwd web --frozen-lockfile` e quality web completa: format, typecheck, ESLint, Stylelint, allowlist, build e verify; 83 arquivos Vitest e 499 testes passaram.
- PASS — `go fmt`, `go vet`, `golangci-lint` (0 issues), `govulncheck` (nenhuma vulnerabilidade) e `go test -count=1 ./...`; regressões de control/runtime, terminal e projeção OpenCode também passaram.
- PASS — oito suítes browser E2E, mais verificações de acessibilidade Axe e visual; asserções de layout do Project Hub passaram no viewport mínimo do Windows.
- PASS — builds CLI, Desktop e HUD e smokes nativos isolados incluídos no gate.
- PASS — capturador documental em checkout limpo no `CODE_SHA`; 10 superfícies sintéticas e manifesto associado ao mesmo SHA. Screenshot e quadros revisados mostram caminhos sintéticos, sem o perfil do host.

Os testes Go que exigem permissões POSIX ou criação de symlink sem privilégios foram marcados como skip pelo próprio teste no Windows NTFS. Não houve falha de teste.

## Pendências de distribuição

O PASS acima certifica os gates automatizados locais Windows. Não é GO de publicação. `MANUAL_CHECKS.md` na evidência lista os smokes interativos de provider/Mission/Voice ainda pendentes; também faltam CI Windows remoto exact-SHA, build e instalação do NSIS final, Authenticode e provisionamento da chave privada de atualização. A branch não foi integrada diretamente à release e nenhuma tag foi criada.

Evidência bruta local: `DEV/validation/local-beta/1.0.0-beta.8/7d2ef8aab669f88f22d09a86ec6442f761d0f7b5/windows-AMD64/` (diretório local ignorado pelo Git).

## Auditoria de branches remotas

No momento da consulta, `origin/release/v1.0.0` apontava para `2abbf31a`. A branch `fix/beta8-manual-opaque-quota` (`edcd2ab`) ainda tem uma mudança exclusiva sobre o tratamento de quota `UNSUPPORTED`; ela não deve ser integrada sem decidir o contrato e alinhar seus testes. A branch Beta 7 KIVER (`d14154d2`) não tem trabalho novo a portar. Esta branch Windows beta.8 é nova e deve ser vinculada à release por PR/revisão após o push.
