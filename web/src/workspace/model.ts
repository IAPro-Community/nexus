export type WorkspaceDirection = 'horizontal' | 'vertical';

export interface WorkspaceSurface {
  id: string;
  /** Stable visual identity; id remains the compatibility identifier. */
  viewId?: string;
  /** Previous identifier retained for migration/addressing compatibility. */
  legacyId?: string;
  /** Stable semantic identity used to avoid duplicate logical Views. */
  logicalKey?: string;
  type: string;
  title: string;
  titleKey?: string;
  titleParams?: Record<string, string | number>;
  subtitle?: string;
  icon?: string;
  closable?: boolean;
  data?: Record<string, string>;
}
export interface WorkspaceStack {
  kind: 'stack';
  id: string;
  tabs: WorkspaceSurface[];
  activeId: string;
}

export interface WorkspaceSplit {
  kind: 'split';
  id: string;
  direction: WorkspaceDirection;
  ratio: number;
  first: WorkspaceNode;
  second: WorkspaceNode;
}

export type WorkspaceNode = WorkspaceStack | WorkspaceSplit;

export interface WorkspaceModel {
  version: 2;
  root: WorkspaceNode;
  maximizedSurfaceId?: string;
}

let idCounter = 0;
const nextId = (prefix: string) =>
  `${prefix}_${Date.now().toString(36)}_${(++idCounter).toString(36)}`;

export function surfaceViewId(surface: WorkspaceSurface): string {
  return surface.viewId || surface.id;
}

export function surfaceLogicalKey(surface: WorkspaceSurface): string {
  return surface.logicalKey || surface.id;
}

export function isSurfaceMatch(surface: WorkspaceSurface, target: string): boolean {
  return (
    surface.id === target ||
    surface.viewId === target ||
    surface.legacyId === target ||
    surface.logicalKey === target
  );
}

export function normalizeSurface(surface: WorkspaceSurface): WorkspaceSurface {
  return {
    ...surface,
    viewId: surface.viewId || surface.id,
    logicalKey: surface.logicalKey || surface.id,
    closable: surface.closable ?? true,
  };
}

export function createStack(surface: WorkspaceSurface): WorkspaceStack {
  const normalized = normalizeSurface(surface);
  return { kind: 'stack', id: nextId('stack'), tabs: [normalized], activeId: normalized.id };
}

export function createWorkspace(surface: WorkspaceSurface): WorkspaceModel {
  return { version: 2, root: createStack(surface) };
}

/** Collapse split/maximize layouts into one tab stack so PTYs stay under Terminais. */
export function flattenToSingleStack(model: WorkspaceModel): WorkspaceModel {
  const tabs = listSurfaces(model.root);
  if (model.root.kind === 'stack' && !model.maximizedSurfaceId) return model;
  const stacks = listStacks(model.root);
  const preferredActive =
    stacks.map((stack) => stack.activeId).find((id) => tabs.some((tab) => tab.id === id)) ||
    tabs[0]?.id ||
    '';
  return {
    version: 2,
    root: {
      kind: 'stack',
      id: model.root.kind === 'stack' ? model.root.id : nextId('stack'),
      tabs,
      activeId: preferredActive,
    },
  };
}

export function clampRatio(ratio: number): number {
  return Math.min(0.8, Math.max(0.2, Number.isFinite(ratio) ? ratio : 0.5));
}

export function findStackContaining(node: WorkspaceNode, surfaceId: string): WorkspaceStack | null {
  if (node.kind === 'stack')
    return node.tabs.some((tab) => isSurfaceMatch(tab, surfaceId)) ? node : null;
  return findStackContaining(node.first, surfaceId) ?? findStackContaining(node.second, surfaceId);
}

export function findStackById(node: WorkspaceNode, stackId: string): WorkspaceStack | null {
  if (node.kind === 'stack') return node.id === stackId ? node : null;
  return findStackById(node.first, stackId) ?? findStackById(node.second, stackId);
}

export function listStacks(node: WorkspaceNode): WorkspaceStack[] {
  if (node.kind === 'stack') return [node];
  return [...listStacks(node.first), ...listStacks(node.second)];
}

export function listSurfaces(node: WorkspaceNode): WorkspaceSurface[] {
  return listStacks(node).flatMap((stack) => stack.tabs);
}

