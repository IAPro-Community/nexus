# IAPro Nexus — Workspace OS + Composer + Flow Run Design

**Status:** Design approved for implementation planning  
**Repository:** `kivervinicius/ai-cli`  
**Target branch:** `feat/nexus-maximum-delivery`  
**Baseline commit at freeze:** `c96885218b3cff319f481aed4423dc5fd7b52a30`  
**Visual reference:** `docs/superpowers/references/iapro-nexus-final-base.html`

---

## 1. Purpose

This document freezes the target product architecture for the next IAPro Nexus refactor.

The refactor MUST preserve proven Core behavior and MUST NOT reinterpret the approved HTML as permission to rebuild the application from scratch.

The implementation strategy is:

> preserve the Core → repair production blockers → adapt existing surfaces and domains → add only the missing contracts.

The final product is an **Agentic Product Development Workspace**, not a generic automation platform, Git client, full IDE, or single-provider coding agent.

---

# 2. Product North Star

IAPro Nexus is a **local-first, project-centric, provider-agnostic development workspace** for building software with multiple coding Agents.

A user can:

1. work directly with persistent AI Agents;
2. open normal project shells;
3. organize those views as tabs, splits, or desktop windows;
4. use Maestro-managed project knowledge and skills;
5. open Composer to convert an intention into an approved Flow;
6. execute the approved Flow through dependency-aware Agents;
7. let Nexus continue automatically through safe, bounded, verifiable steps;
8. intervene through pause, terminal inspection, or Take Control;
9. receive a verified final result rather than an unproven “done”.

In short:

```text
Workspace OS
+
Persistent Agents
+
Project Shells
+
Composer
+
Static DAG Flow
+
Dependency-aware Flow Runner
+
Fresh Task Context
+
Work Receipts
+
Verification
+
Maestro
```

---

# 3. Product invariants

These rules are architectural contracts.

## 3.1 Project is the center

Opening a Project opens its Workspace.

The primary experience is NOT:

```text
Project → Overview → Agents → Work → Terminal
```

It is:

```text
Project
  ↓
Workspace
  ├ Agent Views
  ├ Shell Views
  ├ Composer Views
  └ Flow Run Views
```

Administrative surfaces such as Agents, Resources, Maestro and Settings remain available, but the Workspace is where work happens.

## 3.2 Agent is a persistent identity

```text
Agent != Runtime
Agent != Session
Agent != Terminal View
```

A stable Agent may have different runtime generations and task sessions over time.

Closing a View MUST NOT implicitly stop an Agent.

## 3.3 View is presentation

A logical View may be rendered as:

- tab;
- split pane;
- desktop window.

Changing presentation MUST NOT create another Agent or another Flow Run.

## 3.4 Direct work is first-class

The user MUST be able to:

```text
Project → Existing Agent → Open / Ask
```

without Composer, Flow, Mission or Maestro orchestration.

Direct work MUST still function when Composer is unavailable.

Maestro project hydration MUST NOT block direct Agent or normal Shell use.

## 3.5 Normal Shell is first-class

A Project can open one or more normal shells.

A Shell:

- is not an Agent;
- does not require AI;
- starts at a Project working directory;
- can run normal commands such as tests, build tools, Git, Docker and Kubernetes;
- participates in tabs/splits/windows like every other Workspace Surface.

## 3.6 Git is context, not product

V0 displays only useful context such as:

- current branch;
- optional clean/modified indicator;
- Agent worktree branch when relevant.

V0 MUST NOT grow into a full Git client.

## 3.7 Maestro remains independent

Maestro is an independent source of:

- skills;
- rules;
- methodology;
- durable project memory conventions.

Nexus may:

- detect;
- install;
- update;
- verify;
- index;
- browse;
- suggest;
- request hydration/refresh.

Nexus MUST NOT:

- emulate Maestro;
- invent Maestro skills;
- create a competing skill store;
- create competing semantic project memory;
- make Maestro depend on Nexus;
- intercept every terminal command as a Maestro instruction.

## 3.8 Composer is the intelligent editor

```text
Composer = editor/environment
Flow = artifact
Flow Run = execution
```

There MUST NOT be separate competing Composer Editor and Flow Editor products.

## 3.9 Flow execution is bounded

V0 Flow is a static approved DAG.

Runtime Agents MUST NOT mutate the approved graph silently.

If graph structure needs to change:

