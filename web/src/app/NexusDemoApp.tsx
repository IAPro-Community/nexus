import React, { useCallback, useMemo, useState } from 'react';
import { Badge, Card, ThemeProvider } from '../design-system';
import type { Agent, Project } from '../types';
import { WorkspaceProvider, useWorkspace } from '../workspace/WorkspaceProvider';
import { WorkspaceRenderer } from '../workspace/WorkspaceRenderer';
import type { WorkspaceSurface } from '../workspace/model';
import { ProjectOverviewSurface } from '../features/overview/ProjectOverviewSurface';
import { WorkSurface } from '../features/work/WorkSurface';
import { SettingsSurface } from '../features/settings/SettingsSurface';
import { NexusShell } from './NexusShell';
import { CommandPalette } from './commands/CommandPalette';
import type { NexusCommand } from './commands/registry';
import { ProductTour } from './tour/ProductTour';
import { projectSurface } from './surfaces';

const project: Project = {
  id: 'demo-project',
  name: 'Payment Platform',
  slug: 'payment-platform',
  canonical_path: '/demo/payment-platform',
  repo_remote: '',
  repo_url: '',
  default_branch: 'feat/payment-flow',
  maestro_mode: 'ASSIST',
  resource_policy: 'BALANCED',
  default_isolation: 'worktree',
  settings: '',
  created_at: '',
  updated_at: '',
};
const agents: Agent[] = [
  {
    id: 'demo-backend',
    project_id: project.id,
    name: 'Backend',
    role: 'Backend Developer',
    status: 'WORKING',
    continuity_status: 'NATIVE_RESUME_VERIFIED',
    created_at: '',
    updated_at: '',
    last_started_at: '2m ago',
  },
  {
    id: 'demo-frontend',
    project_id: project.id,
    name: 'Frontend',
    role: 'Frontend Developer',
    status: 'WORKING',
    continuity_status: 'LIVE_SAME_RUNTIME',
    created_at: '',
    updated_at: '',
    last_started_at: '5m ago',
  },
  {
    id: 'demo-mobile',
    project_id: project.id,
    name: 'Mobile',
    role: 'Mobile Developer',
    status: 'WAITING',
    continuity_status: 'REATTACHED_SAME_RUNTIME',
    created_at: '',
    updated_at: '',
    last_started_at: '12m ago',
  },
  {
    id: 'demo-reviewer',
    project_id: project.id,
    name: 'Reviewer',
    role: 'Independent Reviewer',
    status: 'RECOVERABLE',
    continuity_status: 'CONTEXT_RECOVERED_NEW_SESSION',
    created_at: '',
    updated_at: '',
    last_started_at: '1h ago',
  },
];

export const NexusDemoApp: React.FC = () => (
  <ThemeProvider>
    <WorkspaceProvider projectId={project.id}>
      <DemoCoordinator />
    </WorkspaceProvider>
  </ThemeProvider>
);
const DemoCoordinator: React.FC = () => {
  const workspace = useWorkspace();
  const [palette, setPalette] = useState(false);
  const [tour, setTour] = useState(false);
  const openKind = useCallback(
    (kind: string) => workspace.open(projectSurface(project.id, kind as any)),
    [workspace],
  );
  const commands = useMemo<NexusCommand[]>(
    () => [
      { id: 'overview', label: 'Open Overview', group: 'Demo', run: () => openKind('overview') },
      { id: 'work', label: 'Open Work', group: 'Demo', run: () => openKind('work') },
      { id: 'settings', label: 'Open Settings', group: 'Demo', run: () => openKind('settings') },
      { id: 'tour', label: 'Take Product Tour', group: 'Demo', run: () => setTour(true) },
    ],
    [openKind],
  );
  const render = (surface: WorkspaceSurface) => {
    if (surface.type === 'overview')
      return (
        <ProjectOverviewSurface
          project={project}
          agents={agents}
          onOpenAgent={() => openKind('agents')}
          onNewAISession={() => openKind('work')}
          onProjectShell={() => openKind('work')}
          onOpenComposer={() => openKind('work')}
          onOpenFlow={() => openKind('missions')}
        />
      );
    if (surface.type === 'work')
      return <WorkSurface project={project} agents={agents} onDirect={() => openKind('agents')} />;
    if (surface.type === 'settings') return <SettingsSurface onTour={() => setTour(true)} />;
    return (
      <div className="nx-surface-scroll">
        <div className="nx-page-header">
          <div>
            <span className="nx-eyebrow">Synthetic Demo</span>
            <h1>{surface.title}</h1>
            <p>
              This demo surface uses synthetic data only. Connect the real backend for operational
              actions.
            </p>
          </div>
          <Badge tone="warning">DEMO</Badge>
        </div>
        <Card>
          <strong>Workspace OS preview</strong>
          <p className="nx-muted-copy">
            Use product tabs, the Terminals tab, taskbar and the command palette without touching
            real provider sessions.
          </p>
        </Card>
      </div>
    );
  };
  return (
    <>
      <NexusShell
        project={project}
        agents={agents}
        rail={
          <div className="nx-demo-rail">
            <span className="nx-brand-mark">
              <img src="./nexus-icon.png" alt="Nexus" className="nx-brand-mark__img" />
            </span>
            <strong>DEMO</strong>
            <small>Synthetic data</small>
          </div>
        }
        onOpenRail={() => undefined}
        onOpenSurface={openKind}
        onCommand={() => setPalette(true)}
        onOpenWelcome={() => setTour(true)}
        onOpenProjectManager={() => undefined}
        onSettings={() => openKind('settings')}
      >
        <WorkspaceRenderer renderSurface={render} />
      </NexusShell>
      <CommandPalette open={palette} onClose={() => setPalette(false)} commands={commands} />
      <ProductTour open={tour} onClose={() => setTour(false)} />
    </>
  );
};
