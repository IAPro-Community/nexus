# Nexus V1 — Baseline Audit

- **Date:** 2026-08-28
- **Repo:** `github.com/IAPro-Community/nexus` (evolving to IAPro Nexus)
- **Integration branch:** `feat/nexus-v1`
- **Baseline HEAD:** `f9cd67988ac295b9a5c6b2f35e964243ad4e6917` (validated production-readiness branch, previously pushed to origin)

## Environment

| Item | Value |
|---|---|
| OS | Linux amd64 (container) |
| Go | go1.25.0 |
| Frontend | Bun 1.4.0 (canonical, bun.lock), React 19, TS 5.9.3, Tailwind v4 |
| SQLite driver | modernc.org/sqlite v1.57.0 (pure Go, no CGO) |

## Baseline verification (before feature work)

| Command | Result |
|---|---|
| `go vet ./...` | 0 warnings |
| `go test -race ./...` | 24 packages ok, 0 fail |
| `bun run typecheck` / `lint` / `test` / `build` | pass |
| GoReleaser snapshot (prior gate) | 6 artifacts + checksums |
| Windows/macOS runtime E2E | CI + user-local pending (prior gate) |

## Pre-existing architecture (verified present)

- Control Core: RuntimeRegistry, SessionHost (persistent `__control-host`), Linux PTY, Windows ConPTY (real), UDS + Named Pipes, WebSocket terminal, Web Control, Projects (workspace store), Usage/Quota, Profiles, Account Handoff (transactional + ResumeVerifier), Context Handoff, ULID runtime IDs, bounded framing, protocol version, security headers + strict bind policy, `internal/buildinfo`.

## Gaps for Nexus V1 (this initiative)

| Gap | Status |
|---|---|
| Durable product state (SQLite) | ADDED — `internal/nexus/store` + migrations |
| Project domain (ULID/slug/canonical/repo) | ADDED — Project CRUD + path canonicalization |
| Persistent Agent domain (Agent/Revision/Generation/Lineage) | ADDED — full store + start/stop orchestration |
| Project/Agent Web API | ADDED — `/api/v1/projects`, `/api/v1/agents`, agent terminal WS |
| Web-first UI (AppShell, tokens, primitives, Projects→Agents→Terminal) | ADDED — Nexus shell + vertical slice |
| Maestro machine-readable contract | PENDING — Maestro repo owns schemas; Nexus bridge planned (Gate 6) |
| Resource Scheduler (cross-provider) | PENDING — Gate 5 |
| Mission Beta | PENDING — Gate 7 (Maestro salvage candidate) |
| `iapro` CLI + `iapro doctor` | PENDING — Gate 8 |

## Verification of this initiative

- `go test -race ./...` → 25 packages ok
- Nexus store tests: migrations idempotent, Project CRUD/MRU, Agent lifecycle, revisions, generations, lineage, IDOR guard, layouts
- Web API test: project/agent/layout round-trip + terminal 404-without-runtime + CSRF
- Live E2E: create Project (ULID `prj_…`), create Agent (`agt_…`), start agent → runtime `fake-…` RUNNING, generation recorded, agent terminal WS upgrade **101**, project **persists across web restart** (same ID)
