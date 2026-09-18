import { surfaceViewId, type WorkspaceSurface } from './model';
import {
  applySharedEdgeDelta,
  arrangeSmart,
  scaleTilesToCanvas,
  type ArrangeBounds,
} from './arrange';

export type WorkspacePresentationMode = 'TABS' | 'DESKTOP' | 'MOSAIC';

export interface DesktopWindowState {
  viewId: string;
  x: number;
  y: number;
  width: number;
  height: number;
  zIndex: number;
  minimized: boolean;
  maximized: boolean;
  /** User-chosen identity label; never overwritten by PTY OSC titles. */
  customTitle?: string;
  /** Accent color token (css color). */
  accent?: string;
  /** Fixed icon key from the mosaic chrome palette. */
  icon?: string;
}

export type MosaicResizeEdge = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw';

export interface SavedWindowGeometry {
  x: number;
  y: number;
  width: number;
  height: number;
  maximized?: boolean;
}

export interface ModeLayoutSnapshot {
  canvas: ArrangeBounds;
  windows: Record<string, SavedWindowGeometry>;
}

export type LastArrangePreset =
  'automatic' | 'two-columns' | 'three-columns' | 'terminal-focus' | 'focus-mode';

export interface WorkspacePresentationState {
  version: 2;
  mode: WorkspacePresentationMode;
  windows: Record<string, DesktopWindowState>;
  nextZ: number;
  canvas: ArrangeBounds;
  /**
   * True while MOSAIC keeps Smart-tiled geometry. Free-float DESKTOP keeps this false.
   */
  tiled: boolean;
  /** viewId of the PTY the user is viewing (inner tabs or desktop focus). */
  activePtyViewId: string;
  /** Last layout preset chosen from Arranjar / keyboard palette. */
  lastArrangePreset: LastArrangePreset;
  /** Last floating layout while in DESKTOP (restored when leaving mosaic/tabs). */
  desktopLayout: ModeLayoutSnapshot;
  /** Last tiled layout while in MOSAIC (restored when leaving windows/tabs). */
  mosaicLayout: ModeLayoutSnapshot;
  /** Fullscreen/Focus mode for maximizing screen estate on notebooks/small viewports */
  zenMode?: boolean;
}

const DEFAULT_CANVAS: ArrangeBounds = { x: 8, y: 8, width: 1200, height: 720 };
const DEFAULT_WINDOW_W = 640;
const DEFAULT_WINDOW_H = 420;
const CASCADE_OFFSET = 28;
const MIN_W = 280;
const MIN_H = 220;
const FLOATING_TYPES = new Set(['terminal', 'project-shell']);
/** Bottom restore shelf when mosaic has minimized tiles. Must match CSS. */
export const MOSAIC_MINIMIZED_SHELF_H = 42;

export function hasMinimizedWindows(state: WorkspacePresentationState): boolean {
  return Object.values(state.windows).some((win) => win.minimized);
}

/** Mosaic tiles leave a bottom strip for restore chips whenever anything is minimized. */
export function layoutCanvas(state: WorkspacePresentationState): ArrangeBounds {
  if (state.mode !== 'MOSAIC' || !hasMinimizedWindows(state)) return state.canvas;
  return {
    ...state.canvas,
    height: Math.max(MIN_H, state.canvas.height - MOSAIC_MINIMIZED_SHELF_H),
  };
}

export function isDesktopFloatingSurface(surface: WorkspaceSurface): boolean {
  return FLOATING_TYPES.has(surface.type);
}

export function isWindowedPresentationMode(mode: WorkspacePresentationMode): boolean {
  return mode === 'DESKTOP' || mode === 'MOSAIC';
}

/** Default presentation is floating windows (DESKTOP) inside the Terminals tab. */
function emptyLayout(canvas: ArrangeBounds = DEFAULT_CANVAS): ModeLayoutSnapshot {
  return { canvas: { ...canvas }, windows: {} };
}

