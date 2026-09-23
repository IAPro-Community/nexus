# Composer Execution Spine — Nexus 1.0

Status: ACTIVE / P0 product path
Parent: `DEV/SPECS/NEXUS_1_0_CLOSURE.md`

## Problem

Composer, Flow and Mission are individually capable but the user journey still feels
unfinished because the execution decisions are distributed across different moments:

- Composer refines intent and creates a PromptArtifact.
- Flow decomposes the artifact into a WorkPlan/DAG.
- AgentMatcher chooses a persistent specialist only when allocation starts.
- ResourceScheduler chooses provider/profile only when allocation starts.
- model selection happens after account selection.
- Mission freezes and executes the approved plan.

The result is technically valid but product-incomplete: after refining a prompt the user
cannot inspect one coherent answer to "what will Nexus actually do, who will do it, with
which LLM/resource, and why?" before approving execution.

## North-star contract

Composer is the primary entry point for complex work.

It accepts both:

1. an incomplete idea that needs clarification; and
2. an already-complete master prompt that should not be needlessly rewritten or
   interrogated.

The pipeline is:

```text
User objective / complete prompt
        |
        v
Composer discovery + context + memory
        |
        v
PromptArtifact (canonical intent artifact)
        |
        v
Flow decomposition / WorkPlan
        |
        v
Execution Preview (SIDE EFFECT FREE)
        |
        +--> TaskRequirements per WorkPackage
        +--> AgentMatcher: WHO
        +--> ResourceScheduler: provider/profile/account
        +--> model resolver: model when evidence exists
        +--> Skills / Maestro guidance
        +--> isolation policy
        +--> capacity evidence
        +--> budget/cost estimate when available
        +--> verification / evidence gates
        |
        v
Human review / edit / approve
        |
        v
Immutable execution snapshot
        |
        v
Mission Runner
        |
        v
Evidence Bundle / verification
```

## Critical rule: preview is not execution

Execution Preview MUST NOT:

- start an Agent;
- create a runtime;
- mutate Agent configuration;
- allocate an account;
- consume quota intentionally;
- create a worktree;
- start a Mission.

It is an explainable read model over the same canonical matching/routing functions used
at execution time.

Mission admission MUST re-evaluate live constraints immediately before dispatch because
capacity/auth/health may change after preview.

## Prompt completeness

Composer MUST distinguish:

- `NEEDS_DISCOVERY` — blocking decisions are genuinely missing;
- `READY_WITH_ASSUMPTIONS` — safe defaults/assumptions are explicit;
- `READY` — the supplied prompt already provides enough information.

A complete imported prompt is first-class input. Composer should preserve its intent and
only ask a question when the answer materially changes execution or safety.

No "question quota" and no forced rewriting.

## PromptArtifact responsibilities

The PromptArtifact is the durable source of user intent and should preserve:

- original objective/prompt lineage;
- refined canonical prompt;
- structured brief;
- explicit facts;
- assumptions;
- unresolved unknowns and their severity;
- constraints;
- expected artifacts;
- acceptance criteria;
- selected skills;
- relevant context references;
- source revision/hash.

It does NOT own runtime/provider allocation.

## Flow responsibilities

Flow/WorkPlan is the canonical semantic execution plan.

Every WorkPackage should carry, as applicable:

- goal;
- dependencies;
- parallel group;
- TaskRequirements;
- role/specialization need;
- assignment strategy;
- resource policy;
- desired provider/profile affinity when explicit;
- skills;
- relevant paths;
- acceptance criteria;
- verification requirements;
- required evidence;
- handoff contract.

Flow is editable. Graph layout is not execution semantics.

## Execution Preview responsibilities

For every WorkPackage, preview returns:

### Specialist
- assignment strategy;
- selected existing Agent when one matches;
- or "create specialist at execution" when AUTO/CREATE requires it;
- AgentMatch score/confidence/reason;
- rejected candidates summarized.

### Resource
- selected/recommended provider/profile/account;
- scheduler policy;
- recommendation score/confidence/reason;
- capacity freshness/remaining/health when known;
- explicit blocker when no eligible resource exists.

### Model
- selected model only when the existing model inventory supports an evidence-based
  decision;
