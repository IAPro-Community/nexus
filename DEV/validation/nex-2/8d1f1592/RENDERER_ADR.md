# Nex 2.0 — Renderer Architecture Decision Record (ADR / P1)

- **Status:** Accepted for R1/Nex 2.0 desktop shell (behind default-OFF `NEX_V2` feature flag)
- **Date:** 2026-11-25
- **Baseline Base:** `origin/release/v1.0.0` (`8d1f1592`) + P0 (`fb22bbb7`)
- **Worktree:** `/projetos/worktrees/nex-2-foundation`

## 1. Decision

Select **Wails v3 (`v3.0.0-beta.25`) with an isolated lightweight entrypoint (`/nex-v2.html`)** as the single production desktop runtime for Nex 2.0, rejecting Qt Quick/QML for the current release path.

## 2. Measured Evidence (Wails v3 Lightweight Candidate vs Legacy Wails HUD)

### 2.1 Bundle & Surface Isolation
| Artifact | Legacy HUD (`/?nativeHud=1`) | Nex 2.0 Entrypoint (`/nex-v2.html`) | Reduction |
|---|---|---|---|
| Entrypoint JS | `dist/bundle.js` (`458 KB` + 64 split Workspace chunks) | `dist/nex-v2.js` (`264.4 KB`, 0 Workspace chunks) | Zero Workspace/xterm/xyflow/Flow trees |
| Entrypoint CSS | `dist/bundle.css` (`484 KB`) | `dist/nex-v2.css` (`7.8 KB` / `14 KB` on disk) | **~97% CSS reduction** |
| Service Worker | Registers `/sw.js` | None | Eliminated SW cache overhead |
| Eager Popover WebView | Spawns 2nd WebView on X11/Windows/macOS | Deferred (lazy panel on demand) | Avoids eager 2nd WebProcess |

### 2.2 Motion, Clipping & Frame Pacing (Playwright P1 Lab — `p1-lab/p1-lab-metrics.json`)
- **Cold Navigation to Paint:** `329 ms` (rig first paint: `155 ms`, within `<= 1,000 ms` budget).
- **Idle Motion Pacing:** `20 FPS` target (`50 ms` interval), within `15–30 FPS` target (down from 60 FPS continuous idle RAF in legacy `nex-rig-player.mjs`).
- **Interactive Frame Pacing:** `60 FPS` target, `p50FrameMs = 16.7 ms`, `p95FrameMs = 16.8 ms` (`<= 33 ms` budget), `longStuttersOver100ms = 0`.
- **Zero Clipping & Geometry Stability:** `edgeClip = 0` across all 5 operational moods (`idle`, `working`, `attention`, `critical`, `offline`), `resizeCountDelta = 0`, and unit tests (`nexEnvelope.test.ts`) verify `isWithinSafeEnvelope` across all 9 expressions × 24 time samples × orbit phases × presets (`128`/`160`/`192`) × DPIs (`1.0`/`1.25`/`1.5`/`2.0`).
- **Pause / Reduced Motion:** `reducedMotionFramesDelta = 0` (`targetFps = 0`, clock mode `paused`).

### 2.3 Native Linux Process-Tree Memory & CPU (`Xvfb :97` + isolated `dbus-run-session`, software Mesa, 11s warm-up)
| Process | Legacy Live HUD (GNOME Wayland idle) | Nex 2.0 Native Lab (`Xvfb`, cold->11s) | Notes |
|---|---|---|---|
| Go/Wails Host (`nexus-hud` / `nex2lab`) | `189.5 MiB` RSS / `84.1 MiB` PSS / `0.4%` CPU | `176.0 MiB` RSS / `93.2 MiB` PSS / `5.6% -> 3.4%` CPU | Includes embedded Core (`~22 MiB` PSS) when standalone |
| `WebKitNetworkProcess` | `47.4 MiB` RSS / `11.7 MiB` PSS / `0.0%` CPU | `45.7 MiB` RSS / `14.6 MiB` PSS / `0.7% -> 0.3%` CPU | Network process baseline |
| `WebKitWebProcess` | `478.9 MiB` RSS / `353.6 MiB` PSS / `0.5%` CPU | `284.2 MiB` RSS / `202.9 MiB` PSS (`135.4 MiB` `Private_Dirty`) | **-195 MiB RSS / -151 MiB PSS (~43% PSS drop)** vs legacy |
| **Total Process Tree** | **`~711 MiB` RSS / `~447 MiB` PSS** | **`~506 MiB` RSS / `~311 MiB` PSS** | `-136 MiB` PSS reduction even with embedded Core |

### 2.4 Documented Platform Memory Exception (`<= 60 MiB` Target)
- The `<= 60 MiB` incremental process-tree memory target is **not achievable with WebKitGTK (`webkit2gtk-4.1` 2.52.3)** on Linux because the WebKitWebProcess + WebKitNetworkProcess runtime baseline alone requires `~143 MiB` `Private_Dirty` (`~217 MiB` PSS) even for a minimal `<canvas>` + 14 KB stylesheet, and the Go/Wails host binary adds `~70.7 MiB` `Private_Dirty`.
- **Decision & Trade-off:** Accept this platform baseline exception for Linux WebKitGTK (while continuing to keep the lazy interaction panel split out and avoiding any eager 2nd WebView), rather than introducing a second UI stack. Measure Windows (WebView2) and macOS (WKWebView) on real physical hardware before promotion.

## 3. Qt Quick/QML Comparison & Rejection Rationale

| Criterion | Wails v3 Isolated Entrypoint (Selected) | Qt Quick / QML (Deferred Contingency) |
|---|---|---|
| Host Availability | Installed (`wails v3.0.0-beta.25`, `webkit2gtk-4.1` `2.52.3`, `gtk+-3.0` `3.24.50`) | `Qt6Quick` / `Qt5Quick` not installed on host |
| Licensing & Redistribution | Permissive Go/MIT + system WebView (`WebKitGTK`/`WebView2`/`WKWebView`) | LGPLv3 / GPLv3 dynamic linking & source-distribution compliance burden, or commercial license |
| Packaging & Toolchain | Single Go binary (`~46 MB`), zero new C++/CMake/QML build chain, reuses existing updater | Adds C++/QML cross-compile toolchain, Qt runtime DLLs/frameworks (`+40–100 MB` installer delta), and separate IPC process |
| Wayland Capabilities | Capability-detected (`GlobalPositioning=false`, tray/anchored panel fallback) | **Cannot bypass Wayland compositor security** (no global positioning/click-through override either) |
| Shared Core / Auth | Directly binds `OpenSharedBackend` + existing loopback/CSRF auth | Would require out-of-process IPC bridge + separate UI test harness |

**Conclusion:** Qt Quick/QML does not unlock Wayland capabilities and introduces severe dual-stack maintenance and LGPLv3 packaging costs. Keep one active renderer path (Wails v3 isolated `/nex-v2.html`). Revisit Qt only if physical hardware testing proves Wails v3 fails CPU or stability gates after P2 optimizations.

## 4. Art Provenance & Gate Status

- **Character Identity (`NEX-ID-v1`):** Uses locked `/nex/hybrid-v1/` layers and `/nex/hud-32bit/nex-idle-atlas-128.png` fallback byte-for-byte (`node scripts/verify-nex-identity-lock.mjs` PASS, `FROZEN`).
- **Joint Release Readiness (`RELEASE_READINESS.json`):** Remains `BLOCKED (2/11)`. Owner visual approval on physical displays and cross-platform Windows/macOS QA remain mandatory before public promotion.
