import { describe, expect, it } from 'vitest';
import {
  VISUAL_DIFF_DEFAULTS,
  buildVisualDiffRequest,
  formatDiffRatio,
  formatPixelCount,
  parseRegionInput,
  visualDiffTone,
  visualDiffVerdictKey,
  type VisualDiffResult,
} from './visualDiffModel';

const baseResult: VisualDiffResult = {
  total_pixels: 10000,
  compared_pixels: 10000,
  different_pixels: 12,
  diff_ratio: 0.0012,
  pixel_threshold: 12,
  max_diff_ratio: 0.01,
  passed: true,
};

describe('buildVisualDiffRequest', () => {
  it('builds a recheck request clamped to the 5s server window', () => {
    const req = buildVisualDiffRequest({ kind: 'recheck', delayMs: 99999 }, 10, 0.02);
    expect(req.recheck_delay_ms).toBe(5000);
    expect(req.pixel_threshold).toBe(10);
    expect(req.max_diff_ratio).toBe(0.02);
    expect(req.include_artifact).toBe(true);
    expect(req.before_b64).toBeUndefined();
  });

  it('builds an explicit-pair request', () => {
    const req = buildVisualDiffRequest(
      { kind: 'pair', beforeB64: 'AAA', afterB64: 'BBB' },
      VISUAL_DIFF_DEFAULTS.pixelThreshold,
      VISUAL_DIFF_DEFAULTS.maxDiffRatio,
    );
    expect(req.before_b64).toBe('AAA');
    expect(req.after_b64).toBe('BBB');
    expect(req.recheck_delay_ms).toBeUndefined();
  });
});

describe('visualDiffTone + verdict', () => {
  it('maps server truth to tones without inventing outcomes', () => {
    expect(visualDiffTone(null)).toBe('error');
    expect(visualDiffTone({ ...baseResult, passed: true })).toBe('pass');
    expect(visualDiffTone({ ...baseResult, passed: false })).toBe('fail');
  });

  it('mirrors the verdict key from the server measurement', () => {
    expect(visualDiffVerdictKey({ ...baseResult, passed: false })).toBe(
      'browserVisualDiff.verdict.fail',
    );
    expect(visualDiffVerdictKey(null)).toBe('browserVisualDiff.verdict.error');
  });
});

describe('formatting', () => {
  it('formats diff ratio as localized percent', () => {
    expect(formatDiffRatio(0.0231, 'en')).toBe('2.31%');
  });

  it('formats pixel counts with grouping', () => {
    expect(formatPixelCount(1234567, 'en')).toBe('1,234,567');
  });
});

describe('parseRegionInput', () => {
  it('accepts x,y,w,h', () => {
    expect(parseRegionInput('10, 20, 300, 200')).toEqual({ x: 10, y: 20, width: 300, height: 200 });
  });

  it('rejects partial or negative input and empty means no region', () => {
    expect(parseRegionInput('')).toBeNull();
    expect(parseRegionInput('10,20,300')).toBeNull();
    expect(parseRegionInput('-1,20,300,200')).toBeNull();
    expect(parseRegionInput('10,20,0,200')).toBeNull();
    expect(parseRegionInput('a,b,c,d')).toBeNull();
  });
});
