# Handoff

## Canonical hardening branch handoff — 2026-09-02 (updated 2026-09-03)

Branch: `feat/nexus-canonical-hardening`
HEAD SHA: `4daedcf` (commit: `feat: canonical hardening - global attention, auto arrange, workspace recovery`)

All canonical hardening work is committed. The branch contains a single
commit on top of `d7cb609886762bef9abc89faf028f42c8443564b` with the full
attention system, auto arrange, workspace recovery, agent recovery, provider
session discovery, deep links, global search, notification transport, and
service worker.

### What shipped (commit 4daedcf)

- **Attention system**: State/Source types, MemoryStream (idempotent ingest,
  filter, replay), GlobalAttentionManager (singleton, per-project subscriptions),
  GlobalAttentionStream (single WS connection, dedup, broadcast).
- **Auto arrange**: Smart/Grid/Columns/Rows/FocusActive modes, deterministic,
  minimum size clamping, 10+ tests.
- **Workspace recovery**: VisualStore (geometry persistence per project),
  AgentRecoveryInfo with states (RUNNING/RECOVERABLE/LOST/STOPPED), 10+ tests.
- **Provider session discovery**: ProviderSessionDiscovery interface,
  SessionDiscoveryManager (Register/DiscoverAll/DiscoverByProvider), tests.
- **Deep links**: parseDeepLink/generateDeepLink/applyDeepLink for
  `/projects/:id?view=agent:<id>`, tests.
- **Global search**: Fuzzy matching with scoring across projects/agents/
  provider sessions/flow runs, tests.
- **Attention center panel**: React component with priority sorting.
- **Notification transport**: IN_APP/BROWSER/WEB_PUSH/NATIVE_OS abstraction.
- **Service worker**: push, notificationclick (focus/openWindow),
  pushsubscriptionchange, secret redaction.
- **Notification model**: sanitizeAttentionText for safe titles.
- **Product contract**: PRODUCT_CONTRACT.md with all 15 invariants.
- **Preservation matrix**: All 25+ capabilities marked preserved.
- **Validation reports**: FINAL_VALIDATION_REPORT.md, COMPONENT_AND_TEST_
  COVERAGE.md, CAPABILITY_PRESERVATION_MATRIX.md, AGENT_DECK_HARVEST_REPORT.md.

### Verification

All gates pass as of commit 4daedcf:
- `go test -race -count=1 ./...` — 407+ tests, 0 failures
- `go vet ./...` — 0 warnings
- `go build ./cmd/nexus` — clean
- `vitest run` — 156 tests across 40 files, 0 failures
- `git diff --check` — no whitespace errors

### Remaining items (not in this commit)

- VAPID key management / Push Subscriptions persistence (backend)
- Provider-specific status detectors (Claude Code, Codex, Gemini CLI, Aider)
- Provider Hooks adapter (detect/install/verify/remove)
- tmux/zellij future adapters (ExternalSessionDiscovery)
- Full Golden Journey test with live Nexus process
- Full Golden Flow test with live providers
- Cross-platform verification (macOS, Windows)
- Frontend integration of AttentionCenter, DeepLink, GlobalSearch into main app shell
- WorkspaceVisualState migration to SQLite (currently JSON file)

### How to continue

Pick any remaining item above and implement it on this branch. After each
substantive change, run `go test -race -count=1 ./...` and
`cd web && node node_modules/.bin/vitest run` to verify. Commit with
conventional commit style.
