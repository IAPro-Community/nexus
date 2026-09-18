# NEXUS HUMAN INTERVENTION SAFETY REPORT

## Baseline

```text
initial SHA: e28e38c6380cb639d673231449810d151adf2678
final SHA:   e28e38c6380cb639d673231449810d151adf2678
branch:      feat/nexus-maximum-delivery
```

The final SHA is unchanged because no commit or push was created. The worktree
contained pre-existing staged and unstaged changes; they were preserved.

## P0 reproduction

`internal/nexus/runner/intervention_test.go:31` uses a deterministic executor
spy. Before the fix, the scenario persisted `DispatchIntent`, resolved the
human intervention, and `ExecuteNextStep` emitted the provider operation a
second time. The baseline test failed with:

```text
unknown external outcome must remain blocked instead of redispatching
FAIL
```

After the fix, the same test passes and asserts the side-effect counter remains
`1` after resolution and resume.

## Architecture change

| Before | After |
| --- | --- |
| Free-form decision/chosen values | `interventionId + version + optionId`, with options owned by Core |
| In-memory/implicit resolution | Durable `InterventionResolution` and deterministic semantic key |
| Save then publish/schedule | SQLite transaction persists run, resolution, and resume outbox intent before worker scheduling |
| `INTENT` could be cleared on resume | `INTENT` becomes `UNKNOWN_EXTERNAL_OUTCOME`; no automatic redispatch |
| Generic package reset | Explicit package-scoped operation; unrelated dispatch IDs and receipts are preserved |
| Worker map cleanup by mission only | Worker generation identity plus compare-and-delete and cancel/join handoff |
| Blocked state could lack action | `BLOCKED_NEEDS_USER` requires a versioned intervention with typed options |

## Invariants

| Invariant | Test | Result |
| --- | --- | --- |
| I1 blocked implies actionable intervention | `TestSaveRunRejectsBlockedMissionWithoutActionableIntervention` | PASS |
| I2 same decision is semantic-idempotent | `TestResolveIntervention_DoesNotRedispatchUnknownExternalOutcome`, `TestResolveIntervention_Idempotent` | PASS |
| I3 different option after resolution is rejected | `TestResolveIntervention_ConflictingDecisionIsRejected` | PASS |
| I4 stale version/ID has no mutation | `TestResolveIntervention_StaleRequestHasNoBusinessMutation`, `TestResolveIntervention_StaleVersionAfterResolutionIsRejected` | PASS |
| I5 unknown outcome is never redispatched | `TestResolveIntervention_DoesNotRedispatchUnknownExternalOutcome` | PASS |
| persisted completed dispatch checkpoint is never re-entered | `TestExecuteNextStep_DoesNotRedispatchCompletedDispatchCheckpoint` | PASS |
| I6 completed package is not re-executed | `TestResolveIntervention_PreservesParallelAndCompletedPackageIdentity` | PASS |
| I7 unrelated dispatch identity is preserved | `TestResolveIntervention_PreservesParallelAndCompletedPackageIdentity` | PASS |
| I8 resolution survives restart | `TestInterventionResolutionSurvivesStoreReopen` | PASS |
| I9 durable resume is one semantic continuation | `TestInterventionResolutionSurvivesStoreReopen` | PASS |
| I10 old worker cannot delete newer owner | `TestReleaseMissionWorkerCannotDeleteNewGeneration` | PASS |
| I11 executing run has recovery path | `TestInterventionResolutionSurvivesStoreReopen` via `RecoverMissionRuns` | PASS |
| I12 no-progress failure stays distinct | `TestExecuteNextStepUsesFailedNoProgressWithoutHumanBlocker` | PASS |

Additional policy coverage (`TestResolveIntervention_RejectsOptionOutsideAutonomyContract`
and `TestResolveIntervention_AllowsRetryOnlyAfterProvenPreDispatchFailure`) also
passes.

## Side-effect proof

The P0 spy starts with one already-sent operation:

```text
dispatch A: 1
resume requests: 1 semantic resolution / 1 durable outbox intent
external effect after decision: counter remains 1
```

The safe retry test starts at `FAILED_BEFORE_DISPATCH` and observes exactly one
new execution, proving fail-closed does not mean “never retry.”

## Restart proof

`TestInterventionResolutionSurvivesStoreReopen` performs:

```text
blocked run -> resolve -> close SQLite -> reopen SQLite
-> recover MissionRun -> execute continuation -> COMPLETED_VERIFIED
```

Observed result: resolution persisted; resume was `PENDING` after decision and
`COMPLETED` after recovery; two durable events were present
(`human_intervention.resolved`, `mission.resume_requested`); continuation count
was exactly `1`; duplicate resolution after reopen returned the same key and no
additional events.

## Race proof

```text
go test -race -count=1 ./internal/nexus/runner  PASS
go test -race -count=1 ./internal/nexus        PASS (38.4s)
go test -race -count=50 ./internal/nexus/runner PASS (96.4s)
```

The directed worker-generation test uses no timing sleep. It installs a newer
owner, releases the old owner, and verifies the newer owner remains registered.

## Compatibility

The Web endpoint now requires:

```json
{
  "intervention_id": "...",
  "version": 3,
  "option_id": "retry-safe-package",
  "resolved_by": "..."
}
```

Free-form `decision`/`chosen` payloads are no longer executable. The backend
returns explicit stable error codes for stale, conflicting, invalid-option,
policy-denied, and unknown-outcome cases. The Web client renders Core-owned
options and submits their stable IDs. Compatibility cannot reintroduce unsafe
retries.

## Verification

```text
go test -count=1 ./internal/nexus/runner       PASS
go test -race -count=1 ./internal/nexus/runner PASS
go test -count=1 ./...                         FAIL (unrelated AGY fixture mismatch in current dirty worktree)
go vet ./...                                   PASS
npm --prefix web run quality:full             PASS (66 suites, 336 tests, build)
git diff --check                               PASS
gofmt campaign files                           PASS
```

The full Go suite passed after the campaign's mechanical fixture cleanup, but a
later concurrent worktree change in `internal/core/provider/adapters/agy/agy.go`
made three existing AGY tests fail because incomplete/single-group TSV output
is now accepted. Nexus/runner/store and the scoped race gates remain PASS; the
AGY change was not modified because it is outside this campaign. The current
worktree-wide `gofmt -l` also reports the unrelated AGY file and
`internal/control/terminal/terminal_windows.go`; campaign files are formatted.

## Remaining gaps

- `UNKNOWN_EXTERNAL_OUTCOME` still requires explicit human reconciliation;
  provider-side reconciliation tooling is not introduced here.
- The durable outbox records the resolution and resume request. Delivery to a
  future generic notification system remains outside this gate.

Attention Center final UX, Project Intelligence expansion, provider UX/failover,
native platform campaigns, redesign, and Maestro changes remain out of scope.

## Verdict

```text
P0 CLOSED
```

This closes only the HumanIntervention/resume promotion gate. It does not
declare the broader Nexus release GO.
