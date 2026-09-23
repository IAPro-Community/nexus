# IAPro Nexus 1.0 — Product Closure Program

Status: ACTIVE  
Branch: `feat/nexus-1.0-closure`  
Baseline: `main@7605f702cce220b4111d51d06a34b37f7e0a9a8d`  
Baseline source: PR #3, merged after same-SHA CI #149 was fully green.

## 1. Product contract

Nexus 1.0 is a local-first agentic development workstation that accepts an objective,
preserves context, selects an appropriate specialist and execution resource, isolates
work when needed, controls capacity and cost, requests human intervention at the
correct boundary, verifies the result and presents evidence of completion.

The primary user journey is:

```text
Goal
  -> understand context
  -> choose execution depth
       Direct | Composer | Mission
  -> choose specialist Agent
  -> choose provider/profile/model
  -> choose isolation + budget
  -> execute
  -> observe capacity + cost
  -> intervene when required
  -> review
  -> verify
  -> evidence
  -> DONE
```

Internal concepts such as RuntimeGeneration, ContextCapsule, WorkReceipt, quota
observations, scheduler affinity and leases must remain implementation details unless
shown under diagnostics/evidence.

## 2. Architecture invariants

The closure program MUST extend the existing seams and MUST NOT create competing
subsystems.

Canonical authorities:

- Agent identity and specialization: existing Agent / AgentRevision / AgentSpec.
- Execution incarnation: existing RuntimeGeneration / ProviderSession.
- Resource selection: existing ResourceScheduler / recommendation path.
- Task planning: existing WorkPlan / Flow projection.
- Durable execution: existing Mission runner.
- Memory: existing IntelligenceMemory.
- Human intervention: existing Attention / intervention contract.
- Capacity: existing Capacity Monitor and quota contracts.
- Terminal: existing SessionHost / PTY / TerminalsHost.
- Methodology/skills/risk when enabled: Maestro boundary.
- Entitlements: existing entitlement capability boundary.

Forbidden:

- second scheduler;
- second memory engine;
- second Mission runner;
- second workspace manager;
- provider identity embedded into Agent identity;
- UI-only entitlement enforcement;
- treating UNKNOWN quota/cost/support as healthy or verified;
- hidden automatic push/merge/deploy permission;
- giant untyped prompts as a replacement for AgentSpec/WorkPackage contracts.

## 3. Release definition

Nexus 1.0 can be released only when the release candidate has:

### Core
- Direct execution reliable.
- Persistent Agents reliable.
- Agent Modes/presets backed by AgentSpec.
- One canonical AgentMatcher.
- Real resource selection evidence.
- Production Capacity probes for the providers claimed as supported.

### Work
- Composer certified.
- Flow certified.
- Mission graduated from experimental for the supported 1.0 scenarios.
- Human intervention resume certified.
- worktree isolation policy certified.
- Evidence Bundle produced for autonomous work.

### Intelligence
- Project Intelligence certified.
- bounded Recall over existing memory.
- memory provenance visible and testable.
- Skills v1 discovery/loading contract.
- Maestro contract v1 optional and gracefully degraded.

### Control
- token usage attribution.
- cost attribution with explicit source/quality.
- budgets.
- anomaly alerts.
- routing explainability.

### Security
- capability/permission policy.
- attachment/path/IDOR/CSRF/Origin/WebSocket gates.
- secret/log/telemetry redaction.
- autonomous-execution threat model.

### Platforms
- Linux native same-SHA.

### Distribution
- signed artifacts.
- install / upgrade / uninstall.
- updater trust chain.
- entitlement server-side gates for any paid capability.

### Quality
- dogfooding scenarios.
- soak/overnight campaign.
- native provider smokes.
- BLOCKER=0 and HIGH=0.

## 4. Milestones

### P0 — Composer Execution Spine

Composer → Flow → Mission is the primary product spine for Nexus 1.0. The canonical
contract is [`DEV/SPECS/COMPOSER_EXECUTION_SPINE.md`](./COMPOSER_EXECUTION_SPINE.md).

Before treating Capacity, Agent Modes, Skills or Cost as independent feature work, Nexus
must expose a revision-bound, side-effect-free Execution Preview for a WorkPlan. Each
WorkPackage must be able to explain:

- the classified TaskRequirements;
- WHO would execute it (existing Agent or specialist creation);
- WITH WHAT provider/profile/account;
- model selection when real inventory/evidence exists;
- skills, isolation and verification requirements;
- current blockers;
- later, capacity and cost/budget evidence.

Mission remains the only execution authority and revalidates live state at GO time.
Preview may differ from actual routing only with an explicit persisted reason.

