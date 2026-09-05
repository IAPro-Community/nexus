# Nexus Capacity Monitor & Notifications — Linux Beta Validation Report

**Date:** 2026-09-05  
**Release Target:** Nexus v0.4.0 (Capacity Monitor Linux Beta)  
**Branch:** `feat/capacity-monitor-notifications`  
**Status:** VALIDATED — ALL GATES GREEN

---

## 1. Executive Summary

The **Nexus Capacity Monitor & Proactive Notification System** has been fully implemented and verified against the engineering authorities:
1. `AGENTS.md` (Design tokens, SCSS modules, keyboard a11y, strict TypeScript, zero inline static styles)
2. `docs/engineering/ENGINEERING_STANDARDS.md`
3. `docs/engineering/WEB_ENGINEERING_STANDARDS.md`
4. `docs/superpowers/specs/2026-09-05-capacity-monitor-notifications-design.md`
5. `docs/superpowers/plans/2026-09-05-capacity-monitor-notifications-implementation.md`

All 10 implementation tasks have completed through test-driven development (RED -> GREEN), isolated to a dedicated worktree with zero modifications to the main tree.

---

## 2. Architecture & Neutral Quota Contracts

- **Probe Independence:** `internal/core/quota` provides probe interfaces without hardcoding provider internals.
- **Model Attribution Honesty:** Model attribution only emits a concrete model name when `UsageGranularityModel` and non-empty `ModelName` are provided by the provider adapter; otherwise it safely attributes to `QuotaGroup` / `ProviderID`.
- **Single-Leader Concurrency:** Coordinated via SQLite leases (`capacity_leases`) with a 30s TTL. Concurrency is strictly bounded:
  - Global maximum concurrent probes: 2
  - Per-profile mutex (singleflight deduplication): 1

---

## 3. Provider Adapter Coverage

| Provider | Probe Mechanism | Cache / Granularity | Fallback / Safety |
|---|---|---|---|
| **Codex** | Local OAuth JSON & usage snapshots | Group level, 5m TTL | Non-blocking, read-only |
| **AGY** | CLI telemetry metadata | Group level, 5m TTL | Non-blocking, read-only |
| **Claude** | Profile config & usage file | Group level, 5m TTL | Non-blocking, read-only |
| **Gemini** | Quota snapshot file | Group level, 5m TTL | Non-blocking, read-only |
| **Cursor** | Session quota record | Group level, 5m TTL | Non-blocking, read-only |
| **OpenCode** | Profile balance record | Group level, 5m TTL | Non-blocking, read-only |

---

## 4. State Machine & Transitions

State machine transitions strictly respect threshold boundaries (default: 20% warning, 10% critical, 0% exhausted, 25% recovery):
- `HEALTHY` -> `WARNING` (<= 20% remaining)
- `WARNING` -> `CRITICAL` (<= 10% remaining)
- `CRITICAL` -> `EXHAUSTED` (0% remaining or 429 rate limited)
- `EXHAUSTED` -> `RECOVERED` (>= 25% remaining)
- `RECOVERED` -> `HEALTHY` (>= 25% remaining)
- `*` -> `UNKNOWN` (observation expired / stale)

---

## 5. Multi-Channel Delivery & Privacy Guarantees

1. **Delivery Router:** Multi-channel delivery to in-app notification center, browser notifications, and native OS notifications.
2. **Quiet Hours:** Configurable UTC window (default 22:00-08:00) with optional bypass for critical 100% exhaustion alerts.
3. **Lock Screen Privacy:** Hides project titles, exact models, and sensitive parameters from OS notification previews when `lock_screen_safe_copy` is enabled.
4. **Receipt Tracking:** Delivery receipts persisted to `capacity_delivery_receipts` table with per-channel delivery timestamps and error tracking.

---

## 6. Frontend Presentation & Settings

1. **`CapacityAlertCard`:**
   - Encapsulated SCSS Module (`CapacityAlertCard.module.scss`).
   - Semantic HTML with WCAG AA compliance.
   - Severity badges, dynamic progress bar, estimated reset timer, and 1-click provider switch action.
2. **`CapacityMonitorSettings`:**
   - Dedicated configuration card integrated into Settings under Notifications.
   - Threshold editors, quiet hours configuration, delivery channel switches, and background consent toggle.
3. **i18n:**
   - Complete translations in English (`en`), Portuguese (`ptBR`), and Spanish (`es`).

---

## 7. Linux Background Autostart

1. **Consent-Gated:** Only configured when the user explicitly enables `background_enabled`.
2. **User-Scoped Systemd Unit:** Written to `$XDG_CONFIG_HOME/systemd/user/nexus-capacity-monitor.service`.
3. **CLI Daemon:** Executed via `nexus capacity-monitor run` with graceful signal shutdown on `SIGINT`/`SIGTERM`.
4. **Clean Removal:** Safe uninstallation handled by `uninstall.sh`.

---

## 8. Verification & Load Evidence

### Load Simulation
- **Configuration:** 50 profiles evaluated continuously across 60 ticks (30 simulated minutes).
- **Concurrency Result:** Peak observed concurrency: **2** (strictly within `MaxConcurrency = 2`).
- **Memory & Goroutines:** Zero goroutine leaks (stable goroutine count after full execution).
- **Race Detector:** `go test -race ./...` passed with **0 race conditions detected**.

### Frontend Quality Gates
- **Allowlist Styles Check:** PASSED (`check-css-allowlist.js`)
- **Prettier Format:** PASSED (`format:check`)
- **ESLint:** PASSED (`lint`)
- **Stylelint:** PASSED (`lint:styles`)
- **TypeScript Typecheck:** PASSED (`typecheck`)
- **Vitest Suite:** PASSED (53 test files, 262 tests)
- **Production Build:** PASSED (esbuild / tailwind v4 bundles generated cleanly)

---

## 9. Residual Risks & Next Milestones

1. **Windows / Darwin Background Daemon:** Currently Windows and Darwin report degraded/unsupported status for background systemd autostart. Native service wrappers (e.g. launchd / Windows service) can be added in post-v0.4.0.
2. **Live Provider Streaming:** In-app updates currently poll on tick and receive push alerts via server-sent events. Future enhancements could stream live token counts directly from active PTY sessions.
