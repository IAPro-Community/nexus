import { describe, expect, it } from 'vitest';
import { collectHudHitRegions, hudRegionFromCandidate, type HudHitCandidate } from './hudHitRegions';

const base: HudHitCandidate = {
  ignore: false,
  pointerEvents: 'auto',
  display: 'block',
  visibility: 'visible',
  opacity: '1',
  rect: { left: 40, top: 0, width: 180, height: 60, right: 220, bottom: 60 },
};

describe('hudRegionFromCandidate', () => {
  it('keeps an interactive rectangle', () => {
    expect(hudRegionFromCandidate(base)).toEqual({ x: 40, y: 0, width: 180, height: 60 });
  });

  it('drops transparent, hidden and zero-sized nodes', () => {
    expect(hudRegionFromCandidate({ ...base, pointerEvents: 'none' })).toBeNull();
    expect(hudRegionFromCandidate({ ...base, display: 'none' })).toBeNull();
    expect(hudRegionFromCandidate({ ...base, visibility: 'hidden' })).toBeNull();
    expect(hudRegionFromCandidate({ ...base, opacity: '0' })).toBeNull();
    expect(
      hudRegionFromCandidate({
        ...base,
        rect: { left: 10, top: 10, width: 0, height: 0, right: 10, bottom: 10 },
      }),
    ).toBeNull();
  });

  it('drops decorative overlays marked with data-hud-hit=ignore', () => {
    expect(hudRegionFromCandidate({ ...base, ignore: true })).toBeNull();
  });

  it('drops nodes entirely outside the viewport', () => {
    expect(
      hudRegionFromCandidate({
        ...base,
        rect: { left: -400, top: 0, width: 100, height: 60, right: -300, bottom: 60 },
      }),
    ).toBeNull();
  });

  it('treats an empty opacity as fully visible', () => {
    expect(hudRegionFromCandidate({ ...base, opacity: '' })).not.toBeNull();
  });
});

describe('native click-through surface collection', () => {
  it('selects explicit controls instead of the full transparent panel rectangle', () => {
    let queried = '';
    const control = {
      dataset: {},
      getBoundingClientRect: () => ({
        left: 40,
        top: 8,
        width: 172,
        height: 43,
        right: 212,
        bottom: 51,
      }),
    };
    const root = {
      ownerDocument: {
        defaultView: {
          getComputedStyle: () => ({
            pointerEvents: 'auto',
            display: 'block',
            visibility: 'visible',
            opacity: '1',
          }),
        },
      },
      querySelectorAll: (selector: string) => {
        queried = selector;
        return [control];
      },
    } as unknown as ParentNode;
    expect(collectHudHitRegions(root)).toEqual([
      { x: 40, y: 8, width: 172, height: 43 },
    ]);
    expect(queried).toBe('[data-hud-hit-area]');
  });

  it('filters decorative or invisible control rectangles, never the backdrop', () => {
    const fakeNodes = [
      {
        dataset: { hudHit: 'ignore' },
        getBoundingClientRect: () => ({ left: 0, top: 0, width: 500, height: 500, right: 500, bottom: 500 }),
      },
    ];
    const root = {
      ownerDocument: {
        defaultView: {
          getComputedStyle: () => ({
            pointerEvents: 'auto',
            display: 'block',
            visibility: 'visible',
            opacity: '1',
          }),
        },
      },
      querySelectorAll: () => fakeNodes,
    } as unknown as ParentNode;
    expect(collectHudHitRegions(root)).toEqual([]);
  });
});
