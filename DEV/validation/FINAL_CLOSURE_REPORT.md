> **HISTORICAL — NOT CURRENT RELEASE EVIDENCE**
>
> Canonical current status: `DEV/validation/current/RELEASE_STATUS.md`.

# NEXUS FINAL CLOSURE REPORT

## BASE SHA

`2925ca746c198334f20d1e0cef7feb51e4f4e3`

## FINAL SHA

`42c22137a4a57ff6b6b80df8125b5a138f32b9e1` is the current commit on the target
branch and matches `origin/feat/nexus-maximum-delivery`. The working tree has
complete-stream evidence projection tests, provider-safety follow-ups and
final documentation updates not yet included in that commit. The preceding
finalization commit was created externally by the campaign autopilot and is
preserved.

## REUSED

- Existing Project Intelligence/context snapshot and bounded scanner.
- Existing WorkPlan/Flow, AgentSpec/MatchAgents, Resource Scheduler and Mission Runner.
- Existing ContextCapsule/WorkReceipt/handoff and Attention/restart boundaries.
- Existing append-only `ValidationEvidenceStream` and hash-chain verification.
- Existing Maestro client as an optional lifecycle/advice and Skill source.

## EXTENDED

- Native source-agnostic `SkillCatalog`, Builtin/Project/User/External and optional Maestro sources.
- Explicit `DIRECT | CLARIFY | PLAN` intent decision with bounded intelligence grounding.
- `AUTO | PREFER | PIN` desired affinity and task-aware model selection/escalation contracts.
- Durable routing decision projection with Agent evidence, identity-scoped
  account allocation and Skill resolution provenance.
- Scanner facts for commands, frameworks, topology, Go workspaces, nested packages and CI.
- Web routing report types and API projection.
- Canonical validation-evidence report projection through
  `RunApplicationService.ValidationEvidence` and
  `GET /api/v1/runs/{id}/validation-evidence`; missing streams remain explicit
  and chain verification is required before entries are exposed.
- Web client types/API now consume the same projection without recomputing or
  interpreting evidence locally.
- The projection verifies the shared stream before filtering by persisted
  `run_id`, preventing cross-Mission evidence leakage when Missions share a
  project; missing/invalid mission identity fails closed.
- The report path uses an explicit complete-stream read, avoiding silent
  truncation at the ordinary 100-entry list page.
- Codex legacy `home/.codex/config.toml` migration is now tested and ordered
  before canonical defaults, so existing settings are preserved.
- Codex profile preparation also honors the per-home TUI lock before mutating
  configuration/session artifacts.
- The provider E2E harness resolves imported profiles from the real host home,
  not a provider-rewritten `HOME`, with explicit override precedence tested.

## REFACTORED

- Canonical prompt and Intelligence paths now consume generic Skills/ExecutionGuidance.
- Maestro-specific prompt validation is isolated to a compatibility adapter.
- Optional Skill source failures degrade safely, including cyclic optional roots.
- Codex account attribution preserves durable identity boundaries.

## REPLACED

None. No parallel Project Intelligence, WorkPlan, Mission, Scheduler, Handoff,
Attention or evidence subsystem was introduced.

## NATIVE SKILLS

PASS locally: builtin and project Skills resolve deterministically without
Maestro; source-aware deduplication, provenance, version/hash and generic
prompt consumption are covered by `internal/nexus/skills` and Nexus tests.
Maestro remains optional through `MaestroSource`.

## PROJECT INTELLIGENCE

PASS locally: the existing bounded scanner supplies evidence-backed language,
framework, topology, package, command, CI and identity facts while preserving
limits, redaction, safe paths and no command execution during discovery.

## INTENT ROUTING

PASS locally: direct, clarification and planning strategies are explicit and
persisted; focused tests cover direct requests, discover-before-ask and full
prompts.

## COMPOSER GROUNDING

PASS locally: Composer receives selected bounded project context and uses the
generic catalog; direct work does not require a ritual Composer pass.

## AGENT RESOLUTION

PASS locally: deterministic matching, reuse/auto-creation and Agent score,
confidence and reason are persisted without making provider/model identity part
of the Agent.

## RUNTIME AFFINITY

PASS at contract/integration-test level: desired and actual allocations are
separate; PREFER may fall back without mutating preference; PIN fails closed.

## MODEL ROUTING

PASS at contract/integration-test level: selection is task-scoped, capability
aware, cheap-capable-first, deterministic and can re-enter allocation after
verification failure for escalation. Live provider/model escalation is not
authenticated evidence.

