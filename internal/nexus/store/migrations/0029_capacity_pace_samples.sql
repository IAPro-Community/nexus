-- Advisory capacity pace samples. Existing observations remain authoritative.
-- A cycle is keyed by the provider's absolute reset time, never a countdown.
-- Account scope is required by the writer; keep history isolated on re-auth.
CREATE TABLE IF NOT EXISTS capacity_pace_samples (
  provider TEXT NOT NULL,
  profile TEXT NOT NULL,
  account_scope TEXT NOT NULL,
  quota_group TEXT NOT NULL DEFAULT '',
  window_kind TEXT NOT NULL,
  reset_at TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  used_percent REAL NOT NULL CHECK (used_percent >= 0 AND used_percent <= 100),
  source TEXT NOT NULL,
  PRIMARY KEY (provider, profile, account_scope, quota_group, window_kind, reset_at, observed_at)
);
CREATE INDEX IF NOT EXISTS idx_capacity_pace_recent
  ON capacity_pace_samples(observed_at DESC);
