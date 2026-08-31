import type { WorkspaceSurface } from '../workspace/model';

export type ProjectSurfaceKind =
  | 'projects'
  | 'overview'
  | 'work'
  | 'missions'
  | 'agents'
  | 'project-shell'
  | 'flow-run'
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
  work: 'Work',
  missions: 'Plan',
  agents: 'Agents',
  'project-shell': 'Project Shell',
  'flow-run': 'Flow Run',
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
  return {
    id: `project:${projectId}:${kind}`,
    viewId: `view:project:${projectId}:${kind}`,
    logicalKey: `project:${projectId}:${kind}`,
    type: kind,
    title: labels[kind],
    titleKey,
    closable: kind !== 'overview',
    data: { projectId },
  };
}

export function agentTerminalSurface(agentId: string, agentName: string, initialPrompt = ''): WorkspaceSurface {
  return {
    id: `agent:${agentId}:terminal`,
    viewId: `view:agent:${agentId}:terminal`,
    logicalKey: `session:${agentId}`,
    type: 'terminal',
    title: agentName,
    subtitle: 'Persistent Agent terminal',
    data: initialPrompt ? { agentId, initialPrompt } : { agentId },
    closable: true,
  };
}

export function agentConfigSurface(agentId: string, agentName: string): WorkspaceSurface {
  return {
    id: `agent:${agentId}:config`,
    viewId: `view:agent:${agentId}:config`,
    logicalKey: `agent:${agentId}:config`,
    type: 'agent-config',
    title: `${agentName} · Configure`,
    titleKey: 'workspace.configureAgent',
    titleParams: { name: agentName },
    data: { agentId },
    closable: true,
  };
}

export function flowRunSurface(runId: string, title = 'Flow Run'): WorkspaceSurface {
  return {
    id: `flow-run:${runId}`,
    viewId: `view:flow-run:${runId}`,
    logicalKey: `flow-run:${runId}`,
    type: 'flow-run',
    title,
    data: { runId },
    closable: true,
  };
}
