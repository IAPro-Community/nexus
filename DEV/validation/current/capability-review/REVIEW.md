---
phase: capability-review-hotpaths
reviewed: 2026-09-12T16:42:41Z
branch: feat/nexus-maximum-delivery
depth: deep
files_reviewed: 18
files_reviewed_list:
  - internal/core/provider/adapters/codex/cross_account_sessions.go
  - internal/core/provider/adapters/codex/codex.go
  - internal/core/provider/adapters/codex/tui_lock.go
  - internal/core/provider/adapters/codex/tui_lock_unix.go
  - internal/core/provider/adapters/codex/tui_lock_windows.go
  - internal/core/provider/adapters/codex/app_server_usage.go
  - internal/control/flags/normalizer.go
  - internal/control/flags/help.go
  - internal/app/app.go
  - internal/app/tunnel_cmd.go
  - internal/control/web/auth.go
  - internal/control/web/auth_store.go
  - internal/control/web/tunnel.go
  - internal/control/web/server.go
  - internal/control/web/handlers_tunnel.go
  - internal/core/provider/adapters/agy/agy.go
  - internal/core/provider/adapters/claude/claude.go
  - docs/design/ai-cli-control-plane.md
findings:
  critical: 2
  warning: 5
  info: 3
  total: 10
status: issues_found
---

# Capability Hot-Path Code Review

**Reviewed:** 2026-09-12T16:42:41Z  
**Depth:** deep (scoped hot paths only)  
**Branch:** `feat/nexus-maximum-delivery`  
**Status:** issues_found

## Summary

Scoped adversarial review of Codex CrossAccountResume/quota attribution, TUI lock vs app-server, flag aliases, dispatcher/help parity, web auth/bootstrap/tunnel, and AGY/Claude CrossAccountResume claims.

Highest risk: `nexus tunnel` never calls `SetTunnelActive(true)`, so loopback-style bootstrap reuse and WebSocket query-token auth remain enabled on a public Cloudflare URL. Secondary risks: hardlinked cross-account rollouts + ignored marker write failures can corrupt quota attribution or share mutable session inodes across profiles.

## Narrative Findings (AI reviewer)

### CRITICAL

#### CR-01: `nexus tunnel` never arms tunnel auth mode

**Severity:** CRITICAL  
**File:** `internal/app/tunnel_cmd.go:84-90` (also `internal/control/web/server.go:180-184`)  
**Issue:** CLI tunnel starts Core with `TunnelHost` (origin allowlist only) but never calls `auth.SetTunnelActive(true)`. API path in `handlers_tunnel.go:147` does. Consequences while traffic is public:

1. Bootstrap remains reusable (`auth.go:298-299`: `reusable := loopback && !a.tunnelActive`).
2. WebSocket may accept `?token=` / `?session=` (`auth.go:184-188`) — session IDs leak via public proxies/logs.
3. `cookieSecure()` stays false on loopback (`server.go:64-66`) despite HTTPS tunnel.

**Fix:**

```go
// In NewServer when opts.TunnelHost != "":
if opts.TunnelHost != "" {
    s.tunnelHost = opts.TunnelHost
    originpolicy.RegisterTunnelHost(opts.TunnelHost)
    s.auth.SetTunnelActive(true) // required for CLI + API parity
}
```

Or call `SetTunnelActive(true)` immediately after Core is ready in `tunnel_cmd.go`. Add a regression test that `TunnelHost != ""` ⇒ `IsTunnelActive()==true`.

---

#### CR-02: Cross-account hardlink + ignored marker → foreign quota / shared inode

**Severity:** CRITICAL  
**File:** `internal/core/provider/adapters/codex/cross_account_sessions.go:90-100,251-260,455-481` + `codex.go:527-531`  
**Issue:**

1. `hardlinkOrCopy` prefers `os.Link` — destination and source share one inode; either profile can mutate/truncate the other’s rollout.
2. `recordCrossAccountSession` errors are discarded (`_ = recordCrossAccountSession(...)`).
3. If link/copy succeeds but marker write fails (or pre-existing untracked import), `rolloutBelongsToProfile` will **not** exclude the file → foreign account usage can feed this profile’s quota.
4. `adoptSessionIntoProfile` early-return when rollout already exists (`117-123`) does not ensure a marker exists.

**Fix:** Prefer copy (not hardlink) for cross-account imports; fail closed if marker cannot be written (rollback dst); on “already present”, call `recordCrossAccountSession` before returning; treat marker write failure as hard error in `seedCrossAccountSessions` / `adoptSessionIntoProfile`.

---

### HIGH

#### HI-01: Help advertises universal `-c` / `-p`; Codex intentionally excludes them

**Severity:** HIGH  
**File:** `internal/app/app.go:507-511` + `internal/control/flags/normalizer.go:40-46,73-78`  
**Issue:** `usage()` lists `--continue / -c` and `--print / -p` as universal. Normalizer correctly omits Codex from `-c`/`-p` so native `--config` / `--profile` pass through. Users following top-level help who run `nexus codex -c` get a bare `-c` (config), not continue — silent wrong behavior.

**Fix:** Split help rows: document Codex exceptions next to the aliases (mirror `BuiltinAliases` descriptions), or stop listing `-c`/`-p` as universal in `usage()`.

