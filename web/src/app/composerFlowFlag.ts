import { useSyncExternalStore } from 'react';
import {
  createStack,
  type WorkspaceModel,
  type WorkspaceNode,
  type WorkspaceSurface,
} from '../workspace/model';

import { request } from '../api';

export const OPTIONAL_FEATURE_KEYS = [
  'COMPOSER',
  'FLOW',
  'MISSIONS',
  'PARALLEL_CHALLENGE',
  'COMPARISON_LAB',
  'VOICE_ACTIONS',
  'NEX_PROACTIVITY',
  'DESIGN_INSPECT',
  'VISUAL_DIFF',
  'SMART_LAYOUTS',
  'DEVICE_PREVIEW',
] as const;

export type OptionalFeatureKey = (typeof OPTIONAL_FEATURE_KEYS)[number];

// One immutable, observable snapshot per frontend instance. The server remains
// the only execution authority. Legacy exported bindings are compatibility
// projections for callers not yet migrated, not independent configuration.
function parseOptionalFeatures(snapshot: unknown) {
  const record =
    snapshot && typeof snapshot === 'object' ? (snapshot as Record<string, unknown>) : {};
  const features =
    record.features && typeof record.features === 'object'
      ? (record.features as Record<string, unknown>)
      : {};
  const global = record.global === true;
  const enabled = (name: string) => global && features[name] === true;
  const flags = Object.freeze(
    Object.fromEntries(OPTIONAL_FEATURE_KEYS.map((key) => [key, enabled(key)])) as Record<
      OptionalFeatureKey,
      boolean
    >,
  );
  return Object.freeze({
    OPTIONAL_FEATURES_ENABLED: global,
    COMPOSER_FLOW_ENABLED: enabled('COMPOSER') && enabled('FLOW') && enabled('MISSIONS'),
    VOICE_ACTIONS_ENABLED: enabled('VOICE_ACTIONS'),
    DESIGN_INSPECT_ENABLED: enabled('DESIGN_INSPECT'),
    VISUAL_DIFF_ENABLED: enabled('VISUAL_DIFF'),
    NEX_PROACTIVITY_ENABLED: enabled('NEX_PROACTIVITY'),
    COMPARISON_LAB_ENABLED: enabled('COMPARISON_LAB'),
    SMART_LAYOUTS_ENABLED: enabled('SMART_LAYOUTS'),
    DEVICE_PREVIEW_ENABLED: enabled('DEVICE_PREVIEW'),
    FLAGS: flags,
  });
}

const disabled = parseOptionalFeatures(null);
let effective = disabled;
const subscribers = new Set<() => void>();
let loadGeneration = 0;

export function getOptionalFeatures() {
  return effective;
}

export function subscribeOptionalFeatures(listener: () => void) {
  subscribers.add(listener);
  return () => {
    subscribers.delete(listener);
  };
}

export function useOptionalFeatures() {
  return useSyncExternalStore(subscribeOptionalFeatures, getOptionalFeatures, () => disabled);
}

// Compatibility bindings. New React consumers use the observable snapshot.
export let {
  OPTIONAL_FEATURES_ENABLED,
  COMPOSER_FLOW_ENABLED,
  VOICE_ACTIONS_ENABLED,
  DESIGN_INSPECT_ENABLED,
  VISUAL_DIFF_ENABLED,
  NEX_PROACTIVITY_ENABLED,
  COMPARISON_LAB_ENABLED,
  SMART_LAYOUTS_ENABLED,
  DEVICE_PREVIEW_ENABLED,
} = disabled;

function publish(snapshot: unknown) {
  effective = parseOptionalFeatures(snapshot);
  ({
    OPTIONAL_FEATURES_ENABLED,
    COMPOSER_FLOW_ENABLED,
    VOICE_ACTIONS_ENABLED,
    DESIGN_INSPECT_ENABLED,
    VISUAL_DIFF_ENABLED,
    NEX_PROACTIVITY_ENABLED,
    COMPARISON_LAB_ENABLED,
    SMART_LAYOUTS_ENABLED,
    DEVICE_PREVIEW_ENABLED,
  } = effective);
  for (const listener of subscribers) listener();
}

export function applyOptionalFeatureSnapshot(snapshot: unknown): void {
  // Explicit reset/update invalidates older in-flight authentication requests.
  loadGeneration++;
  publish(snapshot);
}

export async function loadOptionalFeatures(): Promise<void> {
  const generation = ++loadGeneration;
  publish(null);
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 4000);
  try {
    const snapshot = await request<unknown>('/api/v1/system/features', {
      signal: controller.signal,
    });
    if (generation === loadGeneration) publish(snapshot);
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
