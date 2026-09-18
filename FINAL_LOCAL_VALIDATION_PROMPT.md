# IAPro Nexus — Independent Local Validation Prompt

Validate the current candidate from scratch. Do not trust prior reports.

1. Confirm branch, SHA, remote, dirty files, and untracked files. Never reset or
   clean the worktree.
2. Read `AGENTS.md`, `DEV/validation/CURRENT_STATE_AUDIT.md`, and the platform
   support matrix. Treat only fresh command output as evidence.
3. Install frontend dependencies with the repository's frozen Bun workflow, then
   run from `web/`: format check, typecheck, ESLint, Stylelint, stylesheet
   allowlist, unit tests, build, E2E, Axe, and visual smoke.
4. Run sequentially after the frontend embed is complete: `gofmt -l .`,
   `go vet ./...`, `go test ./...`, and `go test -race ./...`.
5. Run `make build-desktop-wails` on Linux and record whether required
   WebKit/GTK dependencies are present; `make build-desktop` is only the Go
   compile fallback, not packaging evidence. Confirm the artifact under
   `cmd/nexus-desktop/build/bin/`.
6. On real Windows, run the full Go suite, ConPTY/Named Pipe tests, runtime/web
   tests, PowerShell smoke, and the Desktop build/smoke. Record each failing test
   by name; do not use sleeps, skips, or retries to mask failures.
7. On real macOS, run the full race suite, PTY/socket/runtime/web tests, binary
   build, installer smoke, and Desktop build/smoke.
8. Run the security scanner and update/installer negative tests. Verify unsigned,
   tampered, wrong-target, wrong-architecture, expired, downgrade, and unknown
   key inputs fail closed.
9. Run GoReleaser snapshot only after all required platform jobs pass, and record
   artifact names, checksums, component, OS, architecture, and source SHA.
10. Inspect the final GitHub CI run for this exact SHA. A different SHA cannot be
    combined with this candidate. Do not call the candidate GO unless Frontend,
    Linux, Windows, macOS, Desktop, Browser, Axe, Visual, Security, Packaging,
    and Snapshot all have fresh PASS evidence.
11. Audit public readiness: confirm `LICENSE` matches README badges/text,
    platform claims match `docs/platform/PLATFORM_SUPPORT_MATRIX.md`, and the
    Community Preview files (`CODE_OF_CONDUCT.md`, `SECURITY.md`, `SUPPORT.md`,
    `ROADMAP.md`, `GOVERNANCE.md`, `CHANGELOG.md`, `.github/CODEOWNERS`, issue
    and pull-request templates) contain no invented maintainers, signing,
    support, or release claims. Confirm installer docs state the current
    checksum-only limitation until a trusted Ed25519 keyring is published.

Required final output: command, timestamp, SHA, result, raw failure summary, and
remaining blockers. Include the public-readiness audit and distinguish the
remote committed tree from any dirty local worktree. Valid verdicts are `GO`,
`CONDITIONAL_GO`, or `NO_GO`; never convert unavailable native evidence into
PASS.
