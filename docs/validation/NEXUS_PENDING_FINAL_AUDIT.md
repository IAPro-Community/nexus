# Nexus Pending Final Audit

Status: **NO-GO**  
Audit type: evidence-first validation, first pass only  
Scope: Nexus autonomous loop, project intelligence, routing, WorkPlan, Agent/resource selection, recovery, human intervention, Attention, terminal continuity, web, security and platform evidence.  
Benchmark Nexus versus baseline: **out of scope**, owned by Maestro.

## 1. Frozen reality

The repository was fetched and inspected before analysis. The campaign baseline is frozen at:

| Field | Evidence |
|---|---|
| Repository | `kivervinicius/ai-cli` |
| Branch | `feat/nexus-maximum-delivery` |
| HEAD / baseline SHA | `e28e38c6380cb639d673231449810d151adf2678` |
| Historical reference | same SHA; no commits after it |
| Remote tracking | `HEAD...@{u}` = `0 0` |
| HEAD commit | `feat: implement agent matching and classification improvements` |
| Commit time | `2026-09-11T10:28:42-04:00` |
| Audit time | `2026-09-11T12:35:59-04:00` |
| Platform | Linux `x86_64`, kernel `6.17.0-23-generic` |
| Go | `go1.25.0 linux/amd64` |
| Node | `v22.17.0` |
| npm | `10.9.2` |
| Bun | local `1.4.0`; project/CI package manager `bun@1.3.9` |
| Worktree | dirty |

The dirty worktree contains an uncommitted Attention attempt: runner/application changes, `AttentionCenter`, policy/tests, and web API/types. Those files are not part of the frozen SHA and are treated as **working-tree evidence only**, never as proof for the baseline.

Relevant dirty files:

`internal/browser/browser.go`, `internal/runtime/runtime.go`, `internal/control/web/handlers_planning.go`, `internal/control/web/routes.go`, `internal/nexus/nexus.go`, `internal/nexus/run_application.go`, `internal/nexus/runner/runner.go`, `internal/nexus/runner/types.go`, `web/src/nexus/api.ts`, `web/src/types.ts`, plus the untracked runner Attention files and `web/src/features/work/AttentionCenter*`.

Working-tree change domains, all excluded from baseline proof: browser/runtime helper hardening; Attention/mission/application/web; testing; and documentation. There are no commits after the frozen SHA, so the committed “since reference” delta is empty.

## 2. Documentation trust

The source and tests are newer evidence than historical documents. The following conflicts were found:

| Document | Assessment | Evidence |
|---|---|---|
| `DEV/VERIFY.md` | current verification ledger, but explicitly leaves live provider, authenticated PTY/failover, native Windows/macOS and full race unverified | document claims compared with source and executed commands |
| `DEV/SPECS/ACTIVE.md` | current handoff, useful for known limits; not proof of runtime behavior | claims require source/test corroboration |
| `DEV/NEXUS_WORKSPACE_OS_FINAL_QA.md` | stale in places; still describes Missions/runner as not implemented although the SHA contains them | contradicted by `internal/nexus/mission_service.go` and runner tests |
| `docs/validation/EVOLUTION_FINAL_AUDIT.md` | historical/partial; correctly records several unverified areas, but is not this SHA's final proof | source and current commands take precedence |
| `docs/validation/EVOLUTION_FINAL_VALIDATION.md` | historical `NO-GO`/partial validation; not a current gate | must not be promoted without rerun |
| `DEV/WORKLOG.md` and related finalization notes | historical narrative; contains claims that exceed current executable evidence in some areas | no single reproducible command proves all claims |

No existing document was rewritten in this first pass.

## 3. Architecture discovered and reuse classification

`REPLACE` was not justified by the evidence. Existing infrastructure is reusable, but several contracts need extension and proof.

