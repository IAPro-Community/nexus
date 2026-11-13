# NEX Companion 2D Lab — evidência

- Data: 2026-11-13T17:58:32.633Z
- SHA: e1e2b750
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
