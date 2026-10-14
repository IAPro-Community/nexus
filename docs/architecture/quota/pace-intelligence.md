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
- The local environment available for this change is Go 1.23 and lacks
  external repo DNS, so **these runtime gates have NOT been executed here**.

## Further roadmap (not implemented by this slice)

- **P1 external Claude transcripts**: opt-in, read-only incremental ingestion;
  require account attribution, stable event dedup and a separate external
  evidence channel until reconciliation proves an execution was not also
  included in the Nexus native cost ledger. Never silently add subscription
  API-equivalent cost to actual billed spend. A new collector MUST pass
  privacy and performance gates before any background collection.
- **P2 service status**: public incident feeds with independent TTL/cache and
  no user credential forwarding; never turn provider outage into fabricated
  account quota; keep service status separate from quota and auth state.
- **P2 cost history/export**: the existing Cost Intelligence ledger remains
  the source of truth; this feature exports only pace measurements.