```text
Pause → Composer revision → Review → Approve new revision
```

## 3.10 Verification defines completion

A Flow is not finished because an Agent said it is finished.

Completion requires explicit verification evidence.

---

# 4. Visual and UX baseline

The existing Nexus visual language remains the product baseline.

Preserve:

- Project Rail;
- Topbar;
- Workspace chrome;
- dark elevated surfaces;
- existing design-system primitives;
- current purple Nexus accent;
- terminal visual conventions;
- Taskbar/status semantics.

The approved visual reference adds functionality, not a new design system.

The reference intentionally demonstrates:

- desktop-like Workspace;
- multiple movable windows;
- terminal windows;
- Composer window;
- Flow Run window;
- Project Toolbox;
- `+ Novo` launcher;
- Tabs/Janelas presentation switch.

Any divergence from the approved reference during implementation MUST be intentional and documented.

---

# 5. Workspace OS

## 5.1 Existing foundation to preserve

Current Workspace V2 already models:

```text
WorkspaceModel
WorkspaceNode
WorkspaceStack
WorkspaceSplit
WorkspaceSurface
viewId
logicalKey
focusedStackId
maximizedViewId
```

It already supports important semantics including:

- stable view identity;
- focus existing View rather than duplicate it;
- tab stacks;
- splits;
- moving surfaces;
- maximizing;
- closing Views independently.

This foundation MUST be reused.

## 5.2 New presentation mode

Add a presentation concept rather than replacing Workspace V2:

```ts
type WorkspacePresentationMode = 'TABS' | 'DESKTOP'
```

Conceptually:

```ts
interface DesktopWindowState {
  viewId: string
  x: number
  y: number
  width: number
  height: number
  zIndex: number
  minimized: boolean
  maximized: boolean
}
```

Persistence SHOULD be per Project Workspace.

The same `viewId` and `logicalKey` are shared between Tabs and Desktop presentations.

## 5.3 V0 desktop window features

Required:

- move;
- resize;
- focus / z-order;
- minimize;
- maximize;
- close View;
- simple bounded positioning;
- persistence.

Not required V0:

- arbitrary virtual desktops;
- OS-level window integrations;
- complex snap zones;
- freeform canvas objects unrelated to Workspace Surfaces;
- desktop widgets.

## 5.4 Launcher

The Workspace exposes a concise `+ Novo` launcher:

```text
Agent
Terminal
Composer
Active Flow Run
```

Do not turn it into a generic application marketplace.

---

# 6. Agent Fleet

## 6.1 Existing capabilities to preserve

Preserve:

- AgentID;
- persistent Agent records;
- Agent lifecycle;
- runtime generations;
- reconnect;
- continuity;
- Safe Apply;
- terminal broker;
- CONTROL / VIEW_ONLY;
- provider/profile/model configuration;
- isolation;
- worktrees.

## 6.2 Open existing Agent

`Open Agent` MUST:

1. find the Agent by AgentID;
2. focus an existing logical View if already open;
3. otherwise open a new View attached to that Agent;
4. start/recover the Agent only when required by state;
5. never create a second Agent as a side effect.

## 6.3 Ask Agent

V0 adds explicit work submission to an existing Agent.

UX:

```text
Target Agent
[ Codex Backend ]

Instruction
[ ... ]

[ Send ]
```

Semantics:

```text
existing AgentID
→ active/recovered runtime
→ explicit user instruction
```

`Ask` MUST NOT implicitly call `createAgent`.

If provider capabilities differ, Nexus must use the safest real submission path and remain capability-truthful.

## 6.4 Start & Ask

If Agent is stopped:

```text
Start
→ establish valid runtime/session
→ submit explicit instruction
```

Errors remain visible.

## 6.5 New Agent

`+ Agent` is used when a new specialist is actually needed.

Configuration includes:

- name;
- role;
- resource selection;
- provider/profile/model where applicable;
- execution mode;
- isolation.

Default resource selection:

```text
Auto / Recommended
```

Manual override remains available.

## 6.6 Execution modes

Expose normalized semantic modes:

```text
Safe
Plan
YOLO
Custom
```

Provider drivers determine which modes are valid.

Nexus MUST NOT fabricate provider support.

Agent Execution Mode and Flow Autonomy Policy are different axes.

---

# 7. Resource Scheduling

Resource Scheduler is Core and shared by all work entry points.

