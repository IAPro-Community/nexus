# ADR — renderização do NEX interativo

Data: 2026-11-01. Estado: decisão técnica preliminar; publicação bloqueada.
Base auditada: `8123c9ff` (`release/v1.0.0`). Relaciona #119, #120, #129 e #141.

## Contexto e decisão

O HUD deve continuar usando o SVG existente até que o master retro, as derivações e a integração recebam aprovação. Para um futuro renderer do HUD, preferir atlas 2D em Canvas ou camadas DOM/SVG, com estado e agendamento independentes da apresentação. Para Hero, preferir camadas raster/DOM e carregamento sob demanda; o giro depende de 16 vistas reais. A escolha entre Canvas e DOM para cada superfície depende de protótipo comparativo medido no WebView. Esta decisão não autoriza substituir arte nem desbloqueia a publicação.

| Alternativa | Avaliação e condições |
| --- | --- |
| Canvas 2D e atlas | Sem WebGL obrigatório; precisão de pixels e alpha controláveis. Exige desenho somente quando necessário, fallback e botão/texto acessíveis externos. Custo de cópia, memória e latência precisam de medição no Wails. |
| DOM/SVG e Web Animations | Camadas existentes, controle por compositor e sem dependência adicional. Pode preservar interação nativa; muitos elementos, sombras e raster grande precisam de medição. Escala do raster editorial não equivale a pixel art desenhada. |
| PixiJS v8 | Pode ajudar com muitos sprites e batching, mas adiciona dependência, gerenciamento de textura e compatibilidade GPU. Só adotar após benchmark demonstrar benefício; descarregar recursos e evitar dois loops. |
| Three.js | Custo de bundle/GPU e manutenção injustificado para imagens 2D. Somente reavaliar com modelo editável fiel, materiais, pivôs e vistas validadas; girar um plano frontal não produz costas. |

Em todos os casos, usar controle acessível nativo, respeitar movimento reduzido, visibilidade e estados existentes, sem polling de providers ou inferência de conclusão. O renderer não deve possuir estado de negócio nem duplicar scheduler. Nenhum benchmark de latência, CPU, GPU, bundle ou LCP foi executado nesta decisão.

## Evidências e consequência

`identity-lock.json` permanece `CANDIDATE_PENDING_APPROVAL`; `RELEASE_READINESS.json` permanece BLOCKED. Os seis estudos 256 possuem fundo opaco e ângulos não calibrados; os 16 quadros `hover256-v1` são frontais e não são 16 vistas. É necessário master editável, desenho ótico 64/128, rig e underpainting, testes de performance, revisão humana e QA nativa antes de qualquer promoção. Reavaliar a escolha no mesmo SHA dos assets candidatos e anexar benchmarks reproduzíveis.
