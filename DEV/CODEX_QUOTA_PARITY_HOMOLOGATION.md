# Nexus 1.0.0 — Codex quota parity and account selection

## Problem reproduced from user evidence (2026-10-20)

`nexus usage` displayed a 25-day-old Codex estimate of 12% weekly remaining,
with a September reset date, while an active `/status` in the same isolated
profile reported 84% weekly remaining and a reset in October. Ranking multiple
Codex accounts using historical quota is unsafe.

## Corrective contract

- Codex observations older than **15 minutes** may remain in last-known history
  but MUST appear as **UNKNOWN**, with no historic percentages or reset deadlines
  presented as current capacity in TUI, Web and scheduler.
- `nexus usage` displays cached information immediately and then schedules an
  asynchronous refresh of the configured providers. `r` requests a refresh
  for the selected account and `R` refreshes all accounts. While the Usage
  TUI remains open, Codex profiles are re-probed every five minutes (off the
  UI event loop) after a previously locked profile becomes available.
- `nexus codex` defaults to supervised TTY mode. Both supervised and direct
  automatic launches now run the same identity-scoped live quota preflight
  before ranking Codex accounts. The total probe budget is ten seconds, shared
  across at most five simultaneous profile reads; explicitly requested
  profiles skip this work. No quotas or identity are shared across profiles.
- Subscription quota probes remove any inherited host `OPENAI_API_KEY` so the
  isolated Codex app-server cannot silently select an unrelated API identity.
  This applies only to the quota subprocess, not to provider runtimes.
- Fresh official `account/rateLimits/read` responses must come from the
  same isolated CODEX_HOME and matching chatgpt_account_id; a mismatch is
  discarded. Rate-limited responses and independent provider quota policies
  remain protected.
- A running interactive Codex TUI holds a maintenance lock. Do not launch a
  competing app-server process against its home. If the official probe is
  unavailable, NEVER claim the last-known quota is current.
- The Codex selector always prefers currently measurable eligible quota.
  When all quota is unverifiable, it may honor a configured project binding or
  explicit default account, with a reason that says quota is unverified. It must
  not infer the 'best' account from weeks-old numbers or choose an arbitrary
  unpinned profile.

## Linux validation (same branch/SHA)

1. Run `git checkout release/v1.0.0 && git pull --ff-only`.
2. Run `go test ./internal/core/quota ./internal/profile ./internal/core/scheduler ./internal/tui ./internal/core/provider/adapters/codex ./internal/app ./internal/nexus`. Run `go test -race ./internal/app ./internal/tui ./internal/core/provider/adapters/codex` on Linux where supported.
3. Build the local Nexus binary using the repository release build procedure.
4. Close *all* Codex interactive sessions before the first live probe; confirm
   the correct profile's isolated CODEX_HOME matches `nexus profiles --json`.
5. Launch `nexus usage`, select each Codex profile and press `r`.
   Compare account, percentage, reset and timestamp against `/status`
   in the *same exact* profile/session. Do not compare two different accounts.
6. With the Codex TUI still active in a profile, open a second `nexus usage`.
   Verify it does not steal the session lock, display a bogus fresh reading,
   or reuse expired percentages.
7. Seed a valid identity-scoped 25-day-old quota cache. Verify `nexus usage`
   displays no percentage from it and auto-routing ignores it.
8. Test a fresh high-capacity account against a stale high-capacity account:
   fresh wins. Test that an explicitly bound/default profile is selected if
   everything is UNKNOWN; no preference must fail closed with guidance.
9. Exercise 4–5 separate Codex profiles: logging out or switching one must not
   leak another profile's quota or identity. Check account_id rather than email
   alone.
10. Repeat on Windows/ConPTY and Linux/PTY with the same commit.

## Investigation if parity still fails

Inspect availability of `codex app-server`, the active CODEX_HOME and the
profile-specific auth.json, without printing tokens. Confirm the app-server
JSON-RPC `account/rateLimits/read` is supported and returns `accountId`
matching the profile. Check whether the interactive TUI lock prevents a
concurrent probe; stop the session and refresh again. `NEXUS_CODEX_APP_SERVER=0`
is the optional kill switch and will prevent this official read.

Use `NEXUS_DEBUG=1` only in a safe, private test environment. Redact logs
before sharing: homes, account IDs, email addresses, tokens, headers and
session contents must never be copied into public CI artifacts.

## Gate

This is a code fix, **not a claim of Windows/Linux certification**. GO requires
the above tests and fresh account-specific official rate-limit evidence.
