import {
  flattenToSingleStack,
  normalizeSurface,
  surfaceLogicalKey,
  type WorkspaceModel,
  type WorkspaceNode,
  type WorkspaceSurface,
} from './model';

function validNode(node: unknown, depth = 0): node is WorkspaceNode {
  // Imported or corrupted layouts must not be able to recurse forever or
  // construct unbounded tab arrays when loaded from browser storage.
  if (!node || typeof node !== 'object' || depth > 16) return false;
  const candidate = node as Partial<WorkspaceNode> & Record<string, unknown>;
  if (candidate.kind === 'stack') {
    return (
      typeof candidate.id === 'string' &&
      Array.isArray(candidate.tabs) &&
      candidate.tabs.length <= 512 &&
      candidate.tabs.every((tab: unknown) => {
        if (!tab || typeof tab !== 'object') return false;
        const surface = tab as Partial<WorkspaceSurface>;
        return (
          typeof surface.id === 'string' &&
          typeof surface.type === 'string' &&
          typeof surface.title === 'string'
        );
      }) &&
      typeof candidate.activeId === 'string'
    );
  }
  if (candidate.kind === 'split') {
    return (
      typeof candidate.id === 'string' &&
      (candidate.direction === 'horizontal' || candidate.direction === 'vertical') &&
      typeof candidate.ratio === 'number' &&
      Number.isFinite(candidate.ratio) &&
      validNode(candidate.first, depth + 1) &&
      validNode(candidate.second, depth + 1)
    );
  }
  return false;
}

/** Validate untrusted local/imported workspace envelopes before normalization. */
export function isValidWorkspaceModel(raw: unknown): raw is WorkspaceModel {
  if (!raw || typeof raw !== 'object') return false;
  const candidate = raw as Partial<WorkspaceModel>;
  return (candidate.version === 1 || candidate.version === 2) && validNode(candidate.root);
}

export const workspaceStorageKey = (projectId: string) => `iapro:nexus:workspace:${projectId}:v2`;

export function serializeWorkspace(model: WorkspaceModel): string {
  return JSON.stringify(model);
}

const PROJECT_SURFACE_KINDS = new Set([
  'projects',
  'overview',
  'work',
  'missions',
  'agents',
  'maestro',
  'sessions',
  'settings',
  'resources',
  'legacy-runtimes',
  'legacy-providers',
  'legacy-events',
]);

function canonicalizeSurface(
  surface: WorkspaceSurface,
  fallbackProjectId?: string,
): WorkspaceSurface {
  const norm = normalizeSurface(surface);
  const projId = fallbackProjectId || norm.data?.projectId;

  if (norm.id === 'project-overview' || (norm.type === 'overview' && projId)) {
    const effectiveProj = projId || 'default';
    const logicalKey = `project:${effectiveProj}:overview`;
    return {
      ...norm,
      id: logicalKey,
      viewId: `view:${logicalKey}`,
      logicalKey,
      type: 'overview',
      closable: false,
      data: { ...norm.data, projectId: effectiveProj },
    };
  }

  if (PROJECT_SURFACE_KINDS.has(norm.type) && projId) {
    const logicalKey = `project:${projId}:${norm.type}`;
    return {
      ...norm,
      id: logicalKey,
      viewId: `view:${logicalKey}`,
      logicalKey,
      data: { ...norm.data, projectId: projId },
    };
  }

  if (norm.type === 'terminal' && norm.data?.agentId) {
    const logicalKey = `agent:${norm.data.agentId}:terminal`;
    return {
      ...norm,
      id: logicalKey,
      viewId: `view:${logicalKey}`,
      logicalKey,
    };
  }

  if (norm.type === 'agent-config' && norm.data?.agentId) {
    const logicalKey = `agent:${norm.data.agentId}:config`;
    return {
      ...norm,
      id: logicalKey,
      viewId: `view:${logicalKey}`,
      logicalKey,
    };
  }

  return norm;
}

function normalizeNode(node: WorkspaceNode, seen: Set<string>, projectId?: string): WorkspaceNode {
  if (node.kind === 'stack') {
    const tabs = node.tabs
      .filter((tab) => {
        // Discard project surfaces belonging to another project
        if (projectId && tab.data?.projectId && tab.data.projectId !== projectId) {
          // If it is a project surface from another project, drop it so it doesn't pollute the layout
          if (PROJECT_SURFACE_KINDS.has(tab.type) || tab.type === 'overview') {
            return false;
          }
        }
        return true;
      })
      .map((tab) => canonicalizeSurface(tab, projectId))
      .filter((tab) => {
        const key = surfaceLogicalKey(tab);
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
      });
    const active = tabs.find(
      (tab) =>
        tab.id === node.activeId ||
        tab.viewId === node.activeId ||
        tab.legacyId === node.activeId ||
        tab.logicalKey === node.activeId,
    );
    return { ...node, tabs, activeId: active?.id || tabs[0]?.id || '' };
  }
  return {
    ...node,
    first: normalizeNode(node.first, seen, projectId),
    second: normalizeNode(node.second, seen, projectId),
  };
}

function pruneNode(node: WorkspaceNode): WorkspaceNode | null {
  if (node.kind === 'stack') {
    return node.tabs.length > 0 ? node : null;
  }
  const first = pruneNode(node.first);
  const second = pruneNode(node.second);
  if (!first && !second) return null;
  if (!first) return second;
  if (!second) return first;
  return { ...node, first, second };
}

function migrate(
  root: WorkspaceNode,
  fallback: WorkspaceModel,
  projectId?: string,
): WorkspaceModel {
  const seen = new Set<string>();
  const normalizedRoot = normalizeNode(root, seen, projectId);
  const pruned = pruneNode(normalizedRoot);
  if (!pruned) return fallback;
  return flattenToSingleStack({ version: 2, root: pruned });
}

export function deserializeWorkspace(
  raw: string | null | undefined,
  fallback: WorkspaceModel,
  projectId?: string,
): WorkspaceModel {
  if (!raw) return fallback;
  try {
    const parsed = JSON.parse(raw) as {
      version?: number;
      root?: WorkspaceNode;
      maximizedSurfaceId?: string;
    };
    const effectiveProj =
      projectId ||
      (fallback.root.kind === 'stack' ? fallback.root.tabs[0]?.data?.projectId : undefined);
    if (isValidWorkspaceModel(parsed)) {
      return migrate(parsed.root, fallback, effectiveProj);
    }
    return fallback;
  } catch {
    return fallback;
  }
}