export function createPresentationState(
  mode: WorkspacePresentationMode = 'DESKTOP',
): WorkspacePresentationState {
  const canvas = { ...DEFAULT_CANVAS };
  return {
    version: 2,
    mode,
    windows: {},
    nextZ: 1,
    canvas,
    tiled: mode === 'MOSAIC',
    activePtyViewId: '',
    lastArrangePreset: 'automatic',
    desktopLayout: emptyLayout(canvas),
    mosaicLayout: emptyLayout(canvas),
    zenMode: false,
  };
}

export function migratePresentationState(raw: unknown): WorkspacePresentationState {
  const fallback = createPresentationState();
  if (!raw || typeof raw !== 'object') return fallback;
  const parsed = raw as Partial<WorkspacePresentationState> & { version?: number };
  const mode =
    parsed.mode === 'TABS' || parsed.mode === 'DESKTOP' || parsed.mode === 'MOSAIC'
      ? parsed.mode
      : 'DESKTOP';
  const windows =
    parsed.windows && typeof parsed.windows === 'object'
      ? (parsed.windows as Record<string, DesktopWindowState>)
      : {};
  const nextZ = typeof parsed.nextZ === 'number' && parsed.nextZ > 0 ? parsed.nextZ : 1;
  const canvas =
    parsed.canvas &&
    typeof parsed.canvas.width === 'number' &&
    typeof parsed.canvas.height === 'number'
      ? {
          x: Number(parsed.canvas.x) || 8,
          y: Number(parsed.canvas.y) || 8,
          width: Math.max(320, Number(parsed.canvas.width) || DEFAULT_CANVAS.width),
          height: Math.max(240, Number(parsed.canvas.height) || DEFAULT_CANVAS.height),
        }
      : { ...DEFAULT_CANVAS };

  const activePtyViewId =
    typeof parsed.activePtyViewId === 'string' && parsed.activePtyViewId
      ? parsed.activePtyViewId
      : '';
  const tiled = mode === 'MOSAIC' ? parsed.tiled !== false : false;
  const lastRaw = (parsed as { lastArrangePreset?: string }).lastArrangePreset;
  const lastArrangePreset: LastArrangePreset =
    lastRaw === 'automatic' ||
    lastRaw === 'two-columns' ||
    lastRaw === 'three-columns' ||
    lastRaw === 'terminal-focus' ||
    lastRaw === 'focus-mode'
      ? lastRaw
      : 'automatic';
  const zenMode = Boolean((parsed as { zenMode?: boolean }).zenMode);
  return {
    version: 2,
    mode,
    windows,
    nextZ,
    canvas,
    tiled,
    activePtyViewId,
    lastArrangePreset,
    desktopLayout: parseLayoutSnapshot(parsed.desktopLayout, canvas),
    mosaicLayout: parseLayoutSnapshot(parsed.mosaicLayout, canvas),
    zenMode,
  };
}

function parseLayoutSnapshot(raw: unknown, fallbackCanvas: ArrangeBounds): ModeLayoutSnapshot {
  if (!raw || typeof raw !== 'object') return emptyLayout(fallbackCanvas);
  const parsed = raw as Partial<ModeLayoutSnapshot>;
  const canvas =
    parsed.canvas &&
    typeof parsed.canvas.width === 'number' &&
    typeof parsed.canvas.height === 'number'
      ? {
          x: Number(parsed.canvas.x) || fallbackCanvas.x,
          y: Number(parsed.canvas.y) || fallbackCanvas.y,
          width: Math.max(320, Number(parsed.canvas.width) || fallbackCanvas.width),
          height: Math.max(240, Number(parsed.canvas.height) || fallbackCanvas.height),
        }
      : { ...fallbackCanvas };
  const windows: Record<string, SavedWindowGeometry> = {};
  if (parsed.windows && typeof parsed.windows === 'object') {
    for (const [id, value] of Object.entries(parsed.windows)) {
      if (!value || typeof value !== 'object') continue;
      const geo = value as SavedWindowGeometry;
      windows[id] = {
        x: Number(geo.x) || 0,
        y: Number(geo.y) || 0,
        width: Math.max(MIN_W, Number(geo.width) || DEFAULT_WINDOW_W),
        height: Math.max(MIN_H, Number(geo.height) || DEFAULT_WINDOW_H),
        maximized: Boolean(geo.maximized),
      };
    }
  }
  return { canvas, windows };
}