Consumers:

```text
Direct Agent creation
Direct Agent recovery
Flow Step execution
Specialist Agent creation
```

Default:

```text
AUTO / RECOMMENDED
```

Manual override:

```text
MANUAL
```

The current Direct Session path MUST be corrected so it no longer treats first eligible/manual selection as the implicit default when the recommendation engine is available.

Recommendation may consider:

- health;
- authentication;
- quota;
- cooldown/rate limit;
- provider capability;
- continuity;
- configured policy.

---

# 8. Project Shell

Add a dedicated Shell Surface.

Concept:

```ts
interface ProjectShellSurfaceData {
  projectId: string
  cwd: string
  shellProfile?: string
}
```

V0:

- starts at Project canonical path;
- supports multiple shells;
- supports Workspace tabs/splits/windows;
- preserves terminal ANSI;
- survives View layout changes;
- may be named (`Shell · dev`, `Shell · tests`, etc.).

Worktree-specific Shell is useful but may be delayed until the canonical Project Shell is production-grade.

---

# 9. Maestro Integration

## 9.1 Nexus responsibility

The Maestro surface becomes an integration manager for:

```text
Installed
Version
Health
Update
Verify
Project context readiness
Skills catalog
```

## 9.2 Project memory

Canonical memory remains:

```text
nearest AGENTS.md
DEV/README.md
DEV/INDEX.md
DEV/CONTEXT.md
DEV/WORKLOG.md
DEV/ARCHITECTURE.md
DEV/DECISIONS.md
DEV/ADR/
DEV/API/
DEV/DATABASE/
DEV/TESTING.md
DEV/RUNBOOKS/
DEV/TASKS/
DEV/ROADMAP.md
DEV/RESEARCH/
DEV/HANDOFFS/
```

Only task-relevant documents should be loaded into a task context.

## 9.3 Semantic hydration

Repository semantic hydration belongs to Maestro itself.

Required Maestro behavior:

1. inspect existing repository;
2. preserve existing project knowledge;
3. initialize missing canonical DEV structure;
4. infer useful architecture/testing/context facts from real source;
5. write/update durable project knowledge;
6. validate memory against repository state;
7. support a refresh contract.

Nexus consumes this behavior; it does not reimplement it.

---

# 10. Context Readiness

Composer requires trustworthy project context.

V0 states:

```text
MISSING
HYDRATING
READY
STALE
FAILED
```

Potential future extension:

```text
REFRESHING
CONFLICTED
```

Context readiness metadata includes enough repository identity to detect drift, conceptually:

```text
project
branch
HEAD
working tree fingerprint
hydratedAt
Maestro version
```

This metadata is not a second semantic knowledge store.

## 10.1 Hard Gate

```text
Context != READY
→ Composer planning/refinement disabled
```

Direct Agents and Project Shell remain usable.

## 10.2 Branch-awareness

If repository branch/revision changes materially after hydration, Composer context becomes stale or is incrementally refreshed.

Branch is contextual information in the V0 UI; it is not a Git-management surface.

---

# 11. Composer

## 11.1 Migration strategy

Current `WorkSurface` evolves into Composer.

Do not create a completely separate planning product while leaving `WorkSurface` and `PlanBuilderSurface` untouched.

Target:

```text
WorkSurface
→ ComposerSurface
```

Current `direct / assisted / planned` product language is deprecated.

Primary user choices become:

```text
Direct Work
or
Composer
```

## 11.2 Existing Plan Builder reuse

Current planning infrastructure already includes major reusable pieces:

- AI plan generation;
- blocking clarification;
- WorkPlan persistence;
- plan revisions;
- revision diff;
- restore;
- package editing;
- drag/drop;
- split;
- merge;
- dependencies;
- parallel groups;
- Agent allocation;
- provider/profile locks;
- resource discovery;
- prompt compilation;
- runs;
- autonomy;
- scheduling;
- Take Control / run operations.

These features MUST be mapped and reused rather than rebuilt.

## 11.3 Intelligence Provider

Composer supports:

```text
CLI
API
```

### CLI

CLI can work in the local Project workspace.

Composer still supplies scoped intent and relevant context.

### API

Remote APIs do not see the local repository.

Nexus builds a bounded Context Envelope.

Conceptual payload:

