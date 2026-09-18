import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { NexusWorkspaceApp } from './app/NexusWorkspaceApp';
import { NexusDemoApp } from './app/NexusDemoApp';
import { ConversationSpikePage } from './app/ConversationSpikePage';
import { ErrorBoundary } from './components/ErrorBoundary';
import type { WorkspaceSurface } from './workspace/model';

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
  if (
    typeof window !== 'undefined' &&
    new URLSearchParams(window.location.search).get('demo') === '1'
  ) {
    return <NexusDemoApp />;
  }

  return (
    <BrowserRouter>
      <ErrorBoundary>
        <AppRoutes />
      </ErrorBoundary>
    </BrowserRouter>
  );
};