Exit:
- complete prompt → PromptArtifact → Flow → Execution Preview works;
- vague prompt/clarification converges to the same downstream path;
- preview starts no Agent/runtime/Mission and mutates no allocation;
- browser North-star coverage exists;
- no new planner/scheduler/runner is introduced.

### M0 — baseline closure and truth alignment

Baseline PR #3 is merged and CI #149 is green.

Tasks:
- [x] verify `main` points at or contains merge commit `7605f702...`.
- [x] record main CI evidence when available.
- [x] align VERSION / README / CHANGELOG / current release status.
- [x] remove stale beta.1/beta.2 “current” claims where they are no longer current.
- [x] preserve historical validation reports as historical evidence, not current truth.
- [x] make this spec the active closure contract.

Evidence recorded 2026-09-21:
- `main` points exactly at `7605f702cce220b4111d51d06a34b37f7e0a9a8d`.
- PR-head CI #149 is the green baseline evidence. The later `main` push run
  `35547840452` is recorded as a non-green/partial run: Linux and Security
  completed successfully, while Frontend was cancelled and dependent jobs did
  not execute. It is not promoted to green evidence.
- source truth is `1.0.0-beta.4` across VERSION/README/CHANGELOG/current status.
- repository tag inventory still exposes `v1.0.0-beta.1` as the published beta
  tag; install docs may pin that artifact but must not describe it as current source.
- older validation/candidate material is explicitly historical; this file and
  `DEV/validation/current/RELEASE_STATUS.md` are the active closure authorities.

Exit:
- one authoritative current release-status document;
- no contradictory current-version claims.

### M1 — production Capacity Intelligence

Capacity is an evidence provider for the Composer Execution Spine and Mission routing; it is not a separate routing engine.

Use existing `quota.Probe` / Capacity Monitor contracts.

Provider implementation order:
1. Codex.
2. AGY.
3. Claude where reliable data exists.
4. Gemini where reliable data exists.
5. OpenCode where reliable data exists.

Each provider probe MUST expose only evidence it can actually obtain:

```text
provider
profile/account
quota_group
window_kind
used_percent
remaining_percent
reset_at/reset_description
source
fetched_at
freshness
evidence quality
```

States with unavailable evidence remain UNKNOWN.

Tests:
- authenticated success;
- missing auth;
- malformed payload;
- stale observation;
- 429/rate limit;
- reset boundary;
- multiple accounts;
- no secret leakage;
- restart persistence;
- provider unsupported.

Exit:
- at least Codex production probe certified end-to-end;
- no empty production registry for claimed providers;
- alert/recommendation UI fed by real observations.

### M2 — Agent Modes and AgentMatcher productization

Presets backed by AgentSpec, not prompt aliases:

- Developer
- Senior Developer
- Reviewer
- Investigator
- Architect
- Security Reviewer
- QA Engineer
- Performance Investigator
- Documentation Specialist

Mode policy includes:
- role;
- instructions;
- responsibilities;
- required capabilities;
- constraints;
- default tool permissions.

Reviewer and Investigator are read-only by default.

Use and certify the existing canonical `MatchAgents` boundary everywhere the Composer Execution Spine and Mission need to choose a persistent specialist. Do not create a second matcher.

Input:
- TaskRequirements;
- AgentSpec candidates;
- current Agent state;
- project familiarity/evidence;
- required capabilities.

Output:
- selected Agent;
- candidate reasons;
- explicit rejection reasons.

AgentMatcher chooses WHO. ResourceScheduler chooses WITH WHAT.

Tests:
- deterministic matching;
- capability mismatch;
- least-privilege preference;
- unavailable/busy Agent;
- legacy Agent normalization;
- mission delegation;
- explanation parity Web/API.

### M3 — Skills contract v1

Use a portable `SKILL.md + scripts + resources` shape where possible.

Sources:
- project;
- user/install;
- Maestro;
- built-in Nexus.

Do not load every skill into context.

Pipeline:
```text
Task -> classify -> catalog metadata search -> candidates
     -> compatibility/risk -> selected skills -> bounded load
```

Requirements:
- provenance;
- version/source;
- capability requirements;
- conflict handling;
- duplicate resolution;
- no arbitrary script execution without policy;
- Maestro remains authoritative for methodology when enabled;
- Nexus remains functional without Maestro.

### M4 — Cost Intelligence

Implement an observational ledger before billing.

Canonical observation dimensions:
- ProjectID;
- branch/worktree;
- MissionID;
- RunID;
- AgentID;
- RuntimeGenerationID;
- provider;
- profile;
- model;
- input/cached/output/reasoning tokens when available;
- duration;
- cost value;
- cost kind;
- source;
- timestamp.

