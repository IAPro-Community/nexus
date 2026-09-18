# Real-provider Mission evidence

Status: `PASS` for an autonomous Nexus Mission on this worktree.
Observed: `2026-09-14T02:43:47Z` (Linux amd64, local build).

Earlier direct probes against the local provider inventory did not produce a
durable Mission (AGY was cached/expired and the first Codex attempt was held
by maintenance locking):

- `./nexus codex:kivergmail --direct --print ...` — transient maintenance lock
  while the existing supervised Codex runtime was active.
- `./nexus codex:kiver.omegasistemas --direct --print ...` — same transient
  maintenance lock; the profile is also expired in the local profile store.
- `./nexus agy:kiveromegasistemas --direct --print ...` — provider denied a
  command permission in headless mode and emitted no model output.
- An isolated `nexus run` attempt first hit strict admission because the
  temporary project was not configured for worktree execution. A follow-up
  profile-copy attempt could not complete: the profile tree contains large
  build caches and exhausted the temporary filesystem before the Mission
  started. The temporary copy was stopped and removed; original profiles were
  untouched.
- A minimal AGY profile copy with the local OAuth artifact was recognized as
  `authenticated=true` but `Token expired (refreshable)` and still failed
  strict resource admission before dispatch. No provider request or Mission
  run ID resulted.
- A supervised `./nexus codex:kivergmail --yolo` TTY attempt emitted only the
  startup banner and then failed with the new transient lock error because an
  existing process held an exclusive legacy/profile lock. It did **not** emit
  the removed “já está em execução” singleton message; the attempt was stopped
  by its timeout and no runtime was left in the local registry.

The corrected local binary then executed an authenticated Codex Mission through
the Web Control Center. Run `run_06G9VJV1K6J649PN2EMEHR1EMG` routed to
`codex:kivergmail` with `healthy=true`, `authenticated=true`,
`quota_available=true`, and `fallback=false`. The implementer runtime used a
dedicated worktree, changed `README.md`, and persisted one context capsule and
one work receipt. Package verification (`git diff --check`, `git status
--short`) and the global Definition of Done (`git diff --check`) passed. A
distinct reviewer Agent used the read-only `codex exec` path and returned an
approved structured verdict. The Mission reached `COMPLETED_VERIFIED`.

The earlier run `run_06G9V748P7JYWN940YG7ETGCNG` remains historical evidence of
the pre-change review capability and is retained above for traceability.

The `nexus` executable on `$PATH` was not used as evidence because it is an
older installed binary and still emits the pre-concurrency singleton message.