| Capability | Current implementation / source of truth | Test evidence | Real execution evidence | Status | Action |
|---|---|---|---|---|---|
| Project context | `internal/nexus/contextsnapshot`, `context_readiness.go`; fixed allowlist, bounded excerpts, redaction and fingerprints | `contextsnapshot/snapshot_test.go`, readiness tests | no real context question answered through provider-backed run | PARTIAL | EXTEND |
| Intent / task requirements | `DecomposePromptIntoFlowProposal`, `ClassifyTaskRequirements`, CLI `run` path | decomposition and classification tests | CLI help only; no side-effect-free official scenario run | PARTIAL | EXTEND |
| Composer | `composer.go`, `composer_application.go`, `composer_brief.go`; Maestro catalog is consumed | Composer lifecycle, unknown and revision tests | no provider-grounded Composer session against this repository | PARTIAL | EXTEND |
| Composer optionality | direct decomposition and prompt finalization do not require Composer persistence | `TestComposerFinalizationCreatesPromptWithoutWorkPlan`, flow decomposition tests | no complete CLI/Web proof | PARTIAL | REUSE |
| WorkPlan canonicality | `store.WorkPlan`, immutable `PlanRevision`, mission execution snapshot | store revision tests, `mission_revision_test.go`, flow contract tests | no full mutation-to-execution E2E | PARTIAL | REUSE |
| Flow projection | `FlowFromWorkPlan` / `WorkPlanFromFlow` in `internal/nexus/flow.go` | flow and contract tests, North Star cycle | no browser edit/run proof | PARTIAL | REUSE |
| Agent matching | `MatchAgents` in `internal/nexus/agent_matching.go` | hard gates, deterministic ranking and explanation tests | no real mission dispatch proving selected Agent | PARTIAL | REUSE |
| Ephemeral specialist fallback | `AssignmentStrategy` supports `AUTO`/`CREATE`; mission executor can create an Agent during execution | source/tests cover creation paths, not specialization fallback policy | no proof that an unmatched specialist is safely created or rejected with explanation | UNVERIFIED | EXTEND |
| Resource Scheduler | `scheduler.go`, `resource_recommendation.go`, `mission_executor.go` | deterministic preference, unavailable/auth/quota/capability tests | no authenticated provider failover | PARTIAL | REUSE |
| Quota/rate-limit fallback | scheduler and quota monitor paths; normal fallback is intended to stay internal | unit/package tests | no real quota exhaustion or rate-limit run | UNVERIFIED | EXTEND |
| Same-provider continuity | control driver `CanResume` / `BuildResumeArgs`, continuity and scheduler preference | driver and continuity unit tests | provider-level native resume is explicitly not verified | PARTIAL | EXTEND |
| Cross-provider handoff | `internal/control/handoff/context.go`, bounded `WorkCheckpoint` and lineage | checkpoint/redaction/context handoff tests | no authenticated cross-provider run | PARTIAL | EXTEND |
| Semantic handoff | checkpoint carries goal, Git metadata, changed files, commands, tests, errors and Maestro context | checkpoint tests | no Agent A → Agent B intellectual-continuity run | PARTIAL | EXTEND |
| Mission state machine | canonical durable states in `internal/nexus/runner/types.go` | runner persistence/restart and terminal-state tests | no process-kill/restart campaign | PARTIAL | REUSE |
| Human intervention | `MissionRun.NeedsHuman`, `HumanIntervention`; dirty attempt adds IDs/version/decision fields | dirty runner tests only; one idempotency test asserts the wrong expected behavior | no durable HTTP decision/resume proof | FAIL | EXTEND |
| Attention policy | dirty `attention_policy.go`; existing runtime Attention remains in `GlobalAttentionRadar` | dirty classification tests | no integrated global Mission view | FAIL | EXTEND |
| Attention query | dirty `AttentionCenter` projection over run repository | dirty in-memory projection tests | no workspace integration or restart projection proof | FAIL | EXTEND |
| Notification delivery | existing browser/runtime notifications; no Mission notification outbox consumer found | notification model tests | no durable Mission delivery/dedup/retry proof | FAIL | EXTEND |
| Maestro integration | `maestrogates`, status/catalog/gates/advice contracts and Composer skill proposals | Maestro/status/catalog tests | no end-to-end PLAN/TASK/RECOVERY/VERIFY lifecycle | PARTIAL | EXTEND |
| PTY / terminal | `internal/control/terminal`, host, protocol, registry and platform capabilities | Linux package tests, doctor checks | no authenticated supervised PTY continuity campaign | PARTIAL | EXTEND |
| Browser/runtime helper | dirty `internal/browser/browser.go` and `internal/runtime/runtime.go` avoid `xdg-open` shim recursion | targeted `go test`/`go vet` pass; browser package has no tests | no native browser-helper smoke or cross-platform proof | UNVERIFIED | EXTEND |
| Security boundaries | redaction, deep-link sanitizer, route auth/CSRF/WS tests and security targets | local tests and `make security` evidence in verification ledger | no global adversarial proof across all runtime/provider paths | PARTIAL | EXTEND |
| Platforms | CI definitions for Linux/Windows/macOS and local Linux execution | Linux local evidence | same-SHA Windows/macOS execution unavailable | UNVERIFIED | EXTEND |

