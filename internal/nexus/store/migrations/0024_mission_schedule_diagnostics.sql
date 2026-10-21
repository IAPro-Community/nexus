ALTER TABLE mission_schedules ADD COLUMN last_error TEXT NOT NULL DEFAULT '';
ALTER TABLE mission_schedules ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE mission_schedules ADD COLUMN last_attempt_at TEXT;
