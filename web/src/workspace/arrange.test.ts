import { describe, expect, it } from 'vitest';
import {
  arrangeWindows,
  type ArrangeMode,
  type ArrangeWindowGeometry,
  type ArrangeWindowInput,
} from './arrange';

function windows(
  count: number,
  overrides: Partial<ArrangeWindowInput> = {},
): ArrangeWindowInput[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `view:${index + 1}`,
    minWidth: 360,
    minHeight: 240,
    ...overrides,
  }));
}

function expectWithinViewport(
  geometries: ArrangeWindowGeometry[],
  viewport: { width: number; height: number },
): void {
  geometries.forEach((geometry) => {
    expect(geometry.x).toBeGreaterThanOrEqual(0);
    expect(geometry.y).toBeGreaterThanOrEqual(0);
    expect(geometry.width).toBeGreaterThan(0);
    expect(geometry.height).toBeGreaterThan(0);
    expect(geometry.x + geometry.width).toBeLessThanOrEqual(viewport.width);
    expect(geometry.y + geometry.height).toBeLessThanOrEqual(viewport.height);
  });
}

function expectNoOverlap(geometries: ArrangeWindowGeometry[]): void {
  for (let index = 0; index < geometries.length; index += 1) {
    for (let otherIndex = index + 1; otherIndex < geometries.length; otherIndex += 1) {
      const a = geometries[index];
      const b = geometries[otherIndex];
      const overlaps =
        a.x < b.x + b.width &&
        a.x + a.width > b.x &&
        a.y < b.y + b.height &&
        a.y + a.height > b.y;
      expect(overlaps).toBe(false);
    }
  }
}

describe('arrangeWindows', () => {
  it('returns no geometry for empty inputs across every mode', () => {
    const modes: ArrangeMode[] = ['smart', 'grid', 'columns', 'rows', 'focus-active'];

    modes.forEach((mode) => {
      expect(arrangeWindows({ mode, viewport: { width: 1440, height: 900 }, windows: [] })).toEqual([]);
    });
  });

  it('fills the viewport for a single window', () => {
    const geometries = arrangeWindows({
      mode: 'grid',
      viewport: { width: 1440, height: 900 },
      windows: windows(1),
    });

    expect(geometries).toEqual([{ id: 'view:1', x: 0, y: 0, width: 1440, height: 900 }]);
  });

  it('lays out two windows as equal columns', () => {
    const geometries = arrangeWindows({
      mode: 'columns',
      viewport: { width: 1200, height: 800 },
      windows: windows(2),
    });

    expect(geometries).toEqual([
      { id: 'view:1', x: 0, y: 0, width: 600, height: 800 },
      { id: 'view:2', x: 600, y: 0, width: 600, height: 800 },
    ]);
  });

  it('lays out three windows as equal rows', () => {
    const geometries = arrangeWindows({
      mode: 'rows',
      viewport: { width: 900, height: 900 },
      windows: windows(3),
    });

    expect(geometries).toEqual([
      { id: 'view:1', x: 0, y: 0, width: 900, height: 300 },
      { id: 'view:2', x: 0, y: 300, width: 900, height: 300 },
      { id: 'view:3', x: 0, y: 600, width: 900, height: 300 },
    ]);
  });

  it('gives the active window the primary region in focus-active mode', () => {
    const geometries = arrangeWindows({
      mode: 'focus-active',
      viewport: { width: 1200, height: 800 },
      activeId: 'view:2',
      windows: windows(4),
    });

    expect(geometries).toEqual([
      { id: 'view:1', x: 720, y: 0, width: 480, height: 267 },
      { id: 'view:2', x: 0, y: 0, width: 720, height: 800 },
      { id: 'view:3', x: 720, y: 267, width: 480, height: 267 },
      { id: 'view:4', x: 720, y: 534, width: 480, height: 266 },
    ]);
  });

  it('uses a deterministic 2x2 grid for four windows', () => {
    const geometries = arrangeWindows({
      mode: 'grid',
      viewport: { width: 1200, height: 800 },
      windows: windows(4),
    });

    expect(geometries).toEqual([
      { id: 'view:1', x: 0, y: 0, width: 600, height: 400 },
      { id: 'view:2', x: 600, y: 0, width: 600, height: 400 },
      { id: 'view:3', x: 0, y: 400, width: 600, height: 400 },
      { id: 'view:4', x: 600, y: 400, width: 600, height: 400 },
    ]);
  });

  it('keeps grid layouts deterministic and geometry-only for 5, 6, 10 and 20 windows', () => {
    const counts = [5, 6, 10, 20];

    counts.forEach((count) => {
      const input = { mode: 'grid' as const, viewport: { width: 1600, height: 900 }, windows: windows(count) };
      const first = arrangeWindows(input);
      const second = arrangeWindows(input);

      expect(first).toEqual(second);
      expect(first.map((geometry) => geometry.id)).toEqual(input.windows.map((window) => window.id));
      expectWithinViewport(first, input.viewport);
      expectNoOverlap(first);
    });
  });

  it('chooses columns, rows, focus-active or grid in smart mode based on count and viewport', () => {
    expect(
      arrangeWindows({
        mode: 'smart',
        viewport: { width: 1200, height: 800 },
        windows: windows(2),
      }),
    ).toEqual(arrangeWindows({ mode: 'columns', viewport: { width: 1200, height: 800 }, windows: windows(2) }));

    expect(
      arrangeWindows({
        mode: 'smart',
        viewport: { width: 700, height: 1200 },
        windows: windows(2),
      }),
    ).toEqual(arrangeWindows({ mode: 'rows', viewport: { width: 700, height: 1200 }, windows: windows(2) }));

    expect(
      arrangeWindows({
        mode: 'smart',
        viewport: { width: 1200, height: 800 },
        activeId: 'view:2',
        windows: windows(3),
      }),
    ).toEqual(
      arrangeWindows({
        mode: 'focus-active',
        viewport: { width: 1200, height: 800 },
        activeId: 'view:2',
        windows: windows(3),
      }),
    );

    expect(
      arrangeWindows({
        mode: 'smart',
        viewport: { width: 1200, height: 800 },
        windows: windows(6),
      }),
    ).toEqual(arrangeWindows({ mode: 'grid', viewport: { width: 1200, height: 800 }, windows: windows(6) }));
  });

  it('honors minimum sizes when the viewport can satisfy them', () => {
    const geometries = arrangeWindows({
      mode: 'columns',
      viewport: { width: 1000, height: 700 },
      windows: [
        { id: 'view:1', minWidth: 500, minHeight: 240 },
        { id: 'view:2', minWidth: 300, minHeight: 240 },
      ],
    });

    expect(geometries).toEqual([
      { id: 'view:1', x: 0, y: 0, width: 600, height: 700 },
      { id: 'view:2', x: 600, y: 0, width: 400, height: 700 },
    ]);
  });

  it('clamps geometry inside a very small viewport when minimum sizes cannot fit', () => {
    const input = {
      mode: 'grid' as const,
      viewport: { width: 180, height: 120 },
      windows: windows(20, { minWidth: 360, minHeight: 240 }),
    };

    const geometries = arrangeWindows(input);

    expect(geometries).toHaveLength(20);
    expect(geometries.map((geometry) => geometry.id)).toEqual(input.windows.map((window) => window.id));
    expectWithinViewport(geometries, input.viewport);
    expectNoOverlap(geometries);
  });
});
