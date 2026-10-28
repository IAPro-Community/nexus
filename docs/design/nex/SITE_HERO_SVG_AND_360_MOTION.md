# NEX — SVG Hero animável, 2.5D e giro de 360° sem motor 3D

**Direção:** personagem original NEX selecionado pelo usuário, branco/azul, pose frontal, 2 aletas, 2 acoplamentos laterais, núcleo central e 4 módulos magnéticos. Interpretação **vetorial detalhada para uso de site**, complementar à versão de **pixel art retrô HD** requerida para o HUD.

**Asset:** [nex-site-hero.svg](../../../web/public/nex/nex-site-hero.svg). **Preview:** `cd web && npm run art:nex:hero:preview` produz `web/dist/nex-site-hero-preview.html`, arquivo HTML autossuficiente que abre localmente.

> Não confundir: o SVG desta entrega é uma **reconstrução vetorial animável** baseada na anatomia escolhida e exige revisão de fidelidade. Ele **não é** uma vetorização geometricamente idêntica do PNG selecionado, não é um personagem 3D e não entrega giro real de 360°. O código atual do HUD permanece inalterado.

## Revisão de fidelidade — imagens geradas em 28/10/2026

A diferença percebida pelo usuário entre a referência raster e o SVG anterior foi documentada em [referências comparativas do NEX](reference-pack/2026-10-28/README.md) e no respectivo [manifesto de hashes](reference-pack/2026-10-28/manifest.json). Há uma prancha de auditoria visual e uma versão Hero PNG transparente, porém seus binários permanecem **PENDING_BINARY_UPLOAD** até serem importados por um commit posterior. **Não** alegar que o SVG atual corresponde pixel a pixel ao PNG; a correção de silhueta e riqueza visual requer revisão comparativa, não só keyframes CSS.

## Atualização: Hero híbrido fiel ao PNG (28/10/2026)

Foi desenvolvido um **candidato raster em camadas** que conserva os próprios pixels do PNG selecionado: [manifesto dos assets](hybrid-v1/asset-manifest.json), [guia de revisão](hybrid-v1/README.md). A fonte editável é um projeto OpenRaster (`.ora`) com oito camadas recortadas além da base, e um protótipo HTML oferece seis idles. A recomposição estática mantém os canais RGB originais; o alpha tem diferenças pequenas nas bordas das máscaras. A prévia raster **não foi ainda publicada como arte do site** nem homologada. Enquanto os PNGs não estiverem commitados, o SVG antigo continua apenas como alternativa vetorial candidata, não como substituto fiel.

## O que está realmente disponível agora## O que está realmente disponível agora

O SVG em `viewBox=1024×1024` tem fundo transparente e grupos identificáveis, mantendo a geometria base e permitindo movimentos separados. As peças possuem **âncoras fixas**; CSS anima somente elementos internos onde necessário:

| Parte | ID | Movimento leve viável sem 3D |
| --- | --- | --- |
| Personagem completo | `nex-figure` | flutuação de 4px, preemptada pelo estado |
| Pivô do movimento simulado | `nex-yaw-stage` | `skewY`/compressão horizontal discreta ±12°, **não giro de 360°** |
| Cabeça e olhos | `nex-head`, `nex-face` | piscar, olhar, microinclinação, brilho de expressão |
| Aletas | `nex-fin-left/right` + `nex-fin-motion` | balanços espelhados até ~1,5° |
| Reator | `nex-reactor` | luminosidade/pulsação, nunca “progresso %” |
| 4 módulos | `nex-module-1..4` + `nex-module-motion` | deriva vertical de até 4px em fases distintas; âncoras externas imóveis |
| Anel e sombra | `nex-orbit-front/back`, `nex-shadow` | opacidade/respiração visual |

O SVG pode ser exibido diretamente com `<img src="/nex/nex-site-hero.svg" alt="NEX" />`; assim ele executa apenas o *idle* definido internamente, **sem controle externo de grupos**. Para alterar `data-state`, expressão ou pivôs através do site, **inserir o SVG inline no DOM** a partir desse arquivo confiável (por pipeline do bundler/SVGR ou carregamento controlado), não tentar selecionar elementos internos de um `<img>`.

Estados de apresentação visuais suportados via atributo na raiz `data-state`: `idle`, `working`, `warning`, `critical`, `offline`. `data-motion="off"` e `prefers-reduced-motion: reduce` congelam os loops; isso não adiciona eventos, timers, estados novos ou semânticas ao Core.

## Como reproduzir a volta completa do personagem

**Não dá para fazer uma volta verdadeira a partir de um SVG desenhado apenas de frente.** Aplicar `rotateY(360deg)` ou virar a imagem como um cartão inverte a frente e **não cria** laterais/costas, orelhas ocultas, encaixes, oclusão de módulos ou mudança de volume. As cinco vistas da prancha de referência são **desenhos conceituais pequenos**, não frames automaticamente coerentes.

