# NEX — handoff visual para IA, designers e agentes de codificação

## Leia nesta ordem

1. [Contrato normativo](NEX_32BIT_VISUAL_CONTRACT.md)
2. [Manifesto de máquina](NEX_VISUAL_MANIFEST.json)
3. [Desenho principal frontal](assets/nex/nex-character-reference.svg)
4. [Expressões](assets/nex/nex-expression-states.svg)
5. [Escalas](assets/nex/nex-scales-layout.svg)
6. [Fonte Compact usada no HUD](../../web/src/features/hud/NexPixelSprite.tsx)
7. [Checklist obrigatório](NEX_VISUAL_REVIEW_CHECKLIST.md)

## Briefing reutilizável

> Desenvolva **o mesmo personagem NEX** da identidade oficial do IAPro Nexus. Use a **pose frontal** como default e pixel art retrofuturista **32-bit-inspired**. Conserve a cabeça grande arredondada, visor navy com olhos ciano, duas aletas verticais, dois acoplamentos circulares, corpo pequeno flutuante, núcleo central circular, quatro módulos orbitais separados (dois laterais e dois inferiores) e anel energético azul inferior. A cor laranja representa avisos reais e o vermelho-rosa (`#FF496C`) representa exclusivamente situações críticas comprovadas. Não adicione braços humanos, pernas, boca, capacetes diferentes nem logotipos de provedores. Mantenha volume, proporções e iluminação consistentes com os desenhos versionados. Antes de entregar, inclua uma comparação frontal lado a lado e explicite divergências. Se não puder inspecionar os arquivos, **não alegue paridade**; solicite os assets em vez de recriar de memória.

## Processo para novos assets

- Inspecionar os desenhos e o componente antes de qualquer geração.
- Produzir uma primeira pose frontal com a mesma geometria, sem inventar nova personagem.
- Usar o mesmo master em camadas para as expressões e animações; evitar geração independente por frame.
- Exportar master editável + PNG sem perdas/sprite atlas com fundo transparente e pivôs estáveis.
- Fazer avaliação humana da silhueta no tamanho de uso real. Para Hero, criar arte de alta resolução mantendo a identidade, **não** apenas ampliar pixels.
- Registrar comparação, limitações, arquivo-fonte, hash, licença/proveniência, data e resultado da checklist.
- Exigir evidência funcional/nativa antes de distribuir o personagem em binário.

## Estados: não confundir design com runtime

A v1 observa: `idle`, `working`, `warning`, `critical`, `offline`. O mood de apresentação `attention` é convertido em `warning` na arte; severidade crítica comprovada produz `critical`. As expressões de curiosidade, sucesso e roteamento podem constar em arte conceitual, porém **não representam transições disponíveis** até a aplicação emitir eventos canônicos correspondentes.

## Regra de conflito

O contrato decide **quem NEX é**; o fonte Compact decide **como o HUD o renderiza**; as pranchas ilustram **sua anatomia e aplicações**. Quando duas representações divergirem, não “corrija” a direção oficial ao gosto da IA: documente a diferença, preserve o comportamento em runtime e proponha uma mudança explícita para revisão. O logotipo institucional permanece separado.
