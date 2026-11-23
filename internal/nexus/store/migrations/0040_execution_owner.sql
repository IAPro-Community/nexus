-- 0040: persist the single execution owner of every work plan and mission run.
-- The owner is immutable for the life of a plan/run: NEXUS runs are supervised
-- by the local MissionRunner, MAESTRO runs are dispatched only by the Maestro
-- worker. Existing rows were created by the local runner, so they are NEXUS.
ALTER TABLE work_plans ADD COLUMN execution_owner TEXT NOT NULL DEFAULT 'NEXUS';
ALTER TABLE mission_runs ADD COLUMN execution_owner TEXT NOT NULL DEFAULT 'NEXUS';
CREATE INDEX IF NOT EXISTS idx_mission_runs_execution_owner ON mission_runs(execution_owner, updated_at DESC);
