# Current release status

STATUS: CURRENT
GIT_SHA: be6c793
GENERATED_AT: 2026-09-14T21:35:00Z
ENVIRONMENT: linux/amd64 go1.25.0
EVIDENCE_STREAM_ID: local-validation-2026-09-14-be6c793-v1.0.0-beta.1
WORKTREE_FROZEN: YES

## Version

`1.0.0-beta.1` (tag: `v1.0.0-beta.1`)

## Local verification (2026-09-14)

- `go test ./...` PASS
- `go test -race ./...` PASS
- `make quality` PASS
- `make attention-e2e` PASS
- `make overnight-smoke` PASS
- `make security` PASS
- `make release-proof` PASS (coverage step skipped: `covdata` missing in go1.25.0)
- `nexus version` confirms `1.0.0-beta.1`
- `VERSION CONTRACT PASS` all sources consistent
- No commit, push, or hosted CI invocation performed

This is the **only** document that should be treated as the current release
status for HEAD. Files named `FINAL_*` under `DEV/validation/` without this
directory are historical projections unless they also carry STATUS: CURRENT
and the same SHA.

Canonical evidence is `ValidationEvidenceStream`, not Markdown PASS claims.

Current local verification: `go test ./...`, `go test -race ./...`, `make quality`, `make test-e2e`, `make security`, `GOTOOLCHAIN=go1.25.1 make release-proof`, frontend lint with zero warnings, and Windows cross-compilation of affected tests PASS. Release-proof reports Go coverage at 52.4%; the frontend coverage plugin is not installed. `make docs-verify` remains BLOCKED because the visual manifest cannot certify the uncommitted visual worktree. Hosted same-SHA CI, 8h soak, heterogeneous multi-provider Mission, package smoke, and branch protection remain UNVERIFIED. The current worktree also contains concurrent project documentation changes and is not frozen.

## Latest execution (2026-09-14)

- `GOTOOLCHAIN=go1.25.1 make release-proof`: PASS for all local gates; soak
  evidence remains intentionally missing
- `make quality`: PASS
- `make test-e2e`: PASS
- `go test -race ./...`: PASS
- `make security`: PASS
- frontend lint `--max-warnings=0`: PASS
- Windows test compilation for `scripts`, `internal/control/web` and
  `internal/control/host`: PASS
- `make docs-verify`: BLOCKED by stale visual manifest on uncommitted changes
- No commit, push, hosted CI invocation, ruleset or branch-protection change
  was performed.

## How to refresh

```bash
make quality
make attention-e2e
make overnight-smoke
git rev-parse HEAD
```

Then update `DEV/validation/current/FINAL_HARDENING_REPORT.md`.
