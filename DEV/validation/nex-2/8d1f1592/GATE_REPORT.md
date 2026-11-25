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
