import { surfaceViewId, listSurfaces, type WorkspaceModel } from '../workspace/model';
import {
  isExperimentalComposerFlowSurface,
  withoutExperimentalComposerFlow,
} from '../app/composerFlowFlag';
import {
  deserializeWorkspace,
  isValidWorkspaceModel,
  serializeWorkspace,
  workspaceStorageKey,
} from '../workspace/state';
import {
  createPresentationState,
  migratePresentationState,
  type WorkspacePresentationState,
} from '../workspace/presentation';

export const WORKSPACE_LAYOUT_VERSION = 4;

export type LayoutFeatureProfile = 'core' | 'full';

export interface AlternateFeatureLayout {
  profile: LayoutFeatureProfile;
  model: WorkspaceModel;
  presentation: WorkspacePresentationState;
}

export interface PersistedWorkbenchLayout {
  version: typeof WORKSPACE_LAYOUT_VERSION;
  workspaceId: string;
  revision: number;
  updatedAt: string;
  model: WorkspaceModel;
  presentation: WorkspacePresentationState;
  activeSurfaceId?: string;
  openSurfaceIds: string[];
  featureProfile?: LayoutFeatureProfile;
  alternateFeatureLayout?: AlternateFeatureLayout;
}

/**
 * Named visual snapshot. It stores logical workspace state only: no process
 * handle, no runtime handle, so restoring it can never resurrect a dead
 * process or duplicate a runtime.
 */
export interface WorkspaceNamedSnapshot {
  id: string;
  name: string;
  createdAt: string;
  revision: number;
  layout: PersistedWorkbenchLayout;
}

const NAMED_SNAPSHOT_LIMIT = 20;

function validFeatureProfile(value: unknown): value is LayoutFeatureProfile {
  return value === 'core' || value === 'full';
}

function normalizeAlternate(value: unknown): AlternateFeatureLayout | undefined {
  if (!value || typeof value !== 'object') return undefined;
  const candidate = value as Partial<AlternateFeatureLayout>;
  if (!validFeatureProfile(candidate.profile) || !isValidWorkspaceModel(candidate.model)) {
    return undefined;
  }
  return {
    profile: candidate.profile,
    model: candidate.model,
    presentation: migratePresentationState(candidate.presentation),
  };
}

/**
 * Keep two layouts in one versioned envelope. A temporary feature opt-out is
 * not a close/delete action; both profiles can be restored independently.
 */
export function selectFeatureLayout(
  layout: PersistedWorkbenchLayout,
  profile: LayoutFeatureProfile,
  fallback: WorkspaceModel,
): PersistedWorkbenchLayout {
  const previous =
    layout.featureProfile ??
    (listSurfaces(layout.model.root).some((tab) => isExperimentalComposerFlowSurface(tab.type))
      ? 'full'
      : 'core');
  if (previous === profile) return { ...layout, featureProfile: profile };
  const other = layout.alternateFeatureLayout;
  const selected = other?.profile === profile ? other : undefined;
  const model =
    selected?.model ??
    (profile === 'core'
      ? withoutExperimentalComposerFlow(layout.model, listSurfaces(fallback.root)[0])
      : layout.model);
  return {
    ...layout,
    featureProfile: profile,
    model,
    presentation: selected?.presentation ?? layout.presentation,
    activeSurfaceId: model.root.kind === 'stack' ? model.root.activeId : undefined,
    openSurfaceIds: listSurfaces(model.root).map(surfaceViewId),
    alternateFeatureLayout: {
      profile: previous,
      model: layout.model,
      presentation: layout.presentation,
    },
  };
}

export class WorkspaceLayoutService {
  private readonly projectId: string;

  constructor(projectId: string) {
    this.projectId = projectId;
  }

  get storageKey(): string {
    return `iapro:nexus:workbench:${this.projectId}:v4`;
  }

  /** Last known-good local snapshot; does not contain a running process. */
  get backupStorageKey(): string {
    return `${this.storageKey}:last-good`;
  }

  get legacyV3StorageKey(): string {
    return `iapro:nexus:workbench:${this.projectId}:v3`;
  }

  get legacyModelKey(): string {
    return workspaceStorageKey(this.projectId);
  }

