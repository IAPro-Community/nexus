CREATE TABLE IF NOT EXISTS context_readiness (
  project_id TEXT PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
  state TEXT NOT NULL,
  fingerprint_json TEXT NOT NULL DEFAULT '{}',
  reason TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);
