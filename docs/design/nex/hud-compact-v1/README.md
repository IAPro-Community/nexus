# NEX hud-compact-v1 — Compact 64px (derivado aprovado, aceite pendente)

**identity_lock_version:** NEX-ID-v1 (FROZEN)
**Origem:** downscale nearest-neighbor do atlas 128px aprovado
(`docs/design/nex/experimental-atlas/nex-idle-atlas-128.png`,
SHA-256 `8256a302fff77996e3563ef42d1e62685b59084c8a17446bb2acb2219e85dfda`)
**Data:** 2026-11-13 · **Output:** este diretório · **Escala de uso:** HUD Compact 64px

## Tratamento

Filtro de pixelização (nearest-neighbor), **mesmo tratamento aprovado
explicitamente pelo owner em 2026-11-12** para o atlas hud-32bit (128px), com a
divergência registrada: não é pixel art desenhada em clusters
(`RETRO_GAME_IDLE_DIRECTION.md`), e o refino true-pixel segue como follow-up
opcional.

⚠️ **A variante 64px NÃO estava coberta pela aprovação original** ("aprovo os
dois" cobriu hero + atlas 128px). Este derivado existe como candidato e aguarda
aceite visual do owner antes de o gate `hud_compact` mover para PASSED.

## Conteúdo

| Arquivo | Papel | SHA-256 |
|---|---|---|
| `nex-idle-atlas-64.png` (512×768) | Atlas idle Compact: 12 ações × 8 quadros × 64px | ver `manifest.json` |
| `nex-compact-64-92-128-comparison.png` | Comparação lado a lado do quadro 0 nas escalas 64/92/128 (critério do COMPANION_2D_CLIPPY_LITE) | ver `manifest.json` |
| `manifest.json` | Proveniência + hashes | — |

Preserva topologia/proporções do master congelado (sem mudança de silhueta,
anatomia ou contagens). Nenhum outro mascote ou identidade alternativa.
