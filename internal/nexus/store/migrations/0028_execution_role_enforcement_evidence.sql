ALTER TABLE runtime_execution_roles ADD COLUMN enforcement_evidence TEXT NOT NULL DEFAULT '';
ALTER TABLE runtime_execution_roles ADD COLUMN enforcement_verified_at TEXT;
