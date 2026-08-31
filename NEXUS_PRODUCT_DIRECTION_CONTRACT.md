# IAPro Nexus — Product Direction Contract

## Baseline

- Repository: `kivervinicius/ai-cli`
- Branch: `feat/nexus-maximum-delivery`
- Audited HEAD for this reference: `c96885218b3cff319f481aed4423dc5fd7b52a30`
- Purpose: prevent future Composer/Flow work from rewriting or misrepresenting capabilities that already exist.

## North Star

IAPro Nexus is a project-centric, multi-agent development workspace. It lets a user work directly with multiple AI CLIs/providers or use Composer to transform validated product context into an editable Flow that is only executed after explicit review and approval.

## Product model

```text
Project
├── Direct Agent Workspace
│   ├── Persistent Agents
│   ├── Runtime generations
│   ├── Terminals
│   ├── Resources / quota / scheduler
│   ├── Worktrees
│   └── Context / attention
│
├── Composer
│   ├── Context Readiness Gate
│   ├── Intelligence Provider (CLI or API)
│   ├── Conversation / clarifications
│   ├── Smart Prompt
│   └── Flow Draft
│       ├── Steps
│       ├── Lanes
│       ├── Dependencies
│       ├── Skills references
│       ├── Agent suggestions
│       └── Compiled Prompt Preview
│
└── Flow Run
    └── Existing durable Mission/Runner infrastructure internally
```

## Critical distinction

- **Composer** = intelligent editor / composition environment.
- **Flow** = artifact edited by Composer.
- **Flow Run** = execution of an approved Flow.
- There must NOT be separate primary “Composer editor” and “Flow editor” experiences.
- Flow history/drafts may be browsed from Composer, but opening a draft returns to Composer.
- A running Flow opens a run/progress surface, not another planning editor.

## Existing UI baseline to preserve

The current application shell remains the visual baseline:

- Project Rail on the left.
- Topbar.
- Workspace OS with stacks/tabs/splits/popouts.
- Workspace Taskbar / status bar.
- Existing dark tokens and accent tokens.
- Existing design-system primitives.
- Existing terminal/xterm surface conventions.

Current global Project Rail concepts remain:

1. Workspaces / Projects
2. Overview
3. Agents
4. Resources
5. Maestro
6. Settings

Do **not** add Composer and Flows as two new equivalent global rail products.

Composer opens as a Workspace surface/tab, consistent with the current architecture.

## KEEP — do not rewrite

These are Core Nexus capabilities and should survive the product evolution:

- Projects and project switching.
- Workspace V2 and persistent layouts.
- Persistent `AgentID`.
- Runtime generations.
- Agent Terminal Broker.
- CONTROL / VIEW_ONLY terminal ownership.
- xterm terminal.
- Direct Agent sessions without Mission/Flow/Composer.
- Provider/profile resources.
- Quota, health, rate-limit/cooldown information.
- Resource scheduler.
- Agent configuration + Safe Apply.
- Continuity / native resume.
- Worktrees.
- Context checkpoint / handoff.
- Attention events.
- Project Rail.
- Topbar.
- Taskbar.
- Settings.
- Plan revisions, split/merge, prompt compilation and run durability that already exist under Plan/Mission infrastructure.

## ADAPT — evolve existing code

### Current `WorkSurface` → Composer

Do not create an unrelated second planning subsystem.

Reuse the current Work entry point/surface. The final UX should replace the current conceptual split of Direct/Assisted/Planned with a clearer model:

- Direct Agent work remains available from Agents / direct launcher / command palette.
- Composer is used when the user wants to refine an intention or construct a Flow.
- A simple Composer request may finish as a compiled prompt sent directly to an Agent without creating a Flow.

Internal surface type may remain `work` during migration for compatibility.

### Current `PlanBuilderSurface` → Flow capabilities inside Composer

Reuse:

- AI plan generation.
- Clarification checkpoints.
- WorkPlan persistence.
- Revisions.
- Diff/apply/reject.
- Phase/package editing.
- Drag/drop.
- Split/merge.
- Provider locks.
- Prompt compilation.
- Schedules.
- Autonomy contract.
- Mission run durability.
- Take Control / return to automation.
- Verification/remediation.

These should be reorganized into the Composer/Flow experience, not duplicated.

### Current Mission terminology

User-facing target:

- WorkPlan → Flow Definition
- WorkPackage → Flow Step
- Mission Run → Flow Run

Existing backend/database types may remain during compatibility migration.

Do NOT delete proven Mission Runner infrastructure just to rename the product.

### Current Maestro integration

Target responsibility of Nexus:

- detect Maestro;
- install Maestro;
- update Maestro;
- verify Maestro;
- expose version/health;
- index/browse existing Maestro skills;
- trigger/validate project context hydration.

Nexus must NOT:

- emulate Maestro;
- invent Maestro skills;
- create a second skill catalog;
- make Maestro depend on Nexus;
- silently interpret every terminal command through Maestro.

Maestro remains independently usable by Codex/Claude/Gemini/etc.

## NEW — required capabilities

### Composer Context Readiness Gate

Composer does not operate unless project context is ready.

Suggested states:

