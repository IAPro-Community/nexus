# NEX — checklist de paridade visual e homologação

> **Status da identidade:** aprovada. **Status de binários:** exige evidência nativa separada. Marcar cada item como PASS / FAIL / NOT_RUN, jamais usar checklist como prova automática de qualidade visual.

## A. Fidelidade da identidade (revisão humana obrigatória)

- [ ] A pose padrão continua **frontal**, simétrica, com cabeça grande.
- [ ] O visor é arredondado e navy/preto; os olhos têm a mesma linguagem expressiva.
- [ ] São exatamente **duas aletas** e **dois acoplamentos circulares laterais**.
- [ ] Há **um** núcleo circular luminoso central.
- [ ] Há exatamente **quatro módulos flutuantes**: dois laterais, dois inferiores.
- [ ] A órbita fica na região inferior e não cobre o rosto.
- [ ] Não foram adicionados dedos, pernas humanas, boca, nariz, capa ou marcas de provedores.
- [ ] Paleta e proporções permanecem equivalentes entre idle, working, warning, offline.
- [ ] Ilustrações novas foram comparadas lado a lado com [desenho frontal](assets/nex/nex-character-reference.svg) e [expressões](assets/nex/nex-expression-states.svg).
- [ ] Diferenças entre o conceito raster aprovado e a arte operacional são reconhecidas/documentadas; não declarar equivalência pixel a pixel sem revisão humana.

## B. Assets, proveniência e produção

- [ ] `NEX_VISUAL_MANIFEST.json` aponta para arquivos existentes e possui estados corretos.
- [ ] SVGs podem ser abertos e têm fundo transparente na camada do personagem (a prancha pode ter fundo editorial).
- [ ] A arte oficial Compact é o componente `NexPixelSprite.tsx`; exports derivam dele.
- [ ] Standard/Hero, caso publicados, têm masters separados e preservam a mesma anatomia.
- [ ] Assets originais aprovados, quando incorporados, têm arquivo, data, origem, responsável e hash.
- [ ] Nenhum link usa `sandbox:` ou imagem temporária/externa como fonte canônica.
- [ ] `node scripts/verify-nex-visual.mjs` passa no checkout da branch (validação estrutural; não substitui revisão visual).

## C. Animação/uso real

- [ ] Personagem legível a 1x, 1.25x e 1.5x no tamanho real do HUD.
- [ ] Sem clipping, tremor de pixel, mudanças inesperadas de anatomia entre quadros.
- [ ] `idle`: flutuação/piscada discretas; `working`: atividade visual; `warning`: apenas mediante atenção real; `offline`: não simula atividade.
- [ ] `prefers-reduced-motion`: animações reduzidas/desativadas, estado ainda compreensível.
- [ ] Contraste em tema claro/escuro, leitor de tela e acessibilidade preservados.
- [ ] CPU/GPU estáveis e sem tarefas/LLMs independentes para movimentar o mascote.
- [ ] Alternar BAR ⇄ MASCOT ⇄ TRAY e reiniciar sem perder preferências.
- [ ] Transparência, drag/click-through, popup e alça preservados em Windows/macOS/Linux X11/Wayland.

## D. Gates da release (não executáveis por inspeção documental)

- [ ] `bun run --cwd web quality:full`.
- [ ] `node scripts/docs-verify.mjs`.
- [ ] Smoke gráfico do HUD para o exato SHA, DPI e SO.
- [ ] Screenshots/vídeo de evidência com checksum, SHA e classificação apropriada.
- [ ] Testes de regressão nativa e rollback/reinstalação.
- [ ] Critérios de aprovação humana da arte registrados antes de alterar a referência.

**Veredito:** apenas uma aprovação formal baseada em evidências fecha os itens. NÃO assumir GO em caso de `NOT_RUN`.
