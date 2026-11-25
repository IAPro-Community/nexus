import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const folder = dirname(fileURLToPath(import.meta.url));
const component = readFileSync(join(folder, 'NotificationCategorySettings.tsx'), 'utf8');
const settings = readFileSync(join(folder, 'SettingsSurface.tsx'), 'utf8');

describe('notification category settings screen contract', () => {
  it('exposes each API category as an accessible switch in the Notifications tab', () => {
    const fields = [
      'notify_action_required',
      'notify_auth_permission',
      'notify_critical_failure',
      'notify_quota_exhausted',
      'notify_quota_low',
      'notify_completion_recovery',
      'notify_runtime_connection',
    ];
    for (const field of fields) expect(component).toContain(field);
    expect(component).toContain('<Switch');
    expect(component).toContain('data-tour="notification-category-settings"');
    expect(component).toContain('patchNotificationCategories');
    expect(settings).toContain('<NotificationCategorySettings />');
  });
});