Cost kinds:
- PROVIDER_REPORTED;
- API_EQUIVALENT_ESTIMATE;
- INFERRED;
- UNKNOWN.

Never present estimated subscription cost as provider-billed fact.

Aggregation:
Installation -> Project -> Branch -> Mission -> Agent -> Runtime/model.

Add:
- daily/project/mission/agent summaries;
- anomaly detection;
- soft/hard/emergency budget policy;
- warnings before new expensive work;
- no arbitrary process kill unless policy explicitly requires it.

### M5 — Unified Work Mode

Composer becomes the primary complex-work entry point without creating a new engine.

The Composer bootstrap now has an explicit **Context Discovery** phase. It composes
existing Nexus sources instead of introducing another planner or context engine:

```text
User goal
  -> Brief seed
  -> Git identity / Context Readiness
  -> IntelligenceMemory recall
  -> Project Intelligence facts
  -> Context Discovery + budget
       -> KEEP            (bounded provider context)
       -> REFERENCE_ONLY  (discoverable evidence/skills/Missions)
       -> DROP            (visible provenance, excluded from model context)
  -> brief extraction / material clarifications
  -> Finalize
  -> Flow
  -> Execution Preview
  -> explicit GO
  -> Mission
```

The durable `ComposerContextManifest` is the explanation boundary for this phase.
It records provenance, estimated token cost, inclusion/exclusion reason and a stable
fingerprint. Final PromptArtifacts freeze the manifest used to produce the Flow.

Rules:
- explicit user file context outranks inferred repository context;
- Project/Install memory is bounded and current user input still wins conflicts;
- Project Intelligence remains static/bounded and never executes project commands;
- recent Missions enter as `REFERENCE_ONLY` pointers, never raw transcripts;
- Skill Catalog candidates enter as `REFERENCE_ONLY` until explicitly applied;
- only `KEEP` items reach the intelligence provider;
- context exhaustion produces `DROP`, never silent truncation or fake completeness;
- no Freebuff runtime/code dependency is introduced; only the context-first pattern is reused.

User sees:
- objective;
- gathered context;
- proposed plan;
- selected team;
- selected resources;
- isolation;
- budget;
- gates;
- Start.

Internally reuse Composer -> WorkPlan/Flow -> Mission.

Do not expose architecture jargon as primary UX.

### M6 — Mission 1.0 certification

Required live scenarios:
- single Agent;
- sequential multi-Agent;
- parallel independent work;
- human intervention;
- quota failure;
- auth failure;
- resource failover;
- Agent failure;
- test failure;
- review rejection;
- verification failure;
- Nexus restart;
- browser reconnect;
- terminal reconnect.

DONE/VERIFIED requires:
- all required packages complete;
- required tests/gates satisfied;
- no unresolved intervention;
- required reviewer accepted;
- verification contract satisfied.

### M7 — Evidence Bundle

Every autonomous Mission can produce:
- summary;
- changed files;
- commits;
- tests;
- logs;
- screenshots when relevant;
- browser assertions;
- performance/security evidence when relevant;
- reviewer findings;
- verification receipt.

Evidence is linked to the run and survives restart.

### M8 — Memory 1.0

Extend IntelligenceMemory; do not replace it.

Conceptual operations:
- Retain;
- Recall;
- Reflect.

For 1.0 scopes:
- INSTALL;
- PROJECT.

Recall must be cheap/bounded and not require an expensive model by default.
Reflect is optional and explicit.

Every fact carries:
- source;
- scope;
- timestamp;
- confidence/evidence;
- provenance.

Current explicit user input always wins.

### M9 — Maestro contract v1

Nexus owns execution. Maestro owns method/skills/risk/gates/verification policy.

Minimum machine-readable Maestro commands:
- `maestro capabilities --json`
- `maestro skills list --json`
- `maestro advise --json`
- `maestro verify --json`

Safe input only:
- project metadata;
- goal/task classification;
- changed paths;
- risk signals;
- available agents/skills.

Never send credentials, .env, auth files or raw private terminal transcripts.

Without Maestro:
- Direct works;
- Agents work;
- Composer works;
- Flow works;
- Mission works;
- methodology recommendations are marked unavailable.

### M10 — commercial/release certification

Use capability-based entitlements:
```text
entitlement.Has(capability)
```

Never `if plan == PRO` throughout feature code.

Potential capability gates:
- multi_agent;
- advanced_cost_controls;
- shared_memory;
- advanced_missions;
- team_policies.