  get legacyPresentationKey(): string {
    return `iapro:nexus:workspace:${this.projectId}:presentation:v1`;
  }

  /**
   * Load workbench state: checks v4 unified key first, then falls back
   * to v3 / legacy model & presentation keys, migrating smoothly into v4 format.
   */
  load(fallbackModel: WorkspaceModel): PersistedWorkbenchLayout {
    if (typeof window !== 'undefined') {
      const read = (key: string): PersistedWorkbenchLayout | null => {
        try {
          const raw = window.localStorage.getItem(key);
          if (!raw) return null;
          const candidate = JSON.parse(raw) as Partial<PersistedWorkbenchLayout>;
          if (
            !candidate ||
            typeof candidate !== 'object' ||
            (candidate.workspaceId && candidate.workspaceId !== this.projectId) ||
            !isValidWorkspaceModel(candidate.model) ||
            ![1, 2, 3, WORKSPACE_LAYOUT_VERSION].includes(Number(candidate.version))
          ) {
            return null;
          }
          return this.migrateAndNormalize(candidate, fallbackModel);
        } catch {
          return null;
        }
      };
      // Corrupt or cross-project current layouts must not shadow the last good state.
      const restored =
        read(this.storageKey) ?? read(this.backupStorageKey) ?? read(this.legacyV3StorageKey);
      if (restored) return restored;
    }
    return this.loadLegacy(fallbackModel);
  }

  /**
   * Load and migrate from legacy v1/v2/v3 storage keys.
   */
  loadLegacy(fallbackModel: WorkspaceModel): PersistedWorkbenchLayout {
    let model = fallbackModel;
    let presentation = createPresentationState();

    if (typeof window !== 'undefined') {
      try {
        const rawModel = window.localStorage.getItem(this.legacyModelKey);
        if (rawModel) {
          model = deserializeWorkspace(rawModel, fallbackModel, this.projectId);
        }
      } catch {
        model = fallbackModel;
      }

      try {
        const rawPres = window.localStorage.getItem(this.legacyPresentationKey);
        if (rawPres) {
          presentation = migratePresentationState(JSON.parse(rawPres));
        }
      } catch {
        presentation = createPresentationState();
      }
    }

    return this.normalize({
      version: WORKSPACE_LAYOUT_VERSION,
      workspaceId: this.projectId,
      revision: 1,
      updatedAt: new Date().toISOString(),
      model,
      presentation,
      activeSurfaceId: model.root.kind === 'stack' ? model.root.activeId : undefined,
      openSurfaceIds: model.root.kind === 'stack' ? model.root.tabs.map(surfaceViewId) : [],
    });
  }

  /**
   * Save the complete layout atomically to localStorage.
   */
  save(layout: PersistedWorkbenchLayout): void {
    if (typeof window === 'undefined') return;
    try {
      const normalized = this.normalize(layout);
      const serialized = JSON.stringify(normalized);
      // Snapshot the preceding verified layout before overwriting it. If the
      // current value is corrupt, preserve the earlier recovery copy.
      const previous = window.localStorage.getItem(this.storageKey);
      if (previous) {
        try {
          const parsed = JSON.parse(previous) as Partial<PersistedWorkbenchLayout>;
          if (
            parsed.version === WORKSPACE_LAYOUT_VERSION &&
            parsed.workspaceId === this.projectId &&
            isValidWorkspaceModel(parsed.model)
          ) {
            window.localStorage.setItem(this.backupStorageKey, previous);
          }
        } catch {
          // Never overwrite last-good backup with corrupt storage contents.
        }
      }
      window.localStorage.setItem(this.storageKey, serialized);
      // Also update compatibility keys so older observers don't crash
      window.localStorage.setItem(this.legacyModelKey, serializeWorkspace(normalized.model));
      window.localStorage.setItem(
        this.legacyPresentationKey,
        JSON.stringify(normalized.presentation),
      );
    } catch {
      // Storage quota or private browsing error — safe ignore
    }
  }

  /** Named snapshot history for this project. */
  get namedSnapshotsStorageKey(): string {
    return `${this.storageKey}:snapshots`;
  }

