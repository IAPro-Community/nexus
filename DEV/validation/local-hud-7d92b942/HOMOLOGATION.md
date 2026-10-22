# Homologação Windows — release/v1.0.0 @ 7d92b942 — 2026-10-21

## Escopo

- Código avaliado: `7d92b9428f5493bab18b2c863532703827e5438f` (worktree limpo).
- HEAD de documentação após a captura sintética: `42a6971b` (somente `docs/assets`, mesmo código).
- Host: Windows 11 `10.0.26200`, AMD64, sessão gráfica desbloqueada, 1920×1080.
- Toolchain canônica: Go `1.26.10` (via `GOTOOLCHAIN=go1.26.10`), Node `24.21.0`, Bun `1.3.9`.
- Sem login em provedores, sem contas reais, sem tag, sem release e sem assinatura.

## Veredito

| Área | Veredito | Base |
|---|---|---|
| Gates automatizados de release | **PASS** | 9/9 gates do certificador oficial |
| Distribuição (instalador assinado) | **NO_GO / BLOCKED_EXTERNAL** | sem certificado Authenticode nem `native-signing` |
| HUD nativo Windows — função | **PASS** | 20/20 ciclos em duas apresentações |
| HUD nativo Windows — click-through | **NO_GO** | clique em área transparente capturado pelo HUD |
| HUD multiplataforma | **INCONCLUSIVO** | sem evidência Linux X11 e macOS no mesmo SHA |
| Release Windows | **NO_GO_WINDOWS** | instalador assinado + smokes interativos pendentes |

## Gates automatizados

| Gate | Resultado |
|---|---|
| `docs-verify` | PASS — 192 Markdown, 33 assets, manifesto em `7d92b942` |
| `verify-version-contract` | PASS — `1.0.0-beta.10` consistente |
| `gofmt -l cmd internal scripts` | PASS (após correção) |
| `go vet ./...` | PASS |
| `go test -count=1 ./...` (Go 1.26.10) | PASS — `internal/app` 88,9 s |
| Web `npm run quality` | PASS — 96 arquivos / 593 testes, typecheck, ESLint, Stylelint, Prettier |
| Web build | PASS |
| HUD build (`production`, ldflags) | PASS — SHA-256 `8dab24a7f616fe6fb720dc50d33cd6f6b6802328993614db2d987eafb48dcefb` |

### Defeitos encontrados e corrigidos nesta passagem

Commit `7d92b942`:

- `gofmt` pendente em `cmd/nexus-hud/main.go`, `internal/desktop/hud_window.go`, `internal/update/service.go`, `internal/update/service_test.go` e `internal/control/web/handlers_nexus_test.go`.
- Stylelint com 2 erros `custom-property-empty-line-before` em `web/src/features/hud/_hudGuardian.scss`.
- `nexus update` (fluxo de apply) retornava `update check failed` quando a pré-checagem falhava; o contrato documentado e o teste exigem `update could not be applied`.

Sem essas correções a branch reprovava em `gofmt`, `stylelint` e `go test ./internal/app`.

## Certificação oficial (`scripts/certify-windows-first-run.ps1`)

SHA exato `7d92b942`, execução limpa, exit code 0:

- PASS: `toolchain-consistency`, `powershell-installer-uninstaller-fixtures`, `project-directory-regression`, `conpty-terminal-regression`, `web-first-run-typecheck`, `web-first-run-unit-tests`, `build-cli-exact-source`, `read-only-provider-discovery`, `windows-project-registration-and-persistence`.
- NOT_RUN: `desktop-wails-first-run`, `codex-conpty-mouse-scroll-live`, `multiple-authenticated-codex-accounts`, `signed-installer-install-update-rollback`.
- `automated_result: PASS`; `release_verdict: NO_GO_WINDOWS`.

Resumo sanitizado: `%TEMP%\nexus-first-run-7d92b9428f54\WINDOWS_FIRST_RUN_SUMMARY.json`.