```json
{
  "project": {
    "id": "...",
    "branch": "...",
    "head": "...",
    "context_state": "READY"
  },
  "memory": [
    "task-relevant durable context"
  ],
  "composer": {
    "flow_revision": 4,
    "scope": "STEP",
    "selected_step": "...",
    "operation": "REFINE"
  },
  "retrieval": {
    "relevant_paths": [],
    "approved_decisions": []
  }
}
```

Never bulk-upload the entire repository by default.

## 11.4 Simple requests

Composer does not force Flow creation.

Example result actions:

```text
Send to existing Agent
Send to new Agent
Copy prompt
Turn into Flow
```

---

# 12. Flow Domain

## 12.1 User-facing model

Target vocabulary:

```text
Flow Definition
Flow Step
Flow Revision
Flow Run
```

Existing internal vocabulary may remain during migration:

```text
WorkPlan
WorkPackage
PlanRevision
MissionRun
```

## 12.2 Compatibility façade

Prefer adapters/facades over destructive data migrations in the first implementation.

Conceptually:

```text
WorkPlan      ↔ FlowDefinition
WorkPackage   ↔ FlowStep
MissionRun    ↔ FlowRun
```

Existing persistence and runner behavior should remain usable until a later safe migration is justified.

## 12.3 V0 DAG

Supported edge semantics:

```text
SEQUENCE
DEPENDENCY
PARALLEL
```

The Flow is acyclic.

V0 intentionally excludes:

- arbitrary conditions;
- generic scripting nodes;
- event triggers;
- webhooks;
- cron;
- infinite/dynamic loops;
- runtime graph mutation;
- generic automation marketplace behavior.

The Flow exists specifically for product/software development.

---

# 13. Flow Editor inside Composer

The Flow canvas is a graphical view of real plan contracts.

Every Step may expose:

```text
Goal
Description
Dependencies
Parallel group
Agent assignment
Resource/provider lock
Execution mode
Isolation
Relevant skills
Acceptance criteria
Verification
Estimated effort
Outputs
```

## 13.1 Localized refinement

Operations:

```text
Discuss
Refine
Expand
Split
Compare
Remove
```

A refinement request receives:

- Project Context Capsule;
- current Flow revision;
- selected Step;
- dependencies;
- dependents;
- approved decisions;
- relevant Maestro skills;
- user instruction.

It MUST NOT regenerate unrelated Flow portions without explicit user approval.

## 13.2 Agent assignment strategy

Every Step supports:

```text
EXISTING
CREATE
AUTO
```

### EXISTING

Reuse Project Agent.

### CREATE

Create a specialist persistent or task-scoped Agent according to the approved implementation contract.

### AUTO

Nexus chooses a suitable executor/resource.

Prefer existing Project Agents when capability, policy and concurrency permit.

---

# 14. Context Capsule

A Flow task SHOULD default to a fresh scoped context.

Agent identity is persistent, but Flow task context does not need to reuse a long interactive transcript.

Conceptually:

```text
Agent
├ Main Interactive Session
├ Flow Task Session A
└ Flow Task Session B
```

V0 UX can expose only:

```text
Context
● Fresh task context
○ Continue current context
```

with Fresh as the preferred default for Flow Steps.

## 14.1 Capsule content

A Context Capsule contains:

```text
Project identity
Relevant durable project memory
Relevant source retrieval
Approved Flow decisions
Current Step definition
Dependency Work Receipts
Relevant Maestro skills
Acceptance criteria
Execution policy
Explicit constraints
```

It intentionally excludes unrelated transcripts and unrelated source.

---

# 15. Work Receipt

Work Receipt is V0 because it is the primary clean handoff between Flow Steps.

Conceptual model:

```ts
interface WorkReceipt {
  stepId: string
  status: string
  summary: string

  changedFiles: string[]
  commands: string[]
  tests: VerificationResult[]

  decisions: string[]
  artifacts: ArtifactRef[]
  remainingIssues: string[]

  verification: VerificationResult[]

  agentId?: string
  sessionId?: string

  baseRevision?: string
  resultRevision?: string

  startedAt: string
  completedAt: string
}
```

Exact schema is subject to existing backend-domain review.

A dependent Step receives relevant Receipts instead of raw upstream Agent transcripts.

---

# 16. Flow Run

## 16.1 Approval boundary

DRAFT performs no implementation execution.

Execution begins only after explicit review and approval.

Review includes:

```text
Flow revision
Steps
Dependencies
Parallel groups
Agent assignment
Resource/provider locks
Execution modes
Isolation/worktrees
Context revision
Maestro skills
Compiled prompts
Acceptance criteria
Autonomy policy
```

## 16.2 Static DAG execution

After approval the Flow revision is frozen for that run.

Example:

```text
A
↓
B || C
↓
D
↓
E
```

Runner may compute:

```text
Wave 1: A
Wave 2: B, C
Wave 3: D
Wave 4: E
```

Only Steps whose dependencies are satisfied become READY.

## 16.3 Step lifecycle

Target normalized user-facing lifecycle:

```text
QUEUED
READY
RUNNING
VERIFYING
COMPLETED
BLOCKED
FAILED
CANCELED
```

Internal Mission states may be mapped through a compatibility layer.

## 16.4 Runtime graph mutation

V0 Agents cannot silently change the approved DAG.

Graph-changing discoveries require controlled replanning.

---

# 17. Flow Autonomy Policy

Flow Autonomy is separate from provider execution permission.

V0:

```text
GUIDED
AUTONOMOUS
```

## 17.1 Guided

Pause/request user intervention for cases such as:

- destructive operation outside approved policy;
- unresolved conflict;
- missing critical fact;
- verification failure that cannot be safely remediated;
- scope expansion;
- security boundary.

## 17.2 Autonomous

Nexus may continue through the approved Flow without routine user confirmation while still obeying:

- provider execution capability;
- security;
- approved DAG;
- resource policy;
- verification gates;
- budget/attempt policies;
- scope.

---

# 18. Verification

Every autonomous segment requires a verifiable stop condition.

Use the principle:

```text
GOAL
→ VERIFY
→ STOP
```

Verification may be:

- tests;
- typecheck;
- lint;
- build;
- browser E2E;
- runtime assertion;
- artifact validation;
- reviewer Agent;
- other explicit acceptance evidence.

Not every tiny Step requires an independent reviewer.

Composer SHOULD recommend milestone verification where parallel lanes converge and a final verification gate at completion.

If completion cannot be proven:

```text
FAILED_VERIFICATION
or
BLOCKED_NEEDS_USER
```

not a false success.

---

# 19. Flow Run Surface

Flow Run is a Workspace Surface.

It shows:

- current Flow revision;
- autonomy policy;
- Step;
- dependencies;
- status;
- assigned Agent;
- context mode;
- duration;
- verification;
- Receipt/result.

It also provides:

```text
Open Agent
Watch Terminal
Pause
Take Control
Resume / Return to Automation
Cancel
```

Where supported by existing runner contracts.

Live terminal previews are supplemental; the canonical terminal remains an Agent View.

---

# 20. Mission Runner reuse

Current Mission/Plan runtime infrastructure MUST be audited before replacement.

Expected reusable areas include:

- durable runs;
- scheduling;
- leases/fencing;
- pause/resume;
- Take Control;
- verification;
- remediation;
- recovery;
- restart persistence;
- autonomy contracts.

User-facing Mission terminology is deprecated, but internal Mission structures may remain during compatibility migration.

Do not rewrite durable execution solely to rename it.

---

# 21. Current production blockers

These are independent of the new UX and must be fixed before production V0.

Known blockers at design freeze include Windows/runtime issues such as:

- SessionHost named-pipe failures;
- standalone runtime handshake timeout;
- ConPTY interactive/echo failures;
- cleanup with `nexus.db` held open;
- filesystem mkdir contract mismatch.

Also harden:

- browser ANSI E2E;
- protocol/control frames not leaking into terminal output;
- terminal lifecycle/reconnect;
- direct Resource Scheduler integration;
- provider/config secret sharing policy;
- shared-workspace writer semantics.

No new visual surface may be used as evidence that these Core defects are solved.

---

# 22. Security

Security remains deny-by-default.

Important design constraints:

- do not broadly expose unrelated host configuration;
- provider secrets require explicit allowlisting;
- Shell and Agent terminal execution remain explicit user/runtime contexts;
- Flow Autonomous policy does not bypass provider/security boundaries;
- destructive actions require explicit contracts;
- API Context Envelope is bounded and traceable.

---

# 23. V0 scope

V0 proves the complete product thesis with a controlled scope.

Required:

### Workspace OS
- Project opens into Workspace;
- Tabs;
- splits;
- Desktop/Janelas presentation;
- persistent View layout;
- launcher.