## 4. Findings by validation phase

### Project Intelligence — PARTIAL

`internal/nexus/contextsnapshot/snapshot.go` proves a conservative bounded context envelope:

- fixed allowlist of durable documents;
- maximum eight excerpts, 6 KiB per excerpt and 24 KiB total;
- redaction before final truncation;
- no recursive repository walk or arbitrary source-file loading.

`context_readiness.go` fingerprints branch, HEAD, dirty files and Maestro version, and has `MISSING`, `HYDRATING`, `READY`, `STALE` and `FAILED` states.

The gap is product-level project intelligence. The allowlist is not an evidenced structured inventory of language stacks, frameworks, packages, entrypoints, test/E2E frameworks, CI, commands, boundaries and architecture hints. No real Nexus run answered “How does this repository test the frontend?” with cited source facts. Verdict: **PARTIAL**, not PASS.

### Intent and routing — PARTIAL

The source path exists: CLI goal form → `DecomposePromptIntoFlowProposal` → `WorkPlanFromFlow` → `CreateWorkPlan` → `StartMissionRun` → runner. Atomic versus multi-step decomposition and `TaskRequirements` persistence are covered by unit tests.

The implementation is heuristic (`isAtomic` checks prompt terms and word count), and no controlled execution of the five official product scenarios was completed. DIRECT/CLARIFY/PLAN equivalence and the “discoverable unknown before asking” rule therefore remain only partially evidenced.

### Composer grounding and optionality — PARTIAL

Composer creates a structured brief and reads project identity/readiness. Turns can invoke `AnalyzeIntent` and `EvaluateAmbiguities`, but the provider payload shown in `composer.go` is primarily `project_id` plus the evolving brief; there is no proof that the bounded repository context inventory is included as grounded evidence.

The tests prove Composer finalization can occur without creating a WorkPlan, and Flow materialization is explicit. They do not prove that a complete intent-to-WorkPlan path works without a ComposerSession in CLI/Web production wiring.

### WorkPlan and Flow — PARTIAL

`store.WorkPlan` and immutable revisions are the strongest canonical model found. Flow is a conversion/projection façade, and runner snapshots freeze plan revision. This is consistent with the desired architecture.

The missing proof is end-to-end semantic editing: task/dependency/Agent/gate/verification changes must create a validated WorkPlan revision, while visual movement must not mutate semantics. Existing tests are structural and in-memory, not browser-to-store-to-run evidence.

### Agent matching and resources — PARTIAL / UNVERIFIED

`MatchAgents` has hard gates, deterministic scoring/tie-break and explanations. Resource recommendation separately filters authentication, availability, cooldown, health and capability constraints, and has explicit unknown-quota scoring.

