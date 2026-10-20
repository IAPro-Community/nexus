import {
  createStack,
  type WorkspaceModel,
  type WorkspaceNode,
  type WorkspaceSurface,
} from '../workspace/model';

// Build-time opt-in. Regular 1.0.0 builds omit the experimental surfaces.
// Enable only in a deliberately built homologation artifact.
declare const __NEXUS_EXPERIMENTAL_COMPOSER_FLOW__: boolean;
export const COMPOSER_FLOW_ENABLED =
  typeof __NEXUS_EXPERIMENTAL_COMPOSER_FLOW__ !== 'undefined' &&
  __NEXUS_EXPERIMENTAL_COMPOSER_FLOW__;

export function isExperimentalComposerFlowSurface(type: string): boolean {
  return type === 'work' || type === 'missions' || type === 'flow-run';
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