## EXECUTION ROUTING DECISION

PASS locally: persisted decisions include requirements, Agent evidence, desired
affinity, actual provider/profile/model/account scope, fallback reason,
canonical Skill refs/resolution provenance and timestamp. SQLite reopen and
read-only report projection are tested.

## MAESTRO LIFECYCLE

PASS locally for optional lifecycle/degraded contracts. Nexus remains the owner
of Agent/resource/runtime state; Maestro does not select concrete accounts,
providers or models. Remaining legacy fields are compatibility transport.

## HANDOFF

PASS locally for deterministic capsule/receipt continuity, failed-strategy
context and restart persistence. Native provider session resume and live
cross-provider continuation remain unverified.

## ATTENTION

PASS locally for durable NeedsUser, restart, idempotent response, independent
missions and silent quota failover tests.

## AUTOPILOT

PASS locally for the deterministic contract test:
bounded discovery → `DIRECT` → builtin Skill → task-aware model → Mission
Runner → `COMPLETED_VERIFIED`. This is not a real authenticated provider
Mission.

## VALIDATION EVIDENCE

PASS for append-only sequencing, idempotency, identity binding, tamper
detection, hash-chain unit/integration tests and the canonical read-only report
projection. No authenticated Mission in this campaign produced a durable
production stream ID, so no real Mission claim is promoted.

## REAL PROVIDER RESULTS

- Installed provider discovery: PASS for Codex, OpenCode, AGY, Claude, Cursor and Gemini in `./nexus doctor --json`.
- The local E2E harness bootstrap exchange and PTY submit path were fixed and regression-tested; this is harness evidence, not provider evidence.
- Codex host `exec` marker: PASS for provider availability. Nexus isolated-profile runtime: UNVERIFIED; the authenticated TUI remained at `model: loading` after prompt submission.
- AGY Nexus runtime: UNVERIFIED; the session reached the provider but reported not signed in and did not complete a Mission.
- OpenCode: UNVERIFIED; profile remains pending authentication.
- Codex/OpenCode/AGY live failover, account handoff, cross-provider handoff and model escalation: UNVERIFIED.
- No secrets were recorded.

## NATIVE PLATFORM RESULTS

- Linux/amd64 CLI build, PTY, WebKitGTK and provider installation checks: PASS.
- Linux desktop shell: SKIPPED; requires native Wails launch.
- Windows/macOS same-SHA native build/smoke: UNVERIFIED.

## OVERNIGHT RESULT

UNVERIFIED. No unattended representative overnight run was available in this
environment.

## GO / NO-GO

**NO-GO**

Local implementation and quality gates are green, but the central product
promise still lacks authenticated Mission/provider evidence, native
Windows/macOS evidence and overnight evidence.

## P0

None found in the final local review.

## P1

- Authenticated Mission with a real durable `ValidationEvidenceStream` ID.
- Live provider/account/model failover, PIN, escalation and semantic handoff proof.
- Native Windows/macOS same-SHA proof.
- Overnight unattended acceptance.

## P2

- Native desktop shell smoke in this Linux environment was skipped.
- Legacy Maestro-named persisted transport fields remain for compatibility migration.

## UNVERIFIED ITEMS

All real authenticated provider execution claims, live failover/escalation,
cross-provider handoff, Windows/macOS, native desktop shell launch and
overnight acceptance.

## VALIDATION EVIDENCE STREAM ID

`NONE — no authenticated Mission produced a production stream in this campaign.`

Local integration-only stream: `ves_06G96S8WCD415YJ44KHDPRDG0M` (temporary
test store, sequence 2, head
`cefd294db712c45cbfe377fdb48d5b62084561977935e07045cb351d2b806ba3`).

## HEAD HASH

Commit HEAD: `42c22137a4a57ff6b6b80df8125b5a138f32b9e1`.
The working-tree follow-up source/tests and evidence-document diff are not
represented by a commit hash.

## EVIDENCE CHAIN VERIFICATION

PASS for the automated append-only/hash-chain/tamper/idempotency tests.
Production promotion: NOT APPLICABLE; there is no real Mission stream ID to
verify or promote.

## LATEST VERIFICATION DELTA — 2026-09-12

- PASS — after the final Codex setup-lock adjustment: `go test ./... -count=1`,
  `go test -race ./... -count=1`, focused touched-package race tests,
  `go vet` and `git diff --check`.
