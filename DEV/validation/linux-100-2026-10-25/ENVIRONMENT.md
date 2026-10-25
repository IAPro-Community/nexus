# Ambiente Linux da homologação

## Host

- Distribuição: Ubuntu 25.10
- Arquitetura: x86_64 / AMD64
- Kernel: Linux 6.17.0-23-generic
- Desktop: Ubuntu GNOME 49
- Sessão do usuário: Wayland
- Resolução: 1920×1080; escala XRandR 1×
- GPU: VMware SVGA II, sem aceleração 3D utilizável no ambiente
- Recursos: 8 CPUs lógicas, 60 GiB RAM e 4 GiB swap; 13 GiB disponíveis durante a coleta

## Ferramentas e bibliotecas

- Go 1.25.14 linux/amd64; `go.mod` declara 1.25.0
- Bun 1.4.2
- Node 22.17.0
- GCC 15.2.0
- Wails 3.0.0-beta.25
- GTK 3.24.50
- WebKitGTK 2.52.3 (`webkit2gtk-4.1`)
- GoReleaser 2.18.0

Nenhum pacote global foi instalado para a homologação. Caches, perfis e artefatos de teste ficaram em `/tmp`.

## Sessão gráfica e limitações

A sessão GNOME usa Wayland. O Desktop Wails e o HUD foram executados com `GDK_BACKEND=x11` em XWayland, usando o D-Bus real da sessão e HOME/XDG Nexus isolados; as duas janelas foram capturadas. O Desktop e o HUD também iniciaram diretamente com `GDK_BACKEND=wayland`. No HUD Wayland, expansão/recolhimento, seletor Barra/Mascote/Bandeja, tray e persistência da apresentação após reinício passaram. O Desktop abriu a página principal e expôs seus controles à árvore AT-SPI. A captura direta do compositor, a confirmação de alpha, o percurso físico de teclado, foco e movimento da janela continuam incompletos.

O canal alpha não pode ser concluído com as capturas disponíveis; elas são RGB e não preservam transparência do compositor. A VM não tem GPU utilizável para uma avaliação representativa de fluidez e consumo de animações.

## Evidências

Os comandos do host estão registrados em [ambiente da execução anterior](../linux-100-2026-10-24/logs/environment-command-output.txt). O overlay atual após FIX-012 tem 1.404 fontes; `logs/source-provenance-gdk-fix.txt` foi conferido contra tamanho e SHA-256 de cada arquivo (zero ausentes/divergentes) e corresponde ao fingerprint `516b706a2c3cdca1a6d76200833a785186063350fa4d41db4494fcb67bdb7f7a`. O resultado da rechecagem está em `logs/source-provenance-gdk-fix-verification.txt`. Manifestos anteriores ficam como histórico de overlays. Versões e hashes dos binários estão em [build-meta.txt](logs/build-meta.txt), [nexus-version.json](logs/nexus-version.json) e [build pós-FIX-012](logs/gdk-popup-fix-build-artifacts.txt).
