import { afterAll, describe, expect, it } from 'vitest';
import {
  DEFAULT_NOTIFICATION_PREFS,
  loadNotificationPrefs,
  saveNotificationPrefs,
} from './notificationPrefs';

const values = new Map<string, string>();
const previousWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');
Object.defineProperty(globalThis, 'window', {
  configurable: true,
  value: {
    localStorage: {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
    },
  },
});

afterAll(() => {
  if (previousWindow) Object.defineProperty(globalThis, 'window', previousWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});

describe('notification preference compatibility', () => {
  it('fills category defaults when reading preferences saved by older versions', () => {
    values.set(
      'iapro:nexus:notification-prefs:v1',
      JSON.stringify({ notificationsEnabled: false }),
    );
    expect(loadNotificationPrefs()).toEqual({
      notificationsEnabled: false,
      soundEnabled: true,
      categories: DEFAULT_NOTIFICATION_PREFS.categories,
    });
  });

  it('round-trips editable categories with the existing local sound/browser settings', () => {
    const preferences = {
      ...DEFAULT_NOTIFICATION_PREFS,
      categories: { ...DEFAULT_NOTIFICATION_PREFS.categories, quota_low: true },
      soundEnabled: false,
    };
    saveNotificationPrefs(preferences);
    expect(loadNotificationPrefs()).toEqual(preferences);
  });
});
