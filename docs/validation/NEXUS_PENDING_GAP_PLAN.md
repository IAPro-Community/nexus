# Nexus Pending Gap Plan

Baseline: `e28e38c6380cb639d673231449810d151adf2678`  
Verdict at audit freeze: **NO-GO**  
This plan is derived from `NEXUS_PENDING_FINAL_AUDIT.md`. It describes closure work; it does not claim that any campaign below has started or passed.

## Campaign P1 — Fail-closed HumanIntervention and resume

**Problem**

The current dirty resolver can accept generic decisions, clear unknown dispatch intents, duplicate external work, reject a repeated response, and resume without a durable decision transaction. A blocked run can also lack an actionable intervention.

**Existing infrastructure to reuse**

- `runner.MissionRun` and its canonical state machine;
- `HumanIntervention` / `MissionRun.NeedsHuman`;
- repository leases and dispatch-intent persistence;
- execution snapshots, package runs and existing runner recovery;
- application service boundary in `run_application.go`;
- canonical event bus/store recorder.

**Files/modules involved**

`internal/nexus/runner/{types.go,runner.go}`, `internal/nexus/{nexus.go,run_application.go,mission_service.go}`, store event/repository modules, web decision handler/API types.

**Required change**

Use typed option IDs and an intervention version/identity. Validate the option against the current intervention and `AutonomyContract`. Persist one decision record/event with an idempotency key before scheduling continuation. Treat unknown external dispatch outcome as non-retryable until an explicit safe reconciliation path exists; never reset unrelated package dispatches. Revalidate plan revision, checkpoint, dependencies and resources before resume. Make duplicate requests return the already persisted semantic resolution and reject stale versions.

**Tests**

Unit and repository tests for accepted, duplicate, stale, changed-run, invalid-option and unknown-dispatch cases; race test around worker stop/start; no duplicate dispatch and no duplicate side-effect spy.

**E2E evidence**

HTTP request → durable decision → restart → one resume → terminal verified state, with a provider/side-effect counter proving exactly one external effect.

**Definition of Done**

Every `BLOCKED_NEEDS_USER` item is answerable; resolution is exactly-once semantically, stale-safe and fail-closed; `FAILED_NO_PROGRESS` remains distinct; completed packages and receipts are not re-executed.

**Regression risks**

Lease races, old clients sending free-form strings, recovery of runs created before the intervention schema, and dispatch outcomes that are genuinely unknown.

## Campaign P2 — Global Attention projection and workspace integration

**Problem**

The dirty Mission Attention component is orphaned. The reachable radar is runtime-centric and does not provide a proven global Mission/Project Needs You, Completed and Failed/No Progress surface.

**Existing infrastructure to reuse**

- `MissionRun` as business truth;
- `GlobalAttentionRadar` / `InAppNotificationCenter` presentation patterns;
- existing workspace shell routes, deep-link sanitizer and design-system primitives;
- `AttentionFromRun` only as a read-model starting point, not as a new state machine.

**Files/modules involved**

`internal/nexus/runner/attention_*` (after contract repair), `internal/control/web/routes.go` and handlers, `web/src/app/NexusShell.tsx`, workspace navigation and the work feature.

**Required change**

Expose a project-isolated, workspace-global query with filters for all projects/project/mission. Wire the component and badge into the shell. Open Mission/decision surface first; use terminal only as supporting evidence. Ensure completed items are Mission-level and relevant, not micro-task noise.

**Tests**

Projection tests across multiple projects and missions; authorization isolation; empty/loading/error states; keyboard/focus and responsive tests; browser route test.

**E2E evidence**

Create A/B/C/D/E with deterministic fakes, verify C quota fallback is absent, restart server, verify A/D/E remain correct and B/C continue according to durable state.

**Definition of Done**

Global Attention answers “what work needs me now?” and never becomes a second Mission truth.

**Regression risks**

Mixing runtime Attention with Mission Attention, leaking cross-project data, showing stale completed runs, and making polling itself noisy.

## Campaign P3 — Mission notification delivery

**Problem**