---

#### HI-02: Prefer PATH `cloudflared` with zero integrity check

**Severity:** HIGH  
**File:** `internal/control/web/tunnel.go:131-135,163-173`  
**Issue:** `EnsureCloudflared` returns the first PATH hit without SHA-256 verification. Cached `$DATA/bin/cloudflared` re-verify uses `filepath.Base(path)` (`cloudflared`) which is **not** in `cloudflaredChecksums` keys (`cloudflared-linux-amd64`, …), so verify always no-ops after first install. Compromised local binary becomes the tunnel endpoint for the control plane.

**Fix:** Never prefer unverified PATH for Nexus-managed tunnels (or require explicit opt-in). Key checksum map by GOOS/GOARCH (same as download), re-verify on every use, quarantine on mismatch.

---

#### HI-03: Darwin cloudflared checksums are identical placeholders

**Severity:** HIGH  
**File:** `internal/control/web/tunnel.go:42-43`  
**Issue:** `cloudflared-darwin-amd64` and `cloudflared-darwin-arm64` share the same hash `0019dfc4…49b5`. Real multi-arch release binaries cannot share one digest. Download path fail-closes if wrong; if ever “correct” by accident, integrity is meaningless.

**Fix:** Pin real GitHub release SHA-256 per arch (or remove Darwin entries until filled so download stays fail-closed).

---

### MEDIUM

#### ME-01: Fuzzy session-ID matching in marker / index merge

**Severity:** MEDIUM  
**File:** `cross_account_sessions.go:521-524,364-365` + `codex.go:1126`  
**Issue:** `strings.Contains` on session IDs can false-positive exclude (or find) rollouts when one ID is a substring of another / filename noise. Quota under-attribution or wrong index merge.

**Fix:** Exact match on `extractSessionIDFromRolloutName` only; drop `Contains` fallbacks.

---

#### ME-02: `IsTUILocked` treats I/O errors as “unlocked”

**Severity:** MEDIUM  
**File:** `internal/core/provider/adapters/codex/tui_lock.go:73-79`  
**Issue:** On open/flock error, returns `false`, so `usage()` may spawn app-server while the home is actually contested or unreadable.

**Fix:** Propagate error to callers; on uncertainty skip app-server (fail soft to rollout / unknown), do not assume unlocked.

---

#### ME-03: Dispatcher vs `usage()` — minor alias gaps only

**Severity:** MEDIUM (docs completeness)  
**File:** `internal/app/app.go:104-208` vs `444-514`  
**Issue:** Switch covers all documented commands. Undocumented switch aliases: `open`, `upgrade`, `quota`, `swap`, `list`/`ls`, `plans`/`agent`/`project`, `ui`, `__control-host`. Help `completion <bash|zsh|fish>` omits `powershell` handled at `1759`. No missing *implemented* primary command found.

**Fix:** Document public aliases; keep `__control-host` internal-only; add `powershell` to help.

---

### LOW

#### LO-01: AGY `CrossAccountResume: true` is host-brain share, not sibling import

**Severity:** LOW  
**File:** `internal/core/provider/adapters/agy/agy.go:57,99-105,957-974`  
**Issue:** Capability flag is `true`; implementation only `linkSharedAgyItems` from host `~/.gemini` (symlink/copy). No sibling-profile seed like Codex. Matches design doc (“shared .gemini brain”) but is weaker than Codex CrossAccountResume. Claude correctly `false` (`claude.go:35`).

**Fix:** Keep flag if product intent is host-brain visibility; otherwise narrow docs/capability naming to avoid equating with Codex sibling import.

---

#### LO-02: Windows `isLockBusy` only maps `ERROR_LOCK_VIOLATION`

**Severity:** LOW  
**File:** `tui_lock_windows.go:66-68`  
**Issue:** Other LockFileEx failure codes may surface as hard errors instead of “busy”, changing probe vs TUI contention behavior on Windows.

**Fix:** Treat known busy/lock-conflict errnos uniformly as busy.

---

#### LO-03: Debug / silent continues in seed walk

**Severity:** LOW  
**File:** `cross_account_sessions.go:257-258`  
**Issue:** Failed `hardlinkOrCopy` is silently skipped — fine for best-effort seed, but hides partial CrossAccountResume failures from operators.

**Fix:** Optional debug log / doctor check for seed failures.

---

## Cross-file notes (deep)

| Chain | Verdict |
|---|---|
| TUI `AcquireTUILock` → `runAppServerRateLimits` `TryAcquireTUILock` | Correct mutual exclusion when both use same `CODEX_HOME` (`app_server_usage.go:281-288`). |
| `Prepare` releases lock before `Run` re-acquires | Window allows probes between prepare and interactive start; flock still serializes — OK. |
| API tunnel start | Correctly sets `SetTunnelActive(true)` — CLI path does not (CR-01). |
| Flag `-c`/`-p` for Codex | Implementation intentional; help wrong (HI-01). |
| Claude CrossAccountResume | Correctly `false`. AGY claim partial (LO-01). |

---

_Reviewed: 2026-09-12T16:42:41Z_  
_Reviewer: gsd-code-reviewer_  
_Depth: deep (hot paths only)_
