# Nexus 1.0.0-beta.7 — Windows constrained user-space homologation

Date: 2026-09-30 (America/Sao_Paulo)  
Candidate: `release/v1.0.0`  
Expected version: `1.0.0-beta.7`

## Candidate and source identity

- Repository: `https://github.com/IAPro-Community/nexus.git`
- Remote check: `git fetch origin --prune` completed; `git pull --ff-only origin release/v1.0.0` previously returned `Already up to date`.
- Remote `origin/release/v1.0.0` at final fetch: `deab51effe896d7a63eac2c07e86bdf8752a76c5`.
- Linux homologated SHA from `DEV/validation/BETA7_LINUX_FINAL_20260930.md`: `6bf1b9fef578822d2be9d542409a44a2e154ed96`.
- Windows source under test (local fix commit): `ff5a0277bc609009d5c595ad9b9d652429c12d9c`.
- Version file: `1.0.0-beta.7`.
- Same SHA as Linux: **NO**. Linux revalidation required: **YES**.
- Fix branch: `fix/beta7-windows-user-space-20260930`; one local commit ahead of `origin/release/v1.0.0` at the time of report. No force update or reset was used.

## Environment

- OS: Windows 11 / build 10.0.26200; architecture: AMD64.
- Go 1.25.14; Bun 1.3.9; Node 20.15.1; npm 10.7.0; Git 2.45.2.
- Playwright Chromium 1243 from user cache; WebView2 runtime present.
- 12 logical CPUs, approximately 32.5 GiB visible RAM.
- Docker, NSIS, `signtool`, PowerShell 7, GCC/Clang, OpenCode and AGY unavailable. `CGO_ENABLED=0`.
- Administrator rights used: **NO**. No UAC/policy, registry, hosts, system ACL, driver, service, machine-wide install or security settings were changed.
- Tests and browser servers ran serially in non-interactive batches; no extra visible terminal windows were opened.
- The official homologation PowerShell script is unsigned and blocked by the machine's execution policy; policy was not bypassed. Equivalent gates were run directly where possible.

## Results

