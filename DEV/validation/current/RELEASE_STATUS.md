# Current release status

STATUS: BETA_LINUX_CI_RECONFIGURED_PENDING_RUNNER_AND_HOSTED_EVIDENCE
VERSION: 1.0.0-beta.1
HEAD_AT_WRITE: e2f40a50dbcca78986796c2d5a16cac5e8a7d7ff
GENERATED_AT: 2026-09-17T00:00:00Z

The source repository remains private and the beta distribution is
closed-source. GitHub-hosted runners are blocked before job start by the
organization Actions budget/payment state. The red hosted run at the current
SHA is therefore infrastructure evidence, not a failed test result.

## Beta certification scope

This beta is Linux-only until native Windows and macOS runners are available.
The CI workflow requires seven same-SHA Linux gates:

1. Frontend (format, lint, styles, typecheck, unit tests and build)
2. Security (`govulncheck`)
3. Browser E2E, Axe and visual smoke
4. Linux E2E (race, terminal, runtime, web and build)
5. Desktop Linux Wails native build
6. GoReleaser snapshot (Linux archives and DEB only)
7. DEB install smoke

Windows E2E/ConPTY/Named Pipe, Windows Desktop/NSIS, macOS E2E/Desktop and
RPM packaging remain in source but are **UNVERIFIED** for this beta. They must
not be described as passed or shipped as beta artifacts.

## Remaining closure gates

1. Register the repository-scoped `nexus-ci` Linux self-hosted runner and
   prove it can start a minimal job.
2. Push one closure commit, obtain 7/7 green on its exact SHA, then rerun the
   Linux CI on that same SHA.
3. Configure `NEXUS_DIST_TOKEN` with least-privilege write access to
   `IAPro-Community/nexus-releases`; the release workflow fails closed without
   it. Ensure `beta-signing` permits `v*` tags.
4. Update the stale `v1.0.0-beta.1` tag only after a merge commit and a green
   `main` CI run, then run Signed Release and verify its manifest, signature,
   checksums, Linux CLI/Desktop artifacts, DEB and `channel-beta`.
5. `main` branch protection is unavailable on the current private-plan tier;
   use the documented maintainer-only merge checklist until that changes.

No release is authorized from this document. The existing remote beta tag
points to `cf2eb41` and is premature because that SHA also has a failed hosted
workflow record.