function captureLayout(state: WorkspacePresentationState): ModeLayoutSnapshot {
  const windows: Record<string, SavedWindowGeometry> = {};
  for (const [id, win] of Object.entries(state.windows)) {
    windows[id] = {
      x: win.x,
      y: win.y,
      width: win.width,
      height: win.height,
      maximized: win.maximized,
    };
  }
  return { canvas: { ...state.canvas }, windows };
}

function snapshotLeavingMode(state: WorkspacePresentationState): WorkspacePresentationState {
  if (state.mode === 'MOSAIC') return { ...state, mosaicLayout: captureLayout(state) };
  if (state.mode === 'DESKTOP') return { ...state, desktopLayout: captureLayout(state) };
  return state;
}

function applyMosaicLayout(state: WorkspacePresentationState): WorkspacePresentationState {
  const snapshot = state.mosaicLayout;
  const canvas = layoutCanvas(state);
  const visible = Object.values(state.windows).filter((win) => !win.minimized);
  if (visible.length === 0) return { ...state, tiled: true };
  if (visible.some((win) => !snapshot.windows[win.viewId])) {
    return tileMosaicWindows(state);
  }
  const tiles = visible.map((win) => ({ viewId: win.viewId, ...snapshot.windows[win.viewId] }));
  const scaled = scaleTilesToCanvas(tiles, snapshot.canvas, canvas);
  const windows = { ...state.windows };
  scaled.forEach((tile) => {
    const current = windows[tile.viewId];
    if (!current) return;
    windows[tile.viewId] = preserveChrome(current, {
      ...current,
      ...clampWindow(canvas, tile),
      maximized: false,
    });
  });
  return packMosaic({ ...state, windows, tiled: true });
}

function applyDesktopLayout(state: WorkspacePresentationState): WorkspacePresentationState {
  const snapshot = state.desktopLayout;
  const visible = Object.values(state.windows).filter((win) => !win.minimized);
  const savedTiles = visible
    .filter((win) => snapshot.windows[win.viewId])
    .map((win) => ({ viewId: win.viewId, ...snapshot.windows[win.viewId] }));
  if (savedTiles.length === 0) {
    return rearrangeSmart({ ...state, tiled: false });
  }
  const scaled = scaleTilesToCanvas(savedTiles, snapshot.canvas, state.canvas);
  const windows = { ...state.windows };
  scaled.forEach((tile) => {
    const current = windows[tile.viewId];
    const saved = snapshot.windows[tile.viewId];
    if (!current) return;
    windows[tile.viewId] = preserveChrome(current, {
      ...current,
      ...clampWindow(state.canvas, tile),
      maximized: Boolean(saved?.maximized),
    });
  });
  let cascadeIndex = 0;
  visible.forEach((win) => {
    if (snapshot.windows[win.viewId]) return;
    windows[win.viewId] = preserveChrome(
      win,
      createCascadedWindow(state, win.viewId, cascadeIndex++, win.zIndex),
    );
  });
  return { ...state, windows, tiled: false };
}

