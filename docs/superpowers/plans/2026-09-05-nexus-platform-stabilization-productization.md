# Nexus Platform Stabilization & Productization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to execute this plan task-by-task. Use `superpowers:test-driven-development` for every behavior change and `superpowers:verification-before-completion` before each gate.

**Campaign ID:** `01a07224-1c17-7863-881a-6b1963a6ce43`

**Goal:** Take Nexus from the current red multiplatform branch to a release-grade, diagnosable and safely updateable product without adding Desktop or new product features.

**Architecture:** Preserve the mature domain core and harden the operational boundary around it. Platform-specific code owns executable resolution, path identity, credentials, terminal/IPC lifecycle and process supervision. Runtime status remains backward compatible while gaining an additive startup stage and structured fault. Release artifacts remain immutable GitHub Release assets; a signed static registry only selects and describes those artifacts.

**Tech Stack:** Go 1.25, SQLite, React 19, TypeScript, SCSS Modules, Bun 1.3.9, GitHub Actions, golangci-lint v2.12.2, GoReleaser OSS v2.18.0, nFPM, Ed25519 signatures.

**Starting point:** `feat/nexus-maximum-delivery`; audited published SHA `56bcd5b044d9c9398a721839b41526cdbe7514c1`; planning-time local HEAD `ab88fcbfddb5d99cf77c5e6f651aa2e5aa281770`. Before implementation, record the actual immutable starting SHA and do not assume either value is still current.

## Decision record

### Principles

- Runtime truth comes before distribution: a platform is supported only after native runtime, doctor, build and smoke evidence pass on its native runner.
- Compatibility changes are additive: existing runtime states, JSON fields, protocol framing and stored records remain readable during migration.
- Tooling, artifacts and update decisions are exactly versioned and cryptographically verifiable.
- Diagnostics are read-only by default; updates are checked automatically but applied only after explicit user confirmation.
- Desktop, Flow Control Center, Permission Center, Take Control UX, media, ACP and licensing remain outside this campaign.

### Drivers

1. Eliminate Windows hangs, partial startup and orphaned process trees.
2. Establish one cross-platform contract for executables, filesystem identity and credentials.
3. Make failures actionable locally and make releases reproducible, installable and safely updateable.

### Chosen delivery model

Use one campaign branch and one Draft PR, with internally serial milestones and atomic Conventional Commits. Do not partially merge the branch. Multiple smaller PRs would reduce review surface, but would expose incompatible intermediate states across protocol, persistence, runtime and updater. A single unstructured branch is also rejected: each milestone below has an explicit gate and bisectable commits.

### Known trade-off

Binding IPC early improves diagnosis and observability, but an open endpoint must never be interpreted as a ready runtime. Preserve `RuntimeState` for operational compatibility and add a separate `StartupStage`; only `RUNNING` means the provider is usable.

## Global constraints

- Do not weaken `--frozen-lockfile`, race tests, security checks or native-platform tests.
- Do not use `latest` for Bun, golangci-lint or GoReleaser.
- Do not silently fall back from a secure platform facility to a weaker one.
- Do not compare display paths for authorization, equality or workspace bounds.
- Do not execute `.cmd`, `.bat` or `.ps1` as though they were PE executables.
- Do not use `KILL_ON_JOB_CLOSE` from the browser/window lifecycle. The detached SessionHost owns the Job Object.
- Do not let doctor persist synthetic projects, sessions or registry entries.
- Do not include environment variables, arguments, prompts, transcripts, raw provider logs or secrets in diagnostic bundles.
- New or touched frontend component styles use SCSS Modules, semantic `--nx-*` tokens and i18n; no static inline styles, local CSS, `!important` or hardcoded visible strings.
- Preserve unrelated local changes. Prefer an isolated worktree when execution begins.

## Gate policy

The campaign has two immutable-SHA evidence checkpoints:

1. **Runtime checkpoint:** after Tasks 1–8, obtain three complete green CI runs on the same SHA.
2. **Release checkpoint:** after Tasks 9–13, obtain three complete green CI runs on the final SHA.

Each sequence must be independent and complete: no selective reruns, skipped/quarantined tests, cancelled jobs or different SHAs. Required evidence includes frontend, Linux, Windows, macOS and snapshot/release gates. `.deb` and `.rpm` install smoke tests are blocking. MSIX remains experimental and nonblocking until publisher identity, certificate storage and Authenticode signing are operational.

---

### Task 0: Freeze scope and capture a reproducible baseline

