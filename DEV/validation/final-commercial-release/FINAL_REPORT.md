# FINAL REPORT — Nexus v1.0.0-beta.1 commercial release mission

## Executive verdict

`CONDITIONAL_GO`

Locally executable evidence is green (Go full suite, race on critical
paths, vet, lint on touched packages, govulncheck clean, frontend full
chain + build + 409 unit tests, SBOM generated). The remaining gates are
genuinely external: hosted CI on the frozen SHA (incl. browser/Windows/
macOS jobs), code signing/notarization secrets, distribution DNS/storage,
and GitHub admin for branch protection. Exact commands for each gate are
listed below. Nothing was greenwashed: Playwright/axe did not run locally
(missing OS libs, no sudo) and is marked NOT RUN, not PASS.

## Release candidate

- version: `1.0.0-beta.1` (`VERSION` + `web/package.json`)
- branch: `feat/nexus-maximum-delivery`
- SHA: `919c192` at report freeze (this report + SBOM commit moves it;
  frozen SHA is stated in the delivery message; verify with
  `git rev-parse HEAD` and `git status --short`)
- date: 2026-09-17 (UTC)

## Commits produced (this mission, after baseline b3db75c)

1. `da7434a` fix(ci): restore deterministic frontend dependency lock
2. `51a09c2` fix(terminal): reassert keyboard modes after Ctrl+C (preserved working tree)
3. `05db540` test(providers): make auth fixtures cross-platform
4. `876c271` refactor(conductor): model non-terminal dispatch states honestly
5. `e1f5efa` fix(security): harden conversation attachment ingestion
6. `d6ac138` fix(store): preserve data during schema repair with ledger
7. `b9aedb4` chore(deps): bump golang.org/x/image to v0.45.0 (vuln fix)
8. `0911e9d` style(test): clear ineffassign findings
9. `598b21c` feat(release): decouple binary registry from source repository
10. `ba52cae` feat(entitlement): capability boundary, device identity, analytics split
11. `1389f0c` docs(beta): proprietary licensing, beta playbook, repo hygiene
12. `919c192` docs(install): mark source build as internal-only path

## Tests

| Gate | Result | Evidence |
|---|---|---|
| `go build ./...` | PASS | clean |
| `go test ./...` (full suite) | PASS | no failures |
| `go test -race` (runtime, app, store, update, entitlement, device, flags, control/web attachment subset) | PASS | |
| `go vet` (touched pkgs) | PASS | |
| `golangci-lint run` (web, store, nexus, runtime, spec, update, new pkgs) | PASS | 0 issues |
| `govulncheck ./...` | PASS | clean after x/image→v0.45.0 (fixed GO-2026-4961/6222/5061) |
| Frontend `format:check` | PASS | prettier clean |
| Frontend `lint` / `lint:styles` / `check:styles` | PASS | |
| Frontend `typecheck` | PASS | |
| Frontend `test` (vitest) | PASS | 74 files, 409 tests |
| Frontend `build` + `verify` | PASS | incl. embed-sync |
| `bun install --frozen-lockfile` (bun 1.3.9) | PASS | after lock regen |
| Attachment adversarial (polyglot, fake MIME, trunc/giant, 413s, traversal, 401/403/404) | PASS | 20+ new cases |
| Conductor dispatch states + ULID goal IDs | PASS | |
| Store upgrade fixtures + ledger | PASS | caught 2 real repair bugs |
| Provider HOME/USERPROFILE isolation + PATH separator | PASS | goldens updated (USERPROFILE only) |
| Key rotation + registry backends | PASS | |
| Entitlement/device/flags/analytics | PASS | |
| Coverage (statements): entitlement 100%, flags 100%, telemetry 84.6%, device 79.2%, update 64.7%, nexus 57.6%, store 54.9%, runtime 46.1%, web 36.2%, spec 32.3%, app 29.2% | measured | no P0 without tests |
| Playwright/axe/visual | NOT RUN | env-blocked (libatk missing, no sudo) — hosted CI gate |
| Windows/macOS E2E, installers, DEB/RPM smoke, desktop packaging | NOT RUN | need hosted runners |
| Update E2E live (A→B + negatives) | PARTIAL | unit/negative coverage exists; live flow needs release artifacts |

## Platforms

- Linux: build + full test evidence above. Fresh-install/upgrade/uninstall
  of DEB/RPM and desktop WebKitGTK smoke: pending hosted runner.
- Windows: NSIS script + winget manifest present (license corrected);
  ConPTY/named-pipe/PowerShell coverage is in-repo but installer +
  Authenticode need `EXTERNAL_SIGNING_GATE`.
- macOS: `.app`/dmg pipeline per release workflow; Developer ID +
  notarization need `EXTERNAL_NOTARIZATION_GATE`.

## Security

