# Nexus Product Consolidation

## Context and baseline

The branch `feat/nexus-maximum-delivery` is at `cf1c633` and the worktree is
dirty at the start of this campaign. Pre-existing staged/deleted `.omx`,
`.superpowers`, `loadtest`, and `DEV/` changes are out of scope and must be
preserved. `main` is 117 commits behind the branch, so the
branch-vs-main diff is historical context rather than an instruction to clean
all changed files. Go tests and vet are being run as the baseline; frontend
and repository gates are recorded in `docs/refactoring/BASELINE.md`.

The code already contains useful foundations that must be preserved:

- `internal/core/provider` has segregated provider interfaces and a registry.
- `internal/core/quota` has honest `UNKNOWN`, source, and observation handling.
- `internal/control/events` has structured lifecycle events.
- `web/src/nexus/api.ts` is the existing web API boundary.
- Its transport is now shared with the legacy runtime client in
  `web/src/api.ts`; domain methods retain the Nexus facade while auth, CSRF,
  base URL and error handling have one implementation.
- CLI/Web/Desktop ownership is recorded in `docs/architecture/consumer-matrix.md`;
  the CLI remains a direct local Core consumer by design.
- The evidence-based DoD audit is maintained in
  `docs/refactoring/definition-of-done-audit.md`; it records partial areas and
  deferred work instead of treating local green gates as global completion.
- loopback authentication, CSRF, filesystem restrictions, and platform-specific
  process code are existing security contracts.

## Global constraints

- Preserve CLI aliases and existing `/api/v1` responses.
- No new web framework, provider integration, scraping mechanism, or visual
  redesign.
- Do not alter unrelated dirty worktree changes or automatically commit/push.
- Every behavior change gets a focused test before or with the implementation.
- Keep dependencies directed from transport/UI to application/core and keep
  Maestro optional and outside Nexus business rules.

## Execution tasks

### Task 1 — Baseline and architecture inventory

Create the baseline ledger and a compact responsibility map covering the Core,
API, CLI, Web, Desktop, providers, quota, runtime, events, filesystem, and
Maestro boundary. Capture exact commands, exit codes, and pre-existing failures.
Files: `docs/refactoring/BASELINE.md`, `docs/architecture/core.md`.

### Task 2 — Modular API route registration

Extract route registration from `internal/control/web/server.go` into cohesive
registration functions/files in the same package. Keep the existing handlers,
middleware, paths, methods, and response bodies unchanged. The server should
compose route groups through one registration entrypoint; route tests must prove
health, auth, projects, agents, runtime, filesystem, plans, missions, and
Maestro routes remain reachable.

### Task 3 — API metadata and stable error contract

Document `/api/v1`, add a read-only system metadata endpoint only if no existing
equivalent exists, and expose version/capability information from existing
build information and registered providers. Normalize only newly introduced
metadata/errors; preserve legacy endpoint payloads. Add semantic handler tests.

### Task 4 — Provider and usage contract documentation/tests

Document the existing segregated provider interfaces, capability model, direct
provider command compatibility, and quota evidence model. Add tests for the
invariants that `UNKNOWN` is not zero/unlimited and that source/observed time,
window, reset, status, and confidence are preserved where the current model
supports them. Do not add scraping or providers.

### Task 5 — Lifecycle and product interaction model

Validate terminology against code and write `runtime-lifecycle.md` and
`interaction-model.md`, explicitly separating Project, Agent, Session, Runtime,
Process, PTY, Workspace, WorkPlan, Flow, Run, Mission, and optional Maestro.
Document states that actually exist and label proposed states as future work.

### Task 6 — CLI compatibility matrix and focused contracts

Generate a command matrix from `Run` dispatch and `usage()`, identify aliases,
help-only entries, exit-code behavior, and implemented subcommands. Add focused
tests for parsing/help/unknown command/provider aliases without changing the
manual parser or public command names unless a proven inconsistency requires it.

### Task 7 — Client/API and security review

Review Web API access for scattered HTTP calls, replace only concrete duplicates
with the existing Nexus API boundary, and add contract assertions for auth,
CSRF, error handling, and version metadata. Review filesystem traversal,
credential logging, context propagation, and process cancellation; fix only
issues evidenced by code/tests.

### Task 8 — Durable docs and final verification

Create/update the required architecture, product, compatibility, and deferred
work documentation. Refresh `DEV/WORKLOG.md`, `DEV/VERIFY.md`, `DEV/HANDOFF.md`,
`DEV/CONTEXT.md`, and `DEV/SPECS/ACTIVE.md`. Run all applicable Go/frontend
quality gates, race/vet/security checks, CLI smoke tests, and review the final
diff. Record Linux/Windows/macOS evidence separately and mark unverified
platforms as `NOT VERIFIED`.

## Review checklist

- No handler became an application service.
- No public CLI/API contract was silently changed.
- No business rule is duplicated in Web/Desktop/CLI by the changes.
- No provider-specific switch was added to shared policy code.
- No secret, token, or credential is logged.
- New tests assert behavior rather than snapshots or implementation details.
- Documentation distinguishes current evidence from deferred intent.

## Execution status — 2026-09-10 continuation

- Tasks 1–6: baseline, route composition, metadata contract, existing provider/
  quota documentation, lifecycle docs, and focused CLI contracts completed.
- Task 7: Web API boundary and security review completed for the touched scope;
  Projects/Agents transport handlers and dispatch are now separated into
  `handlers_projects.go`, `handlers_agents.go`, and
  `routes_projects_agents.go`; resource transport handlers are in
  `handlers_resources.go`. Planning, Intelligence, Missions, Git, Maestro,
  schedules, and plan revision handlers are also separated by domain.
- Task 8: durable docs updated; the latest full quality gate passed after the
  Mission planning boundary and MissionRun cancellation-contract slices.
  Remaining Core extraction is
  intentionally incremental and must continue by domain with contract
  coverage.

Core application-service slices now cover Resources, Projects, Agents,
WorkPlan CRUD/revisions, Missions planning and the Composer transport contract,
with cancellation/CRUD/delegation tests. MissionRun transport operations now
use a thin `RunApplicationService`; lifecycle policy, intelligence,
compilation, runtime and execution remain deliberately on the Nexus aggregate
where dependencies still cross multiple domains. This is incremental progress,
not a claim that every domain has been extracted.

The runtime slice now also has `RuntimeApplicationService` for raw runtime
list/start/detail/capabilities/cleanup/delete/title/stop/respond/handoff
operations. The service delegates protocol-specific work to control-plane
dependencies, preserving those boundaries without leaving orchestration in the
HTTP adapter.
