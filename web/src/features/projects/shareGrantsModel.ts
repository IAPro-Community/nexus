/**
 * Pure presentation model for the workspace-sharing card
 * (capability workspace-sharing, viewer grants).
 */

import type { ObservationShareGrant } from '../../types';

export type ShareGrantTone = 'active' | 'revoked' | 'expired';

/**
 * Mirrors the server truth without inventing outcomes: revoked wins over
 * expired (a revoked grant never becomes active again), then expiry.
 */
export function shareGrantTone(
  grant: Pick<ObservationShareGrant, 'revoked_at' | 'expires_at'>,
  now: Date = new Date(),
): ShareGrantTone {
  if (grant.revoked_at) return 'revoked';
  if (grant.expires_at && new Date(grant.expires_at).getTime() <= now.getTime()) return 'expired';
  return 'active';
}

/** Formats the expiry as a localized date-time (Intl, not hand-rolled). */
export function formatShareExpiry(iso: string, locale: string): string {
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(iso));
}

/** Relative remaining time bucket for an active TTL grant. */
export type ShareRemaining = 'days' | 'hours' | 'soon' | 'unknown';

export function shareRemainingBucket(
  grant: Pick<ObservationShareGrant, 'expires_at'>,
  now: Date = new Date(),
): ShareRemaining {
  if (!grant.expires_at) return 'unknown';
  const remainingMs = new Date(grant.expires_at).getTime() - now.getTime();
  if (remainingMs <= 0) return 'unknown';
  if (remainingMs > 48 * 60 * 60 * 1000) return 'days';
  if (remainingMs > 2 * 60 * 60 * 1000) return 'hours';
  return 'soon';
}

/** The TTL input accepts seconds; the server caps at 7d and refuses >cap. */
export const SHARE_TTL_MAX_SECONDS = 7 * 24 * 60 * 60;

export function parseShareTTLSeconds(raw: string): number | null {
  const trimmed = raw.trim();
  if (!trimmed) return null;
  const value = Number(trimmed);
  if (!Number.isFinite(value) || !Number.isInteger(value) || value <= 0) return null;
  if (value > SHARE_TTL_MAX_SECONDS) return null;
  return value;
}

/** Active grants first, then by newest creation — stable and derived. */
export function sortShareGrants(grants: ObservationShareGrant[]): ObservationShareGrant[] {
  const toneRank: Record<ShareGrantTone, number> = { active: 0, expired: 1, revoked: 2 };
  return [...grants].sort((a, b) => {
    const tone = toneRank[shareGrantTone(a)] - toneRank[shareGrantTone(b)];
    if (tone !== 0) return tone;
    return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
  });
}
