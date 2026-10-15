-- Opt-in, unattributed usage from the user's local Claude Code history.
-- This table is NEVER merged into measured/provider-billed cost observations.
-- File paths, messages, session prompts, API keys and account identifiers
-- are NOT persisted. source_scope is a fixed, non-account identifier.
CREATE TABLE IF NOT EXISTS external_claude_messages (
  source_scope TEXT NOT NULL,
  message_hash TEXT NOT NULL,
  model TEXT NOT NULL DEFAULT '',
  input_tokens INTEGER NOT NULL CHECK (input_tokens >= 0),
  output_tokens INTEGER NOT NULL CHECK (output_tokens >= 0),
  cache_read_tokens INTEGER NOT NULL CHECK (cache_read_tokens >= 0),
  cache_write_tokens INTEGER NOT NULL CHECK (cache_write_tokens >= 0),
  observed_at TEXT NOT NULL,
  PRIMARY KEY (source_scope, message_hash)
);
CREATE INDEX IF NOT EXISTS idx_external_claude_time
  ON external_claude_messages(source_scope, observed_at DESC);
CREATE TABLE IF NOT EXISTS external_claude_offsets (
  source_scope TEXT NOT NULL,
  file_hash TEXT NOT NULL,
  offset_bytes INTEGER NOT NULL CHECK (offset_bytes >= 0),
  updated_at TEXT NOT NULL,
  PRIMARY KEY (source_scope, file_hash)
);
