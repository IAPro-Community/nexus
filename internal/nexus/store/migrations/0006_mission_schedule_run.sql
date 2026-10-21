ALTER TABLE mission_schedules ADD COLUMN run_id TEXT;
CREATE INDEX IF NOT EXISTS idx_mission_schedules_run ON mission_schedules(run_id);
