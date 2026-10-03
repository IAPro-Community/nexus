import type { Agent } from '../types';
import type { WorkspaceSurface } from '../workspace/model';
import {
  agentConfigSurface,
  agentTerminalSurface,
  flowRunSurface,
  projectShellSurface,
  projectSurface,
  type ProjectSurfaceKind,
} from './surfaces';

export type GlobalSurfaceKind = 'projects' | 'settings' | 'updates' | 'welcome';

export type ParsedRoute =
  | { kind: 'root' }
  | { kind: 'global'; surface: GlobalSurfaceKind }
  | {
      kind: 'project';
      projectId: string;
      surface: ProjectSurfaceKind;
      subId?: string;
      action?: 'config' | 'view';
    }
  | {
      kind: 'popout';
      projectId: string;
      surface: string;
    };

export const validProjectSurfaces = new Set<ProjectSurfaceKind>([
  'projects',
  'overview',
  'work',
  'missions',
  'terminals',
  'agents',
  'maestro',
  'sessions',
  'settings',
  'resources',
  'design-mode',
  'legacy-runtimes',
  'legacy-providers',
  'legacy-events',
]);

export const validGlobalSurfaces = new Set<GlobalSurfaceKind>([
  'projects',
  'settings',
  'updates',
  'welcome',
]);

export function globalSurfaceToProjectSurface(
  surface: GlobalSurfaceKind,
): ProjectSurfaceKind | null {
  return surface === 'updates' || surface === 'settings' ? 'settings' : null;
}

function decodeSegment(value: string): string | null {
  try {
    const decoded = decodeURIComponent(value);
    return decoded ? decoded : null;
  } catch {
    return null;
  }
}

export function parseRouteLocation(pathname: string, _search = ''): ParsedRoute {
  const normalized = pathname.replace(/\/+$/, '') || '/';

  if (normalized === '/') {
    return { kind: 'root' };
  }

  const parts = normalized.split('/').filter(Boolean);

  // Global routes: /projects, /settings, /updates, /welcome
  if (parts.length === 1 && validGlobalSurfaces.has(parts[0] as GlobalSurfaceKind)) {
    return { kind: 'global', surface: parts[0] as GlobalSurfaceKind };
  }

  // Project routes: /p/:projectId...
  if (parts[0] === 'p' && parts[1]) {
    const projectId = decodeSegment(parts[1]);
    if (!projectId) return { kind: 'root' };

    // Popout route: /p/:projectId/popout/:surface
    if (parts[2] === 'popout' && parts[3]) {
      return {
        kind: 'popout',
        projectId,
        surface: decodeSegment(parts[3]) || 'overview',
      };
    }

    const surfaceParam = parts[2] ? decodeSegment(parts[2]) || 'overview' : 'overview';
    const surface = validProjectSurfaces.has(surfaceParam as ProjectSurfaceKind)
      ? (surfaceParam as ProjectSurfaceKind)
      : 'overview';

    const subId = parts[3] ? decodeSegment(parts[3]) || undefined : undefined;
    const action = parts[4] === 'config' ? 'config' : parts[4] ? 'view' : undefined;

    return {
      kind: 'project',
      projectId,
      surface,
      subId,
      action,
    };
  }

  // Fallback for unrecognized paths: treat as root
  return { kind: 'root' };
}

export function buildProjectRoute(
  projectId: string,
  surface: ProjectSurfaceKind = 'overview',
  subId?: string,
  action?: string,
): string {
  const encProj = encodeURIComponent(projectId);
  if (!subId) {
    return `/p/${encProj}/${surface}`;
  }
  const encSub = encodeURIComponent(subId);
  if (action) {
    return `/p/${encProj}/${surface}/${encSub}/${encodeURIComponent(action)}`;
  }
  return `/p/${encProj}/${surface}/${encSub}`;
}

export function buildPopoutRoute(projectId: string, surface: string): string {
  return `/p/${encodeURIComponent(projectId)}/popout/${encodeURIComponent(surface)}`;
}

export function routeToWorkspaceSurface(
  route: ParsedRoute,
  context?: { agents?: Agent[] },
): WorkspaceSurface | null {
  if (route.kind !== 'project') {
    return null;
  }

  const { projectId, surface, subId, action } = route;

  if (surface === 'missions' && subId) {
    return flowRunSurface(subId, `Flow Run · ${subId.slice(-6)}`);
  }

  if (surface === 'terminals' && subId) {
    return projectShellSurface(projectId, subId, 'Terminal');
  }

  if (surface === 'agents' && subId) {
    const agent = context?.agents?.find((a) => a.id === subId);
    const agentName = agent?.name || subId;
    if (action === 'config') {
      return agentConfigSurface(subId, agentName);
    }
    return agentTerminalSurface(subId, agentName);
  }

  return projectSurface(projectId, surface);
}
