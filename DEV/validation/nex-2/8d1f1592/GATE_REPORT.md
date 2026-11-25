# Gate G0 Report — Phase P0 (NEX-00)

- **Baseline SHA:** `8d1f1592ce8c4b12afa66585232ff2d09bd41d93`
- **Platform Tested:** Linux (`x86_64`, Ubuntu 25.10, kernel `6.17.0-23-generic`, GNOME Wayland + XWayland `:0`, WebKitGTK `2.52.3`, GTK3 `3.24.50`, VMware SVGA II Adapter, 8 cores / 60 GiB RAM)
- **Verdict:** **PASS (Linux Foundation)** / **BLOCKED (`Windows`, `macOS` — no native test hardware on this host)** / **BLOCKED (`e2e-hud`, `e2e-companion-lab` baseline regression recorded)**

## Gate G0 Checklist

| Criterion | Status | Evidence |
|---|---|---|
| Reconcile source & branches without mutating dirty `/projetos/nexus` | **PASS** | `BASELINE.md` §2; `/projetos/nexus` `git status --short` preserved untouched; isolated worktree `/projetos/worktrees/nex-2-foundation` created |
| Identify installed binary/version/OS/compositor/WebKit/DPI | **PASS** | `BASELINE.md` §1 (`nexus` 1.0.1 `bff681fd`, `nexus-hud` 1.0.1 `bff681fd`, Wails `v3.0.0-beta.25`, WebKitGTK `2.52.3`, GTK3 `3.24.50`, GNOME Wayland) |
| Reproduce Linux HUD memory/clipping/motion/race defects | **PASS** | Live `nexus-hud` process tree measured at `~728 MiB` RSS / `~447 MiB` PSS (`WebKitWebProcess` `~354 MiB` PSS); code-level races and `SetHUDBubbleVisible` resize mutations documented in `BASELINE.md` §4 |
| Produce Capability Ledger | **PASS** | `CAPABILITY_LEDGER.md` covers all 17 domains with decisions, owners, callers, files, and tests |
| Feature flag default OFF + mutual exclusion + rollback contract | **PASS** | `FeatureNexV2` in `internal/nexus/feature_flags.go`, `featureNexV2` in `internal/control/web/feature_flags.go`, `internal/desktop/nex_v2_mode.go` + unit tests |
| Baseline unit tests & visual lock verification | **PASS** | Go unit/race tests PASS; `npm --prefix web run typecheck` PASS; Web HUD Vitest 16/16 files (97 tests) + node-test 8/8 PASS; `verify-nex-identity-lock.mjs` PASS (`FROZEN`) |
| Baseline Playwright HUD E2E (`e2e-hud`, `e2e-companion-lab`) | **FAIL (Baseline)** | Both fail on `8d1f1592` because `NexusHud.tsx:171` gates `getAttention()` behind `COMPOSER_FLOW_ENABLED` (default OFF); captures archived in `e2e-hud/` & `companion-lab/` |
| Native Windows & macOS validation | **BLOCKED / UNVERIFIED** | No physical Windows or macOS machine available on this Linux host |

## Remediation & Next Step
1. Hand back for Gate G0 plan review (`enter_plan_mode`).
2. Proceed to **P1 (NEX-01)**: build the lightweight Wails v3 HUD entrypoint (`DEV` / `NEX_V2` flagged) isolating avatar bounds and animation clock from Workspace OS and fixing the 60 FPS / bubble-resize / startup-race defects.
3. Address the `NexusHud.tsx:171` attention-fetch decoupling during P2/P3 parity work (or before re-enabling `e2e-hud` in R1).

---

# Gate G1 Report — Phase P1 (NEX-01)

- **P0 Commit SHA:** `fb22bbb777f5cb26eb83954373706664823ae569`
- **Platform Tested:** Linux (`x86_64`, Ubuntu 25.10, Chromium Playwright lab + isolated `Xvfb :97` / `dbus-run-session` native GTK3/WebKitGTK `2.52.3` run)
- **Verdict:** **PASS (Linux Lab & ADR)** / **BLOCKED (`Windows`, `macOS` native hardware)** / **BLOCKED (`owner_visual_approval` in `RELEASE_READINESS.json`)**

## Gate G1 Checklist

| Criterion | Status | Evidence |
|---|---|---|
| Single selected runtime & reproducible ADR | **PASS** | `RENDERER_ADR.md` selects Wails v3 isolated `/nex-v2.html` entrypoint (`7.8 KB` minified CSS, `264.4 KB` JS, zero Workspace chunks/SW); Qt Quick/QML evaluated and deferred |
| Approved locked `NEX-ID-v1` artwork & fallback | **PASS** | `NexRigCanvas` renders `/nex/hybrid-v1/` layers with `/nex/hud-32bit/nex-idle-atlas-128.png` fallback; `verify-nex-identity-lock.mjs` PASS (`FROZEN`); `export-nex-art.test.mjs` PASS |
| Zero clipping over all motions / presets / DPIs | **PASS** | `nexEnvelope.test.ts` sweeps 9 expressions × 24 time samples × orbit blends × presets (`128`/`160`/`192`) × DPI (`1`–`2`); `e2e-nex-v2-lab.mjs` verifies `edgeClip=0` across all 5 moods |
| Zero movement-driven window resize | **PASS** | Fixed square viewport presets in `nex_v2_geometry.go` + `MinWidth==MaxWidth` in `nex_v2.go`; `e2e-nex-v2-lab.mjs` measured `resizeCountDelta=0` |
| Motion contract (5 moods, 8 behaviors, single clock, reduced-motion pause) | **PASS** | `nexBehaviorState.test.ts`, `nexMotionClock.test.ts`, and `p1-lab/p1-lab-metrics.json` (`155ms` paint, `20 FPS` idle, `16.8ms` p95 interactive, `0` stutters `>100ms`, `0` frames under reduced-motion) |
| Baseline benchmark budgets / documented exception | **PASS (with exception)** | `WebKitWebProcess` PSS dropped `353.6 MiB -> 202.9 MiB` (`-43%`); Linux WebKitGTK platform baseline exception (`> 60 MiB`) explicitly documented in `RENDERER_ADR.md` |
| Native Windows/macOS & final owner visual approval | **BLOCKED** | No physical Windows/macOS devices on this Linux host; `RELEASE_READINESS.json` stays `BLOCKED (2/11)` |

