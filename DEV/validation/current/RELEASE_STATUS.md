# Current release status

STATUS: CURRENT  
GIT_SHA: cdb81f1f2faa7b260af071fab1272b6b475e3470
GENERATED_AT: 2026-09-14T02:58:38Z
ENVIRONMENT: linux/amd64 go1.25.0
EVIDENCE_STREAM_ID: local-validation-2026-09-13-cdb81f1
WORKTREE_FROZEN: NO (334 tracked/untracked changes observed)

This is the **only** document that should be treated as the current release
status for HEAD. Files named `FINAL_*` under `DEV/validation/` without this
directory are historical projections unless they also carry STATUS: CURRENT
and the same SHA.

Canonical evidence is `ValidationEvidenceStream`, not Markdown PASS claims.

Current local verification: `make quality`, `make web-verify`, `make security`,
`make build`, `make build-desktop-wails`, `go test ./... -count=1`,
`go test -race ./... -count=1`, `go vet ./...`, `make lint-go`,
`make attention-e2e`, `make overnight-smoke` and `make overnight-soak
DURATION=45m` PASS. The default Go 1.25.0 `make release-proof` is blocked
because that toolchain lacks `covdata`; the complete gate passes with
`GOTOOLCHAIN=go1.25.1 make release-proof` (coverage 52.4%).
`make docs-verify` is BLOCKED because the visual manifest is stale against
uncommitted frontend changes. Direct smoke for providers without a valid
session is recorded as UNVERIFIED in
`validation/current/real-provider-mission/`; authenticated Codex Mission
`run_06G9VJV1K6J649PN2EMEHR1EMG` reached `COMPLETED_VERIFIED` with a distinct
read-only reviewer Agent and package/global verification evidence. Production
trust-root provisioning, native Windows/macOS, hosted same-SHA browser/release
evidence, and an eight-hour soak remain UNVERIFIED.

## Latest execution (2026-09-14)

- `make attention-e2e`: PASS
- `make overnight-smoke`: PASS
- `make overnight-soak DURATION=45m`: PASS (bounded harness completed)
- `make release-proof` (default Go 1.25.0): BLOCKED at `go tool covdata`
- `GOTOOLCHAIN=go1.25.1 make release-proof`: PASS (coverage total 52.4%)
- Codex Mission `run_06G9VJV1K6J649PN2EMEHR1EMG`: PASS;
  authenticated implementer routing, distinct read-only review, package and
  global verification receipts all passed.
- `make smoke-providers`: BLOCKED because live usage had no LIVE AGY/Claude/
  Gemini rows; direct and isolated autonomous attempts are detailed in
  `real-provider-mission/README.md` and remain UNVERIFIED.
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
