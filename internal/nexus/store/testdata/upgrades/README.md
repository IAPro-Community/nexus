# Store upgrade fixtures

SQL fixtures simulating database shapes written by older Nexus builds.
Each fixture is applied over a live database by an upgrade test in
`store_upgrade_test.go`, which then reopens the store (running migrations
plus idempotent schema repairs) and asserts that legitimate rows survive
with content intact and every repair recorded in `schema_repair_ledger`.

| Fixture | Simulates | Asserts |
|---|---|---|
| `0008_capsules_legacy.sql` | Pre-0008 `flow_context_capsules` (no `flow_revision`/`updated_at`, `payload_json` naming) | Rows migrated to canonical shape, content preserved, ledger entry with `copied_rows` |

Rules for new fixtures: only additive, historically plausible shapes; every
fixture must have a test that fails if any row becomes query-invisible.