## HUD nativo — evidência interativa

### Barra (janela única)

- `scripts/homologate-hud-native-windows.ps1`, 20 ciclos, exit 0.
- `verdict: PASS_FUNCTIONAL_PENDING_VISUAL`, `cycles_completed: 20`.
- Geometria: compacto `220×60`, aberto `334×496`.
- Evidência: `native-windows-bar/` (`HUD_NATIVE_RESULTS.json`, `HUD_BEFORE.png`, `HUD_EXPANDED.png`).

### Popover destacado (`-Detached`)

- 20 ciclos, exit 0, mesmo `binary_sha256`.
- `verdict: PASS_FUNCTIONAL_PENDING_VISUAL`, `cycles_completed: 20`.
- Evidência: `native-windows-detached/`.

### Revisão visual com UI carregada

- Capturas próprias após 9 s de carga: `visual-load/HUD_LOADED_COMPACT.png` e `visual-load/HUD_LOADED_EXPANDED.png`.
- Observado: pill e painel com cantos arredondados, sem clipping, sem retângulo opaco visível, controles de tamanho (`Compacto`/`Confortável`/`Grande`) e modos (`Barra`/`Mascote`/`Bandeja`) renderizados.
- Revisão feita por agente automatizado; **confirmação humana ainda pendente** (`visual_shadow_review` permanece `NOT_TESTED` no JSON oficial).

### Click-through — FALHA

Teste real com janela alvo atrás do HUD (`click-through/HUD_CLICK_THROUGH.png`):

- HUD em `1688,964,220,60`; janela alvo cobrindo a região; HUD em foreground.
- Clique em `1696,972` (8 px dentro da janela, fora do pill — área transparente).
- Foreground após o clique: `Nexus HUD` (esperado: `Nexus ClickTarget`).
- **`CLICK_THROUGH=FAIL`**: a superfície transparente consome o clique.
- Causa provável: `TransparentWindow` do Wails deixa o WebView transparente, mas não há `WS_EX_TRANSPARENT`/hit-test nativo; `pointer-events` do CSS não atravessa a janela do SO.

O documento `docs/product/hud-native-visual-homologation.md` trata falha de click-through como **NO_GO**.

### Agregação cross-platform

`scripts/verify-hud-native-evidence.py` exige Windows + Linux X11 + macOS e `--output`; com apenas Windows o script recusa emitir veredito. Resultado: **INCONCLUSIVO** para a matriz HUD.

## CI remoto

- `CI` para `ae305de0`/`2c9d13e3` ficou `queued` e foi cancelado por pushes seguintes.
- `Windows release 1.0.0 exact-SHA homologation (no signing)` falhou em `e913d08a` antes dos testes:
  `ERR_TLS_CERT_ALTNAME_INVALID` ao baixar Chromium de `cdn.playwright.dev`
  (certificado apresentado é `*.azureedge.net`). Falha de infraestrutura/CDN, não de código.
- `Refresh synthetic documentation captures`: `success` em `ae305de0`.

## Bloqueios para GO_WINDOWS

1. **Assinatura Authenticode**: `native-signing` tem 0 secrets; não há certificado no store local; `signtool` e `makensis` ausentes.
2. **Sessão Codex interativa prolongada**: exige conta real; não executado.
3. **Quatro/cinco contas Codex autenticadas**: exige contas reais; não executado.
4. **Desktop Wails first-run**: evidência manual com screenshot ainda não produzida.
5. **HUD click-through**: defeito aberto; bloqueia o gate visual do HUD.

## Arquivos de evidência

```
DEV/validation/local-hud-7d92b942/
  HOMOLOGATION.md            (este relatório)
  native-windows-bar/        (20 ciclos, HUD_NATIVE_RESULTS.json + PNGs)
  native-windows-detached/   (20 ciclos, popover destacado)
  visual-load/               (UI carregada, compacto e aberto)
  click-through/             (teste real, resultado FAIL + PNG)
```
