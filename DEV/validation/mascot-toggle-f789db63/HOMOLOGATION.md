# Homologação Windows — toggle do popup independente e Mascote @ f789db63 — 2026-10-22

## Escopo

- Código avaliado: `f789db63c8ec9f3498c3822a94ee7c823ec7086c` (worktree limpo durante o build).
- Binário: `nexus-hud.exe`, SHA-256 `1f3ced07acc02672dd786c3c39cf87a390da9f315e0e654a9e2505f2ab7929de`, tag `production`, ldflags com versão/commit/data.
- Host: Windows 11 `10.0.26200`, AMD64, sessão gráfica desbloqueada, 1920x1080, escala 100%.
- Toolchain: Go `1.26.10` (`GOTOOLCHAIN`), Node `24.21.0`.
- Sem login em provedores, sem contas reais, sem tag, release ou assinatura.

## Problema

O modo **Mascote** existia no produto, mas era inalcançável na instalação:

1. `SetHUDAppearance("MASCOT")` exige o popup nativo independente (`internal/desktop/hud_appearance.go`).
2. O popup só era criado quando `NEXUS_HUD_DETACHED_POPOVER=1` estava definido no start (`cmd/nexus-hud/main.go`).
3. Nenhum caminho de produção define essa variável: `hudlink.Spawn()` (usado por `nexus.exe` e `nexus-desktop.exe`) define apenas `NEXUS_HUD_LINKED=1`. Somente o harness de homologação a definia com `-Detached`.
4. A UI refletia o gate: o botão **Mascote** ficava desabilitado com o tooltip "Mascote requer popup independente".

## Correção

- `HUDWindowState.DetachedPopover` persiste a escolha em `hud-window-state.json`.
- `HUDController.Get/SetHUDDetachedPopoverPreference` aplica a escolha **em runtime**, sem reiniciar o app:
  - habilitar cria a segunda janela nativa sob demanda (`EnsureDetachedPopup` + factory instalada pelo binário);
  - desabilitar esconde o popup, restaura a barra e devolve a cápsula ao tamanho compacto;
  - ambas as rotas falham fechadas: se a janela não puder ser criada, nada é persistido.
- Ao habilitar, a cápsula volta ao tamanho compacto preservando a âncora inferior direita (antes ela ficava no tamanho expandido da janela única).
- `main.go` passa a considerar `env || preferência persistida` no start e reutiliza a mesma factory do toggle.
- Frontend (`NexusHud.tsx`): botão **"Popup independente: Sim/Não"** no rodapé do painel (cápsula e popup), com estado pendente e toast.
- Sincronização entre as duas WebViews: o poll de 800 ms passou a reler `getDetachedPopoverEnabled()` além da aparência, corrigindo o estado velho do capsule depois de uma mudança feita no popup (o capsule ficava com `expanded`/`detached` antigos e podia renderizar a barra dentro da janela do mascote).

## Veredito

| Área | Veredito | Base |
|---|---|---|
| Toggle do popup (habilitar em runtime) | **PASS** | UI Automation real; `RESULT.json` |
| Persistência após reinício | **PASS** | Sessão B abre em `MASCOT` + popup habilitado |
| Mascote disponível e aplicável | **PASS** | botão habilita, janela vira `136x112` (ver nota) |
| Volta para a barra e painel inline | **PASS** | toggle off + clique na cápsula |
| Testes automatizados | **PASS** | `gofmt`, `go vet`, `go test ./...`, Web quality 97/599 |
| Click-through (janela única) | **PASS (herdado)** | `DEV/validation/local-hud-047f4348/click-through-official/RESULT.json` |
| Click-through no popover destacado | **NOT_TESTED** | watcher permanece desativado em modo detached |
| Confirmação visual humana | **NOT_TESTED** | capturas geradas por agente |
| HUD multiplataforma | **INCONCLUSIVO** | sem Linux X11 e macOS no mesmo SHA |
| Release Windows | **NO_GO_WINDOWS** | instalador assinado e smokes interativos pendentes |

Nota de plataforma: `HUDMascotWidth = 108` DIP, mas o Windows limita a largura de janelas redimensionáveis a `SM_CXMINTRACK`. Neste host `SM_CXMINTRACK = 136`, então a janela do mascote mede `136x112` físicos (altura exata). O conteúdo é centralizado e a margem sobrando é click-through. O harness calcula o valor esperado como `max(108 * escala, SM_CXMINTRACK)`.

## Evidência (`homologation/RESULT.json`)

- `hud_sha256` = `1f3ced07...7929de`, `commit` = `f789db63...7086c`, `verdict` = `PASS`.
- Sessão A (instalação limpa): `a_state_detached_before=true`, `a_toggle_before="Popup independente: Não"`, `a_mascot_disabled_before=true`.
- Habilitar: `a_state_detached_on=true`, `a_popup_window_created=true`, `a_popup_hidden_before_expand=true`, `a_capsule_restored=true` (`1688,964,220,60`).
- Expandir: `a_popup_visible_on_expand=true` (popup `1574,460,334,496`), `a_capsule_kept_compact=true`.
- Mascote: `a_mascot_enabled_after=true`, `a_state_appearance_mascot=true`, `a_capsule_mascot_size="136x112"`, `a_capsule_renders_mascot=true`, `a_popup_hidden_after_mascot=true`.
- Reinício: `b_state_restored=true`, `b_capsule_at_start="1812,912,136,112"`, `b_popup_created_at_start=true`, `b_popup_hidden_at_start=true`.
- Desabilitar no popup: `b_toggle_off_before="Popup independente: Sim"`, `b_state_detached_off=true`, `b_capsule_after_disable="1728,964,220,60"`, `b_popup_hidden_after_disable=true`.
- Regressão do painel inline: `b_inline_panel_after_disable=true`, `b_toggle_after_disable="Popup independente: Não"` (janela `334x496`).

Capturas: `homologation/MASCOT.png` (cápsula em modo mascote) e `homologation/BAR_RESTORED.png` (barra restaurada com o painel inline e o toggle em "Não").

## Harness

`scripts/homologate-hud-mascot-toggle-windows.ps1` opera a WebView real por UI Automation (sem mocks), com estado isolado (`NEXUS_DATA_DIR`/`NEXUS_CONFIG_DIR`/`NEXUS_STATE_DIR`), `ExpectedSha` obrigatório na prática e duas sessões para provar persistência. Falha fechada: qualquer check ausente gera `FAIL` e exit não-zero.

## Limites e pendências

- O clique/toque foi dirigido por UI Automation; não houve confirmação visual humana das capturas.
- O modo detached continua com click-through desativado na segunda janela; o popover é um painel sólido, mas arraste em área transparente do popup não foi medido.
- A matriz Linux X11/macOS não foi executada neste SHA.
- O veredito de release não muda: `NO_GO_WINDOWS` (assinatura Authenticode, instalador, smokes interativos de Codex e multi-conta seguem pendentes).
