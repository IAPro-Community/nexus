# P6 — Agent fallback and resource proof

Date: 2026-10-25  
Branch: `release/v1.0.0`  
Commit base and remote at test: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`  
Source fingerprint: `bb575423508a0c773580804a476b0888ff2449135045a74f80aa6efaffedc6e1`  
Source manifest: `logs/source-provenance-p6-agent-fallback-2026-10-25.txt` (1,421 product source files)

## Scope and product contract

P6 checks deterministic resource matching, fallback provenance, and the lifetime of mission-scoped specialists. This campaign used controlled Go fixtures; it did not call a real external provider and does not establish live provider continuity or native resume.

The agent identity policy exercised is:

- An existing assignment remains pinned to its identity unless the selected policy permits fallback.
- An `AUTO` assignment can reuse an eligible persistent Agent or create a mission-scoped specialist when no match is available.
- An explicit `CREATE` specialist is ephemeral and tied to the mission/worktree.
- A `PackageRun` durably records the assigned Agent ID and identity kind.
- Ephemeral specialists stay out of the persistent Agent list. Cleanup removes only stopped/failed, unreferenced ephemeral workers; active, resumable, working, and persistent Agents are retained.
- Equal scheduler scores resolve by stable resource identity, independent of input order.
- Authentication, availability, quota, rate-limit, and model-capability checks remain hard gates; every permitted fallback records the requested and selected resource and the cause.

The product mascot remains one NEX. This campaign does not add a Companion 2D mode or a second mascot.

## Test matrix

| ID | Area | Procedure | Expected | Actual | Status | Evidence | Severity | Commit tested |
|---|---|---|---|---|---|---|---|---|
| P6-001 | Agent matching | `go test ./internal/nexus -run '^TestMatchAgentsUsesHardGatesAndDeterministicRanking$' -count=1` | Reject hard-gate failures and rank eligible matches deterministically. | Fixture verifies availability/authentication/capability gates and deterministic ranking. | PASS | `logs/go-test-p6-agent-fallback.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P6-002 | Resource scheduler | `go test ./internal/nexus -run '^TestSchedulerUsesStableResourceIdentityTieBreak$' -count=1` | Equal scores select the same resource regardless of input order. | Regression failed before the comparator change and passes with provider/profile/ID/display-name tie-break. | PASS | `logs/go-test-p6-agent-fallback.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P6-003 | Provider/model fallback | Run focused fallback regressions for quota, rate limit, authentication, and model incapability. | Allowed fallback records requested and selected identities plus cause; unauthenticated profiles are rejected. | Same-provider and cross-provider fixture transitions pass; unavailable-model provenance is persisted. No live provider was called. | PASS | `logs/go-test-p6-agent-fallback.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P6-004 | Specialist identity | `go test ./internal/nexus -run '^TestCreateMissionAgentPersistsMissionScopedSpecialistWithoutRuntimeAffinity$' -count=1` | Specialist has an ephemeral identity and is absent from persistent Agent listings. | Regression failed when identity defaulted to persistent; it now verifies ephemeral identity, mission visibility, and no persistent listing. | PASS | `logs/go-test-p6-agent-fallback.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P6-005 | Worker cleanup | Run reconciliation tests for terminal/orphaned, active/resumable, working, and persistent Agents, including service startup. | Remove only stopped/failed, unreferenced ephemeral workers. | Fixtures confirm active references and working/persistent identities survive; terminal/orphaned workers are removed after exit and at startup. | PASS | `logs/go-test-p6-agent-fallback.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P6-006 | Durable assignment | Runner lifecycle test persists the assigned Agent identity kind with its ID and workspace. | Restart/reload retains the assignment identity kind. | `PackageRun.AssignedAgentKind` is stored with the selected identity; durable runner regression passes. | PASS | `logs/go-test-p6-agent-fallback.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P6-007 | Go quality gates | `go test ./internal/nexus/... ./internal/control/... -count=1`; focused `go test -race ... -count=5`; `go vet ./internal/nexus/... ./internal/control/...` | Covered packages pass tests, repeated race suite, and vet. | All commands exit 0. This is not the full-repository race gate requested by P7. | PASS | `logs/go-test-p6-agent-fallback.log`; `logs/go-race-p6-agent-fallback.log`; `logs/go-vet-p6-agent-fallback.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |

## Result and limits

P6 deterministic matching, fallback provenance, ephemeral identity, persistence, and cleanup regressions pass. The focused race and vet checks pass for the recorded package scope. Full-suite race behavior, authenticated same-provider resume, cross-provider semantic handoff, and process-level kill/restart remain assigned to P7. External live provider behavior remains unverified. The overall Linux release decision remains `NO_GO_LINUX`.

## Reproduction

From the repository root:

```bash
go test ./internal/nexus/... ./internal/control/... -count=1
go test -race ./internal/nexus ./internal/nexus/runner -run '^(TestCreateMissionAgentPersistsMissionScopedSpecialistWithoutRuntimeAffinity|TestReconcileEphemeralMissionAgentsPreservesActiveIdentity|TestRunApplicationServiceReconcilesEphemeralWorkersAtStartup|TestMissionFallbackPreservesResourceAndFailureProvenance|TestMissionModelFallbackPersistsUnavailableModelProvenance|TestSchedulerUsesStableResourceIdentityTieBreak|TestMissionRunnerRequiresRealExecutorAndRespectsDependencies)$' -count=5
go vet ./internal/nexus/... ./internal/control/...
```
