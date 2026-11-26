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

---

# Gate G2 Report — Phase P2 (NEX-02)

- **P1 Commit SHA:** `f80151e758c4a8237898a78d4c88b9acb57037d9`
- **Platform Tested:** Linux (`x86_64`, Ubuntu 25.10, Go `-race` unit tests + Chromium Playwright fixture server against `bin/nexus`)
- **Verdict:** **PASS (Linux Presence & 100-Cycle Stability)** / **BLOCKED (`Windows`, `macOS` native hardware)** / **PENDING (`72h soak` before R1 promotion)**

## Gate G2 Checklist

| Criterion | Status | Evidence |
|---|---|---|
| Go lifecycle supervisor, tray menu & state isolation | **PASS** | `internal/desktop/nex_v2_lifecycle.go`, `nex_v2_controller.go`, `cmd/nexus-hud/nex_v2.go`; persists `nex-v2-state.json` without mutating `hud-window-state.json`; `go test -tags=gtk3 -race -count=1 ./internal/desktop/... ./cmd/nexus-hud/...` PASS |
| Bounded native surface transitions (`panelOpen` / `compactBar` / `preset`) | **PASS** | `ResolveNexV2SurfaceRect` + `TestNexV2ControllerSurfaceRectTransitions` + `TestNexV2WindowOptionsFixedBoundsAndTransparentChrome` PASS (`440x540` panel, `360x56` compact bar, fixed square preset collapsed) |
| Lazy panel code-splitting & latency budgets | **PASS** | `web/scripts/build.mjs` emits `dist/nex-v2-chunks/NexInteractionPanel-MMPEGD4W.js` (`6.5kb`); `e2e-nex-v2-presence.mjs` verified `initialPanelChunks: 0`, `postOpenPanelChunks: 1`, `panelOpenColdMs: 13.5` (`<= 1500ms`), `panelOpenWarmMs: 0.9` (`<= 500ms`) |
| 3-tab panel, separate settings, avatar-hidden compact bar & DND | **PASS** | `NexInteractionPanel` (`Chat` \| `Resources` \| `Attention` tabs + `ArrowLeft`/`ArrowRight`/`Escape` focus restore), `NexSettingsDrawer`, `NexCompactBar`, `NexNotificationBubble`; verified in `NexPresenceApp.test.tsx` (6 tests) & Playwright |
| Honest `UNKNOWN`/`STALE` quota & Core-offline handling | **PASS** | Missing quotas show `UNKNOWN` (never `0%`); HTTP `503` renders `[data-nex-offline-banner]` and switches avatar mood to `offline` |
| 100-cycle stability without errors or geometry drift | **PASS** | `e2e-nex-v2-presence.mjs` executed 100 cycles (`204` panel toggles, tabs, settings, hide/show avatar, compact bar, DND): `pageErrors: 0`, `consoleErrors: 0`, `requestFailures: 0`, avatar stayed `160x160` |
| 72h continuous desktop soak | **PENDING_BEFORE_R1_PROMOTION** | Not executed in single-session P2 gate; required before R1 promotion |
| Native Windows & macOS validation | **BLOCKED** | No physical Windows or macOS hardware on this Linux host |

---

# Gate G3 Report — Phase P3 (NEX-03) & Slice R1 (Foundation + Guardian)

- **P2 Commit SHA:** `ec012135`
- **Platform Tested:** Linux (`x86_64`, Ubuntu 25.10, Go `-race` test suites + Chromium Playwright against `bin/nexus`)
- **Verdict:** **PASS (Linux Foundation + Guardian Parity, Zero LLM Required)** / **BLOCKED (`Windows`, `macOS` native hardware)** / **PENDING (`72h soak` before R1 promotion; Legacy retirement remains blocked until P8)**

## Gate G3 & Slice R1 Checklist

| Criterion | Status | Evidence |
|---|---|---|
| Canonical read-only Model Catalog Watch (no external CLI probes) | **PASS** | `internal/nexus/model_catalog_watch.go` + `_test.go` (`ADDED`, `REMOVED`, `DEPRECATED`, `DEFAULT_CHANGED`, `UNKNOWN_SOURCE`, idempotent re-observation, persistence reload, `UNKNOWN`/`STALE`/`FRESH`, `-race` PASS) |
| Cursor-indexed event stream, fingerprint dedupe & notification policy | **PASS** | `internal/nexus/nex_event_stream.go` + `_test.go` (`Since(cursor)` replay & `ReplayTruncated`, DND/quiet-hours suppression, burst cap, `-race` PASS) |
| Authenticated HTTP endpoints (`/api/v1/nex/events`, `/api/v1/nex/models/watch`) & Privacy Mode persistence | **PASS** | `internal/control/web/handlers_nex_stream.go`, `handlers_nex_stream_test.go`, `routes.go`; `internal/desktop/nex_v2_controller.go` `SetPrivacyMode`; `e2e-nex-v2-guardian.mjs` confirms `401` unauth / `200` auth |
| Multi-window quotas, resets, honest `UNKNOWN`/`STALE`, Cost evidence & unified Attention parity without LLM | **PASS** | `NexResourcesTab`, `NexAttentionTab`, `NexSettingsDrawer`, `NexNotificationBubble`, `NexPresenceApp`; `NexGuardianParity.test.tsx` (21/21 Vitest files, 113 tests + 8 node-tests PASS) |
| Guardian Playwright E2E + no regression in P1 Lab or P2 100-cycle suite | **PASS** | `e2e-nex-v2-lab.mjs` PASS (`670ms` load, `328.9ms` paint); `e2e-nex-v2-presence.mjs` 100-cycle PASS (`5.9ms` cold, `1.1ms` warm, `0` errors); `e2e-nex-v2-guardian.mjs` PASS (`privacyMode=true`, `eventRevision=1`, `FRESH`, `0` errors) |
| 72h continuous desktop soak & native Windows/macOS QA | **PENDING / BLOCKED** | Windows/macOS hardware unavailable (`BLOCKED`); 72h soak required prior to Slice R1 opt-in promotion; legacy HUD remains untouched until P8 |

