# NEX — referências visuais geradas em 28/10/2026

Este registro corresponde às **duas imagens PNG efetivamente geradas** durante a avaliação da diferença entre o NEX do PNG e o SVG usado pelo site.

| Arquivo | Resolução | Propósito |
| --- | --- | --- |
| `nex-visual-audit-2026-10-28.png` | 1672×941 | Auditoria de proporções, visor, aletas, acoplamentos, núcleo e módulos |
| `nex-hero-raster-2026-10-28.png` | 1254×1254, alpha | NEX isolado e detalhado como referência raster para Hero |

**Status binário:** `PENDING_BINARY_UPLOAD`. Os PNGs foram gerados e os hashes registrados, mas **ainda não estão nos objetos Git**. A API conectada nesta sessão permite gravar texto e SVG, porém não transportar diretamente os arquivos binários locais. Não marcar essas imagens como publicadas no GitHub, nem criar links GitHub para PNGs inexistentes.

Os dois arquivos originais estão disponíveis no chat em um ZIP chamado `nex_visual_github_upload_2026-10-28.zip`, com estrutura de diretórios pronta para extração na raiz do clone. Não usar miniaturas ou versões regeneradas no lugar deles.

## Incorporar binários sem alterar a identidade

No clone do repositório, estando na `release/v1.0.0` ou em uma branch de trabalho:

```bash
unzip -o /caminho/nex_visual_github_upload_2026-10-28.zip -d .
node scripts/verify-nex-image-references.mjs --require-binaries
node scripts/verify-nex-image-references.mjs --promote
git add docs/design/nex/reference-pack/2026-10-28/
git commit -m "assets(nex): include visual audit and transparent Hero PNG references"
git push
```

O `--promote` valida SHA-256, bytes e dimensão PNG de ambos e, **apenas se todos passarem**, marca a publicação como `VERSIONED` no manifesto para o commit local.

## Precedência e uso correto

1. Personagem originalmente selecionado pelo usuário, definido em `docs/design/nex/identity-lock.json`.
2. `docs/design/nex/RETRO_GAME_IDLE_DIRECTION.md` e `docs/design/NEX_32BIT_VISUAL_CONTRACT.md`.
3. Estes dois PNGs como **materiais auxiliares**, para avaliar a fidelidade da nova arte e orientar um master visual.
4. O `web/public/nex/nex-site-hero.svg` atual é um **vetor editável em camadas**, mas ainda não é uma cópia fiel do PNG; nunca alegar que a simples presença do SVG resolve a divergência.

Os PNGs não viram spritesheets nem SVGs automaticamente. A aprovação visual para site e HUD depende de comparação renderizada em ângulos e escalas reais.
