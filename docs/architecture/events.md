# Nexus events and runtime timeline

Nexus has two compatible event representations:

- `internal/control/events.Event` is the live in-memory event bus contract used
  by runtime subscribers, terminal streams and the API history endpoint.
- `internal/nexus/store.EventMetadata` is the durable activity projection used
  by project timelines. It intentionally stores metadata rather than raw
  provider output.

The event `type` values are stable uppercase identifiers for compatibility with
existing clients. They are grouped by the current taxonomy:

| Family | Current identifiers | Durable correlation |
| --- | --- | --- |
| Process | `PROCESS_STARTED`, `PROCESS_EXITED` | runtime ID by default |
| Runtime | `RUNTIME_STARTED`, `RUNTIME_STOPPED`, `RUNTIME_FAILED` | optional lineage ID; runtime ID by default |
| Session/agent | `SESSION_STARTED`, `SESSION_RESUMED`, `SESSION_ENDED`, `AGENT_WORKING`, `AGENT_WAITING` | runtime ID by default |
| Tool/approval | `TOOL_STARTED`, `TOOL_FINISHED`, `APPROVAL_REQUIRED`, `APPROVED`, `REJECTED` | runtime ID by default |
| Quota/failover | `RATE_LIMITED`, `QUOTA_LOW`, `QUOTA_EXHAUSTED`, `QUOTA_FAILOVER_*`, `QUOTA_MONITOR_*` | MissionRun ID for automated failover; otherwise runtime/empty |
| Handoff | `HANDOFF_COMPLETED` | handoff lineage ID |
| MissionRun | `MISSION_STARTED`, `MISSION_STEP_STARTED`, `MISSION_STEP_COMPLETED`, `MISSION_PAUSED`, `MISSION_RESUMED`, `MISSION_CANCELED`, `MISSION_COMPLETED`, `MISSION_FAILED` | MissionRun ID; runtime ID when the current package has one |
| Error | `ERROR` | operation-specific when available |

`correlation_id` links events belonging to a broader operation without
changing the event ID or the runtime subscription key. MissionRun lifecycle
and failover use the run ID. Both account and context handoff use the lineage ID. The
durable projection stores the field through migration `0015_event_correlation`;
legacy rows remain valid with an empty value.

## Timeline rules

1. Every event keeps its concrete `runtime_id` when one exists.
2. `correlation_id` is optional and must never be fabricated from an unrelated
   provider, profile, or project ID.
3. MissionRun events carry the run's `project_id` and current package's
   `agent_id` when those values exist, allowing the durable recorder to project
   them into project timelines.
4. A missing correlation is represented as empty/absent, not as a synthetic
   timeline.
5. New event types must be added to `internal/control/events/events.go`,
   documented in this table, and covered by a semantic test before a producer
   is introduced.
6. Provider-native logs and `internal/core/telemetry` remain separate legacy
   diagnostics; they are not silently reclassified as runtime events.

7. SessionHost emits both process facts and runtime lifecycle facts. The former
   describes the operating-system child; the latter describes the Nexus-owned
   runtime state transition. Both preserve `project_id`/`agent_id` only when
   supplied by the launcher and never infer ownership from an ID.

## Intentionally deferred

The current scope does not add a new event broker, rename the uppercase public
values, or invent synthetic `runtime.*` events where the existing process and
session events already provide evidence. A future timeline UI can query the
durable projection by project, agent, and correlation ID without changing the
live bus contract.
