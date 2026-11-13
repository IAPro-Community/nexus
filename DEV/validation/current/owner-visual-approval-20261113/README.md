# Pacote de aceite visual do owner — 2026-11-13

Uma única imagem reúne as três superfícies que dependem do seu aceite:

`owner-visual-approval-package.png`

| # | Superfície | O que aprova | Gate |
|---|---|---|---|
| 1 | `hud_compact` 64px (comparação 64/92/128) | O derivado compacto do atlas locked | `hud_compact` |
| 2 | HUD com o **atlas locked** (`nex-idle-atlas-128`, SHA `8256a302…`) | O render novo do mascote no HUD (12 ações de idle, moods por tratamento) | `owner_visual_approval` (HUD) |
| 3 | Hero contextual no **lab do site** (PT/EN, 8 comportamentos demo) | A direção do hero para o flip público atômico | `owner_visual_approval` (site) |

## Como aprovar

- Responder, por item: **aprovado** / **ajustar X**.
- O aceite vale para o SHA atual: Nexus `2d55a8e1…`, Site `799ef1f…`, atlas `8256a302…`, master `68a9b3b9…`.
- Itens aprovados movem os gates correspondentes para PASSED com este pacote como evidência; ajustes geram nova rodada com diff visual.

## Fontes

- `docs/design/nex/hud-compact-v1/nex-compact-64-92-128-comparison.png`
- `DEV/validation/current/companion-lab/captures/01-ambient.png` (HUD rodando o atlas)
- Nexus-Site `DEV/validation/current/nex-hero-qa/captures/hero-768.png`
