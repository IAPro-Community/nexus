# Nexus Product Consolidation Baseline

Date: 2026-09-10
Branch: `feat/nexus-maximum-delivery`
HEAD: `cf1c633`

## Repository state

- `git status --short`: dirty at baseline capture. Pre-existing staged/deleted
  `.omx`, `.superpowers`, `loadtest`, and `DEV/` entries are preserved and are
  not part of this campaign.
- `main...HEAD`: 117 commits of Maximum Delivery history are present; this is
  not treated as permission to rewrite historical changes.
- Go module: `github.com/IAPro-Community/nexus`, Go 1.25.
- Frontend: React/TypeScript under `web/`, Bun/Yarn-compatible scripts.

## Baseline commands

| Command | Result | Evidence |
| --- | --- | --- |
| `go test ./...` | PASS | Fresh run; all listed packages exited 0. |
| `go vet ./...` | PASS | Fresh run exited 0. |
| `make format-check` | PASS | Fresh run exited 0. |
| Frontend `bun run quality:full` | PASS | 62 test files / 314 tests; build completed. One pre-existing ESLint warning. |
| Frontend `bun run verify` | PASS | Format, typecheck, style, test, i18n, build, embed-sync, and UI markers passed. |
| `go test -race ./...` | PASS | Fresh run; all packages exited 0. |
| `make lint-go` | PASS | `0 issues.` |
| `make security` | PASS | `No vulnerabilities found.` |

## Structural findings

- `internal/control/web/server.go` registers dozens of routes directly and
  also owns server construction, auth session endpoints, SPA fallback, and
  security middleware.
- `internal/control/web/handlers_nexus.go` combines projects, agents,
  resources, Maestro, plans, flows, runs, missions, intelligence, and
  filesystem handlers.
- `internal/app/app.go` is the CLI composition/dispatch root and contains the
  public help contract plus provider dispatch.
- Provider and quota foundations are already present and should be documented
  and tested before considering deeper extraction.
- Web API access is already mostly centralized in `web/src/nexus/api.ts`; this
  is an existing boundary, not a missing feature.
- Frontend dependencies were initially incomplete (`eslint-plugin-react-hooks`
  missing); `bun install --frozen-lockfile` restored the lockfile-defined
  packages without changing the dependency contract.

## Classification policy

Any failure observed before the first code change is `PRE-EXISTING`; a failure
introduced after a change is `INTRODUCED`; a pre-existing failure fixed during
the campaign is `FIXED`; unavailable platform/tooling evidence is `BLOCKED` or
`NOT VERIFIED`, never `PASS`.

## Continuation evidence — 2026-09-10

The Projects/Agents handler and route extraction was verified incrementally and
again through the full gates: `go test ./...`, `go test -race ./...`, `go vet
./...`, and `make quality-full` all exited 0.

## Final evidence for the current extraction milestone

- `make quality-full`: **PASS**.
- `go test -race ./...`: **PASS** independently.
- `gofmt -l` for all touched Go files: no output; scoped `git diff --check`:
  **PASS**.
- Frontend: 62 Vitest files / 314 tests **PASS**; the existing React hook
  dependency warning remains a warning and was not suppressed.
- `make security`: **PASS**, no vulnerabilities found.
- Linux **VERIFIED**; Windows and macOS **NOT VERIFIED** locally.

The gate exposed a test-isolation issue in the process-wide Nexus singleton
when tests replaced temporary data directories. `ResetDefaultForTest` now
closes the store and resets the singleton between those tests. Concurrent Core
work outside the transport extraction was preserved, with only formatting and
mechanical lint corrections applied where required by the gate.
