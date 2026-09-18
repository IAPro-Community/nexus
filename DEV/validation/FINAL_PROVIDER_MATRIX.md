> **HISTORICAL — NOT CURRENT RELEASE EVIDENCE**
>
> Canonical current status: `DEV/validation/current/RELEASE_STATUS.md`.

# Final Provider Matrix

> Historical snapshot. For the current closure decision use the canonical
> `ValidationEvidenceStream` projections and `./nexus doctor --json` recorded
> at HEAD `42c22137a4a57ff6b6b80df8125b5a138f32b9e1`. Binary installation is
> not authenticated Mission evidence; current release verdict is `NO-GO`.

The current doctor snapshot was refreshed at `2026-09-12T03:34:53Z`. The
2026-09-12 retry used the explicit stable-root harness mode and canonical
Codex home/profile lock so Codex was not placed under `/tmp`; it still remained
at `model: loading`. AGY reached
its runtime but reported `not signed in`. Both remain `UNVERIFIED`.

| Provider | Binary probe | Authenticated real E2E |
|---|---|---|
| Codex | INSTALLED (`0.154.0`); direct host marker PASS; usage LIVE for two configured profiles | Nexus isolated runtime UNVERIFIED (`model: loading`) |
| Claude | INSTALLED (`2.1.250 (Claude Code)`) | NOT_AUTHENTICATED/NOT_TESTED |
| Gemini | INSTALLED (`0.28.2`) | NOT_AUTHENTICATED/NOT_TESTED |
| AGY | INSTALLED (`1.2.1`); registered/authenticated status; usage UNKNOWN | Nexus runtime UNVERIFIED (provider reported not signed in) |
| OpenCode | INSTALLED (`1.18.30`); pending auth | UNVERIFIED |
| Cursor | INSTALLED; non-interactive version probe hung | UNSUPPORTED FOR THIS RUN |

No provider is represented as functional based on binary presence alone.
The direct Codex host marker only proves host provider availability; it is not
proof of Nexus profile isolation, Mission completion, failover or evidence
ledger emission.
