# Nexus Production Finalization Report

## Current audit addendum — 2026-09-13

Current observed SHA: `eed65e5a7e46adc896165c94accb7fc73fc6750f`. The worktree
is not frozen, so this addendum is not a certification record. Local Go,
frontend, security, CLI build, Linux Wails build and updater gates passed;
`make docs-verify` is blocked by a stale visual manifest. Windows/macOS native
execution, hosted same-SHA evidence, production trust-root provisioning and a
real authenticated Mission remain unverified. The historical candidate below
must not be reused as evidence for the current SHA.

## Verdict

NOT_READY_FOR_OVERNIGHT_CERTIFICATION

## Historical candidate SHA

`e53f8352e4c3647632ebac7877cf0a6a3bd62647` — code candidate audited by this campaign.

## Git state

- Branch: `feat/nexus-maximum-delivery`
- Base: `origin/main` at `f71eb515278168e33d626fd631cd89dfb5e58faf`
- Merge-base: `1899ca6334576d859056d48a394e51d03758f313`
- Divergence at report time: 1 commit ahead / 88 commits behind `origin/main`
- Working tree: documentation/evidence changes plus pre-existing untracked validation artifacts remain; no push, merge, deploy or force-push was performed.

All executable evidence below was run against the candidate code state. Historical reports were not used as proof.

## Windows

| Gate | Evidence | Status |
| --- | --- | --- |
| Go build, vet and tests | Strict jobs defined in `.github/workflows/ci.yml`; no native Windows runner executed from this environment | NOT VERIFIED |
| ConPTY and Named Pipe E2E | Strict Windows job present; no native runner result | NOT VERIFIED |
| Runtime/Web E2E | Strict Windows job present; no native runner result | NOT VERIFIED |
| PowerShell installer syntax | `pwsh` unavailable locally; no native parser result | NOT VERIFIED |
| CLI smoke (`version`, `doctor`, `status`, `providers`, `profiles`, `projects`) | Windows smoke job defines available commands, but no native run | NOT VERIFIED |
| Windows binary and Wails desktop build | Workflow contains version/trust-root ldflags; no Windows build artifact for this SHA | NOT VERIFIED |
| NSIS package, install, launch, uninstall smoke | Workflow now builds and exercises install/uninstall paths; no native runner result | NOT VERIFIED |
| Windows paths, spaces, UTF-8, cleanup, restart/reattach, long paths | No native execution evidence | NOT VERIFIED |
| Diagnostics assertion | Strict outcome assertion is present; no native diagnostic log | NOT VERIFIED |

## Linux

| Gate | Evidence | Status |
| --- | --- | --- |
| Formatting, lint, typecheck, style lint, frontend tests | `make quality`; 68 frontend files and 350 frontend tests passed | PASS |
| Go tests | `go test -json -count=1 ./...`; 1,078 test actions and 49 package results passed | PASS |
| Race detector | `go test -race ./...` passed | PASS |
| Vet | `go vet ./...` passed | PASS |
| Security gate | `make security` / govulncheck reported no Go vulnerabilities | PASS |
| CLI build | `make build` passed; version `0.5.0-beta.23` | PASS |
| Linux Wails desktop package | `make build-desktop-wails` produced `cmd/nexus-desktop/build/bin/nexus-desktop` | PASS |
| Update gate | `bash scripts/verify-update-gate.sh` passed version, build, manifest, checksum, rollback and native archive-safety tests | PASS |
| Documentation links/assets | `make docs-verify` passed: 134 Markdown files and 10 required assets/files | PASS |

## macOS

| Gate | Evidence | Status |
| --- | --- | --- |
| Go build, vet, race and runtime/Web tests | Strict workflow definitions exist; no macOS runner result | NOT VERIFIED |
| Installer smoke | No native macOS install execution | NOT VERIFIED |
| Wails desktop build/package | No macOS artifact for this SHA | NOT VERIFIED |
| Paths, cleanup, restart/reattach and UTF-8 | No native execution evidence | NOT VERIFIED |
| Cloudflared checksum pin | Darwin digest is intentionally not configured until a real pin is supplied | NOT VERIFIED |

## Browser

| Gate | Evidence | Status |
| --- | --- | --- |
| Hermetic Chromium E2E | `npm --prefix web run test:e2e`; isolated data/project bootstrap, persistence, deep links and six breakpoints passed | PASS |
| Accessibility smoke | `npm --prefix web run test:a11y` passed required severity threshold | PASS |
| Visual smoke | `npm --prefix web run test:visual` passed the existing smoke script | PASS |
| Independent pixel-diff baseline | Existing visual script is a smoke check, not a pixel-diff comparator | NOT VERIFIED |
| Auth/session/WebSocket/reconnect browser paths | Covered by the executed hardening flow and Go web tests; no separate hosted browser deployment | PASS |

## Desktop

| Platform | Gate | Status |
| --- | --- | --- |
| Linux | Wails production package built locally | PASS |
| Windows | Native Wails build/package/install smoke not executed | NOT VERIFIED |
| macOS | Native Wails build/package not executed | NOT VERIFIED |
| Embedded frontend synchronization | `make build` regenerated the embedded bundle in the build flow; ignored generated output is not treated as source evidence | PASS |

