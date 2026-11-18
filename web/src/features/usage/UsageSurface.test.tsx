import { describe, expect, it } from 'vitest';
import { FRESHNESS_TICK_MS, USAGE_POLL_MS } from './usageRefresh';
import { normalizeUsageArea, USAGE_AREAS, USAGE_AREA_STORAGE_KEY } from './usageModel';

describe('usage responsiveness contract', () => {
  it('polls soft cache on a one-minute cadence', () => {
    expect(USAGE_POLL_MS).toBe(60_000);
  });

  it('ticks relative freshness without waiting for the next poll', () => {
    expect(FRESHNESS_TICK_MS).toBe(30_000);
    expect(FRESHNESS_TICK_MS).toBeLessThan(USAGE_POLL_MS);
  });
});

describe('usage area contract', () => {
  it('orders the three areas as accounts, consumption and diagnostics', () => {
    expect([...USAGE_AREAS]).toEqual(['accounts', 'consumption', 'diagnostics']);
  });

  it('restores the persisted area only when it is a known value', () => {
    expect(normalizeUsageArea('diagnostics')).toBe('diagnostics');
    expect(normalizeUsageArea('accounts')).toBe('accounts');
    expect(normalizeUsageArea('nope')).toBe('accounts');
    expect(USAGE_AREA_STORAGE_KEY).toBe('nexus.usage.area');
  });
});
