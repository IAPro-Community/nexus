import type { WorkspaceModel } from './model';

export type WorkspacePresentationMode = 'TABS' | 'DESKTOP';

export interface DesktopWindowState {
  viewId: string;
  x: number;
  y: number;
  width: number;
  height: number;
  zIndex: number;
  minimized: boolean;
  maximized: boolean;
}

export interface WorkspacePresentationState {
  mode: WorkspacePresentationMode;
  windows: Record<string, DesktopWindowState>;
}

export function createPresentationState(_workspace: WorkspaceModel): WorkspacePresentationState {
  return { mode: 'TABS', windows: {} };
}

export function setPresentationMode(state: WorkspacePresentationState, mode: WorkspacePresentationMode): WorkspacePresentationState {
  return state.mode === mode ? state : { ...state, mode };
}

export function ensureDesktopWindow(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  if (state.windows[viewId]) return state;
  const offset = Object.keys(state.windows).length * 28;
  return {
    ...state,
    windows: {
      ...state.windows,
      [viewId]: { viewId, x: 24 + offset, y: 24 + offset, width: 720, height: 480, zIndex: offset + 1, minimized: false, maximized: false },
    },
  };
}

export function moveDesktopWindow(state: WorkspacePresentationState, viewId: string, x: number, y: number): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  return { ...state, windows: { ...state.windows, [viewId]: { ...current, x, y } } };
}

export function resizeDesktopWindow(state: WorkspacePresentationState, viewId: string, width: number, height: number): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  return { ...state, windows: { ...state.windows, [viewId]: { ...current, width: Math.max(280, width), height: Math.max(180, height) } } };
}

export function minimizeDesktopWindow(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  return { ...state, windows: { ...state.windows, [viewId]: { ...current, minimized: !current.minimized } } };
}

export function focusDesktopWindow(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  const zIndex = Math.max(0, ...Object.values(state.windows).map((item) => item.zIndex)) + 1;
  return { ...state, windows: { ...state.windows, [viewId]: { ...current, zIndex, minimized: false } } };
}

export function toggleDesktopMaximize(state: WorkspacePresentationState, viewId: string): WorkspacePresentationState {
  const current = state.windows[viewId];
  if (!current) return state;
  return { ...state, windows: { ...state.windows, [viewId]: { ...current, maximized: !current.maximized } } };
}