- PASS — final `make quality`, `make build`, `make build-desktop`,
  `make security` and `make web-verify`; frontend remains 345/345 tests and
  the existing non-blocking ESLint warning is unchanged.
- PASS — `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c` for the Codex
  adapter; this is a cross-compile check, not native Windows evidence.
- PASS — `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `git diff --check`.
- PASS — `make security`, `make build`, `make build-desktop`, `make web-verify`.
- PASS — `make quality` after fixing one misspelled diagnostic comment and
  two staticcheck switch suggestions; all aggregated quality stages pass.
  ESLint retains one non-blocking existing warning at
  `web/src/nexus/AgentTerminal.tsx:218`.
- PASS — routing decision now persists canonical `skill_refs`, per-Skill
  source/version/hash/reason provenance and a real source-agnostic guidance
  reference when present; the Web type projects the resolution data.
- PASS — local E2E harness regression tests cover bootstrap token exchange,
  carriage-return prompt submission, symlink/cycle-safe profile copying and
  explicit stable-root resolution for providers that reject temporary homes.
- PASS — local autopilot integration now records package and global validation
  results into the canonical stream, verifies the chain and requires a real
  Git SHA with `VERIFIED` confidence. Latest ephemeral test run: stream
  `ves_06G96S8WCD415YJ44KHDPRDG0M`, sequence `2`, head
  `cefd294db712c45cbfe377fdb48d5b62084561977935e07045cb351d2b806ba3`.
  This stream lives in the temporary test store and is not production Mission
  evidence.
- PASS — canonical Mission evidence now records bounded Go/OS/architecture
  metadata and detects Node/package-manager versions when the project uses
  them; version probes use fixed arguments, no shell and a two-second timeout.
- PASS — Mission evidence projection isolates entries by persisted `run_id`,
  rejects missing mission identity, and returns all 101 entries in the
  complete-stream regression test instead of truncating at the default page.
- PASS — Codex legacy configuration migration preserves existing settings
  before `ensureConfigFile` adds the canonical default.
- PASS — `Prepare` waits for an active Codex TUI lock before setup mutation.
- PASS — E2E profile-source resolution uses `security.FindHostHome` and honors
  `NEXUS_E2E_PROFILE_SOURCE` overrides.
- OBSERVED — fresh `./nexus doctor --json` at `2026-09-12T03:34:53Z` confirms
  all announced provider binaries are installed, but this read-only probe does
  not establish provider authentication or Mission completion; desktop shell
  remains explicitly `SKIPPED`.
- PASS — the run application now projects the canonical validation stream via
  `ValidationEvidence`, verifies the chain before returning entries and exposes
  `GET /api/v1/runs/{id}/validation-evidence`; a missing stream is explicit
  `chain_verified=false`, never a fabricated PASS.
- PASS — the evidence projection was reopened against the same SQLite database
  after store close; run identity, stream, entry and verified chain survived
  the restart test.
- PASS — Codex TUI/app-server ownership now uses an OS-specific profile lock;
  quota probes skip mutation/app-server access while the TUI owns the profile,
  with focused lock, migration and rate-limit tests.
- PASS — Web client RED → GREEN coverage adds typed
  `getRunValidationEvidence`; `npm --prefix web run typecheck` and the full
  frontend test suite pass (345 tests).
- PASS — `./nexus doctor --json` at `2026-09-12T03:02:22Z` reports Linux/amd64
  directories, Secret Service, PTY, WebKitGTK and installed providers; the
  desktop shell remains explicitly `SKIPPED`.
- PASS — AGY background quota hardening is covered locally: expired access
  tokens fail closed, no-browser helpers are inert, and account email fallback
  parsing is tested without recording token contents.
- OBSERVED — provider inventory reports Codex profiles with live quota data and
  AGY registered/authenticated status, but this does not prove a completed
  Nexus Mission or evidence-ledger emission.
- UNVERIFIED — the 2026-09-12 stable-root Codex retry reached the authenticated
  TUI but remained at `model: loading`; the canonical-home/profile-lock change
  did not remove the provider boundary. The local startup also could not update
  `/etc/hosts` because sudo authentication was not available. The failure is
  retained as evidence, not promoted to PASS.
- UNVERIFIED — AGY Direct Work reached the real runtime but reported `not
  signed in` and stayed in its login flow; no provider marker was received.
- UNVERIFIED — no authenticated Nexus Mission emitted a durable evidence
  stream; no provider/platform/overnight blocker was removed.