function clampWindow(
  canvas: ArrangeBounds,
  win: Pick<DesktopWindowState, 'x' | 'y' | 'width' | 'height'>,
): Pick<DesktopWindowState, 'x' | 'y' | 'width' | 'height'> {
  const width = Math.min(Math.max(MIN_W, Math.round(win.width)), Math.max(MIN_W, canvas.width));
  const height = Math.min(Math.max(MIN_H, Math.round(win.height)), Math.max(MIN_H, canvas.height));
  const maxX = canvas.x + Math.max(0, canvas.width - width);
  const maxY = canvas.y + Math.max(0, canvas.height - height);
  return {
    x: Math.min(Math.max(canvas.x, Math.round(win.x)), maxX),
    y: Math.min(Math.max(canvas.y, Math.round(win.y)), maxY),
    width,
    height,
  };
}

function cascadeOrigin(canvas: ArrangeBounds, index: number): { x: number; y: number } {
  const maxStepsX = Math.max(0, Math.floor((canvas.width - DEFAULT_WINDOW_W) / CASCADE_OFFSET));
  const maxStepsY = Math.max(0, Math.floor((canvas.height - DEFAULT_WINDOW_H) / CASCADE_OFFSET));
  const step =
    maxStepsX === 0 && maxStepsY === 0
      ? 0
      : index % (Math.max(1, Math.min(maxStepsX, maxStepsY)) + 1);
  return {
    x: canvas.x + step * CASCADE_OFFSET,
    y: canvas.y + step * CASCADE_OFFSET,
  };
}

function createCascadedWindow(
  state: WorkspacePresentationState,
  viewId: string,
  cascadeIndex: number,
  zIndex: number,
): DesktopWindowState {
  const origin = cascadeOrigin(state.canvas, cascadeIndex);
  const geometry = clampWindow(state.canvas, {
    x: origin.x,
    y: origin.y,
    width: DEFAULT_WINDOW_W,
    height: DEFAULT_WINDOW_H,
  });
  return {
    viewId,
    ...geometry,
    zIndex,
    minimized: false,
    maximized: false,
  };
}

function preserveChrome(
  previous: DesktopWindowState | undefined,
  next: DesktopWindowState,
): DesktopWindowState {
  if (!previous) return next;
  return {
    ...next,
    customTitle: previous.customTitle,
    accent: previous.accent,
    icon: previous.icon,
  };
}

function visibleViewIds(state: WorkspacePresentationState, preferred?: string[]): string[] {
  const ids =
    preferred ??
    Object.values(state.windows)
      .filter((win) => !win.minimized)
      .sort((a, b) => a.zIndex - b.zIndex)
      .map((win) => win.viewId);
  return ids.filter((id) => state.windows[id] && !state.windows[id].minimized);
}

/** Pack visible windows with Smart mosaic geometry. */
export function tileMosaicWindows(
  state: WorkspacePresentationState,
  viewIds?: string[],
): WorkspacePresentationState {
  const visible = visibleViewIds(state, viewIds);
  const canvas = layoutCanvas(state);
  const tiles = arrangeSmart(canvas, visible);
  const windows = { ...state.windows };
  let nextZ = Math.max(1, state.nextZ);
  tiles.forEach((tile) => {
    const previous = windows[tile.viewId];
    const zIndex = previous?.zIndex ?? nextZ++;
    const geometry = clampWindow(canvas, tile);
    windows[tile.viewId] = preserveChrome(previous, {
      viewId: tile.viewId,
      ...geometry,
      zIndex,
      minimized: false,
      maximized: false,
    });
    nextZ = Math.max(nextZ, zIndex + 1);
  });
  return { ...state, windows, nextZ, tiled: true };
}

