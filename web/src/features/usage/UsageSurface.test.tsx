import { describe, expect, it } from 'vitest';
import { FRESHNESS_TICK_MS, USAGE_POLL_MS } from './usageRefresh';

describe('usage responsiveness contract', () => {
  it('polls soft cache on a one-minute cadence', () => {
    expect(USAGE_POLL_MS).toBe(60_000);
  });

  it('ticks relative freshness without waiting for the next poll', () => {
    expect(FRESHNESS_TICK_MS).toBe(30_000);
    expect(FRESHNESS_TICK_MS).toBeLessThan(USAGE_POLL_MS);
  });
});
