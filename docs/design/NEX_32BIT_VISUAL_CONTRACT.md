# NEX — contrato visual para a linha 32-bit-inspired (candidato)

> Estado: DIREÇÃO PROPOSTA, NÃO HOMOLOGADA. Ilustrações geradas por IA são referências; não são sprite sheets de produção nem assets prontos para recorte.
> Referência: NEX frontal 32-bit-inspired (23/10/2026). O símbolo institucional do Nexus continua separado do mascote.

## Identidade invariável

- Pose neutra frontal, cabeça proporcionalmente grande e visor navy preto arredondado com olhos emissivos ciano/azul.
- Duas aletas laterais simétricas, acoplamentos circulares, carcaça metálica branco-frio e detalhes navy.
- Corpo compacto com **um** núcleo circular luminoso central.
- **Quatro** módulos independentes orbitais como anatomia canônica: dois próximos à lateral do torso e dois inferiores. O anel orbital azul fica atrás/abaixo e não deve cobrir o visor.
- Aletas, olhos, núcleo, módulos, anel e proporções devem ser os mesmos em todos os estados. Não criar dedos, braços humanos, pernas ou capa em variações.
- Vista frontal como padrão. 3/4, lateral e traseira somente como vistas auxiliares coerentes, não poses-padrão do HUD.

## Resolução e consistência

- '32 bits' é uma referência de estética retro/pixel-art rica, não formato nem profundidade obrigatória de cor do arquivo.
- Primeiro produzir e aprovar um master editável com camadas e grade fixa. Não gerar cada estado com texto-para-imagem independente.
- Compact: 64x64 base desenhada manualmente para HUD (32x32 somente símbolo simplificado). Standard: 128x128. Hero: 256x256+; a escala não substitui o redesenho ótico.
- Exibir sem filtragem bilinear quando pixel art for a intenção; testar em DPI e zoom reais. Usar tamanho de viewport coerente com 108x112 DIP do HUD nativo.

## Estados, dados e movimento

- Core canônico na release 1.0: idle, working, warning, offline. Pensando, roteando, verificado etc. são **visuais candidatos**, não afirmar telemetria disponível sem evento canônico.
- Frames em grade fixa, pivôs constantes, transparência real e nenhum salto de centro de massa entre quadros.
- Idle: 6–8 frames, respiração/flutuação quase imperceptível. Blink: 3–4 frames. Working: 6–8 frames, pulso de núcleo. Warning: 4–6 frames, cor laranja reservada à atenção comprovada. Offline: 1–2 frames, sem atividade aparente.
- Atenção deve permanecer legível por texto/ícone sem depender de cor ou movimento. Respeitar prefers-reduced-motion.
- Spritesheets PNG/atlas ou WebP sem perdas + manifesto de pivô/frame/estado; evitar GIF como fonte de produção.

## Critérios de aprovação

1. Reconhecimento do mesmo personagem em frontal, 3/4, lado, costas e quatro estados.
2. Sem diferença de módulo/antena/visor entre frames.
3. Preview HUD real a 1x/1.25x/1.5x e múltiplos temas; legibilidade sem pixel shimmer/clipping.
4. Teste de animation loops, pause/reduce-motion e quiescência do estado offline.
5. Build/testes frontend; X11/Wayland/Windows/macOS e hit regions certificadas por SHA antes de merge/release.

## PR #66

O atual `NexPixelSprite.tsx` é apenas um **placeholder de integração**, não a arte oficial 32-bit. Substituí-lo depois de criar o master e o atlas aprovados, sem modificar o contrato de `GuardianMascot`, os moods existentes ou a geometria nativa.
