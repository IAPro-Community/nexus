# Nexus Core

Nexus Core is the shared operational product used by CLI, Web, and Desktop.
The surfaces select, render, and transport commands; durable state, provider
selection, runtime control, projects, workspaces, events, and usage evidence
live under `internal/`.

See [`consumer-matrix.md`](consumer-matrix.md) for the current ownership
contract between CLI, Web, Desktop and the Core/API boundaries.

## Current ownership

| Domain | Current owner | Persistence/side effects | Consumers |
| --- | --- | --- | --- |
| Projects, agents, plans, missions | `internal/nexus` and `internal/nexus/store` | Local SQLite/store files | CLI, Web, Desktop through Core/API |
| Runtime/process sessions | `internal/control/registry`, `internal/control/driver`, `internal/runtime` | Runtime registry and OS processes | CLI, Web, Desktop |
| PTY/terminal transport | `internal/control/terminal`, `internal/control/web` | OS PTY/ConPTY and WebSocket | Web/Desktop/CLI attach |
| Providers | `internal/core/provider`, `internal/control/driver` | Provider-local config and profiles | Core, CLI, API |
| Usage/quota | `internal/core/quota`, provider adapters | Evidence-bearing usage cache | Core, CLI, Web |
| Events | `internal/control/events`, Nexus event store | Structured event records and correlated timeline metadata | API, timeline, diagnostics |
| Auth/security | `internal/control/web`, `internal/core/security` | Loopback session store, CSRF | Web/Desktop API |
| Maestro | `internal/nexus` integration boundary | Optional external catalog/status | Explicitly optional |

## Dependency direction

```text
CLI / Web / Desktop transport
             ↓
      application handlers
             ↓
       Nexus/Core packages
             ↓
 domain interfaces and models
             ↑
 infrastructure (OS, provider CLI, filesystem, network)
```

`internal/control/web` is a transport adapter. Its route registration is now
grouped in `routes.go`; handler extraction remains incremental and must not
move product rules into HTTP handlers.

The first domain extraction is complete: project and agent handlers are in
`handlers_projects.go` and `handlers_agents.go`, while their authentication,
CSRF, and path dispatch are in `routes_projects_agents.go`. The methods still
use `NexusHandler` during this compatibility phase; moving application logic
into narrower Core services is a later step backed by domain contract tests.

Resource listing, allocation, and recommendation transport handlers are now
isolated in `handlers_resources.go`; provider selection remains delegated to
the existing Nexus scheduler and resource APIs.

Resources also has an explicit application boundary in
`internal/nexus/resource_application.go`. The transport depends on its narrow
list/allocate/recommend contract, while recommendation scoring remains in the
domain package and cancellation is checked at the application boundary.

Projects, Agents and WorkPlan CRUD/revisions follow the same pattern through
`project_application.go`, `agent_application.go` and `plan_application.go`.
Composer transport operations use `composer_application.go`; its workflow
rules remain on Nexus because intelligence and optional Maestro still cross the
same boundary and are not duplicated in a second service.

Mission planning transport operations use `mission_application.go` for CRUD,
detail, tasks and assignments. This boundary is deliberately distinct from
`MissionRun` execution: run scheduling, process lifecycle and execution remain
in the existing mission service/runner until their cross-domain contracts can
be isolated without duplicating runtime rules.

The Web transport-facing MissionRun calls use `run_application.go` for start,
list, read, step and control actions. The service is a thin application
boundary; it delegates lifecycle policy to Nexus/runner rather than becoming a
second state machine.

Raw runtime API operations use `runtime_application.go` for list, start,
detail/capabilities, cleanup/delete/title, stop, interactive input, and
handoff delegation. The service owns the application contract and delegates
process creation and IPC to supervised control-plane infrastructure; it does
not duplicate process state or provider policy.

The transport ownership map is now:

| Transport module | Responsibilities |
| --- | --- |
| `handlers_projects.go` | projects, context, layout, project shell/events |
| `handlers_agents.go` | agents, lifecycle actions, config, runtime resolution |
| `handlers_resources.go` | resource listing, allocation, recommendation |
| `handlers_planning.go` | plans, Composer, prompt artifacts, flows, runs |
| `handlers_plan_revisions.go` | plan restore and revision diff |
| `handlers_missions.go` | missions, tasks, assignments |
| `handlers_mission_schedules.go` | durable mission schedules |
| `handlers_intelligence.go` | intelligence and clarification endpoints |
| `handlers_git.go` | project Git inspection and checkout |
| `handlers_maestro.go` | optional Maestro integration and updates |

`handlers_nexus.go` is now limited to shared handler construction, doctor, and
common configuration rather than domain endpoint implementations.

## Explicitly unchanged

This consolidation does not add providers, scraping, an orchestration engine,
a visual redesign, or mandatory Maestro installation. Those remain deferred.

## Agent execution semantics

An Agent is a persistent operational identity. Its provider is an execution
adapter, not its identity. `AgentSpec` is the typed, provider-independent
specialization persisted in the effective `AgentRevision.Config`; legacy
role-only Agents normalize to a minimal spec without invented behavior.

`RuntimeGeneration` keeps the revision that produced the execution, so a
provider switch (for example Codex to Claude) preserves the Agent identity and
specialization. `WorkPackage.role` is a task-time role and is composed after
the persistent `AgentSpec.role`, e.g. “Senior Developer acting as Reviewer”.

Direct, automated, and orchestrated UX modes converge on the existing
intelligence compiler through `CompileExecutionContext`. Its sections retain
provenance for agent, task, project, optional Maestro guidance, and runtime
constraints. Maestro contributes methodology and gates only when enabled; it
is not a Nexus runtime.
