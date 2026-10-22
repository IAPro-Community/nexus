# Capacity Pace Intelligence (TokenBar-inspired, Nexus-native)

Implemented directly on `release/v1.0.0`, October 2026. No code was copied from TokenBar (GPL-3.0-or-later).

## Data path

`QuotaMonitorService` (canonical live probe) → existing `capacitymonitor.Monitor`
→ read-only `UsageSnapshot` → `capacity_pace_samples` SQLite (0029) →
`PredictPace` → `CapacityApplicationService.Status` →
`UsageAccountCard` and existing `NexusHud`.

**No new polling subprocess, no browser-side quota estimator, no modifications to
Codex PTY/ConPTY, no changes to provider auth.** The monitor's existing opt-in
`enabled && background_enabled` remains authoritative.

## Storage and privacy

- Retention: 72 hours, pruned only by the active monitor leader.
- Samples: at most one observation per 5-minute UTC bucket per provider,
  profile, verified account scope, quota group, window and *absolute reset*.
- `AccountScope.Verifiable()` and fresh source-backed `LIVE` are required
  to write evidence; `UNKNOWN`, `ESTIMATED`, `CACHED`, unauthorized or
  unbound observations cannot be used as new trend samples.
- Percentages must be finite and between 0–100; malformed metrics are skipped.
- The sample table stores no token, API key, email, prompt, log path or transcript.
- CSV export is authenticated at `GET /api/v1/capacity/pace.csv`, omits
  account scope identifiers and sets `Cache-Control: private, no-store`.
- Missing/failed optional pace persistence never changes the canonical
  `quota_remaining`, `UsageSnapshot` or resource scheduler score.

## Forecast semantics

A forecast is advisory, not a provider-reported limit:

1. Match the exact provider/profile/verified-account/group/window/reset cycle
   against the currently visible **LIVE** observation.
2. Require 3 or more samples, 10+ minutes of observation, and a reading no
   more than 10 minutes old.
3. Reject missing resets, expired cycles, decreasing usage suggesting a refill,
   mixed identities, mismatched current usage and non-finite observations.
4. Extrapolate recent net used-percent slope; return `RISING` only when
   extrapolated exhaustion precedes the actual reset. Otherwise
   `RESET_FIRST` or `STABLE`, with explicit `LOW/MEDIUM/HIGH` evidence
   confidence.
5. Present only `RISING` in Usage and HUD, labelled *estimate*, subject to
   HUD privacy controls. Never alter the real observed quota or routing gates.

The confidence label expresses sample depth/duration, not statistical
calibration. The forecast is not suitable for automated provider failover
without future empirical evaluation.

## Gates before certifying

- Go 1.25: `go test ./internal/nexus/store ./internal/nexus/capacitymonitor ./internal/control/web ./internal/nexus`
- Frontend: `cd web && npm run quality:full && npm run test:e2e:capacity`
- Verify 5h/weekly reset, stale caches, multiple scoped Codex accounts,
  account re-login, zero/unknown quota, CSV auth, Firefox/Chromium and Windows
  WebView2; check quota remains visible when pace storage is unavailable.
- Check Go formatting, TS/SCSS lint and exact-SHA release matrix.
- A nota histórica do ambiente inicial registrou Go 1.23 e ausência de DNS externo; **nenhuma execução posterior fica comprovada por esta página**. Para certificar o SHA atual, exigir logs dos comandos e dos smokes nativos correspondentes.

## Related follow-up now integrated (not part of the original P0 slice)

- **P1 external Claude transcripts:** manual, explicitly consented scanner with
  incremental offsets and message dedup is now available via Cost Intelligence.
  The account scope is unverified, so token observations live in separate SQLite
  tables and are deliberately **not** reconciled into actual cost or budget.
  See [external usage and status](EXTERNAL_USAGE_AND_PUBLIC_STATUS.md).
- **P2 service status:** read-only allowlisted public incident feeds are now
  visible in Capacity Monitor Settings, with independent cache/timeout.
- **P2 cost export:** canonical cost observations are now exportable in
  authenticated CSV, without account scope IDs or fabricated zero prices.

## Future upgrades requiring additional verification

The remaining enhancement is **verified** cross-account/provider session
attribution before ever merging local transcript tokens into the canonical
ledger; retain explicit opt-in and permission/privacy gates. Public service
incidents must never be interpreted as subscription quota.

