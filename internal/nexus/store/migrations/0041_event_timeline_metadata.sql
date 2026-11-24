-- 0041: durable timeline metadata for the Agent timeline projection.
-- source identifies the producer family of the event, runtime_id keeps the
-- concrete runtime that emitted it, and runtime_generation_id links the row to
-- the durable runtime generation when one already exists. Legacy rows keep
-- empty values instead of fabricated identities.
ALTER TABLE events_metadata ADD COLUMN source TEXT NOT NULL DEFAULT '';
ALTER TABLE events_metadata ADD COLUMN runtime_id TEXT NOT NULL DEFAULT '';
ALTER TABLE events_metadata ADD COLUMN runtime_generation_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_events_source ON events_metadata(source, ts);
CREATE INDEX IF NOT EXISTS idx_events_runtime_generation ON events_metadata(runtime_generation_id, ts);
