# Ambiente Linux da homologação

## Host

- Distribuição: Ubuntu 25.10 (Questing Quokka)
- Arquitetura: x86_64 / AMD64
- Kernel: Linux 6.17.0-23-generic
- Desktop: Ubuntu GNOME 49
- Sessão do usuário: Wayland
- Resolução observada: 1920×1080, escala reportada pelo XRandR como 1×
- GPU: VMware SVGA II; sem aceleração 3D utilizável pelo ambiente de teste
- Recursos: 8 CPUs lógicas, 60 GiB RAM, 4 GiB swap; durante a coleta havia 13 GiB disponíveis

## Ferramentas

- Go disponível no PATH: go1.25.14 linux/amd64
- Toolchain efetiva observada nos builds: Go 1.25.14; go.mod declara go 1.25.0
- Bun: 1.4.2
- Node: v22.17.0
- GCC: Ubuntu 15.2.0
- Wails: v3.0.0-beta.25
- GTK: 3.24.50
- WebKitGTK: 2.52.3, pacote webkit2gtk-4.1
- GoReleaser: 2.18.0

O projeto não instalou ferramentas globalmente para esta homologação. O cache temporário de compilação foi criado sob /tmp/nexus-linux-100-2026-10-24.

## Desktop testado

A sessão do host é Wayland, mas a interface Wails foi executada com GDK_BACKEND=x11 no XWayland. Isso permitiu interação e capturas de tela nativas de GTK/WebKit, mas não comprova a interação direta com Wayland. GNOME Shell não ofereceu uma captura com alpha confiável pelo caminho disponível; capturas XWD mostraram preto nas regiões transparentes. Não se concluiu que o fundo visual está correto apenas a partir dessas imagens.

## Evidência de ambiente

O registro bruto de comandos está em logs/environment-command-output.txt. Os binários Desktop e HUD carregaram suas bibliotecas GTK/WebKitGTK sem entradas “not found” em ldd. A sessão não tinha aceleração gráfica suficiente para homologar animação e consumo GPU em condições representativas.

## Recaptura após correções de Core — 2026-10-25

A primeira tentativa usou um D-Bus privado e não expôs as janelas; o compositor e o processo ficaram indistinguíveis de uma falha de captura. Repetindo com o D-Bus real da sessão GNOME, mas perfil HOME/XDG Nexus isolado e GTK X11, a janela Desktop 1280×800 e a janela HUD 224×158 apareceram em XWayland. Ambos os binários exatos do fingerprint `84bef2759af6b975d73e0488ab80e5fced2df70f17855597d512dcdef1f29ec8` responderam HTTP 200 e foram capturados; o HUD mostra um NEX. A evidência está em `logs/native-final-resilience-visual-evidence.txt` e nas duas screenshots `native-*-resilience-final-xwayland.png`. A captura não valida Wayland nativo nem alpha.
