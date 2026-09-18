> **HISTORICAL — NOT CURRENT RELEASE EVIDENCE**
>
> Canonical current status: `DEV/validation/current/RELEASE_STATUS.md`.

# Nexus Final Closure Checkpoints

## Work Package E — Codex legacy configuration migration

- `work_package`: E/provider continuity
- `status`: GREEN locally; release remains `NO-GO`
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: `42c22137a4a57ff6b6b80df8125b5a138f32b9e1` plus uncommitted
  provider-safety code, regression test and documentation
- `files_changed`: `internal/core/provider/adapters/codex/codex.go` and
  `internal/core/provider/adapters/codex/tui_lock_test.go`
- `tests_run`: RED test first observed the canonical config losing the legacy
  contents; GREEN focused migration/canonical-home tests; focused provider,
  host, flags and profile tests; race tests; Windows cross-compile; `git diff --check`
- `test_results`: PASS. `Prepare` copies a non-empty legacy config before
  appending the canonical credentials-store default and waits for an active
  TUI lock before mutating profile artifacts.
- `decisions`: preserve canonical single-home writes while keeping readable
  legacy migration; never create a competing `.codex/sessions` tree.
- `evidence`: `TestPrepareMigratesLegacyConfigBeforeCreatingCanonicalConfig`,
  `TestPrepareUsesCanonicalHomeOnly` and `TestPrepareWaitsForActiveTUILock`.
- `known_risks`: native provider authentication and live Mission evidence remain
  unavailable; Windows cross-compile is not native Windows validation.
- `remaining_work`: obtain authenticated Mission/provider, native platform and
  overnight evidence before reconsidering the release verdict.
- `unlocked_dependencies`: none for local implementation; external evidence is
  still the release blocker.

## Work Package P — E2E profile-source hardening

- `work_package`: P/provider proof harness
- `status`: GREEN locally; release remains `NO-GO`
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: `42c22137a4a57ff6b6b80df8125b5a138f32b9e1` plus uncommitted
  harness code, tests and documentation
- `files_changed`: `scripts/nexus-e2e-local.go` and
  `scripts/nexus-e2e-local_test.go`
- `tests_run`: RED compile test for missing resolver; GREEN focused resolver
  tests and `git diff --check`
- `test_results`: PASS. Imported profile source defaults to the real host home
  and an explicit `NEXUS_E2E_PROFILE_SOURCE` remains authoritative.
- `decisions`: reuse `security.FindHostHome`; do not infer host profile data
  from mutable provider `HOME`.
- `evidence`: `TestResolveE2EProfileSourceUsesHostHomeNotProviderHome` and
  `TestResolveE2EProfileSourceHonorsExplicitOverride`.
- `known_risks`: real Codex remains at `model: loading`; AGY reports `not
  signed in`; no authenticated Mission stream was produced.
- `remaining_work`: external authentication/provider health and native platform
  evidence remain required for GO.
- `unlocked_dependencies`: the E2E harness can now discover imported profiles
  correctly without a manual source override.

## Final local verification — 2026-09-12

- `work_package`: FINAL-LOCAL-VERIFICATION
- `status`: NO-GO; all local implementation gates are GREEN, external proof is
  still absent
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: `42c22137a4a57ff6b6b80df8125b5a138f32b9e1` plus uncommitted
  source, tests and evidence documentation
- `files_changed`: Codex per-profile lock/migration/flag handling, complete
  Mission evidence projection, quota safety, focused tests and closure docs.
- `tests_run`: `go test ./... -count=1`, `go test -race ./... -count=1`,
  touched-package race tests, `go vet ./...`, `make quality`, `make security`,
  `make build`, `make build-desktop`, `make web-verify`, Windows Codex
  cross-compile and `git diff --check`.
- `test_results`: PASS. No race, vet, build, frontend, security or diff-check
  failure was observed. Native Windows/macOS and real provider Mission proof
  remain unverified.