function updateNode(
  node: WorkspaceNode,
  fn: (node: WorkspaceNode) => WorkspaceNode,
): WorkspaceNode {
  const transformed =
    node.kind === 'split'
      ? { ...node, first: updateNode(node.first, fn), second: updateNode(node.second, fn) }
      : node;
  return fn(transformed);
}

function replaceStack(node: WorkspaceNode, stackId: string, next: WorkspaceNode): WorkspaceNode {
  if (node.kind === 'stack') return node.id === stackId ? next : node;
  return {
    ...node,
    first: replaceStack(node.first, stackId, next),
    second: replaceStack(node.second, stackId, next),
  };
}

function deduplicateNode(node: WorkspaceNode, seen: Set<string>): WorkspaceNode {
  if (node.kind === 'split') {
    return {
      ...node,
      first: deduplicateNode(node.first, seen),
      second: deduplicateNode(node.second, seen),
    };
  }

  const tabs = node.tabs.filter((tab) => {
    const key = surfaceDeduplicationKey(tab);
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
  const active = tabs.find((tab) => tab.id === node.activeId);
  return {
    ...node,
    tabs,
    activeId: active?.id || tabs[0]?.id || '',
  };
}

/**
 * Builds the identity used when repairing persisted layouts. Older versions
 * could persist the same logical surface with a new id and without a
 * logicalKey, so id alone is not safe here.
 */
function surfaceDeduplicationKey(surface: WorkspaceSurface): string {
  if (surface.logicalKey) return `logical:${surface.logicalKey}`;

  const data = surface.data ?? {};
  const identity =
    data.projectId || data.agentId || data.runtimeId || data.terminalId || data.flowId;
  if (identity) return `data:${surface.type}:${identity}`;

  return `id:${surface.id}`;
}

/** Remove duplicate logical tabs left by older navigation/layout versions. */
export function deduplicateWorkspaceSurfaces(model: WorkspaceModel): WorkspaceModel {
  return {
    ...model,
    root: deduplicateNode(model.root, new Set<string>()),
  };
}

export function setActiveSurface(model: WorkspaceModel, surfaceId: string): WorkspaceModel {
  const root = updateNode(model.root, (node) => {
    if (node.kind !== 'stack') return node;
    const found = node.tabs.find((tab) => isSurfaceMatch(tab, surfaceId));
    if (!found) return node;
    return { ...node, activeId: found.id };
  });
  return { ...model, root };
}

export function openSurface(
  model: WorkspaceModel,
  surface: WorkspaceSurface,
  targetStackId?: string,
): WorkspaceModel {
  model = deduplicateWorkspaceSurfaces(model);
  const normalized = normalizeSurface(surface);
  const existing = listSurfaces(model.root).find(
    (candidate) =>
      isSurfaceMatch(candidate, normalized.id) ||
      surfaceLogicalKey(candidate) === surfaceLogicalKey(normalized),
  );
  if (existing) return setActiveSurface(model, existing.id);
  const stacks = listStacks(model.root);
  const target = (targetStackId && findStackById(model.root, targetStackId)) || stacks[0];
  if (!target) return createWorkspace(surface);
  const nextStack: WorkspaceStack = {
    ...target,
    tabs: [...target.tabs, normalized],
    activeId: normalized.id,
  };
  return { ...model, root: replaceStack(model.root, target.id, nextStack) };
}

/** Add a surface if missing, without changing the active tab. */
export function ensureSurface(
  model: WorkspaceModel,
  surface: WorkspaceSurface,
  targetStackId?: string,
): WorkspaceModel {
  model = deduplicateWorkspaceSurfaces(model);
  const normalized = normalizeSurface(surface);
  const existing = listSurfaces(model.root).find(
    (candidate) =>
      isSurfaceMatch(candidate, normalized.id) ||
      surfaceLogicalKey(candidate) === surfaceLogicalKey(normalized),
  );
  if (existing) {
    if (normalized.closable === false && existing.closable !== false) {
      return updateSurface(model, existing.id, { closable: false });
    }
    return model;
  }
  const stacks = listStacks(model.root);
  const target = (targetStackId && findStackById(model.root, targetStackId)) || stacks[0];
  if (!target) return createWorkspace(surface);
  const nextStack: WorkspaceStack = {
    ...target,
    tabs: [...target.tabs, normalized],
    // Keep current focus — Terminais must exist without stealing Overview.
    activeId: target.activeId || normalized.id,
  };
  return { ...model, root: replaceStack(model.root, target.id, nextStack) };
}

function removeSurface(
  node: WorkspaceNode,
  surfaceId: string,
): { node: WorkspaceNode | null; removed?: WorkspaceSurface } {
  if (node.kind === 'stack') {
    const removed = node.tabs.find((tab) => isSurfaceMatch(tab, surfaceId));
    if (!removed || removed.closable === false) return { node };
    const tabs = node.tabs.filter((tab) => !isSurfaceMatch(tab, surfaceId));
    if (tabs.length === 0) return { node: null, removed };
    const activeId =
      node.activeId === surfaceId
        ? (tabs[Math.max(0, node.tabs.findIndex((tab) => isSurfaceMatch(tab, surfaceId)) - 1)]
            ?.id ?? tabs[0].id)
        : node.activeId;
    return { node: { ...node, tabs, activeId }, removed };
  }
  const first = removeSurface(node.first, surfaceId);
  if (first.removed) {
    if (!first.node) return { node: node.second, removed: first.removed };
    return { node: { ...node, first: first.node }, removed: first.removed };
  }
  const second = removeSurface(node.second, surfaceId);
  if (second.removed) {
    if (!second.node) return { node: node.first, removed: second.removed };
    return { node: { ...node, second: second.node }, removed: second.removed };
  }
  return { node };
}

export function closeSurface(model: WorkspaceModel, surfaceId: string): WorkspaceModel {
  const result = removeSurface(model.root, surfaceId);
  if (!result.removed || !result.node) return model;
  return {
    ...model,
    root: result.node,
    maximizedSurfaceId:
      model.maximizedSurfaceId === surfaceId ? undefined : model.maximizedSurfaceId,
  };
}

export function splitWithSurface(
  model: WorkspaceModel,
  relativeSurfaceId: string,
  surface: WorkspaceSurface,
  direction: WorkspaceDirection,
  before = false,
): WorkspaceModel {
  const source = findStackContaining(model.root, relativeSurfaceId) ?? listStacks(model.root)[0];
  if (!source) return createWorkspace(surface);
  if (listSurfaces(model.root).some((candidate) => candidate.id === surface.id))
    return setActiveSurface(model, surface.logicalKey || surface.id);
  const newStack = createStack(surface);
  const split: WorkspaceSplit = {
    kind: 'split',
    id: nextId('split'),
    direction,
    ratio: 0.5,
    first: before ? newStack : source,
    second: before ? source : newStack,
  };
  return { ...model, root: replaceStack(model.root, source.id, split) };
}

export function moveSurface(
  model: WorkspaceModel,
  surfaceId: string,
  targetStackId: string,
): WorkspaceModel {
  const source = findStackContaining(model.root, surfaceId);
  const target = findStackById(model.root, targetStackId);
  const surface = source?.tabs.find((tab) => isSurfaceMatch(tab, surfaceId));
  if (!surface || !target || source?.id === target.id) return setActiveSurface(model, surfaceId);
  const removed = removeSurface(model.root, surfaceId);
  if (!removed.node) return model;
  const targetAfter = findStackById(removed.node, targetStackId);
  if (!targetAfter) return model;
  const nextTarget: WorkspaceStack = {
    ...targetAfter,
    tabs: [...targetAfter.tabs, surface],
    activeId: surfaceId,
  };
  return { ...model, root: replaceStack(removed.node, targetStackId, nextTarget) };
}

export function setSplitRatio(
  model: WorkspaceModel,
  splitId: string,
  ratio: number,
): WorkspaceModel {
  const root = updateNode(model.root, (node) =>
    node.kind === 'split' && node.id === splitId ? { ...node, ratio: clampRatio(ratio) } : node,
  );
  return { ...model, root };
}

export function toggleMaximize(model: WorkspaceModel, surfaceId: string): WorkspaceModel {
  return {
    ...model,
    maximizedSurfaceId: model.maximizedSurfaceId === surfaceId ? undefined : surfaceId,
  };
}

export function updateSurface(
  model: WorkspaceModel,
  surfaceId: string,
  patch: Partial<WorkspaceSurface>,
): WorkspaceModel {
  const root = updateNode(model.root, (node) => {
    if (node.kind !== 'stack') return node;
    const tabIdx = node.tabs.findIndex((tab) => isSurfaceMatch(tab, surfaceId));
    if (tabIdx === -1) return node;
    const nextTabs = [...node.tabs];
    nextTabs[tabIdx] = { ...nextTabs[tabIdx], ...patch };
    return { ...node, tabs: nextTabs };
  });
  return { ...model, root };
}