No real provider-authenticated mission proves the selected Agent, account/profile, model and runtime are actually dispatched. No definitive evidence proves safe ephemeral specialist fallback when no persistent Agent matches. These are not reasons to create a second resolver; they are extension/proof work on existing contracts.

### Recovery and handoff — PARTIAL

The handoff package has bounded/redacted checkpoints, Git metadata, Maestro read order, lineage, same-provider resume argument construction and explicit `NATIVE_RESUME_UNVERIFIED`. This honesty is correct.

The checkpoint does not itself establish a complete Mission `WorkPlan`/package receipt/acceptance-criteria/failed-strategy envelope, and no Agent A → Agent B intellectual-continuity scenario was executed. Native provider continuation and cross-provider continuation remain unverified without authenticated runtime evidence.

### Human intervention and Attention — FAIL

The dirty Attention attempt is not promotable and is not baseline evidence. It exposes concrete correctness failures:

1. `internal/nexus/runner/runner.go:495-503` rejects a repeated response because it checks `run.State != BLOCKED_NEEDS_USER` before the already-resolved branch. The corresponding dirty test expects an error despite commenting that the operation should be idempotent. Required “same answer twice → one semantic resolution” is not satisfied.
2. `runner.go:519-526` resets every executing package with a dispatch intent after any accepted decision. An unresolved provider dispatch has unknown external outcome; clearing its dispatch identity can issue a duplicate provider call or side effect. It also touches unrelated parallel packages. This is a P0 unsafe-execution risk.
3. `run_application.go:161-170` persists the run and then publishes an in-memory resume event. There is no atomic durable decision event/outbox, no explicit decision-option validation, and no dependency/resource revalidation or checkpoint restore in this path.
4. `mission_service.go:389-417` can race worker removal: resolution starts a worker while the old blocked worker may still occupy `n.workers`; the start returns, then the old defer deletes the entry, potentially leaving `EXECUTING` with no worker.
5. `runner.go:157-162` and `runner.go:428-433` can set `BLOCKED_NEEDS_USER` without creating a `NeedsHuman` intervention. The dirty policy classifies every such state as `REQUIRE_USER`, producing a blocker with no answerable contract.
6. `web/src/features/work/AttentionCenter.tsx:63-76` sends generic `proceed`/`dismiss` strings instead of a validated structured decision. The backend accepts raw decision/chosen values and does not enforce the operation/policy boundary.
7. The new component is not imported by the workspace shell. Existing `GlobalAttentionRadar` and `InAppNotificationCenter` are runtime-centric; the mission-level center has no proven global badge, project aggregation or mission navigation integration.

The canonical Mission state machine remains the source of truth; the failure is in the attempted application/projection contract, not a justification for a second state machine.

### Notifications — FAIL

`AttentionDedupKey` exists in the dirty policy file but has no production consumer. No `NotificationIntent`, durable outbox/delivery state, retry receipt or channel adapter integration for Mission Attention was found. Existing notification tests cover browser/runtime attention, not durable Mission blocker/completion delivery. Therefore deduplication after restart and transport-failure isolation are unproven.

### PTY, concurrency and platform evidence — PARTIAL / UNVERIFIED

Linux unit and package tests, `nexus doctor --json`, deep-link tests and terminal protocol tests provide local evidence. `doctor` reports PTY and WebKitGTK capability as available, while desktop shell native launch is skipped.

The full non-race Go suite passed. A full `go test -race -count=1 ./...` reached the 600-second timeout with `internal/app` consuming CPU and no final result. It is therefore **UNVERIFIED**, not PASS. No process kill/restart of an active multi-Mission campaign, authenticated provider continuity, or same-SHA Windows/macOS run was executed.

## 5. Executed evidence ledger

All repository source references above are anchored to baseline SHA `e28e38c6380cb639d673231449810d151adf2678`. Commands that inspect the dirty Attention attempt are explicitly labeled working-tree evidence.

