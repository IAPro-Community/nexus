CREATE TABLE IF NOT EXISTS agent_prompt_receipts (
    id          TEXT PRIMARY KEY,
    agent_id    TEXT NOT NULL,
    runtime_id  TEXT NOT NULL,
    skill_ids   TEXT NOT NULL,
    prompt_hash TEXT NOT NULL,
    source      TEXT NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_agent_prompt_receipts_agent ON agent_prompt_receipts(agent_id, created_at DESC);
