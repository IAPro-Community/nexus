# Validação do HUD nativo Linux

Commit-base: bf6adf0c80ac3531d7291844512a14a017ab0002. Worktree local dirty; ver relatório principal.

## Rebuild e reinício final — Go 1.26.10

Após reinstalar o build final, o HUD ligado foi iniciado como aplicação nativa Wails, PID `610408`, com Core em `127.0.0.1:34885`. `GET /api/v1/health` retornou HTTP 200 e `1.0.0-beta.10`; `/proc/610408/exe` tem SHA-256 `89b2b2f79ae0254d3ef23761f07ac9ee8b60f5325d4aeb437b80cc68f6706b52`, igual ao artefato instalado. O Desktop final (PID `565592`) e o HUD permaneceram em execução. Logs: `logs/processes-final-go1.26.10-bf6adf0c.log`.

O reinício confirma inicialização/health, não a matriz de interação. A captura nativa disponível continua mostrando o Guardian anterior e não foi tratada como Companion 2D.

## Executado

- PASS: build do CLI, Desktop Wails e HUD separado com tag GTK 3.
- PASS: Desktop iniciou como aplicação Wails nativa e criou Core embarcado saudável em 127.0.0.1:44085.
- PASS: serviço Nexus Web permaneceu ativo e saudável em 127.0.0.1:34717.
- PASS: executável HUD linked iniciou junto ao Desktop. A janela mostra o Guardian pixel art antigo, em fundo transparente, na captura XWayland screenshots/native-hud-guardian-xwayland-bf6adf0c.png.
- PASS: controles nativos foram abertos; a captura screenshots/native-hud-after-import-bf6adf0c.png exibe Barra, Mascote, Bandeja e o estado “Indisponível no Wayland”.
- PASS: testes automatizados do contrato nativo do HUD passaram em make quality e no reteste com Go 1.26.10.
- Logs dos builds, processos e health checks: logs/build-final-bf6adf0c.log e logs/process-refresh-final-bf6adf0c.log.

## Limites e bloqueios

- O compositor disponível foi GNOME Wayland. O popup destacado informa que não está disponível no Wayland; interação nativa independente, click-through e convivência do popup com outras janelas ficaram BLOCKED.
- A captura do HUD foi feita via XWayland; não existia sessão X11 independente para a matriz pedida. X11 separado: NOT_RUN.
- Não foi concluída a matriz de Barra/Mascote/Bandeja, startup automático, show/hide, minimização/restauração, popover separado, auto-hide, foco/blur, persistência de geometria/posição, múltiplos monitores e reinício do HUD em cada modo.
- A renderização observada é do Guardian antigo. Companion 2D: NO-GO — COMPANION_2D_NOT_INTEGRATED. Detalhes em COMPANION_VALIDATION.md.

## Evidência

- HUD visível: screenshots/native-hud-guardian-xwayland-bf6adf0c.png.
- Preferências e restrição Wayland: screenshots/native-hud-after-import-bf6adf0c.png.
- Hashes das capturas: logs/screenshots.sha256.