**Files:** `.github/workflows/ci.yml`, `docs/engineering/QUALITY_AUDIT.md`, this plan

- [ ] Create or reuse the campaign branch and Draft PR; record starting SHA, dirty-tree state and CI run links.
- [ ] Add a PR checklist that rejects Desktop/Wails, Flow UX expansion, media, ACP and licensing changes.
- [ ] Record every current failure by job, test and OS. Classify it as tooling, product defect or blocked downstream gate.
- [ ] Confirm branch protection targets for `main` and the campaign branch: PR required, branch up to date, required checks, approval, resolved conversations, no force-push and no deletion.
- [ ] Commit planning/baseline metadata separately with `docs(release): define stabilization campaign gates`.

**Acceptance:** A reviewer can identify the exact starting SHA, current failures, permitted scope and required checks without reading CI history manually.

### Task 1: Make CI and dependency installation deterministic

**Files:** `web/bun.lock`, `web/package.json`, `.github/workflows/ci.yml`, `.golangci.yml`, `.goreleaser.yaml`

- [ ] With Bun 1.3.9, run a clean non-frozen install, inspect the lockfile-only diff and commit the regenerated `web/bun.lock`.
- [ ] Prove `bun install --frozen-lockfile` succeeds from a clean checkout.
- [ ] Pin golangci-lint to `v2.12.2`; migrate `.golangci.yml` to schema v2 and run `golangci-lint config verify`.
- [ ] Pin GoReleaser OSS to `v2.18.0` in CI.
- [ ] Add a CI assertion that fails if these tools return an unexpected version.
- [ ] Run frontend format, typecheck, ESLint, Stylelint, style allowlist, unit tests, build and embed verification.
- [ ] Run Go format, vet, lint, unit tests and security checks before relying on platform jobs.
- [ ] Commit lockfile and CI/toolchain changes separately: `build(web): synchronize bun lockfile` and `ci(infra): pin reproducible quality tooling`.

**Acceptance:** A clean checkout installs with the frozen lockfile and every quality tool reports the pinned version; Linux no longer stops at an incompatible linter.

### Task 2: Introduce a command-resolution contract for every provider

**Files:** `internal/runtime/runtime.go`, new `internal/runtime/executable.go`, new OS-specific resolver files, `internal/runtime/*_test.go`, `internal/control/driver/*.go`, `internal/core/provider/adapters/*/*.go`, `internal/nexus/maestro.go`, `internal/control/launcher/template.go`

- [ ] Write table tests for explicit overrides, `PATH`, `PATHEXT`, spaces, Unicode and shell metacharacters.
- [ ] Define:

```go
type ResolvedCommand struct {
    ArtifactPath string
    LauncherPath string
    PrefixArgs   []string
    Kind         string
    SearchedPaths []string
}
```

- [ ] Implement precedence on Windows: explicit override, `PATH`/`PATHEXT`, `%APPDATA%\\npm`, `%LOCALAPPDATA%`, then documented `%USERPROFILE%` tool directories.
- [ ] Launch `.exe` directly; launch `.cmd`/`.bat` through `%ComSpec% /D /S /C` with tested quoting; launch `.ps1` via `pwsh`, then Windows PowerShell, with `-File` and without bypassing execution policy.
- [ ] Keep npm global-bin probing out of normal startup; expose it only as a doctor diagnostic.
- [ ] Route Codex, Claude, Gemini, OpenCode, AGY, Cursor, Maestro and custom templates through the resolver in both control drivers and core adapters.
- [ ] Return `SearchedPaths` in structured `PROVIDER_NOT_FOUND` diagnostics without exposing unrelated environment contents.
- [ ] Commit as `feat(providers): centralize cross-platform executable resolution`.

**Acceptance:** Windows tests resolve and launch representative `.exe`, `.cmd`, `.bat` and `.ps1` shims from paths containing spaces and Unicode; no supported provider calls `exec.LookPath` directly.

### Task 3: Separate terminal preparation from provider process launch

**Files:** `internal/control/terminal/terminal.go`, `terminal_unix.go`, `terminal_windows.go`, platform tests, `internal/control/host/host.go`

