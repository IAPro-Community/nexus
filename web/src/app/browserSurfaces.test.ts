import { describe, expect, it } from 'vitest';
import {
  browserSurface,
  isPtySurface,
  isWorkspaceWindowSurface,
  projectShellSurface,
} from './surfaces';
import { createPresentationState, syncDesktopWindows } from '../workspace/presentation';

describe('Browser inside terminal workspace', () => {
  it('creates unique browser windows without classifying them as PTYs', () => {
    const first = browserSurface('p1');
    const second = browserSurface('p1');
    expect(first.id).not.toBe(second.id);
    expect(first.type).toBe('browser');
    expect(isPtySurface(first)).toBe(false);
    expect(isWorkspaceWindowSurface(first)).toBe(true);
  });
  it('coexists with a terminal in the desktop window registry', () => {
    const browser = browserSurface('p1');
    const shell = projectShellSurface('p1', 'runtime-one');
    const state = syncDesktopWindows(createPresentationState('DESKTOP'), [browser, shell]);
    expect(Object.keys(state.windows)).toHaveLength(2);
    expect(state.windows[browser.viewId!]).toBeDefined();
    expect(state.windows[shell.viewId!]).toBeDefined();
  });
});
