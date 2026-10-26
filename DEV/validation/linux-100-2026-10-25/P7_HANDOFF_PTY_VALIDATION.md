# P7 — Handoff, PTY, and concurrency validation

Date: 2026-10-25  
Branch: `release/v1.0.0`  
Commit base/remoto: `7a04f1b4cf5950b97b4ea1e156dda31ac513823b`  
Product source fingerprint: `bb575423508a0c773580804a476b0888ff2449135045a74f80aa6efaffedc6e1`  
Source manifest: `logs/source-provenance-p6-agent-fallback-2026-10-25.txt` (1,421 files)

## Automated process and concurrency coverage

| ID | Area | Procedure | Expected | Actual | Status | Evidence | Severity | Commit tested |
|---|---|---|---|---|---|---|---|---|
| P7-001 | Full Go race suite | `go test -race -count=1 ./...` | All Go packages complete without a race report. | Exit 0 across all 82 package result lines; no `DATA RACE`, panic, or failed package. Slow packages: `internal/nexus` 261.558 s, `internal/nexus/store` 79.035 s, `capacitymonitor` 25.074 s. | PASS | `logs/go-race-p7-full-repository.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P7-002 | PTY attach/detach and writer lease | `go test -race -count=1 -timeout=5m ./internal/control/host -run '^(TestQA_RapidAttachDetachSpam|TestQA_TwoWritersLeaseHandover)$'` | Concurrent attach/detach does not deadlock; a surviving writer can send after the prior writer disconnects. | Both child-process host tests passed; host stayed responsive and writer B sent after writer A disconnected. | PASS | `logs/go-race-p7-host-lease.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P7-003 | Process ownership/replacement | Race tests for orphan host reap, stale registry ownership, in-place provider binary replacement, and child termination. | Stale/unregistered hosts are cleaned safely; a replaced binary is not mistaken for the old runtime; child processes terminate. | All focused launcher/registry regressions passed. | PASS | `logs/go-race-p7-process-ownership.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P7-004 | Resume and durable runner recovery | Race tests for resume continuity predicate, handoff state transitions, live lease renewal, stale lease rejection, and refusing unknown in-flight dispatch after restart. | Only a matching live process/session passes the continuity predicate; uncertain in-flight work is not replayed. | Focused tests passed. `VerifyResumeContinuity` only verifies process liveness and matching resume arguments; it is not provider acknowledgement. Runtime state remains `NATIVE_RESUME_UNVERIFIED`. | PASS | `logs/go-race-p7-handoff.log`; `logs/go-race-p7-runner-recovery.log` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P7-005 | Live AGY runtime response | `nexus run agy kivervinicius-gmail -- --print '<safe READY prompt>'` in a temporary workspace. | An authenticated provider call returns the bounded expected response without editing the repository. | AGY returned exact `READY`, exit 0, in 8.2 s. The prompt was print-only; response bytes were suppressed from logs. | PASS | `logs/live-provider-smoke-p7-agy-secondary-profile.txt` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P7-006 | Live AGY exhausted-quota path | Explicit-profile print smoke using `agy:kiveromegasistemas`; inspect only profile auth/quota status. | A quota-exhausted authenticated profile fails visibly; it is not reported as a successful model response. | Profile auth was healthy, but cached quota was exhausted and provider returned HTTP 429 `RESOURCE_EXHAUSTED`; Nexus exited non-zero and surfaced the unavailable-resource condition. No login was attempted. | PASS | `logs/live-provider-smoke-p7-agy-quota-profile.txt` | — | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P7-007 | Live same-provider profile handoff | Authenticated different-profile resume with a provider acknowledgement. | Target confirms the source conversation resumed in the target account. | BLOCKED: the available Codex profiles already had active user runtimes and were not interrupted; the AGY source profile with a suitable alternate account had an exhausted target profile, and AGY does not advertise cross-account resume. No unsafe profile switch was attempted. | BLOCKED | `logs/live-provider-smoke-p7-agy-quota-profile.txt`; `logs/live-provider-smoke-p7-agy-secondary-profile.txt`; `./nexus ps --json` was inspected with profile/state/PID only | P1 | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |
| P7-008 | Live cross-provider checkpoint continuation | Start a live source, checkpoint it, resume through a different authenticated provider, and verify target response. | The target consumes the checkpoint and produces a verified answer; source is quiesced. | BLOCKED: current OpenCode smoke did not produce a final answer within 90 s and emitted tool-use events despite the bounded prompt. The process group was terminated; the isolated workspace remained empty and no Nexus AGY/OpenCode runtime remained. Existing Codex runtimes were left untouched. | BLOCKED | `logs/live-provider-smoke-p7-opencode-target.txt`; temp workspace `/tmp/nexus-homologation-p7-20261025` was verified empty | P1 | `7a04f1b4cf5950b97b4ea1e156dda31ac513823b` + overlay `bb575423…` |

## Interpretation

The full race gate and deterministic process/lease/recovery regressions pass. One authenticated AGY profile returned a real bounded answer; another returned a clear quota error. This does not validate same-provider resume or cross-provider checkpoint continuation. `NATIVE_RESUME_UNVERIFIED` remains the honest state until a provider confirms the session ID and continuation. P7 is **PARTIAL/BLOCKED**; P2, P7 live handoff, P8, P9, native/distribution gates remain open and the release verdict is `NO_GO_LINUX`.

## Reproduction

```bash
go test -race -count=1 ./...
go test -race -count=1 -timeout=5m ./internal/control/host -run '^(TestQA_RapidAttachDetachSpam|TestQA_TwoWritersLeaseHandover)$'
go test -race -count=1 -timeout=5m -run '^(TestReapOrphanHostsFromKillsUnregisteredHost|TestCleanupStaleUsesHostPIDWhenChildSurvives|TestBinaryFingerprintDetectsInPlaceReplacement|TestKillProcessTerminatesChild)$' ./internal/control/launcher ./internal/control/registry
go test -race -count=1 -timeout=5m -run '^(TestVerifyResumeContinuity|TestHandoffStates_HappyPath|TestAccountHandoffValidation)$' ./internal/control/handoff
go test -race -count=1 -timeout=5m -run '^(TestMissionRunnerRestartRefusesUnknownInFlightDispatch|TestMemoryRunRepositoryRejectsLiveLeaseReacquireBySameOwner|TestMissionRunnerRenewsLeaseDuringLongProviderOperation)$' ./internal/nexus/runner
```

Live provider commands are intentionally omitted from the reproduction block because they use the configured user's account and quota. Sanitized outcomes are recorded in the evidence logs above.
