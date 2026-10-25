# Linux homologation environment

- Date: 2026-10-22 local time (America/Cuiaba)
- Repository branch: `release/v1.0.0`
- Remote and local source SHA: `f62fdec0c571ac6a19a904164d3f580799da6bf3`
- Architecture: `x86_64` / AMD64
- Distribution: Ubuntu 25.10
- Kernel: Linux `6.17.0-23-generic`
- Desktop: GNOME 49.0, Mutter, Wayland session; XWayland is present, but no separate X11 login session was available.
- Display: one 2560x1080 monitor; display scaling was not independently verified and no second monitor was available.
- CPU and memory: 8 virtual CPUs on VMware / AMD EPYC 7313 host; about 60 GiB RAM visible to the session.
- Graphics: VMware SVGA II (`vmwgfx`, `/dev/dri/card0` and `renderD128`). WebKit reported no 3D acceleration and failed DRI2/DRI3 initialization; GPU acceleration could not be certified.
- Go: default environment `go1.25.14`; homologation builds/tests used pinned `go1.26.10` (project toolchain).
- Bun: `1.3.9` (project-pinned).
- Node.js: `v22.17.0`.
- Wails module: `v3.0.0-beta.25`; the separately installed Wails v2 CLI is not used by the project build.
- Native build libraries: GTK 3.24.50, WebKitGTK 4.1 / WebKit 2.52.3, libsoup 3.6.5, GLib/GIO 2.86.0. `pkg-config` checks for `gtk+-3.0`, `webkit2gtk-4.1`, `libsoup-3.0`, and `gio-unix-2.0` passed.
- Build tags: `production,gtk3` for the Wails Desktop and HUD targets.
- Desktop tests used temporary `NEXUS_DATA_DIR` and XDG config/data/cache paths. Native Desktop and its linked HUD were started from the built artifacts; the embedded Core returned HTTP 200.
- Wayland permissions allowed launch and AT-SPI inspection. Native screen capture was unavailable: `grim` reported that the compositor does not support `wlr-screencopy-unstable-v1`; GNOME Shell's screenshot method returned `AccessDenied: Screenshot is not allowed`; Flameshot timed out. No screenshot files were fabricated.
- AT-SPI exposed the native window title and the live home/onboarding content. Invoking an accessible action timed out, and no native pointer/keyboard automation was available in this session, so native navigation and settings changes remain unverified.

