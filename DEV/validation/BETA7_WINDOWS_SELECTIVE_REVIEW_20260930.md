# Nexus 1.0 Beta 7 — Windows selective review and current blockers

Date: 2026-09-30

## Scope and release truth

- Canonical release branch: `release/v1.0.0`.
- Functional baseline reviewed: `0aba43fcddde8ab4aeb45df641ff2ca732002409` (`1.0.0-beta.7`).
- Review continuation on 2026-09-30 identified and corrected a canonical delegation default that still fell back to `AUTO` outside the interactive adapter. The release now fails closed to `OFF`; intentional orchestration paths (`nexus run`, Composer → Flow, explicit promotion) declare `AUTO`/`ASK` explicitly.
- Historical Windows source reviewed: `codex/windows-terminal-functional` at `d73d75afe8ad6b0e1f0ba1c1a20991839f64fb5c`.
- Historical branch preserved as `archive/2026-09-30/codex/windows-terminal-functional`.
- Pre-hardening release snapshot preserved as `archive/release-v1.0.0-beta7-pre-final-hardening-20260930`.

No merge, rebase, or bulk cherry-pick from the historical Windows branch was performed.

## CI failure 36593519307

The failing job was Frontend / Verify documentation. The failure was not a Windows runtime or Codex terminal failure.

Exact failing gate:

`bun scripts/docs-verify.mjs`

The verifier reported the visual manifest stale after changes under the watched Web paths. Release config, Linux E2E, cross-platform compile and security jobs in that run were green; downstream/native jobs skipped after the frontend failure are not considered PASS.

## Historical Windows branch classification

### Already present / superseded — do not port

- Windows process/window suppression is centralized through `internal/executil` and current launcher/runtime code.
- `.cmd/.bat` execution is handled by the current command resolver and `ComSpec`; npm `.cmd` shims are resolved to their real target when possible.
- Windows process supervision uses the current Job Object/lifecycle implementation.
- UTF-16 environment block, process creation, resize, wait/exit-code and cleanup behavior exist in the current terminal backend.
- ConPTY remains opt-in via `AI_CLI_ENABLE_CONPTY=1`; safe pipes remain the default.
- Launcher, registry cleanup and host lifecycle have materially evolved beyond the historical branch.
- `install.ps1` is substantially newer than the historical implementation.
- Old runtime helpers such as `windowsShellCommandLine` / `quoteWindowsArg` are superseded by `internal/runtime/executable.go`.

### Regressions that must remain discarded

- Any change making ConPTY mandatory/default.
- Any code that bypasses the current terminal input/output scrubbers.
- Any behavior that lets `--lead` imply Mission authorization.
- Any default that enables LineShadow/Promoter for a normal interactive provider session.
- Any replacement of current process/Job Object/registry generation safety with the older implementation.

### Exclusive valid code to port

No exclusive functional Windows symbol or behavior was found that is still missing from the current Beta 7 architecture in the reviewed scope. Therefore no Windows code was ported from `d73d75a`.

## Codex regression boundaries verified in source

Current release source preserves:

- interactive supervised sessions with delegation `OFF` by default;
- `--lead` binds Lead identity/context but keeps delegation `OFF`;
- `OFF` bypasses LineShadow/Promoter and uses byte-transparent attach;
- invalid/missing interactive delegation mode fails closed to `OFF`;
- `CODEX_TUI_DISABLE_KEYBOARD_ENHANCEMENT=1`;
- deep Unix `CODEX_HOME` TUI paths add `--no-daemon` when the app-server socket would be unsafe;
- `resume`/interactive TUI are covered while headless/service subcommands retain native daemon semantics;
- SGR mouse input `CSI <button;x;y M/m` is discarded;
- DEC mouse/focus modes including 1000/1002/1003/1004/1005/1006/1007/1015 are filtered from provider output;
- split CSI/SS3 selector sequences are coalesced before forwarding.

Existing focused tests cover the no-daemon rules, delegation labels/defaults, SGR mouse filtering and split arrow/SS3 behavior.

## Documentation correction

`docs/reference/configuration.md` incorrectly described `NEXUS_DELEGATION_MODE` as defaulting to `AUTO`. Interactive provider sessions actually default to `OFF`. The public reference was corrected in `964edb6` to match the runtime contract.

## Canonical delegation default hardening

The interactive provider path was already correct, but the shared `NormalizeDelegationMode`
and `StartDispatch` still defaulted an omitted mode to `AUTO`. That created an
architectural inconsistency: a caller outside the interactive attach path could omit
the policy and regain automatic delegation.

The release now enforces:

- empty/unknown delegation mode → `OFF`;
- explicit `OFF` → direct/provider-owned behavior;
- explicit `ASK` → approval-gated delegation;
- explicit `AUTO` → automatic delegation where the caller intentionally selected orchestration;
- `nexus run` and Composer → Flow pass `AUTO` explicitly instead of depending on a hidden default;
- ordinary `StartDispatch` without a mode is rejected by the dispatch brake rather than silently creating a Mission.

Regression tests were added/updated for the fail-closed normalizer, implicit dispatch
rejection, explicit autonomous run/Flow paths and multi-step decomposition.

## Current visual-manifest blocker

The current screenshot manifest is anchored at:

`102213a5436f0f4f53aed5763949ce25460ec2b8`

The current release contains later changes under the verifier's watched Web paths. At least one is a real presentation change: `ComposerStart.tsx` adds the visible Interview Depth selector (QUICK / STANDARD / DEEP).

The versioned `docs/assets/screenshots/composer.png` does not contain that selector. Therefore changing only `manifest.source_sha` would fabricate visual evidence and is explicitly rejected.

Required resolution:

1. run the isolated synthetic docs capture against the final candidate;
2. regenerate the affected screenshots, at minimum Composer;
3. update the visual manifest with the exact source SHA produced by the capture;
4. rerun `bun scripts/docs-verify.mjs` and the browser/Axe/visual gates on that same candidate.

Until that happens, documentation/visual verification remains FAIL by design.

## Validation status

This review is source/evidence based. It does not claim native Windows, Linux, macOS, Browser E2E, Wails Desktop/HUD or full race-suite PASS for the new final SHA.

The release must remain prerelease/fail-closed until exact-SHA gates and native smokes are recorded.
