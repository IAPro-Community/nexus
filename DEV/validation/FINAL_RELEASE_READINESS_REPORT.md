# IAPro Nexus Final Release Readiness

## Baseline

- main SHA: `f71eb515278168e33d626fd631cd89dfb5e58faf`
- feature initial SHA: `217eada2469333565b2a9d5d818b66a43a51c65b`
- feature final SHA: `c2d41635937d51a7c2f33a9f71dc794337c25f16`
- merge-base: `1899ca6334576d859056d48a394e51d03758f313`
- latest remote CI observed: `34124005108`, at initial SHA `217eada`; failed.
- remote feature branch after local correction: still `217eada`; no push was performed.

## Findings

| ID | Finding | Root cause | Fix | Regression test | Status |
| --- | --- | --- | --- | --- | --- |
| FC-01 | Generated `.test` binaries tracked | `go test -c` outputs were committed | Removed `handoff.test`, `host.test`, `nexus.test`; added `*.test` ignore rule | `test -z "$(git ls-files '*.test')"` | PASS locally; remote SHA stale |
| FC-02 | macOS `/var` versus `/private/var` recency comparison | Test compared display spelling instead of filesystem equivalence | Workspace recency assertion now uses identity-aware equivalence | `go test ./internal/control/workspace -count=20` | PASS locally; native macOS rerun pending |
| FC-03 | Native macOS Wails build failed | Failure stage observed; detailed log unavailable (GitHub API returned HTTP 403) | No speculative change made | Native macOS CI required | FAIL / UNKNOWN root cause |
| FC-04 | Browser 320px topbar obstruction | Context/status stacking and overflow allowed status control to cover create control | Established explicit topbar context/status stacking and visible overflow | `bun run test:e2e` | PASS locally; remote SHA stale |
| FC-05 | Axe/visual blocked by browser failure | CI dependency chain skipped them after Browser failure | Browser gate now reaches Axe/visual assertions locally | `bun run test:a11y`, `bun run test:visual` (same script, sequential) | PASS locally; remote SHA stale |
| FC-06 | Windows full test failed before platform E2E | Failure stage observed; detailed log unavailable (GitHub API returned HTTP 403) | No platform-specific guess applied | Native Windows CI required | FAIL / UNKNOWN root cause |
| FC-07 | Linux regression | No failure reproduced | None | `go test ./...`, `go test -race ./...`, integrated tree tests | PASS locally |
| FC-08 | Frontend regression | No failure reproduced after correction | None | `make web-verify`; Browser E2E | PASS locally |
| FC-09 | Security | CI and local vulnerability scan found no vulnerabilities | None | `govulncheck` | PASS |
| FC-10 | Go toolchain parity | Project is `go 1.25.0`; CI pins `1.25.14` | No mismatch found in project configuration | CI metadata + local checks | PASS by inspection; platform CI stale |
| FC-11 | Native Wails matrix | Linux/Windows CI passed; macOS CI failed at initial SHA | Linux build rerun locally; no macOS speculation | `make build-desktop-wails` | FAIL pending macOS |
| FC-12 | GoReleaser snapshot | Initial CI snapshot skipped because dependencies failed | Snapshot executed locally on final SHA | `go run github.com/goreleaser/goreleaser/v2@v2.18.0 release --snapshot --clean` | PASS locally |
| FC-13 | Installer contract | Existing installer tests passed; native installer smoke is platform-dependent | None | `go test ./internal/release -count=10`; CI metadata | PASS locally; native evidence incomplete |
| FC-14 | Update engine | Existing update tests passed | None | `go test ./internal/update -count=10` | PASS locally |
| FC-15 | Repository hygiene | Generated binaries were tracked | Removed artifacts and added ignore rule | `git ls-files '*.test' '*.tmp' '*.log'`; `git diff --check` | PASS locally |

## Local verification

