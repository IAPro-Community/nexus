import {
  createStack,
  type WorkspaceModel,
  type WorkspaceNode,
  type WorkspaceSurface,
} from '../workspace/model';

// Experimental product surfaces are OFF in every build by default.
// Even explicit frontend opt-ins cannot authorize server-side execution.
declare const __NEXUS_OPTIONAL_FEATURES__: boolean | undefined;
declare const __NEXUS_EXPERIMENTAL_COMPOSER_FLOW__: boolean | undefined;
declare const __NEXUS_VOICE_ACTIONS__: boolean | undefined;
declare const __NEXUS_DESIGN_INSPECT__: boolean | undefined;
declare const __NEXUS_VISUAL_DIFF__: boolean | undefined;

// All optional UI is disabled unless the global and resource flags are
// explicitly set at build time. Backend authorization is separate and required.
export const OPTIONAL_FEATURES_ENABLED =
  typeof __NEXUS_OPTIONAL_FEATURES__ !== 'undefined' &&
  __NEXUS_OPTIONAL_FEATURES__ === true;
export const COMPOSER_FLOW_ENABLED =
  OPTIONAL_FEATURES_ENABLED &&
  typeof __NEXUS_EXPERIMENTAL_COMPOSER_FLOW__ !== 'undefined' &&
  __NEXUS_EXPERIMENTAL_COMPOSER_FLOW__ === true;
export const VOICE_ACTIONS_ENABLED =
  OPTIONAL_FEATURES_ENABLED &&
  typeof __NEXUS_VOICE_ACTIONS__ !== 'undefined' &&
  __NEXUS_VOICE_ACTIONS__ === true;
export const DESIGN_INSPECT_ENABLED =
  OPTIONAL_FEATURES_ENABLED &&
  typeof __NEXUS_DESIGN_INSPECT__ !== 'undefined' &&
  __NEXUS_DESIGN_INSPECT__ === true;
export const VISUAL_DIFF_ENABLED =
  OPTIONAL_FEATURES_ENABLED &&
  typeof __NEXUS_VISUAL_DIFF__ !== 'undefined' &&
  __NEXUS_VISUAL_DIFF__ === true;

export function isExperimentalComposerFlowSurface(type: string): boolean {
  return type === 'work' || type === 'missions' || type === 'flow-run';
}

/**
 * All Composer, Flow and Mission surfaces are disabled by default,
 * including deep links to historic runs; their data is not deleted.
 */
export function isExperimentalRouteBlocked(type: string, _subId?: string): boolean {
  if (!isExperimentalComposerFlowSurface(type)) return false;
  return true;
}

/**
 * Restored experimental surfaces are hidden, including run detail tabs.
 * Backend history remains available through authenticated read-only APIs.
 */
export function isExperimentalWorkspaceSurfaceBlocked(
  surface: Pick<WorkspaceSurface, 'type' | 'data'>,
): boolean {
  if (!isExperimentalComposerFlowSurface(surface.type)) return false;
  return true;
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
