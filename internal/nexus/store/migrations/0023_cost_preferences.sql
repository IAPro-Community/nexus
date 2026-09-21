-- 0023: local Cost Intelligence preferences.
-- Threshold value 0 disables that threshold. Preferences never imply billing
-- truth and never authorize killing an in-flight process.
CREATE TABLE IF NOT EXISTS cost_preferences (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 1,
    window_days INTEGER NOT NULL DEFAULT 7 CHECK (window_days BETWEEN 1 AND 365),
    soft_limit_microusd INTEGER NOT NULL DEFAULT 0 CHECK (soft_limit_microusd >= 0),
    hard_limit_microusd INTEGER NOT NULL DEFAULT 0 CHECK (hard_limit_microusd >= 0),
    emergency_limit_microusd INTEGER NOT NULL DEFAULT 0 CHECK (emergency_limit_microusd >= 0),
    anomaly_multiplier REAL NOT NULL DEFAULT 3.0 CHECK (anomaly_multiplier >= 1.0),
    anomaly_minimum_samples INTEGER NOT NULL DEFAULT 5 CHECK (anomaly_minimum_samples BETWEEN 3 AND 1000),
    anomaly_min_cost_microusd INTEGER NOT NULL DEFAULT 100000 CHECK (anomaly_min_cost_microusd >= 0),
    updated_at TEXT NOT NULL
);