### Direct Work
- persistent Agents;
- Open existing Agent;
- Ask existing Agent;
- Start & Ask;
- new Agent;
- normalized execution modes;
- Project Shell;
- Resource Auto/Recommended.

### Maestro
- detect;
- install;
- update;
- verify;
- skills catalog;
- project context hydration/readiness.

### Composer
- Context READY gate;
- CLI/API intelligence;
- Context Envelope;
- clarification;
- simple compiled prompt;
- convert complex work to Flow.

### Flow
- static DAG;
- sequence;
- dependencies;
- parallel groups;
- Step inspector;
- localized refinement;
- EXISTING/CREATE/AUTO assignment;
- relevant skills;
- acceptance criteria.

### Flow Run
- explicit approval;
- dependency-aware execution;
- safe parallel waves;
- fresh Context Capsule;
- Work Receipts;
- verification gates;
- Guided/Autonomous;
- pause/Take Control where supported;
- completion evidence.

---

# 24. V1 candidates

V1 should polish the validated model rather than introduce a different product.

Candidates:

- `@Agent` mentions;
- Agent work queue;
- richer Project Toolbox;
- default skills/expertise per Agent;
- incremental semantic hydration;
- richer Work Receipt visualization;
- Flow revision visual diff;
- evidence screenshots/videos for browser tasks;
- worktree-specific normal Shell;
- better Flow Run analytics;
- desktop snap/layout templates.

---

# 25. Explicitly out of scope

Do not add to V0 unless a separate design decision changes scope:

- full IDE/code editor clone;
- full Git client;
- PR management;
- Git graph;
- generic n8n-like workflow automation;
- cron automation;
- webhook triggers;
- event-driven arbitrary workflows;
- dynamic DAG mutation by Agents;
- general scripting-node engine;
- infinite agent loops;
- MCP marketplace owned by Nexus;
- Cloud Agents;
- remote runtime fleet;
- distributed multi-machine scheduler;
- multi-user collaboration;
- arbitrary desktop widgets;
- full operating-system window manager;
- second skill ecosystem;
- second semantic memory tree.

---

# 26. Refactoring classification

## KEEP

- Project model;
- Project Rail;
- Topbar;
- Taskbar;
- Workspace V2;
- WorkspaceSurface/view identity;
- stacks/splits;
- persistent AgentID;
- RuntimeGeneration;
- terminal broker;
- xterm;
- CONTROL/VIEW_ONLY;
- Resources/quota/health;
- scheduler;
- Safe Apply;
- continuity;
- checkpoints/handoff;
- worktrees;
- current WorkPlan persistence;
- plan revisions;
- clarification;
- package split/merge;
- dependency and parallel-group data;
- Mission durable runner;
- existing verification/recovery primitives.

## FIX

- Windows runtime/ConPTY/named pipes;
- terminal protocol drift;
- ANSI E2E;
- cleanup/lifecycle defects;
- filesystem contract defect;
- Direct resource recommendation;
- provider config security;
- shared writer semantics.

## ADAPT

- Project opening behavior → Workspace first;
- Workspace V2 → Tabs + Desktop presentation;
- Agents surface → operational Agent Fleet;
- WorkSurface → Composer;
- PlanBuilder → Flow capabilities inside Composer;
- WorkPlan → Flow Definition façade;
- WorkPackage → Flow Step façade;
- MissionRun → Flow Run façade;
- Mission Autonomy → Guided/Autonomous UX;
- Maestro Assist UX → Maestro Integration Manager;
- existing Intelligence → Composer Intelligence infrastructure.

## CREATE

- Desktop window state/presentation;
- Workspace launcher;
- Project Shell Surface;
- Ask existing Agent;
- Context Readiness;
- Context Envelope;
- Context Capsule;
- Flow graph visualization;
- Flow Step inspector/refinement;
- Flow execution timeline/surface;
- Work Receipt;
- missing dependency-wave scheduler behavior;
- clean Flow-to-task session boundary.

## DEPRECATE FROM USER LANGUAGE

- Mission as the main product noun;
- Direct/Assisted/Planned three-mode product model;
- Nexus Intelligence as a separate user-facing brain;
- Ask Maestro as a generic workflow;
- Maestro OFF/ASSIST/ORCHESTRATE as primary product modes.

---