  /**
   * List named snapshots, newest first. Foreign-project or corrupted entries
   * are ignored instead of shadowing the local history.
   */
  listNamedSnapshots(): WorkspaceNamedSnapshot[] {
    if (typeof window === 'undefined') return [];
    try {
      const raw = window.localStorage.getItem(this.namedSnapshotsStorageKey);
      if (!raw) return [];
      const parsed: unknown = JSON.parse(raw);
      if (!Array.isArray(parsed)) return [];
      return parsed.filter((entry) => this.isNamedSnapshot(entry));
    } catch {
      return [];
    }
  }

  /**
   * Store the current layout under a name so a previous visual state can be
   * restored later. The snapshot keeps logical state only.
   */
  saveNamedSnapshot(name: string, layout: PersistedWorkbenchLayout): WorkspaceNamedSnapshot {
    const trimmed = name.trim();
    if (!trimmed) {
      throw new Error('Snapshot name is required');
    }
    const normalized = this.normalize(layout);
    const snapshot: WorkspaceNamedSnapshot = {
      id: `snap_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`,
      name: trimmed,
      createdAt: new Date().toISOString(),
      revision: normalized.revision,
      layout: normalized,
    };
    if (typeof window !== 'undefined') {
      const history = [snapshot, ...this.listNamedSnapshots()].slice(0, NAMED_SNAPSHOT_LIMIT);
      try {
        window.localStorage.setItem(this.namedSnapshotsStorageKey, JSON.stringify(history));
      } catch {
        // Storage quota error — the caller still receives the snapshot.
      }
    }
    return snapshot;
  }

  /**
   * Restore a named snapshot as the current layout. Restoring only writes
   * logical state; the host reattaches existing surfaces by identity and
   * never starts a process from snapshot data.
   */
  restoreNamedSnapshot(id: string, fallbackModel: WorkspaceModel): PersistedWorkbenchLayout {
    const snapshot = this.listNamedSnapshots().find((entry) => entry.id === id);
    if (!snapshot) {
      throw new Error('Unknown workspace snapshot');
    }
    const restored = this.normalize({
      ...snapshot.layout,
      model: deserializeWorkspace(
        JSON.stringify(snapshot.layout.model),
        fallbackModel,
        this.projectId,
      ),
    });
    this.save(restored);
    return restored;
  }

  /** Remove a named snapshot. Returns false when the snapshot is unknown. */
  deleteNamedSnapshot(id: string): boolean {
    const history = this.listNamedSnapshots();
    const next = history.filter((entry) => entry.id !== id);
    if (next.length === history.length) return false;
    if (typeof window !== 'undefined') {
      try {
        window.localStorage.setItem(this.namedSnapshotsStorageKey, JSON.stringify(next));
      } catch {
        return false;
      }
    }
    return true;
  }

  private isNamedSnapshot(entry: unknown): entry is WorkspaceNamedSnapshot {
    if (!entry || typeof entry !== 'object') return false;
    const candidate = entry as Partial<WorkspaceNamedSnapshot>;
    if (
      typeof candidate.id !== 'string' ||
      !candidate.id ||
      typeof candidate.name !== 'string' ||
      !candidate.name ||
      typeof candidate.createdAt !== 'string' ||
      !candidate.layout
    ) {
      return false;
    }
    return (
      candidate.layout.workspaceId === this.projectId &&
      isValidWorkspaceModel(candidate.layout.model)
    );
  }