/** Keep mosaic tiles covering the canvas without discarding relative sizes. */
export function packMosaic(state: WorkspacePresentationState): WorkspacePresentationState {
  if (state.mode !== 'MOSAIC') return state;
  const canvas = layoutCanvas(state);
  const visible = Object.values(state.windows).filter((win) => !win.minimized);
  if (visible.length === 0) return { ...state, tiled: true };
  if (visible.length === 1) {
    const only = visible[0];
    return {
      ...state,
      windows: {
        ...state.windows,
        [only.viewId]: preserveChrome(only, {
          ...only,
          x: canvas.x,
          y: canvas.y,
          width: canvas.width,
          height: canvas.height,
          maximized: false,
        }),
      },
      tiled: true,
    };
  }
  const scaled = scaleTilesToCanvas(visible, state.canvas, canvas);
  const area = visible.reduce((sum, win) => sum + win.width * win.height, 0);
  const canvasArea = canvas.width * canvas.height;
  if (area < canvasArea * 0.85) {
    return tileMosaicWindows(state);
  }
  const windows = { ...state.windows };
  scaled.forEach((tile) => {
    const current = windows[tile.viewId];
    if (!current) return;
    windows[tile.viewId] = preserveChrome(current, {
      ...current,
      ...clampWindow(canvas, tile),
      maximized: false,
    });
  });
  const packed = Object.values(windows).filter((win) => !win.minimized);
  if (mosaicTilesOverlap(packed)) {
    return tileMosaicWindows(state);
  }
  return { ...state, windows, tiled: true };
}

function mosaicTilesOverlap(windows: DesktopWindowState[]): boolean {
  for (let i = 0; i < windows.length; i += 1) {
    for (let j = i + 1; j < windows.length; j += 1) {
      const a = windows[i];
      const b = windows[j];
      if (
        a.x < b.x + b.width - 8 &&
        b.x < a.x + a.width - 8 &&
        a.y < b.y + b.height - 8 &&
        b.y < a.y + a.height - 8
      ) {
        return true;
      }
    }
  }
  return false;
}

/** Cascade (DESKTOP) or Smart tile (MOSAIC). */
export function rearrangeSmart(
  state: WorkspacePresentationState,
  viewIds?: string[],
): WorkspacePresentationState {
  if (state.mode === 'MOSAIC') {
    return tileMosaicWindows(state, viewIds);
  }
  const visible = visibleViewIds(state, viewIds);
  const windows = { ...state.windows };
  let nextZ = Math.max(1, state.nextZ);
  visible.forEach((viewId, index) => {
    const previous = windows[viewId];
    const zIndex = previous?.zIndex ?? nextZ++;
    windows[viewId] = preserveChrome(previous, createCascadedWindow(state, viewId, index, zIndex));
    nextZ = Math.max(nextZ, zIndex + 1);
  });
  return { ...state, windows, nextZ, tiled: false };
}

export function setPresentationMode(
  state: WorkspacePresentationState,
  mode: WorkspacePresentationMode,
): WorkspacePresentationState {
  if (state.mode === mode) return state;
  const snapped = snapshotLeavingMode(state);
  const next = { ...snapped, mode, tiled: mode === 'MOSAIC' };
  if (mode === 'MOSAIC') return applyMosaicLayout(next);
  if (mode === 'DESKTOP') return applyDesktopLayout(next);
  return { ...next, tiled: false };
}

export function toggleZenMode(state: WorkspacePresentationState): WorkspacePresentationState {
  return { ...state, zenMode: !state.zenMode };
}

export function setZenMode(
  state: WorkspacePresentationState,
  enabled: boolean,
): WorkspacePresentationState {
  return { ...state, zenMode: enabled };
}

