# Homologação Windows — Nexus 1.0.0-beta.8 — 2026-10-08

STATUS: PASS (gates automatizados locais Windows AMD64 + build/ciclo de smoke NSIS unsigned)
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

## Auditoria de branches remotas

Auditoria remota em 2026-10-08: `origin/release/v1.0.0`=`2abbf31a770162e845e708695fbe397d02a5374e`; `origin/fix/windows-beta8-homologation-20261008`=`41a3ec0e42c36e3effda6139b98ad56f62fed925`. A branch Windows está vinculada à release pela PR draft #45, aberta e ainda sem merge/revisão: [PR #45](https://github.com/IAPro-Community/nexus/pull/45). `fix/beta8-manual-opaque-quota` (`edcd2ab`) não tem PR aberta e mantém alteração exclusiva de contrato de quota `UNSUPPORTED` com cobertura desalinhada; não está pronta para integrar. A branch Beta 7 KIVER (`d14154d2`) não tem trabalho novo a portar. O único runner Actions do repositório estava offline; nenhum runner online tem os labels Windows/X64/nexus-windows. Na PR, o check de certificação Windows está skipped.

Vínculo de revisão: [PR #45 — fix/windows-beta8-homologation-20261008 → release/v1.0.0](https://github.com/IAPro-Community/nexus/pull/45) (draft; sem merge).

## Reteste interativo Windows — ConPTY via Nexus — 2026-10-08

Smoke isolado com o binário `nexus.exe` beta.8, commit embutido `7d2ef8aa`, perfil Codex `windows-smoke` sem login e diretórios de dados temporários. `AI_REAL_HOME`, `USERPROFILE` e `HOME` apontaram para uma pasta vazia de teste, sem links para o perfil real do host.

- Com o padrão de release (sem `AI_CLI_ENABLE_CONPTY=1`), o runtime usou standard pipes e o Codex terminou com `stdin is not a terminal`. Isso confirma que sessões Codex supervisionadas precisam do ConPTY opt-in neste fluxo.
- Com `AI_CLI_ENABLE_CONPTY=1`, `TERM=xterm-256color` e `nexus codex:windows-smoke --no-daemon`, o Nexus anexou à sessão supervisionada e o Codex v0.160.1 exibiu a tela de boas-vindas/login. O runtime ficou ativo, recebeu uma tecla de seta e foi encerrado com Ctrl+C; o registro passou a `STOPPED` sem erro.
- Nenhum prompt foi enviado, login iniciado ou servidor compartilhado reiniciado. A saída capturada não confirmou visualmente a mudança de seleção após a seta, então a navegação semântica do TUI permanece parcialmente verificada.

O padrão ConPTY permanece opt-in por decisão anterior (`cc6400db`) para evitar regressões de teclado, mouse e CSI. Este reteste confirma o caminho de inicialização/attach com a opção habilitada; não substitui o smoke autenticado do provider nem os testes manuais de Mission e Voice. Nenhum código foi alterado nesta continuação e nenhum gate Linux foi executado.
