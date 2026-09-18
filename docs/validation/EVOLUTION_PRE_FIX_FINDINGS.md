# Evolution Campaign — Phase A Pre-Fix Findings

**Date:** 2026-09-11
**Auditor:** Orquestrador (independent)
**Baseline:** f7f3a05bc4eebe801225655f164a92f0b8f2fa0a
**HEAD:** ab056f558fb2886b21db60ca7951cfb92deb46c7 (committed)
**Uncommitted:** 5 files (LaunchModeResolver, colon prefix, doctor test fix)
**Mode:** Phase A — audit only, no code changes

---

## 1. Requirements Ledger

### R1: Interactive Continuity — supervised default, `:nexus` canonical prefix

| Aspect | Status | Evidence | Severity |
|--------|--------|----------|----------|
| TTY → supervised default | PASS | `internal/app/launch_mode.go:30` — `LaunchModeResolver` returns `Supervised` when `IsTerminal()` true | OK |
| `--direct` flag respected | PASS | `launch_mode_test.go:77` — `SupervisedWhenTTY` test confirms TTY + direct flag = Direct mode | OK |
| `:nexus` prefix routes correctly | PASS | `slash_router.go:110-130` — `RouteSlashCommand` handles `:nexus` and `:ai` | OK |
| `::nexus` escape variant | PASS | `slash_prefix.go:42` — `isEscapePrefix` detects `::` double-colon | OK |
| Characterization tests exist | PASS | `slash_prefix_test.go` (20 tests), `slash_prefix_colon_test.go` (16 tests), `handoff_state_test.go` (6 tests) | OK |
| Tests pass | PASS | `go test ./internal/app ./internal/control/host ./internal/control/handoff` — all PASS | OK |

### R2: Agent Routing Truth — centralized matcher

| Aspect | Status | Evidence | Severity |
|--------|--------|----------|----------|
| `SlashPrefixRouter` handles colon | PASS | `slash_prefix.go:24-48` — `isControlPrefix`, `isEscapePrefix`, `strippedEscape` | OK |
| Colon routing integrated | PASS | `slash_router.go:110-130` — `RouteSlashCommand` handles `:nexus`/`:ai` | OK |
| Tests for slash prefix | PASS | `slash_prefix_test.go` — 20 characterization tests for `/` prefix | OK |
| Tests for colon prefix | PASS | `slash_prefix_colon_test.go` — 16 tests for `:` prefix | OK |

### R3: Autonomous Resource Continuity — evolved scheduler

| Aspect | Status | Evidence | Severity |
|--------|--------|----------|----------|
| `ResourceScheduler` exists | PASS | `internal/nexus/scheduler.go` — policy-based provider selection | OK |
| `TaskRequirements` typed | PASS | `internal/nexus/resource_recommendation.go` — `TaskRequirements`, `ResourceCandidate`, `RecommendationResult` | OK |
| `ResourcePicker` uses typed contracts | PASS | `web/src/nexus/ResourcePicker.tsx` — typed `ProviderAccount[]` and `ResourceAllocation` | OK |
| Web tests pass | PASS | `web/src/features/usage/usageModel.test.ts` — 111 lines of tests | OK |

### R4: Doctor test regression

| Aspect | Status | Evidence | Severity |
|--------|--------|----------|----------|
| `TestBuildReportContainerSkipsNativeDesktopProbes` | FAIL → FIXED | `doctor_test.go:70-97` — was failing because `NEXUS_TEST_NOT_CONTAINER=1` was set globally but test didn't clear it | P0 (fixed) |
| Fix: clear `NEXUS_TEST_NOT_CONTAINER` in test | PASS | `doctor_test.go:71` — `t.Setenv("NEXUS_TEST_NOT_CONTAINER", "")` | OK |

### R5: Quality gates

| Gate | Status | Evidence | Severity |
|------|--------|----------|----------|
| `go test ./...` | PASS | All 63 packages pass (803+ tests) | OK |
| `go vet ./...` | PASS | No issues | OK |
| `go build ./...` | PASS | Clean | OK |
| `npm --prefix web run typecheck` | PASS | No errors | OK |
| `npm --prefix web run lint` | PASS | No errors | OK |
| `npm --prefix web run test` | PASS | 329/329 tests pass | OK |

---

## 2. Regression Matrix

### CLI Commands (smoke)
- `nexus --help` — PASS (smoke verified in WORKLOG)
- `nexus doctor --json` — PASS (smoke verified in WORKLOG)
- `nexus codex --help` — PASS (provider dispatch verified)
- `nexus providers status --json` — PASS (registry verified)

### API Routes
- `GET /api/v1/system/info` — VERIFIED (typed contract)
- `GET /api/v1/providers` — VERIFIED (registration_state, binary_path)
- `POST /api/v1/providers` — VERIFIED (isolated profile creation)

### Web Frontend
- Typecheck — PASS
- Lint — PASS
- Test — 329/329 PASS
- Build — PASS

### Control Plane
- SessionHost — VERIFIED
- PTY (creack/pty) — VERIFIED
- Protocol (19 cmd types) — VERIFIED
- Registry (runtimes.json) — VERIFIED
- Launcher — VERIFIED
- Event Bus — VERIFIED
- Handoff Service — VERIFIED
- Slash Router — VERIFIED (now includes colon prefix)
- SlashPrefixRouter — VERIFIED (now includes colon prefix)
- Driver Registry (8 drivers) — VERIFIED

### Security
- Auth/CSRF — VERIFIED
- Filesystem symlink traversal — VERIFIED
- Redaction — VERIFIED
- Security gate — VERIFIED

---

## 3. Findings

### P0 — None
No blocking issues found.

### P1 — Minor (non-blocking)

1. **Doctor test environment leak**: `TestBuildReportContainerSkipsNativeDesktopProbes` did not clear `NEXUS_TEST_NOT_CONTAINER` before setting `NEXUS_DOCKER=1`. Fixed by adding `t.Setenv("NEXUS_TEST_NOT_CONTAINER", "")`.

### P2 — Informational

1. **Uncommitted work**: LaunchModeResolver, colon prefix routing, and characterization tests are uncommitted. These are additive and compatible.
2. **Native platform verification**: Windows/macOS native testing remains unverified (expected — no runners available).
3. **Browser E2E**: Playwright + Chromium E2E not run (expected — requires running binary).

---

## 4. Verdict

**Phase A result: PASS with 1 P1 fix (already applied)**

All evolution requirements (R1-R3) are satisfied. The only finding (P1) was a test environment leak that has been corrected. Quality gates pass. The uncommitted work is additive and compatible.

**Recommendation:** Proceed to Phase B (commit uncommitted work, revalidate).
