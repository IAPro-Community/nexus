# NEX Hero híbrido v1.1 — relatório de QA no Chromium

**Data:** 2026-10-28  
**Alvo:** protótipo HTML com camadas extraídas do PNG original, NÃO o HUD nativo  
**Base:** `docs/design/nex/hybrid-v1/asset-manifest.json`, pacote `nex_hybrid_github_assets_v1_1.zip` fornecido nesta conversa

## Falha reproduzida

A arte reconstruída parecia correta em repouso. Em testes reais com a camada de olhos comprimida/oculta, porém, **apareciam grandes retângulos azulados no visor**. A diferença de opacidade da base original deixava o fundo editorial vazar por trás dos olhos.

## Correção na arte

- Reconstrução de fundo navy suave, usando máscaras elípticas com bordas esfumaçadas e retomada de alpha original do visor no centro dos olhos.
- **PNG original intocado.** Modificadas somente `body_base.png`, `reconstructed_idle.png`, `nex_hybrid_master.ora` e o manifesto derivado.
- Exportador `build_layers.py` agora é portátil dentro do checkout, com SHA no manifesto; não utiliza caminhos do container.
- Comparação estática v1.1: RGB MAE **0,78704 / 255**; alpha MAE **0,16822 / 255**; 12,3111% de pixels com alguma diferença. A variação maior de RGB em relação à v1.0 é intencional para que **a piscada** não revele artefatos retangulares.
- A imagem de frente do PNG original continua no pacote como prova de identidade; o master de arte retrô final **não** foi homologado.

## Execução do Chromium (testes observados)

Utilizada a implementação do protótipo local e os PNGs gerados nesta sessão, disponibilizados à página por data URLs **sem alterar os bytes**. Isso contorna a restrição deste executor sobre `file://`/localhost, mas usa o motor Chromium real.

| Teste | Resultado | Observação |
| --- | --- | --- |
| Renderização inicial, 4 módulos + 2 olhos | PASS | Sem erros JavaScript |
| Piscar: visor com olhos fechados | PASS visual observado após correção | Sem retângulos vazados |
| Olhar (`idle_glance`) | PASS funcional | Ação disparada no DOM |
| Módulos (`idle_orbit_breathe`) | PASS funcional | Ação disparada no DOM |
| `critical` | PASS funcional | `data-state=critical`, idle interrompido |
| `offline` | PASS | Módulos: `animation-name=none` |
| Preferência explícita de movimento desligado | PASS | `animation-name=none` |
| Mobile 390×844, DPR 2 | PASS | Sem overflow horizontal, 4 módulos |
| `prefers-reduced-motion: reduce` no mobile | PASS | `animation-name=none` |
| Chrome + Safari + Firefox reais no site publicado | NOT_RUN | Exige build implantado e binários versionados |
| LCP, energia GPU, perfis móveis físicos | NOT_RUN | Exige laboratório em dispositivos |
| HUD nativo Windows/macOS/Linux | NOT_RUN | Não foi alterado |

**Teste reproduzível no repositório:** `cd web && npm run art:nex:hybrid:qa`. Exige binários previamente importados; o teste estrito falha se as imagens não estiverem presentes. Gera capturas e JSON em `web/dist/nex-hybrid-qa/`.

## Situação de integração

O pacote v1.1 real está disponível como **arquivo ZIP nesta conversa**, e os novos hashes estão em GitHub. **Ainda não foi incluído como blobs binários no GitHub** nesta etapa. Não marcar a revisão 1.1 como publicada, não substituir automaticamente o NEX atual do site ou do HUD e não declarar o Core homologado.

**Próximos gates:** ingestão com `node scripts/verify-nex-hybrid-assets.mjs --require-binaries`, execução da suíte Node do repositório, build do site com a arte, revisão humana da piscada e, apenas então, adoção na landing.
