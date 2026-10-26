# Nexus Pending Gap Plan

Baseline: `e28e38c6380cb639d673231449810d151adf2678`  
Verdict at audit freeze: **NO-GO**  
This plan is derived from `NEXUS_PENDING_FINAL_AUDIT.md`. It is being executed incrementally; completion requires the campaign-specific evidence below, not code presence alone.

Historical progress snapshot (2026-10-25): P2 has a connected Attention Center, native fixture navigation, SQLite reopen coverage, repeated authenticated HTTP projection tests, and Chromium E2E coverage for loading, needs-user/completed/failed, intersecting project/mission filters, empty/error states, keyboard navigation, mission routing, and a 390 px viewport. The deterministic A/B/C/D/E backend E2E passed across five projects through authenticated HTTP plus SQLite/service/server reconstruction: the pre-restart projection was A/D/E, active B/C stayed out of Attention, and recovery completed B/C once. A separate real Wails session displayed fixture transitions from A/D/E to A/B/C/D/E and recovered the projection after restarting the isolated Web Core without reloading the window. This proves local single-user projection/polling, not a provider-backed run; multi-user ACL remains unsupported and unverified.

Linux homologation follow-up (2026-10-25): one NEX remains the mascot contract; the separate Companion 2D implementation is outside the product scope and is absent from the active HUD. `SKL-FIXTURE-001` now passes explicit and automatic skill selection through an isolated, supervised CLI fixture. It does not cover a real authenticated provider, so release gate `BLK-008` remains blocked. The full Linux result is still `NO_GO_LINUX`; current evidence and per-case counts are in `DEV/validation/linux-100-2026-10-25/`.

