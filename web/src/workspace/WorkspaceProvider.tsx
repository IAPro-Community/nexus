import React, { createContext, useContext, useEffect, useMemo, useReducer, useRef } from 'react';
import {
  closeSurface,
  createWorkspace,
  ensureSurface,
  moveSurface,
  openSurface,
  setActiveSurface,
  setSplitRatio,
  splitWithSurface,
  toggleMaximize,
  updateSurface,
  listSurfaces,
  type WorkspaceDirection,
  type WorkspaceModel,
  type WorkspaceSurface,
} from './model';
import { WorkspaceLayoutService } from '../services/WorkspaceLayoutService';
import type { WorkspacePresentationState } from './presentation';

interface WorkspaceContextValue {
  model: WorkspaceModel;
  open: (surface: WorkspaceSurface, targetStackId?: string) => void;
  ensure: (surface: WorkspaceSurface, targetStackId?: string) => void;
  activate: (surfaceId: string) => void;
  close: (surfaceId: string) => void;
  updateSurface: (surfaceId: string, patch: Partial<WorkspaceSurface>) => void;
  split: (
    relativeSurfaceId: string,
    surface: WorkspaceSurface,
    direction: WorkspaceDirection,
  ) => void;
  move: (surfaceId: string, targetStackId: string) => void;
  resize: (splitId: string, ratio: number) => void;
  maximize: (surfaceId: string) => void;
  reset: () => void;
}

export interface WorkspaceLayoutPersistence {
  initialPresentation: WorkspacePresentationState;
  registerPresentation: (state: WorkspacePresentationState) => void;
}

const WorkspaceLayoutPersistenceContext = createContext<WorkspaceLayoutPersistence | null>(null);

export function useWorkspaceLayoutPersistence(): WorkspaceLayoutPersistence {
  const context = useContext(WorkspaceLayoutPersistenceContext);
  if (!context)
    throw new Error('useWorkspaceLayoutPersistence must be used inside WorkspaceProvider');
  return context;
}

type Action =
  | { type: 'replace'; model: WorkspaceModel }
  | { type: 'open'; surface: WorkspaceSurface; stackId?: string }
  | { type: 'ensure'; surface: WorkspaceSurface; stackId?: string }
  | { type: 'activate'; surfaceId: string }
  | { type: 'close'; surfaceId: string }
  | { type: 'updateSurface'; surfaceId: string; patch: Partial<WorkspaceSurface> }
  | {
      type: 'split';
      relativeSurfaceId: string;
      surface: WorkspaceSurface;
      direction: WorkspaceDirection;
    }
  | { type: 'move'; surfaceId: string; targetStackId: string }
  | { type: 'resize'; splitId: string; ratio: number }
  | { type: 'maximize'; surfaceId: string };

function reducer(model: WorkspaceModel, action: Action): WorkspaceModel {
  switch (action.type) {
    case 'replace':
      return action.model;
    case 'open':
      return openSurface(model, action.surface, action.stackId);
    case 'ensure':
      return ensureSurface(model, action.surface, action.stackId);
    case 'activate':
      return setActiveSurface(model, action.surfaceId);
    case 'close':
      return closeSurface(model, action.surfaceId);
    case 'updateSurface':
      return updateSurface(model, action.surfaceId, action.patch);
    case 'split':
      return splitWithSurface(model, action.relativeSurfaceId, action.surface, action.direction);
    case 'move':
      return moveSurface(model, action.surfaceId, action.targetStackId);
    case 'resize':
      return setSplitRatio(model, action.splitId, action.ratio);
    case 'maximize':
      return toggleMaximize(model, action.surfaceId);
  }
}

const WorkspaceContext = createContext<WorkspaceContextValue | null>(null);

const defaultSurface = (projectId: string): WorkspaceSurface => {
  const logicalKey = `project:${projectId}:overview`;
  return {
    id: logicalKey,
    viewId: `view:${logicalKey}`,
    logicalKey,
    type: 'overview',
    title: 'Overview',
    titleKey: 'nav.overview',
    closable: false,
    data: { projectId },
  };
};

