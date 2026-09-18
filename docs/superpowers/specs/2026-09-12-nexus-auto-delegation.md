# Nexus automatic delegation

## Scope

The current interactive/lead Agent remains the owner of goal continuity and
global verification. A deterministic delegation policy may project a compound
goal into bounded Flow steps, each carrying `TaskRequirements` and ownership.

## Contract

- `AUTO` is the default and delegates only when at least two recognized,
  specialized workstreams have compatible ownership.
- `ASK` persists the same proposal with `pending_approval=true`; Mission
  dispatch is rejected until `ApproveDelegation` records a WorkPlan revision.
- `OFF` always keeps a single lead WorkUnit.
- Workstreams express roles/capabilities/domains/paths only. `MatchAgents`
  selects persistent Agents, and the existing resource scheduler selects the
  provider/profile/account/model afterward.
- Backend and frontend can run in the implementation parallel group; QA is
  dependent on both. Collision phrases such as same file/shared contract
  activate the dispatch brake.
- WorkPlan structured facts persist the decision. MissionRun payload persists
  `LeadAgentID`, package assignment and routing decisions through existing
  execution snapshots and repositories.

## Non-goals

No second Agent system, scheduler, Composer prerequisite, Flow prerequisite or
anonymous worker pool is introduced. Provider-specific identity is not encoded
in AgentSpec.

## Verification

Focused tests cover anti-overdelegation, modes, collisions, deterministic
workstream requirements, rich auto-created AgentSpec, and lead/delegation fact
round trips. The supervised `nexus <provider>` entrypoint now resolves a
persistent Lead by the current workspace, routes complete prompts through the
same policy, and returns Mission completion to that Lead for synthesis. The
full Nexus suite and repository gates remain required before promotion; real
authenticated provider E2E is an external evidence gate.
