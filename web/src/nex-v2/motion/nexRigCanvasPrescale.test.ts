import { describe, expect, it, vi } from 'vitest';
import { ART_CANVAS_SIZE } from './nexEnvelope';
import { prescaleLayer, prescaleRatio, prescaleTargetPx } from './nexRigCanvas';

describe('NexRigCanvas texture pre-scaling', () => {
  it('bakes at the HUD viewport scale instead of the full 1254px artboard', () => {
    // 256 CSS px at DPR 2 -> 512px backing store; 1.25 headroom -> 0.51 ratio.
    const ratio = prescaleRatio(512);
    expect(ratio).toBeCloseTo((512 / ART_CANVAS_SIZE) * 1.25, 6);
    expect(ratio).toBeLessThan(1);
    expect(prescaleTargetPx(ART_CANVAS_SIZE, ratio)).toBe(512);
    expect(prescaleTargetPx(253, ratio)).toBe(130);
  });

  it('never upscales and clamps the bake to the HUD ceiling', () => {
    expect(prescaleRatio(0)).toBeLessThanOrEqual(1);
    expect(prescaleRatio(Number.NaN)).toBeLessThanOrEqual(1);
    expect(prescaleTargetPx(ART_CANVAS_SIZE, 1)).toBe(512);
    expect(prescaleTargetPx(921, 0.5)).toBe(461);
    expect(prescaleTargetPx(600, 1)).toBe(512);
  });

  it('draws the source once into a smaller canvas and keeps the ratio', () => {
    const drawImage = vi.fn();
    const createElement = vi.fn(() => ({
      width: 0,
      height: 0,
      getContext: () => ({
        imageSmoothingEnabled: false,
        imageSmoothingQuality: 'low',
        drawImage,
      }),
    }));
    vi.stubGlobal('document', { createElement });
    const source = {
      width: ART_CANVAS_SIZE,
      height: ART_CANVAS_SIZE,
    } as unknown as HTMLImageElement;

    const surface = prescaleLayer(
      source,
      ART_CANVAS_SIZE,
      ART_CANVAS_SIZE,
      0.5,
    ) as HTMLCanvasElement;

    expect(surface).not.toBe(source);
    expect(surface.width).toBe(512);
    expect(surface.height).toBe(512);
    expect(drawImage).toHaveBeenCalledWith(source, 0, 0, 512, 512);
    vi.unstubAllGlobals();
  });

  it('falls back to the source bitmap when no 2D surface is available', () => {
    const createElement = vi.fn(() => ({
      width: 0,
      height: 0,
      getContext: () => null,
    }));
    vi.stubGlobal('document', { createElement });
    const source = { width: 800, height: 800 } as unknown as HTMLImageElement;
    expect(prescaleLayer(source, 800, 800, 0.5)).toBe(source);
    vi.unstubAllGlobals();

    vi.stubGlobal('document', undefined);
    expect(prescaleLayer(source, 800, 800, 0.5)).toBe(source);
    expect(prescaleLayer(source, 800, 800, 1)).toBe(source);
    vi.unstubAllGlobals();
  });
});
