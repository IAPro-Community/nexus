# Mission Teams

Mission Teams are a data-driven projection over Nexus's existing execution
contracts. They do not replace Agents, WorkPlans, MissionRunner,
ResourceScheduler, MatchAgents, ContextCapsule, WorkReceipt or
ValidationEvidenceStream.

## Ownership

The foreground persistent Agent remains the Lead. A team phase requests a
professional role (`architect`, `implementer`, `reviewer`, `qa-engineer`, and
specialized variants). `MatchAgents` selects the persistent Agent identity;
the existing resource scheduler then selects provider/profile/model according
to the phase affinity policy. Desired and actual resources are recorded
separately, so fallback does not mutate the Agent or the desired preference.

## Presets

The initial presets are `FEATURE_DELIVERY`, `BUG_FIX`, `REFACTOR`,
`ARCHITECTURE_CHANGE` and `RELEASE_CANDIDATE`. They are serialized in existing
WorkPlan `structured_facts`; phase type, role, dependencies, evidence and
handoff metadata are serialized in existing WorkPlan phase/package JSON.

Typical Feature Delivery is:

```text
Planning (architect)
  -> Implementation (implementer; backend/frontend may fan out)
  -> Review (reviewer; read-only)
  -> Verification (qa-engineer/verifier)
```

The preset is policy, not a provider mapping. No phase requires Codex, AGY or
Claude unless a user/project policy explicitly sets a `PREFER` or `PIN`
affinity through the existing routing contract.

The decomposition contract accepts an optional `mission_team_preset` and
phase-keyed `mission_team_runtime_overrides` map. An explicit request override
wins over the preset, while `PREFER` remains fallible and `PIN` remains
fail-closed. The selected Agent identity is never replaced when the scheduler
falls back to another configured resource. A `DelegationMode=OFF` request is
projected as a Lead/direct plan and cannot create a specialized team behind the
user's back.

## Durable handoff

Every generated package carries bounded JSON handoff data with the mission and
phase objective, prior result, decisions, artifacts, evidence, failures,
rejected approaches, risks, acceptance criteria and next action. The runner
copies this into `ContextCapsule` and `WorkReceipt` before/after dispatch.

## Lifecycle

Package state remains the execution authority. `MissionRun.PhaseRuns` is a
durable projection grouped by phase, used by CLI/Web read models. Review and
verification phases do not execute a write-capable provider turn: review
consumes implementation dependencies and verification executes canonical
commands.

## Selection

Intent classification remains the only direct-vs-plan router. Atomic typo,
rename and small isolated fixes stay DIRECT. Substantial implementation,
production-readiness, architecture, security and release signals select a
preset. The selected preset and reason are persisted for routing reports.
