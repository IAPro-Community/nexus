import { describe, expect, it } from 'vitest';
import { cacheRate, costEvidenceLabel } from './consumptionAnalyticsModel';

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