Current Linux recheck (2026-10-26): the product has one NEX mascot and no separate Companion 2D mode or renderer. The active HUD path is `NexusHud → GuardianMascot → NexPixelSprite`; old preference keys are retained only for migration/persistence compatibility. Six mascot suites (35/35), official art validation and HUD E2E passed; native Wails/XWayland shows one NEX, while alpha, complete Wayland interaction and live-agent attention remain pending. The current Composer copy fix passed 20 focused tests, complete/vague browser E2Es and `quality:full` (110 files/681 tests); without provider accounts the preview remains blocked and creates no Mission/Agent. A direct host-session import copied Gemini into the real Nexus profile, verified `0600` and idempotency; Gemini still needs refresh, and Claude has no token to copy. No Gemini model request was made; a separate direct Codex smoke through the installed Nexus CLI passed in ephemeral/read-only mode. Latest build/install overlay: HEAD `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + fingerprint `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c` (1,430 sources; worktree not frozen). Matrix: `NO_GO_LINUX`, 169 PASS / 1 FAIL / 21 BLOCKED / 0 NOT_RUN. P5 remains partial because native Wails/provider continuity is unverified; stable signed distribution, immutable provenance, alpha/GPU/compositor and multi-monitor/focus gates also remain. P9's deterministic eight-hour fixture soak is complete and does not validate external providers. See `DEV/validation/linux-100-2026-10-25/TEST_MATRIX.md`, `LINUX_HOMOLOGATION_REPORT.md`, `PROVIDER_SESSION_IMPORT_VALIDATION.md`, and `P5_CURRENT_BROWSER_RECHECK.md`.

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

P2 está completo para o contrato exercitado do workspace local single-user. Não há isolamento por usuário/ACL e não se declara suporte multiusuário; a transição nativa usou estado fixture, sem execução de provedor. Essas limitações não servem como evidência de integração externa aprovada, que permanece em gates próprios.

**Regression risks**

Lease races, old clients sending free-form strings, recovery of runs created before the intervention schema, and dispatch outcomes that are genuinely unknown.

**Progress — 2026-10-25 (COMPLETE)**

- Reproduced and fixed the P1 unsafe replay: an unsafe or unproven completed coordination dispatch now persists `ReplayProhibited`, offers verification without provider replay, preserves dispatch identity, and blocks retry/replan. SQLite reopen coverage protects this marker.
- Retry/replan from a proven pre-dispatch failure now re-enters `ALLOCATING`, clears the old prompt artifact, allocates/compiles again, and waits in `PENDING` until package dependencies are verified.
- A worker handoff that exceeds its first wait now schedules a single-owner retry after the canceled worker exits. Handler validation rejects malformed/trailing JSON and missing/non-positive intervention versions with HTTP 400.
- Validation passed: `go test ./internal/nexus/runner -count=1 -v`, `go test ./internal/nexus -count=1`, `go test ./internal/nexus/...`, `go test ./internal/control/web -count=1`, `go test -race ./internal/nexus/runner -count=1`, and `go vet ./internal/control/web ./internal/nexus/...`. Detailed evidence: `DEV/validation/linux-100-2026-10-25/P1_INTERVENTION_VALIDATION.md`.
- The plan-mandated integrated E2E now passes: authenticated HTTP resolution, durable decision/outbox, duplicate request, server/store restart, one resumed execution and verified terminal state. `go test ./internal/control/web -run '^TestResolveInterventionHTTPDecisionSurvivesServerRestartExactlyOnce$' -count=10` and the five-run race test both passed; logs and exact overlay fingerprint `ddad56697609de1374cdede949e794316e3263357b052f745defb64f11a4be5b` are recorded in `DEV/validation/linux-100-2026-10-25/P1_INTERVENTION_VALIDATION.md`.
- P1 is closed for this dirty overlay, not for a frozen release SHA. P2–P9 and global release gates remain open; overall recommendation stays `NO_GO_LINUX`.

## Campaign P2 — Global Attention projection and workspace integration

**Problem**

At the audit baseline, Mission Attention was orphaned and the reachable radar was runtime-centric. The current dirty overlay connects a workspace-wide Mission Attention panel, but a multi-mission durable-restart scenario has not yet proven the complete behavior.

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

Create A/B/C/D/E with deterministic fakes, verify C's active quota-remediation fallback does not become an Attention item, restart server, verify A/D/E remain correct and B/C continue according to durable state. C may appear under Completed after its retry succeeds.

**Definition of Done**

Global Attention answers “what work needs me now?” and never becomes a second Mission truth.

**Progress — 2026-10-25 (COMPLETE FOR LOCAL SINGLE-USER SCOPE)**

- The global shell panel, project/Mission filters, count badge and Mission detail navigation are connected. Backend filter tests, frontend route/gate tests, full Web quality and a native XWayland fixture click passed. A rendering defect in the Flow Run status summary was fixed and captured.
- A SQLite close/reopen integration test now proves the A/D/E projection remains canonical, executing B/C stay out of Attention, and project/mission filters do not cross scopes.
- Evidence: `DEV/validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md`; exact tested overlay fingerprint `7368531977b7422eccdf00e7c676f57d440e8d736d9e114bcdc071b9d5885fe6` over base `bf6adf0c80ac3531d7291844512a14a017ab0002`.
- Follow-up on HEAD `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`, overlay `340800c27162b0d340ac10a4d65e32af2efb09f470b1c17a12e9ba594c0ff98f`: authenticated HTTP projection survived SQLite/service/httptest-server close and reopen across two projects (10 normal, 5 race runs); unauthenticated access returned 401; project/mission filters intersected correctly. The Chromium HUD E2E passed fixture attention, snooze persistence, reduced-motion and desktop/mobile Axe checks. Detailed evidence: `DEV/validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md`.
- Follow-up UI on HEAD `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`, overlay `ee8a19c0cf340086f8f1cfc258d50a98ec29719952b978401be55c977a693b6`: the new Chromium E2E covered loading, categories, intersecting filters, empty/error states, no stale cards on API failure, keyboard focus, loaded mission navigation and 390 px responsive layout. It reproduced and fixed the stale-card and drawer-overflow defects; the complete frontend gate also passed. Details and screenshots: `DEV/validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md`.
- Follow-up A/B/C/D/E on HEAD `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`, source overlay `30223d700c7d7395705571ad72d2ea30055d191e181b03cb114a922395aef18b`: `TestAttentionHTTPMultiMissionQuotaRecoverySurvivesRestart` passed 10 repetitions, the dedicated race test passed 5 repetitions, the full Web handler package and `go vet` passed. A is needs-user, B is executing, C persists typed quota remediation without an Attention item, D completes and E fails. After SQLite/service/HTTP-server reconstruction, B/C recover once and complete. Before recovery the HTTP projection is A/D/E; after recovery it contains A/B/C/D/E. B has one allocation/execution; C has two (quota-limited attempt and successful retry). Detailed evidence: `DEV/validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md` and its linked logs.
- Follow-up nativo no overlay `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa`: a mesma janela Wails exibiu A/D/E, manteve a projeção enquanto B/C estavam ativos, atualizou para A/B/C/D/E após a fixture SQLite concluir B/C e recuperou o mesmo estado após restart do Core sem F5. Isso fecha o requisito de projeção cross-project para o produto local single-user. Log, procedimento e capturas estão em `DEV/validation/linux-100-2026-10-25/MISSION_ATTENTION_VALIDATION.md` e `TEST_MATRIX.md`.

**Regression risks**

Mixing runtime Attention with Mission Attention, leaking cross-project data, showing stale completed runs, and making polling itself noisy.

## Campaign P3 — Mission notification delivery

**Problem**

`AttentionDedupKey` had no delivery consumer. Mission notifications had no proven durable intent, retry, delivery receipt or restart dedup.

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

**Progress — 2026-10-25 (IMPLEMENTED; P3 validation complete)**

- Added delivery-only SQLite cursor, intent and per-channel retry state in migration 0031. `MissionRun` and the existing Attention Center remain canonical; delivery state never changes Mission state.
- `RunApplicationService` observes persisted Mission states after event publication. Application startup reconciles current runs and retries due deliveries, recovering missed events where a previous cursor exists. First observation establishes a quiet baseline, avoiding toast floods for historical runs.
- Native delivery is capability-gated, best-effort, deduplicated by stable run/intervention identity, and retried with bounded exponential backoff. Newer Mission state supersedes obsolete pending delivery. Completion remains in-app; only actionable blockers and contract-escalated terminal failures can produce native toasts. Quota/rate-limit/retry/remediation transitions are excluded.
- Payloads contain generic copy only, with no prompt or deep link. Transport errors are redacted before persistence. Fixed a shared notifier throttle bug that could suppress a retry and record a failed first send as success.
- Validation: full `go test ./internal/nexus/... ./internal/control/notify -count=1`, focused race suite (5 repeats), `go vet`, native GNOME/Wayland `notify-send` smoke, store close/reopen and deterministic unavailable-capability recovery passed. Detailed matrix and limits: `DEV/validation/linux-100-2026-10-25/MISSION_NOTIFICATIONS_VALIDATION.md`.
- The desktop transport was smoke-tested against the current GNOME session; toast pixels and behavior on another notification daemon remain outside this Linux session's visual evidence. P2 and P4–P9 remain open; overall release verdict stays `NO_GO_LINUX`.

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

**Progress — 2026-10-25 (IMPLEMENTED; deterministic E2E passed; provider smoke blocked)**

- Bounded Composer project context to 16 KiB/64 facts with per-value/list bounds, repeated secret redaction, UTF-8-safe truncation, relative provenance, per-fact observation time and current repository identity. Absolute local paths and scanner warnings no longer enter model context.
- Added regressions for byte limits, secret redaction, UTF-8, exact freshness invalidation and symlink containment. `go test ./internal/nexus/contextsnapshot ./internal/nexus -count=1`, the focused race suite (3 repetitions) and `go vet` all passed; see the P4 logs.
- A dedicated isolated Chromium E2E confirms Composer derives `vitest run` from fixture `package.json`, cites `package.json:scripts.test` with HIGH confidence and freshness, asks one material question for a vague goal, and asks zero blocking questions for a complete prompt.
- The local environment does not contain `ai-cli`; a real provider-backed question remains BLOCKED and no external account was used. Evidence: `DEV/validation/linux-100-2026-10-25/PROJECT_INTELLIGENCE_VALIDATION.md`.
- The broader P5 Composer execution spine advanced past prompt resolution after its fixture was corrected to answer the visible highest-priority unknown. Its first replay read the session before awaiting the `/finalize` response and briefly observed `READY_WITH_GAPS`; the test now awaits HTTP 201 and a Go regression confirms `FINALIZED` persists. The rerun passed with two clarifications and a five-step plan. The initial log and successful retest remain in `DEV/validation/linux-100-2026-10-25/logs/composer-vague-spine-p5-finalization-failure.txt` and `composer-vague-spine-p5-repro.log`.

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

**Progress update (2026-10-25):** stale WorkPlan edits now use compare-and-swap on `current_revision`. The store returns `ErrPlanRevisionConflict` without overwriting the latest title or appending an invalid revision; the HTTP handler returns 409. Focused race repetitions, affected package tests and vet passed on the overlay recorded in `DEV/validation/linux-100-2026-10-25/logs/source-provenance-p5-plan-revision-cas-2026-10-25.txt`. This closes the semantic conflict slice only; the five official scenarios and full browser-to-runner evidence remain incomplete, so P5 is still partial.

**Follow-up (2026-10-25):** FIX-025 separates Agent task specialties (`security`, `code_review`) from provider control capabilities in the scheduler gate, while retaining hard checks for explicitly requested provider capabilities such as `read_only_review`. The isolated Composer → WorkPlan → ready preview → approval token → immutable snapshot → MissionRunner fixture passed; `go test ./internal/nexus/... -count=1 -timeout=4m`, `go vet ./internal/nexus/...` and the focused regressions under `-race -count=3` passed. The fixture used synthetic Codex profile/auth/quota/CLI and deterministic executor/reviewer, with no external request. The five official P5 product scenarios, authenticated provider continuity and the equivalent browser-to-native UI path remain unverified; P5 is partial. Evidence: `DEV/validation/linux-100-2026-10-25/P5_COMPOSER_MISSION_VALIDATION.md` and its referenced logs. The overall Linux verdict remains `NO_GO_LINUX`.

**Follow-up semântico Composer→Flow→MissionRunner (2026-10-25):** a edição semântica agora tem regressão combinada: token anterior rejeitado sem Mission; preview renovado; snapshot do runner mantém o objetivo editado. Browser E2E confirma a persistência da revisão 2 e bloqueio sem provider. Suite Nexus, vet, race x3 e E2E passaram no overlay `ee6a9dd…`. Capturas/logs em `DEV/validation/linux-100-2026-10-25/P5_COMPOSER_MISSION_VALIDATION.md`. Isso fecha essa semântica em fixtures, mas não os cinco cenários oficiais completos, provider real ou UI Wails; P5 continua PARTIAL e a homologação NO_GO_LINUX.

**Reteste de cópia do preview (2026-10-26):** reproduzi e corrigi no overlay `101e6e62b8d4b9903cf4d40309caeea54c809183e5dba017d9e6a8770302669c` a duplicação do motivo “sem contas de provider” e a mensagem de orçamento local em inglês. Testes focados 20/20, E2E Composer completo e vago, e `quality:full` (110 arquivos/681 testes/build) passaram. A captura Chromium mostra as mensagens em português sem duplicação, com preview honestamente bloqueado e nenhum Mission/Agent criado. P5 continua PARTIAL: fluxo nativo Wails e handoff autenticado real ainda não validados. Evidência: `DEV/validation/linux-100-2026-10-25/P5_CURRENT_BROWSER_RECHECK.md` e o manifesto/hash list em `p5-current-recheck-2026-10-26-final/logs/`.

**Follow-up de falha pré-dispatch e retomada (2026-10-25):** `executeAgentPrompt` agora separa falhas de preparação local antes de `Launcher.Launch` dos resultados ambíguos retornados após entrar no launcher. A regressão comprova zero chamadas no erro local e preserva o bloqueio para erro do launcher. O cenário Composer OAuth multiárea injeta falha comprovadamente não enviada, fecha/reabre SQLite, recria o runner, conclui o plano e verifica que nenhum dispatch bem-sucedido se repete. Testes focados, race x3, suíte `internal/nexus/...` e vet passaram no overlay `2c23ebb0c7726a200acbde9326c23ad6023d8fad0238c2adfef993677203f2fa`, documentado em `DEV/validation/linux-100-2026-10-25/P5_COMPOSER_MISSION_VALIDATION.md`. Isto fecha a regressão local de falha/continuidade com fixture; não comprova handoff autenticado real nem fluxo Wails. P5 permanece parcial e o resultado global continua `NO_GO_LINUX`.

## Campaign P6 — Agent fallback and resource proof — COMPLETE (fixtures)

**Result (2026-10-25):** deterministic matching, stable scheduler tie-break, same/cross-provider and model fallback provenance, mission-scoped ephemeral specialists, durable assignment identity kind, and safe exit/startup cleanup passed. Broad `internal/nexus/...` + `internal/control/...` tests, focused race x5, and vet passed. Evidence: `DEV/validation/linux-100-2026-10-25/AGENT_FALLBACK_VALIDATION.md`. Real authenticated provider handoff remains unverified and is part of P7; global release result remains `NO_GO_LINUX`.

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

## Campaign P7 — Handoff, PTY and concurrency proof — PARTIAL/BLOCKED

**Result (2026-10-25):** the complete repository race suite passed (82 package result lines, exit 0), as did focused process-level PTY lease, detach/disconnect, stale ownership, binary replacement, and durable runner recovery tests. One live AGY profile returned READY; an exhausted profile returned the expected HTTP 429. Live authenticated same-provider resume and cross-provider continuation remain BLOCKED: Codex runtimes already active were preserved, AGY had no eligible target account, and the current OpenCode profile did not produce a final response within 90 seconds. `NATIVE_RESUME_UNVERIFIED` remains explicit. Evidence: `DEV/validation/linux-100-2026-10-25/P7_HANDOFF_PTY_VALIDATION.md`.

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

**Progress — Linux local slice (2026-10-25, PARTIAL)**

- `make security` passou com govulncheck v1.7.0 (`No vulnerabilities found.`); a evidência está em `DEV/validation/linux-100-2026-10-25/SECURITY_VALIDATION.md`.
- `make release-dry-run` passou; GoReleaser v2.18.0 gerou snapshot CLI tar.gz/DEB para Linux AMD64. Checksums, igualdade dos payloads e instalação/consulta/remoção de DEB em raiz dpkg isolada passaram no overlay fingerprint `268a9e9a0036e89e39bcff07eca880869534c9c75cc13765384bb3fa3fc30b32` sobre HEAD/remote `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`.
- O snapshot é beta.10 sem assinatura ou publicação. A versão estável/updater segue sem validação; a revisão arquitetural continua pendente. Este resultado não encerra P8 nem certifica uma árvore imutável.

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

**Progress — Linux fixture slice (2026-10-25, PARTIAL)**

- `make overnight-smoke` passou: etapas do runner, dependência, falha injetada, remediação limitada e verificação global foram exercitadas por executores determinísticos em workspaces temporários.
- `make overnight-soak DURATION=5m` passou em 300,012 s; o harness afirma timeout finito, ciclos não vazios e estados terminais explícitos. Evidências: `DEV/validation/linux-100-2026-10-25/logs/p9-overnight-smoke.log` e `p9-overnight-soak-5m.log`.
- P9 permanece parcial: não foi provado o reinício de uma execução persistente seguido de inspeção no dia seguinte, nem a expiração de autenticação/continuidade real com provedores. As fixtures não são evidência de sucesso externo.

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


## P5 follow-up — multi-area OAuth and semantic approval (2026-10-25)

A larger synthetic OAuth feature now exercises Composer → multi-area WorkPlan → semantic edit → stale-token rejection → refreshed approval → MissionRunner completion with at least four packages. `go test ./internal/nexus/...`, `go vet ./internal/nexus/...`, and the focused race regression repeated three times passed after a race in the test executor counter was fixed with a mutex. The initial race log and successful retry are preserved in `DEV/validation/linux-100-2026-10-25/logs/`. The synthetic profile and executor do not validate a real provider, cross-provider continuity, or the native Wails flow; P5 remains partial. Current tested overlay: `0c522c885e700dc52d6d03a0feb313bc7d5e420915ca0188fe6c554b49fe9841` (`DEV/validation/linux-100-2026-10-25/logs/source-provenance-p5-large-feature-mission-fixture-2026-10-25.txt`).
