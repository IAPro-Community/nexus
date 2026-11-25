export type NotificationPrefs = {
  notificationsEnabled: boolean;
  soundEnabled: boolean;
  categories: NotificationCategoryPrefs;
};

export interface NotificationCategoryPrefs {
  action_required: boolean;
  auth_permission: boolean;
  critical_failure: boolean;
  quota_exhausted: boolean;
  quota_low: boolean;
  completion_recovery: boolean;
  runtime_connection: boolean;
}

export interface CapacityNotificationCategoryFields {
  notify_action_required: boolean;
  notify_auth_permission: boolean;
  notify_critical_failure: boolean;
  notify_quota_exhausted: boolean;
  notify_quota_low: boolean;
  notify_completion_recovery: boolean;
  notify_runtime_connection: boolean;
}

const STORAGE_KEY = 'iapro:nexus:notification-prefs:v1';

export const DEFAULT_NOTIFICATION_PREFS: NotificationPrefs = {
  notificationsEnabled: true,
  soundEnabled: true,
  categories: {
    action_required: true,
    auth_permission: true,
    critical_failure: true,
    quota_exhausted: true,
    quota_low: false,
    completion_recovery: false,
    runtime_connection: false,
  },
};

export function loadNotificationPrefs(): NotificationPrefs {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return { ...DEFAULT_NOTIFICATION_PREFS };
    const parsed = JSON.parse(raw) as Partial<NotificationPrefs> & {
      categories?: Partial<NotificationCategoryPrefs>;
    };
    return {
      notificationsEnabled: parsed.notificationsEnabled !== false,
      soundEnabled: parsed.soundEnabled !== false,
      categories: {
        ...DEFAULT_NOTIFICATION_PREFS.categories,
        ...parsed.categories,
      },
    };
  } catch {
    return { ...DEFAULT_NOTIFICATION_PREFS };
  }
}

export function cacheNotificationCategories(categories: NotificationCategoryPrefs): void {
  saveNotificationPrefs({ ...loadNotificationPrefs(), categories });
}

export function cacheCapacityNotificationCategories(
  preferences: CapacityNotificationCategoryFields,
): void {
  cacheNotificationCategories({
    action_required: preferences.notify_action_required,
    auth_permission: preferences.notify_auth_permission,
    critical_failure: preferences.notify_critical_failure,
    quota_exhausted: preferences.notify_quota_exhausted,
    quota_low: preferences.notify_quota_low,
    completion_recovery: preferences.notify_completion_recovery,
    runtime_connection: preferences.notify_runtime_connection,
  });
}

export function saveNotificationPrefs(prefs: NotificationPrefs): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    // ignore quota / private mode
  }
}

let audioCtx: AudioContext | null = null;

export function playAttentionSound(): void {
  try {
    const Ctx =
      window.AudioContext ||
      (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
    if (!Ctx) return;
    if (!audioCtx) audioCtx = new Ctx();
    const ctx = audioCtx;
    if (ctx.state === 'suspended') {
      void ctx.resume();
    }
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.type = 'sine';
    osc.frequency.value = 880;
    gain.gain.value = 0.0001;
    osc.connect(gain);
    gain.connect(ctx.destination);
    const now = ctx.currentTime;
    gain.gain.exponentialRampToValueAtTime(0.08, now + 0.02);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.22);
    osc.start(now);
    osc.stop(now + 0.25);
  } catch {
    // browsers may block until a user gesture
  }
}
