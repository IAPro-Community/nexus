# Capacity Monitor Revival — Final Report (T9, 2026-09-20)

Branch: `feat/capacity-monitor-revival` · Base: `origin/main` (`b8d9962`).
HEAD at writing: `4cb4e26` + uncommitted round (minimize-hide, capacity E2E,
beta.2 truth docs — committed after gates).

## 1. Scope
T4–T9 of `DEV/SPECS/CAPACITY_MONITOR_REVIVAL.md` (all boxes checked with
evidence below). Terminal hardening preserved and synced from the maximus
line (`7813105`).

## 2. Architecture reused (no second subsystem)
- Ranking: existing `RecommendResources` via `CapacityRecommender` adapter
  (`internal/nexus/capacity_recommendation_bridge.go`); cadence-only
  `DetermineCollectionInterval`, canonical `ResourceScheduler` untouched.
- Delivery over existing `internal/control/events` bus; dedup is an
  owner-tagged `Coordinator` gate, not a bus.
- No second store/router/runner/memory; `Monitor.Tick/Run` is a jittered poll
  loop hosted only via explicit `StartCapacityMonitor`.

## 3. Tasks completed (evidence)
- **T4**: `capacitymonitor/recommendation.go`, bridge + tests; advisory-only,
  nil-safe, never auto-migrates.
- **T5**: `ResetCycleKey`, `ShouldDeliver`, event-before-publish + `Forget`,
  receipts, ack/snooze enforcement; `capacity_alert_states` table.
- **T6**: `CapacityApplicationService` + 5 auth-wrapped routes
  (status/history/preferences/ack/snooze); UNKNOWN never healthy; handler
  tests incl. CSRF harness.
- **T7**: `CapacityMonitorSettings` tab, `capacityAlertFromEvent`
  (fingerprint ids, recovery, recommendation), usage monitor strip + ack,
  flow execution summary; i18n en/pt-BR/es.
- **T8**: `Start/StopCapacityMonitor`, consent-gated
  (`Enabled+BackgroundEnabled`, default OFF), idempotent, lease-gated,
  `Shutdown` hook; `emptyProbeRegistry` until real probes.
- **T9**: this report + gates below.

## 4. Tests (executed 2026-09-20, HEAD `4cb4e26` + round)
- `go test ./internal/...`: zero FAIL.
- `-race`: `internal/control/...` (15 pkgs), `capacitymonitor`, `store`,
  `runner`: all ok.
- `go vet ./...`: clean. `git diff --check`: clean.
- `make build`, `make test-e2e` (terminal/protocol/host/web): PASS.
- `make web-verify`: 10/10 PASS (format, typecheck, lint, stylelint,
  null-arrays, test, i18n, build, embed-sync, ui-markers).
- Vitest full: 76 files / 418 tests PASS.
- `make security` (govulncheck): no vulnerabilities.
- New `web/scripts/e2e-capacity.mjs` (`test:e2e:capacity`): PASS against a
  real isolated server (auth denial, CSRF negative, empty-not-healthy,
  preferences validation + round-trip, ack + snooze lifecycle).
- Browser E2E (`e2e-surface-isolation.mjs`): NOT_EXECUTED here — Playwright
  chromium lacks system libs (`libatk`) in this container (ENVIRONMENTAL);
  must run in CI/with-deps environment.

## 5. Restart / dedup / snooze / recovery
Event-before-publish + `Forget` on write failure; reminder-gated repeats;
snooze/ack enforced in delivery path; lease leadership across restarts;
recovery via `RECOVERED` transition + hysteresis (spec thresholds).

## 6. Known limitations
- Real provider probes: `emptyProbeRegistry` (UNKNOWN visible, never alerts).
- `runs.json` files predating terminal metadata-only persistence still hold
  old raw args/output (no backfill scrub).
- Terminal output buffers capped in memory (50 × 100k), freed only by cap.
- Browser E2E + native providers + overnight: UNVERIFIED in this environment.

## 7. Verdict
**GO locally** (all executable gates green, no new architecture). Global
**MERGE_CANDIDATE = CONDITIONAL**: push + same-SHA CI + provider-native
evidence require network/credentials unavailable here.