- fallback reason when a preference cannot be honored;
- UNKNOWN when inventory/evidence is absent.

### Execution contract
- skills;
- isolation intent;
- acceptance/verification gates;
- estimated tokens/cost when the cost subsystem can support it.

The preview is revision-bound: `plan_id + plan_revision`.

## Best LLM does not mean global model ranking

Nexus must not maintain a universal "Claude > Codex > Gemini" list.

"Best" means best eligible execution resource for this WorkPackage under:

1. task requirements;
2. Agent specialization;
3. provider capabilities;
4. explicit user/project affinity;
5. authentication/health/cooldown;
6. real Capacity evidence;
7. model capabilities/inventory;
8. budget/cost policy;
9. continuity/stickiness;
10. fallback/escalation policy.

Unknown evidence is never converted into fake confidence.

## Interaction model

After Composer finalization:

### Direct fit

User may:
- copy prompt;
- send to an Agent;
- run directly when the task does not benefit from orchestration;
- optionally expand into Flow.

### Flow beneficial / strongly recommended

Primary action becomes:

`Review execution plan`

The surface should show a compact plan first:

```text
Goal
Implement authentication hardening

Plan
1 Investigator     inspect auth/session surface
2 Backend          implement server fixes
3 Frontend         update login/session handling
4 QA               execute regression suite
5 Reviewer         independent review

Execution
Investigator  -> existing "Investigator" Agent -> Codex/account A
Backend       -> Senior Developer             -> Claude/account B
Frontend      -> Frontend specialist          -> Codex/account A
QA            -> QA specialist                -> Codex/account C
Reviewer      -> Reviewer                     -> Claude/account B

Capacity / budget / gates
...
```

Technical scores remain available under details; explanations are user-facing.

## Actual routing versus preview

Preview and Mission allocation use the same underlying contracts, but actual allocation
may differ because live state changed.

If it differs, Mission persists an ExecutionRoutingDecision that records:

- previewed preference where available;
- actual selection;
- fallback/switch reason;
- capacity/auth/health evidence;
- Agent selection evidence;
- model decision;
- timestamp.

UI should communicate "plan changed because capacity/auth changed", not silently swap.

## Integration with 1.0 milestones

This spine is P0 and controls the order of adjacent work:

1. Execution Preview + North-star E2E.
2. Capacity probes improve the resource preview.
3. Agent Modes improve the specialist preview.
4. Skills v1 improves task-specific context.
5. Cost Intelligence adds budget/cost preview and live accounting.
6. Unified Work Mode removes unnecessary surface transitions.
7. Mission certification proves preview -> approve -> execution -> evidence.

## Required end-to-end scenarios

### S1 — complete prompt
Paste a complete master prompt.
Expected:
- Composer recognizes it as ready or near-ready;
- no unnecessary interrogation;
- PromptArtifact finalized;
- Flow decomposed;
- execution preview generated;
- explicit approve;
- Mission receives same lineage.

### S2 — vague goal
"melhore a autenticação"
Expected:
- Composer asks only material blocking questions;
- repository-discoverable facts are discovered before asking;
- once ready, same downstream path as S1.

### S3 — multi-domain feature
Backend + frontend + tests + review.
Expected:
- multi-step DAG;
- independent work parallelized;
- specialist requirements differ per step;
- resource/model recommendation can differ per step.

### S4 — resource change after preview
Preview chooses resource A; A becomes rate-limited before GO.
Expected:
- strict admission/routing re-evaluates;
- resource B selected only if policy allows;
- durable reason explains divergence.

### S5 — no eligible resource
Expected:
- preview is NOT_READY;
- actionable blocker shown;
- no runtime starts.

## Definition of done

Composer Execution Spine is complete only when:

- one browser E2E covers complete prompt -> Composer -> Flow -> execution preview;
- one browser E2E covers vague prompt/clarification -> same downstream path;
- preview is side-effect free;
- AgentMatcher and ResourceScheduler explanations are visible;
- Mission actual routing is comparable to preview;
- prompt/plan/run lineage survives restart;
- no duplicate planner/router/runner was introduced;
- same-SHA CI is green.