export const WorkspaceProvider: React.FC<{
  projectId: string;
  initialLayout?: string;
  initialRevision?: number;
  saveLayout?: (layout: string, revision?: number) => Promise<unknown>;
  onSurfaceClosed?: (surface: WorkspaceSurface) => void | Promise<void>;
  children: React.ReactNode;
}> = ({ projectId, initialLayout, initialRevision, saveLayout, onSurfaceClosed, children }) => {
  const layoutService = useMemo(() => new WorkspaceLayoutService(projectId), [projectId]);
  const fallback = useMemo(() => createWorkspace(defaultSurface(projectId)), [projectId]);
  const initialPersisted = useMemo(() => {
    if (!initialLayout) return layoutService.load(fallback);
    try {
      return layoutService.migrateAndNormalize(JSON.parse(initialLayout), fallback);
    } catch {
      return layoutService.load(fallback);
    }
  }, [fallback, initialLayout, layoutService]);
  const revisionRef = useRef<number>(initialRevision || initialPersisted.revision || 1);
  const [presentation, setPresentation] = React.useState(initialPersisted.presentation);
  const hydratedLayoutRef = useRef<string | undefined>(undefined);

  useEffect(() => {
    if (typeof initialRevision === 'number' && initialRevision > 0) {
      revisionRef.current = initialRevision;
    }
  }, [initialRevision]);

  const [model, dispatch] = useReducer(reducer, initialPersisted.model);

  useEffect(() => {
    if (!initialLayout || hydratedLayoutRef.current === initialLayout) return;
    hydratedLayoutRef.current = initialLayout;
    dispatch({
      type: 'replace',
      model: initialPersisted.model,
    });
  }, [initialLayout, initialPersisted.model]);

  useEffect(() => {
    if (!saveLayout) return;
    const envelope = layoutService.normalize({
      ...initialPersisted,
      revision: revisionRef.current,
      model,
      presentation,
      updatedAt: new Date().toISOString(),
    });
    const timer = window.setTimeout(async () => {
      try {
        const res = (await saveLayout(JSON.stringify(envelope), revisionRef.current)) as
          { revision?: number } | undefined;
        if (res && typeof res.revision === 'number') {
          revisionRef.current = res.revision;
        }
      } catch {
        // Revision conflict or network error — handled gracefully
      }
    }, 500);
    return () => window.clearTimeout(timer);
  }, [layoutService, model, initialPersisted, presentation, projectId, saveLayout]);

  const persistence = useMemo<WorkspaceLayoutPersistence>(
    () => ({
      initialPresentation: initialPersisted.presentation,
      registerPresentation: setPresentation,
    }),
    [initialPersisted.presentation],
  );

  const value = useMemo<WorkspaceContextValue>(
    () => ({
      model,
      open: (surface, stackId) => dispatch({ type: 'open', surface, stackId }),
      ensure: (surface, stackId) => dispatch({ type: 'ensure', surface, stackId }),
      activate: (surfaceId) => dispatch({ type: 'activate', surfaceId }),
      close: (surfaceId) => {
        const surface = listSurfaces(model.root).find(
          (candidate) =>
            candidate.id === surfaceId ||
            candidate.viewId === surfaceId ||
            candidate.legacyId === surfaceId ||
            candidate.logicalKey === surfaceId,
        );
        if (surface && onSurfaceClosed)
          void Promise.resolve(onSurfaceClosed(surface)).catch(() => undefined);
        dispatch({ type: 'close', surfaceId });
      },
      updateSurface: (surfaceId, patch) => dispatch({ type: 'updateSurface', surfaceId, patch }),
      split: (relativeSurfaceId, surface, direction) =>
        dispatch({ type: 'split', relativeSurfaceId, surface, direction }),
      move: (surfaceId, targetStackId) => dispatch({ type: 'move', surfaceId, targetStackId }),
      resize: (splitId, ratio) => dispatch({ type: 'resize', splitId, ratio }),
      maximize: (surfaceId) => dispatch({ type: 'maximize', surfaceId }),
      reset: () => dispatch({ type: 'replace', model: fallback }),
    }),
    [model, fallback, onSurfaceClosed],
  );

  return (
    <WorkspaceLayoutPersistenceContext.Provider value={persistence}>
      <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>
    </WorkspaceLayoutPersistenceContext.Provider>
  );
};

export function useWorkspace(): WorkspaceContextValue {
  const context = useContext(WorkspaceContext);
  if (!context) throw new Error('useWorkspace must be used inside WorkspaceProvider');
  return context;
}
