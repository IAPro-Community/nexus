import React, { Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { isDesktopHudApp } from './app/desktopHudDetect';
import { ErrorBoundary } from './components/ErrorBoundary';
import { SurfaceLoadingFallback } from './app/SurfaceLoadingFallback';
import type { WorkspaceSurface } from './workspace/model';

// Secondary surfaces stay out of the boot chunk: the workspace is the only
// route every session needs, while the demo, the spike page and the native HUD
// window are loaded on demand so their chunks are never parsed at startup.
const NexusWorkspaceApp = React.lazy(() =>
  import('./app/NexusWorkspaceApp').then((m) => ({ default: m.NexusWorkspaceApp })),
);
const NexusDemoApp = React.lazy(() =>
  import('./app/NexusDemoApp').then((m) => ({ default: m.NexusDemoApp })),
);
const ConversationSpikePage = React.lazy(() =>
  import('./app/ConversationSpikePage').then((m) => ({ default: m.ConversationSpikePage })),
);
const DesktopHudApp = React.lazy(() =>
  import('./app/DesktopHudApp').then((m) => ({ default: m.DesktopHudApp })),
);

function isWorkspaceSurface(obj: unknown): obj is WorkspaceSurface {
  if (typeof obj !== 'object' || obj === null) return false;
  const o = obj as Record<string, unknown>;
  return typeof o.id === 'string' && typeof o.type === 'string' && typeof o.title === 'string';
}

function parseLegacyPopoutSurface(): WorkspaceSurface | undefined {
  if (typeof window === 'undefined') return undefined;
  const raw = new URLSearchParams(window.location.search).get('popout');
  if (!raw) return undefined;
  try {
    const parsed = JSON.parse(raw);
    if (!isWorkspaceSurface(parsed)) {
      console.error('Invalid workspace surface from URL');
      return undefined;
    }
    return parsed;
  } catch {
    try {
      const parsed = JSON.parse(decodeURIComponent(raw));
      if (!isWorkspaceSurface(parsed)) {
        console.error('Invalid workspace surface from URL (decoded)');
        return undefined;
      }
      return parsed;
    } catch {
      return undefined;
    }
  }
}

export const AppRoutes: React.FC = () => {
  const legacyPopout = parseLegacyPopoutSurface();
  if (legacyPopout) {
    return <NexusWorkspaceApp popoutSurface={legacyPopout} />;
  }

  return (
    <Routes>
      <Route path="/demo" element={<NexusDemoApp />} />
      <Route path="/dev/conversation-spike" element={<ConversationSpikePage />} />
      <Route path="/p/:projectId/popout/:surface" element={<NexusWorkspaceApp />} />
      <Route path="/p/:projectId/*" element={<NexusWorkspaceApp />} />
      <Route path="/projects" element={<NexusWorkspaceApp initialGlobalSurface="projects" />} />
      <Route path="/settings" element={<NexusWorkspaceApp initialGlobalSurface="settings" />} />
      <Route path="/updates" element={<NexusWorkspaceApp initialGlobalSurface="updates" />} />
      <Route path="/welcome" element={<NexusWorkspaceApp initialGlobalSurface="welcome" />} />
      <Route path="/" element={<NexusWorkspaceApp />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
};

export const App: React.FC = () => {
  if (isDesktopHudApp()) {
    return (
      <Suspense fallback={null}>
        <DesktopHudApp />
      </Suspense>
    );
  }

  if (
    typeof window !== 'undefined' &&
    new URLSearchParams(window.location.search).get('demo') === '1'
  ) {
    return (
      <Suspense fallback={<SurfaceLoadingFallback />}>
        <NexusDemoApp />
      </Suspense>
    );
  }

  return (
    <BrowserRouter>
      <ErrorBoundary>
        <Suspense fallback={<SurfaceLoadingFallback />}>
          <AppRoutes />
        </Suspense>
      </ErrorBoundary>
    </BrowserRouter>
  );
};
