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
- Recheck de recursos (26/10, overlay FIX-032): a causa da CPU alta observada no overlay anterior foi reproduzida como ciclo de identidade no workspace; a deduplicação sem alterações criava um novo modelo, e o efeito de superfícies fixadas reiniciava renderizações. Corrigida em web/src/workspace/model.ts, com regressões. O Desktop Wails de produção, em perfil/D-Bus isolados e Xvfb/software, mediu 0,5% de um núcleo no WebKit principal por 12 s (RSS 456436→454408 kB); o Inspector pós-fix observou 0 escritas de document.title, um timer inicial de persistência e 0 mutações DOM em 6 s. A medição Xvfb não é comparação controlada com as amostras GNOME/Wayland históricas e não valida GPU; NR-010 segue BLOCKED. Log: logs/current-workspace-layout-loop-native-idle-xvfb-2026-10-26.txt; captura: screenshots/native-production-overview-idle-fixed-2026-10-26.png. As amostras de 111–116% e 853→999 MiB citadas em registros anteriores são pré-correção e permanecem como histórico, não como leitura do overlay atual.

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

## Reteste do Desktop instalado — 2026-10-25 local / 2026-10-26 UTC

O Desktop beta.10 e o HUD linkado foram reabertos no perfil normal após o teste isolado; o Core respondeu HTTP 200. Nesta sessão XWayland, `xrandr` reportou root `0×0` e `grim` não encontrou `wl_output`. Um teste funcional Wails separado passou em Xvfb com HOME vazio, mas não valida o compositor Wayland/XWayland do host.

Sem interação manual por 45 segundos, a unidade do Desktop passou de 853,5 MiB para 998,9 MiB e acumulou 57,4 segundos de CPU; o processo principal WebKit ficou em 115–116% de CPU média. Como a GPU está sem aceleração e provider/startup não foram isolados, isso não é um benchmark idle válido nem comprova vazamento. O gate de consumo permanece **BLOCKED** até medição controlada. Evidência: [amostra de recursos](logs/current-installed-desktop-resource-observation-2026-10-26.txt) e [restore do perfil](logs/current-installed-desktop-user-profile-restored-2026-10-26.txt).

## Sessão gráfica e limitações

A sessão GNOME usa Wayland. O Desktop Wails e o HUD foram executados com `GDK_BACKEND=x11` em XWayland, usando o D-Bus real da sessão e HOME/XDG Nexus isolados; as duas janelas foram capturadas. O Desktop e o HUD também iniciaram diretamente com `GDK_BACKEND=wayland`. No HUD Wayland, expansão/recolhimento, seletor Barra/Mascote/Bandeja, tray e persistência da apresentação após reinício passaram. O Desktop abriu a página principal e expôs seus controles à árvore AT-SPI. A captura direta do compositor, a confirmação de alpha, o percurso físico de teclado, foco e movimento da janela continuam incompletos.

O canal alpha não pode ser concluído com as capturas disponíveis; elas são RGB e não preservam transparência do compositor. A VM não tem GPU utilizável para uma avaliação representativa de fluidez e consumo de animações.

## Proveniência e amostra atual — 2026-10-26

- Base local/remota: 7a04f1b4cf5950b97b4ea1e156dda31ac513823b; overlay efetivamente compilado: eb038f52b8555c362a9129a3680746e0769ba4cbe733450b5c7a3b34a3ecbe05; 1.428 fontes. Manifesto: logs/source-provenance-workspace-idle-fix-2026-10-26.txt.
- Produção Linux Wails: Xvfb 1280×800, D-Bus e perfil isolados, HUD spawn/DevTools desativados; janela Overview aberta e capturada. Este caminho confirma inicialização/renderização nativa Wails sob X11 virtual, não o compositor Wayland da sessão real.

## Evidências

Os comandos do host estão registrados em [ambiente da execução anterior](../linux-100-2026-10-24/logs/environment-command-output.txt). O overlay atual após FIX-012 tem 1.404 fontes; `logs/source-provenance-gdk-fix.txt` foi conferido contra tamanho e SHA-256 de cada arquivo (zero ausentes/divergentes) e corresponde ao fingerprint `516b706a2c3cdca1a6d76200833a785186063350fa4d41db4494fcb67bdb7f7a`. O resultado da rechecagem está em `logs/source-provenance-gdk-fix-verification.txt`. Manifestos anteriores ficam como histórico de overlays. Versões e hashes dos binários estão em [build-meta.txt](logs/build-meta.txt), [nexus-version.json](logs/nexus-version.json) e [build pós-FIX-012](logs/gdk-popup-fix-build-artifacts.txt).