- `decisions`: retain `NO-GO`; do not promote installed-provider, cross-compile
  or deterministic fake-Mission evidence to authenticated production proof.
- `evidence`: `DEV/validation/FINAL_CLOSURE_REPORT.md`,
  `DEV/validation/FINAL_PROVIDER_MATRIX.md`, `DEV/VERIFY.md` and the focused
  Codex/evidence test names above.
- `known_risks`: no durable authenticated Mission stream; live failover,
  escalation, handoff, native platforms, desktop shell and overnight remain
  unavailable.
- `remaining_work`: obtain external authenticated/provider/platform/overnight
  evidence, then rerun the release verdict.
- `unlocked_dependencies`: none for local implementation; release remains
  externally blocked.

## Work Package L — per-Mission evidence isolation

- `work_package`: L/read-model isolation
- `status`: GREEN locally; release remains `NO-GO`
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: `42c22137a4a57ff6b6b80df8125b5a138f32b9e1` plus uncommitted
  evidence-isolation code, tests and documentation
- `files_changed`: `internal/nexus/store/validation_evidence.go`,
  `internal/nexus/validation_evidence.go` and focused tests/docs
- `tests_run`: focused Nexus/Store tests, Nexus/Store race tests, full
  `go test ./...`, full `go test -race ./...`, `go vet ./...`, `make quality`,
  `make security`, `make build`, `make build-desktop`, `make web-verify`, and
  `git diff --check`
- `test_results`: PASS. The full canonical stream is hash-verified first;
  projection then selects the persisted `run_id`, and malformed/missing
  mission identity fails closed. Complete-stream projection does not truncate
  after the default 100-entry list page.
- `decisions`: retain the existing bounded list API for ordinary consumers and
  add an explicit complete-stream read only at the canonical report boundary;
  do not create a second evidence store or infer claims from documentation.
- `evidence`: `TestRunApplicationProjectsOnlyEvidenceForRequestedMission`
  persists two Missions in one project and proves one-entry isolation for each;
  `TestRunApplicationRejectsMissionEvidenceWithoutRunID` proves the negative
  boundary; `TestRunApplicationProjectsCompleteMissionEvidenceBeyondDefaultPage`
  proves 101-entry completeness.
- `known_risks`: no authenticated Mission produced a durable production stream;
  external provider/platform/overnight proof remains unavailable.
- `remaining_work`: authenticate a real Mission/provider and rerun the global
  release evidence gate.
- `unlocked_dependencies`: long-run evidence reports are now complete and
  Mission-isolated; only external authenticated/native/overnight proof remains
  release-blocking.

## Work Package L — canonical validation evidence projection

- `work_package`: L/report projection
- `status`: GREEN locally; release remains `NO-GO`
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: `42c22137a4a57ff6b6b80df8125b5a138f32b9e1` plus uncommitted
  evidence projection, route, tests, Codex locking and documentation
- `files_changed`: `internal/nexus/validation_evidence.go`, its focused test,
  `internal/control/web/server.go`, `handlers_planning.go`, typed Web client
  API/test and closure docs
- `tests_run`: `go test ./internal/nexus ./internal/control/web -count=1`;
  restart projection and full Codex lock tests
- `test_results`: PASS. The application projection verifies the canonical hash
  chain before exposing entries; missing streams remain explicit with
  `chain_verified=false`; the same projection survives close/reopen of the
  SQLite store. Web client transport/typecheck/build also pass.
- `decisions`: reuse `ValidationEvidenceStream` as the sole source of truth;
  expose a read-only run projection and do not synthesize validation claims.
- `evidence`: focused Nexus/Web test output and the canonical stream contract.
- `known_risks`: no authenticated Mission produced a durable production stream;
  external provider/platform proof stays unverified.
- `remaining_work`: authenticate a real Mission/provider, then prove live
  failover/escalation/handoff, native platforms and overnight acceptance.