Fixed: client-MIME trust, unbounded multipart, sessionID traversal,
absolute-path StorageRef leak, ServeFile polyglot serving, missing
dimension/bomb checks, non-atomic ingest (now sniff→decode→reencode,
MaxBytesReader, quotas, att_ULID names, ServeContent+nosniff+disposition,
single-INSERT + temp-rename cleanup); conductor COMPLETED lie (now
BLOCKED/PAUSED/CANCELED + ACCEPTED entrypoint); repair data loss (now
migrate-all + ledger); Windows profile leak (USERPROFILE isolation);
known WebP decoder CVEs (bumped to v0.45.0, scan clean).
Preserved: loopback-only FS, tunnel=PUBLIC_REMOTE + session wipe,
WS query-token loopback-only, atomic session rotation, 0600 device
material, fail-closed manifest verification + downgrade refusal.
Adversarial tests added for upload/serve/auth/CSRF/IDs; control-plane
origin/bootstrap/tunnel properties re-verified by code read (existing
loopback/desktop/exposure/remote suites green).

## Distribution

Beta users install signed binaries only (`install.sh`/`install.ps1`,
DEB/RPM/NSIS/winget). The updater core now resolves manifests through
the `ArtifactRegistry` interface: GitHub Releases on a binaries-only repo
today, any HTTPS static backend (R2/S3/CDN) via `NEXUS_UPDATE_REGISTRY_URL`
tomorrow — no source-repo coupling. Source is never required, never
referenced by the updater, and install docs mark source builds
internal-only.

## Licensing

`LICENSE` is now proprietary beta (holder name must be owner-confirmed
before external distribution). `docs/legal/` holds BETA_TERMS, PRIVACY,
THIRD_PARTY_NOTICES + pinned `GO_MODULES.txt`. Contributor audit:
239+70 commits from project role identities, 1 maintainer address, no
external CLA on file — owner must confirm relicensing clearance
(legal blocker, not a conclusion).

## Entitlements

`EntitlementService` (capability gates, beta grants), offline-signed-doc
shape reserved behind the interface (no Stripe, as scoped),
privacy-safe `installation_id`, opt-in `ProductAnalytics`, kill-switches
for tunnel/autonomous/remote/experimental. Team context stays gated.

## Providers

Contract goldens re-certified (only USERPROFILE added). Classification:
unit/contract VERIFIED for detect/auth-shape hermetics; real-provider
behavior remains UNVERIFIED-by-mocks by definition (documented, not
claimed). Maestro stays optional (degraded, never breaks).

## External gates

1. Hosted CI green on frozen SHA (`.github/workflows/ci.yml` incl.
   browser + OS jobs). Command: push branch, open PR #2, watch checks.
2. `EXTERNAL_SIGNING_GATE`: provide Windows Authenticode cert; wire into
   `packaging/windows` + release workflow secrets.
3. `EXTERNAL_NOTARIZATION_GATE`: provide Apple Developer ID + notary
   credentials; wire signing/stapling in release workflow.
4. Distribution: provide `updates.<dominio>` storage/CDN or binaries-only
   repo; set `NEXUS_UPDATE_REGISTRY_URL` or `NEXUS_RELEASE_REPO`;
   inject `NEXUS_UPDATE_PUBLIC_KEY` at build; keep
   `NEXUS_UPDATE_PRIVATE_KEY` in the release environment only.
5. `EXTERNAL_GITHUB_ADMIN_GATE`: `GITHUB_TOKEN=<admin> ./scripts/apply-branch-protection.sh`
   (verify required-check names against ci.yml job names first).
6. Release: follow `docs/release/BETA_PLAYBOOK.md`; tag `v1.0.0-beta.1`
   exactly on the certified SHA; never reuse a stale tag.

## Known limitations

- Playwright/axe/visual not executed in this environment (missing system
  libs, no sudo); must pass in hosted CI before GO.
- `nexus.Default()` process singleton: test servers share one store per
  test process (production single-server unaffected); tests were written
  accordingly.
- `PRAGMA wal_checkpoint` return still ignored at Close (pre-existing;
  close never fails because of it).
- Doctor bundle has allowlist + redaction tests; explicit pre-create
  preview UX in CLI should be confirmed during beta smoke.
- Coverage is module-honest (no P0 untested) rather than a global number.

## Rollback

Move the channel pointer to the last good signed manifest (downgrade is
refused automatically; publish previous version as a new entry or use
explicit admin downgrade). Client mid-apply failures restore via
`Updater.Rollback()`. Code rollback: `git revert` the listed commits in
reverse; DB repairs are additive and backward-compatible.

## Final installation instructions (beta tester)

Linux/macOS:
`curl -fsSL https://raw.githubusercontent.com/IAPro-Community/nexus/main/install.sh | bash -s -- --version=v1.0.0-beta.1`
then `nexus doctor` and `nexus web`.
Windows (PowerShell):
`irm <install.ps1 URL> -OutFile install.ps1; .\install.ps1 -Version v1.0.0-beta.1`.
No GitHub access, no source, no build tools required. Beta terms and
privacy: `docs/legal/BETA_TERMS.md`, `docs/legal/PRIVACY.md`.
