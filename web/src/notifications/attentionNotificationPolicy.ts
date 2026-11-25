import type { NotificationCategoryPrefs } from './notificationPrefs';

export type NativePopupCategory = keyof NotificationCategoryPrefs;

const EPISODE_STORAGE_KEY = 'iapro:nexus:notification-episodes:v1';
const MAX_REMEMBERED_EPISODES = 500;

export function notificationCategoryForReason(reason: string): NativePopupCategory | null {
  switch (reason.trim().toUpperCase()) {
    case 'QUESTION':
    case 'APPROVAL':
    case 'ACTION_REQUIRED':
      return 'action_required';
    case 'AUTHENTICATION':
    case 'AUTH_REQUIRED':
    case 'AUTH':
    case 'PERMISSION':
    case 'PERMISSION_REQUIRED':
      return 'auth_permission';
    case 'ERROR':
    case 'CRITICAL_FAILURE':
    case 'FAILED':
      return 'critical_failure';
    case 'QUOTA_EXHAUSTED':
      return 'quota_exhausted';
    case 'QUOTA_LOW':
      return 'quota_low';
    case 'TASK_COMPLETED':
    case 'RECOVERED':
      return 'completion_recovery';
    case 'CONNECTION':
    case 'RUNTIME_CONNECTION':
    case 'CONNECTED':
    case 'DISCONNECTED':
      return 'runtime_connection';
    default:
      return null;
  }
}

export function isNotificationCategoryEnabled(
  reason: string,
  categories: NotificationCategoryPrefs,
): boolean {
  const category = notificationCategoryForReason(reason);
  return category !== null && categories[category];
}

/** Stable, bounded local record so polling and browser restarts do not repeat an episode popup. */
export function markNotificationEpisodeDelivered(episodeKey: string): boolean {
  if (!episodeKey) return false;
  try {
    const storage = window.localStorage;
    const remembered = JSON.parse(storage.getItem(EPISODE_STORAGE_KEY) || '[]') as unknown;
    const episodes = Array.isArray(remembered)
      ? remembered.filter((item): item is string => typeof item === 'string')
      : [];
    const identity = hashEpisodeKey(episodeKey);
    if (episodes.includes(identity)) return false;
    storage.setItem(
      EPISODE_STORAGE_KEY,
      JSON.stringify([...episodes.slice(-(MAX_REMEMBERED_EPISODES - 1)), identity]),
    );
    return true;
  } catch {
    return true;
  }
}

export function clearNotificationEpisode(episodeKey: string): void {
  if (!episodeKey) return;
  try {
    const remembered = JSON.parse(
      window.localStorage.getItem(EPISODE_STORAGE_KEY) || '[]',
    ) as unknown;
    if (!Array.isArray(remembered)) return;
    const identity = hashEpisodeKey(episodeKey);
    window.localStorage.setItem(
      EPISODE_STORAGE_KEY,
      JSON.stringify(
        remembered.filter((item): item is string => typeof item === 'string' && item !== identity),
      ),
    );
  } catch {
    // Storage may be unavailable in private or restricted browser contexts.
  }
}

function hashEpisodeKey(value: string): string {
  let hash = 0xcbf29ce484222325n;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= BigInt(value.charCodeAt(index));
    hash = BigInt.asUintN(64, hash * 0x100000001b3n);
  }
  return hash.toString(16).padStart(16, '0');
}
