ALTER TABLE composer_sessions ADD COLUMN context_manifest_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE composer_skill_proposals ADD COLUMN recommendation_score INTEGER NOT NULL DEFAULT 0;
