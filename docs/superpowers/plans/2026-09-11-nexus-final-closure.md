# Nexus Final Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate the existing Nexus intelligence, skill, routing and evidence contracts into one explainable autonomous execution path without duplicating existing Mission, Scheduler, Handoff or Attention systems.

**Architecture:** Keep `WorkPlan`, `ProjectContextSnapshot`, `MatchAgents`, `ResourceScheduler`, `MissionRunner`, `ContextCapsule`, `WorkReceipt`, Attention and `ValidationEvidenceStream` as canonical owners. Add one generic `internal/nexus/skills` boundary, one deterministic intent decision contract, one desired-vs-actual affinity/model resolution layer, and one persisted execution-routing decision that records why existing components were selected.

**Tech Stack:** Go 1.25, standard library, SQLite store, existing React/Vitest Web contracts, existing Go test/race/vet/build gates.

**Spec:** `DEV/validation/FINAL_CLOSURE_REALITY_AUDIT.md` plus the user-provided Nexus Final Closure contract.

## Global Constraints

- Do not create parallel Project Intelligence, Flow, WorkPlan, Agent Resolver, Resource Scheduler, Mission Runner, Attention, Handoff or Evidence Ledger systems.
- Skill consumers receive generic `Skill` values; Maestro is an optional `MaestroSource` only.
- Preserve `Agent`/`Persona`/`Runtime` and `Preference`/`Allocation` separation.
- `AUTO` may choose freely, `PREFER` may fall back without mutating preference, and `PIN` must fail closed when unavailable.
- Facts are bounded, redacted, provenance-bearing and never synthetic observations.
- Every behavior change follows RED → GREEN → focused verification → refactor.
- No automatic commit or push.
- Live-provider, overnight and unavailable native-platform results remain `UNVERIFIED`; never promote them to PASS.

---

### Task 1: Generic Native Skill Contracts and Builtin/Local Sources

**Files:**
- Create: `internal/nexus/skills/types.go`
- Create: `internal/nexus/skills/catalog.go`
- Create: `internal/nexus/skills/sources.go`
- Create: `internal/nexus/skills/catalog_test.go`
- Modify: `internal/nexus/maestro.go`
- Modify: `internal/nexus/prompt_compiler.go`
- Modify: `internal/nexus/intelligence/types.go`
- Modify: `internal/nexus/intelligence/engine.go`

**Interfaces:**
- Produces `skills.Skill`, `skills.Source`, `skills.Catalog`, `skills.Resolution` and `skills.NewBuiltinSource`/`skills.NewDirectorySource`.
- `MaestroClient` adapts its current discovery into a `skills.Source`; it does not define the consumer contract.
- Prompt compilation consumes generic `skills.Skill`/resolution references and preserves existing JSON compatibility where the HTTP surface requires it.

- [x] **Step 1: Write failing tests** for a builtin catalog resolving deterministic core skills, project/user SKILL.md discovery, source-aware deduplication, stable hash and Maestro absence.
- [x] **Step 2: Run the focused skill tests** and observe missing generic contracts/source behavior.
- [x] **Step 3: Implement the minimal generic types, bounded SKILL.md parser, builtin source and catalog resolver.** Do not execute skill content as a plugin.
- [x] **Step 4: Adapt existing Maestro catalog discovery as an optional source and replace `MaestroSkillDesc` consumer parameters with generic skill values.** Keep legacy endpoint aliases only as transport compatibility.
- [x] **Step 5: Run focused Go tests and `go vet` for the changed packages.**

### Task 2: Project Intelligence Inventory and Composer Grounding

**Files:**
- Modify: `internal/nexus/contextsnapshot/discovery.go`
- Modify: `internal/nexus/contextsnapshot/snapshot.go`
- Modify: `internal/nexus/context_readiness.go`
- Modify: `internal/nexus/composer.go`
- Create/modify tests next to the touched packages.

**Interfaces:**
- Consumes the existing `ProjectContextSnapshot` and `ContextReadiness` contracts.
- Produces a bounded context envelope containing only relevant facts with basis, confidence, source and identity metadata.

- [x] **Step 1: Add failing tests** proving Go/Node workspace topology, package/module paths, test/E2E/build/lint/CI command facts and secret redaction are represented with provenance.
- [x] **Step 2: Run tests and confirm current scanner/Composer fail the new assertions.**
- [x] **Step 3: Extend the existing scanner/parser only for evidence-backed metadata files and preserve all bounds/path protections.**
- [x] **Step 4: Pass a selected/bounded snapshot envelope to Composer `AnalyzeIntent` and `EvaluateAmbiguities`; do not send raw repository content.**
- [x] **Step 5: Verify stale identity invalidation and focused Composer/context tests.**

### Task 3: Explicit Intent Decision and CLI/Composer Routing

**Files:**
- Create: `internal/nexus/intelligence/routing.go`
- Create: `internal/nexus/intelligence/routing_test.go`
- Modify: `internal/app/app.go`
- Modify: `internal/nexus/flow_decomposition.go`
- Modify: relevant transport types/tests.

