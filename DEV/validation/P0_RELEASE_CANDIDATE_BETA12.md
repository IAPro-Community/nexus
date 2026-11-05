# Nexus P0 candidate — release/v1.0.0-beta.12

**Release readiness: NO-GO.** This is the only active Nexus Composer/Flow
development branch. The base is `release/v1.0.0`; source versions were
incremented together in `VERSION` and `web/package.json`.

The release combines the complete changes from PR #207
(Composer handoff provenance and invalidation of obsolete preview requests)
and PR #208 (persisted routing choices, explicit Flow draft save, no
Maestro-owned execution fallback). Both source heads are Git merge ancestors
of this branch and all **nine changed source/test blobs** were verified
identical after consolidation.

The original PRs were closed **as superseded, not as merged to the old base**.
Future changes related to Nexus–Maestro P0 must be committed only to this
release branch.

## Fast local QA

Use the **single canonical procedure** maintained in Maestro:
[Maestro P0 consolidated candidates](https://github.com/IAPro-Community/Orquestrador-Maestro/blob/release/v1.1.2/docs/release/P0_CONSOLIDATED_CANDIDATES.md).

From a clean checkout of Maestro `release/v1.1.2`, with this Nexus checkout
at `release/v1.0.0-beta.12`:

```sh
node scripts/p0-release-qa.cjs --nexus=../nexus
```

Run `--full` for the wider deterministic regression suite. The command
reports actual test results against exact source SHAs. **No mocked provider
output is certified as real execution.**

## Visual check — original cause of release quality failure

The old screenshot manifest uses source
`517a4fa7d6449d51c1a24bb74bfd7c10384608ae` and is stale after
substantial changes to Browser/Agent/Composer/Flow surfaces. The new release
branch cannot be approved using that manifest.

On an OS with the Nexus build dependencies and Playwright Chromium installed,
run the Maestro QA command with `--capture`, review the generated screenshots,
and commit only the genuine synthetic captures/manifest to **this release
branch**. Do not write a fabricated SHA or change screenshot statuses by hand.

This is separate from native acceptance on real Linux/Windows/macOS devices.

## Functional blockers

- Maestro `runs.submit` remains disabled, so **actual** AGY→OpenCode
  multi-task execution is not yet a supported Flow feature.
- The Nexus editor is a reviewed plan projection, not a duplicate scheduler.
- No native-device signed evidence bundle or independent human acceptance is
  attached to this candidate yet.
- The 48-obligation matrix from issue #115 remains mandatory for release GO.

The cleanup manifest for merged/superseded work branches is kept in Maestro;
remote ref deletion is a separate authenticated step after source verification.