- [ ] Replace the conflated `Start` lifecycle with explicit terminal preparation and process-start operations while keeping a compatibility wrapper where necessary.
- [ ] Write a Windows ConPTY test whose timeout can actually interrupt a blocked read; never place a blocking `Read` inside a deadline loop on the same goroutine.
- [ ] Add echo, stdin EOF, resize, provider exit, repeated close and partial-start cleanup tests.
- [ ] Audit ownership of every pipe, process/thread handle, pseudoconsole handle and goroutine.
- [ ] Correct `ClosePseudoConsole` binding semantics; it is not a BOOL-returning API.
- [ ] Mirror lifecycle invariants in Unix PTY tests so the interface cannot diverge by OS.
- [ ] Commit as `refactor(terminal): split terminal preparation from process startup`.

**Acceptance:** ConPTY echo completes within a bounded test deadline; all start-failure paths release handles and goroutines; `Close` is idempotent on every OS.

### Task 4: Make SessionHost startup observable and handshake truthful

**Files:** `internal/control/registry/models.go`, `internal/control/protocol/*.go`, `internal/control/host/host.go`, `internal/control/launcher/launcher.go`, `internal/control/protocol/endpoint_windows.go`, related tests

- [ ] Preserve existing `RuntimeState` values and add optional `startup_stage`, `stage_changed_at` and `last_fault` fields.
- [ ] Implement stages: `CREATED`, `HOST_STARTING`, `IPC_BINDING`, `IPC_BOUND`, `PROTOCOL_READY`, `TERMINAL_STARTING`, `TERMINAL_READY`, `PROVIDER_STARTING`, `RUNNING`.
- [ ] Implement faults: `IPC_BIND_FAILED`, `IPC_TIMEOUT`, `PROTOCOL_FAILED`, `CONPTY_START_FAILED`, `PROVIDER_NOT_FOUND`, `PROVIDER_EXITED_EARLY`, `WORKSPACE_INVALID`, `PERMISSION_DENIED`, `CREDENTIAL_ISOLATION_UNAVAILABLE`, `HOST_EXITED_EARLY`, `PROCESS_SUPERVISION_FAILED`.
- [ ] Reorder startup: persist `CREATED`; spawn host; bind IPC; self-probe protocol; prepare terminal; start supervised provider; prove child alive through status RPC; then mark `RUNNING`.
- [ ] Remove the permissive empty-security-descriptor fallback for Named Pipes. A secure owner-only descriptor failure becomes `IPC_BIND_FAILED`.
- [ ] Make `WaitForEndpoint` return the last observed stage/fault and host-exit evidence instead of a generic timeout.
- [ ] Add protocol compatibility tests proving older clients ignore additive fields and existing framing remains unchanged.
- [ ] Commit as `feat(terminal): expose structured runtime startup diagnostics`.

**Acceptance:** Every injected startup failure reports its exact stage and stable fault code; an open pipe is never reported as runtime-ready; the opaque handshake timeout is eliminated.

### Task 5: Supervise Windows runtime trees with Job Objects

**Files:** new `internal/control/host/job_windows.go`, `proc_windows.go`, `host.go`, Windows lifecycle tests

- [ ] Create one Job Object per detached SessionHost with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`.
- [ ] Create the provider suspended, assign it to the Job Object, then resume it. Treat assignment failure as `PROCESS_SUPERVISION_FAILED`.
- [ ] Define ownership: browser/window close does nothing; explicit runtime stop sends graceful termination then closes the job after the grace period; SessionHost process/service shutdown and host crash close the job and kill the tree.
- [ ] Test child and grandchild cleanup, explicit stop, host crash, repeated stop and browser disconnect.
- [ ] Add handle/goroutine leak assertions around repeated start/stop cycles.
- [ ] Commit as `feat(terminal): supervise windows runtimes with job objects`.

**Acceptance:** No descendant survives explicit runtime stop or SessionHost crash, while closing the Web UI does not terminate a background agent.

### Task 6: Establish DisplayPath, CanonicalPath and IdentityPath

**Files:** `internal/core/config/canonical_path.go`, new OS-specific identity files, `internal/nexus/store/models.go`, `projects.go`, new migration, `internal/control/workspace/workspace.go`, registry persistence, filesystem/security handlers and tests

- [ ] Define additive `PathRef` and `FilesystemIdentity` values:

```go
type PathRef struct {
    DisplayPath   string             `json:"display_path"`
    CanonicalPath string             `json:"canonical_path"`
    Identity      FilesystemIdentity `json:"identity"`
}