- `MISSING`
- `HYDRATING`
- `READY`
- `STALE`
- `REFRESHING`
- `FAILED`

Direct Agent work is never blocked by Composer context state.

### Canonical project memory

Reuse Maestro's project memory conventions:

- `AGENTS.md`
- `DEV/README.md`
- `DEV/INDEX.md`
- `DEV/CONTEXT.md`
- `DEV/WORKLOG.md`
- `DEV/ARCHITECTURE.md`
- `DEV/DECISIONS.md`
- `DEV/TESTING.md`
- relevant `DEV/*` detail files.

Do not create a competing `.nexus/context` knowledge base.

### Semantic hydration

The Maestro ecosystem should gain a canonical repository hydration capability that:

- understands an existing repository;
- initializes missing DEV hierarchy without overwriting existing knowledge;
- validates documentation against current source;
- records repository revision/freshness;
- supports incremental refresh.

This capability belongs to Maestro so it remains usable without Nexus.

### Composer Intelligence Provider

Composer must have one intelligence provider:

- CLI, or
- API.

CLI mode:
- uses the real project workspace and Maestro memory.

API mode:
- Nexus injects a bounded Context Envelope because a remote API cannot see the repository directly.

The Context Envelope must be traceable and task-scoped. It should contain only relevant context rather than blindly uploading the repository.

### Flow step refinement

Every Flow Step must support localized intelligence:

- Discuss
- Refine
- Expand
- Split
- Compare
- Replace
- Remove

A step refinement must receive:

- project context;
- current Flow;
- selected step;
- dependencies;
- approved decisions;
- relevant Maestro skills;
- user instruction.

It must not regenerate unrelated steps without explicit approval.

### Execution modes

Nexus should expose normalized Agent execution modes such as:

- Safe
- Plan
- YOLO
- Custom

Provider drivers translate these semantic modes into provider-specific flags. Do not expose raw dangerous flags as the primary UX.

Execution mode must remain configurable after Agent creation through Safe Apply / continuity impact.

### Resource selection

Direct and Flow-created Agents must use the same Core resource recommendation engine.

Default:
- Auto / Recommended.

Manual provider/profile selection remains possible.

## DEPRECATE FROM PRODUCT LANGUAGE

These may remain internally during migration but should not compete in the final UX:

- “Mission” as a first-class user-facing product concept.
- “Nexus Intelligence” as a separate product area competing with Composer.
- “Ask Maestro” as a generic project-analysis workflow.
- Direct / Assisted / Planned as three equal top-level work modes.
- Maestro `OFF / ASSIST / ORCHESTRATE` presented as if Maestro were a central Nexus-owned AI.

## DO NOT DO

Future refactors must not:

1. Rewrite Nexus from scratch.
2. Add a second Flow editor separate from Composer.
3. Add Composer and Flow as equivalent global Project Rail entries.
4. Force Flow for simple direct work.
5. Force Composer for opening an Agent.
6. Force Maestro for Nexus Core direct Agent work.
7. Create a second skill catalog.
8. Create a second durable project-memory tree that competes with Maestro `DEV/`.
9. Pretend an API knows the local repository without a Context Envelope.
10. Bulk-send the whole repository to a remote API by default.
11. Destroy Mission Runner durability when changing user-facing terminology.
12. Change the current visual language without a separate explicit design decision.
13. Replace Project Rail / Topbar / Workspace stacks / Taskbar merely to accommodate Composer.
14. Tie Agent identity to a single runtime or terminal view.
15. Allow closing a tab to implicitly stop an Agent.
16. Treat shared checkout as equivalent to isolated worktree.
17. Invent unsupported YOLO/permission flags for a provider.
18. Present unverified terminal/ANSI/Windows behavior as production-ready.

## Reference migration

```text
CURRENT                                 TARGET

Project Rail                            Project Rail (preserved)
Overview                                Overview (preserved, actions simplified)
Agents                                  Agents (preserved + execution mode/resource UX)
Resources                               Resources (preserved + unified recommendation)
Maestro                                 Maestro Manager / Integration
WorkSurface                             Composer
PlanBuilderSurface                      Flow capabilities inside Composer
WorkPlan                                Flow Definition (user-facing)
WorkPackage                             Flow Step (user-facing)
MissionRun                              Flow Run (user-facing)
Mission Runner                          Internal durable execution engine
SessionsSurface                         Advanced session/history view
Nexus Intelligence                     Composer intelligence infrastructure
```

## Definition of success

A future implementation is aligned only if all of these remain true:

1. User can open a Project and start multiple independent Agents without Composer, Flow or Mission.
2. Composer cannot propose work until project context is `READY`.
3. Composer can use CLI or API intelligence.
4. API mode injects a bounded Context Envelope.
5. Composer can improve a simple prompt without forcing a Flow.
6. Complex work can become an editable Flow.
7. Flow Steps can be refined independently.
8. User reviews Flow, compiled prompts, skills, Agents and execution policies before running.
9. Existing durable Mission/Plan capabilities are reused under Flow Run.
10. Maestro remains independent and is managed, not reimplemented, by Nexus.
11. Current Nexus visual language and Workspace OS remain recognizable.
12. Core Windows/terminal/security/resource-selection defects are fixed rather than hidden by the redesign.
