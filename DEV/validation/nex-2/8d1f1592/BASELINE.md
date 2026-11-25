# Nex 2.0 — Phase P0 (NEX-00) Reproducible Baseline

- **Date (UTC):** 2026-11-25
- **Authoritative Integration Base:** `origin/release/v1.0.0` @ `8d1f1592ce8c4b12afa66585232ff2d09bd41d93`
- **Isolated Worktree:** `/projetos/worktrees/nex-2-foundation` (detached HEAD; primary `/projetos/nexus` dirty worktree at `91752a84` preserved untouched)
- **Repository Version (`VERSION`):** `1.0.1`
- **Desktop Runtime Dependency (`go.mod` / `web/package.json`):** `github.com/wailsapp/wails/v3 v3.0.0-beta.25`, `@wailsio/runtime 3.0.0-beta.25`, Go `1.26.11`, Node `v22.17.0`, Bun `1.3.9`

## 1. Host Environment & Installed Binaries

| Field | Observed Value |
|---|---|
| Host OS / Kernel | Ubuntu 25.10 (Questing Quokka), Linux `6.17.0-23-generic #23-Ubuntu SMP PREEMPT_DYNAMIC x86_64` |
| CPU / RAM | 8 logical cores, 60 GiB RAM, 4.0 GiB swap |
| GPU | `VMware SVGA II Adapter` |
| Display / Compositor | `XDG_CURRENT_DESKTOP=ubuntu:GNOME`, `XDG_SESSION_TYPE=wayland`, `WAYLAND_DISPLAY=wayland-0`, `DISPLAY=:0` (XWayland) |
| Native Toolkits (`pkg-config`) | `webkit2gtk-4.1` `2.52.3`, `gtk+-3.0` `3.24.50`, `gtk4` `4.20.1`; `webkit2gtk-4.0`, `Qt6Quick`, `Qt5Quick` not installed |
| Installed `nexus` | `/home/desenvolvedor/.local/bin/nexus` — `IAPro Nexus 1.0.1 (linux/amd64, go1.26.11) commit bff681fd built 2026-11-24T18:42:01Z` |
| Installed `nexus-hud` | `/home/desenvolvedor/.local/bin/nexus-hud` — `1.0.1` commit `bff681fd` built `2026-11-24T19:19:34Z` (Wails `v3.0.0-beta.25`) |
| Installed `nexus-desktop` | `/home/desenvolvedor/.local/bin/nexus-desktop` — built `2026-11-24 15:19` local |
| Autostart Config | `/home/desenvolvedor/.config/autostart/nexus-hud.desktop` (`Exec=/home/desenvolvedor/.local/bin/nexus-hud`) |

## 2. Branch Reconciliation vs `origin/release/v1.0.0` (`8d1f1592`)

| Ref | Observed SHA | Merge Base | Left/Right (`base...ref`) | Status & Reconciliation Verdict |
|---|---|---|---|---|
| `main` | `ab1efac2c80dddb7b3aa60016baf1d4791bc3203` | `61b87409ba` | `2651 \| 6` | Ahead by 6 CI/MIT-license commits only; 0 HUD/Nex diffs vs base; older HUD (Wails v2). **Not base.** |
| `release/v1.0.0` | `8d1f1592ce8c4b12afa66585232ff2d09bd41d93` | `8d1f1592ce` | `0 \| 0` | **Authoritative integration base.** Contains Wails v3 HUD, Guardian, Nex Assistant/Conversation/Proactivity, and default-off advanced flags. |
| `release/v1.1.0` | `c01cd04a07b91afc1ed71ad480abb6e220ee98a2` | `0ac9d8ed91` | `1413 \| 252` | Dated `2026-10-22` (`VERSION`: `1.0.0-beta.10`). Missing 40 HUD/Nex files (GuardianMascot, NexAssistant, NexProactivity, click-through, attention monitor). **Do not merge.** |
| `safety/advanced-flags-default-off-20261124` | `dede6f0aeeaf64f5882a42e2231d7f9b0c9bc230` | `dede6f0aee` | `52 \| 0` | Direct ancestor of `origin/release/v1.0.0`. **Already integrated.** |
| `feat/nex-durable-conversation-v1-20261117` | `acd3c9355c22322d50ed2c5210a30525dc9b6541` | `ad3ec0a3f4` | `331 \| 6` | Content already integrated + extended in `origin/release/v1.0.0` (`0037_nex_conversation.sql`, `nex_conversation.go`, `handlers_nex_conversation.go`). Only minor comment/error-casing diffs remain; base also includes proactivity & action intents. |
| `feat/nex-proactivity-core-20261117` | `8077969b5778428897edbd2906fd0bd81a05bc4b` | `3fee69fe4a` | `293 \| 14` | Content already integrated + extended in `origin/release/v1.0.0` (`0038_nex_proactivity.sql`, `nex_proactivity.go`, `handlers_nex_proactivity.go`, `NexProactivityControls.tsx`). Base adds `featureMissions` gate (`handlers_nex_proactivity.go:166`). |
| `feat/nex-safe-conversational-actions-20261117` | `f2ee29683732c987e19ce8147d067e30876b614c` | `7dc1914376` | `330 \| 4` | Content already integrated in `origin/release/v1.0.0` (`nex_action_intents.go`, `NexAssistantPanel.tsx` action proposals). |

