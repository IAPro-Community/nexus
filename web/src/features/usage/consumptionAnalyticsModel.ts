import type { CostAnalyticsTotals, CostObservation, DailyCostAnalytics } from '../../types';

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
export function costEvidenceLabel(known: number, total: number): 'NONE' | 'PARTIAL' | 'COMPLETE' {
  if (total <= 0 || known <= 0) return 'NONE';
  return known >= total ? 'COMPLETE' : 'PARTIAL';
}

/* ---- Daily chart metrics (F2.6a) ---- */

export type DailyMetric = 'invocations' | 'tokens' | 'cost';

/**
 * Returns the measured value of a daily metric, or null when the day carries
 * no evidence for it. Missing evidence is never coerced to zero. Cost keeps
 * provider-reported values only: estimates and inferred amounts are separate
 * evidence classes and must not be merged into a fictional bill.
 */
export function dailyMetricValue(bucket: DailyCostAnalytics, metric: DailyMetric): number | null {
  if (metric === 'invocations') return bucket.invocations_total;
  if (metric === 'tokens') {
    if (bucket.known_token_observations === 0) return null;
    return bucket.input_tokens + bucket.output_tokens;
  }
  if (bucket.reported_cost_observations === 0) return null;
  return bucket.provider_reported_microusd;
}

/** A metric is offered only when at least one day actually measured it. */
export function dailyMetricAvailable(
  buckets: readonly DailyCostAnalytics[],
  metric: DailyMetric,
): boolean {
  return buckets.some((bucket) => dailyMetricValue(bucket, metric) !== null);
}

/** Max measured value across days for proportional bar widths; at least 1. */
export function dailyMetricMax(
  buckets: readonly DailyCostAnalytics[],
  metric: DailyMetric,
): number {
  return Math.max(1, ...buckets.map((bucket) => dailyMetricValue(bucket, metric) ?? 0));
}

/* ---- Session / mission drill-down grouping (F2.6a) ---- */

export type BreakdownGroupKind = 'mission' | 'agent';

export interface BreakdownGroup {
  /** Raw mission_run_id / agent_id; empty string means unattributed rows. */
  key: string;
  kind: BreakdownGroupKind;
  /** Number of observations in this group; groups always have at least one. */
  count: number;
}

/**
 * Groups observations by mission_run_id or agent_id using only real evidence:
 * a group appears only when at least one observation carries that id, and
 * counts never extrapolate beyond the fetched observations. Observations
 * without the id stay visible under the unattributed group (empty key)
 * instead of being silently dropped.
 */
export function groupObservations(
  observations: readonly CostObservation[],
  kind: BreakdownGroupKind,
): BreakdownGroup[] {
  const counts = new Map<string, number>();
  for (const observation of observations) {
    const key =
      kind === 'mission' ? (observation.mission_run_id ?? '') : (observation.agent_id ?? '');
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  return [...counts.entries()]
    .map(([key, count]) => ({ key, kind, count }))
    .sort((a, b) => b.count - a.count || a.key.localeCompare(b.key));
}

/** Observations are fetched most-recent-first by the backend; keep that
 * contract even if the producer changes, so drill-down stays deterministic. */
export function byRecencyDesc(a: CostObservation, b: CostObservation): number {
  return b.observed_at.localeCompare(a.observed_at);
}

/* ---- Formatting helpers shared by the consumption surfaces ---- */

export function formatMicroUSD(microUSD: number): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 4,
  }).format(microUSD / 1_000_000);
}

export function formatAmount(value: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(value);
}

export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return '—';
  if (ms < 1000) return `${formatAmount(ms)} ms`;
  return `${new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(ms / 1000)} s`;
}

export function formatTokens(tokens: number): string {
  return formatAmount(tokens);
}

export function formatThroughput(tokensPerSecond: number): string {
  return `${new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(tokensPerSecond)} t/s`;
}
