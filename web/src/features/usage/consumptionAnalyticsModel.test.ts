import { describe, expect, it } from 'vitest';
import type { CostObservation, DailyCostAnalytics } from '../../types';
import {
  byRecencyDesc,
  cacheRate,
  costEvidenceLabel,
  dailyMetricAvailable,
  dailyMetricMax,
  dailyMetricValue,
  formatDuration,
  groupObservations,
} from './consumptionAnalyticsModel';

describe('consumption analytics evidence semantics', () => {
  it('never treats absent cache data as a zero percent rate', () => {
    expect(
      cacheRate({
        known_cache_observations: 0,
        cache_prompt_input_tokens: 200,
        cache_prompt_read_tokens: 0,
      }),
    ).toBeNull();

    expect(
      cacheRate({
        known_cache_observations: 1,
        cache_prompt_input_tokens: 0,
        cache_prompt_read_tokens: 0,
      }),
    ).toBeNull();
  });

  it('calculates cache rate only from paired input/cache-read evidence', () => {
    expect(
      cacheRate({
        known_cache_observations: 2,
        cache_prompt_input_tokens: 75,
        cache_prompt_read_tokens: 25,
      }),
    ).toBeCloseTo(0.25);

    expect(
      cacheRate({
        known_cache_observations: 1,
        cache_prompt_input_tokens: 0,
        cache_prompt_read_tokens: 100,
      }),
    ).toBe(1);
  });

  it('describes numerical coverage, not billing confidence', () => {
    expect(costEvidenceLabel(0, 0)).toBe('NONE');
    expect(costEvidenceLabel(0, 8)).toBe('NONE');
    expect(costEvidenceLabel(3, 8)).toBe('PARTIAL');
    expect(costEvidenceLabel(8, 8)).toBe('COMPLETE');
  });
});

describe('daily chart metrics', () => {
  const bucket: DailyCostAnalytics = {
    date: '2026-11-17',
    observation_count: 3,
    invocations_total: 3,
    invocations_failed: 1,
    known_cost_observations: 0,
    unknown_cost_observations: 3,
    known_token_observations: 0,
    reported_cost_observations: 0,
    equivalent_cost_observations: 0,
    inferred_cost_observations: 0,
    known_cache_observations: 0,
    cache_prompt_input_tokens: 0,
    cache_prompt_read_tokens: 0,
    input_tokens: 0,
    cached_input_tokens: 0,
    output_tokens: 0,
    reasoning_tokens: 0,
    duration_ms: 0,
    provider_reported_microusd: 0,
    api_equivalent_microusd: 0,
    inferred_microusd: 0,
    unclassified_microusd: 0,
  };

  it('always exposes invocations, which exist for every observed day', () => {
    expect(dailyMetricValue(bucket, 'invocations')).toBe(3);
    expect(dailyMetricAvailable([bucket], 'invocations')).toBe(true);
  });

  it('returns null for tokens when the day has no measured token evidence', () => {
    expect(dailyMetricValue(bucket, 'tokens')).toBeNull();
    expect(dailyMetricAvailable([bucket], 'tokens')).toBe(false);
  });

  it('sums measured input and output tokens only when evidence exists', () => {
    const measured = {
      ...bucket,
      known_token_observations: 2,
      input_tokens: 120,
      output_tokens: 80,
    };
    expect(dailyMetricValue(measured, 'tokens')).toBe(200);
    expect(dailyMetricAvailable([measured], 'tokens')).toBe(true);
  });

  it('returns null for cost when the day has no provider-reported value', () => {
    expect(dailyMetricValue(bucket, 'cost')).toBeNull();
    expect(dailyMetricAvailable([bucket], 'cost')).toBe(false);
  });

  it('keeps provider-reported cost separate from estimates and inferred amounts', () => {
    const mixed = {
      ...bucket,
      reported_cost_observations: 0,
      equivalent_cost_observations: 2,
      api_equivalent_microusd: 5_000_000,
      inferred_microusd: 3_000_000,
    };
    expect(dailyMetricValue(mixed, 'cost')).toBeNull();

    const reported = { ...mixed, reported_cost_observations: 1, provider_reported_microusd: 750 };
    expect(dailyMetricValue(reported, 'cost')).toBe(750);
  });

  it('computes proportional maxima ignoring missing evidence', () => {
    const strong = { ...bucket, invocations_total: 9 };
    const empty: DailyCostAnalytics = { ...bucket, invocations_total: 0, observation_count: 0 };
    expect(dailyMetricMax([strong, empty], 'invocations')).toBe(9);
    expect(dailyMetricMax([], 'invocations')).toBe(1);
  });
});

describe('observation drill-down grouping', () => {
  const observation = (overrides: Partial<CostObservation>): CostObservation => ({
    id: overrides.id ?? Math.random().toString(36).slice(2),
    duration_ms: 0,
    cost_kind: 'UNKNOWN',
    source: 'test',
    observed_at: '2026-11-17T12:00:00Z',
    ...overrides,
  });

  it('only creates groups backed by at least one observation', () => {
    const groups = groupObservations(
      [
        observation({ id: 'a', mission_run_id: 'run-1' }),
        observation({ id: 'b', mission_run_id: 'run-1' }),
        observation({ id: 'c', mission_run_id: 'run-2' }),
      ],
      'mission',
    );
    expect(groups).toEqual([
      { key: 'run-1', kind: 'mission', count: 2 },
      { key: 'run-2', kind: 'mission', count: 1 },
    ]);
  });

  it('keeps unattributed rows visible instead of dropping them', () => {
    const groups = groupObservations(
      [
        observation({ id: 'a', agent_id: 'agent-7' }),
        observation({ id: 'b' }),
        observation({ id: 'c' }),
      ],
      'agent',
    );
    expect(groups).toEqual([
      { key: '', kind: 'agent', count: 2 },
      { key: 'agent-7', kind: 'agent', count: 1 },
    ]);
  });

  it('returns no groups for an empty observation window', () => {
    expect(groupObservations([], 'mission')).toEqual([]);
    expect(groupObservations([], 'agent')).toEqual([]);
  });

  it('orders observations by most recent observed_at first', () => {
    const older = observation({ id: 'old', observed_at: '2026-11-01T00:00:00Z' });
    const newer = observation({ id: 'new', observed_at: '2026-11-17T00:00:00Z' });
    expect([older, newer].sort(byRecencyDesc).map((row) => row.id)).toEqual(['new', 'old']);
  });
});

describe('formatting helpers', () => {
  it('keeps unmeasured durations explicit instead of zero', () => {
    expect(formatDuration(0)).toBe('—');
  });
});
