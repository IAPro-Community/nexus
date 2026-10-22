# Homologação Windows — correção do click-through do HUD @ 047f4348 — 2026-10-21

## Escopo

- Código avaliado: `047f43481dfdcd6f2f77072cc2002ea8ee5209df` (worktree limpo durante o build).
- Passagem anterior: `7d92b942` — dossiê em `DEV/validation/local-hud-7d92b942/HOMOLOGATION.md`; o click-through falhava e o gate estava em NO_GO.
- Host: Windows 11 `10.0.26200`, AMD64, sessão gráfica desbloqueada, 1920x1080.
- Toolchain: Go `1.26.10` (`GOTOOLCHAIN`), Node `24.21.0`.
- Binário HUD: `nexus-hud.exe`, SHA-256 `19c5e299f5efeaa91e30deadda81e326571cac699ec19dd21147375992a1e6a5`, `production`, ldflags com versão/commit/data.
- Sem login em provedores, sem contas reais, sem tag, release ou assinatura.

## Correção

Causa raiz: a janela nativa do HUD é um retângulo, mas só a cápsula/painel é UI real. `pointer-events: none` no CSS não atravessa o hit-test do sistema operacional, então o Windows entregava o clique à janela do HUD mesmo em áreas transparentes.

Implementação:

- `web/src/features/hud/hudHitRegions.ts` coleta os retângulos interativos de `[data-hud-panel]` (o painel do HUD) em pixels CSS; elementos decorativos podem optar por sair com `data-hud-hit="ignore"`.
- `web/src/features/hud/useHudHitRegions.ts` envia os retângulos por `HUDController.SetHUDHitRegions` no mount, em mudanças de layout (`ResizeObserver`/`MutationObserver`, debounce 120 ms) e a cada 2 s.
- `internal/desktop/hud_clickthrough.go` mantém as regiões e um watcher de 30 ms. `internal/desktop/hud_clickthrough_windows.go` lê `GetCursorPos`, converte para DIP com `application.PhysicalToDipPoint` e compara com `window.Position()/Size()`.
- Quando o cursor está fora das regiões: `window.SetIgnoreMouseEvents(true)` (`WS_EX_LAYERED|WS_EX_TRANSPARENT`). Dentro delas: `SetIgnoreMouseEvents(false)`. O estado nunca é reaplicado sem mudança, para evitar flicker.
- Fail-open: com zero regiões (bridge lento, renderer quebrado) a janela permanece interativa; o cursor nunca fica preso numa janela invisível.
- Escopo: janela-cápsula. O popover destacado é uma segunda janela nativa com ciclo de foco próprio e mantém o comportamento anterior — click-through nele é `NOT_TESTED`.

## Veredito

| Área | Veredito | Base |
|---|---|---|
| HUD nativo — click-through (janela única) | **PASS** | 4 execuções reais; `RESULT.json` com corner/capsule/restore |
| HUD nativo — função (arraste, tamanhos, modos) | **PASS** | 20/20 ciclos, `PASS_FUNCTIONAL_PENDING_VISUAL` |
| HUD nativo — revisão visual | **PASS (agente)** | capturas com UI carregada; confirmação humana `NOT_TESTED` |
| HUD nativo — popover destacado (click-through) | **NOT_TESTED** | segunda janela não instrumentada |
| Gates automatizados | **PASS** | `gofmt`, `go vet`, `go test ./...`, Web quality, `docs-verify` |
| HUD multiplataforma | **INCONCLUSIVO** | sem Linux X11 e macOS no mesmo SHA |
| Release Windows | **NO_GO_WINDOWS** | instalador assinado + smokes interativos pendentes |

O click-through deixou de ser bloqueio do HUD. O veredito de release **não** muda: continua `NO_GO_WINDOWS`.

## Evidência de click-through

`DEV/validation/local-hud-047f4348/click-through-official/RESULT.json`:

- `hud_sha256` = `19c5e299...a1e6a5`, `commit` = `047f4348...`, `verdict` = `PASS`.
- `foreground_before` = `Nexus HUD`; `click_point` = `1696,972` (8 px dentro da janela, fora da cápsula).
- `click_through_wait_ms` = 147 (watcher ativo antes do clique; sem espera fixa artificial).
- `transparent_corner_click_through` = `true`; `capsule_click_through` = `false` (controle: a cápsula continua interativa).
- `click_through_restored` = `true` após sair e voltar ao canto.
- `foreground_after` = `Nexus ClickTarget`; `click_reached_target` = `true`.
- Captura: `HUD_CLICK_THROUGH.png`.

Antes da correção, o mesmo teste produzia `CLICK_THROUGH=FAIL` com `foreground_after=Nexus HUD` (`DEV/validation/local-hud-7d92b942/click-through/RESULT.json`).

Harness versionado: `scripts/homologate-hud-click-through-windows.ps1` (pwsh 7, SHA esperado opcional, exit code fail-closed).

## HUD nativo — função e visual

- `scripts/homologate-hud-native-windows.ps1`, 20 ciclos, exit 0: `PASS_FUNCTIONAL_PENDING_VISUAL`, compacto `220x60`, aberto `334x496`.
- Evidência: `native-windows-bar-fixed/`.
- `visual-load-fixed/HUD_LOADED_COMPACT.png` e `HUD_LOADED_EXPANDED.png`: pill e painel arredondados, sem clipping, modos `Barra`/`Mascote`/`Bandeja` e presets renderizados após o toggle de `WS_EX_LAYERED` (que não quebrou a composição do WebView2).
- Revisão por agente; confirmação humana segue pendente.

## Gates automatizados

| Gate | Resultado |
|---|---|
| `gofmt -l cmd internal scripts` | PASS |
| `go vet ./...` | PASS |
| `go test -count=1 ./...` (Go 1.26.10) | PASS |
| Web `npm run quality` | PASS — 97 arquivos / 598 testes, typecheck, ESLint, Stylelint, Prettier |
| Web build | PASS |
| HUD build (`production`, ldflags) | PASS — SHA-256 `19c5e299f5efeaa91e30deadda81e326571cac699ec19dd21147375992a1e6a5` |
| `docs-verify` | PASS — 192 Markdown, 33 assets, manifesto em `047f4348` |

Testes novos: `TestHUDPointerWithinRegions*` (Go) e `hudRegionFromCandidate` (Vitest), cobrindo margem transparente, bordas, múltiplas regiões, ausência de regiões (fail-open) e filtros de visibilidade.

## Defeitos encontrados durante a correção

1. `collectHudHitRegions(document)` retornava `[]` porque `document.ownerDocument` é `null`; corrigido com fallback para o próprio documento.
2. A primeira heurística (todo elemento com `pointer-events: auto` até a fronteira `none`) reportava o container da aplicação inteiro, mantendo a janela toda interativa; substituída por opt-in explícito `[data-hud-panel]`.
3. O harness PowerShell precisou de parênteses em literais `@(a - 1, b - 1)` (o parser tratava a vírgula antes da subtração) e de elevar a janela-alvo antes do HUD, senão o clique atravessava para uma janela intermediária (Edge).
4. O teste inicial clicava após um sleep fixo e falhava quando o renderer ainda não havia reportado as regiões; agora espera o estado nativo e valida o controle da cápsula.

## Pendências e bloqueios (inalterados)

- Instalador assinado: `native-signing` com 0 secrets, sem PFX local, sem `signtool`/`makensis` — `BLOCKED_EXTERNAL`.
- Gates `NOT_RUN`: `desktop-wails-first-run`, `codex-conpty-mouse-scroll-live`, `multiple-authenticated-codex-accounts`, `signed-installer-install-update-rollback`.
- CI remoto vermelho por infraestrutura (`ERR_TLS_CERT_ALTNAME_INVALID` ao baixar Chromium de `cdn.playwright.dev`).
- Matriz HUD cross-platform `INCONCLUSIVO` (sem Linux X11/macOS no mesmo SHA).
- Popover destacado com click-through `NOT_TESTED`; confirmação visual humana `NOT_TESTED`.

## Próximos passos

1. Validar click-through no popover destacado (segunda janela) e, se necessário, instrumentá-lo por janela.
2. Obter PFX + senha no environment `native-signing` e rodar o instalador assinado + rollback.
3. Executar os smokes interativos de Codex e a matriz Linux X11/macOS no mesmo SHA.
