# Conversational Mission Promotion and Dynamic Coordination

## Goal

A supervised Agent conversation may remain a direct interaction for atomic work,
but substantial work can be promoted into the existing WorkPlan → Flow →
MissionRunner path. Flow is never a second source of truth: it is the visual
projection of the same persisted Mission execution.

## Non-negotiable invariants

- No orchestrated work exists outside a Mission.
- Agents never spawn providers, Agents, Missions or Flow nodes directly.
- Agents submit typed coordination requests to the Mission.
- MissionRunner validates policy and owns state transitions.
- ResourceScheduler remains the only provider/profile/model allocator.
- WorkPlan/Flow remains the canonical plan projection.
- PackageRun/PhaseRun remains the execution authority.
- ContextCapsule and WorkReceipt remain the bounded handoff/evidence contracts.
- Conversation is an input/control surface, never execution history.
- Dynamic work must remain visible in the owning Project/Mission history.
- Existing memory is queried before waking another Agent for NEED_INFO.
- No free-form peer-to-peer Agent mesh.

## Conversation promotion

A supervised conversation has two paths:

DIRECT:
atomic work remains in the supervised Agent session.

MISSION:
substantial, delegated, asynchronous or autonomous work is promoted through the
existing Conductor. Promotion creates/persists the WorkPlan and Mission before
provider dispatch. The resulting Flow is visible without requiring the user to
open or manually edit Flow.

The promotion boundary must reuse existing intent classification and
Conductor/StartMissionRun contracts rather than adding another scheduler.

## Dynamic coordination

Initial operations are intentionally closed:

- NEED_INFO: resolve from memory/artifacts/receipts first; create work only when
  the answer requires new investigation.
- DELEGATE: propose bounded child work. Mission validates it and ResourceScheduler
  chooses the execution resource. Preferred provider is a preference, not an
  authority bypass.
- ARTIFACT_REQUEST: request an existing or newly produced artifact through
  Mission state.
- REVIEW_REQUEST: request independent review through Mission state.

A request that requires new work becomes a durable child work package linked to
its source package. It is projected into Flow and retained in Mission history
after completion. A request that only reads existing durable context does not
create visual Flow noise.

## Plan-change boundary

Dynamic coordination may refine execution without silently rewriting the sealed
mission objective. If a request changes scope, acceptance criteria, architecture
or user-authorized side effects, it must become a plan revision/intervention
instead of an implicit child task.

## History and observability

Project → Mission history must be able to explain:

- which conversation/prompt promoted the Mission;
- the WorkPlan/Flow revision that was authorized;
- dynamically created child work and why it was created;
- source Agent/package and selected executing Agent/resource;
- dependency/handoff path;
- WorkReceipt, evidence, retries and failures;
- final Definition of Done result.

The Flow UI is optional for operating the product, but not optional as a
projection of persisted orchestration truth.

## Delivery sequence

First establish the typed Mission coordination contract and durability. Then
add MissionRunner request resolution and child-package creation. Then expose
promotion/coordination through supervised conversation. Finally project dynamic
work and history in Flow and certify restart/replay/idempotency behavior.

This work extends the current Mission authority; it must not introduce a second
Mission runner, scheduler, memory, workspace manager or provider identity model.
