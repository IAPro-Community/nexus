import { describe, expect, it } from 'vitest';
import {
  SHARE_TTL_MAX_SECONDS,
  formatShareExpiry,
  parseShareTTLSeconds,
  shareGrantTone,
  shareRemainingBucket,
  sortShareGrants,
} from './shareGrantsModel';
import type { ObservationShareGrant } from '../../types';

const now = new Date('2026-11-12T12:00:00Z');

const grant = (over: Partial<ObservationShareGrant>): ObservationShareGrant => ({
  id: 'g1',
  project_id: 'p1',
  role: 'viewer',
  created_at: '2026-11-10T12:00:00Z',
  ...over,
});

describe('shareGrantTone', () => {
  it('mirrors server state: revoked wins, then expiry, then active', () => {
    expect(shareGrantTone(grant({}), now)).toBe('active');
    expect(shareGrantTone(grant({ expires_at: '2026-11-12T11:00:00Z' }), now)).toBe('expired');
    expect(
      shareGrantTone(
        grant({ expires_at: '2026-11-12T13:00:00Z', revoked_at: '2026-11-12T09:00:00Z' }),
        now,
      ),
    ).toBe('revoked');
  });
});

describe('expiry formatting', () => {
  it('uses Intl with the active locale', () => {
    const out = formatShareExpiry('2026-11-12T13:45:00Z', 'en');
    expect(out).toMatch(/Nov|13/);
  });
});

describe('shareRemainingBucket', () => {
  it('buckets remaining time for active TTL grants', () => {
    expect(shareRemainingBucket(grant({}), now)).toBe('unknown');
    expect(shareRemainingBucket(grant({ expires_at: '2026-11-15T12:00:00Z' }), now)).toBe('days');
    expect(shareRemainingBucket(grant({ expires_at: '2026-11-12T15:00:00Z' }), now)).toBe('hours');
    expect(shareRemainingBucket(grant({ expires_at: '2026-11-12T12:30:00Z' }), now)).toBe('soon');
  });
});

describe('parseShareTTLSeconds', () => {
  it('accepts positive integers up to the 7d cap', () => {
    expect(parseShareTTLSeconds('3600')).toBe(3600);
    expect(parseShareTTLSeconds(String(SHARE_TTL_MAX_SECONDS))).toBe(SHARE_TTL_MAX_SECONDS);
  });

  it('rejects empty, non-integer, non-positive and over-cap input', () => {
    expect(parseShareTTLSeconds('')).toBeNull();
    expect(parseShareTTLSeconds('abc')).toBeNull();
    expect(parseShareTTLSeconds('0')).toBeNull();
    expect(parseShareTTLSeconds('-5')).toBeNull();
    expect(parseShareTTLSeconds('1.5')).toBeNull();
    expect(parseShareTTLSeconds(String(SHARE_TTL_MAX_SECONDS + 1))).toBeNull();
  });
});

describe('sortShareGrants', () => {
  it('orders active first, then newest within each tone', () => {
    const sorted = sortShareGrants([
      grant({
        id: 'revoked-old',
        revoked_at: '2026-11-11T00:00:00Z',
        created_at: '2026-11-09T00:00:00Z',
      }),
      grant({ id: 'expired', expires_at: '2026-11-12T11:00:00Z' }),
      grant({ id: 'active-old', created_at: '2026-11-10T00:00:00Z' }),
      grant({ id: 'active-new', created_at: '2026-11-12T00:00:00Z' }),
    ]);
    expect(sorted.map((item) => item.id)).toEqual([
      'active-new',
      'active-old',
      'expired',
      'revoked-old',
    ]);
  });
});
