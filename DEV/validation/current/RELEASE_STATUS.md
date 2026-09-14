# Current release status

STATUS: CURRENT
GIT_SHA: a560cf701d3bf7b43ce74b638014ab943037e7e4
GENERATED_AT: 2026-09-14T03:56:10Z
ENVIRONMENT: linux/amd64 go1.25.0
EVIDENCE_STREAM_ID: local-validation-2026-09-14-a560cf7
WORKTREE_FROZEN: NO (uncommitted autopilot fixes present)

This is the **only** document that should be treated as the current release
status for HEAD. Files named `FINAL_*` under `DEV/validation/` without this
directory are historical projections unless they also carry STATUS: CURRENT
and the same SHA.

Canonical evidence is `ValidationEvidenceStream`, not Markdown PASS claims.

Current local verification: focused `go test ./internal/control/web/ -run TestIsWithinAllowedRoots`, `go test ./internal/runtime/ -run TestNoBrowserBinDir`, `go test ./scripts/ -run TestResolveE2ERoot`, `go test ./internal/nexus/promotion/ -v` and `go vet ./internal/app/` PASS. Full `make quality`, hosted Windows E2E on this SHA, 8h soak, heterogeneous multi-provider Mission and branch protection remain UNVERIFIED. Autopilot fixes in worktree: Windows portability (fs_test TempDir guard, credentials exec-bit skip, e2e chmod tolerate), CLI Nexus UX to stderr (provider stdin isolated), bracketed paste captured with 32KiB/200-line bound.

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
