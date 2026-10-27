import type { WorkspaceSurface } from '../workspace/model';

export type ProjectSurfaceKind =
  | 'projects'
  | 'overview'
  | 'work'
  | 'missions'
  | 'terminals'
  | 'agents'
  | 'maestro'
  | 'sessions'
  | 'settings'
  | 'resources'
  | 'legacy-runtimes'
  | 'legacy-providers'
  | 'legacy-events';

const labels: Record<ProjectSurfaceKind, string> = {
  projects: 'Workspace Desktops',
  overview: 'Overview',
  work: 'Composer',
  missions: 'Flow',
  terminals: 'Terminals',
  agents: 'Agents',
  maestro: 'Maestro',
  sessions: 'Sessions',
  settings: 'Settings',
  resources: 'Resources',
  'legacy-runtimes': 'Runtime Control',
  'legacy-providers': 'Providers',
  'legacy-events': 'Events',
};

export function projectSurface(projectId: string, kind: ProjectSurfaceKind): WorkspaceSurface {
  const key =
    kind === 'projects'
      ? 'desktopsTitle'
      : kind === 'legacy-runtimes'
        ? 'runtimes'
        : kind === 'legacy-providers'
          ? 'providers'
          : kind === 'legacy-events'
            ? 'events'
            : kind;
  const titleKey = kind === 'projects' ? 'projectManager.desktopsTitle' : `nav.${key}`;
  const logicalKey = `project:${projectId}:${kind}`;
  return {
    id: logicalKey,
    viewId: `view:${logicalKey}`,
    logicalKey,
    type: kind,
    title: labels[kind],
    titleKey,
    closable: kind !== 'overview' && kind !== 'terminals' && kind !== 'work' && kind !== 'missions',
    data: { projectId },
  };
}

export function isPtySurface(surface: WorkspaceSurface): boolean {
  return surface.type === 'terminal' || surface.type === 'project-shell';
}

export function isProductSurface(surface: WorkspaceSurface): boolean {
  return !isPtySurface(surface);
}

export function agentTerminalSurface(
  agentId: string,
  agentName: string,
  initialPrompt = '',
  runtimeId = '',
): WorkspaceSurface {
  const data: Record<string, string> = { agentId };
  if (initialPrompt) data.initialPrompt = initialPrompt;
  if (runtimeId) data.runtimeId = runtimeId;
  const logicalKey = `agent:${agentId}:terminal`;
  return {
    id: logicalKey,
    viewId: `view:${logicalKey}`,
    logicalKey,
    type: 'terminal',
    title: agentName,
    subtitle: 'Persistent Agent terminal',
    data,
    closable: true,
  };
}

export function agentConfigSurface(agentId: string, agentName: string): WorkspaceSurface {
  const logicalKey = `agent:${agentId}:config`;
  return {
    id: logicalKey,
    viewId: `view:${logicalKey}`,
    logicalKey,
    type: 'agent-config',
    title: `${agentName} · Configure`,
    titleKey: 'workspace.configureAgent',
    titleParams: { name: agentName },
    data: { agentId },
    closable: true,
  };
}

export function projectShellSurface(
  projectId: string,
  runtimeId: string,
  title = 'Project Shell',
): WorkspaceSurface {
  const logicalKey = `shell:${runtimeId}`;
  return {
    id: logicalKey,
    viewId: `view:${logicalKey}`,
    logicalKey,
    type: 'project-shell',
    title,
    subtitle: 'Project shell',
    data: { projectId, runtimeId },
    closable: true,
  };
}

export function flowRunSurface(runId: string, title = 'Flow Run'): WorkspaceSurface {
  const logicalKey = `flow-run:${runId}`;
  return {
    id: logicalKey,
    viewId: `view:${logicalKey}`,
    logicalKey,
    type: 'flow-run',
    title,
    subtitle: 'Durable Flow execution',
    data: { runId },
    closable: true,
  };
}

/** Browser lives in the same window manager as terminals, with its own session identity. */
export function browserSurface(projectId: string): WorkspaceSurface {
  const browserId = crypto.randomUUID();
  const logicalKey = `browser:${projectId}:${browserId}`;
  return {
    id: logicalKey,
    viewId: `view:${logicalKey}`,
    logicalKey,
    type: 'browser',
    title: 'Browser',
    data: { projectId, initialUrl: 'http://localhost:3000' },
    closable: true,
  };
}

/** Presentation-only predicate; a browser is never a PTY runtime. */
export function isWorkspaceWindowSurface(surface: WorkspaceSurface): boolean {
  return isPtySurface(surface) || surface.type === 'browser';
}
