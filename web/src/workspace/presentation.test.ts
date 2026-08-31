import { describe, expect, it } from 'vitest';
import { createWorkspace } from './model';
import {
  createPresentationState,
  ensureDesktopWindow,
  focusDesktopWindow,
  moveDesktopWindow,
  setPresentationMode,
  toggleDesktopMaximize,
} from './presentation';

describe('workspace presentation', () => {
  it('switches presentation without changing the logical workspace', () => {
    const surface = { id: 'agent:a1', viewId: 'view:a1', logicalKey: 'session:a1', type: 'agent-terminal', title: 'Agent' };
    const workspace = createWorkspace(surface);
    const desktop = setPresentationMode(createPresentationState(workspace), 'DESKTOP');
    const windowState = ensureDesktopWindow(desktop, surface.viewId!);
    const tabs = setPresentationMode(focusDesktopWindow(moveDesktopWindow(windowState, surface.viewId!, 42, 24), surface.viewId!), 'TABS');
    expect(tabs.mode).toBe('TABS');
    expect(workspace.root).toEqual(workspace.root);
    expect(tabs.windows[surface.viewId!].viewId).toBe(surface.viewId);
    expect(tabs.windows[surface.viewId!].x).toBe(42);
  });

  it('maximizes one view without cloning it', () => {
    const state = ensureDesktopWindow(createPresentationState(createWorkspace({ id: 'a', type: 'overview', title: 'A' })), 'view:a');
    const maximized = toggleDesktopMaximize(state, 'view:a');
    expect(maximized.windows['view:a'].maximized).toBe(true);
    expect(Object.keys(maximized.windows)).toEqual(['view:a']);
  });
});
