ALTER TABLE events_metadata ADD COLUMN account_provider_id TEXT NOT NULL DEFAULT '';
ALTER TABLE events_metadata ADD COLUMN account_id TEXT NOT NULL DEFAULT '';
ALTER TABLE events_metadata ADD COLUMN identity_version TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_events_account_scope ON events_metadata(account_provider_id, account_id, identity_version, ts);
