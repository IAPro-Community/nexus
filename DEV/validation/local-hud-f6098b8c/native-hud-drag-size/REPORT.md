# HUD nativo — arraste e presets de tamanho — 2026-10-21

## Escopo

- Worktree: `C:\Dev\nexus-hud-validation`
- Branch: `release/v1.0.0`
- Base: `f6098b8c684a97bf7e8e56804109ca491c78cfa2`
- `VERSION`: `1.0.0-beta.10`
- Estado: worktree sujo, **sem commit, push, merge, tag ou release**
- Binário de evidência: `.tmp/homolog/bin/nexus-hud-diag-v3.exe`
- Single-instance ID isolado: `com.iapro.nexus.hud.homolog.v3`
- Dados: diretório sintético `.tmp/homolog/data`; nenhuma conta real foi usada

## Implementação verificada

1. A cápsula nativa mantém o clique para abrir/fechar o resumo.
2. Uma alça de arraste dedicada de 24 px, à esquerda da cápsula, move a janela.
3. O cabeçalho do popover também é uma região arrastável.
4. A alça aceita teclado: setas movem 16 px; `Shift` + seta move 64 px.
5. O popover nativo oferece três presets persistentes:

| Preset | Compacto | Expandido |
| --- | --- | --- |
| Compacto | 220 × 60 | 334 × 496 |
| Confortável | 260 × 70 | 380 × 560 |
| Grande | 320 × 84 | 440 × 640 |

6. A troca de preset preserva a âncora inferior direita.
7. O primeiro show reaplica a geometria persistida; o antigo primeiro paint em 190 × 48 foi corrigido.

## Resultados

| Verificação | Resultado | Evidência |
| --- | --- | --- |
| `go test -count=1 ./internal/desktop/...` | **PASS** | saída local, 2,1 s |
| `go vet ./internal/desktop ./cmd/nexus-hud` | **PASS** | saída local |
| `npm run quality` no `web/` (check:styles, format, ESLint, Stylelint, `tsc`, Vitest) | **PASS** | 93 arquivos / 578 testes |
| Build Web (`node scripts/build.mjs`) | **PASS** | `internal/control/web/embedded/` |
| Build Go do HUD | **PASS** | `nexus-hud-diag-v3.exe` |
| Primeiro paint com estado limpo | **PASS** | 220 × 60 no canto inferior direito |
| Presets via UI Automation | **PASS** | 220×60 → 334×496 → 380×560 → 440×640 → 334×496 → 220×60 |
| Arraste real pela alça | **PASS** | `-144,-108` px (pedido `-160,-120`; início após limiar) |
| Seta direita | **PASS** | `+16` px |
| `Shift` + seta direita | **PASS** | `+64` px |
| Reinício com preset `comfortable` | **PASS** | abriu em 260 × 70 na posição persistida `1516,866` |
| Linux/macOS deste slice | **NOT_TESTED** | sem execução nativa |
| `node scripts/docs-verify.mjs` | **BLOCKED** | exige alterações visuais commitadas; o worktree está sujo |

## Capturas

- `01-expanded-334x496.png` — popover aberto com os controles de tamanho
- `02-preset-compact-334x496.png` — preset Compacto aplicado
- `03-preset-comfortable-380x560.png` — preset Confortável aplicado
- `04-preset-large-440x640.png` — preset Grande aplicado
- `05-back-to-compact-334x496.png` — retorno ao Compacto
- `06-collapsed-220x60.png` — cápsula compacta com a alça de arraste
- `07-after-drag.png` — cápsula após arraste (tooltip da alça visível)
- `08-restart-comfortable-260x70.png` — primeira pintura após reinício
- `state-after-restart.json` — estado persistido

## Limites

- O slice foi exercitado apenas em Windows 11 `10.0.26200`, scale 100%.
- O arraste é intencionalmente separado do clique; o corpo da cápsula não é uma região de arraste.
- Os três presets são a geometria suportada; redimensionamento livre pelas bordas não é certificado.
- `docs-verify` só pode passar quando as alterações visuais estiverem em um SHA imutável.
- Nenhum resultado acima autoriza tag, release, assinatura ou alegação de GO.