export function setPresentationCanvas(
  state: WorkspacePresentationState,
  canvas: ArrangeBounds,
): WorkspacePresentationState {
  const nextCanvas: ArrangeBounds = {
    x: Math.max(0, Math.round(canvas.x)),
    y: Math.max(0, Math.round(canvas.y)),
    width: Math.max(320, Math.round(canvas.width)),
    height: Math.max(240, Math.round(canvas.height)),
  };
  if (
    state.canvas.width === nextCanvas.width &&
    state.canvas.height === nextCanvas.height &&
    state.canvas.x === nextCanvas.x &&
    state.canvas.y === nextCanvas.y
  ) {
    return state;
  }

  const withCanvas = { ...state, canvas: nextCanvas };
  if (state.mode === 'MOSAIC') {
    const visible = Object.values(state.windows).filter((win) => !win.minimized);
    if (visible.length === 0) return { ...withCanvas, tiled: true };
    const scaled = scaleTilesToCanvas(visible, state.canvas, nextCanvas);
    const windows = { ...state.windows };
    scaled.forEach((tile) => {
      const current = windows[tile.viewId];
      if (!current) return;
      windows[tile.viewId] = preserveChrome(current, {
        ...current,
        ...clampWindow(nextCanvas, tile),
        maximized: false,
      });
    });
    return packMosaic({ ...withCanvas, windows, tiled: true });
  }

  // Free DESKTOP mode keeps exact coordinates, only clamping to canvas boundaries
  const windows = { ...state.windows };
  Object.keys(windows).forEach((id) => {
    const current = windows[id];
    if (!current || current.minimized || current.maximized) return;
    const clamped = clampWindow(nextCanvas, current);
    windows[id] = { ...current, ...clamped };
  });
  return { ...withCanvas, windows, tiled: false };
}

export function syncDesktopWindows(
  state: WorkspacePresentationState,
  surfaces: WorkspaceSurface[],
): WorkspacePresentationState {
  const floating = surfaces.filter(isDesktopFloatingSurface);
  const nextIds = floating.map(surfaceViewId);
  const preserved: Record<string, DesktopWindowState> = {};
  let nextZ = Math.max(1, state.nextZ);

  nextIds.forEach((viewId) => {
    const current = state.windows[viewId];
    if (current) {
      preserved[viewId] = current;
      nextZ = Math.max(nextZ, current.zIndex + 1);
    }
  });

  // If surfaces is transiently empty but state already has persisted windows, preserve them
  if (nextIds.length === 0 && Object.keys(state.windows).length > 0) {
    return state;
  }

  const added = nextIds.filter((id) => !state.windows[id]);
  const removed = Object.keys(state.windows).filter((id) => !nextIds.includes(id));
  if (added.length === 0 && removed.length === 0) {
    if (Object.keys(state.windows).every((id) => nextIds.includes(id))) {
      if (state.activePtyViewId && nextIds.includes(state.activePtyViewId)) return state;
      return { ...state, activePtyViewId: nextIds[0] || '' };
    }
  }

  const windows: Record<string, DesktopWindowState> = {};
  const existingCount = Object.keys(preserved).length;
  nextIds.forEach((viewId, index) => {
    const previous = preserved[viewId];
    if (previous) {
      windows[viewId] = previous;
      return;
    }
    const cascadeIndex = existingCount + added.indexOf(viewId);
    const zIndex = nextZ++;
    windows[viewId] = createCascadedWindow(
      state,
      viewId,
      cascadeIndex >= 0 ? cascadeIndex : index,
      zIndex,
    );
  });

  const activePtyViewId = added.length
    ? added[added.length - 1]
    : nextIds.includes(state.activePtyViewId)
      ? state.activePtyViewId
      : nextIds[0] || '';

  const synced: WorkspacePresentationState = {
    ...state,
    windows,
    nextZ,
    tiled: state.mode === 'MOSAIC',
    activePtyViewId,
  };
  if (state.mode === 'MOSAIC' && (added.length > 0 || removed.length > 0)) {
    return tileMosaicWindows(synced);
  }
  return { ...synced, tiled: false };
}

function patchWindow(
  state: WorkspacePresentationState,
  viewId: string,
  patch: Partial<DesktopWindowState>,
): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  return { ...state, windows: { ...state.windows, [viewId]: { ...current, ...patch } } };
}

export function setActivePtyView(
  state: WorkspacePresentationState,
  viewId: string,
): WorkspacePresentationState {
  const next = (viewId || '').trim();
  if (state.activePtyViewId === next) return state;
  return { ...state, activePtyViewId: next };
}

