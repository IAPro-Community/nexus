import {
  type ArrangeBounds,
  type TileRect,
  arrangeSmart,
  ARRANGE_GAP,
  ARRANGE_MIN_W,
  ARRANGE_MIN_H,
} from './arrange';

/** Canonical UI presets (Portuguese menu). Legacy names remain accepted as aliases. */
export type ArrangeMenuPreset =
  'automatic' | 'two-columns' | 'three-columns' | 'terminal-focus' | 'focus-mode';

export type ArrangePresetName =
  ArrangeMenuPreset | 'terminal-chat' | 'terminal-flow' | 'agents' | 'restore-default';

export const ARRANGE_MENU_PRESETS: ReadonlyArray<{
  id: ArrangeMenuPreset;
  label: string;
  hint: string;
}> = [
  { id: 'automatic', label: 'Automático', hint: 'Smart' },
  { id: 'two-columns', label: 'Duas colunas', hint: '50/50' },
  { id: 'three-columns', label: 'Três colunas', hint: '33/33/33' },
  { id: 'terminal-focus', label: 'Principal + fila', hint: '68/32' },
  { id: 'focus-mode', label: 'Só a ativa', hint: 'Tela cheia' },
];

/** User-facing workspace recipes reuse existing layout primitives.
 * They never spawn Agents, terminals or browsers. */
export const WORKSPACE_RECIPES = [
  { id: 'frontend', preset: 'two-columns', labelKey: 'workspace.recipeFrontend', hintKey: 'workspace.recipeFrontendHint' },
  { id: 'backend', preset: 'terminal-focus', labelKey: 'workspace.recipeBackend', hintKey: 'workspace.recipeBackendHint' },
  { id: 'review', preset: 'three-columns', labelKey: 'workspace.recipeReview', hintKey: 'workspace.recipeReviewHint' },
  { id: 'debug', preset: 'two-columns', labelKey: 'workspace.recipeDebug', hintKey: 'workspace.recipeDebugHint' },
  { id: 'release', preset: 'automatic', labelKey: 'workspace.recipeRelease', hintKey: 'workspace.recipeReleaseHint' },
] as const satisfies ReadonlyArray<{
  id: string;
  preset: ArrangeMenuPreset;
  labelKey: string;
  hintKey: string;
}>;

const GAP = ARRANGE_GAP;
const MIN_W = ARRANGE_MIN_W;
const MIN_H = ARRANGE_MIN_H;

function roundRect(rect: Omit<TileRect, 'viewId'> & { viewId: string }): TileRect {
  return {
    viewId: rect.viewId,
    x: Math.round(rect.x),
    y: Math.round(rect.y),
    width: Math.max(MIN_W, Math.round(rect.width)),
    height: Math.max(MIN_H, Math.round(rect.height)),
  };
}

/** Resolve legacy / alias preset names to the five canonical layouts. */
export function resolveArrangePreset(preset: ArrangePresetName): ArrangeMenuPreset {
  switch (preset) {
    case 'agents':
    case 'restore-default':
      return 'automatic';
    case 'terminal-chat':
    case 'terminal-flow':
      return 'terminal-focus';
    case 'automatic':
    case 'two-columns':
    case 'three-columns':
    case 'terminal-focus':
    case 'focus-mode':
      return preset;
    default:
      return 'automatic';
  }
}

/**
 * Arrange layout presets for the Workbench.
 * Guarantees zero involuntary overlap and respect for minimum pane boundaries.
 */
export function arrangeByPreset(
  preset: ArrangePresetName,
  bounds: ArrangeBounds,
  viewIds: string[],
  activeViewId?: string,
): TileRect[] {
  const ids = viewIds.filter(Boolean);
  if (ids.length === 0) return [];

  // If bounds are constrained below minimums, stack at min dimensions
  if (bounds.width < MIN_W || bounds.height < MIN_H) {
    return ids.map((viewId) =>
      roundRect({
        viewId,
        x: bounds.x,
        y: bounds.y,
        width: Math.max(MIN_W, bounds.width),
        height: Math.max(MIN_H, bounds.height),
      }),
    );
  }

  // Ensure active pane is prioritized first if specified
  let prioritized = [...ids];
  if (activeViewId && prioritized.includes(activeViewId)) {
    prioritized = [activeViewId, ...prioritized.filter((id) => id !== activeViewId)];
  }

  const resolved = resolveArrangePreset(preset);

  switch (resolved) {
    case 'automatic':
      return arrangeSmart(bounds, prioritized);

    case 'focus-mode': {
      return [
        roundRect({
          viewId: prioritized[0],
          x: bounds.x,
          y: bounds.y,
          width: bounds.width,
          height: bounds.height,
        }),
      ];
    }

    case 'terminal-focus': {
      if (prioritized.length === 1) return arrangeSmart(bounds, prioritized);
      const mainW = Math.max(MIN_W, Math.floor((bounds.width - GAP) * 0.68));
      const sideW = Math.max(MIN_W, bounds.width - mainW - GAP);

      const main = roundRect({
        viewId: prioritized[0],
        x: bounds.x,
        y: bounds.y,
        width: mainW,
        height: bounds.height,
      });

      const sideIds = prioritized.slice(1);
      const sideTotalGap = GAP * (sideIds.length - 1);
      const sideH = (bounds.height - sideTotalGap) / sideIds.length;

      const sideTiles = sideIds.map((viewId, idx) =>
        roundRect({
          viewId,
          x: bounds.x + mainW + GAP,
          y: bounds.y + idx * (sideH + GAP),
          width: sideW,
          height: sideH,
        }),
      );

      return [main, ...sideTiles];
    }

    case 'two-columns': {
      if (prioritized.length === 1) return arrangeSmart(bounds, prioritized);
      const halfW = (bounds.width - GAP) / 2;
      const col1 = prioritized.filter((_, i) => i % 2 === 0);
      const col2 = prioritized.filter((_, i) => i % 2 === 1);

      const col1Gap = GAP * (col1.length - 1);
      const col1H = (bounds.height - col1Gap) / col1.length;
      const col2Gap = GAP * (col2.length - 1);
      const col2H = (bounds.height - col2Gap) / col2.length;

      const tiles1 = col1.map((viewId, idx) =>
        roundRect({
          viewId,
          x: bounds.x,
          y: bounds.y + idx * (col1H + GAP),
          width: halfW,
          height: col1H,
        }),
      );

      const tiles2 = col2.map((viewId, idx) =>
        roundRect({
          viewId,
          x: bounds.x + halfW + GAP,
          y: bounds.y + idx * (col2H + GAP),
          width: halfW,
          height: col2H,
        }),
      );

      return [...tiles1, ...tiles2];
    }

    case 'three-columns': {
      if (prioritized.length < 3) {
        return arrangeByPreset('two-columns', bounds, prioritized, activeViewId);
      }
      const numCols = 3;
      const colW = (bounds.width - GAP * 2) / numCols;
      const cols: string[][] = [[], [], []];
      prioritized.forEach((id, idx) => cols[idx % numCols].push(id));

      const tiles: TileRect[] = [];
      cols.forEach((colIds, colIdx) => {
        const colGap = GAP * (colIds.length - 1);
        const colH = (bounds.height - colGap) / colIds.length;
        colIds.forEach((viewId, rowIdx) => {
          tiles.push(
            roundRect({
              viewId,
              x: bounds.x + colIdx * (colW + GAP),
              y: bounds.y + rowIdx * (colH + GAP),
              width: colW,
              height: colH,
            }),
          );
        });
      });

      return tiles;
    }

    default:
      return arrangeSmart(bounds, prioritized);
  }
}
