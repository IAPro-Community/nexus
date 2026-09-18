> **HISTORICAL — NOT CURRENT RELEASE EVIDENCE**
>
> Canonical current status: `DEV/validation/current/RELEASE_STATUS.md`.

# IAPro Nexus — Current State Audit

Updated: 2026-09-10 (consolidation continuation)
Branch: `feat/nexus-maximum-delivery`
HEAD: local worktree based on `bfc90fc`; uncommitted consolidation changes are
present and intentionally not published.
Remote: `origin/feat/nexus-maximum-delivery` remains at the branch base; no
commit or push was created by this campaign.

## Evidence boundary

This audit distinguishes local Linux evidence from native platform evidence. A
cross-compile or a green unit test on Linux is not treated as native Windows,
macOS, or Desktop verification.

## Inventory

| Area | State | Evidence / finding |
| --- | --- | --- |
| Core lifecycle | SHIPPED | `internal/app/core.go`; `go test ./...` passes locally on Linux. |
| CLI | SHIPPED | `cmd/nexus`, CLI package tests pass locally. |
| Web | SHIPPED | `make web-verify` is 10/10 green locally after formatting the local notification change. |
| Shared React build | SHIPPED | Web build and embedded bundle equality gate pass locally. |
| Desktop shell | PARTIAL | Wails entrypoint and shared embedded frontend exist; the config is now adjacent to `cmd/nexus-desktop` and Linux Wails packaging passes locally, while native Windows/macOS smoke evidence is unavailable. |
| PlatformBridge | PARTIAL | Web/Desktop bridge exists; capability claims were corrected to reflect actual bindings, fallback availability, and unimplemented features. |
| Terminal/runtime | PARTIAL | Linux PTY/runtime evidence passes; the Windows ConPTY attribute ABI was corrected locally, but native Windows/macOS execution evidence is still missing. |
| Windows | UNVERIFIED (historical CI failed) | CI run `34012236345` failed during the full test step before the local Windows fixes; the updated dirty worktree has no native runner evidence. Logs require repository admin permission. |
| macOS | UNVERIFIED (historical CI failed) | CI run `34012236345` failed during the race step before the local candidate was published; the updated dirty worktree has no native runner evidence. Logs require repository admin permission. |
| Linux | SHIPPED | CI run `34012236345` Linux E2E passed; local `go test ./...` and `go vet ./...` pass. |
| Path identity/workspace | PARTIAL | Path identity implementation and tests exist; native alias cases require Windows/macOS evidence. |
| Providers/quota | PARTIAL | Quota monitor and provider adapters exist; truthfulness rules require continued regression coverage. |
| Scheduler/mission | SHIPPED | Existing package tests pass locally; no evidence justifies reopening architecture. |
| Update service | PARTIAL | CLI, Web, and Desktop now use the shared `internal/update` service for Nexus status; unsigned manifests fail closed and the signer emits absolute artifact URLs, but default key publication/installer integration is incomplete. |
| Installer | PARTIAL | `install.sh` and `install.ps1` now require a pinned version, verify `checksums.txt`, and require explicit source-build intent/ref; detached Ed25519 manifest verification/public key distribution is still not wired. |
| Release pipeline | PARTIAL | CI has native jobs, packaged artifact uploads, and a snapshot job; local GoReleaser v2.18.0 snapshot passes, while publication is statically same-SHA-gated and promotes selected CI artifacts; the gate has not executed successfully and native CI remains red. |
| Maestro integration | PARTIAL | Opt-in installer path exists; CLI now separates `nexus update` from explicit `nexus maestro status|doctor|update`, with truthful degraded status when unavailable. |
| Doctor | PARTIAL | Doctor is read-only and now uses evidence-bound WebKitGTK/WebView2 probes; native shell/ConPTY smoke remains unavailable, so unverified capabilities are WARN/SKIPPED rather than PASS. |
| Public support docs | PARTIAL | Platform matrix was corrected to evidence-backed `PARTIAL` and `UNVERIFIED` states, with historical CI failures kept as historical evidence rather than current support claims. |
| Community repository | PARTIAL | Added truthful `CODE_OF_CONDUCT.md`, `SECURITY.md`, `SUPPORT.md`, `ROADMAP.md`, `GOVERNANCE.md`, `CHANGELOG.md`, and a PR template; no repository transfer/rename or formal CODEOWNERS assignment is authorized. |

## Current mandatory reds / missing evidence

The local consolidation gates are currently green after the Missions planning
and MissionRun cancellation-contract slices. This does not promote the
worktree to release status: native Windows/
macOS and same-SHA remote CI evidence remain unavailable, and the local tree
contains unrelated/pre-existing changes that are preserved.

- Remote CI run `34012236345` is failed for the committed tree at this HEAD;
  the current worktree contains additional uncommitted changes and has not
  been published for a fresh run.
- Failed jobs: Frontend format, Windows full Go tests, macOS race tests.
- Browser, Desktop Linux/Windows/macOS, and GoReleaser snapshot were skipped because
  their dependencies did not pass.
- GitHub API permits run/job metadata but denies failed-log download with HTTP 403
  because repository admin rights are unavailable.

## Immediate next findings

1. Run the corrected Windows ConPTY/SessionHost implementation on a native
   Windows runner and obtain macOS race logs or a native runner; no
   platform-specific PASS is claimed from cross-compilation or Wine.
2. Move production installers from checksum-only release downloads to the
   existing signed update-manifest trust chain, with a published trusted
   keyring.
3. Publish the corrected candidate and obtain a successful same-SHA CI run so
   Browser, Desktop, and GoReleaser jobs execute rather than being skipped.

## Security scan

`make security` completed with `No vulnerabilities found`; the target now uses
an installed `govulncheck` when present and a pinned `go run` fallback otherwise.

## Local uncommitted work

The worktree contains user changes related to quota monitoring, notifications,
handoff ordering, and generated verification reports. They are preserved and are
not treated as disposable baseline noise.