## 3. Live Linux Process-Tree Resource Profile (Read-Only Observation)

Sampled across 6 snapshots at 3-second intervals (`23:53:37`–`23:53:54`) on the live host processes (unmodified `1.0.1` commit `bff681fd`):

| Process Tree | PID | RSS Range (KiB) | PSS (KiB) | CPU % | Threads | Notes |
|---|---|---|---|---|---|---|
| `nexus-hud --linked` (Go/Wails/GTK) | `500708` | `189,472 – 189,664` | `84,078` | `0.4%` | 64 | Hosts native window + tray + `hudAttentionMonitor` |
| `WebKitNetworkProcess` (HUD child) | `500882` | `48,496` | `12,005` | `0.0%` | 10 | Network/cache child |
| `WebKitWebProcess` (HUD child) | `500885` | `489,388 – 490,372` | `362,065` | `0.5%` | 33 | Loads full Workspace OS bundle (`bundle.js` 457.1 kB + `bundle.css` 484 kB + `sw.js`); RSS grew `+568 KiB` in 15s idle |
| **HUD Process-Tree Total** | — | **`~727,356 – 728,532` (~711 MiB)** | **`458,148` (~447 MiB)** | **`~0.9%`** | **107** | Far exceeds Nex 2.0 target (`<= 60 MiB` incremental) |
| `nexus-desktop` (+ children `500816`/`500817`) | `500690` | `~849,128` (~829 MiB) | `564,745` (~551 MiB) | `~0.6%` | 116 | Main desktop shell |
| `nexus web --port 0 --listen 127.0.0.1 --no-open` | `4086703` | `23,640` (~23.1 MiB) | `21,982` (~21.5 MiB) | `0.0%` | 13 | Shared Core HTTP server |

## 4. Reproduced Linux Defects & Architectural Bottlenecks

1. **Full Workspace OS Bundle & Service Worker Loaded for HUD (`web/src/index.tsx`, `web/src/app/DesktopHudApp.tsx`)**:
   - `nexus-hud` opens `/?nativeHud=1`, which boots the same `web/src/index.tsx` entrypoint as the full Workspace OS (`bundle.js` 457.1 kB, `bundle.css` 484 kB embedded, `/sw.js` service worker, and React router/shell chunk graph), driving `WebKitWebProcess` to `~354 MiB` PSS / `~479 MiB` RSS.
2. **Continuous 60 FPS Idle RAF Loop & Large Layer Decoding (`web/public/nex/rig-v1/nex-rig-player.mjs:196`)**:
   - `const fpsInterval = this.smallScreen.matches ? 1 / 30 : 1 / 60;` runs a continuous 60 FPS `requestAnimationFrame` canvas redraw loop even during `idle` mode, compositing 1254x1254-derived PNG layers (`body_base.png` 1.49 MB + 9 layer PNGs).
3. **Animation/Gesture-Driven Window Resize & State Mutation (`internal/desktop/hud_window.go:730-769`, `web/src/features/hud/NexusHud.tsx:418-430`)**:
   - `NexusHud.tsx` calls `hudV3.setBubbleVisible(true/false)` whenever `mascotBubble` opens or dismisses, invoking `HUDController.SetHUDBubbleVisible`, which calls `window.SetSize`, `window.SetPosition`, and `h.manager.Save(state)`. Transient UI bubbles mutate persistent window geometry and trigger resize/position races.
