# PR #45 + #46: conciliação controlada para Nexus 1.0 beta.10

Data: 2026-10-08
Destino: `release/v1.0.0` (beta.10). Este documento NÃO declara homologação.
Baseline: `5aa694d98ae223294739708e1b1adadd44028c4d`; após a criação da branch, a release incorporou `2ac2dc69d1a47e80fb57eec8542ba41f391703f2` (strings de título IA). As nove novas traduções foram reproduzidas nesta branch sem apagar o patch da PR #45.
Fonte PR #45: `e057ff2e0c0f86d2b5a671a17a9e20c9857f54fc`.
Fonte PR #46: `3a2a689ba677266071c82442b6361b7083a292a2`.

## O que foi portado de #45

A revisão verificou os hunks, preservou o contexto vigente e portou os arquivos operacionais de Go, TS, scripts e testes, sem sobrescrever os arquivos completos com uma versão beta.8 antiga:

- ConPTY por provedor; Codex com console virtual por padrão (opt-out via `AI_CLI_DISABLE_CONPTY=1`); seleção de backend e capacidades atualizadas; testes Windows para ConPTY, seta para baixo e sessão completa.
- Reuso de identidade/contexto do projeto no Composer para reduzir varreduras redundantes, com testes para a mesma identidade.
- Políticas e regressões de autenticação remota, localização, invitação, frontend e diagnóstico de terminal.
- `docs-capture.mjs` com redaction de caminhos de ambiente de teste em conteúdo DOM/valores/atributos antes de capturas e verificações negativas; maior segurança de evidências.
- Placeholders de caminho por sistema operacional com i18n e testes.
- `BETA8_WINDOWS_HOMOLOGATION_20261008.md` mantido como evidência HISTÓRICA, com o SHA da beta.8 original.

**Não transportados automaticamente:** `DEV/validation/FRONTEND_LATEST.md`, relatórios de status marcados beta.7/8, metadados `manifest.json` da captura antiga, imagens e vídeo gerados na PR #45, e notas históricas que poderiam sobrescrever a verdade da beta.10. O código de sanitização foi portado; as imagens devem ser recriadas e verificadas contra o SHA congelado da candidata atual.

## O que foi portado de #46

- `scripts/release-version.mjs` encapsula versão + arquivo web de forma testável e rollback dos conteúdos originais.
- `scripts/release-beta.mjs` utiliza a transação e diferencia compilação de instalação/publicação.
- Testes do contrato de incremento, consistência e restauração; CI inclui os testes de versionamento e verificação explícita do contrato Go.
- `Makefile release-dry-run` deixou de anunciar pronto para publicação usando um alvo inexistente.
- Auditoria de engenharia beta.10 foi preservada.

## Ajustes adicionais

- Extração da captura de Ctrl/Cmd + +/-/0 para `terminalZoomKey.ts` compartilhado entre `AgentTerminal.tsx` e `TerminalPane.tsx`, protegida por teste unitário da identificação de teclas; outros atalhos seguem byte-transparentes.
- `ProtocolScrubber` limita a retenção de candidato de envelope em 64 KiB, à semelhança do buffer Go, com teste de saída grande.
- Um único hardening E2E contempla também Axe e visual: removidas duas invocações idênticas e seus assertions redundantes em `ci.yml`.

## Evidências e riscos remanescentes

Status desta integração: **PENDING / NO-GO para merge incondicional, tag ou distribuição**.

- CI real exact-SHA, Go+frontend+Playwright e Windows nativo para o HEAD desta branch: não executados/sem resultado PASS neste trabalho. Runs conhecidos na fila ou cancelados por runners indisponíveis.
- PR #45 atesta Windows local beta.8, mas seus checks e binários não autorizam a beta.10.
- Codex/ConPTY: a política padrão foi alterada no Windows e **precisa** de smoke interativo Codex, testes de ConPTY e fallback `AI_CLI_DISABLE_CONPTY=1` no mesmo SHA antes do merge da release.
- Gerar capturas de documentação novas para o SHA final; verificar redaction, build Wails Desktop/HUD, NSIS, rollback de PATH e WebView2.
- Credenciais/assinatura Authenticode, trust root do updater e build de distribuição assinada continuam externos.
- `main` e `release/v1.0.0` apareciam como `protected:false`, sem checks obrigatórios; o plano GitHub restringiu acesso a rulesets.
- Dívida arquitetural residual continua registrada no relatório de engenharia: arquivos grandes, algumas regras de ESLint permissivas, coverage frontend não verificado e revisão modular aprofundada ainda incompleta. NÃO declarar "zero gaps" enquanto essas verificações não forem feitas.

## Gate objetivo antes do merge para release/v1.0.0

1. Congelar SHA da integração e certificar que contém o SHA-base mais atual da release.
2. `node --test scripts/release-version.test.mjs`, `go run ./scripts/verify-version-contract.go .`, `bun install --cwd web --frozen-lockfile`, `bun run --cwd web quality:full`, `go test -race ./...`, `go vet ./...`, `golangci-lint run ./...`, `govulncheck ./...`.
3. Browser E2E, Axe, visual, build embed, `docs:capture` e `docs-verify`.
4. Windows nativo ConPTY/Codex/Cloudflared/WebView2 e controle de erros `terminal`/sessão com testes reais. Linux correspondentes.
5. Revisão de regressões da PR #45 e #46 por arquivos; nenhuma evidência de versão anterior pode substituir exact-SHA.
6. Merge de consolidação para a release quando os gates de código passarem. Publicação/tag somente após certificação nativa/assinaturas.

## Decisão de resolução

Preservar os heads das PRs #45 e #46 como pais de um commit de resolução na branch de integração, documentando intencionalmente que relatórios e capturas antigas foram **omitidos do tree resultante** por obsolescência. A branch canônica não foi alterada automaticamente por esta decisão.