# 27. Recommended implementation order

## Phase 0 — Freeze
- commit this specification;
- commit approved visual reference;
- verify baseline SHA and source contracts.

## Phase 1 — Core Integrity
- Windows;
- terminal protocol;
- ANSI;
- lifecycle;
- security;
- Resource Scheduler;
- Agent/View contracts.

**Gate:** same-SHA Linux/macOS/Windows/browser Core evidence.

## Phase 2 — Workspace OS
- Project opens Workspace;
- Tabs/Desktop;
- window persistence;
- launcher;
- Agent open/focus;
- Ask Agent;
- Project Shell;
- execution-mode UX.

**Gate:** complete multi-Agent workspace usable without Composer.

## Phase 3 — Maestro + Context
- correct Maestro detection/install/update/verify;
- skill catalog;
- Maestro semantic hydration contract;
- Nexus Context Readiness.

**Gate:** an existing repository reaches READY from a reproducible hydration path.

## Phase 4 — Composer
- evolve WorkSurface;
- CLI/API;
- Context Envelope;
- clarification;
- simple prompt output;
- no forced Flow.

**Gate:** Composer can reason usefully about a real Project before Flow.

## Phase 5 — Flow Editor
- compatibility façade;
- DAG Canvas;
- Step inspector;
- localized refinement;
- assignment;
- skills;
- acceptance criteria;
- review.

**Gate:** complex project can be designed/reviewed without execution.

## Phase 6 — Flow Run
- audit/reuse Mission Runner;
- static DAG execution;
- safe parallel waves;
- fresh task sessions;
- Context Capsules;
- Work Receipts;
- verification;
- Flow Run Surface.

**Gate:** approved Flow reaches verified completion without routine intervention.

## Phase 7 — Production hardening
- recovery;
- interruption;
- restart;
- failure remediation;
- concurrency;
- Windows;
- browser;
- security;
- installer/release.

**Gate:** production GO only with same-SHA evidence.

---

# 28. End-to-end acceptance scenario

The final implementation must demonstrate:

1. open Nexus;
2. open `ai-cli`;
3. Project opens its Workspace;
4. existing Agent and Shell Views restore;
5. toggle Tabs → Janelas without duplicating Agent identity;
6. move/resize/maximize a window;
7. open `+ Terminal`;
8. execute a normal project command;
9. open an existing Agent;
10. close its View;
11. prove Agent keeps running;
12. reopen same Agent;
13. use Ask Agent;
14. create a new Agent only when explicitly requested;
15. use Auto/Recommended Resource selection;
16. open Composer as a Workspace View;
17. context is READY;
18. describe a complex change;
19. Composer asks blocking clarifications where needed;
20. Flow DAG is produced;
21. edit/refine only one Step;
22. assign existing Agent to one Step;
23. leave another Step AUTO;
24. review dependencies, parallelism, skills and acceptance criteria;
25. Approve & Run;
26. independent ready Steps execute in parallel;
27. dependent Step waits;
28. each completed Step emits Work Receipt;
29. dependent Context Capsule receives relevant Receipts rather than transcripts;
30. milestone verification runs;
31. one failure can enter remediation or user-blocked state without false success;
32. Flow Run resumes/retries according to policy;
33. final verification passes;
34. run reaches verified completion;
35. receipts/evidence remain inspectable.

---

# 29. Success definition

The refactor is successful only if Nexus becomes simpler to understand while becoming more capable.

The user should need to understand only these main concepts:

```text
Project
Workspace
Agent
Terminal
Composer
Flow
Flow Run
Maestro
Resource
```

And the product should prove three core jobs:

```text
1. WORK DIRECTLY WITH AGENTS
2. KEEP DEVELOPMENT IN ONE WORKSPACE
3. COMPOSE AND EXECUTE COMPLEX PRODUCT WORK RELIABLY
```

Any feature that does not materially support these jobs should remain outside V0.

---

# 30. Final implementation principle

The approved visual reference and this specification are constraints, not excuses for a rewrite.

Every implementation task must first classify the affected code as:

```text
KEEP
FIX
ADAPT
CREATE
DEPRECATE
```

If an existing capability already satisfies the target contract, preserve it.

If a feature is only visually missing, do not rebuild its backend.

If the approved UX conflicts with a proven Core invariant, preserve the invariant and adjust the UX.

Evidence must precede production claims.
