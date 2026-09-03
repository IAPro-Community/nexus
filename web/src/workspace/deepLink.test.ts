import { describe, expect, it, vi } from 'vitest';
import { applyDeepLink, parseDeepLink, generateDeepLink } from './deepLink';

describe('parseDeepLink', () => {
  it('parses a project-only deep link', () => {
    const result = parseDeepLink('/projects/proj-1');
    expect(result).toEqual({ projectId: 'proj-1', focus: undefined });
  });

  it('parses a deep link with agent focus', () => {
    const result = parseDeepLink('/projects/proj-1?view=agent:agt-1');
    expect(result).toEqual({
      projectId: 'proj-1',
      focus: { type: 'agent', id: 'agt-1' },
    });
  });

  it('parses a deep link with flow-run focus', () => {
    const result = parseDeepLink('/projects/proj-1?view=flow-run:run-1');
    expect(result).toEqual({
      projectId: 'proj-1',
      focus: { type: 'flow-run', id: 'run-1' },
    });
  });

  it('returns null for non-project paths', () => {
    expect(parseDeepLink('/settings')).toBeNull();
    expect(parseDeepLink('/')).toBeNull();
    expect(parseDeepLink('/agents')).toBeNull();
  });

  it('returns null for malformed URLs', () => {
    expect(parseDeepLink('not-a-url')).toBeNull();
  });
});

describe('generateDeepLink', () => {
  it('generates a project-only link', () => {
    const link = generateDeepLink('proj-1');
    expect(link).toBe('/projects/proj-1');
  });

  it('generates a link with focus', () => {
    const link = generateDeepLink('proj-1', { type: 'agent', id: 'agt-1' });
    expect(link).toBe('/projects/proj-1?view=agent%3Aagt-1');
  });
});

describe('applyDeepLink', () => {
  it('reconstructs a missing logical view without creating a duplicate', () => {
    const openView = vi.fn();
    const focusView = vi.fn(() => false);
    applyDeepLink({ projectId: 'proj-1', focus: { type: 'agent', id: 'agent-1' } }, openView, focusView);
    expect(focusView).toHaveBeenCalledWith('agent-1');
    expect(openView).toHaveBeenCalledWith('proj-1', 'agent-1');
  });
});
