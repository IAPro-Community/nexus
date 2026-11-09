# Attention Radar: structured choice — 2026-11-09

- Branch: `release/v1.0.0`; repository HEAD at test time: `e75e03c1`.
- Worktree was dirty: the implementation under test was uncommitted. This is not exact-SHA release evidence.
- Command: `NEXUS_E2E_ARTIFACTS_DIR=DEV/validation/current/attention-structured-choice npm --prefix web run test:e2e:attention-radar-process-api-focus` — PASS.
- The browser used authenticated Nexus APIs and two real supervised ACP fixture processes. The UI submitted Alpha's listed `Allow once` option with its interaction ID; Alpha settled and Beta stayed pending. The test also opened Beta's real SessionHost terminal WebSocket.
- Screenshots: `process-attention-structured-options.png` and `process-attention-radar-beta-pending.png`.
- The ACP processes are deterministic protocol fixtures. No external model/provider was called; no distributed Nexus app was tested. Screenshots are real browser captures of this fixture scenario, not release homologation.
- Focused tests: Web 18/18; Go attention/structured packages passed. `make quality-full` passed (Web 131 files/826 tests, Go full and race suites, Go lint, govulncheck); six pre-existing ESLint warnings, zero errors. `make build` passed.
- Result: implementation/evidence checkpoint only. A4, the 48-capability acceptance matrix, provider verification, exact-SHA distribution, platform/signing gates, and overall GO remain open.
