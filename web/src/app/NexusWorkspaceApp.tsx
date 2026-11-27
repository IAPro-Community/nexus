import React, { Suspense } from 'react';
import { NexusSplashScreen } from './NexusSplashScreen';
import type { GlobalSurfaceKind } from './routes';
import type { WorkspaceSurface } from '../workspace/model';

// Split point: the session shell (data, routing, workspace) is its own chunk so
// the boot bundle only carries this container and the lazily imported surfaces.
const NexusWorkspaceSession = React.lazy(() =>
  import('./NexusWorkspaceSession').then((m) => ({ default: m.NexusWorkspaceSession })),
);

const NexusWorkspaceAppInner: React.FC<{
  popoutSurface?: WorkspaceSurface;
  initialGlobalSurface?: GlobalSurfaceKind;
}> = ({ popoutSurface, initialGlobalSurface }) => (
  <Suspense fallback={<NexusSplashScreen stage="loading" />}>
    <NexusWorkspaceSession
      popoutSurface={popoutSurface}
      initialGlobalSurface={initialGlobalSurface}
    />
  </Suspense>
);

export const NexusWorkspaceApp = React.memo(NexusWorkspaceAppInner);
