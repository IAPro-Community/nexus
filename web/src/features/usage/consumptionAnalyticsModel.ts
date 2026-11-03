import type { CostAnalyticsTotals } from '../../types';

// A cache rate is valid only for observations that reported BOTH plain input
// and cache-read token counts. Missing cached tokens must not be treated as 0.
export function cacheRate(
  totals: Pick<
    CostAnalyticsTotals,
    'known_cache_observations' | 'cache_prompt_input_tokens' | 'cache_prompt_read_tokens'
  >,
): number | null {
  if (totals.known_cache_observations === 0) return null;
  const denominator = totals.cache_prompt_input_tokens + totals.cache_prompt_read_tokens;
  if (!Number.isFinite(denominator) || denominator <= 0) return null;
  return Math.max(0, Math.min(1, totals.cache_prompt_read_tokens / denominator));
}

// Coverage describes presence of a numerical amount, not billing certainty.
// An API-equivalent estimate is a known value but never a measured invoice.
export function costEvidenceLabel(
  known: number,
  total: number,
): 'NONE' | 'PARTIAL' | 'COMPLETE' {
  if (total <= 0 || known <= 0) return 'NONE';
  return known >= total ? 'COMPLETE' : 'PARTIAL';
}
