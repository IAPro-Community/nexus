# Nexus V0 checkpoint

- Phase: Plans 01-02 partial.
- Baseline: `c96885218b3cff319f481aed4423dc5fd7b52a30`.
- Worktree: `.worktrees/nexus-v0-implementation`.
- Baseline evidence: `go test ./...`, `npm run web:typecheck`, `npm run web:test`, and `npm run web:build` passed.
- Latest validation: Go full suite, `go vet ./...`, frontend typecheck/lint/tests/build, embedded bundle sync, and `git diff --check` PASS.
- Additions: presentation persistence, desktop renderer, launcher routing, Ask existing Agent API/UI, stale-attention filtering and durable browser dismissal.
- Next: Project Shell and remaining plans; Windows/macOS runtime and real-provider E2E remain external/platform gates.
