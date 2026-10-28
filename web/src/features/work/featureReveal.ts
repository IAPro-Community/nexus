import {
  featureAnnouncements, type FeatureAnnouncement,
} from './featureAnnouncements';
import type { FeatureTourSurface } from './featureTourDefinitions';

/**
 * Ephemeral hand-off between the Nexus global "What's New" menu and a tour
 * mounted in a workspace tab. No route mutation and no persistent tracking.
 *
 * Only explicit user selection creates a request. An unmatched item expires
 * instead of unexpectedly opening a tour in an unrelated future session.
 */
export const FEATURE_REVEAL_REQUEST_EVENT = 'nexus:feature-reveal-request';

interface PendingReveal {
  itemId: string;
  projectId: string;
  createdAt: number;
}

let pending: PendingReveal | null = null;
const MAX_REVEAL_AGE_MS = 30_000;

export function requestFeatureReveal(
  itemId: string,
  projectId: string,
  now: number = Date.now(),
): boolean {
  if (!projectId || !featureAnnouncements.some((item) => item.id === itemId)) {
    return false;
  }
  pending = { itemId, projectId, createdAt: now };
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event(FEATURE_REVEAL_REQUEST_EVENT));
  }
  return true;
}

export function consumeFeatureReveal(
  projectId: string,
  surface: FeatureTourSurface,
  isPresent: (target: string) => boolean,
  now: number = Date.now(),
): FeatureAnnouncement | null {
  if (!pending) return null;
  if (now - pending.createdAt > MAX_REVEAL_AGE_MS || now < pending.createdAt) {
    pending = null;
    return null;
  }
  if (pending.projectId !== projectId) return null;
  const item = featureAnnouncements.find((entry) => entry.id === pending?.itemId);
  if (!item) {
    pending = null;
    return null;
  }
  if (!item.surfaces.includes(surface) || !isPresent(item.target)) return null;
  pending = null;
  return item;
}

export function hasPendingFeatureReveal(
  projectId: string,
  surface: FeatureTourSurface,
  now: number = Date.now(),
): boolean {
  if (!pending || pending.projectId !== projectId) return false;
  if (now < pending.createdAt || now - pending.createdAt > MAX_REVEAL_AGE_MS) {
    pending = null;
    return false;
  }
  const item = featureAnnouncements.find((entry) => entry.id === pending?.itemId);
  return Boolean(item && item.surfaces.includes(surface));
}

export function cancelFeatureReveal(): void {
  pending = null;
}
