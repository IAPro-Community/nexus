import { surfaceViewId, type WorkspaceModel } from '../workspace/model';
import { deserializeWorkspace, serializeWorkspace, workspaceStorageKey } from '../workspace/state';
import {
  createPresentationState,
  migratePresentationState,
  type WorkspacePresentationState,
} from '../workspace/presentation';

export const WORKSPACE_LAYOUT_VERSION = 4;

export interface PersistedWorkbenchLayout {
  version: typeof WORKSPACE_LAYOUT_VERSION;
  workspaceId: string;
  revision: number;
  updatedAt: string;
  model: WorkspaceModel;
  presentation: WorkspacePresentationState;
  activeSurfaceId?: string;
  openSurfaceIds: string[];
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
            !candidate.model?.root ||
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
        read(this.storageKey) ??
        read(this.backupStorageKey) ??
        read(this.legacyV3StorageKey);
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
            parsed.model?.root
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
    const parsed = JSON.parse(json);
    const migrated = this.migrateAndNormalize(parsed, fallbackModel);
    this.save(migrated);
    return migrated;
  }
}
