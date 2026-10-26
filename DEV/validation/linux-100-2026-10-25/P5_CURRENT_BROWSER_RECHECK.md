# P5 — Reteste atual do preview Composer/Flow

Data: 2026-10-26  
Resultado do caso: `PASS`  
Resultado geral da homologação: `NO_GO_LINUX`

## Identidade do código

- Branch: `release/v1.0.0`
- HEAD local/remoto consultado: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`
- Fingerprint das fontes efetivamente testadas: `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c` (1.430 arquivos)
- Manifesto: `p5-current-recheck-2026-10-26-final/logs/source-provenance-preview-copy-fix-2026-10-26.txt`
- A árvore de trabalho compartilhada está modificada; o fingerprint identifica o overlay, não um SHA Git imutável.

## Defeito e correção

Na prévia de execução, uma indisponibilidade sem contas de provedor aparecia repetida em inglês em cada etapa: o mesmo motivo vinha de `resource.reason` e de `blockers`. A mensagem de ausência de orçamento local também permanecia em inglês no locale `pt-BR`.

A correção concentra a normalização em `web/src/features/work/executionPreviewDetails.ts`: remove diferenças de caixa, pontuação e espaços, elimina mensagens repetidas e localiza esses dois estados estáveis. As traduções foram adicionadas a `en`, `pt-BR` e `es`. O fluxo continua bloqueado quando não há contas elegíveis; a mudança não esconde o bloqueio nem inicia Mission/Agent.

## Verificações

| ID | Procedimento | Resultado |
|---|---|---|
| P5-010-A | `bun run test -- executionPreviewDetails.test.ts flowModel.test.ts` em `web/` | PASS — 20/20 testes focados. |
| P5-010-B | `bun run test:e2e:composer-spine` | PASS — preview de cinco etapas; `ready=false` sem conta de provedor; sem Mission/Agent; mensagens duplicadas e texto inglês antigo ausentes. |
| P5-010-C | `bun run test:e2e:composer-vague-spine` | PASS — duas clarificações, plano de cinco etapas e nenhum run sem provider. |
| P5-010-D | `bun run quality:full` | PASS — 110 arquivos, 681 testes, gates de formatação, lint, Stylelint, typecheck e build de produção. |
| P5-010-E | Chromium E2E com fixture local | PASS visual para o preview no browser. Não substitui UI Wails nativa e não valida provider externo. |
| P5-010-F | Build/instalação CLI, Desktop e HUD Linux AMD64 | PASS — ELF x86-64; `ldd` sem dependências ausentes; Desktop e HUD ativos, Core HTTP 200; hash instalado igual ao build. |

Logs detalhados: `p5-current-recheck-2026-10-26-final/logs/`. O E2E usa fixture HTTP/SQLite e Chromium isolado; não usou credenciais, conta ou serviço de provider externo.

## Capturas

- Preview completo corrigido: `p5-current-recheck-2026-10-26-final/composer-spine/north-star.png`, SHA-256 `bdcf659839a1aed6c82ccade7583ebf8cd372f64d3ea09f0ba72077fd5a30592`.
- Prompt vago corrigido: `p5-current-recheck-2026-10-26-final/composer-vague/north-star-vague.png`, SHA-256 `1e74dae0df31df5943eb1bba09dceb06501d882ac33709110cc3d954ccd3b95e`.
- Capturas são do Chromium automatizado, não do WebView Wails. A validação visual nativa geral segue limitada às evidências próprias do HUD/Desktop.

## Build atualmente instalado

`make build build-desktop-wails build-hud-wails` gerou CLI `76c347d5a12697215c150aa87f6e2e52cf11e199af862c4fa16ca4e3ddc87fbe`, Desktop `4c68324423fce7e591c8e532092964375cccde2b0ccb725ee5066a65f830c6e2` e HUD `9a143bef1d9be86542dfb92da7248cadb9a3db88c18f508f38c30f78ef83e55a`. Os mesmos hashes foram verificados em `~/.local/bin`. A unidade `nexus-homologation-final-restored-20261026.service` está ativa e `http://127.0.0.1:37753/api/v1/health` retornou `{"status":"ok","version":"1.0.0-beta.10"}`. Evidência: `p5-current-recheck-2026-10-26-final/logs/install-latest-overlay-health-2026-10-26.txt`.

## Limites e decisão

P5 permanece parcial: os fluxos de browser e fixtures determinísticas passaram, porém provider/handoff externo autenticado e o mesmo preview dentro da janela Wails ainda não foram homologados. A release continua `1.0.0-beta.10`, sem artefatos estáveis/assinatura de produção e sem SHA Git congelado para este overlay. O veredito Linux permanece `NO_GO_LINUX`.

O produto continua com um único mascote NEX (`NexusHud → GuardianMascot → NexPixelSprite`). O modo/personagem Companion 2D da PR #55 não é requisito e não foi integrado.
