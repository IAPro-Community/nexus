import React, { useEffect } from 'react';
import { listSurfaces } from './model';
import { DesktopWindow } from './DesktopWindow';
import { useWorkspace } from './WorkspaceProvider';
import { ensureDesktopWindow, focusDesktopWindow, moveDesktopWindow, toggleDesktopMaximize } from './presentation';
import type { WorkspaceSurface } from './model';

export const DesktopWorkspace: React.FC<{ renderSurface: (surface: WorkspaceSurface) => React.ReactNode }> = ({ renderSurface }) => {
  const workspace = useWorkspace();
  const surfaces = listSurfaces(workspace.model.root);

  useEffect(() => {
    for (const surface of surfaces) {
      const viewId = surface.viewId || surface.id;
      if (!workspace.presentation.windows[viewId]) {
        workspace.updatePresentation((state) => ensureDesktopWindow(state, viewId));
        break;
      }
    }
  }, [surfaces, workspace.presentation.windows, workspace.setPresentationMode]);

  return <div className="nx-desktop-workspace" data-tour="workspace">
    {surfaces.map((surface) => {
      const viewId = surface.viewId || surface.id;
      const state = workspace.presentation.windows[viewId];
      if (!state) return null;
      return <DesktopWindow
        key={viewId}
        surface={surface}
        windowState={state}
        onFocus={() => workspace.updatePresentation((state) => focusDesktopWindow(state, viewId))}
        onMove={(x, y) => workspace.updatePresentation((state) => moveDesktopWindow(state, viewId, x, y))}
        onMaximize={() => workspace.updatePresentation((state) => toggleDesktopMaximize(state, viewId))}
        onClose={() => workspace.close(surface.id)}
      >{renderSurface(surface)}</DesktopWindow>;
    })}
  </div>;
};