---

# Gate G4 Report — Phase P4 (NEX-04)

- **P4 Commit SHA:** `4e0a3afe`
- **Platform Tested:** Linux (`x86_64`, Ubuntu 25.10, Go `-race` + Chromium Playwright against rebuilt `./nexus`)
- **Verdict:** **PASS (Linux Assistant, `@personal` Memory & Spend Guard)** / **BLOCKED (`Windows`, `macOS` native hardware)**

## Gate G4 Checklist

| Criterion | Status | Evidence |
|---|---|---|
| Bounded assistant context & fail-closed spend guard | **PASS** | `internal/nexus/nex_assistant_context.go`, `nex_spend_guard.go` + `_test.go` (`TestBuildNexContextSummary`, `TestEvaluateNexSpendGuard` `-race` PASS) |
| Isolated `@personal` memory store & authenticated endpoints | **PASS** | `internal/nexus/nex_personal_memory.go`, `internal/control/web/handlers_nex_memory.go`, `handlers_nex_assistant.go`, `routes.go` (`/api/v1/nex/memory` `401` unauth, CRUD + reload persistence verified in Go `-race` & `e2e-nex-v2-assistant.mjs`) |
| Durable chat & personal memory UI in `NexChatTab` | **PASS** | `web/src/nex-v2/NexChatTab.tsx`, `NexChatTab.module.scss`, `nexV2Client.ts`, `i18n.ts`; `e2e-nex-v2-assistant.mjs` verifies reload persistence (`durableChatVerified: true`, `personalMemoryCrudVerified: true`) |

---

# Gate G5 Report — Phase P5 (NEX-05) & Slice R2 (Assistant + Audited Actions)

- **P5 Commit SHA:** `pending-commit` (over P4 `4e0a3afe`)
- **Platform Tested:** Linux (`x86_64`, Ubuntu 25.10, Go `-race` + Vitest/node-test + Chromium Playwright against rebuilt `./nexus`)
- **Verdict:** **PASS (Linux Slice R2 Assistant + Audited Actions)** / **BLOCKED (`Windows`, `macOS` native hardware)** / **DEFERRED (`Legacy HUD retirement` until P8)**

## Gate G5 & Slice R2 Checklist

| Criterion | Status | Evidence |
|---|---|---|
| Allowlisted typed action registry, single-use confirmation & redacted receipts | **PASS** | `internal/nexus/nex_actions.go` + `_test.go` (6 allowlisted actions, `shell`/`exec`/`push`/`deploy`/`delete_repo`/`paid_fallback` forbidden, single-use `Confirm`, `security.Redact`, disk reload persistence; `-race` PASS) |
| Authenticated HTTP endpoints & canonical owner routing | **PASS** | `internal/control/web/handlers_nex_actions.go` + `_test.go`, `handlers_nexus.go`, `routes.go` (`/api/v1/nex/actions`, `/propose`, `/confirm`; `401` unauth, `403` missing CSRF / forbidden action, `UNAVAILABLE` honest delegation; `-race` PASS) |
| Audited receipts UI (`NexReceiptsPanel`) & proposal wiring (`NexChatTab`) | **PASS** | `web/src/nex-v2/NexReceiptsPanel.tsx`, `NexReceiptsPanel.module.scss`, `NexChatTab.tsx`, `NexChatTab.module.scss`, `NexAssistantActions.test.tsx`; Privacy Mode masks summaries (`•••`) |
| Web formatting, lint, stylelint, style allowlist, typecheck, unit tests & bundle build | **PASS** | `format:check`, `lint`, `lint:styles`, `check:styles`, `typecheck`, `test` (`159/159` Vitest files / `1032` tests + node-tests), `build` (`dist/nex-v2.js` `220.5kb`, lazy `NexInteractionPanel-*.js` `17.5kb`, `nex-v2.css` `7.8kb`) |
| Go `-race` test suites | **PASS (Scoped)** | `go test -tags=gtk3 -race -count=1` PASS on `./internal/control/web/...`, `./internal/desktop/...`, `./cmd/nexus-hud/...`, all `./internal/nexus/...` subpackages, and targeted `TestNex|TestModelCatalog|TestHUD|TestEvaluateNexSpendGuard` in `./internal/nexus`. Full `./internal/nexus` has 7 pre-existing non-P5 failures (`5` `seedCodexExecutionFixture` tests hitting `error obtaining VCS status: exit status 128` in worktree without `-buildvcs=false`, and `2` `memory_migration_test.go` tests seeing `InstallCount` `4`/`5` vs `1` on this host) |
| Playwright E2E (`p4-p5-assistant`, `p3-guardian`, `p2-presence` 100-cycle) | **PASS** | `e2e-nex-v2-assistant.mjs` PASS (`receiptsVerified: 4`, `privacyMaskingVerified: true`, `0` errors); `e2e-nex-v2-guardian.mjs` PASS (`authEventsStatus: 200`, `authWatchStatus: 200`, `privacyMode: true`, `FRESH`, `0` errors); `e2e-nex-v2-presence.mjs` PASS (`initialPanelChunks: 0`, `postOpenPanelChunks: 1`, `panelOpenColdMs: 4`, `panelOpenWarmMs: 1`, `100` cycles / `204` toggles, `0` errors) |
| Native Windows/macOS QA & Legacy HUD retirement | **BLOCKED / DEFERRED** | Windows/macOS native hardware unavailable (`BLOCKED`); legacy HUD remains untouched until P8 |

