# IAPro Nexus Canonical Hardening Design

## Scope

Harden the existing Nexus Workspace OS around the approved canonical model:
Project → Workspace → Persistent Agents, Project Shells, Files/Tools,
Composer/Flow Drafts and Flow Runs. Preserve existing SessionHost, Mission
Runner, Resource Scheduler, Maestro boundary and provider drivers.

## Consistency findings

- Project, Workspace V2, stable logical Views, persistent Agents and ephemeral
  runtimes already exist in `internal/nexus` and `web/src/workspace`.
- EventBus, terminal heuristics, authenticated WebSocket/REST, SQLite
  migrations, Mission Runner, ContextCapsule and WorkReceipt already exist.
- Existing attention is runtime-local and heuristic-driven; it needs a global,
  persisted normalized stream and explicit notification transport policy.
- Existing layout persistence is a project API/cache path but does not yet
  expose pure geometry arrangement and full recovery semantics.
- Existing provider drivers expose continuity-related data, but capability
  contracts for verified resume, fork, discovery and adoption need explicit
  typed boundaries.
- Baseline `go test ./...` passes. Baseline `go test -race -count=1 ./...`
  exposes a real registry singleton race in test reset versus detector use;
  this is a P0 prerequisite and must be fixed first.

## Architecture

Add focused, testable contracts rather than parallel orchestration systems:

1. `internal/control/registry` becomes race-safe for singleton lifecycle.
2. `internal/nexus/attention` owns normalized states, bounded redacted events,
   idempotent replay and project summaries; existing EventBus remains the event
   fanout primitive.
3. Driver capability interfaces own provider-native resume, fork, session
   discovery and provider status detection. Core never fabricates evidence.
4. `web/src/workspace/arrange.ts` is a pure deterministic geometry function;
   layout persistence remains SQLite-backed with browser cache only as an
   optimistic layer.
5. Browser notification policy chooses in-app, browser or push transport from
   visibility/focus state. A service worker handles push clicks without
   duplicate Nexus tabs. Payloads contain only bounded redacted metadata.
6. Recovery reconstructs Project/Workspace/View/geometry/focus before runtime
   treatment; dead runtimes receive new generations and shells are marked LOST.

## Verification strategy

Every new behavior is introduced by a failing unit test, then minimal code,
then refactor. Fitness tests cover identity, Flow DAG ordering, capsule/receipt
  boundaries, scheduler reuse, notification deduplication, geometry
determinism, persistence/reload and recovery. Existing full Go and web gates
are rerun at the final HEAD. Unsupported native platforms are reported as
`UNVERIFIED_PLATFORM`, never inferred as passing.

## Explicit non-goals

No new planner, scheduler, runner, memory, terminal host, tmux core,
monetization, cloud account, team SaaS, semantic memory or Maestro replacement.
