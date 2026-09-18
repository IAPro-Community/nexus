# Runtime lifecycle

These terms describe the current Nexus model and are intentionally distinct.

| Concept | Meaning | Current owner |
| --- | --- | --- |
| Project | Persistent work context and canonical path | `internal/nexus/store` |
| Agent | Durable project-scoped identity/configuration | `internal/nexus/store` |
| Session | Logical provider conversation or resumable context | provider/session indexes |
| Runtime | Live execution of a provider/agent session | runtime application service + injected control registry |
| Process | OS process backing a runtime | control launcher/driver |
| Terminal/PTTY | Interactive I/O channel for a runtime when supported | terminal package |
| Workspace | Physical directory/worktree used by a project/runtime | workspace store |
| WorkPlan | Persisted representation of planned work | Nexus plan/store |
| Flow | Executable decomposition of a WorkPlan | Nexus flow packages |
| Run | One execution attempt of a plan/flow | Nexus runner |
| Mission | Goal-oriented run with tasks and verification intent | Nexus mission packages |
| Maestro | Optional external method/policy integration | explicit integration boundary |

## Direct lifecycle

```text
Project → Agent/Session → Provider/Profile → Runtime → Process → PTY/events
```

The process can exit while the Agent and Session remain durable. Runtime state
is operational and may be stopped, failed, or recovered without deleting the
project identity.

## Automated and orchestrated lifecycles

Automated work adds `Composer → WorkPlan → Flow → Run`. Orchestrated work adds
`Mission` and may consult optional Maestro policies. Maestro does not own
processes, providers, projects, or terminal state.

Mission planning CRUD (including tasks and assignments) is exposed through the
Core application boundary, while `MissionRun` execution remains owned by the
runner and its lifecycle service. The `RunRepository` contract propagates
cancellation for persistence and lease operations in both memory and SQLite
implementations, preventing canceled transport requests from mutating run
state.

Web start/list/detail/control calls pass through
`internal/nexus/run_application.go` for MissionRun operations; the application
service composes the existing runner and lifecycle methods without duplicating
their state machine.

Raw runtime list/start/detail/delete/title, stop, respond and handoff calls pass
through `internal/nexus/runtime_application.go`. The service delegates IPC and
provider-specific handoff to control-plane packages while preserving request
cancellation and process boundaries.

The protocol client preserves the legacy `Send` API and additionally exposes
`SendContext`, `StopContext`, and `SubmitPromptContext`. Runtime stop and input
therefore interrupt blocked local socket/pipe operations when the owning
request is canceled; the five-second RPC deadline remains the final fallback.

The launcher now passes its owning registry into `SessionHost`; host lifecycle
and attention metadata updates therefore stay in the same registry instance as
the application service. `SessionHost` retains a singleton fallback only for
legacy direct constructors that do not provide a registry. This prevents a
custom/test registry from silently diverging from the process-global runtime
catalog.

The registry now exposes `CanTransition` and `TransitionState` for new
application-owned changes. The operational path is intentionally small:

```text
STARTING → RUNNING → WAITING|APPROVAL|HANDOFF|STOPPING
STOPPING → STOPPED|FAILED
STOPPED|FAILED → STARTING (explicit restart)
```

`UpdateState` remains available for backward-compatible administrative
reconciliation and stale-record repair; it is not the preferred API for new
runtime lifecycle code.

Lifecycle events carry an optional `CorrelationID`. Mission failover and
MissionRun lifecycle events use the `MissionRun` ID; account and context
handoff events use the lineage ID. The durable event metadata store persists
this value through migration 0015. Existing events without a broader operation
retain an empty correlation value. See [`events.md`](events.md) for the stable
event taxonomy and timeline rules.

## Execution context

Direct, Automated, and Orchestrated work converge on the same context compiler
and runtime lifecycle. The compiled result keeps provenance sections (`agent`,
`project`, `task`, optional `maestro`, and `runtime`) before rendering provider
text, so Agent identity remains separate from the executor and diagnostics can
explain why an instruction was applied.

The codebase has legacy string states for compatibility. New state changes
should use the registry transition contract and corresponding events. Provider
attention states remain operational sub-states of the runtime and do not create
a second process lifecycle.