- `unlocked_dependencies`: Web/report consumers can now read canonical Mission
  evidence without accessing the store or reconstructing claims.

## Work Package E/G + P — final local continuation

- `work_package`: E/G/P
- `status`: GREEN locally; release remains `NO-GO`
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4f4e3`
- `current_sha`: `42c22137a4a57ff6b6b80df8125b5a138f32b9e1` plus uncommitted routing/evidence tests, AGY/runtime hardening and documentation
- `files_changed`: existing routing/prompt contracts, Web routing type,
  Skill resolution provenance, local E2E/evidence tests and stable-root
  harness support, bounded toolchain evidence capture, AGY/runtime quota
  hardening, Codex profile locking/native flag handling, validation reports and
  durable project docs.
- `tests_run`: `go test ./... -count=1`; `go test -race ./... -count=1`;
  `go vet ./...`; `git diff --check`; `make security`; `make build`;
  `make build-desktop`; `make web-verify`; `make quality`.
- `test_results`: PASS. Focused routing/guidance tests and scripts tests also
  PASS. Frontend verification is 10/10 PASS. Local autopilot also emits and
  verifies package/global canonical evidence entries with a Git SHA. AGY and
  runtime focused tests pass, including expired-token/no-browser behavior.
  Codex TUI/app-server lock and native flag tests pass.
  The aggregate quality gate also passes; one existing ESLint unused-variable
  warning remains non-blocking. The stable-root E2E smoke was retried against
  Codex and remained `model: loading`; AGY reported `not signed in`.
- `decisions`: generic guidance is persisted only as task input; routing now
  persists canonical Skill resolution provenance from the same catalog; Maestro
  guidance is source-labeled and reference-based; provider/account/model remain
  allocation data, not Agent identity. Direct host Codex evidence is not
  promoted to Nexus Mission evidence.
- `evidence`: `DEV/validation/FINAL_CLOSURE_REPORT.md`,
  `DEV/validation/FINAL_PROVIDER_MATRIX.md`, `DEV/VERIFY.md`, and the current
  frontend report `DEV/validation/FRONTEND_LATEST.md`; focused test output
  records the ephemeral stream head and sequence.
- `known_risks`: no authenticated Mission emitted a durable stream ID; live
  failover/escalation/handoff, native Windows/macOS and overnight remain
  unverified; Linux desktop shell launch remains skipped.
- `remaining_work`: obtain external authenticated Mission/provider evidence,
  native platform runs and overnight run, then append stream evidence and rerun
  the GO gate.
- `unlocked_dependencies`: none for release GO; implementation gates are green
  but external evidence remains blocking.

## Work Package B — Native Skills foundation

- `work_package`: B
- `status`: GREEN (foundation slice)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: `internal/nexus/skills/{types,catalog,sources}.go`, `internal/nexus/{skill_catalog,skill_contract}.go`, WorkPlan/Flow/runner transport, and focused Nexus tests
- `tests_run`: `go test ./internal/nexus/skills -count=1`
- `test_results`: PASS after RED → GREEN; initial RED was an expected compile failure before implementation.
- `decisions`: canonical source-agnostic `Skill`, `Source`, bounded SKILL.md parser, deterministic version/source deduplication, optional source degradation.
- `evidence`: builtin resolution and package freeze without Maestro, deterministic
  version/deduplication, bounded markdown discovery, optional-source unavailability,
  and canonical SkillIDs transport through Flow/runner/capsule.
- `known_risks`: Maestro and Composer still expose some legacy compatibility contracts; project/user source wiring and full resolution provenance remain pending.
- `remaining_work`: encapsulate/remove remaining legacy Maestro-named consumer contracts, add store-level skill resolution evidence, and complete project/user source provenance.
- `unlocked_dependencies`: generic catalog is now available to Composer and routing work.

## Work Package C/D — Project Intelligence and explicit intent routing

- `work_package`: C/D
- `status`: GREEN (bounded integration slice)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: `internal/nexus/project_intelligence_context.go`, `internal/nexus/intent_router.go`, Composer/intelligence provider integration, CLI goal routing.
- `tests_run`: `go test ./internal/nexus/... -count=1`; focused intent and grounding tests.
- `test_results`: PASS.
- `decisions`: prefer the persisted identity-bound snapshot; bounded static discovery is the safe fallback; provider ambiguity analysis receives the same bounded envelope when supported; `DIRECT`, `CLARIFY`, and `PLAN` are deterministic and explainable.
- `evidence`: atomic test routes DIRECT; repository facts prevent ritual clarification; complete multi-step goals route PLAN; empty/material objective routes CLARIFY; grounded facts include basis/confidence/source.
- `known_risks`: persisted Project Intelligence may be absent and is synchronously discovered; external intelligence/provider and Composer E2E remain unverified.
- `remaining_work`: persist intent decisions as first-class mission evidence and expand provider-backed end-to-end tests.
- `unlocked_dependencies`: routing can now feed WorkPlan creation and runtime resolution.

## Work Package E/G — Runtime affinity and routing decision

- `work_package`: E/G
- `status`: GREEN (deterministic resolution and mission persistence slice)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: `internal/nexus/runtime_routing.go`, runner/store WorkPackage contracts, `internal/nexus/mission_executor.go`.
- `tests_run`: `go test ./internal/nexus/... -count=1`; focused RuntimeRouting tests.
- `test_results`: PASS.
- `decisions`: AUTO/PREFER/PIN are explicit; desired provider/profile are copied into durable package state before actual allocation; fallback does not overwrite desired affinity; PIN returns `BLOCKED_RESOURCE`; task class chooses the cheapest capable model candidate.
- `evidence`: PREFER fallback preserves desired values; PIN unavailable fails closed; coding selects cheap capable model; architecture raises reasoning floor; package runs persist `routing_decision` JSON.
- `known_risks`: current scheduler still owns the final account allocation; model inventory is a contract-level candidate set and live provider/model evidence is unavailable.
- `remaining_work`: add store-level query/report projection for routing decisions and append mission verification entries to the canonical evidence ledger.
- `unlocked_dependencies`: explainability can be surfaced without changing Agent identity.

## Work Package L — Canonical validation evidence integration

- `work_package`: L
- `status`: GREEN (integration slice; campaign remains NO-GO)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: `internal/nexus/validation_evidence.go`, `internal/nexus/validation_evidence_test.go`, `internal/nexus/runner/{executor,runner}.go`.
- `tests_run`: `go test ./internal/nexus/... -count=1`; `go test ./... -count=1`; `go test -race ./... -count=1`; `go vet ./...`; `make quality-full`.
- `test_results`: PASS. Package and global verification results now append to the existing `ValidationEvidenceStream`, bind identity/environment/provider/profile/model where available, and verify the hash chain before promotion.
- `decisions`: recorder is an optional runner capability to preserve existing test executors; the real Nexus executor fails closed with `VALIDATION_EVIDENCE_UNAVAILABLE`; non-Git or missing-SHA observations remain `OBSERVED/NOT_VERIFIED`.
- `evidence`: unit coverage for identity binding, stable idempotency ID, append-only store chain, CAS and tamper detection; full race and security gates green.
- `known_risks`: no completed production Mission was run in this campaign, so no durable stream instance ID or end-to-end mission evidence can be claimed.
- `remaining_work`: execute an authenticated Mission that emits a real stream ID and project/provider/model evidence; add report projections and broader global DoD coverage.
- `unlocked_dependencies`: final reports can consume the canonical ledger once a real Mission is executed.

## Work Package P — Final verification and release verdict

- `work_package`: P
- `status`: NO-GO (evidence blockers remain)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: `web/src/features/work/ProjectIntelligenceInspector.tsx`, frontend verification reports, this checkpoint and handoff documentation.
- `tests_run`: `go test ./... -count=1`; `go test -race ./... -count=1`; `go vet ./...`; `make build`; `make build-desktop`; `make web-verify`; `git diff --check`; `./nexus doctor --json`; redacted real-provider smoke attempts.
- `test_results`: Go/frontend/security/build/race/vet gates PASS. Linux CLI and desktop builds PASS. `make web-verify` PASS with the current frontend report. AGY smoke PASS. Codex smoke reached the authenticated CLI but was rejected by provider usage limit; the other Codex profile has a local rules symlink loop.
- `decisions`: no PASS is assigned to Windows/macOS, overnight, OpenCode, or authenticated Mission E2E without direct evidence. No commit/push was created.
- `evidence`: `DEV/validation/FRONTEND_LATEST.md`, command outputs from the final gate runs, `./nexus doctor --json`, and the test suites above.
- `known_risks`: legacy Maestro-named compatibility contracts remain in Nexus types/Flow payloads; live account/model routing and model escalation remain partial; no real Mission stream ID was produced.
- `remaining_work`: remove/encapsulate remaining Maestro skill-type leakage; execute authenticated Mission scenarios 1–15; prove provider failover/PIN/model escalation/handoff/restart; native Windows/macOS and overnight runs.
- `unlocked_dependencies`: none for GO; campaign is correctly stopped at NO-GO until external/runtime evidence exists.

## Work Package B/E/G — native consumer decoupling and decision projection continuation

- `work_package`: B/E/G continuation
- `status`: GREEN (local implementation slice; campaign remains NO-GO)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: generic Agent prompt path and builtin Skills, generic intelligence guidance, prompt variants, legacy gate/Skill boundary, mission routing projection, Codex identity continuity, Flow canonical `skill_ids` transport.
- `tests_run`: `go test ./... -count=1`; `go test -race ./... -count=1`; `go vet ./...`; `make build`; `make build-desktop`; `make web-verify`; `git diff --check`.
- `test_results`: PASS. Focused RED→GREEN tests also pass for builtin/project-independent Skills without Maestro, generic guidance, legacy gate separation, persisted routing selection, and Codex identity history.
- `decisions`: Agent execution now resolves through the source-agnostic SkillCatalog; `MaestroSkillDesc` is no longer accepted by prompt variant compilation; `ExecutionGuidance` is canonical with a legacy Maestro alias; `MaestroGates` are not interpreted as Skills; mission allocation persists desired affinity and actual selected provider/profile/model without mutating preference.
- `evidence`: backend full and race suites, frontend `make web-verify` report `DEV/validation/FRONTEND_LATEST.md`, Linux CLI/Desktop builds, and clean `git diff --check`.
- `known_risks`: live model inventory/escalation is still contract-level in the Mission executor; Maestro lifecycle gate advice remains compatibility-oriented; no authenticated Mission emitted a real evidence stream instance in this campaign.
- `remaining_work`: authenticated provider Mission scenarios, account/provider/model failover and handoff proof, dedicated routing report projection, native Windows/macOS execution, overnight acceptance, and external evidence reconciliation.
- `unlocked_dependencies`: local code-quality gates are green; external/runtime evidence remains the release blocker.

## Work Package C/E/I — intent and explainability projection continuation

- `work_package`: C/E/I continuation
- `status`: GREEN (local contract slice; campaign remains NO-GO)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: working tree (no commit created)
- `files_changed`: intent decision metadata in existing WorkPlan structured facts, typed routing report projection and `/api/v1/runs/{id}/routing`, Web API client/types, Maestro lifecycle metadata.
- `tests_run`: focused Nexus/Web tests, `go test ./internal/nexus ./internal/control/web -count=1`, frontend routing API test and typecheck.
- `test_results`: PASS. Corrupt persisted routing JSON fails closed; valid decisions project selected provider/profile/model and reason; intent metadata preserves existing facts; degraded Maestro preserves lifecycle without fabricating advice.
- `decisions`: explainability is read from persisted decisions and never recomputed from live quota/health; the routing report reuses RunApplicationService; lifecycle is additive and defaults to `TASK`.
- `evidence`: `internal/nexus/routing_report_test.go`, `internal/nexus/intent_router_test.go`, `internal/nexus/maestro_test.go`, `web/src/nexus/api.test.ts`.
- `known_risks`: full task-aware live model escalation and lifecycle matrix remain incomplete; intent evidence is stored in WorkPlan metadata and report projection, but not yet appended as a separate ledger entry.
- `remaining_work`: complete model/escalation integration, append intent/closure evidence to the canonical ledger, and obtain authenticated Mission/native/overnight proof.
- `unlocked_dependencies`: consumers can query durable routing explanations without provider access.

## Work Package A/B/C/I — scanner and decision-report continuation

- `work_package`: A/B/C/I continuation
- `status`: GREEN (local implementation slice; campaign remains NO-GO)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: `a58cca4d73bdfd55678649b30c0ca64f73b7d410` plus uncommitted worktree changes
- `files_changed`: `internal/nexus/contextsnapshot/discovery.go`, its discovery tests, `internal/nexus/routing_report.go`, `internal/nexus/routing_report_test.go`, `internal/nexus/run_application.go`, plan/audit docs.
- `tests_run`: `go test ./internal/nexus/contextsnapshot -count=1`; focused report/intent tests; `go test ./internal/nexus/... -count=1`.
- `test_results`: PASS. The new scanner test was first observed RED, then GREEN. Operational facts now include package scripts, recognized frameworks, Go workspace uses, nested package topology and CI `run` commands with provenance. Routing reports project persisted intent and reject corrupt persisted intent/routing JSON.
- `decisions`: reused `contextsnapshot.Discover`; no second scanner or report store was created. CI and manifests remain static inputs; no project commands are executed during discovery. Explainability remains a projection of durable WorkPlan/package state.
- `evidence`: focused command outputs and package test results; no authenticated provider Mission was produced.
- `known_risks`: model escalation/live allocation, restart reload, authenticated Mission evidence, native Windows/macOS and overnight proof remain unverified. Existing Maestro-named compatibility payloads remain intentionally readable.
- `remaining_work`: complete live runtime integration and evidence scenarios, then rerun global quality gates and adversarial review.
- `unlocked_dependencies`: Project Intelligence facts and intent/routing projections are available to downstream Composer/Web consumers.

## External smoke status — 2026-09-12

- `NEXUS_E2E_PROFILE_SOURCE=/home/desenvolvedor/.local/share/ai-manager NEXUS_E2E_PROVIDER=agy go run ./scripts/nexus-e2e-local.go -start` reached bootstrap but could not establish an authenticated session because `sudo` authentication failed while updating `/etc/hosts`.
- Classification: `UNVERIFIED` / environment bootstrap blocker; this is not provider PASS evidence and no secret was recorded.

## Work Package E/G — task-aware model escalation and coupling cleanup — 2026-09-12

- `work_package`: E/G continuation
- `status`: GREEN (local contract slice; campaign remains NO-GO)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4e3`
- `current_sha`: `a58cca4d73bdfd55678649b30c0ca64f73b7d410` plus uncommitted worktree changes
- `files_changed`: `internal/nexus/runtime_routing.go`, `runtime_routing_test.go`, `resource_recommendation.go`, `config.go`, `mission_executor.go`, `runner/runner.go`, `runner/runner_durable_test.go`, generic prompt/intelligence adapters and optional Skill source handling.
- `tests_run`: focused routing/runner/skills/intelligence tests; `go test ./... -count=1`; `go test -race ./... -count=1`; `go vet ./...`; `make security`; `make build`; `make build-desktop`; `make web-verify`; `git diff --check`.
- `test_results`: PASS after RED→GREEN. Architecture/security tasks reject incapable preferred models; PREFER fallback preserves desired policy; PIN fails closed; escalation re-enters allocation after verification failure; tie-break is deterministic; cyclic optional Skill roots degrade safely; Intelligence no longer accepts a Maestro-specific guidance field; frontend and build gates are green.
- `decisions`: shared task-model pool resolution is used by pure and integrated runtime routing; live account health/auth/quota overwrites model metadata before selection; compatibility validation remains isolated in `maestro_prompt_compat.go`; canonical prompt/intelligence paths consume generic contracts.
- `evidence`: frontend report `DEV/validation/FRONTEND_LATEST.md` generated `2026-09-12T01:10:08Z`; Linux `./nexus doctor --json` generated `2026-09-12T01:10:47Z`.
- `known_risks`: no authenticated Mission produced a durable evidence stream ID in this campaign; live provider failover/model escalation, cross-provider semantic handoff, native Windows/macOS and overnight acceptance remain unverified. Compatibility payload fields still exist in Flow/runner/store for persisted transport migration.
- `remaining_work`: append real Mission closure scenarios to `ValidationEvidenceStream`; prove provider/account/model failover and handoff; complete restart/Attention matrix; native platform and overnight proof; final adversarial review and release verdict.
- `unlocked_dependencies`: local task-aware routing and generic prompt/intelligence boundaries are ready for authenticated Mission validation.

