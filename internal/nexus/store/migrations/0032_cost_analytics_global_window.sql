-- Global consumption analytics filters by observed_at even without a project.
-- Existing project/time index does not cover global time-window queries.
-- Reuse cost_observations; do not create a parallel stats table.
CREATE INDEX IF NOT EXISTS idx_cost_observations_global_time
    ON cost_observations(observed_at DESC);