`AttentionDedupKey` has no delivery consumer. Mission notifications have no proven durable intent, retry, delivery receipt or restart dedup.

**Existing infrastructure to reuse**

- canonical Mission events and recorder;
- existing browser notification/capability adapters;
- desktop capability and deep-link sanitizer;
- in-app Attention as mandatory fallback.

**Files/modules involved**

`internal/control/events`, Nexus store delivery area if justified, notification adapters and existing web/desktop notification modules.

**Required change**

Derive notification intents from canonical events using source ID/version/kind/channel. Persist delivery-only state, retry transport independently, dedup successful deliveries and redact deep links. Native notification must be capability-gated and best-effort.

**Tests**

Dedup, retry, transport failure isolation, restart recovery, no notification for quota/rate-limit/retry/remediation, deep-link sanitizer and secret redaction.

**E2E evidence**

Blocker/completion in-app plus native-capable and native-unavailable environments, demonstrating identical Mission correctness.

**Definition of Done**

Notification failure cannot change Mission state; one blocker does not produce repeated user notifications across cycles/restarts.

**Regression risks**

Turning delivery state into business truth, duplicate polling notifications, and platform permission assumptions.

## Campaign P4 — Project Intelligence and Composer grounding

**Problem**

Context is safely bounded but does not yet prove a structured inventory of stack, packages, tests, CI and commands, nor that Composer receives those facts as evidence.

**Existing infrastructure to reuse**

- `internal/nexus/contextsnapshot` allowlist/bounds/redaction;
- `context_readiness.go` fingerprints;
- Composer brief, intelligence provider and Maestro catalog.

**Files/modules involved**

`internal/nexus/contextsnapshot`, `context_readiness.go`, Composer/intelligence application contracts and tests.

**Required change**

Add typed, bounded inventory facts with source path, confidence and freshness. Discover only approved metadata/configuration files, redact secrets and invalidate on HEAD/dirty fingerprint changes. Pass relevant facts to intent/ambiguity analysis; do not send the repository wholesale.

**Tests**

Real ai-cli fixture question, source citations, stale fingerprint, symlink/path boundary, secret redaction and byte limits.

**E2E evidence**

Composer answers frontend test-stack question from repository evidence, then asks only a material unresolved question; complete prompt bypasses ritual questioning.

**Definition of Done**

Discoverable facts are investigated automatically, non-discoverable material ambiguity is asked, and no arbitrary secret/source loading occurs.

**Regression risks**

Context bloat, prompt leakage, stale inventory and overconfident inferred facts.

## Campaign P5 — Intent, WorkPlan and Flow acceptance

**Problem**

Direct, Composer, Flow and CLI paths have compatible source contracts but lack complete production/E2E proof of DIRECT/CLARIFY/PLAN and revision semantics.

**Existing infrastructure to reuse**

`DecomposePromptIntoFlowProposal`, `store.WorkPlan`, `PlanRevision`, Flow conversion, preflight and mission snapshots.

**Files/modules involved**

`internal/nexus/flow*.go`, plan application/store modules, CLI run command and relevant web surfaces.

**Required change**

Make routing decisions explicit and explainable. Ensure Composer is optional, Flow always projects from a WorkPlan, semantic edits create validated revisions, and visual edits do not.

**Tests / E2E evidence**

The five official product scenarios, including direct small fix, grounded ambiguous request, complete prompt, large feature and failure/continuity. Browser edits must be observable in persisted revisions and runner snapshots.

**Definition of Done**

One canonical operational WorkPlan exists for every execution path, with provenance and revision conflict protection.

**Regression risks**

Reintroducing Composer as a mandatory router or allowing Flow-local state to diverge.

## Campaign P6 — Agent fallback and resource proof

**Problem**

Deterministic matching and scheduler unit contracts exist, but unmatched specialization behavior and real resource transitions are unproven.

**Existing infrastructure to reuse**

`MatchAgents`, `TaskRequirements`, `RecommendResources`, `ResourceScheduler`, assignment strategies and mission executor.

**Required change**

