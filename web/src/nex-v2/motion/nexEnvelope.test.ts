import { describe, expect, it } from 'vitest';
import {
  NEX_EXPRESSIONS,
  VIRTUAL_ARTBOARD_SIZE,
  computeContainTransform,
  computeLayerExtents,
  isWithinSafeEnvelope,
} from './nexEnvelope';

describe('Nex 2.0 safe motion envelope', () => {
  it('contains all 9 expressions across 24 time samples and orbit phases without clipping', () => {
    const blends = [0, 0.5, 1];
    for (const expr of NEX_EXPRESSIONS) {
      for (let i = 0; i < 24; i++) {
        const t = i * 0.25;
        for (const blend of blends) {
          const extents = computeLayerExtents(expr, t, t, blend);
          expect(
            isWithinSafeEnvelope(extents),
            `Clipping detected for expr=${expr} t=${t} blend=${blend}: ${JSON.stringify(extents)}`,
          ).toBe(true);
        }
      }
    }
  });

  it('produces bounded contain transforms for every viewport preset and DPI', () => {
    const presets = [128, 160, 192];
    const dpis = [1, 1.25, 1.5, 2];
    for (const size of presets) {
      for (const dpr of dpis) {
        const transform = computeContainTransform(size, size, dpr);
        expect(transform.backingSizePx).toBe(Math.ceil(size * dpr));
        expect(transform.scale * VIRTUAL_ARTBOARD_SIZE).toBeCloseTo(transform.backingSizePx, 5);
        expect(transform.offsetX).toBeGreaterThan(0);
        expect(transform.offsetY).toBeGreaterThan(0);
      }
    }
  });
});
