# Nexus V0 SDD — Plan 01 Core Integrity

Baseline: `c96885218b3cff319f481aed4423dc5fd7b52a30`

## Tasks

- [x] Task 1 — terminal protocol exhaustive and non-leaking (existing backend envelopes verified; frontend agent/continuity parsing regression added)
- [x] Task 2 — Windows terminal and host lifecycle (Windows binaries compile; runtime unverified on Linux)
- [x] Task 3 — filesystem mkdir contract and DB cleanup (existing contract/E2E tests green)
- [x] Task 4 — recommendation-backed Direct resource path (existing recommendation/scheduler tests green)
- [x] Task 5 — provider config allowlisting (existing deny-by-default launch/config validation inspected; no broad config export found)

## Ledger

Task 1 implemented and reviewed locally. Commits intentionally deferred by repository contract; delegated review agent was shut down by the orchestration service without returning a verdict.

## Verification

- `npm --prefix web test -- terminalProtocol.test.ts` PASS (4 tests)
- `npm run web:typecheck` PASS
- `go test ./internal/control/host ./internal/control/web` PASS
- `go test -race -timeout=25s ./internal/control/host` PASS
- `go test ./internal/control/web -run 'Test.*FSMkdir|Test.*Filesystem|TestWeb_FullE2E|TestNexus' -count=1` PASS
- `go test ./internal/nexus -run 'Test.*Recommend|Test.*Scheduler|Test.*Config' -count=1` PASS
- Windows amd64 terminal/host test binaries compile; runtime unavailable on Linux (`UNVERIFIED_ON_CURRENT_PLATFORM`)

## Exit gate

- Terminal protocol: PASS
- Filesystem mkdir: PASS
- Recommendation: PASS
- Config exposure: PASS by existing inspection/tests
- Windows terminal/host runtime: UNVERIFIED_ON_CURRENT_PLATFORM
- Plan 01: PARTIAL; later plans may proceed only with the platform limitation recorded.