Document and test the safe `AUTO`/`CREATE` policy for ephemeral or mission-scoped specialists. Preserve stable Agent identity and separate matching from resource selection. Unknown quota must remain bounded, never “unlimited”.

**Tests / E2E evidence**

Hard-gate rejection, deterministic tie, unauthenticated/unavailable resource, same-provider profile fallback, cross-provider fallback, quota exhaustion, rate limit and model-unavailable transitions with provenance.

**Definition of Done**

Safe fallback continues silently when policy permits; only a real human boundary produces Attention.

**Regression risks**

Creating persistent Agents unexpectedly, selecting unauthenticated accounts, or losing Agent/work state during handoff.

## Campaign P7 — Handoff, PTY and concurrency proof

**Problem**

Checkpoint and terminal infrastructure are present, but native resume, semantic continuation, real PTY ownership and full-suite race behavior are not proven.

**Existing infrastructure to reuse**

`internal/control/handoff`, driver interfaces, launcher, registry, host/protocol, leases and runner recovery.

**Required change**

Complete process-level tests for detach/reconnect, writer lease, stale ownership, runtime replacement and safe cross-provider context handoff. Preserve honest `NATIVE_RESUME_UNVERIFIED` until a provider acknowledgement exists.

**Tests / E2E evidence**

Authenticated same-provider resume, different-profile handoff, cross-provider checkpoint continuation, kill/restart, no duplicate writer/deadlock, and complete `go test -race -count=1 ./...` result with slow packages documented.

**Definition of Done**

No stale runtime claims active ownership; no duplicate dispatch/writer; recovery state is durable and explainable.

**Regression risks**

Provider CLI differences, process timing, orphaned runtimes and false “native resume verified” claims.

## Campaign P8 — Platform and security acceptance

**Problem**

Only Linux local evidence exists; Windows/macOS same-SHA execution and full architectural security review are unavailable.

**Existing infrastructure to reuse**

CI platform jobs, `nexus doctor`, desktop capabilities/deep links, route/CSRF/WS authorization and existing security targets.

**Required change**

Run the exact frozen/clean SHA on Linux, Windows and macOS. Record Go/Web build, CLI, paths, process management, PTY and desktop adapter evidence separately. Test project/runtime ownership, path traversal, command injection, provider args, URL sanitization and secret redaction.

**Tests / E2E evidence**

Strict same-SHA CI artifacts; native smoke where runners permit; security local gates plus a separate architectural review.

**Definition of Done**

No platform is labeled supported without native evidence; no dependency scanner result is presented as global security proof.

**Regression risks**

Cross-build being mistaken for native validation, platform-specific filesystem behavior and credential/log leakage.

## Campaign P9 — Final overnight acceptance

**Problem**

No long controlled campaign has demonstrated that a user can leave and return to verified, comprehensible state.

**Existing infrastructure to reuse**

All prior canonical contracts; Maestro owns benchmark scope and methodology.

**Required change**

After P1–P8, run a bounded campaign with project investigation, plan, dependencies, parallel work, Agent/resource selection, recoverable failure, remediation, verification and global DoD. Include resource transition if available.

**Tests / E2E evidence**

Finite timeout, bounded retries, durable event/evidence ledger, terminal states limited to verified done, failed no-progress or actionable blocked. Capture restart and next-morning inspection artifact.

**Definition of Done**

No silent success, infinite retry or unnecessary human interruption; verification is independent of Agent self-report.

**Regression risks**

Long-running resource leaks, stale UI projections, hidden provider authentication expiry and false evidence from mocks.

## Promotion gate

Promote to **GO** only when:

- the P0 unsafe-resolution risk is closed;
- no core-product P1 remains;
- S1–S5 execute with evidence;
- intervention is durable, actionable, stale-safe and idempotent;
- WorkPlan remains canonical and Composer remains optional;
- project grounding, Agent matching, resource failover and semantic continuity are proven;
- verification is independent of Agent claims;
- race/concurrency and officially supported platforms have same-SHA evidence;
- documentation reflects the evidence rather than historical claims.

Until then the correct status is **NO-GO**.
