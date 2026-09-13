# Current release status

STATUS: CURRENT  
GIT_SHA: eed65e5a7e46adc896165c94accb7fc73fc6750f
GENERATED_AT: 2026-09-13T19:02:00Z
ENVIRONMENT: linux/amd64 go1.25.0
EVIDENCE_STREAM_ID: local-validation-2026-09-13-eed65e5
WORKTREE_FROZEN: NO (88 tracked/untracked changes observed)

This is the **only** document that should be treated as the current release
status for HEAD. Files named `FINAL_*` under `DEV/validation/` without this
directory are historical projections unless they also carry STATUS: CURRENT
and the same SHA.

Canonical evidence is `ValidationEvidenceStream`, not Markdown PASS claims.

Current local verification: `make quality`, `make web-verify`, `make security`,
`make build`, `make build-desktop-wails`, `go test ./... -count=1`,
`go test -race ./... -count=1`, `go vet ./...`, `make lint-go`, and the update
gate PASS. `make docs-verify` is BLOCKED because the visual manifest is stale
against uncommitted frontend changes. Authenticated provider Missions,
production trust-root provisioning, native Windows/macOS, hosted same-SHA
browser/release evidence, and an eight-hour soak remain UNVERIFIED.

## How to refresh

```bash
make quality
make attention-e2e
make overnight-smoke
git rev-parse HEAD
```

Then update `DEV/validation/current/FINAL_HARDENING_REPORT.md`.