### Follow-up verification

- Routing decision JSON now has a SQLite reopen regression: the persisted
  package decision reloads after restart and is projected by
  `BuildRoutingDecisionReport` without recomputation. Deterministic provider,
  profile and model tie-break coverage remains green.

- Added `TestLocalAutopilotContractTraversesDiscoveryRoutingSkillsAndVerification`:
  bounded Project Intelligence → `DIRECT` intent → generic Builtin Skill →
  task-aware model selection → Mission Runner → verified terminal state. This
  is deterministic local proof only and does not replace authenticated provider
  or native-platform evidence.

- `RuntimeRoutingDecision` now persists the selected `AccountScope` alongside
  provider/profile/model, keeping explainability and future account handoff
  evidence identity-scoped.

- Agent matching evidence (`score`, `confidence`, `reason`) is now projected at
  the canonical allocation point into the same durable routing decision; Agent
  identity remains independent from runtime allocation.

## Final local gate checkpoint — 2026-09-12

- `work_package`: FINAL-LOCAL-VERIFICATION
- `status`: NO-GO (implementation gates green; external evidence blockers remain)
- `base_sha`: `2925ca746c198334f20d1e0cef7feb51e4f4f4e3`
- `current_sha`: `a58cca4d73bdfd55678649b30c0ca64f73b7d410` plus uncommitted worktree changes
- `files_changed`: `web/src/types.ts` contract projection plus generated frontend verification report and durable closure docs.
- `tests_run`: `make web-verify`; `go test ./... -count=1`; `go test -race ./... -count=1`; `go vet ./...`; `make security`; `make build`; `make build-desktop`; `./nexus doctor --json`; `git diff --check`.
- `test_results`: all commands PASS. Frontend report generated at `2026-09-12T01:34:01Z`; Linux/amd64 doctor PASS for installed providers, PTY and WebKitGTK; desktop shell SKIPPED.
- `decisions`: no provider installation status is promoted to authenticated Mission success; no platform or overnight claim is promoted without same-SHA evidence; no stream ID is fabricated.
- `evidence`: `DEV/validation/FRONTEND_LATEST.md`, doctor JSON output, full Go/race/vet/security/build outputs and deterministic local autopilot/routing/evidence-chain tests.
- `known_risks`: live provider failover, model escalation, cross-provider semantic handoff, native Windows/macOS, native desktop shell and overnight acceptance remain unverified; compatibility Maestro payloads remain for migration.
- `remaining_work`: obtain authenticated Mission evidence and external/native runners, then verify the real stream chain and rerun release review.
- `unlocked_dependencies`: local implementation and quality gates are green; release promotion is still blocked by external evidence.
