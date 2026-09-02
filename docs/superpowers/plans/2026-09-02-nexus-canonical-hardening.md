# IAPro Nexus Canonical Hardening Implementation Plan

## Global constraints

- Work only on `feat/nexus-canonical-hardening`.
- Preserve existing abstractions; no second runner/scheduler/host.
- TDD for every code change: RED, GREEN, REFACTOR.
- Keep event payloads bounded and redacted.
- After each task: focused tests, commit, task review and P0/P1 resolution.

## Tasks

### 1. Fix baseline registry race

Files: `internal/control/registry/registry.go`, focused registry/host tests.
Add a lifecycle mutex protecting singleton pointer and `sync.Once` replacement;
prove concurrent detector access and test reset are race-free. Run focused race
test, then full Go tests and commit.

### 2. Canonical product contracts and fitness tests

Files: `PRODUCT_CONTRACT.md`, `internal/nexus/*_contract_test.go`,
`internal/nexus/*_fitness_test.go`, `DEV/CAPABILITY_PRESERVATION_MATRIX.md`.
Encode stable AgentID/runtime-generation, direct-with-missing-context, shell
and view invariants, Flow DAG/capsule/receipt/verification invariants, and
single scheduler/runner checks against existing implementations.

### 3. Typed attention domain and global aggregation

Files: `internal/nexus/attention/*`, event/store integration files and tests.
Implement normalized states and sources, bounded/redacted AttentionEvent,
idempotent ingestion, bounded replay and ProjectAttentionSummary with the
specified priority. Reuse EventBus and existing persistence seams.

### 4. Global web stream and notification policy

Files: `internal/control/web/*`, `web/src/notifications/*`, app integration,
service-worker asset/tests and security docs.
Expose one authenticated global stream with reconnect/backoff/replay IDs;
implement IN_APP/BROWSER/WEB_PUSH policy and event-ID deduplication. Add safe
VAPID configuration/subscription endpoints only when the existing server can
support them without secrets in logs or client storage.

### 5. Deterministic auto-arrange and durable visual state

Files: `web/src/workspace/arrange.ts`, workspace provider/renderer/tests,
backend layout model/migration/tests, UI controls and reports.
Implement Smart/Grid/Columns/Rows/Focus/Undo/Reset as geometry-only actions;
cover 0/1/2/3/4/5/6/10/20, minimum sizes, small viewports, determinism and
undo. Persist geometry/focus/presentation by project/workspace/view.

### 6. Recovery, provider capabilities and sessions

Files: `internal/control/driver/*`, `internal/nexus/continuity.go`, store/API,
frontend Sessions/Recovery/Search/preview models and tests.
Make resume/fork/discovery/adopt explicit driver capabilities; restore Views
before runtimes; create new runtime generations; mark dead shells LOST; never
claim verified resume without driver evidence. Add local fuzzy search and
bounded previews without transcript persistence.

### 7. Agent-Deck harvest and documentation

Files: `DEV/AGENT_DECK_HARVEST_REPORT.md`, `THIRD_PARTY_NOTICES.md` only if
literal code is copied, docs/security updates.
Record upstream feature/area/behavior, ADOPT/ADAPT/REJECT, literal-copy and
attribution decisions, and Nexus tests. Prefer behavior adaptation; no tmux
dependency or Bubble Tea port.

### 8. Golden journeys and final validation

Files: golden journey tests, all required validation reports.
Automate project switch/background attention/deep link/restart and canonical
Flow A → B || C → D. Run every executable frontend/backend/security command,
record exact results and limitations, create a clean source ZIP only for GO,
then compute SHA-256.