  /**
   * Validate and migrate previous versions into current v4 schema.
   */
  migrateAndNormalize(raw: unknown, fallbackModel: WorkspaceModel): PersistedWorkbenchLayout {
    if (!raw || typeof raw !== 'object') {
      return this.loadLegacy(fallbackModel);
    }

    const candidate = raw as Partial<PersistedWorkbenchLayout> & { version?: number };

    // Migrate from v1, v2 or v3 if encountered
    if (candidate.version !== WORKSPACE_LAYOUT_VERSION) {
      const model = deserializeWorkspace(
        candidate.model ? JSON.stringify(candidate.model) : null,
        fallbackModel,
        this.projectId,
      );
      const presentation = migratePresentationState(candidate.presentation);

      return this.normalize({
        version: WORKSPACE_LAYOUT_VERSION,
        workspaceId: this.projectId,
        revision: candidate.revision && candidate.revision > 0 ? candidate.revision : 1,
        updatedAt: new Date().toISOString(),
        model,
        presentation,
        activeSurfaceId: model.root.kind === 'stack' ? model.root.activeId : undefined,
        openSurfaceIds: model.root.kind === 'stack' ? model.root.tabs.map(surfaceViewId) : [],
      });
    }

    const model = deserializeWorkspace(
      candidate.model ? JSON.stringify(candidate.model) : null,
      fallbackModel,
      this.projectId,
    );
    const presentation = migratePresentationState(candidate.presentation);

    return this.normalize({
      version: WORKSPACE_LAYOUT_VERSION,
      workspaceId: candidate.workspaceId || this.projectId,
      revision: candidate.revision && candidate.revision > 0 ? candidate.revision : 1,
      updatedAt: candidate.updatedAt || new Date().toISOString(),
      model,
      presentation,
      activeSurfaceId: candidate.activeSurfaceId,
      openSurfaceIds: Array.isArray(candidate.openSurfaceIds) ? candidate.openSurfaceIds : [],
      featureProfile: validFeatureProfile(candidate.featureProfile)
        ? candidate.featureProfile
        : undefined,
      alternateFeatureLayout: normalizeAlternate(candidate.alternateFeatureLayout),
    });
  }

  /**
   * Normalize state ensuring canvas boundaries, valid tabs, and no orphaned active IDs.
   */
  normalize(layout: PersistedWorkbenchLayout): PersistedWorkbenchLayout {
    const model = layout.model;
    const presentation = layout.presentation;

    // Ensure model root is valid and has active tab
    let activeSurfaceId = layout.activeSurfaceId;
    let openSurfaceIds: string[] = [];

    const root = model.root;
    if (root.kind === 'stack') {
      openSurfaceIds = root.tabs.map(surfaceViewId);
      if (!root.tabs.some((t) => t.id === root.activeId)) {
        root.activeId = root.tabs[0]?.id || '';
      }
      activeSurfaceId = root.activeId;
    }

    return {
      version: WORKSPACE_LAYOUT_VERSION,
      workspaceId: this.projectId,
      revision: layout.revision && layout.revision > 0 ? layout.revision : 1,
      updatedAt: layout.updatedAt || new Date().toISOString(),
      model,
      presentation,
      activeSurfaceId,
      openSurfaceIds,
      featureProfile: validFeatureProfile(layout.featureProfile)
        ? layout.featureProfile
        : undefined,
      alternateFeatureLayout: normalizeAlternate(layout.alternateFeatureLayout),
    };
  }

  /**
   * Reset layout back to initial default model.
   */
  reset(fallbackModel: WorkspaceModel): PersistedWorkbenchLayout {
    const layout = this.normalize({
      version: WORKSPACE_LAYOUT_VERSION,
      workspaceId: this.projectId,
      revision: 1,
      updatedAt: new Date().toISOString(),
      model: fallbackModel,
      presentation: createPresentationState(),
      activeSurfaceId:
        fallbackModel.root.kind === 'stack' ? fallbackModel.root.activeId : undefined,
      openSurfaceIds:
        fallbackModel.root.kind === 'stack' ? fallbackModel.root.tabs.map(surfaceViewId) : [],
    });
    this.save(layout);
    return layout;
  }

  export(layout: PersistedWorkbenchLayout): string {
    return JSON.stringify(this.normalize(layout), null, 2);
  }

  import(json: string, fallbackModel: WorkspaceModel): PersistedWorkbenchLayout {
    const parsed: unknown = JSON.parse(json);
    if (!parsed || typeof parsed !== 'object') {
      throw new Error('Invalid workspace layout document');
    }
    const candidate = parsed as Partial<PersistedWorkbenchLayout>;
    if (
      !isValidWorkspaceModel(candidate.model) ||
      ![1, 2, 3, WORKSPACE_LAYOUT_VERSION].includes(Number(candidate.version)) ||
      (candidate.workspaceId && candidate.workspaceId !== this.projectId)
    ) {
      throw new Error('Layout is invalid or belongs to a different project');
    }
    const migrated = this.migrateAndNormalize(parsed, fallbackModel);
    this.save(migrated);
    return migrated;
  }
}
