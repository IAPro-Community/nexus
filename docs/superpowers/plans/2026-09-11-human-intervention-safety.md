# Human intervention/resume safety plan

Date: 2026-09-11
Baseline: `e28e38c6380cb639d673231449810d151adf2678`
Scope: close only the Nexus P0 human-intervention/resume risk and the P1 contracts directly required for that flow.

## Diagnosis to preserve as regression evidence

The existing dirty implementation marks an intervention resolved, changes the
mission to `EXECUTING`, and clears every executing package's `DispatchIntent`.
That makes an external request whose outcome is unknown eligible for a second
provider execution. Its existing resolver also publishes resume only after a
normal in-memory save, has no typed option/policy validation, and starts a new
worker without an ownership handoff.

## Execution slices

1. Add a deterministic regression test with a provider side-effect spy that
   reproduces the duplicate dispatch after an unknown outcome. Add adjacent
   tests for completed and unrelated packages.
2. Inspect and extend the existing runner/repository/event abstractions with a
   typed intervention option/resolution, stable semantic idempotency key,
   version and scope checks, explicit autonomy-policy validation, and a
   durable resume request. Reuse existing event/store abstractions; do not add
   a parallel state machine.
3. Make unknown dispatch outcomes explicit and fail-closed. Preserve dispatch
   identity, receipts, completed packages, and unrelated package state during
   resolution. Allow retry only for a failure proven before dispatch.
4. Repair all transitions into `BLOCKED_NEEDS_USER` so they create an
   actionable intervention, while preserving `FAILED_NO_PROGRESS` as a
   distinct terminal state.
5. Make worker ownership generation/lease-safe and add deterministic
   replacement/recovery tests. Ensure `EXECUTING` has a durable worker or
   recoverable resume request.
6. Stabilize the application/API contract around
   `interventionId`, `version`, and `optionId`; keep any legacy mapping
   fail-closed and never allow free-form decisions to authorize work.
7. Add restart, duplicate HTTP/application, stale, conflicting decision,
   unknown dispatch, known-safe retry, parallel-package, and race tests.
8. Run focused tests first, then runner/Nexus/web packages, full non-race Go
   tests and vet. Run race tests and record any timeout honestly. Update only
   the durable DEV worklog/verify/handoff and this campaign report; do not
   implement Attention Center or unrelated product work.

## Safety invariants

- `BLOCKED_NEEDS_USER` always has an actionable intervention.
- Same intervention/version/option returns the same persisted resolution and
  schedules at most one semantic resume.
- A stale or conflicting decision has no mutation or side effect.
- Unknown external outcome is never reset or redispatched automatically.
- Completed and unrelated package dispatch identities remain unchanged.
- Durable decision and resume intent survive repository reopen/restart.
- Old workers cannot remove ownership of newer workers.
- `EXECUTING` has an owner or a durable recovery path.

## Verification evidence required

The final report will distinguish baseline reproduction from fixed-tree
evidence and will include side-effect counters, restart evidence, race command
results, compatibility impact, remaining gaps, and a P0-only verdict. No
release/GO claim is made by this plan.
