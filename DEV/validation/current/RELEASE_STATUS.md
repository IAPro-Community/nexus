# Current release status

STATUS: NEXUS_1_0_BETA6_CANDIDATE_UNVERIFIED
SOURCE_VERSION: 1.0.0-beta.6
CANDIDATE_REF: release/v1.0.0
PUBLIC_GITHUB_RELEASE: NONE
LATEST_SOURCE_BETA_TAG: v1.0.0-beta.1
CI_STATE: UNAVAILABLE_BEFORE_JOBS_ACCOUNT_CREDITS
INTEGRATION_TRACKER: #27
INTEGRATION_BASE_SHA: PENDING_FINAL_BETA4_SHA
UPDATED_AT: 2026-09-27

> **Current release truth moved to `docs/release-status.md`.** This DEV file
> remains an evidence ledger. The 2026-09-23 SHA/run records below are historical
> and must not override the current `VERSION`, candidate ref, platform policy,
> tag inventory, release inventory or current CI state.

## Canonical beta integration status — 2026-09-27

`release/v1.0.0` is the only integration line for Nexus 1.0 retesting. Beta 6 is the current internal candidate. Wails v3 has already been selectively ported onto this line while preserving newer Agent Projection, ExecutionRole, Direct/Mission, OpenCode and homologation contracts. Future beta increments change the internal version/manifest, not the release branch.

The authoritative open-gap checklist is GitHub issue #27. Source integration remains **UNVERIFIED** until native same-SHA evidence closes the required platform gates.

## Current candidate summary

- source version: `1.0.0-beta.6`;
- candidate ref: `release/v1.0.0`;
- source matrix integrated: Linux amd64 + Windows amd64 + macOS arm64;
- public binary release inventory: empty in both the private source repository
  and `IAPro-Community/nexus-releases`;
- `v1.0.0-beta.1` is a historical source tag, not a published binary release;
- current GitHub Actions runs terminate before jobs because account credits are
  unavailable; this is neither PASS nor functional FAIL;
- beta publication remains fail-closed until local homologation plus the
  exact-SHA CI/native certification policy defined by the current workflows.

For actionable release state, gates and commands use
[`docs/release-status.md`](../../../docs/release-status.md).

## Historical 2026-09-23 closure evidence

PR #4 (`feat/nexus-1.0-closure`) was squash-merged into `main` on 2026-09-23.
The source-closure head `0d6e9c667af39b7534002d060848acfc9eb8e265` passed CI #668 before merge.
The resulting `main` SHA `5a61d3152b23b002b39e8b4b8988924db39956b2` then passed exact-SHA push CI #669
(run `35849031522`) with the Linux amd64 v1 matrix green: Frontend, Linux E2E,
Security, Release configuration, Browser E2E/Axe/visual, Desktop Linux Wails,
GoReleaser Snapshot and Smoke DEB. Windows amd64 and macOS arm64 remained
DEFERRED / UNVERIFIED and were skipped by policy, not counted as PASS.

CI #669 produced exact-SHA artifacts named with `5a61d3152b23b002b39e8b4b8988924db39956b2`.
No stable `v1.0.0` publication is authorized yet: Stable Certification, signing,
authenticated provider/Mission checkpoints and the live 8-hour soak are still pending.
Any commit after `5a61d315...` creates a new candidate SHA and requires fresh exact-SHA
CI before Stable Certification can authorize publication.

### Historical baseline

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

The final candidate source version now reports `1.0.0`. This does NOT mean a public
`v1.0.0` tag or stable release has already been certified or published. Repository
tag inventory still exposes `v1.0.0-beta.1` as the published beta tag, so installation
documentation that pins beta.1 remains an artifact reference until stable certification
authorizes the final tag.

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

The source-closure PR has been merged. The active program is now the Linux amd64
stable-certification and publication path. The canonical closure contract remains
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
- native Linux amd64 release certification;
- Windows/macOS certification tracked as post-v1 platform expansion, not a v1 blocker;
- signed install/update/upgrade/uninstall evidence;
- real dogfooding and soak.

## Platform truth

The beta.4 source integrates **Linux amd64, Windows amd64 and macOS arm64**.

Public support remains evidence-bound: each platform advertised by a release must carry
fresh same-SHA native evidence for its applicable CLI/Core, Browser/Web, Desktop/HUD,
packaging and install/update lifecycle gates.

Windows and macOS are therefore **UNVERIFIED until native same-SHA certification passes**;
source integration is not itself a support claim. A public beta/stable release must not
advertise a platform whose required native evidence is missing.

Cross-compilation still must not be described as native certification.

## Distribution truth

The repository is private and the intended beta/customer distribution path is binary
artifacts, not source access. Release/update infrastructure exists, but a stable 1.0
release is NOT authorized by this document.

Before any stable release:

- final candidate CI must be green on the exact release SHA;
- every platform advertised by the release must complete its native same-SHA gates;
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
- GoReleaser v1 publication scope is Linux amd64 only; Windows/macOS source support
  remains in-tree but is not published or advertised as certified v1 support;
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

The source-level closure and CI #669 are not equivalent to a published stable release.
Before stable `v1.0.0` can be published, the final immutable candidate SHA still needs:

1. green exact-SHA Linux/Web/release-preparation CI on that final SHA;
2. the protected signing environment with the canonical key/key-id pair;
3. successful Stable Certification on that same SHA, including signed lifecycle,
   authenticated provider/Mission checkpoints and both live-soak segments.

CI #669 is exact-SHA evidence for `5a61d3152b23b002b39e8b4b8988924db39956b2` only.
If this status synchronization or any later change is merged, #669 becomes historical
post-merge evidence and the new `main` SHA must pass exact-SHA CI again before certification.

Windows/macOS remain blockers for any release that advertises them until their native same-SHA certification is complete.

## Current verdict

`NEXUS_1_0_CANDIDATE = NO_GO`

Reason: beta.4 source closure is merged, but this newer candidate includes changes after
the historical CI #669 baseline. The final immutable candidate still needs fresh exact-SHA
homologation/certification for every platform it advertises, plus Stable Certification.
Windows/macOS remain UNVERIFIED until their native same-SHA evidence is complete.
