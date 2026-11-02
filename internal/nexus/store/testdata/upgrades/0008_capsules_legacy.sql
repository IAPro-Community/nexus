-- Upgrade fixture: flow_context_capsules as shaped by pre-0008 builds.
-- Missing flow_revision and updated_at, content carried as payload_json.
-- Used by TestLegacyCapsulesAreMigratedWithLedger: applied over a database
-- that already contains the canonical schema to simulate version drift.
DROP TABLE IF EXISTS flow_context_capsules;
CREATE TABLE flow_context_capsules (
    id           TEXT PRIMARY KEY,
    run_id       TEXT NOT NULL,
    step_id      TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    created_at   TEXT NOT NULL
);
