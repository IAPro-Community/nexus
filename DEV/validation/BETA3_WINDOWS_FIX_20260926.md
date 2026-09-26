# Beta.3 Windows fix validation — 2026-09-26

This report records the Windows shutdown fix applied on top of the current convergence branch. It is a local technical validation record, not a release approval.

## Candidate and fix

- Candidate branch: `fix/beta3-homologation-convergence-20260925`
- Candidate SHA used as base: `b23df384e2aab50545040475e3cff80cfc0f4fb6`
- Fix branch: `fix/beta3-windows-autopilot-20260926`
- Code commit tested: `047f3621fe2a5d268f945e9685ee655c09478178`
- Platform: Windows 10.0.26200, AMD64
- Toolchain: Go 1.25.14, Node 20.15.1

## Change and targeted results

The shared E2E shutdown helper now waits for the spawned server process to exit after termination; it no longer treats taskkill's completion as proof that the server has released its files. A real-child regression test covers that contract.

| Check | Result |
|---|---|
| `node --test web/scripts/process-termination.test.mjs` | PASS |
| `go test -count=1 ./internal/nexus -run 'TestMaestro'` | PASS |
| `go vet ./internal/testutil ./internal/nexus` | PASS |
| Frontend build from this worktree | PASS |
| `go build -trimpath -buildvcs=false -o nexus.exe ./cmd/nexus` | PASS |
| E2E Surface Isolation | PASS; Axe had zero violations; 390px viewport passed |
| E2E HUD | PASS |
| `go test -count=1 -timeout 20m ./...` | FAIL; exit code 1 |

The local `nexus.exe` built for this run reports `version=dev` because release linker flags were not used. Its SHA-256 was `251B20C3D4BD3F13AB0F8D4EFF474892AC612DE4CFD746D3FFFB553B76DD6CDA`.

## Remaining full-suite failures

- `internal/control/driver`: Windows denied removal of a temporary test executable (`ai-browser.exe`); root cause not established.
- `internal/control/web`: three test cleanups could not remove `nexus.db` while another process held it open. The optional `nexus.dev` hosts-file write was denied; no system file was modified.
- `internal/control/workspace`, `internal/core/config`, and `internal/core/provider/adapters/codex`: symlink tests require a Windows privilege unavailable to this account.
- `internal/conversation`: fixture test read unrelated Codex history and did not find its synthetic conversation IDs.
- `internal/nexus`: OpenCode 1.18.30 returned verified enforcement with the canary absent, but the test expects `Tool == "edit"` and received an empty tool field.

Overall result: **FAIL**. The targeted E2E shutdown fix passes, but the full Windows Go suite is not clean; this does not authorize release.

The full raw Go log is retained in the local evidence folder, not committed here, because it includes private local conversation identifiers and filesystem paths. This summary omits those details.