**Interfaces:**
- Produces `IntentDecision{Strategy, Confidence, KnownFacts, Assumptions, Unknowns, BlockingQuestions, Evidence, RecommendedNextAction}` with `DIRECT`, `CLARIFY` and `PLAN` strategies.
- Uses bounded Project Intelligence facts before classifying unknowns as blocking.

- [x] **Step 1: Write failing golden tests** for simple direct, discover-before-ask vague request, complete master prompt and material unknown.
- [x] **Step 2: Run the tests and confirm no explicit decision exists.**
- [x] **Step 3: Implement deterministic routing around existing archetype/requirements classification; Composer remains optional for direct tasks.**
- [x] **Step 4: Persist the decision/provenance in the existing WorkPlan/mission metadata path and expose it in `nexus run --json` without changing provider-native execution.**
- [x] **Step 5: Run CLI, Nexus and intelligence focused tests.**

### Task 4: Desired Runtime Affinity and Task-Aware Model Policy

**Files:**
- Create: `internal/nexus/runtime_affinity.go`
- Create: `internal/nexus/model_routing.go`
- Create: focused tests.
- Modify: existing resource recommendation/mission assignment integration points.

**Interfaces:**
- Produces policy values with `AUTO`, `PREFER`, `PIN`, desired resource identity and actual allocation/reason.
- Produces task-scoped model candidates/escalation without provider-name role hardcodes.

- [x] **Step 1: Write failing tests** for AUTO, PREFER fallback, PIN blocked, task override non-mutation, quota exhaustion, deterministic tie-break and model escalation after verification failure.
- [x] **Step 2: Run focused tests to confirm missing semantics.**
- [x] **Step 3: Implement resolution as a pure layer over existing `ProviderAccount`/scheduler candidates.**
- [x] **Step 4: Integrate task requirements/AgentSpec/optional Maestro guidance as inputs while keeping provider/account/model allocation out of Agent identity.**
- [x] **Step 5: Verify focused routing tests, race-safe immutability and full Nexus package tests.**

### Task 5: Persisted Execution Routing Decision

**Files:**
- Create: `internal/nexus/routing_decision.go`
- Create: `internal/nexus/routing_decision_test.go`
- Modify: existing Mission/plan store payload path and Web types only where an existing response already exposes routing.

**Interfaces:**
- Produces one `ExecutionRoutingDecision` containing task requirements, Agent score/reason, affinity desired/actual, selected engine/provider/profile/account/model, alternatives/rejections, fallback reason, health/quota snapshots, skill refs, Maestro guidance reference and timestamp.
- Reads as the source for explainability; no retrospective LLM explanation.

- [x] **Step 1: Write failing persistence/idempotency/explainability tests.**
- [x] **Step 2: Run them RED.**
- [x] **Step 3: Add the smallest durable field/record using the existing Mission/WorkPackage persistence boundary; do not introduce another store.**
- [x] **Step 4: Populate it at the canonical routing point and include desired-vs-actual data on fallback.**
- [x] **Step 5: Verify serialization, restart reload and deterministic tie-break tests.**

### Task 6: Maestro Lifecycle Adapter, Handoff Provenance and Evidence Projection

**Files:**
- Modify: `internal/nexus/maestro.go`, `internal/nexus/intelligence/types.go`, existing handoff/capsule/receipt adapters and validation evidence helpers.
- Create/modify focused tests and `DEV/validation/FINAL_CLOSURE_REPORT.md` only after fresh evidence.

- [x] **Step 1: Add failing tests** for PLAN/TASK/RECOVERY/VERIFY lifecycle advice, degraded Maestro mode, generic skill refs in capsules and evidence-chain projection.
- [x] **Step 2: Implement additive lifecycle metadata and generic skill provenance without changing Maestro ownership.**
- [x] **Step 3: Append closure scenario evidence to the existing `ValidationEvidenceStream`; verify the hash chain before projecting reports.** The local autopilot integration proves this with an ephemeral Git-bound stream; production authenticated Mission evidence remains required for GO.
- [x] **Step 4: Run the adversarial code review and classify every finding using technical verification before changes.**

### Task 7: Global Verification and Release Verdict

- [x] Run fresh `go test ./...`, `go test -race ./...`, `go vet ./...`, Go build, `git diff --check`, Web format/lint/style/typecheck/test/build and repository canonical gates.
- [x] Run deterministic local autopilot scenarios and mark live/native/platform scenarios `UNVERIFIED` where no runner/credential exists.
- [x] Verify the local evidence chain and record BASE SHA, FINAL SHA, ephemeral stream ID/head hash, commands, results, risks and remaining work in `DEV/WORKLOG.md`, `DEV/VERIFY.md`, `DEV/HANDOFF.md` and the final report. Production authenticated stream remains unavailable and is explicitly `UNVERIFIED`.
- [x] Emit `NO-GO` with specific blockers because authenticated Mission/provider, native platform and overnight evidence are still unavailable; `GO` remains forbidden until the central P1 evidence gates are cleared.
