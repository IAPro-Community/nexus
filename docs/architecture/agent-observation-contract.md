# Agent observation contract (Herdr-inspired, Nexus-native)

Status: **experimental, read-only**. These endpoints project the current persistent
Agent and the already-owned RuntimeGeneration/SessionHost registry. They do not start
a second supervisor, control Maestro, or implement the Attention Center.

## Endpoints

- `GET /api/v1/agents/{agent_id}/explain?project_id={project_id}`
- `GET /api/v1/agents/{agent_id}/wait?project_id={project_id}&until=blocked&wait_ms=5000&after_revision=3&after_generation={generation_id}`

Requests use the existing Nexus session authentication and project-scoped Agent
ownership checks. Both endpoints are **GET**, return `Cache-Control: no-store`,
and disclose redacted operational metadata only; they never include transcript
text, credential scopes, prompt summaries or provider environment variables.
Mutating requests are not supported.

### Observation fields

| Field | Meaning |
| --- | --- |
| `agent_id`, `project_id` | Persistent Agent identity and required scope |
| `generation_id`, `runtime_id` | Current Nexus generation and supervised runtime, when known |
| `process_state` | `UNKNOWN`, `LIVE` or `NOT_LIVE` based on actual SessionHost liveness |
| `agent_state` | `unknown`, `working`, `blocked`, `idle`, `done` or `error` |
| `signal_source` | Event provenance `AUTHORITATIVE`, `SIGNALLED`, `INFERRED` or `UNKNOWN` |
| `reason_code` | A machine-readable diagnostic explanation |
| `attention_revision` | Monotonically changing registry attention revision **within one generation** |

**A capability is not evidence.** A driver that can produce structured events
does not prove that this specific session emitted one. `signal_source` is read
from the recorded event provenance, not inferred from the driver's control level.

**An idle or done conversation is not a verified task.** `done` means the
provider/terminal's observed *turn* was settled. It must never be mapped to
`COMPLETED_VERIFIED`, Mission success, approval granted, or output validity.

## Bounded waiting and concurrency

`wait` accepts `until=blocked|working|idle|done|error` and
`wait_ms=0..15000`. A response includes:

- `matched`: an observed state and revision satisfied the predicate on a
  **live** current generation.
- `timed_out`: the requested observation did not arrive before the bounded deadline.
- `generation_changed`: the supplied `after_generation` no longer matches
  the Agent's current generation; restart observation with a new cursor.
- `observation`: the latest redacted observation, including on unsuccessful waits.

Clients should first call `explain` and supply **both** `after_revision` and
`after_generation` on a subsequent `wait` call. Revision numbers alone are
insufficient after a process restart or runtime replacement. Abort a pending
request when an Agent panel is closed or a project changes. A `NOT_LIVE` host
terminates waiting without claiming that the Agent finished its task.

Do **not** wire this endpoint into automatic prompt submission, approval,
Goal Loop advance, or Mission state transitions. Those require independently
verified owner capabilities and evidence contracts.

## Verification and remaining gates

The new handlers have regression tests for project isolation, redaction,
read-only methods, generation-aware waiting, and non-verification of `done`.
The Director UI projects the same registry signals with optional filters.

This does **not** certify native event detection for Codex, OpenCode V2, AGY,
Pi or OMP. Adapter-specific protocol fixtures and real Windows/Linux/macOS
terminal/session smoke tests are still required, especially for issue #196.

Out of scope for this work: Attention Center changes (step 3), SSH multi-host
and plugins (step 7), or a second runtime/scheduler.