| Gate | Command | Result |
| --- | --- | --- |
| Go unit/integration | `go test ./...` | PASS @ `c2d4163` |
| Go race | `go test -race ./...` | PASS @ `c2d4163` |
| Go vet/format | `go vet ./...`; tracked-file `gofmt` check | PASS @ `c2d4163` |
| Frontend quality | `make web-verify` | PASS (10/10 gates) @ `c2d4163` |
| Browser functional/Axe/visual smoke | `bun run test:e2e` sequentially | PASS @ `c2d4163`; two non-blocking Axe findings remain |
| Workspace stress | `go test ./internal/control/workspace -count=20` | PASS |
| Runtime stress | host/launcher/terminal/web/desktop/update/release repeated tests | PASS locally |
| Security | `GOTOOLCHAIN=auto go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...` | PASS: No vulnerabilities found |
| Linux desktop | `make build-desktop-wails` | PASS |
| Release snapshot | GoReleaser v2.18.0 snapshot | PASS; six archives, four Linux packages, checksums |
| Main integration | no-commit merge of final SHA into fresh `origin/main`; `go test ./... && go vet ./... && git diff --check` | PASS; no merge conflict |

## Platform CI

| Platform/job | SHA | Result |
| --- | --- | --- |
| Frontend | `217eada` | PASS |
| Security | `217eada` | PASS |
| Linux E2E | `217eada` | PASS |
| Desktop Linux | `217eada` | PASS |
| Desktop Windows | `217eada` | PASS |
| Windows E2E | `217eada` | FAIL; test stage; detailed logs unavailable |
| macOS E2E | `217eada` | FAIL; race stage; detailed logs unavailable |
| Desktop macOS | `217eada` | FAIL; native Wails build stage; detailed logs unavailable |
| Browser E2E/Axe/visual | `217eada` | FAIL; Browser verification stage |
| GoReleaser Snapshot | `217eada` | SKIPPED because required jobs failed |
| Required same-SHA CI for final | `c2d4163` | UNKNOWN; branch was not pushed |

## Desktop artifacts

| OS | Artifact | Build | Verified |
| --- | --- | --- | --- |
| Linux | local Wails binary `cmd/nexus-desktop/build/bin/nexus-desktop` | native Linux Wails | PASS |
| Windows | CI artifact at `217eada` | native Wails | PASS in observed CI job |
| macOS | expected native artifact | native Wails | UNKNOWN; observed job failed |

## Browser

- functional smoke: PASS locally at final SHA
- Axe: PASS for serious/critical violations; two moderate non-blocking findings reported by the existing script
- visual: PASS through all six viewport assertions locally
- CI same-SHA evidence: UNKNOWN; final SHA was not published

## Release

- GoReleaser snapshot: PASS locally at `c2d4163`
- expected CLI targets: Linux amd64/arm64, Darwin amd64/arm64, Windows amd64/arm64 generated locally
- installers: syntax/tests pass locally; native OS installer evidence is not complete
- update engine: package tests pass locally; no release publication performed

## Main integration

- current main SHA: `f71eb515278168e33d626fd631cd89dfb5e58faf`
- merge simulation: clean, no conflicts
- integrated verification: `go test ./...`, `go vet ./...`, `git diff --check` passed
- integration worktree was isolated from the user checkout

## Remaining risks

1. Windows native test failure is not root-caused because GitHub job logs/artifacts require repository admin permission in this environment.
2. macOS native race and Wails build failures have no final-SHA evidence and no accessible detailed logs.
3. No remote CI run exists for final SHA `c2d4163`, so same-SHA release gating is not proven.
4. Native macOS `/var` alias regression was corrected in the test consumer but still requires native runner confirmation.
5. The existing browser script reports two moderate non-blocking Axe findings (`page-has-heading-one`, `region`).

## Final verdict

**NO-GO**

The final local correction is green and integrates cleanly with current `main`, but release certification requires native Windows/macOS evidence and all mandatory jobs passing on exactly the final SHA. Those gates are not available and the remote feature branch was intentionally not pushed.