| Command | Result | Evidence class | Limitations |
|---|---|---|---|
| `git fetch --all --prune` | PASS | baseline identity | remote state only |
| `git checkout feat/nexus-maximum-delivery` / `git pull --ff-only` | PASS / already current | baseline identity | no changes made |
| `git rev-parse HEAD`, `git status --short`, `git log -20`, `git rev-list --left-right --count HEAD...@{u}` | PASS | frozen identity | worktree is dirty |
| `go test -count=1 ./...` | PASS | repository-wide Go tests | not proof of live provider/platform behavior |
| `go vet ./...` | PASS | static Go gate | not a security audit |
| `go test -race -count=1 ./internal/nexus/runner` | PASS | focused race gate | not full-suite race proof |
| `go test -race -count=1 ./...` | TIMEOUT after 600s; no final result | unverified | long-running `internal/app`; no PASS assigned |
| `go test ./internal/nexus/runner ./internal/nexus ./internal/control/web` | PASS | dirty working-tree targeted tests | includes uncommitted Attention attempt; not baseline SHA proof |
| `go vet ./internal/nexus/runner ./internal/nexus ./internal/control/web` | PASS | dirty working-tree targeted vet | same limitation |
| `go test -count=1 ./internal/browser ./internal/runtime` | PASS; browser has no test files | dirty working-tree targeted test | no browser-helper behavior assertion |
| `go vet ./internal/browser ./internal/runtime` | PASS | dirty working-tree targeted vet | no cross-platform proof |
| `npm --prefix web run typecheck` | PASS | local web gate | Linux only |
| `npm --prefix web run lint -- --no-warn-ignored` | PASS | local web gate | does not replace browser E2E |
| `npm --prefix web run lint:styles` | PASS | local web gate | does not prove visual UX |
| `npm --prefix web run check:styles` | PASS | local web policy gate | does not prove accessibility |
| `npm --prefix web run test` | PASS, 66 files / 336 tests | local web tests | no browser/backend contract E2E |
| `npm --prefix web run test -- --run src/features/work/AttentionCenter.test.ts` | PASS, 1 file / 6 tests | dirty working-tree targeted test | test mocks API; does not render the component or hit backend |
| `npm --prefix web run build` | PASS | local web production build | Linux only; not desktop build |
| `npm --prefix web run format:check` | FAIL | local formatting gate | fails new `AttentionCenter.tsx` and `AttentionCenter.test.ts` |
| `gofmt -l` on changed Nexus files | FAIL | local formatting gate | reports six dirty Go files |
| `make quality` | FAIL | canonical quality gate | stops at frontend format check |
| `make docs-verify` | FAIL | documentation gate | visual manifest is stale and cannot certify dirty visual changes |
| `go run ./cmd/nexus --help` | PASS | CLI surface smoke | no mission was started |
| `go run ./cmd/nexus doctor --json` | PASS on Linux | capability smoke | desktop shell is skipped; no authenticated mission |
| `git diff --check` | PASS | whitespace check | does not include all untracked files |

## 6. Official scenarios and required proofs

| Scenario / proof | Status | Reason |
|---|---|---|
| S1 simple direct work | UNVERIFIED | source/unit decomposition exists; no controlled real invocation |
| S2 ambiguous improvement request | UNVERIFIED | no grounded Composer/intent run with discoverable repository facts |
| S3 complete master prompt | PARTIAL | Composer prompt review/finalization tests exist; no complete production route proof |
| S4 large feature PLAN → Flow → execution → verification | PARTIAL | North Star decomposition/preflight tests exist; no real autonomous execution |
| S5 provider problem → recovery → semantic continuation | UNVERIFIED | no authenticated provider transition or Agent-to-Agent continuity run |
| Multi-Mission A/B/C/D/E attention scenario | FAIL | no process-level E2E; dirty projection is in-memory and not integrated |
| Kill/restart during multi-Mission scenario | UNVERIFIED | no process kill/restart execution |
| Resolve Mission A exactly once | FAIL | dirty resolver is not idempotent and resets unknown dispatch intents |
| Stale intervention rejection | PARTIAL | version/ID checks exist in dirty runner code/tests; no durable HTTP/restart proof |
| Real provider-authenticated run | UNVERIFIED | not executed |
| Real quota/failover | UNVERIFIED | not executed |
| Overnight autonomy | UNVERIFIED | not executed |
| Native Linux/Windows/macOS matrix | UNVERIFIED | only Linux local evidence; other runners unavailable |

