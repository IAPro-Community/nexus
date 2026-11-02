-- Nexus Conductor: global singleton settings and ephemeral worker identity.
-- Additive only: no existing table is altered beyond a backward-compatible column.
CREATE TABLE IF NOT EXISTS nexus_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '{}',
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_nexus_settings_updated ON nexus_settings(updated_at);
ALTER TABLE agents ADD COLUMN identity_kind TEXT NOT NULL DEFAULT 'PERSISTENT';
