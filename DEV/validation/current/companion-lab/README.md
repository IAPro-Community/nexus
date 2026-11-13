# NEX Companion 2D Lab — evidência

- Data: 2026-11-13T19:51:13.043Z
- SHA: ab8546dd
- Script: `web/scripts/e2e-companion-lab.mjs` (fixture Core isolado por contexto de browser)
- Comportamentos capturados: ambient, greet, look, thinking, attention, error, resolved, offline-still.
- `rest` (drowse 180s): validado por `nexIdleDirector.test.ts` com clock injetado; sem captura visual em tempo real.
- O mock vive apenas em `context.route`; nenhuma rota de produção é tocada e nenhum estado é inferido.

## Capturas

- `captures/01-ambient.png`
- `captures/02-greet.png`
- `captures/03-look.png`
- `captures/04-thinking.png`
- `captures/05-attention.png`
- `captures/06-error.png`
- `captures/07-resolved.png`
- `captures/08-offline-still.png`

## Re-execução com o atlas raster locked NEX-ID-v1 (2026-11-13T19:54:07Z, branch working tree)

- O `NexPixelSprite` passou a renderizar o atlas `web/public/nex/hud-32bit/nex-idle-atlas-128.png`
  (SHA `8256a302…` verificado contra `identity-lock.json`), substituindo o desenho vetorial rejeitado.
- Capturas regeneradas em 2026-11-13T19:54:07Z: 01-ambient (idle azul), 06-error (véu critical + badge),
  08-offline-still (grayscale imóvel) conferidas visualmente — anatomia do raster aprovado preservada.
- 8/8 comportamentos PASS; web tests 901/901; gates de identidade e release-readiness sem regressão.
