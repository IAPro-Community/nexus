import { describe, expect, it } from 'vitest';
import {
  ARRANGE_MENU_PRESETS,
  WORKSPACE_RECIPES,
  arrangeByPreset,
  resolveArrangePreset,
} from './arrangePresets';

describe('arrangeByPreset', () => {
  const bounds = { x: 8, y: 8, width: 1200, height: 800 };

  it('exposes exactly five canonical menu presets in Portuguese', () => {
    expect(ARRANGE_MENU_PRESETS.map((p) => p.id)).toEqual([
      'automatic',
      'two-columns',
      'three-columns',
      'terminal-focus',
      'focus-mode',
    ]);
    expect(ARRANGE_MENU_PRESETS.map((p) => p.label)).toEqual([
      'Automático',
      'Duas colunas',
      'Três colunas',
      'Principal + fila',
      'Só a ativa',
    ]);
  });

  it('offers task recipes without creating sessions or inventing new layouts', () => {
    expect(WORKSPACE_RECIPES.map((recipe) => recipe.id)).toEqual([
      'frontend',
      'backend',
      'review',
      'debug',
      'release',
    ]);
    const ids = ['agent-codex', 'agent-claude', 'browser', 'review'];
    for (const recipe of WORKSPACE_RECIPES) {
      expect(ARRANGE_MENU_PRESETS.some((preset) => preset.id === recipe.preset)).toBe(true);
      const arranged = arrangeByPreset(recipe.preset, bounds, ids);
      expect(arranged.length).toBeGreaterThan(0);
      expect(arranged.every((tile) => ids.includes(tile.viewId))).toBe(true);
      expect(new Set(arranged.map((tile) => tile.viewId)).size).toBe(arranged.length);
    }
  });

  it('resolves legacy aliases without breaking arrangeByPreset', () => {
    expect(resolveArrangePreset('agents')).toBe('automatic');
    expect(resolveArrangePreset('restore-default')).toBe('automatic');
    expect(resolveArrangePreset('terminal-chat')).toBe('terminal-focus');
    expect(resolveArrangePreset('terminal-flow')).toBe('terminal-focus');

    const viaAlias = arrangeByPreset('terminal-chat', bounds, ['a', 'b', 'c'], 'b');
    const viaCanonical = arrangeByPreset('terminal-focus', bounds, ['a', 'b', 'c'], 'b');
    expect(viaAlias).toEqual(viaCanonical);
  });

  it('handles automatic preset', () => {
    const tiles = arrangeByPreset('automatic', bounds, ['term-1', 'term-2']);
    expect(tiles).toHaveLength(2);
    expect(tiles[0].viewId).toBe('term-1');
  });

  it('prioritizes activeViewId in terminal-focus', () => {
    const tiles = arrangeByPreset(
      'terminal-focus',
      bounds,
      ['term-1', 'term-2', 'term-3'],
      'term-2',
    );
    expect(tiles[0].viewId).toBe('term-2');
    expect(tiles[0].width).toBeGreaterThan(tiles[1].width);
  });

  it('arranges two columns cleanly', () => {
    const tiles = arrangeByPreset('two-columns', bounds, ['a', 'b', 'c', 'd']);
    expect(tiles).toHaveLength(4);
    const leftCol = tiles.filter((t) => t.x === bounds.x);
    const rightCol = tiles.filter((t) => t.x > bounds.x);
    expect(leftCol).toHaveLength(2);
    expect(rightCol).toHaveLength(2);
  });

  it('arranges three columns cleanly', () => {
    const tiles = arrangeByPreset('three-columns', bounds, ['a', 'b', 'c']);
    expect(tiles).toHaveLength(3);
    const xPositions = new Set(tiles.map((t) => t.x));
    expect(xPositions.size).toBe(3);
  });

  it('falls back to two columns when three-columns has fewer than 3 windows', () => {
    const tiles = arrangeByPreset('three-columns', bounds, ['a', 'b']);
    expect(tiles).toHaveLength(2);
    const xPositions = new Set(tiles.map((t) => t.x));
    expect(xPositions.size).toBe(2);
  });

  it('focus-mode gives single active tile full canvas', () => {
    const tiles = arrangeByPreset('focus-mode', bounds, ['a', 'b'], 'b');
    expect(tiles).toHaveLength(1);
    expect(tiles[0].viewId).toBe('b');
    expect(tiles[0].width).toBe(bounds.width);
    expect(tiles[0].height).toBe(bounds.height);
  });

  it('zero overlap between any arranged tiles', () => {
    const presets = [
      'automatic',
      'terminal-focus',
      'two-columns',
      'three-columns',
      'terminal-chat',
    ] as const;
    const views = ['v1', 'v2', 'v3', 'v4'];

    for (const p of presets) {
      const tiles = arrangeByPreset(p, bounds, views);
      for (let i = 0; i < tiles.length; i++) {
        for (let j = i + 1; j < tiles.length; j++) {
          const t1 = tiles[i];
          const t2 = tiles[j];
          const overlapX = t1.x < t2.x + t2.width - 4 && t1.x + t1.width - 4 > t2.x;
          const overlapY = t1.y < t2.y + t2.height - 4 && t1.y + t1.height - 4 > t2.y;
          expect(overlapX && overlapY).toBe(false);
        }
      }
    }
  });
});