| Estratégia | Fidelidade | Custo de execução | Limitações | Papel indicado |
| --- | --- | --- | --- | --- |
| **A. SVG 2D por camadas** (esta entrega) | Alta na frente e nos idles; nenhuma vista traseira | Baixo, CSS+SVG com poucos keyframes | cabeça e torso permanecem frontais; sem parallax de verdade | **HUD** e site em idle |
| **B. 2.5D de pequena amplitude** (esta prévia, ±12°) | Sugere profundidade, mantém personagem legível | Baixo; `transform`/composição GPU | oclusão e perspectiva apenas simuladas; deformações >15° | Site hero leve, curiosidade, hover |
| **C. Sprite atlas 8 vistas** | Frente/3/4/perfil/costas representados | Baixo em runtime; um atlas + steps | salto visual a cada 45°; qualidade depende de desenho consistente | Preview/turnaround rápido |
| **D. Sprite atlas 16–24 vistas** | Pode parecer um giro quase contínuo em 360° | Baixo–médio no navegador, maior em arquivos | requer **16–24 vistas autorais alinhadas**, silhueta/iluminação/oclusão; custo alto de produção de arte | **Melhor opção para site** sem modelo 3D |
| **E. Imagens separadas interpoladas por morph 2D** | Intermediário entre vistas vizinhas, se bem planejado | Médio, arte + interpolação | distorções em visor/rosto e módulos; não substitui vistas reais | Transições específicas, não o giro principal |
| **F. Modelo 3D glTF otimizado** | Giro real livre | Maior integração, GPU e manutenção | contraria a preferência por produção simples 2D; não é obrigatório | Futuro, somente se câmera livre for requisito |

**Recomendação técnica:** preservar **A** para o HUD; usar **A+B** no Hero do site agora; produzir **D (16 vistas)** para uma volta completa futura, com keyframes em ritmo moderado quando o visitante clicar, passar o mouse ou rolar a página. Para “giro suave” aumentar para 24 vistas se a revisão visual mostrar saltos. Guardar 8 vistas inicialmente apenas como teste de anatomia.

### Produção do atlas 360°

1. Definir um **único master de personagem**, incluindo o original aprovado, volumes, sombras, posição de torso/núcleo, orelhas, aletas e os quatro módulos.
2. Desenhar 16 keyposes de yaw em passos de **22,5°**: 0°, 22,5°, 45°, ... 337,5°. Para 24 vistas, passos de 15°. **Nenhuma vista pode ser obtida espelhando a frente**.
3. Fazer os módulos orbitarem coerentemente com o ponto de vista, respeitando perspectiva, oclusões frente/costas e identificadores.
4. Exportar PNGs com alpha, mesma grade 256/320/512, pivô do personagem estável e bounding boxes consistentes; compactar com spritesheet WebP lossless ou PNG atlas conforme suporte/qualidade.
5. Criar manifesto `yaw-turn.json` com IDs/ângulos, duração, fps, frames, pivô, ordem Z e critério `return_to_front`. Uma volta acionada deve voltar para a pose frontal quando a interação terminar.
6. Pré-carregar frames próximos ao atual e carregar a sequência completa **sob demanda**. Para o HUD, não baixar a coleção Hero por padrão.
7. Revalidar a **mesma anatomia** em 0°, 90°, 180°, 270° e 360° e pausas, em fundo claro/escuro e `prefers-reduced-motion`.

Não extrair frames de uma única prancha de marketing cheia de textos e efeitos. As imagens existentes servem como guia de vistas, não um atlas pronto.

## HUD vs site: limite de recursos e distração

**No Nexus HUD:** 64–128px efetivos, interações frequentes, maior sensibilidade a CPU/temperatura e pouca área para curvas detalhadas. Recomendado: desenho Compact **pixel-perfect** em atlas pequeno, idles 8–12fps opcionais, piscar/olhar/módulos, pausas em inatividade, `offline` imóvel, e eventualmente um gesto 3/4 curto. **Não** manter 360° em loop; somente se houver gesto intencional e espaço.

**No site:** pode usar Hero SVG com camadas 1024 viewBox e CSS; variações de sorriso, curiosidade, reação e hover; 2.5D discreto no scroll; sequência de 360° sob demanda com 16–24 vistas reais. Quando fora do viewport, pausar/ocultar animações, evitar filters `blur` grandes e evitar multiplicar vários SVGs Hero ativos. `<img>` oferece excelente isolamento, mas não permite interatividade por parte interna.

**No mobile:** preferir um único Hero animado, tempo curto de reação, versão estática sob `prefers-reduced-motion`, e cuidado com bateria; reduzir partículas/glow em telas pequenas se o perfil mostrar alto gasto.

## Checklist para aceitar

- [ ] SVG abre como imagem real em Chrome/Firefox/Safari, sem erro XML, sem fundo opaco, e mantém os grupos esperados.
- [ ] Preview no navegador reproduz `idle`, `working`, `warning`, `critical`, `offline`.
- [ ] Modo de movimento desligado congela a animação; `prefers-reduced-motion` funciona.
- [ ] Em pequenas inclinações a frente permanece legível; nenhuma distorção é vendida como rotação 360°.
- [ ] Módulos preservam as quatro âncoras e a cabeça não “desgruda” das aletas.
- [ ] Revisão humana compara o SVG com a referência original selecionada.
- [ ] O futuro atlas tem faces, costas, módulos e anel coerentes em todos os ângulos, e pode voltar ao frontal.
- [ ] Antes de integrar no site externo, verificar permissões de reutilização de asset, tamanho transferido, LCP, acessibilidade, FPS em dispositivo médio e animações fora do viewport.

**Status desta entrega:** Hero **SVG vetorial candidato em source**, efeitos 2D e 2.5D de inclinação demonstráveis, integração documental. **Não entregue:** 16–24 vistas para 360°, spritesheet da pose inteira, master pixel-art editável, testes nativos da release ou homologação da fidelidade visual.
