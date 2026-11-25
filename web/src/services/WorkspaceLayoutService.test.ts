import { describe, expect, it, beforeEach } from 'vitest';
import {
  selectFeatureLayout,
  WorkspaceLayoutService,
  WORKSPACE_LAYOUT_VERSION,
} from './WorkspaceLayoutService';
import { createWorkspace, openSurface, splitWithSurface } from '../workspace/model';
import { createPresentationState, setTerminalSplitRatio } from '../workspace/presentation';

// In Node environment vitest tests, provide mock localStorage on global/window
const store = new Map<string, string>();
const mockLocalStorage = {
  getItem: (key: string) => store.get(key) ?? null,
  setItem: (key: string, val: string) => {
    store.set(key, String(val));
  },
  removeItem: (key: string) => {
    store.delete(key);
  },
  clear: () => {
    store.clear();
  },
};

(globalThis as any).window = globalThis;
(globalThis as any).localStorage = mockLocalStorage;

describe('WorkspaceLayoutService', () => {
  const projectId = 'test-proj';
  let service: WorkspaceLayoutService;
  const overviewId = `project:${projectId}:overview`;
  const fallback = createWorkspace({
    id: overviewId,
    logicalKey: overviewId,
    viewId: `view:${overviewId}`,
    type: 'overview',
    title: 'Overview',
    data: { projectId },
  });

  beforeEach(() => {
    mockLocalStorage.clear();
    service = new WorkspaceLayoutService(projectId);
  });

  it('preserves optional tabs and split geometry across OFF/ON and core edits', () => {
    const original = service.load(fallback);
    original.model = splitWithSurface(
      original.model,
      overviewId,
      { id: 'legacy-mission', type: 'missions', title: 'Mission' },
      'horizontal',
    );
    const fullModel = JSON.parse(JSON.stringify(original.model));
    const core = selectFeatureLayout(original, 'core', fallback);
    expect(JSON.stringify(core.model)).not.toContain('legacy-mission');
    core.model = openSurface(core.model, {
      id: 'new-terminal',
      type: 'terminal',
      title: 'Terminal',
    });
    service.save(core);
    const reloaded = service.load(fallback);
    const full = selectFeatureLayout(reloaded, 'full', fallback);
    expect(full.model).toEqual(fullModel);
    service.save(full);
    const restoredCore = selectFeatureLayout(service.load(fallback), 'core', fallback);
    expect(JSON.stringify(restoredCore.model)).toContain('new-terminal');
    expect(JSON.stringify(restoredCore.model)).not.toContain('legacy-mission');
  });

  it('loads fallback when storage is empty', () => {
    const layout = service.load(fallback);
    expect(layout.version).toBe(WORKSPACE_LAYOUT_VERSION);
    expect(layout.workspaceId).toBe(projectId);
    expect(layout.model.root.kind).toBe('stack');
  });

  it('saves and reloads atomically with versioning', () => {
    const initial = service.load(fallback);
    service.save(initial);

    const reloaded = service.load(fallback);
    expect(reloaded.version).toBe(WORKSPACE_LAYOUT_VERSION);
    expect(reloaded.activeSurfaceId).toBe(overviewId);
    expect(reloaded.openSurfaceIds).toContain(`view:${overviewId}`);
  });

  it('recovers a previous good snapshot when the current layout is corrupt', () => {
    const first = service.load(fallback);
    service.save({ ...first, revision: 1 });
    service.save({ ...first, revision: 2 });
    expect(JSON.parse(mockLocalStorage.getItem(service.backupStorageKey) || '{}').revision).toBe(1);
    mockLocalStorage.setItem(service.storageKey, '{invalid');
    const restored = new WorkspaceLayoutService(projectId).load(fallback);
    expect(restored.revision).toBe(1);
    expect(restored.activeSurfaceId).toBe(overviewId);
    // Rewriting cannot replace the valid backup with a damaged current snapshot.
    service.save({ ...first, revision: 3 });
    expect(JSON.parse(mockLocalStorage.getItem(service.backupStorageKey) || '{}').revision).toBe(1);
  });

  it('rejects a foreign-project layout and uses the local recovery snapshot', () => {
    const first = service.load(fallback);
    service.save({ ...first, revision: 1 });
    service.save({ ...first, revision: 2 });
    mockLocalStorage.setItem(
      service.storageKey,
      JSON.stringify({
        ...first,
        workspaceId: 'foreign-project',
        revision: 99,
      }),
    );
    expect(service.load(fallback).revision).toBe(1);
  });

  it('recovers when current layout JSON is valid but its tabs are malformed', () => {
    const first = service.load(fallback);
    service.save({ ...first, revision: 1 });
    service.save({ ...first, revision: 2 });
    mockLocalStorage.setItem(
      service.storageKey,
      JSON.stringify({
        ...first,
        revision: 99,
        model: { version: 2, root: { kind: 'stack', id: 'bad', activeId: 'bad', tabs: [null] } },
      }),
    );
    expect(service.load(fallback).revision).toBe(1);
  });

  it('refuses to import a foreign-project layout without overwriting storage', () => {
    const initial = service.load(fallback);
    service.save(initial);
    const unchanged = mockLocalStorage.getItem(service.storageKey);
    expect(() =>
      service.import(JSON.stringify({ ...initial, workspaceId: 'foreign-project' }), fallback),
    ).toThrow(/different project/);
    expect(() =>
      service.import(
        JSON.stringify({
          ...initial,
          model: { version: 2, root: { kind: 'stack', id: 'bad', activeId: '', tabs: [null] } },
        }),
        fallback,
      ),
    ).toThrow(/invalid/);
    expect(mockLocalStorage.getItem(service.storageKey)).toBe(unchanged);
  });

  it('migrates from legacy v1/v2 storage keys smoothly', () => {
    mockLocalStorage.setItem(service.legacyModelKey, JSON.stringify(fallback));
    mockLocalStorage.setItem(
      service.legacyPresentationKey,
      JSON.stringify({ mode: 'MOSAIC', windows: {}, nextZ: 2 }),
    );

    const migrated = service.load(fallback);
    expect(migrated.version).toBe(WORKSPACE_LAYOUT_VERSION);
    expect(migrated.revision).toBe(1);
    expect(migrated.presentation.mode).toBe('MOSAIC');
  });

  it('migrates from v3 layout preserving custom revision', () => {
    mockLocalStorage.setItem(
      service.legacyV3StorageKey,
      JSON.stringify({
        version: 3,
        workspaceId: projectId,
        updatedAt: '2026-09-01T00:00:00.000Z',
        model: fallback,
        presentation: { mode: 'MOSAIC', windows: {}, nextZ: 2 },
        activeSurfaceId: overviewId,
        openSurfaceIds: [`view:${overviewId}`],
      }),
    );

    const migrated = service.load(fallback);
    expect(migrated.version).toBe(4);
    expect(migrated.revision).toBe(1);
  });

  it('exports and imports valid workbench configurations', () => {
    const initial = service.load(fallback);
    const exported = service.export(initial);
    expect(typeof exported).toBe('string');

    const imported = service.import(exported, fallback);
    expect(imported.version).toBe(WORKSPACE_LAYOUT_VERSION);
    expect(imported.workspaceId).toBe(projectId);
  });

  it('restores two-pane terminal proportions for the same project', () => {
    const initial = service.load(fallback);
    const presentation = setTerminalSplitRatio(
      {
        ...createPresentationState('SPLIT'),
        splitOrientation: 'horizontal',
        splitPrimaryViewId: 'view:pty-a',
        splitSecondaryViewId: 'view:pty-b',
      },
      0.65,
    );
    service.save({ ...initial, presentation });
    const restored = new WorkspaceLayoutService(projectId).load(fallback);
    expect(restored.presentation.mode).toBe('SPLIT');
    expect(restored.presentation.splitRatio).toBe(0.65);
    expect(restored.presentation.splitOrientation).toBe('horizontal');
    expect(restored.presentation.splitPrimaryViewId).toBe('view:pty-a');
    expect(restored.presentation.splitSecondaryViewId).toBe('view:pty-b');
    const other = new WorkspaceLayoutService('another-project').load(fallback);
    expect(other.presentation.mode).not.toBe('SPLIT');
  });

  it('resets layout to fallback defaults', () => {
    const initial = service.load(fallback);
    service.save(initial);

    const reset = service.reset(fallback);
    expect(reset.version).toBe(WORKSPACE_LAYOUT_VERSION);
    expect(reset.model.root.kind).toBe('stack');
  });

  it('keeps a named snapshot history per project', () => {
    const layout = service.load(fallback);
    const first = service.saveNamedSnapshot('before review', layout);
    const second = service.saveNamedSnapshot('after review', layout);

    const history = service.listNamedSnapshots();
    expect(history.map((entry) => entry.id)).toEqual([second.id, first.id]);
    expect(history[0].name).toBe('after review');
    expect(history[0].layout.workspaceId).toBe(projectId);
    expect(service.listNamedSnapshots()[0].layout.model.root.kind).toBe('stack');
  });

  it('restores a named snapshot as the current layout', () => {
    const layout = service.load(fallback);
    const snapshot = service.saveNamedSnapshot('before split', layout);
    service.reset(fallback);

    const restored = service.restoreNamedSnapshot(snapshot.id, fallback);
    expect(restored.workspaceId).toBe(projectId);
    expect(restored.model.root.kind).toBe('stack');
    expect(restored.activeSurfaceId).toBe(overviewId);
    const reloaded = new WorkspaceLayoutService(projectId).load(fallback);
    expect(reloaded.activeSurfaceId).toBe(overviewId);
  });

  it('rejects empty snapshot names and unknown snapshot ids', () => {
    const layout = service.load(fallback);
    expect(() => service.saveNamedSnapshot('   ', layout)).toThrow(/name is required/i);
    expect(() => service.restoreNamedSnapshot('snap_missing', fallback)).toThrow(
      /unknown workspace snapshot/i,
    );
  });

  it('deletes a named snapshot without touching the current layout', () => {
    const layout = service.load(fallback);
    service.save(layout);
    const snapshot = service.saveNamedSnapshot('keep me', layout);

    expect(service.deleteNamedSnapshot(snapshot.id)).toBe(true);
    expect(service.deleteNamedSnapshot(snapshot.id)).toBe(false);
    expect(service.listNamedSnapshots()).toHaveLength(0);
    expect(service.load(fallback).workspaceId).toBe(projectId);
  });

  it('ignores snapshots that belong to another project', () => {
    const layout = service.load(fallback);
    service.saveNamedSnapshot('local', layout);
    const foreign = {
      id: 'snap_foreign',
      name: 'foreign',
      createdAt: new Date().toISOString(),
      revision: 1,
      layout: { ...layout, workspaceId: 'another-project' },
    };
    mockLocalStorage.setItem(service.namedSnapshotsStorageKey, JSON.stringify([foreign]));

    expect(service.listNamedSnapshots()).toHaveLength(0);
  });
});