Enforce server-side first; UI mirrors the decision.

Product analytics remains opt-in and allowlisted. Never transmit prompts, source code,
paths, tokens, credentials or file contents.

Nexus v1 stable release matrix is **Linux amd64 only** and must certify:
- CLI/Core;
- PTY/runtime;
- Direct;
- Agent start;
- Composer;
- Flow;
- Mission smoke;
- Desktop Linux launch;
- update;
- install/upgrade/uninstall;
- DEB lifecycle.

Windows/macOS implementations remain in source but are explicitly post-v1 platform
expansion until native same-SHA certification is completed and the release matrix is
amended.

## 5. Security policy for autonomous work

The **canonical 1.0 enforcement authority is the existing
`runner.AutonomyContract`**, combined with provider-native isolation,
worktree isolation and post-execution path validation. Nexus MUST NOT create a
second permission engine merely to expose different capability names.

The product may project these controls with user-facing capability labels such
as `read.files`, `write.files`, `run.tests`, `run.commands`,
`network`, `browser`, `git.commit`, `git.push`, `git.merge`,
`mcp`, `secrets` and `deploy`. A projected capability that has no
enforced 1.0 mapping remains unavailable/UNKNOWN; it must never become an
implicit grant.

Current enforced mappings include:
- file mutation scope → isolated workspace plus `AllowedFilePatterns`;
- network → `AllowExternalNetwork` plus a verified provider sandbox when the
  contract is restrictive;
- secrets → `AllowSecretAccess`;
- remote git push → `AllowGitPush`;
- deploy/publish/release → `AllowDeploy`;
- destructive git → `DisallowDestructiveGit`;
- test/verification execution → supervised Mission verification commands.

AgentSpec and WorkPackage capability metadata describe task/provider fit; they
are **not** operational permission grants.

Default rules:
- autonomous Reviewer/Investigator execution is read-only and requires a
  provider with verified read-only isolation;
- autonomous write prefers an isolated worktree; parallel autonomous write
  requires isolated worktrees;
- Agents do not own delivery merge: Nexus integrates verified package deltas;
- push/deploy and secret/network access remain denied unless the
  `AutonomyContract` explicitly allows them;
- provider CLI authentication remains provider-owned.

## 6. Worktree policy

- Direct interactive work may use the project workspace.
- Autonomous write work prefers worktree.
- Parallel write work requires isolated worktrees.
- Reviewer can operate read-only against source/diff without write worktree when safe.

No Mission may silently overwrite another Agent's working tree.

## 7. Quality protocol for every milestone

```text
SPEC
 -> baseline characterization
 -> failing/characterization tests
 -> implementation
 -> focused tests
 -> race/concurrency
 -> security review
 -> browser/UX
 -> independent review
 -> fixes
 -> same-SHA CI
 -> docs
 -> merge
```

Every PR/slice documents:
- problem;
- reused architecture seam;
- explicit non-goals;
- threat model where relevant;
- migration/rollback;
- tests/E2E;
- observability;
- performance/security impact.

Merge criteria:
- BLOCKER = 0
- HIGH = 0
- same-SHA required CI green.

## 8. Dogfooding and soak

Real tasks:
- small bug;
- frontend feature;
- Go backend change;
- migration;
- security investigation;
- large refactor;
- documentation;
- CI failure;
- browser regression;
- multi-Agent feature.

Measure:
- success;
- interventions;
- retries;
- provider switches;
- tokens;
- cost;
- duration;
- test result;
- reviewer findings;
- human corrections.

Soak:
- 4h;
- 8h;
- overnight.

Watch:
- runtime/PTY/goroutine leaks;
- memory growth;
- stale sessions;
- lost WebSockets;
- duplicate alerts;
- duplicate Mission execution;
- failover loops;
- runaway token consumption.

## 9. Explicitly post-1.0

Unless required by a blocker, defer:
- universal LLM proxy;
- dozens of new provider adapters;
- public MCP marketplace;
- public skill marketplace;
- voice/Jarvis/Whisper;
- media generation;
- cloud VM agents;
- TEAM shared memory;
- full remote collaboration;
- Stripe/billing;
- a new memory engine;
- a new scheduler;
- a new Mission engine.

## 10. Final release gate

Only declare `NEXUS_1_0_CANDIDATE = GO` when all required 1.0 items above are
implemented and proven on the final same-SHA **Linux amd64** candidate. Windows/macOS
are not part of the v1 GO/NO-GO boundary.

Otherwise report one of:
- CONDITIONAL only for external/non-code evidence blockers;
- NO_GO for functional/security/required-CI blockers.
