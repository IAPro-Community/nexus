import {
  createStack,
  type WorkspaceModel,
  type WorkspaceNode,
  type WorkspaceSurface,
} from '../workspace/model';

// Enabled by default for all builds. Can be explicitly disabled via compile-time define
// __NEXUS_EXPERIMENTAL_COMPOSER_FLOW__ = false.
declare const __NEXUS_EXPERIMENTAL_COMPOSER_FLOW__: boolean | undefined;
export const COMPOSER_FLOW_ENABLED =
  typeof __NEXUS_EXPERIMENTAL_COMPOSER_FLOW__ === 'undefined'
    ? true
    : Boolean(__NEXUS_EXPERIMENTAL_COMPOSER_FLOW__);

export function isExperimentalComposerFlowSurface(type: string): boolean {
  return type === 'work' || type === 'missions' || type === 'flow-run';
}

/**
 * A mission detail is an actionable, stable surface even while mission
 * creation and history remain behind the Composer/Flow release gate.
 */
export function isExperimentalRouteBlocked(type: string, subId?: string): boolean {
  if (!isExperimentalComposerFlowSurface(type)) return false;
  return !(type === 'missions' && Boolean(subId?.trim()));
}

/**
 * Workspace detail tabs are allowed only when they point to a specific
 * mission run. Generic Flow/Composer surfaces remain gated in stable builds.
 */
export function isExperimentalWorkspaceSurfaceBlocked(
  surface: Pick<WorkspaceSurface, 'type' | 'data'>,
): boolean {
  if (!isExperimentalComposerFlowSurface(surface.type)) return false;
  return !(surface.type === 'flow-run' && Boolean(surface.data?.runId?.trim()));
}

/**
 * Remove beta-only product views from a restored layout. Does not delete
 * Composer sessions, plans, mission runs or any backend state.
 * Keep the remaining split layout and restore a valid active tab.
 */
export function withoutExperimentalComposerFlow(
  model: WorkspaceModel,
  fallback: WorkspaceSurface,
): WorkspaceModel {
  const strip = (node: WorkspaceNode): WorkspaceNode | null => {
    if (node.kind === 'stack') {
      const tabs = node.tabs.filter((tab) => !isExperimentalComposerFlowSurface(tab.type));
      if (tabs.length === 0) return null;
      return {
        ...node,
        tabs,
        activeId: tabs.some((tab) => tab.id === node.activeId) ? node.activeId : tabs[0].id,
      };
    }
    const first = strip(node.first);
    const second = strip(node.second);
    if (!first) return second;
    if (!second) return first;
    return { ...node, first, second };
  };
  const root = strip(model.root) ?? createStack(fallback);
  const maximizedSurfaceId = model.maximizedSurfaceId;
  const stillExists = (node: WorkspaceNode): boolean =>
    node.kind === 'stack'
      ? node.tabs.some((tab) => tab.id === maximizedSurfaceId)
      : stillExists(node.first) || stillExists(node.second);
  return {
    ...model,
    root,
    maximizedSurfaceId: maximizedSurfaceId && stillExists(root) ? maximizedSurfaceId : undefined,
  };
}