export function focusDesktopWindow(
  state: WorkspacePresentationState,
  viewId: string,
): WorkspacePresentationState {
  if (!state.windows[viewId]) {
    return setActivePtyView(state, viewId);
  }
  const current = state.windows[viewId];
  if (state.mode === 'MOSAIC') {
    if (current.minimized) return state;
    if (state.activePtyViewId === viewId) return state;
    return { ...state, activePtyViewId: viewId };
  }
  const maxZ = Object.values(state.windows).reduce((max, win) => Math.max(max, win.zIndex), 0);
  if (state.activePtyViewId === viewId && !current.minimized && current.zIndex >= maxZ) {
    return state;
  }
  const zIndex = state.nextZ;
  return {
    ...patchWindow(state, viewId, { zIndex, minimized: false }),
    nextZ: zIndex + 1,
    activePtyViewId: viewId,
  };
}

export function moveDesktopWindow(
  state: WorkspacePresentationState,
  viewId: string,
  x: number,
  y: number,
): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current || current.maximized || current.minimized) return state;
  if (state.mode === 'MOSAIC') {
    // Tiles stay in their slot. Swap happens in commitMosaicMove from the pointer.
    return state;
  }
  const geometry = clampWindow(state.canvas, {
    x,
    y,
    width: current.width,
    height: current.height,
  });
  return {
    ...patchWindow(state, viewId, { x: geometry.x, y: geometry.y }),
    tiled: false,
  };
}

/** Tile under a workspace point — same hit test as commitMosaicMove. */
export function mosaicDropTargetViewId(
  state: WorkspacePresentationState,
  viewId: string,
  point?: { x: number; y: number },
): string {
  if (state.mode !== 'MOSAIC') return '';
  const current = state.windows[viewId];
  if (!current || current.minimized) return '';
  const cx = point?.x ?? current.x + current.width / 2;
  const cy = point?.y ?? current.y + current.height / 2;
  const target = Object.values(state.windows).find(
    (win) =>
      win.viewId !== viewId &&
      !win.minimized &&
      cx >= win.x &&
      cx <= win.x + win.width &&
      cy >= win.y &&
      cy <= win.y + win.height,
  );
  return target?.viewId || '';
}

/** Finish a mosaic drag: swap with the tile under the pointer, else keep slots. */
export function commitMosaicMove(
  state: WorkspacePresentationState,
  viewId: string,
  origin: Pick<DesktopWindowState, 'x' | 'y' | 'width' | 'height'>,
  pointer?: { x: number; y: number },
): WorkspacePresentationState {
  if (state.mode !== 'MOSAIC') return state;
  const current = state.windows[viewId];
  if (!current || current.minimized) return tileMosaicWindows(state);
  const parked = preserveChrome(current, {
    ...current,
    x: origin.x,
    y: origin.y,
    width: origin.width,
    height: origin.height,
    maximized: false,
  });
  const parkedState = packMosaic({
    ...state,
    windows: { ...state.windows, [viewId]: parked },
    tiled: true,
    activePtyViewId: viewId,
  });
  if (!pointer) return parkedState;
  const targetId = mosaicDropTargetViewId(parkedState, viewId, pointer);
  const target = targetId ? parkedState.windows[targetId] : undefined;
  if (!target) return parkedState;
  const windows = {
    ...parkedState.windows,
    [viewId]: preserveChrome(parked, {
      ...parked,
      x: target.x,
      y: target.y,
      width: target.width,
      height: target.height,
      maximized: false,
    }),
    [target.viewId]: preserveChrome(target, {
      ...target,
      x: origin.x,
      y: origin.y,
      width: origin.width,
      height: origin.height,
      maximized: false,
    }),
  };
  return packMosaic({ ...parkedState, windows, tiled: true, activePtyViewId: viewId });
}