type FilesystemIdentity struct {
    Kind      string `json:"kind"`
    StableKey string `json:"stable_key"`
    Available bool   `json:"available"`
}
```

- [ ] On Unix/macOS, derive identity from device and inode. On Windows, open the target and derive volume serial plus file ID.
- [ ] Keep DisplayPath for UI only. Use CanonicalPath for normalized resolution and filesystem identity for equality of existing targets.
- [ ] For a non-existing child, resolve the nearest existing ancestor before security bounding and use a documented textual fallback only for the unresolved suffix.
- [ ] Add SQLite columns for display path, identity kind and identity key while retaining `canonical_path` during compatibility migration.
- [ ] Add read-through/lazy atomic upgrades for legacy `projects.json` and `runtimes.json`; SQLite becomes the authority and legacy project storage stops creating path-hash IDs.
- [ ] Detect identity collisions during backfill. Never auto-merge: mark the affected record blocked and emit doctor reconciliation guidance.
- [ ] Add Windows short/long-name and macOS `/var` versus `/private/var` tests for projects, worktrees, runtime workspaces, isolation and symlink/path-escape security.
- [ ] Add API fields `path_ref`/`workspace_path_ref` without removing current string fields.
- [ ] Commit schema, model, migration and call-site changes in separate atomic commits.

**Acceptance:** Textually different aliases of the same existing directory compare equal; different filesystem objects never compare equal merely because normalized strings collide; legacy records remain readable.

### Task 7: Make credential isolation truthful per operating system

**Files:** `internal/runtime/credentials.go`, OS-specific credential files, tests, provider launch diagnostics

- [ ] Add capability status `SUPPORTED`, `UNSUPPORTED` and `DEGRADED`, with mechanism, reason and error details.
- [ ] Report Linux Secret Service support only when DBus/keyring dependencies and an operational probe pass.
- [ ] Use a real macOS Keychain and Windows Credential Manager integration only if implemented and tested in this campaign; otherwise report truthful `UNSUPPORTED` rather than reusing Linux behavior.
- [ ] Keep home-directory isolation as a separate capability from credential-store isolation.
- [ ] Fail before provider spawn when a required isolation policy is unavailable; warn structurally for optional policies.
- [ ] Add native-runner tests for absent, degraded and supported facilities.
- [ ] Commit as `fix(security): report credential isolation capabilities truthfully`.

**Acceptance:** No OS silently claims secure credential isolation through a missing or foreign platform mechanism.

### Task 8: Consolidate and extend Nexus Doctor

**Files:** `internal/app/app.go`, `internal/app/control_cmd.go`, new `internal/doctor/*`, runtime/path/credential diagnostic adapters, web diagnostic handlers and tests

- [ ] Consolidate the existing `nexus doctor` and `nexus control doctor --json` implementations behind one diagnostic service; do not create a competing command.
- [ ] Cover Nexus version/OS/architecture/data directory/SQLite/permissions/Git/worktree; PTY/ConPTY/resize/stdio; IPC; shells; all providers; Maestro; runtime spawn/handshake/cleanup; HTTP/WebSocket/authentication.
- [ ] Make the default command read-only against real state. Run synthetic probes only in temporary directories and guarantee registry/store cleanup.
- [ ] Define stable check IDs, statuses, duration, evidence and remediation fields for text and JSON output.
- [ ] Add `nexus doctor --bundle` as a ZIP on all platforms with an explicit allowlist.
- [ ] Add a canary secret during bundle tests and fail if any raw or encoded representation appears.
- [ ] Explicitly exclude environment dumps, command arguments, prompts, transcripts, raw provider logs and credentials.
- [ ] Expose the same structured diagnostics to the Web UI without duplicating probing logic.
- [ ] Commit as `feat(go): consolidate cross-platform nexus doctor`.

**Acceptance:** `doctor`, `doctor --json` and `doctor --bundle` report consistent check IDs; bundle tests prove redaction; running doctor leaves persistent state unchanged.

### Runtime checkpoint

- [ ] Run all required jobs three times on one immutable SHA.
- [ ] Verify native Windows ConPTY/Named Pipe/Job Object/runtime/doctor smoke.
- [ ] Verify native macOS PTY/socket/path identity/credential semantics/race/runtime/doctor smoke.
- [ ] Verify native Linux PTY/socket/race/runtime/doctor smoke.
- [ ] Attach run URLs and SHA to the Draft PR. Stop the campaign if any run differs in SHA or uses rerun/skip/quarantine.

---

### Task 9: Produce installable Linux packages and explicit installation receipts

**Files:** `.goreleaser.yaml`, packaging scripts/config, installer smoke tests, release documentation

- [ ] Add nFPM `.deb` and `.rpm` packages for supported Linux architectures with correct binary, license and completion assets.
- [ ] Define package ownership, install paths, upgrade behavior and uninstall behavior; do not add implicit privileged side effects.
- [ ] Add clean-container smoke tests: install, `nexus version`, `nexus doctor --json`, upgrade, uninstall.
- [ ] Preserve tar/zip archives, checksums, SBOMs and embedded Version/Commit/BuildDate.
- [ ] Add MSIX generation only as experimental/nonblocking until a real publisher identity, secured certificate and Authenticode pipeline exist.
- [ ] Commit as `feat(release): add linux packages and installation smoke tests`.

**Acceptance:** `.deb` and `.rpm` install/upgrade/uninstall smoke tests pass from produced artifacts; MSIX cannot affect GA readiness.

### Task 10: Define and verify the signed update registry

**Files:** new `internal/update/*`, manifest schema/testdata, registry publication scripts, release workflow

- [ ] Define a versioned manifest containing schema, generated/expiry timestamps, channel, key ID, Nexus/Maestro versions, changelogs, minimum compatibility and artifacts by OS/architecture/format with URL, size, SHA-256, SBOM and attestation references.
- [ ] Serve static channel manifests at `/v1/stable/manifest.json`, `/v1/beta/manifest.json`, `/v1/nightly/manifest.json` plus detached `.sig` files.
- [ ] Keep immutable GitHub Release assets as the canonical binaries; the registry acts only as signed discovery/CDN redirect metadata.
- [ ] Verify detached Ed25519 signatures over exact manifest bytes before JSON parsing. Embed a key ring with current and next keys and require an overlap window for rotation.
- [ ] Reject invalid signatures/hashes, expired manifests, downgrade, channel mismatch and OS/architecture mismatch.
- [ ] Add ETag/cache, bounded timeouts and client jitter.
- [ ] Produce GitHub artifact attestations/provenance in the tag workflow.
- [ ] Commit registry schema/signing and client verification separately.

**Acceptance:** Golden tests accept current/next keys and reject every negative case; a release asset digest matches manifest, checksum and attestation evidence.

### Task 11: Implement safe update planning, confirmation, apply and rollback

**Files:** `internal/nexus/system_update.go`, new `internal/update/*`, CLI wiring in `internal/app/app.go`, platform apply helpers and tests

- [ ] Replace the current Maestro-only execution path with a shared planner while retaining Maestro support.
- [ ] Add `UpdatePlan`, `ApplyUpdateRequest` and `UpdateReceipt` contracts with manifest digest binding.
- [ ] Support `nexus update`, `nexus update check [--channel] [--json]` and `nexus update apply ... --yes`.
- [ ] Require both explicit confirmation and the checked manifest digest at apply time to prevent TOCTOU updates.
- [ ] Detect `standalone`, `deb`, `rpm` and `msix` installation methods.
- [ ] For standalone Unix, stage on the same filesystem, verify digest, retain backup and use atomic rename.
- [ ] For standalone Windows, use a signed helper that applies only after Nexus exits.
- [ ] Delegate package-managed upgrades to the detected manager without silent elevation.
- [ ] Persist receipts with `STAGED`, `APPLIED`, `RESTART_REQUIRED`, `FAILED` or `ROLLED_BACK`; write a next-start health marker and perform bounded rollback.
- [ ] Add crash/failure injection tests around download, verification, staging, replacement and first restart.
- [ ] Commit planner, platform apply and receipt/rollback behavior separately.

**Acceptance:** No artifact is applied without confirmation, matching manifest digest and matching SHA-256; failed replacement or first-start health check produces an auditable rollback receipt.

### Task 12: Turn the existing Settings update surface into the Update Center

**Files:** `internal/control/web/handlers_nexus.go`, `web/src/nexus/api.ts`, `web/src/features/settings/SettingsSurface.tsx`, extracted update components/tests/SCSS Modules, locale resources

- [ ] Keep GET `/api/v1/system/updates` additive and change POST `/api/v1/system/update` to require product, target version, manifest digest and `confirmed: true`.
- [ ] Normalize nullable API arrays before iteration.
- [ ] Show Nexus and Maestro installed/latest versions, selected channel, changelog, compatibility and installation method.
- [ ] Add an explicit confirmation dialog, progress, restart-required state and receipt history.
- [ ] Extract conceptual components to colocated files before `SettingsSurface.tsx` exceeds the documented size threshold.
- [ ] Research and reuse existing design-system primitives before adding UI; use semantic HTML, keyboard/focus behavior, SCSS Modules, `--nx-*` tokens and `react-i18next`.
- [ ] Migrate any touched static inline styles or hardcoded visible strings in the update area.
- [ ] Add component tests for check, unavailable, invalid signature, confirmation, apply failure, restart required and rollback.
- [ ] Commit as `feat(settings): add verified nexus update center`.

**Acceptance:** The UI cannot invoke apply without explicit confirmation and the digest returned by check; all visible states are accessible, localized and covered by tests.

### Task 13: Complete tagged release automation and support evidence

**Files:** new `.github/workflows/release.yml`, `.goreleaser.yaml`, release scripts, changelog files, README/support matrix

- [ ] Trigger release publication only from signed/approved version tags after all required CI checks pass on the tagged SHA.
- [ ] Generate stable/beta/nightly metadata from explicit tag conventions; never infer channel from mutable branch state.
- [ ] Publish immutable archives/packages, checksums, SBOMs and attestations; then publish the signed registry manifest referencing those exact digests.
- [ ] Fail closed if version/commit/build date, signing key ID, artifact digest or manifest publication does not match.
- [ ] Create a consolidated Nexus/Maestro changelog feed used by CLI and Update Center.
- [ ] Document supported OS/architecture pairs only where native runner runtime, doctor, build and install smoke evidence exist.
- [ ] Add a release dry run and tag-candidate checklist with rollback/revocation procedure.
- [ ] Commit workflow, changelog and support documentation separately.

**Acceptance:** A dry-run tag reproduces all artifacts from a clean checkout, and the signed manifest resolves only to the artifacts produced by that same SHA.

### Task 14: Enforce the final release gate and branch protections

**Files:** `.github/workflows/ci.yml`, `.github/workflows/release.yml`, quality documentation and repository settings

- [ ] Define required jobs with stable names: frontend, Linux, Windows, macOS, snapshot, deb/rpm smoke and release-gate.
- [ ] Make release-gate verify that all artifacts, manifests, signatures, SBOMs, attestations and smoke receipts belong to the same SHA.
- [ ] Run the complete matrix three times on the final immutable SHA.
- [ ] Apply and verify branch protection on `main` and the campaign branch.
- [ ] Produce a final acceptance report listing SHA, commands, native runner evidence, artifacts, known experimental items and rollback path.
- [ ] Keep the PR Draft until all evidence exists; merge only the whole campaign after review.

**Acceptance:** Three complete final CI runs are green on one SHA, repository rules prevent bypass of required checks, and the acceptance report contains reproducible evidence for every supported platform.

## Required verification commands

Use repository-defined commands where available and keep the CI commands identical:

```bash
bun --version
bun install --cwd web --frozen-lockfile
npm --prefix web run format:check
npm --prefix web run lint
npm --prefix web run lint:styles
npm --prefix web run check:styles
npm --prefix web run typecheck
npm --prefix web run test
npm --prefix web run build
golangci-lint version
golangci-lint config verify
gofmt -l .
go vet ./...
go test ./...
go test -race ./...
make security
make quality
goreleaser check
goreleaser release --snapshot --clean
```

Platform-specific runtime, installer and update tests must execute on the corresponding native runner, not under cross-compilation alone.

## Pre-mortem

1. **ConPTY passes echo but still leaks under real providers.** Early signal: handle/goroutine counts grow across repeated start/stop or child processes remain after host exit. Mitigation: lifecycle stress tests with child/grandchild trees, failure injection at every startup stage and Job Object evidence.
2. **Path migration discovers aliases that violate the old uniqueness model.** Early signal: two legacy records resolve to one filesystem identity. Mitigation: transactional backfill, no automatic merge, blocked-record state, doctor report and explicit reconciliation workflow.
3. **An authentic update is applied using the wrong installation method.** Early signal: detected method conflicts with package receipt or target path ownership. Mitigation: installation receipt, method-specific preflight, confirmation bound to manifest digest and fail-closed apply with backup/rollback.

## Definition of done

- All tasks and both three-green checkpoints are complete on immutable SHAs.
- Windows runtime has bounded ConPTY I/O, secure Named Pipe startup, truthful handshake and Job Object lifecycle.
- Windows/macOS path aliases and native credential semantics are covered on native runners.
- Doctor provides text, JSON, Web and redacted bundle outputs from one implementation.
- Linux packages, signed registry, explicit-confirmation updater, receipts and rollback are operational.
- Branch protections and required checks are enforced.
- No excluded feature entered the campaign, and no platform is advertised without native evidence.
