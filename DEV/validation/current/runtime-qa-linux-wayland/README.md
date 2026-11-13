# Runtime QA — Linux Wayland nativo (parcial)

- **Data:** 2026-11-13
- **Ambiente:** Ubuntu 25.10, GNOME Wayland (`XDG_SESSION_TYPE=wayland`), WebKitGTK 2.52.3 (webkit2gtk-4.1)
- **Binários:** `~/.local/bin/nexus-desktop` e `~/.local/bin/nexus-hud` — build `e7268be6+` (go1.26.11, tags `production,gtk3`), conteúdo idêntico ao push `e1e2b750`
- **Contrato:** `docs/design/NEX_VISUAL_REVIEW_CHECKLIST.md` (gate `runtime_qa` do RELEASE_READINESS)

## Executado

| Check | nexus-desktop | nexus-hud |
|---|---|---|
| Inicia no Wayland e embute o Core | ✓ (`embedded Core at http://127.0.0.1:…`) | ✓ |
| WebKitGTK renderiza sem crash (25s/20s vivos) | ✓ | ✓ |
| Encerramento gracioso no SIGTERM (exit 0) | ✓ | ✓ |
| `nexus doctor` — 14 checks, 0 FAIL/WARN | ✓ (inclui platform.pty, webkitgtk, credentials DBus) | — |

## Observações honestas

- Avisos `libEGL`/`MESA-LOADER` = renderização por software neste ambiente sem GPU; não impedem render/click. **DPI/aceleração em GPU real não exercitados aqui.**
- **NÃO executado (NOT_RUN / pendente):** screenshot visual nativo (portal do GNOME exigiria interação), acessibilidade nativa (Orca), arrasto da janela HUD, click-through, BAR/MASCOT/TRAY switching, múltiplos monitores, auto-hide, Windows e macOS (exigem máquinas físicas do owner).
- Testes de interface web do HUD/companion: `web/scripts/e2e-companion-lab.mjs` + `e2e-hud.mjs` (PASS, ver `../companion-lab/`).
