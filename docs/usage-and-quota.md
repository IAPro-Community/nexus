# Usage, Capacity & Honest Quota Engine

**🇬🇧 English** | [🇧🇷 Português](usage-and-quota.pt-BR.md) | [🇪🇸 Español](usage-and-quota.es.md)

A core principle of `ai-cli` is **never to fabricate quota or present unknown values as 100%**.

---

## 1. Usage States

The quota engine operates on explicit states:

| Status | Description | UI Progress Bar |
|---|---|---|
| `LIVE` | Fresh point-in-time quota retrieved from the provider API or CLI `/usage` | `[████████████░░░░░░] 68%` |
| `CACHED` | Valid cached quota within configured TTL (e.g. updated 2m ago) | `[████████████░░░░░░] 68%` |
| `ESTIMATED` | Statistical or heuristic capacity inference | `[████████░░░░░░░░░░] ~50%` |
| `UNKNOWN` | No authentic quota metric is available for this profile | `[????????????????????] UNKNOWN` |
| `RATE_LIMITED` | Profile is temporarily blocked by an active 429 rate limit | `[!!!!!!!!!!!!!!!!!!!!] RATE LIMITED` |
| `UNSUPPORTED` | Provider does not expose quota metrics | `[--------------------] UNSUPPORTED` |
| `ERROR` | External error encountered when querying usage | `[????????????????????] ERROR` |

---

## 2. Cascade Resolution Strategy

When evaluating account capacity, `ai-cli` queries in order of priority:

1. **Level 1 — Official Provider API / Metrics**: Authentic quota endpoints or headers.
2. **Level 2 — CLI Output (`/usage`, `status`)**: Structured CLI output parsers isolated within adapters.
3. **Level 3 — Local CLI State Files**: Real rate-limit or billing state persisted by the CLI.
4. **Level 4 — Observable Response Headers**: `x-ratelimit-remaining`, `retry-after`, `reset`.
5. **Level 5 — Historical Observation**: Recent success timestamps, cooldown status.
6. **Level 6 — UNKNOWN**: If no authentic source exists, the state is strictly reported as `UNKNOWN`.

---

## 3. Freshness & TTL

Usage metrics are cached per profile in `usage.json` with timestamps:
- **Default TTL**: 5 minutes (configurable).
- **Staleness Indication**: Displayed in CLI and TUI (e.g. `LIVE · updated 8s ago` or `CACHED · updated 4m ago`).
- **Bounded Concurrency**: Quota refreshes across multiple profiles are executed in parallel using a bounded worker pool (default: 4 concurrent probes).

Automatic account switching is disabled by default. Explicitly authorize pre-dispatch switching from a busy profile with `nexus config fallback enable`; revoke it with `nexus config fallback disable`. Once a provider process has started, Nexus does not replay a quota/rate-limited operation with ambiguous output or side effects. Recent eligible quota evidence is required in either path.
