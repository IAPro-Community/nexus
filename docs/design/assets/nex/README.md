# Referências visuais do NEX

Estas ilustrações estão **versionadas no repositório** para facilitar o uso por outra IA, revisão de paridade e documentação. A aparência principal foi escolhida pelo usuário: **NEX frontal, 32-bit-inspired**.

![Desenho de anatomia e referência frontal](./nex-character-reference.svg)

![Estados e expressões, incluindo candidatas não operacionais](./nex-expression-states.svg)

![Escalas e aplicações](./nex-scales-layout.svg)

## Arquivos

- `nex-character-reference.svg`: desenho vetorial frontal e anatomia invariável, referência visual reutilizável.
- `nex-expression-states.svg`: exemplos de expressões derivadas da **mesma geometria**, incluindo `critical` e indicação explícita dos estados não operacionais.
- `nex-scales-layout.svg`: exemplos de escala e segurança; **não** significa que existam masters Hero/Standard aprovados.
- `../../NEX_VISUAL_MANIFEST.json`: manifesto para automações.
- `../../NEX_VISUAL_REVIEW_CHECKLIST.md`: critérios de homologação da identidade.
- `../../NEX_AI_GENERATION_GUIDE.md`: instruções reprodutíveis para IA gerar novas poses sem perder a identidade.

## Autoridade e procedência

- A decisão autoritativa é `docs/design/NEX_32BIT_VISUAL_CONTRACT.md`.
- O **source executável do Compact** é `web/src/features/hud/NexPixelSprite.tsx`.
- Os três SVGs desta pasta são **desenhos vetoriais auxiliares derivados da especificação**, não copias pixel a pixel das pranchas conceituais geradas anteriormente no chat. Não tratá-los como prova de fidelidade absoluta, e não recortar os SVGs como sprite sheets animados.
- A **prancha original aprovada** foi identificada: `guia_visual_oficial_do_mascote_nex.png`, 1448 × 1086, SHA-256 `4dff7d9521687f77f666a56dcf04bbe3d04dd4cd895139b1fa98df5f90f35ed5`. Ela ainda precisa ser importada no repositório. Use `node scripts/import-nex-concept.mjs --source <arquivo-original.png>` e inclua o PNG e o manifesto modificado em um commit. Não substitua por imagens comprimidas, miniaturas ou desenhos aproximados. Não fabricar um PNG com nome de arte oficial a partir de uma miniatura.
- O logo institucional Nexus não deve ser substituído pelo mascote.

## Como revisar

Verifique a mesma anatomia em todos os estados, o visor e os quatro módulos destacados. Escalas Compact, Standard e Hero podem ter diferentes níveis de detalhe, mas não podem alterar o personagem. Validação funcional e lançamento binário são gates separados.
