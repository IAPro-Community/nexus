# Production finalization baseline

## Candidate

- Task: certify the current Nexus Maximum Delivery branch as a release
  candidate by closing production blockers.
- Scope: Windows, CI same-SHA, browser, security, updater, documentation,
  provider/runtime behavior, Maestro independence, and release tooling.
- Non-goals: new product features, redesign, merge to `main`, deployment,
  force-push, or changing production secrets.

## Safety boundaries

- Preserve existing user changes and public compatibility.
- Do not weaken or remove CI assertions.
- Do not create, commit, or expose production signing secrets.
- Do not perform irreversible data migrations or destructive Git operations.
- Rollback point: candidate SHA recorded by the discovery command and the
  current working-tree status before this campaign.

## Required evidence

Every platform or release claim must identify the exact candidate SHA and the
command or native CI run that produced it. Unsupported native execution is
recorded as `NOT VERIFIED`, never promoted from a cross-build or mock.

## Current phase

Phase 8 — final local verification and evidence report. Native Windows/macOS
execution, same-SHA hosted CI, authenticated Mission evidence, and the
production updater public key remain explicitly unverified or pending.

## Evidence collected

- Linux quality, build, vet, race, security, update-gate, frontend build/tests,
  browser hardening, accessibility smoke, visual smoke, and Linux Wails build
  were executed locally during this campaign.
- Independent backend, frontend, security, and Windows/release reviews were
  collected; a final red-team review was requested and its availability is
  recorded in the final report.
- Windows/macOS native gates and hosted same-SHA CI were not executed from this
  environment and are not represented as passing.
