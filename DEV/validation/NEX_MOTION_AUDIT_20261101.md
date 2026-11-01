# NEX motion — auditoria parcial, 2026-11-01

Base: `release/v1.0.0@8123c9ff`; branch `feat/nex-interactive-motion-complete`. A auditoria das issues via API pública encontrou #119, #120, #129 e #141 abertas; #147 fechada. A fonte raster `web/public/nex/hybrid-v1/nex_source_original.png` tem SHA-256 `ef7a4aeb7b24d2ab5f199f913d351414b8fafb0b6208b3ab2b665013adcb9118`; reconstrução do ORA reportou `68a9b3b90722721bacd4067bac3e9d6560acf144898ae60c3d7968692c652f32`. O PNG editorial e o ORA híbrido são candidatos, não o master pixel art aprovado.

A release traz seis estudos de perspectiva opacos, sem yaw calibrado e sem perfis, e 16 quadros de hover frontal; nenhum conjunto satisfaz o giro de 16 vistas. O catálogo e scheduler de 12 IDs já existiam em `nexIdleDirector.ts`, mas IDs não comprovam animação renderizável. O runtime usa SVG frontal; o contrato preserva sua substituição até os gates conjuntos.

Alterações deste lote: impedir que callback síncrono do renderer deixe timer após preempção do IdleDirector; teste de regressão; corrigir `Locator.first()` no QA híbrido. Não há mudança no comportamento de produto visível; catálogo de descoberta permanece inalterado. ADR: `DEV/DECISIONS/NEX_INTERACTIVE_RENDERER_20261101.md`.

Validação local Windows, Node 24.21.0, dependências reaproveitadas de outro checkout (instalação `bun install --frozen-lockfile` não concluiu): teste Vitest focado 8/8; `art:nex:check` passou após restauração dos bytes LF da árvore Git; `art:nex:hybrid:qa` passou 18 checks Chromium após a correção e normalização temporária do HTML. Estas checagens não são certificação nativa ou visual. `quality:full` parou no format:check com avisos preexistentes em `BrowserSurface.module.scss` e `BrowserSurface.tsx`, além de um aviso no teste deste lote posteriormente formatado. Não houve QA de 12 ações, 16 vistas, site, plataformas ou aprovação humana.

`node scripts/verify-nex-release-readiness.mjs` reportou BLOCKED 0/11; `--require-ready` falhou como esperado. Não alterar status de gates sem evidência. Não houve medição de desempenho. Próximo passo: terminar instalação por lockfile, rodar gates em checkout LF e continuar rig/arte e integração em laboratório sem publicar.

Resultado adicional: `test:e2e:hud` falhou aguardando `.nx-os-shell` após 10 s; o servidor registrou também acesso negado ao tentar modificar `C:\Windows\System32\drivers\etc\hosts`. Não foi estabelecido que essa mensagem seja a causa do timeout. Build realizado pelo E2E concluiu, mas não certifica o fluxo.
