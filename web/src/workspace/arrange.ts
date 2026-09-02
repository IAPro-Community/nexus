export type ArrangeMode = 'smart' | 'grid' | 'columns' | 'rows' | 'focus-active';

export interface ArrangeWindowInput {
  id: string;
  minWidth?: number;
  minHeight?: number;
  active?: boolean;
}

export interface ArrangeWindowGeometry {
  id: string;
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface ArrangeWindowsInput {
  mode: ArrangeMode;
  windows: ArrangeWindowInput[];
  viewport: {
    width: number;
    height: number;
  };
  activeId?: string;
}

interface NormalizedWindow extends ArrangeWindowInput {
  minWidth: number;
  minHeight: number;
}

interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

const DEFAULT_MIN_WIDTH = 360;
const DEFAULT_MIN_HEIGHT = 240;
const FOCUS_RATIO = 0.6;

export function arrangeWindows(input: ArrangeWindowsInput): ArrangeWindowGeometry[] {
  const viewport = normalizeRect({
    x: 0,
    y: 0,
    width: input.viewport.width,
    height: input.viewport.height,
  });
  const windows = input.windows.map(normalizeWindow);

  if (windows.length === 0) return [];

  const mode = resolveMode(input.mode, windows, viewport, input.activeId);

  switch (mode) {
    case 'columns':
      return arrangeColumns(windows, viewport);
    case 'rows':
      return arrangeRows(windows, viewport);
    case 'focus-active':
      return arrangeFocusActive(windows, viewport, input.activeId);
    case 'grid':
    case 'smart':
    default:
      return arrangeGrid(windows, viewport);
  }
}

function normalizeWindow(window: ArrangeWindowInput): NormalizedWindow {
  return {
    ...window,
    minWidth: normalizeLength(window.minWidth, DEFAULT_MIN_WIDTH),
    minHeight: normalizeLength(window.minHeight, DEFAULT_MIN_HEIGHT),
  };
}

function normalizeRect(rect: Rect): Rect {
  return {
    x: clampStart(rect.x),
    y: clampStart(rect.y),
    width: normalizeNonNegative(rect.width),
    height: normalizeNonNegative(rect.height),
  };
}

function clampStart(value: number): number {
  return Number.isFinite(value) ? Math.max(0, Math.round(value)) : 0;
}

function normalizeNonNegative(value: number): number {
  return Number.isFinite(value) ? Math.max(0, Math.round(value)) : 0;
}

function normalizeLength(value: number | undefined, fallback: number): number {
  if (!Number.isFinite(value)) return fallback;
  return Math.max(1, Math.round(value!));
}

function resolveMode(
  requested: ArrangeMode,
  windows: NormalizedWindow[],
  viewport: Rect,
  activeId?: string,
): ArrangeMode {
  if (requested !== 'smart') return requested;
  if (windows.length <= 1) return 'grid';
  if (windows.length === 2) return viewport.width >= viewport.height ? 'columns' : 'rows';
  if (windows.length === 3 && resolveActiveIndex(windows, activeId) >= 0) return 'focus-active';
  return 'grid';
}

function arrangeColumns(windows: NormalizedWindow[], rect: Rect): ArrangeWindowGeometry[] {
  const widths = distributeLengths(rect.width, windows.map((window) => window.minWidth));
  const xs = offsets(widths, rect.x);

  return windows.map((window, index) => ({
    id: window.id,
    x: xs[index],
    y: rect.y,
    width: widths[index],
    height: rect.height,
  }));
}

function arrangeRows(windows: NormalizedWindow[], rect: Rect): ArrangeWindowGeometry[] {
  const heights = distributeLengths(rect.height, windows.map((window) => window.minHeight));
  const ys = offsets(heights, rect.y);

  return windows.map((window, index) => ({
    id: window.id,
    x: rect.x,
    y: ys[index],
    width: rect.width,
    height: heights[index],
  }));
}

function arrangeGrid(windows: NormalizedWindow[], rect: Rect): ArrangeWindowGeometry[] {
  const [columns, rows] = chooseGrid(windows.length, rect);
  return arrangeGridInRect(windows, rect, columns, rows);
}

function arrangeFocusActive(
  windows: NormalizedWindow[],
  rect: Rect,
  activeId?: string,
): ArrangeWindowGeometry[] {
  if (windows.length === 1) {
    return [{ id: windows[0].id, x: rect.x, y: rect.y, width: rect.width, height: rect.height }];
  }

  const activeIndex = resolveActiveIndex(windows, activeId);
  const activeWindow = windows[activeIndex];
  const secondaryWindows = windows.filter((_, index) => index !== activeIndex);

  if (rect.width >= rect.height) {
    const [primaryWidth, secondaryWidth] = splitPrimarySecondary(
      rect.width,
      activeWindow.minWidth,
      maxOrZero(secondaryWindows.map((window) => window.minWidth)),
    );
    const heights = distributeLengths(rect.height, secondaryWindows.map((window) => window.minHeight));
    const ys = offsets(heights, rect.y);
    const secondaryX = rect.x + primaryWidth;
    const geometry = new Map<string, ArrangeWindowGeometry>([
      [activeWindow.id, { id: activeWindow.id, x: rect.x, y: rect.y, width: primaryWidth, height: rect.height }],
    ]);

    secondaryWindows.forEach((window, index) => {
      geometry.set(window.id, {
        id: window.id,
        x: secondaryX,
        y: ys[index],
        width: secondaryWidth,
        height: heights[index],
      });
    });

    return windows.map((window) => geometry.get(window.id)!);
  }

  const [primaryHeight, secondaryHeight] = splitPrimarySecondary(
    rect.height,
    activeWindow.minHeight,
    maxOrZero(secondaryWindows.map((window) => window.minHeight)),
  );
  const widths = distributeLengths(rect.width, secondaryWindows.map((window) => window.minWidth));
  const xs = offsets(widths, rect.x);
  const secondaryY = rect.y + primaryHeight;
  const geometry = new Map<string, ArrangeWindowGeometry>([
    [activeWindow.id, { id: activeWindow.id, x: rect.x, y: rect.y, width: rect.width, height: primaryHeight }],
  ]);

  secondaryWindows.forEach((window, index) => {
    geometry.set(window.id, {
      id: window.id,
      x: xs[index],
      y: secondaryY,
      width: widths[index],
      height: secondaryHeight,
    });
  });

  return windows.map((window) => geometry.get(window.id)!);
}

function resolveActiveIndex(windows: NormalizedWindow[], activeId?: string): number {
  const explicit = activeId ? windows.findIndex((window) => window.id === activeId) : -1;
  if (explicit >= 0) return explicit;
  const flagged = windows.findIndex((window) => window.active);
  return flagged >= 0 ? flagged : 0;
}

function splitPrimarySecondary(total: number, primaryMin: number, secondaryMin: number): [number, number] {
  if (total <= 0) return [0, 0];
  if (secondaryMin <= 0) return [total, 0];
  if (primaryMin + secondaryMin <= total) {
    const desired = Math.round(total * FOCUS_RATIO);
    const primary = clamp(desired, primaryMin, total - secondaryMin);
    return [primary, total - primary];
  }
  const [primary, secondary] = distributeLengths(total, [primaryMin, secondaryMin], [FOCUS_RATIO, 1 - FOCUS_RATIO]);
  return [primary, secondary];
}

function chooseGrid(count: number, rect: Rect): [number, number] {
  const aspect = rect.height > 0 ? rect.width / rect.height : 1;
  let columns = Math.max(1, Math.ceil(Math.sqrt(count * Math.max(aspect, 0.25))));
  let rows = Math.max(1, Math.ceil(count / columns));

  while (columns > 1) {
    const nextColumns = columns - 1;
    const nextRows = Math.max(1, Math.ceil(count / nextColumns));
    if (nextRows !== rows || nextColumns * nextRows < count) break;
    columns = nextColumns;
    rows = nextRows;
  }

  return [columns, rows];
}

function arrangeGridInRect(
  windows: NormalizedWindow[],
  rect: Rect,
  columns: number,
  rows: number,
): ArrangeWindowGeometry[] {
  const columnMins = Array.from({ length: columns }, () => 1);
  const rowMins = Array.from({ length: rows }, () => 1);

  windows.forEach((window, index) => {
    const column = index % columns;
    const row = Math.floor(index / columns);
    columnMins[column] = Math.max(columnMins[column], window.minWidth);
    rowMins[row] = Math.max(rowMins[row], window.minHeight);
  });

  const widths = distributeLengths(rect.width, columnMins);
  const heights = distributeLengths(rect.height, rowMins);
  const xs = offsets(widths, rect.x);
  const ys = offsets(heights, rect.y);

  return windows.map((window, index) => {
    const column = index % columns;
    const row = Math.floor(index / columns);
    return {
      id: window.id,
      x: xs[column],
      y: ys[row],
      width: widths[column],
      height: heights[row],
    };
  });
}

function distributeLengths(total: number, minimums: number[], weights?: number[]): number[] {
  const normalizedTotal = normalizeNonNegative(total);
  if (minimums.length === 0) return [];
  if (normalizedTotal === 0) return minimums.map(() => 0);

  const mins = minimums.map((value) => normalizeLength(value, 1));
  const sum = mins.reduce((accumulator, value) => accumulator + value, 0);

  if (sum <= normalizedTotal) {
    const extra = normalizedTotal - sum;
    const appliedWeights = normalizeWeights(minimums.length, weights);
    const extras = distributeByWeights(extra, appliedWeights);
    return mins.map((value, index) => value + extras[index]);
  }

  const shrinkWeights = sum === 0 ? minimums.map(() => 1) : mins;
  return distributeByWeights(normalizedTotal, shrinkWeights);
}

function normalizeWeights(length: number, weights?: number[]): number[] {
  if (!weights || weights.length !== length) return Array.from({ length }, () => 1);
  const normalized = weights.map((value) => (Number.isFinite(value) && value > 0 ? value : 0));
  const sum = normalized.reduce((accumulator, value) => accumulator + value, 0);
  return sum > 0 ? normalized : Array.from({ length }, () => 1);
}

function distributeByWeights(total: number, weights: number[]): number[] {
  if (weights.length === 0) return [];
  const normalizedTotal = normalizeNonNegative(total);
  const normalizedWeights = normalizeWeights(weights.length, weights);
  const weightSum = normalizedWeights.reduce((accumulator, value) => accumulator + value, 0);
  const exact = normalizedWeights.map((weight) => (normalizedTotal * weight) / weightSum);
  const base = exact.map((value) => Math.floor(value));
  let remainder = normalizedTotal - base.reduce((accumulator, value) => accumulator + value, 0);

  if (remainder > 0) {
    const ranking = exact
      .map((value, index) => ({ index, fraction: value - base[index] }))
      .sort((left, right) => right.fraction - left.fraction || left.index - right.index);

    for (let rank = 0; rank < ranking.length && remainder > 0; rank += 1, remainder -= 1) {
      base[ranking[rank].index] += 1;
      if (rank === ranking.length - 1 && remainder > 0) rank = -1;
    }
  }

  return base;
}

function offsets(lengths: number[], start: number): number[] {
  let current = start;
  return lengths.map((length) => {
    const offset = current;
    current += length;
    return offset;
  });
}

function maxOrZero(values: number[]): number {
  return values.length === 0 ? 0 : Math.max(...values);
}

function clamp(value: number, minimum: number, maximum: number): number {
  return Math.min(maximum, Math.max(minimum, value));
}
