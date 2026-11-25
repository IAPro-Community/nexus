import { afterAll, afterEach, describe, expect, it } from 'vitest';
import {
  clearNotificationEpisode,
  isNotificationCategoryEnabled,
  markNotificationEpisodeDelivered,
  notificationCategoryForReason,
} from './attentionNotificationPolicy';
import { DEFAULT_NOTIFICATION_PREFS } from './notificationPrefs';

const storage = new Map<string, string>();
const localStorage = {
  getItem: (key: string) => storage.get(key) ?? null,
  setItem: (key: string, value: string) => storage.set(key, value),
};
const previousWindow = Object.getOwnPropertyDescriptor(globalThis, 'window');

Object.defineProperty(globalThis, 'window', {
  configurable: true,
  value: { localStorage },
});

afterEach(() => storage.clear());
afterAll(() => {
  if (previousWindow) Object.defineProperty(globalThis, 'window', previousWindow);
  else Reflect.deleteProperty(globalThis, 'window');
});

describe('notification category policy', () => {
  it.each([
    ['QUESTION', 'action_required'],
    ['AUTHENTICATION', 'auth_permission'],
    ['PERMISSION_REQUIRED', 'auth_permission'],
    ['ERROR', 'critical_failure'],
    ['QUOTA_EXHAUSTED', 'quota_exhausted'],
    ['QUOTA_LOW', 'quota_low'],
    ['TASK_COMPLETED', 'completion_recovery'],
    ['CONNECTED', 'runtime_connection'],
  ])('maps %s to %s', (reason, category) => {
    expect(notificationCategoryForReason(reason)).toBe(category);
  });

  it('uses priority defaults while keeping routine pop-ups off', () => {
    const categories = DEFAULT_NOTIFICATION_PREFS.categories;
    expect(isNotificationCategoryEnabled('QUESTION', categories)).toBe(true);
    expect(isNotificationCategoryEnabled('QUOTA_EXHAUSTED', categories)).toBe(true);
    expect(isNotificationCategoryEnabled('QUOTA_LOW', categories)).toBe(false);
    expect(isNotificationCategoryEnabled('TASK_COMPLETED', categories)).toBe(false);
    expect(isNotificationCategoryEnabled('CONNECTED', categories)).toBe(false);
    expect(isNotificationCategoryEnabled('UNKNOWN', categories)).toBe(false);
  });

  it('suppresses the same episode across reloads but accepts a new episode', () => {
    expect(markNotificationEpisodeDelivered('episode-1')).toBe(true);
    expect(markNotificationEpisodeDelivered('episode-1')).toBe(false);
    expect(markNotificationEpisodeDelivered('episode-2')).toBe(true);
    clearNotificationEpisode('episode-1');
    expect(markNotificationEpisodeDelivered('episode-1')).toBe(true);
  });
});
