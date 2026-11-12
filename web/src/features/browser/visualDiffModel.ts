/**
 * Pure presentation model for the browser visual-diff panel.
 * Server contract: POST /api/v1/browser/{id}/visual-diff — see
 * internal/nexus/visual_diff.go (VisualDiffResult) and
 * internal/control/web/handlers_browser_visual_diff.go.
 */

export type VisualDiffRequestMode =
  { kind: 'recheck'; delayMs: number } | { kind: 'pair'; beforeB64: string; afterB64: string };

export type VisualDiffRequest = {
  recheck_delay_ms?: number;
  before_b64?: string;
  after_b64?: string;
  pixel_threshold: number;
  max_diff_ratio: number;
  include_artifact: boolean;
};

export type VisualDiffResult = {
  total_pixels: number;
  compared_pixels: number;
  different_pixels: number;
  diff_ratio: number;
  pixel_threshold: number;
  max_diff_ratio: number;
  passed: boolean;
  region?: { x: number; y: number; width: number; height: number } | null;
  size_mismatch_note?: string;
  artifact_b64?: string;
  artifact_format?: string;
};

export const VISUAL_DIFF_DEFAULTS = {
  /** Per-channel pixel tolerance (0-255). 12 absorbs JPEG noise without hiding real changes. */
  pixelThreshold: 12,
  /** Max fraction of different pixels for a PASS. */
  maxDiffRatio: 0.01,
  /** Server-side settle window between the two captures (ms). */
  recheckDelayMs: 1200,
} as const;

export function buildVisualDiffRequest(
  mode: VisualDiffRequestMode,
  pixelThreshold: number,
  maxDiffRatio: number,
): VisualDiffRequest {
  const base: VisualDiffRequest = {
    pixel_threshold: pixelThreshold,
    max_diff_ratio: maxDiffRatio,
    include_artifact: true,
  };
  if (mode.kind === 'recheck') {
    return { ...base, recheck_delay_ms: Math.max(0, Math.min(mode.delayMs, 5000)) };
  }
  return { ...base, before_b64: mode.beforeB64, after_b64: mode.afterB64 };
}

export type VisualDiffTone = 'pass' | 'fail' | 'error';

export function visualDiffTone(result: VisualDiffResult | null): VisualDiffTone {
  if (!result) return 'error';
  return result.passed ? 'pass' : 'fail';
}

/** Formats diff_ratio as a localized percentage (e.g. 0.0231 → "2.31%"). */
export function formatDiffRatio(ratio: number, locale: string): string {
  return new Intl.NumberFormat(locale, {
    style: 'percent',
    maximumFractionDigits: 2,
  }).format(ratio);
}

/** Formats pixel counts with locale digit grouping. */
export function formatPixelCount(value: number, locale: string): string {
  return new Intl.NumberFormat(locale).format(value);
}

/**
 * Verdict copy key: the panel mirrors the server measurement without
 * inventing an outcome — the result is the single source of truth.
 */
export function visualDiffVerdictKey(result: VisualDiffResult | null): string {
  if (!result) return 'browserVisualDiff.verdict.error';
  return result.passed ? 'browserVisualDiff.verdict.pass' : 'browserVisualDiff.verdict.fail';
}

/** Guards a region input: empty string → no region; otherwise 4 non-negative ints. */
export function parseRegionInput(value: string): VisualDiffResult['region'] {
  const trimmed = value.trim();
  if (!trimmed) return null;
  const parts = trimmed.split(/[,\s]+/).filter(Boolean);
  if (parts.length !== 4) return null;
  const numbers = parts.map((part) => Number(part));
  if (numbers.some((n) => !Number.isInteger(n) || n < 0)) return null;
  const [x, y, width, height] = numbers;
  if (width === 0 || height === 0) return null;
  return { x, y, width, height };
}
