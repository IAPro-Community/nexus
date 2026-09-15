# Current release status

## Soak de 8h — 2026-09-15

- **PASS** — `make overnight-soak DURATION=8h` terminou normalmente.
- Resultado: `go test` passou após `28800.028s` (8h), no
  `TestOvernightSoakHarnessBounded`.
- Evidência local; não substitui CI hospedado no mesmo SHA nem smoke com
  provedores autenticados.

STATUS: CURRENT
GIT_SHA: 2b9c95693be7bf73a463c6395ec1ea9f131a4c9c
GENERATED_AT: 2026-09-15T03:26:30Z
ENVIRONMENT: Linux x86_64, Windows Server 2022 (CI), macOS 14 (CI)
HOSTED_CI_RUN_ID: 34924168564
HOSTED_CI_URL: https://github.com/IAPro-Community/nexus/actions/runs/34924168564
FINAL_MERGE_CANDIDATE: GO

## Version

`1.0.0-beta.1` (tag: `v1.0.0-beta.1`)

## Hosted CI Verification (Run 34924168564 — 100% PASS)

All 14 matrix jobs succeeded natively on GitHub Actions:

| Job | Platform / Runner | Status | Duration | Evidence |
|---|---|---|---|---|
| Windows E2E (ConPTY, Named Pipe, PowerShell) | Windows Server 2022 / Go 1.25.14 | PASS ✓ | 6m46s | ConPTY, Named Pipe, self-contained echo helper, PowerShell smoke |
| Linux E2E | Ubuntu 24.04 / Go 1.25.14 | PASS ✓ | 5m08s | Race detector, PTY, Unix socket, runtime & web E2E |
| macOS E2E (PTY, Socket, Resize, Install) | macOS 14 / Go 1.25.14 | PASS ✓ | 1m49s | PTY, socket, resize, install.sh smoke |
| Frontend (Format, Lint, Styles, Typecheck, Unit & Build) | Ubuntu 24.04 / Bun | PASS ✓ | 2m02s | Prettier, tsc, ESLint, Stylelint, vitest (388/388), bundle embed sync |
| Security | Ubuntu 24.04 / Go 1.25.14 | PASS ✓ | 27s | govulncheck zero vulnerabilities |
| Desktop Windows (Wails native build) | Windows Server 2022 | PASS ✓ | 2m00s | Wails build, nexus-desktop.exe packaged |
| Desktop macOS (Wails native build) | macOS 14 | PASS ✓ | 28s | Wails native app bundle build |
| Desktop Linux (Wails native build) | Ubuntu 24.04 | PASS ✓ | 2m24s | Wails native Linux desktop artifact |
| Browser E2E, Axe & visual smoke | Ubuntu 24.04 / Playwright | PASS ✓ | 2m13s | Playwright E2E, accessibility (Axe AA), visual layout |
| GoReleaser Snapshot Build | Ubuntu 24.04 | PASS ✓ | 4m48s | Snapshot binary matrix, DEB, RPM, archives |
| Smoke DEB install | Ubuntu 24.04 container | PASS ✓ | 11s | dpkg -i, nexus version assertion |
| Smoke RPM install | Fedora container | PASS ✓ | 21s | rpm -ivh, nexus version assertion |
| Windows NSIS installer | Ubuntu 24.04 / NSIS | PASS ✓ | 1m01s | makensis build, payload stage, installer smoke |

## Local Verification Gates

- `make quality`: PASS (format-check, lint-frontend, lint-styles, typecheck, lint-go, test-go, test-frontend, version-check)
- `make docs-verify`: PASS (159 Markdown files, 22 required assets, 11 isolated synthetic visual captures)
- `make web-verify`: PASS (10/10 gates)
- `version-check`: PASS (VERSION=1.0.0-beta.1, web/package.json, wails.json, .goreleaser.yaml consistent)

## Operational & Review Verdict

- **Platform Parity**: Linux, macOS, and Windows validated natively in CI.
- **Terminal Reliability**: PowerShell start latency replaced with fast self-contained test echo helper.
- **Packaging**: Transparent `ai.exe` alias staged alongside `nexus.exe` for zero-friction upgrades.
- **UI Redesign**: UsageSurface grid adaptativo, AccountLoginModal, RegisterProviderModal, SchedulerExplainModal com 100% i18n (en, pt-BR, es).
- **Branch Integration**: Fast-forward / clean merge with `origin/main` (commit `2b9c956`).
- **Verdict**: **MERGE CANDIDATE = GO**