| Gate | Result | Requires admin? | Evidence / notes |
|---|---|---:|---|
| Remote fetch/pull and candidate identity | PASS | No | Remote unchanged; version 1.0.0-beta.7. |
| Frontend frozen install | PASS | No | Bun 1.3.9, frozen lockfile. |
| Frontend format/typecheck/lint/styles/unit/build | PASS | No | `quality:full`: 82 files, 474 tests; CSS architecture, Prettier, ESLint, Stylelint, TypeScript and build passed. |
| Go formatting | PASS | No | `gofmt -l .` returned no files. |
| Go vet | PASS | No | `go vet ./...`. |
| Go tests | PASS | No | `go test -count=1 -timeout 20m ./...`; Windows terminal interactive opt-in regression included. |
| golangci-lint | PASS | No | v2.12.2, zero issues. |
| govulncheck | PASS | No | v1.7.0, no vulnerabilities found. |
| Go race tests | BLOCKED_ENVIRONMENT | No | Race instrumentation unavailable with CGO disabled and no GCC/Clang. |
| Linux helper cross-compilation | BLOCKED_ENVIRONMENT | No | Linux-target compile of Wails-dependent runtime package failed in upstream Wails Linux files (`undefined: pointer`); Windows host lacks a Linux native build environment. This was compile-only, not a Linux test run. |
| Browser E2E / Axe / responsive | PASS | No | E2E, Surface Isolation, Capacity, Composer execution, vague/clarification, themes, HUD web; tested 320–1440 px including 390x844. Axe had no serious/critical findings. |
| Browser visual smoke | PASS | No | Visual smoke log and screenshots retained in evidence. |
| Docs verification | FAIL | No | `bun scripts/docs-verify.mjs`: visual manifest stale after source changes since `0d8c950b7f61aa37b8f2b7157f7279626916bda0`; listed changed Desktop/terminal/UI files. |
| Docs terminal capture / Windows web terminal input | FAIL | No | `docs:capture` repeatedly showed the `cmd.exe` prompt but typed `echo __NEXUS_DOCS_TERMINAL_OK__` did not appear. Browser emitted input frames, no page errors, but no terminal echo/output. Screenshot and failed capture diagnostics retained. |
| CLI version/help/doctor/providers/usage/projects list | PASS | No | Same-SHA CLI binary; isolated test profile. |
| CLI duplicate project display-name smoke | PASS | No | Two isolated project folders produced distinct IDs. |
| CLI `nexus mission` contract | FAIL | No | `nexus mission --help` exits 1: `unknown command "mission"`; CLI help instead describes `nexus run` and `nexus plan run`. No unsupported alias was added. |
| Mission execution / recovery | NOT_RUN_AUTH_REQUIRED | No | No authorized authenticated provider/profile; no mission was launched. |
| Codex native TUI interactions | NOT_RUN_AUTH_REQUIRED | No | Codex CLI detected, but no enabled authenticated Nexus profile. Arrow, mouse, cancellation and resume interactive matrix not run. |
| AGY | BLOCKED_ENVIRONMENT | No | Binary/profile unavailable. |
| OpenCode | BLOCKED_ENVIRONMENT | No | Binary/profile unavailable. |
| Native Desktop/HUD interaction | BLOCKED_ENVIRONMENT | No | Build passed, but computer UI automation surface was unavailable and native app launch/show/hide/resize/close could not be verified. No app was claimed as launched. |
| CLI and Wails native builds | PASS | No | Windows AMD64 production-tag builds succeeded from tested source. |
| Desktop/HUD process and lifecycle | NOT_RUN | No | Build artifacts exist; interactive process lifecycle not verified. |
| Voice/microphone | NOT_RUN | No | No permission/capture/STT smoke performed. |
| DEB / Linux package lifecycle | NOT_RUN | N/A | Linux host and Docker unavailable in this Windows run. |
| Windows installer / uninstaller | NOT_RUN | N/A | NSIS unavailable; system-wide/protected install intentionally not attempted. |
| Authenticode | BLOCKED_EXTERNAL | No | `signtool` and an authorized signing certificate unavailable. No key was created. |
| Privileged-only machine gates | NOT_RUN_PRIVILEGED | Yes | Machine-wide autostart, HKLM, hosts, services, drivers, protected install and policy-dependent symlink setup intentionally not attempted. No elevation requested. |
| GitHub Actions / release publication | NOT_RUN | N/A | Not part of local user-space test; nothing published. |

## Fixes on the local branch

Commit `ff5a0277bc609009d5c595ad9b9d652429c12d9c` (`fix(windows): resolve beta7 user-space homologation failures`) contains narrowly scoped platform/build/test fixes and an E2E bootstrap synchronization fix. It was followed by the frontend quality suite, full Go suite, vet, linter and targeted browser E2E reruns. The docs terminal input failure and CLI `mission` command mismatch remain unresolved; neither was changed speculatively.

## Artifacts

All are local, Windows AMD64, built from source commit `ff5a0277bc609009d5c595ad9b9d652429c12d9c`; SHA256:

| File | Platform / architecture | SHA256 |
|---|---|---|
| `nexus-beta7-win-amd64.exe` | Windows / AMD64 | `E8A71FFA512D04182E3D84A1B9B055B01E690CB6D3A0F7D71ED71395FE93C1D6` |
| `nexus-desktop-beta7-win-amd64.exe` | Windows / AMD64 | `0293EEB62E27D61B56256C684643A071DEA5FFAC0F678FC33E60E84247DFB849` |
| `nexus-hud-beta7-win-amd64.exe` | Windows / AMD64 | `48F992E24BEF0C6E38ADF8D7C4F4A3DF209993C84D1B016F6DB8ED6EDB345396` |

Evidence directory: `C:\Users\kiver.teixeira\Downloads\nexus-beta7-evidence\` (environment, Go, frontend, browser, CLI and artifacts). No credentials were included.

## Verdict

**FAIL** — substantial user-space gates passed, but the web terminal input smoke and CLI mission command contract failed. Documentation visual manifest is stale, and authenticated/provider/native-UI gates remain unverified or blocked. This is not a release certification and does not authorize publication. The tested source differs from the Linux-certified SHA; Linux revalidation is required for the local fix commit.
