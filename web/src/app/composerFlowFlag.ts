import {
  createStack,
  type WorkspaceModel,
  type WorkspaceNode,
  type WorkspaceSurface,
} from '../workspace/model';

import { request } from '../api';

// The authenticated server snapshot is authoritative. Load it before mounting
// the workspace/HUD; missing or invalid values are never treated as enabled.
export let OPTIONAL_FEATURES_ENABLED = false;
export let COMPOSER_FLOW_ENABLED = false;
export let VOICE_ACTIONS_ENABLED = false;
export let DESIGN_INSPECT_ENABLED = false;
export let VISUAL_DIFF_ENABLED = false;
export let NEX_PROACTIVITY_ENABLED = false;
export let COMPARISON_LAB_ENABLED = false;
export let SMART_LAYOUTS_ENABLED = false;
export let DEVICE_PREVIEW_ENABLED = false;

export function applyOptionalFeatureSnapshot(snapshot: unknown): void {
  const record =
    snapshot && typeof snapshot === 'object' ? (snapshot as Record<string, unknown>) : {};
  const features =
    record.features && typeof record.features === 'object'
      ? (record.features as Record<string, unknown>)
      : {};
  OPTIONAL_FEATURES_ENABLED = record.global === true;
  const enabled = (name: string) => OPTIONAL_FEATURES_ENABLED && features[name] === true;
  COMPOSER_FLOW_ENABLED = enabled('COMPOSER') && enabled('FLOW') && enabled('MISSIONS');
  VOICE_ACTIONS_ENABLED = enabled('VOICE_ACTIONS');
  DESIGN_INSPECT_ENABLED = enabled('DESIGN_INSPECT');
  VISUAL_DIFF_ENABLED = enabled('VISUAL_DIFF');
  NEX_PROACTIVITY_ENABLED = enabled('NEX_PROACTIVITY');
  COMPARISON_LAB_ENABLED = enabled('COMPARISON_LAB');
  SMART_LAYOUTS_ENABLED = enabled('SMART_LAYOUTS');
  DEVICE_PREVIEW_ENABLED = enabled('DEVICE_PREVIEW');
}

export async function loadOptionalFeatures(): Promise<void> {
  applyOptionalFeatureSnapshot(null);
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 4000);
  try {
    const snapshot = await request<unknown>('/api/v1/system/features', {
      signal: controller.signal,
    });
    applyOptionalFeatureSnapshot(snapshot);
  } catch {
    // Older hosts, unavailable hosts and malformed responses remain OFF.
  } finally {
    clearTimeout(timeout);
  }
}

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