4. **Triple Startup `RestoreHUDWindow` Race & Blur Persistence (`cmd/nexus-hud/main.go:148-152`, `internal/desktop/hud_window.go:400,410`, `web/src/app/DesktopHudApp.tsx:84`)**:
   - `ServiceStartup` registers `restoreHUDWindowWhenShown`, calls `h.RestoreHUDWindow()` immediately, and `cmd/nexus-hud/main.go` registers another `initialRestore.Do(func() { go hudController.RestoreHUDWindow() })` on `WindowShow`. Meanwhile `DesktopHudApp.tsx` calls `hudV3.captureWindowState()` on every `window` `blur` event, persisting intermediate resize/focus states (`Placement = HUDPlacementCustom`).
5. **Wayland Capability Constraints (`internal/desktop/hud_window.go:273-274,314-316`, `internal/desktop/hud_clickthrough_linux.go`)**:
   - On GNOME Wayland (`hudSession() == "wayland"`), `GlobalPositioning` is `false` and `detached` is forced `false` (`hud_window.go:314`), forcing single-window expansion that resizes the mascot window directly (`SetHUDExpanded`). `hudHitTestingSupported()` returns `false` on Linux while `applyHUDHitRegions` calls `gtk_widget_input_shape_combine_region` on the GTK widget.
6. **Baseline Playwright HUD E2E Regression on `8d1f1592`**:
   - `npm --prefix web run test:e2e:hud` failed at `web/scripts/e2e-hud.mjs:136` (`Timeout 5000ms waiting for [data-mascot-attention-list]`): `e2e-hud.mjs:26` only enables `['NEX_PROACTIVITY']`, whereas `NexusHud.tsx:171` gates `nexus.getAttention()` behind `COMPOSER_FLOW_ENABLED` (default OFF since `#234`).
   - `npm --prefix web run test:e2e:companion-lab` failed at `web/scripts/e2e-companion-lab.mjs:333` (`AssertionError: fixture failure must reach the HUD via poll (last=attention)` expected `critical`) for the same reason (`startFixtureServer()` leaves `COMPOSER_FLOW_ENABLED` false, blocking `attentionGroup.failed` fetch).

## 5. Maestro, IAPro SDK & NEX Visual Lock Status

- **Maestro (`internal/nexus/maestro.go`, `internal/control/handoff/maestro_*.go`)**: Optional external JSON-RPC stdio bridge (`orquestrador-maestro bridge --stdio`) with capability negotiation (`projectContext`, `contextBrief`, `plansRoutePreview`, `runsSubmissionsPoll`, `runsSubmissionsCancel`, `runsSubmit`, `projectCompose`, `memory.read`). Nexus Core and HUD remain functional without Maestro.
- **IAPro SDK**: Not present as a dependency in `go.mod` or `internal/` on `8d1f1592` (separate unmerged exploration exists in `/projetos/worktrees/nexus-iapro-sdk`).
- **NEX Identity Lock (`scripts/verify-nex-identity-lock.mjs`)**: `PASS` (`status: FROZEN`, lock `NEX-ID-v1`; `selected_reference` binary `NOT_VERSIONED`, true pixel-art production master still pending).
- **NEX Joint Release Readiness (`scripts/verify-nex-release-readiness.mjs`)**: `BLOCKED (2/11 gates); public site: false`. `--require-ready` exits `1` as required.

## 6. Start / Stop / Recovery & Rollback Contract (P0)

- **Default State (Fail-Closed):** `NEX_V2` is registered in `internal/nexus/feature_flags.go` (`advancedFeatures`) and `internal/desktop/nex_v2_mode.go`, defaulting to `false` unless **both** `NEXUS_FEATURES_ENABLED=true` and `NEXUS_FEATURE_NEX_V2=true` are set.
- **Mutual Exclusion:** `cmd/nexus-hud` uses a single `SingleInstance` ID (`com.iapro.nexus.hud`). `desktop.ActiveHUDPresenter()` and `desktop.ValidateHUDPresenterMutualExclusion` ensure Legacy HUD (`legacy`) and Nex 2.0 HUD (`nex_v2`) never run concurrently in the same session.
- **Start / Stop / Recovery:** `OpenSharedBackend` attaches to an existing authenticated loopback Core (`web.ReadListenState`) if healthy, or launches a single embedded Core stopped cleanly in `OnShutdown`. Renderer failures never terminate user Agents or standalone Core.
- **Rollback:** Unset `NEXUS_FEATURE_NEX_V2` (or `NEXUS_FEATURES_ENABLED`) and restart `nexus-hud`. Legacy `hud-window-state.json`, SQLite tables (`0037_nex_conversation`, `0038_nex_proactivity`), and Web HUD remain intact.
