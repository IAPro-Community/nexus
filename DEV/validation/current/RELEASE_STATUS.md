# Current release status

STATUS: NEXUS_1_0_CLOSURE_IN_PROGRESS
SOURCE_VERSION: 1.0.0-beta.4
BASELINE_MAIN_SHA: 7605f702cce220b4111d51d06a34b37f7e0a9a8d
BASELINE_PR3_HEAD: 300ccbeee4f145f80ebce13c9cefafcb01acefc3
BASELINE_CI_RUN: 35547298239 (#149)
BASELINE_MAIN_PUSH_CI_RUN: 35547840452 (PARTIAL / NON-GREEN)
LATEST_PUBLISHED_BETA_TAG: v1.0.0-beta.1
UPDATED_AT: 2026-09-22

## Current truth

PR #3 was merged into `main` as
`7605f702cce220b4111d51d06a34b37f7e0a9a8d` after same-SHA CI #149 completed
successfully on the PR head `300ccbeee4f145f80ebce13c9cefafcb01acefc3`.

The verified PR-head gates were:

1. Frontend format/lint/styles/typecheck/unit/build/verify — PASS
2. Linux Go E2E, full race suite, vet and golangci-lint — PASS
3. Security / govulncheck — PASS
4. Browser hardening — PASS
5. Surface Isolation + terminal continuity — PASS
6. Capacity API/browser E2E — PASS
7. Axe and visual smoke — PASS
8. Desktop Linux Wails build — PASS
9. GoReleaser Linux snapshot — PASS
10. DEB install smoke — PASS

The source version currently reports `1.0.0-beta.4`. This does NOT mean a public
`v1.0.0-beta.4` tag or release has been certified or published. Repository tag
inventory currently exposes `v1.0.0-beta.1` as the published beta tag, so install
documentation that explicitly pins that tag is an artifact reference rather than a
claim that beta.1 is the current source version.

The post-merge `main` push CI run `35547840452` is also retained as evidence:
Linux and Security passed, Frontend was cancelled, and dependent jobs were skipped.
That run is therefore NON-GREEN/PARTIAL and must not replace the fully green PR-head
CI #149 baseline.

## PR merge-result identity

GitHub pull-request CI is intentionally merge-result evidence: the workflow run is
associated with the PR head, while the default checkout executes GitHub's synthetic
merge commit for that head against the target branch. This validates mergeability but
must not be described as literal exact-HEAD execution.

Exact-SHA release certification is a separate gate: the final release/tag workflow must
run and pass on the immutable release SHA itself. Artifact provenance and stable-release
claims must use that exact release SHA, not a pull-request synthetic merge SHA.

## Active release program

The active program is Nexus 1.0 Product Closure on
`feat/nexus-1.0-closure`. The canonical contract is
`DEV/SPECS/NEXUS_1_0_CLOSURE.md`.

The program must close, with evidence:

- production Capacity probes;
- Agent Modes and canonical AgentMatcher;
- Skills v1;
- cost attribution, budgets and anomaly detection;
- unified Work Mode;
- Mission 1.0 live certification and Evidence Bundles;
- Memory 1.0 on the existing IntelligenceMemory;
- Maestro contract v1 and graceful degradation;
- server-side entitlement enforcement for future paid capabilities;
- native Linux/Windows/macOS release certification;
- signed install/update/upgrade/uninstall evidence;
- real dogfooding and soak.

## Platform truth

Linux has fresh same-SHA evidence for the PR #3 baseline described above.

Windows and macOS source/build support exists, but the Nexus 1.0 candidate still requires
fresh native same-SHA execution evidence. Cross-compilation alone must not be described
as native certification.

## Distribution truth

The repository is private and the intended beta/customer distribution path is binary
artifacts, not source access. Release/update infrastructure exists, but a stable 1.0
release is NOT authorized by this document.

Before any stable release:

- final candidate CI must be green on the exact release SHA;
- native platform gates must be complete;
- update manifest/checksum/signature chain must be verified;
- install/upgrade/uninstall must be executed on supported platforms;
- release artifacts must be produced from the final clean candidate;
- external signing/notarization limitations must be stated truthfully.

## Stable certification contract

Stable release now has an executable certification boundary rather than a checklist-only
claim. `.github/workflows/stable-certification.yml` must complete successfully on the
exact candidate SHA before a non-prerelease version can pass `release.yml`.

The stable certification performs:

- exact-SHA and canonical update trust-root checks;
- exact-SHA candidate build and release/update contract tests;
- authenticated direct-provider probes;
- a black-box autonomous Mission that must end in `COMPLETED_VERIFIED`;
- an 8-hour process-level Web soak with HTTP health probes;
- a deliberate Web restart during the soak;
- a second provider/Mission checkpoint after the soak window.

The historical in-memory `TestOvernightSoakHarnessBounded` remains a state-machine
test only and is not accepted as stable-release soak evidence.

## Closure hardening — 2026-09-22

The closure branch was re-audited against the executable product and release
boundaries. The following source-level blockers were corrected rather than
deferred to certification:

- human confirmation of an unknown external dispatch no longer fabricates
  `VERIFIED`; the package returns to `TESTING` and must produce real verification
  evidence;
- Evidence Bundle no longer treats an already resolved human intervention as
  unresolved evidence;
- Composer Stop now aborts the active request, including attachment transfer, and
  asynchronous errors are announced accessibly;
- Composer narrow-layout overflow, mobile touch targets and the full theme/device
  visual matrix were hardened;
- CLI `nexus run` / `plan run` now remain alive with the canonical Mission worker
  instead of abandoning an in-process goroutine; terminal/blocked/failure states
  preserve non-zero CLI semantics;
- project registration can explicitly request `--isolation worktree`, avoiding an
  autonomous-Mission/default-isolation mismatch;
- GoReleaser now defines the CLI artifact matrix for Linux, Windows and macOS and
  packages Windows archives as ZIP;
- Windows and Unix uninstallers are explicit/idempotent, preserve data by default,
  and the lifecycle includes DEB removal;
- the updater trust root and key ID have versioned sources of truth
  (`UPDATE_PUBLIC_KEY`, `UPDATE_KEY_ID`) and release secrets must match them;
- install scripts verify detached Ed25519 manifest signature, signed artifact digest
  and embedded binary version before replacement;
- the macOS installer correctly extracts the verified Desktop ZIP/`.app`; unsupported
  Desktop architectures no longer turn a valid CLI install into a failure;
- `nexus update check` now fails closed on registry/signature errors, update apply
  failures return a non-zero result, and channel migration is explicit via
  `--channel beta|stable`;
- alpha/beta/RC channel classification is consistent between signer, updater and
  release workflow;
- stable certification now exercises signed install, installer upgrade, self-update,
  tamper rejection, uninstall/purge, explicit beta→stable promotion, authenticated
  provider/Mission execution and an 8-hour cumulative process-level soak with restart;
- release provenance rejects pull-request synthetic-merge CI as exact-SHA evidence;
  only exact-SHA `push` or `workflow_dispatch` CI can authorize artifacts;
- public installation/package documentation now matches the private-source /
  binaries-only distribution model and no longer advertises RPM/NSIS/winget as
  produced v1 artifacts.

The 8-hour soak is split into two four-hour execution segments because GitHub limits an
individual Actions step to 360 minutes. Both segments run on the same exact candidate
SHA and retain separate evidence.

### Evidence still external

These corrections do **not** convert the candidate to GO by themselves. The current PR
Actions runs are reaching `pending` without creating jobs; previous superseded runs
were cancelled by workflow concurrency. Therefore there is still no fresh native
Linux/Windows/macOS, browser, security, release snapshot or Desktop execution evidence
for the current closure head.

Stable remains fail-closed until:

1. Actions can actually schedule the required jobs;
2. exact-SHA non-PR CI passes every required platform/release gate;
3. the protected signing environment provides the canonical key/key-id pair;
4. stable certification completes the signed lifecycle and both four-hour live-soak
   segments on that exact SHA.

## Current verdict

`NEXUS_1_0_CANDIDATE = NO_GO`

Reason: the 1.0 closure implementation and native release certification are still in
progress. This is not a regression verdict for the PR #3 baseline: the PR-head
same-SHA evidence is green, while the later `main` push run is explicitly recorded as
partial/non-green. The verdict here is for stable 1.0 release readiness.
