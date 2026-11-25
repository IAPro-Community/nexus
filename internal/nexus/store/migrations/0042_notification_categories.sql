ALTER TABLE capacity_preferences ADD COLUMN notify_action_required INTEGER NOT NULL DEFAULT 1;
ALTER TABLE capacity_preferences ADD COLUMN notify_auth_permission INTEGER NOT NULL DEFAULT 1;
ALTER TABLE capacity_preferences ADD COLUMN notify_critical_failure INTEGER NOT NULL DEFAULT 1;
ALTER TABLE capacity_preferences ADD COLUMN notify_quota_exhausted INTEGER NOT NULL DEFAULT 1;
ALTER TABLE capacity_preferences ADD COLUMN notify_quota_low INTEGER NOT NULL DEFAULT 0;
ALTER TABLE capacity_preferences ADD COLUMN notify_completion_recovery INTEGER NOT NULL DEFAULT 0;
ALTER TABLE capacity_preferences ADD COLUMN notify_runtime_connection INTEGER NOT NULL DEFAULT 0;

-- Preserve the existing global native-notifications opt-out for installations
-- that already have a saved capacity preferences row.
UPDATE capacity_preferences
SET notify_action_required = native_notifications_enabled,
    notify_auth_permission = native_notifications_enabled,
    notify_critical_failure = native_notifications_enabled,
    notify_quota_exhausted = native_notifications_enabled
WHERE id = 1;
