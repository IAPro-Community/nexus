import React, { createContext, useContext, useEffect, useMemo, useReducer } from 'react';
import {
  closeSurface,
  createWorkspace,
  moveSurface,
  openSurface,
  setActiveSurface,
  setSplitRatio,
  splitEmpty as splitEmptyModel,
  splitWithSurface,
  toggleMaximize,
  updateSurface,
  type WorkspaceDirection,
  type WorkspaceModel,
  type WorkspaceSurface,
} from './model';
import { deserializeWorkspace, serializeWorkspace, workspaceStorageKey } from './state';
import { createPresentationState, setPresentationMode, type WorkspacePresentationMode, type WorkspacePresentationState } from './presentation';

interface WorkspaceContextValue {
  model: WorkspaceModel;
  hydrated: boolean;
  open: (surface: WorkspaceSurface, targetStackId?: string) => void;
  activate: (surfaceId: string) => void;
  close: (surfaceId: string) => void;
  updateSurface: (surfaceId: string, patch: Partial<WorkspaceSurface>) => void;
  split: (relativeSurfaceId: string, surface: WorkspaceSurface, direction: WorkspaceDirection) => void;
  move: (surfaceId: string, targetStackId: string) => void;
  resize: (splitId: string, ratio: number) => void;
  maximize: (surfaceId: string) => void;
  splitEmpty: (relativeSurfaceId: string, direction: WorkspaceDirection) => void;
  reset: () => void;
  presentation: WorkspacePresentationState;
  setPresentationMode: (mode: WorkspacePresentationMode) => void;
  updatePresentation: (update: (state: WorkspacePresentationState) => WorkspacePresentationState) => void;
}

type Action =
  | { type: 'replace'; model: WorkspaceModel }
  | { type: 'open'; surface: WorkspaceSurface; stackId?: string }
  | { type: 'activate'; surfaceId: string }
  | { type: 'close'; surfaceId: string }
  | { type: 'updateSurface'; surfaceId: string; patch: Partial<WorkspaceSurface> }
  | { type: 'split'; relativeSurfaceId: string; surface: WorkspaceSurface; direction: WorkspaceDirection }
  | { type: 'splitEmpty'; relativeSurfaceId: string; direction: WorkspaceDirection }
  | { type: 'move'; surfaceId: string; targetStackId: string }
  | { type: 'resize'; splitId: string; ratio: number }
  | { type: 'maximize'; surfaceId: string };

function reducer(model: WorkspaceModel, action: Action): WorkspaceModel {
  switch (action.type) {
    case 'replace': return action.model;
    case 'open': return openSurface(model, action.surface, action.stackId);
    case 'activate': return setActiveSurface(model, action.surfaceId);
    case 'close': return closeSurface(model, action.surfaceId);
    case 'updateSurface': return updateSurface(model, action.surfaceId, action.patch);
    case 'split': return splitWithSurface(model, action.relativeSurfaceId, action.surface, action.direction);
    case 'splitEmpty': return splitEmptyModel(model, action.relativeSurfaceId, action.direction);
    case 'move': return moveSurface(model, action.surfaceId, action.targetStackId);
    case 'resize': return setSplitRatio(model, action.splitId, action.ratio);
    case 'maximize': return toggleMaximize(model, action.surfaceId);
  }
}

const WorkspaceContext = createContext<WorkspaceContextValue | null>(null);

const defaultSurface = (projectId: string): WorkspaceSurface => ({
  id: `project:${projectId}:overview`,
  viewId: `view:project:${projectId}:overview`,
  logicalKey: `project:${projectId}:overview`,
  type: 'overview',
  title: 'Overview',
  titleKey: 'nav.overview',
  closable: false,
  data: { projectId },
});

export const WorkspaceProvider: React.FC<{
  projectId: string;
  initialLayout?: string;
  saveLayout?: (layout: string) => Promise<unknown>;
  children: React.ReactNode;
}> = ({ projectId, initialLayout, saveLayout, children }) => {
  const fallback = useMemo(() => createWorkspace(defaultSurface(projectId)), [projectId]);
  const [model, dispatch] = useReducer(reducer, fallback);
  const [hydrated, setHydrated] = React.useState(false);
  const [presentation, setPresentation] = React.useState<WorkspacePresentationState>(() => createPresentationState(fallback));

  useEffect(() => {
    const local = window.localStorage.getItem(workspaceStorageKey(projectId));
    dispatch({ type: 'replace', model: deserializeWorkspace(initialLayout || local, fallback) });
    const rawPresentation = window.localStorage.getItem(`${workspaceStorageKey(projectId)}:presentation`);
    if (rawPresentation) {
      try {
        const parsed = JSON.parse(rawPresentation) as WorkspacePresentationState;
        if (parsed.mode === 'TABS' || parsed.mode === 'DESKTOP') setPresentation(parsed);
      } catch { /* use default presentation */ }
    } else {
      setPresentation(createPresentationState(fallback));
    }
    setHydrated(true);
  }, [projectId]);

  useEffect(() => {
    if (!hydrated) return;
    const serialized = serializeWorkspace(model);
    window.localStorage.setItem(workspaceStorageKey(projectId), serialized);
    if (!saveLayout) return;
    const timer = window.setTimeout(() => { void saveLayout(serialized).catch(() => undefined); }, 500);
    return () => window.clearTimeout(timer);
  }, [model, projectId, saveLayout, hydrated]);

  useEffect(() => {
    if (hydrated) window.localStorage.setItem(`${workspaceStorageKey(projectId)}:presentation`, JSON.stringify(presentation));
  }, [hydrated, presentation, projectId]);

  const value = useMemo<WorkspaceContextValue>(() => ({
    model,
    hydrated,
    open: (surface, stackId) => dispatch({ type: 'open', surface, stackId }),
    activate: (surfaceId) => dispatch({ type: 'activate', surfaceId }),
    close: (surfaceId) => dispatch({ type: 'close', surfaceId }),
    updateSurface: (surfaceId, patch) => dispatch({ type: 'updateSurface', surfaceId, patch }),
    split: (relativeSurfaceId, surface, direction) => dispatch({ type: 'split', relativeSurfaceId, surface, direction }),
    splitEmpty: (relativeSurfaceId, direction) => dispatch({ type: 'splitEmpty', relativeSurfaceId, direction }),
    move: (surfaceId, targetStackId) => dispatch({ type: 'move', surfaceId, targetStackId }),
    resize: (splitId, ratio) => dispatch({ type: 'resize', splitId, ratio }),
    maximize: (surfaceId) => dispatch({ type: 'maximize', surfaceId }),
    reset: () => dispatch({ type: 'replace', model: fallback }),
    presentation,
    setPresentationMode: (mode) => setPresentation((current) => setPresentationMode(current, mode)),
    updatePresentation: (update) => setPresentation(update),
  }), [model, fallback, hydrated, presentation]);

  return <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>;
};

export function useWorkspace(): WorkspaceContextValue {
  const context = useContext(WorkspaceContext);
  if (!context) throw new Error('useWorkspace must be used inside WorkspaceProvider');
  return context;
}