export function resizeDesktopWindow(
  state: WorkspacePresentationState,
  viewId: string,
  width: number,
  height: number,
): WorkspacePresentationState {
  if (state.mode === 'MOSAIC') return state;
  const current = state.windows[viewId];
  if (!current || current.maximized || current.minimized) return state;
  const geometry = clampWindow(state.canvas, {
    x: current.x,
    y: current.y,
    width,
    height,
  });
  return {
    ...patchWindow(state, viewId, {
      x: geometry.x,
      y: geometry.y,
      width: geometry.width,
      height: geometry.height,
    }),
    tiled: false,
  };
}

/** Grow/shrink a mosaic tile; adjacent tiles on that edge absorb the delta. */
export function resizeMosaicWindow(
  state: WorkspacePresentationState,
  _viewId: string,
  _edge: MosaicResizeEdge,
  _deltaX: number,
  _deltaY: number,
): WorkspacePresentationState {
  // Mosaic size changes only through shared splitters so tiles never cover each other.
  return state;
}

export function patchDesktopWindowChrome(
  state: WorkspacePresentationState,
  viewId: string,
  chrome: Pick<DesktopWindowState, 'customTitle' | 'accent' | 'icon'>,
): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) {
    const stub = createCascadedWindow(
      state,
      viewId,
      Object.keys(state.windows).length,
      state.nextZ,
    );
    return {
      ...state,
      windows: { ...state.windows, [viewId]: { ...stub, ...chrome } },
      nextZ: Math.max(state.nextZ, stub.zIndex + 1),
    };
  }
  return patchWindow(state, viewId, chrome);
}

/** Shared-edge resize for MOSAIC splitters. No-op in DESKTOP/TABS. */
export function resizeAdjacentDesktopWindows(
  state: WorkspacePresentationState,
  firstId: string,
  secondId: string,
  orientation: 'vertical' | 'horizontal',
  delta: number,
): WorkspacePresentationState {
  if (state.mode !== 'MOSAIC') return state;
  const first = state.windows[firstId];
  const second = state.windows[secondId];
  if (!first || !second || first.minimized || second.minimized) return state;
  const patched = applySharedEdgeDelta(first, second, orientation, delta);
  if (!patched) return state;
  const windows = {
    ...state.windows,
    [firstId]: {
      ...first,
      x: patched.first.x,
      y: patched.first.y,
      width: patched.first.width,
      height: patched.first.height,
      maximized: false,
    },
    [secondId]: {
      ...second,
      x: patched.second.x,
      y: patched.second.y,
      width: patched.second.width,
      height: patched.second.height,
      maximized: false,
    },
  };
  return { ...state, windows, tiled: true };
}

export function toggleDesktopMinimize(
  state: WorkspacePresentationState,
  viewId: string,
): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  const minimized = !current.minimized;
  const next: WorkspacePresentationState = {
    ...patchWindow(state, viewId, { minimized, maximized: false }),
    tiled: state.mode === 'MOSAIC',
    activePtyViewId: minimized
      ? state.activePtyViewId === viewId
        ? Object.values(state.windows)
            .filter((win) => win.viewId !== viewId && !win.minimized)
            .sort((a, b) => b.zIndex - a.zIndex)[0]?.viewId || ''
        : state.activePtyViewId
      : viewId,
  };
  if (state.mode === 'MOSAIC') {
    return tileMosaicWindows(next);
  }
  return { ...next, tiled: false };
}

export function toggleDesktopMaximize(
  state: WorkspacePresentationState,
  viewId: string,
): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  // Mosaic uses splitters, not maximize chrome.
  if (state.mode === 'MOSAIC') {
    return focusDesktopWindow(state, viewId);
  }
  return {
    ...patchWindow(state, viewId, { maximized: !current.maximized, minimized: false }),
    tiled: false,
    activePtyViewId: viewId,
  };
}
