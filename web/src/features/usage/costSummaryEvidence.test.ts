import { describe, expect, it } from 'vitest';
import { reportedCostForDisplay } from './costSummaryEvidence';

describe('Cost Summary evidence semantics', () => {
  it('never presents API-equivalent estimates as provider-reported spend', () => {
    expect(reportedCostForDisplay({ reported_cost_observations: 0, reported_cost_microusd: 0 })).toBeNull();
  });

  it('preserves a measured zero instead of treating it as missing', () => {
    expect(reportedCostForDisplay({ reported_cost_observations: 1, reported_cost_microusd: 0 })).toBe(0);
  });

  it('rejects unsafe or invalid amount representations', () => {
    expect(reportedCostForDisplay({ reported_cost_observations: 1, reported_cost_microusd: -1 })).toBeNull();
    expect(
      reportedCostForDisplay({
        reported_cost_observations: 1,
        reported_cost_microusd: Number.MAX_SAFE_INTEGER + 1,
      }),
    ).toBeNull();
  });
});
