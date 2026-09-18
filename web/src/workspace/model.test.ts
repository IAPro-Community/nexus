import { describe, expect, it } from 'vitest';
import {
  clampRatio,
  closeSurface,
  createWorkspace,
  ensureSurface,
  findStackContaining,
  listSurfaces,
  moveSurface,
  openSurface,
  setActiveSurface,
  flattenToSingleStack,
  splitWithSurface,
  surfaceLogicalKey,
  surfaceViewId,
  type WorkspaceSurface,
} from './model';

const surface = (id: string): WorkspaceSurface => ({ id, type: 'test', title: id });

describe('workspace model', () => {
  it('opens a surface once and focuses it when reopened', () => {
    let ws = createWorkspace(surface('home'));
    ws = openSurface(ws, surface('terminal:a'));
    ws = openSurface(ws, surface('terminal:a'));
    const stack = findStackContaining(ws.root, 'terminal:a');
    expect(stack?.tabs.map((tab) => tab.id)).toEqual(['home', 'terminal:a']);
    expect(stack?.activeId).toBe('terminal:a');
  });

  it('removes duplicate logical tabs before activating a surface', () => {
    const duplicate = {
      ...surface('project:p1:settings-copy'),
      logicalKey: 'project:p1:settings',
      type: 'settings',
    };
    let ws = createWorkspace({
      ...surface('project:p1:overview'),
      logicalKey: 'project:p1:overview',
      type: 'overview',
    });
    ws = openSurface(ws, duplicate);
    ws = openSurface(ws, { ...duplicate, id: 'project:p1:settings' });

    expect(ws.root.kind).toBe('stack');
    if (ws.root.kind === 'stack') {
      expect(ws.root.tabs.filter((tab) => tab.logicalKey === 'project:p1:settings')).toHaveLength(
        1,
      );
      expect(ws.root.activeId).toBe('project:p1:settings-copy');
    }
  });

  it('repairs legacy duplicates that have different ids but the same data identity', () => {
    const ws = {
      version: 2 as const,
      root: {
        kind: 'split' as const,
        id: 'split-1',
        direction: 'horizontal' as const,
        ratio: 0.5,
        first: {
          kind: 'stack' as const,
          id: 'stack-1',
          activeId: 'terminal-copy-a',
          tabs: [
            {
              ...surface('terminal-copy-a'),
              type: 'terminal',
              data: { agentId: 'agent-1' },
            },
          ],
        },
        second: {
          kind: 'stack' as const,
          id: 'stack-2',
          activeId: 'terminal-copy-b',
          tabs: [
            {
              ...surface('terminal-copy-b'),
              type: 'terminal',
              data: { agentId: 'agent-1' },
            },
          ],
        },
      },
    };

    const repaired = openSurface(ws, surface('home'));
    expect(listSurfaces(repaired.root)).toHaveLength(2);
    expect(
      listSurfaces(repaired.root).filter((tab) => tab.data?.agentId === 'agent-1'),
    ).toHaveLength(1);
  });

  it('splits an existing stack with a new surface', () => {
    const ws = splitWithSurface(
      createWorkspace(surface('home')),
      'home',
      surface('terminal:a'),
      'horizontal',
    );
    expect(ws.root.kind).toBe('split');
    if (ws.root.kind !== 'split') return;
    expect(ws.root.direction).toBe('horizontal');
    expect(findStackContaining(ws.root, 'terminal:a')?.activeId).toBe('terminal:a');
  });

  it('moves a surface between stacks without duplication', () => {
    let ws = splitWithSurface(
      createWorkspace(surface('home')),
      'home',
      surface('terminal:a'),
      'horizontal',
    );
    if (ws.root.kind !== 'split' || ws.root.second.kind !== 'stack')
      throw new Error('unexpected model');
    ws = openSurface(ws, surface('plan'), ws.root.second.id);
    const targetStack =
      ws.root.kind === 'split' && ws.root.first.kind === 'stack' ? ws.root.first.id : '';
    ws = moveSurface(ws, 'plan', targetStack);
    expect(findStackContaining(ws.root, 'plan')?.id).toBe(targetStack);
    const all = JSON.stringify(ws.root).match(/"id":"plan"/g) ?? [];
    expect(all).toHaveLength(1);
  });

  it('closes a tab and collapses an empty split', () => {
    let ws = splitWithSurface(
      createWorkspace(surface('home')),
      'home',
      surface('terminal:a'),
      'vertical',
    );
    ws = closeSurface(ws, 'terminal:a');
    expect(ws.root.kind).toBe('stack');
    expect(findStackContaining(ws.root, 'home')?.activeId).toBe('home');
  });

  it('keeps non-closable surfaces', () => {
    const ws = closeSurface(createWorkspace({ ...surface('home'), closable: false }), 'home');
    expect(findStackContaining(ws.root, 'home')).not.toBeNull();
  });

  it('activates a surface without changing the tree', () => {
    let ws = openSurface(createWorkspace(surface('home')), surface('plan'));
    ws = setActiveSurface(ws, 'home');
    expect(findStackContaining(ws.root, 'home')?.activeId).toBe('home');
  });

  it('uses logical identity to focus an existing view instead of duplicating it', () => {
    let ws = createWorkspace({
      ...surface('overview-1'),
      logicalKey: 'project:p1:overview',
      viewId: 'view:overview',
    });
    ws = openSurface(ws, {
      ...surface('overview-2'),
      logicalKey: 'project:p1:overview',
      viewId: 'view:overview-new',
    });
    const stack = findStackContaining(ws.root, 'view:overview');
    expect(stack?.tabs).toHaveLength(1);
    expect(surfaceLogicalKey(stack!.tabs[0])).toBe('project:p1:overview');
    expect(surfaceViewId(stack!.tabs[0])).toBe('view:overview');
  });

  it('ensures a surface without stealing the active tab', () => {
    let ws = createWorkspace({ ...surface('overview'), type: 'overview', closable: false });
    ws = ensureSurface(ws, { ...surface('terminals'), type: 'terminals', closable: false });
    const stack = findStackContaining(ws.root, 'overview');
    expect(stack?.activeId).toBe('overview');
    expect(listSurfaces(ws.root).some((tab) => tab.id === 'terminals')).toBe(true);
    const before = stack?.activeId;
    ws = ensureSurface(ws, { ...surface('terminals'), type: 'terminals', closable: false });
    expect(findStackContaining(ws.root, 'overview')?.activeId).toBe(before);
  });

  it('pins an existing Composer tab without stealing focus', () => {
    let ws = createWorkspace({ ...surface('overview'), type: 'overview', closable: false });
    ws = openSurface(ws, { ...surface('work'), type: 'work', closable: true });
    const before = findStackContaining(ws.root, 'overview')?.activeId;
    ws = ensureSurface(ws, { ...surface('work'), type: 'work', closable: false });
    const work = listSurfaces(ws.root).find((tab) => tab.id === 'work');
    expect(work?.closable).toBe(false);
    expect(findStackContaining(ws.root, 'overview')?.activeId).toBe(before);
  });

  it('finds a migrated surface by legacy id and stable view id', () => {
    const ws = createWorkspace({
      id: 'legacy-agent',
      viewId: 'view:agent:1',
      logicalKey: 'agent:1:terminal',
      type: 'terminal',
      title: 'Agent',
    });
    expect(findStackContaining(ws.root, 'legacy-agent')).not.toBeNull();
    expect(findStackContaining(ws.root, 'view:agent:1')).not.toBeNull();
    expect(findStackContaining(ws.root, 'agent:1:terminal')).not.toBeNull();
  });

  it('clamps split ratios to usable bounds', () => {
    expect(clampRatio(0.01)).toBe(0.2);
    expect(clampRatio(0.5)).toBe(0.5);
    expect(clampRatio(0.99)).toBe(0.8);
  });

  it('flattens split layouts into a single stack and drops maximize', () => {
    let ws = splitWithSurface(
      createWorkspace(surface('home')),
      'home',
      surface('terminal:a'),
      'horizontal',
    );
    ws = { ...ws, maximizedSurfaceId: 'home' };
    const flat = flattenToSingleStack(ws);
    expect(flat.root.kind).toBe('stack');
    expect(flat.maximizedSurfaceId).toBeUndefined();
    expect(
      listSurfaces(flat.root)
        .map((tab) => tab.id)
        .sort(),
    ).toEqual(['home', 'terminal:a']);
  });
});
