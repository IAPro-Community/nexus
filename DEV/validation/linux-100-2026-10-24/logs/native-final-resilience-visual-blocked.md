# Tentativa inicial bloqueada pelo harness privado e reteste visual final aprovado

Data: 2026-10-25
Commit base: `bf6adf0c80ac3531d7291844512a14a017ab0002`
Fingerprint de fontes: `84bef2759af6b975d73e0488ab80e5fced2df70f17855597d512dcdef1f29ec8`
Binários tentados: Desktop `2d0c4af7b8717776e7b95133dfcdaa8f16e2495c3b243a9925053ad3aef6dcef`; HUD `57a30228ed3060c134a91d6ee7cf5202ae941799faae7960729f2ac54b2dca7d`.

## Observado

- Primeira tentativa: Desktop e HUD foram iniciados com perfis/XDG isolados e D-Bus privado; o Core de cada processo respondeu HTTP 200, mas essa sessão não mostrou as janelas.
- O compositor da sessão Wayland rejeitou a captura por `grim` (`wlr-screencopy-unstable-v1` não suportado).
- Forçando GTK para X11, a árvore XWayland expôs apenas uma janela shell/tray de 10×10 para cada aplicativo; a janela principal e a janela visual do HUD não ficaram disponíveis para captura.
- Nenhuma screenshot foi criada na tentativa inicial. Os processos de homologação foram encerrados pelas próprias sessões; os processos preexistentes do usuário não foram encerrados.

## Resultado inicial e correção do harness

A primeira tentativa usou uma sessão D-Bus privada e expôs apenas janelas shell/tray 10×10, então foi classificada como inconclusiva. O reteste sob o D-Bus real da sessão GNOME, mantendo HOME/XDG/perfil Nexus isolados e `GDK_BACKEND=x11`, expôs a janela Desktop 1280×800 e HUD 224×158. Ambos os Cores responderam HTTP 200; Desktop e HUD foram capturados com os hashes binários do overlay exato `84bef...`; o HUD mostra um NEX. Os processos de teste receberam SIGTERM e saíram com exit 0.

Evidências finais: `logs/native-final-resilience-visual-evidence.txt`, `screenshots/native-desktop-resilience-final-xwayland.png` e `screenshots/native-hud-resilience-final-xwayland.png`. As screenshots anteriores de `6476b22e...` continuam preservadas como evidência das interações de UI já exercitadas. O teste visual final passou; alpha confiável e Wayland nativo continuam sem validação.
