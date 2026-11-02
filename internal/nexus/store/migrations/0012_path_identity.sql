-- Additive path identity metadata. canonical_path remains for compatibility.
ALTER TABLE projects ADD COLUMN display_path TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN identity_kind TEXT NOT NULL DEFAULT '';
ALTER TABLE projects ADD COLUMN identity_key TEXT NOT NULL DEFAULT '';
UPDATE projects SET display_path = canonical_path WHERE display_path = '';
