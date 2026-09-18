ALTER TABLE events_metadata ADD COLUMN correlation_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_events_correlation ON events_metadata(correlation_id, ts);
