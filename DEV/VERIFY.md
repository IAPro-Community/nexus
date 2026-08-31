# Verification status — 2026-08-30

## Finalization pass

- Source baseline is recorded in `DEV/finalization/SOURCE_BASELINE.md`.
- Workspace v2 and strict terminal protocol changes are present in the working tree; focused regressions pass.
- The working tree is intentionally uncommitted. Remote feature branch remains at `a1f3a573602a7c43404acf5eb1d4ac0cbc380110`.

## Local gates

- Go 1.25.0: `gofmt`, `go vet ./...`, `go test -count=1 ./...`, `go test -count=1 -race ./...`, and `go build ./cmd/nexus` PASS.
- Frontend: frozen Bun install, typecheck, lint, 26 Vitest files / 94 tests, and build PASS.
- Embedded frontend: `web/dist` and `internal/control/web/dist` byte-for-byte synchronized.
- Cross-compilation: host/terminal Windows amd64 and host macOS amd64 test binaries compile.
- SessionHost throughput regression: `go test -race -run TestQA_LargeThroughputStreaming -count=20 ./internal/control/host` PASS.
- Large stdout capture regression: `go test -run TestCaptureStdoutDrainsLargeOutputWithoutDeadlock ./internal/app` PASS.
- Terminal protocol regressions: 3 tests PASS; provider JSON remains output and malformed control frames do not leak.

## Runtime evidence

- Isolated local Codex Direct Work E2E on port 3000 PASSed with exact-line `NEXUS_E2E_OK` and cleanup.
- Chromium was installed locally and the token-safe browser smoke completed with exit 0.
- Context handoff, Safe Apply, real Mission, remediation, restart durability, and Take Control remain NOT_TESTED.

## Not proven in this environment

- Native Windows/macOS runtime, same-SHA GitHub CI, GoReleaser snapshot, real AGY trust persistence, real AGY E2E, full Mission/restart/take-control E2E.
- Available security tools (`govulncheck`, `gitleaks`, `actionlint`) and `goreleaser` are not installed.

## Delivery status

The source changes are intentionally uncommitted in this workspace. The governing execution policy forbids automatic commit/push/PR creation, so GitHub CI cannot be rerun for the new source and the branch is not declared production-ready. See `/tmp/IAPro-Nexus-FINAL-PRODUCTION-100.md`.

## Proxy-nginx attention regression — 2026-08-30

- Root cause: `/respond` sent input over an RPC connection rejected by the terminal lease check; the detector also retained the answered prompt in its sliding buffer; the banner occupied a grid row.
- Fix verified by focused Go host/web tests, host `-race`, frontend typecheck/lint/tests/build, embedded bundle rebuild, and `git diff --check`.
- The live server was rebuilt with `v0.5.0-beta.5` and the current source bundle. Browser interaction still requires the authenticated bootstrap session.

## Nexus V0 execution — 2026-08-30

- Implementation worktree remains based exactly on `c96885218b3cff319f481aed4423dc5fd7b52a30`; no commit or push was created.
- Added workspace presentation state/desktop renderer, launcher surface routing, Ask-existing-Agent API/UI, and stale-attention visibility/durable dismissal regressions.
- Verification: `go test -race -count=1 ./...`, `go vet ./...`, Windows amd64 cross-compile for terminal/host, frontend typecheck/lint/30 files/104 tests/build, embedded bundle sync, and `git diff --check` PASS.
- Native Windows/macOS execution, authenticated browser interaction, real providers, same-SHA CI, and release snapshot remain unverified.

## Nexus V0 continuation — 2026-08-30

- Canonical HEAD: `b23eb3e22f1448ac0e401694ae7bc59c0f4ff68e`.
- Project Shell: `ShellDriver` and `ProjectShellSurface` use the existing terminal runtime; focused driver tests and frontend gates PASS.
- Context Readiness: five-state contract with fingerprint drift tests PASS; persistence/Composer integration remains open.
- Full local Go tests, race, vet, build, frontend typecheck/lint/tests/build, embedded sync and Windows amd64 compile PASS.
- Browser harness: Chromium smoke PASS, authenticated bootstrap FAIL; classify as `SMOKE`, not full user journey. Native Windows/macOS and same-SHA CI remain `UNVERIFIED`.
