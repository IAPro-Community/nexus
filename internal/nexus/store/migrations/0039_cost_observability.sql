-- 0039: cost ledger observability (F2.2).
-- Additive only: session attribution plus TTFT/throughput evidence on the
-- existing cost_observations table, and a partial unique index so provider
-- replay of the same (source, invocation_id) is idempotent. NULL ttft_ms and
-- output_tokens_per_second mean "not measured", never synthetic zero. Legacy
-- rows (and runtime rows) with an empty invocation_id stay outside the
-- dedupe key and remain insertable multiple times.
ALTER TABLE cost_observations ADD COLUMN session_id TEXT NOT NULL DEFAULT '';
ALTER TABLE cost_observations ADD COLUMN ttft_ms INTEGER CHECK (ttft_ms IS NULL OR ttft_ms >= 0);
ALTER TABLE cost_observations ADD COLUMN output_tokens_per_second REAL CHECK (output_tokens_per_second IS NULL OR output_tokens_per_second >= 0);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cost_observations_invocation_source
    ON cost_observations(source, invocation_id)
    WHERE invocation_id != '' AND source != '';