## Security

Closed in the candidate:

- Desktop/session authentication no longer treats Origin/Referer alone as authentication.
- WebSocket query-token use is restricted to loopback and disabled for active tunnel/private remote modes.
- CSRF/session expiry/revocation and origin-policy tests passed.
- Mission assignment rejects cross-project agents/tasks.
- Update manifests require trusted key IDs, Ed25519 signatures, target digests, version policy and safe archive extraction.
- Installers reject missing/invalid signatures, missing trust root, missing signed digest and checksum mismatch; unsigned `checksums.txt` is not an authenticity fallback.
- Cloudflared PATH binaries are checksum-verified before use.
- AGY debug output is redacted before logging.
- Existing path traversal, symlink escape, command-template and workspace-isolation tests passed.

Open security/release risks:

- BLOCKER: production Ed25519 public trust root is not configured in the protected release environment. The binary intentionally fails closed until `NEXUS_UPDATE_PUBLIC_KEY` is supplied; no production secret was invented.
- HIGH: native Windows/macOS security and installer execution are not verified.
- MODERATE dependency debt: `npm audit --audit-level=high` reported two moderate `@vitest/mocker` advisories; remediation requires a breaking Vitest upgrade and was outside this certification scope.

## Release/update

The implemented chain is:

`release artifact -> signed manifest -> detached signature -> embedded build-time public trust root -> Nexus verification -> authorized update`.

Verified locally:

- signer tests cover canonical GoReleaser names, supported targets, public/private key matching and published signed bytes;
- updater tests cover valid/tampered/untrusted/unsigned manifests, target binding, downgrade/expiry, corrupt/oversized/traversal archives and rollback;
- release workflow requires `NEXUS_UPDATE_PUBLIC_KEY`, protected private key/key ID, signed manifest and native artifacts;
- GoReleaser and Wails build definitions propagate the trust root via ldflags;
- NSIS workflow definitions include package existence and silent install/uninstall smoke.

Not verified:

- production key provisioning and actual hosted release signing;
- GitHub Actions same-SHA run covering Linux, Windows, macOS, browser, desktop and release artifacts;
- Authenticode/Apple signing and notarization;
- native installer execution on Windows/macOS.

## Maestro independence

Maestro is optional. Existing no-Maestro tests and capability-degraded behavior passed; Nexus-owned runtime, provider, session, workspace, PTY, execution and evidence boundaries remain in Nexus.

A full authenticated provider-backed Mission run producing a durable evidence stream was not available in this environment. Therefore the following remain NOT VERIFIED: authenticated Mission execution, live provider failover/handoff/escalation, and hosted evidence correlation. No Maestro dependency was added to runtime.

## Independent review

Four reject-oriented reviews were completed for backend/core, frontend/browser,
security and Windows/release/CI. They identified and drove the fixes recorded
above, including unsafe updater archive handling, installer fail-open behavior,
cloudflared pin handling, cross-project Mission assignment and non-hermetic
browser bootstrap. A requested final red-team agent could not run because the
agent service rejected its model and then reached the account usage limit; this
limitation is not treated as a positive review result.

## Remaining debt

### BLOCKER

- Production updater trust root is waiting for human-controlled protected configuration.
- Native Windows gates and Windows installer/Wails smoke are not executed.
- Native macOS gates and Wails/installer smoke are not executed.
- CI same-SHA evidence is not available because this branch was not pushed or run on hosted runners.
- Authenticated provider-backed Mission evidence is absent.

### HIGH

- PowerShell parser, actionlint and yamllint were unavailable locally.
- Independent visual pixel-diff verification is not implemented by the existing visual smoke command.
- Darwin cloudflared checksum pin is absent until a verified digest is supplied.

### ACCEPTABLE BACKLOG

- Broader historical frontend i18n/inline-style/`any` cleanup was not expanded into a redesign or unrelated refactor.
- Overnight eight-hour certification was explicitly out of scope for this campaign.
- Moderate test-dependency advisories remain pending a compatibility-reviewed upgrade.

## Test counts

- Frontend: 68 test files, 350 tests passed in `make quality`.
- Go: 1,078 test actions and 49 package pass results from `go test -json -count=1 ./...`.
- Browser hardening: six responsive breakpoints plus bootstrap/persistence/deep-link/accessibility checks passed.
- Update gate: six verification stages passed.
- Documentation: 134 Markdown files and 10 required assets/files verified.

## Commits

- `e53f8352e4c3647632ebac7877cf0a6a3bd62647` — `fix(release): close production certification blockers`
- `37f970b` — `docs(release): publish production finalization evidence`
- `d22c293` — `docs(release): anchor visual evidence to candidate`
- `61a9531` — `docs(release): update finalization checkpoint`
- `49af5d3` — `docs(release): index certification evidence`

The documentation commits do not alter the candidate executable code SHA.

## Final recommendation

The candidate is not eligible for overnight certification yet. Complete protected production trust-root configuration, run the strict Windows/macOS and hosted same-SHA workflows, and capture authenticated Mission evidence before changing the verdict.
