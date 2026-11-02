-- Structural contract checkpoint for flow evidence.
-- The corresponding repair is intentionally implemented in Go so it can
-- inspect PRAGMA metadata, preserve incompatible legacy tables, and migrate
-- rows without relying on SQLite-specific ALTER TABLE quirks.
SELECT 1;