## 7. Severity findings

### P0

- **Unsafe human-resolution resume can duplicate external effects.** `runner.go:519-526` clears unknown dispatch identity for all executing packages, and the decision path does not validate an allowed structured operation against the autonomy contract. This can cause duplicate provider dispatch/API/side effects and must be fail-closed before promotion.

### P1

- Human intervention idempotency is not implemented; repeated accepted HTTP decisions do not return the existing semantic resolution.
- Resolution lacks a durable decision event/transactional outbox and does not explicitly restore checkpoint or revalidate dependencies/resources.
- Worker start/removal race can leave a run in `EXECUTING` without a worker.
- `BLOCKED_NEEDS_USER` can be persisted without an actionable `NeedsHuman` contract.
- Mission-level Attention is an unintegrated dirty component, while the reachable radar is runtime-oriented rather than global Mission/Work-oriented.
- Mission notification outbox, delivery state, retry isolation and semantic dedup are absent/unproven.
- Authenticated provider failover, native resume, cross-provider continuation and overnight autonomy are unverified.
- Full-suite race result is not available at audit capture.
- Project intelligence is bounded and safe but not yet a real structured repository inventory grounded into intent/Composer.
- Ephemeral specialist fallback policy is not proven.

### P2

- New Attention web files fail formatting; dirty Go files fail `gofmt`.
- Attention UI contains hardcoded user-visible strings and generic fallback text despite i18n requirements.
- Attention UI uses clickable `div[role=button]` for mission items and lacks a demonstrated keyboard/focus contract.
- `AttentionCenter.test.ts` does not render the component despite its test name and uses `any`.
- `make docs-verify` is blocked by stale visual-manifest state and dirty visual files.
- Existing historical documents overstate or contradict current executable evidence.
- Browser-helper changes have no dedicated package tests or native platform smoke evidence.

## 8. Promotion verdict

**NO-GO.** The branch has useful reusable foundations, but the core autonomous promise is not proven and the dirty Attention attempt introduces a P0 duplicate-side-effect risk. No claim of GO is permitted until the P0 is closed, P1 core gaps are closed or explicitly bounded by product policy, and the official scenarios plus platform/provider evidence are rerun on a clean, identified SHA.

## 9. Exact remaining work

1. Make human decision resolution a typed, versioned, durable application transaction with allowed option validation, one semantic resolution, stale rejection, event/outbox durability and fail-closed behavior for unknown external dispatch outcomes.
2. Resume from the persisted checkpoint without resetting unrelated package dispatches; make worker lifecycle restart-safe and prove no duplicate provider dispatch or external side effect.
3. Ensure every `BLOCKED_NEEDS_USER` state has an actionable `HumanIntervention`; preserve `FAILED_NO_PROGRESS` as distinct.
4. Integrate a global Mission/Work Attention projection into the workspace shell and global badge, aggregating projects without making Attention a second business state machine.
5. Add Mission notification intent/delivery semantics only on top of canonical Mission events, with semantic dedup, retry and transport isolation; retain in-app fallback.
6. Extend bounded context into an evidenced project inventory and pass the relevant grounded facts into Composer/intent analysis without reading secrets or whole repositories.
7. Decide and test safe ephemeral Agent fallback using existing assignment/matching contracts.
8. Run authenticated same-provider, cross-provider, quota and provider-failure transitions; capture provenance and honest continuity status.
9. Complete PTY detach/reconnect and process-level multi-Mission restart tests; finish or isolate the full race suite.
10. Run browser E2E/accessibility and the same-SHA Linux/Windows/macOS matrix where runners exist. Update historical docs only after evidence is captured.
