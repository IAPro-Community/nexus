import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { pushNotifications } from './PushNotificationManager';

type MockNotification = {
  close: ReturnType<typeof vi.fn>;
  onclick: null | (() => void);
};

describe('pushNotifications', () => {
  let notificationCtor: ReturnType<typeof vi.fn>;
  let register: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    notificationCtor = vi.fn(function thisNotification(): MockNotification {
      return { close: vi.fn(), onclick: null };
    });
    Object.assign(notificationCtor, {
      permission: 'default' as NotificationPermission,
      requestPermission: vi.fn(async () => {
        (notificationCtor as ReturnType<typeof vi.fn> & { permission: NotificationPermission }).permission = 'granted';
        return 'granted' as NotificationPermission;
      }),
    });

    register = vi.fn(async () => {
      throw new Error('service worker failed');
    });

    Object.defineProperty(globalThis, 'window', {
      value: { hasFocus: () => false, focus: vi.fn(), Notification: notificationCtor },
      configurable: true,
    });
    Object.defineProperty(globalThis, 'document', {
      value: { hidden: true },
      configurable: true,
    });
    Object.defineProperty(globalThis, 'navigator', {
      value: { serviceWorker: { register } },
      configurable: true,
    });
    Object.defineProperty(globalThis, 'Notification', {
      value: notificationCtor,
      configurable: true,
    });

    const state = pushNotifications as unknown as Record<string, unknown>;
    state.permission = 'default';
    state.lastNotifiedKey = '';
    state.lastNotifiedTime = 0;
    state.seenAttentionEvents = new Set<string>();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('keeps browser notification fallback when the service worker fails', async () => {
    await expect(pushNotifications.requestPermission()).resolves.toBe('granted');

    expect(pushNotifications.sendPush({
      runtimeId: 'rt-1',
      projectId: 'prj-1',
      projectName: 'Payments',
      reason: 'QUESTION',
      context: 'Need approval',
      attentionEventId: 'evt-1',
      view: 'work',
    })).toBeUndefined();

    expect(register).toHaveBeenCalledTimes(1);
    expect(notificationCtor).toHaveBeenCalledTimes(1);
  });
});
