# HUD nativo — arraste e presets de tamanho — 2026-10-21

## Escopo

- Worktree: `C:\Dev\nexus-hud-validation`
- Branch: `release/v1.0.0`
- Base: `f6098b8c684a97bf7e8e56804109ca491c78cfa2`
- `VERSION`: `1.0.0-beta.10`
- Estado: worktree sujo, **sem commit, push, merge, tag ou release**
- Binário de evidência final: `.tmp/homolog/bin/nexus-hud-diag-v4.exe`
- SHA-256 do binário: `ae64668c16eedbacabc9f91eb0b05a8a3cb6f28f37800ed59a961e4aa5d49674`
- Single-instance ID isolado: `com.iapro.nexus.hud.homolog.v4`
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
| `go vet ./internal/desktop ./cmd/nexus-hud` | **PASS** | saída local |
| `go test -count=1 ./internal/desktop/...` | **PASS** | `ok ... 1.425s` |
| `npm run quality` no `web/` (check:styles, format, ESLint, Stylelint, `tsc`, Vitest) | **PASS** | 93 arquivos / 578 testes |
| Build Web (`node scripts/build.mjs`) | **PASS** | `web/dist` |
| Build Go do HUD | **PASS** | `nexus-hud-diag-v4.exe` |
| Primeiro paint com estado limpo (execução anterior) | **PASS** | 220 × 60 no canto inferior direito |
| Presets via UI Automation (build final) | **PASS** | 260×70 → 380×560 → 334×496 → 380×560 → 440×640 → 334×496 → 220×60 |
| Arraste real pela alça (build final) | **PASS** | `-132,-88` px (pedido `-144,-96`; início após limiar) |
| Alça acessível com 24 px | **PASS** | retângulo `24 × 42`; nome `Arrastar HUD. Use as setas para mover.` |
| Setas / `Shift` + seta | **PASS** (v3, não repetido no v4) | `+16` px e `+64` px |
| Reinício com preset `compact` (build final) | **PASS** | abriu em 220 × 60 na posição persistida `1524,768` |
| Linux/macOS deste slice | **NOT_TESTED** | sem execução nativa |
| `node scripts/docs-verify.mjs` | **BLOCKED** | exige alterações visuais commitadas; o worktree está sujo |

## Capturas

- `01-expanded-380x560.png` — popover aberto com os controles de tamanho
- `02-preset-compact-334x496.png` — preset Compacto aplicado
- `03-preset-comfortable-380x560.png` — preset Confortável aplicado
- `04-preset-large-440x640.png` — preset Grande aplicado
- `05-back-to-compact-334x496.png` — retorno ao Compacto
- `06-collapsed-220x60.png` — cápsula compacta com a alça de arraste
- `07-after-drag.png` — cápsula após arraste (tooltip da alça visível)
- `08-restart-compact-220x60.png` — primeira pintura após reinício
- `result.json` — medições e veredito por verificação
- `state-after-restart.json` — estado persistido da execução anterior (v3)

## Limitações

- O arraste e os presets foram medidos por UI Automation em um único host Windows 11 `10.0.26200`, escala 100%.
- A captura do arraste usa o tooltip da alça, não um vídeo contínuo.
- O gate `docs-verify` continua bloqueado pela proveniência visual não commitada.
