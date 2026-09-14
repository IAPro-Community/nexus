import { describe, expect, it } from 'vitest';
import { resolveProjectSelection, resolveStaleProjectRoute } from './projectSelection';
import type { Project } from '../types';

const project = (id: string): Project => ({
  id,
  name: id,
  slug: id,
  canonical_path: `/tmp/${id}`,
  repo_remote: '',
  repo_url: '',
  default_branch: 'main',
  maestro_mode: 'ASSIST',
  resource_policy: 'BALANCED',
  default_isolation: 'project',
  settings: '',
  created_at: '',
  updated_at: '',
});

describe('project selection', () => {
  it('keeps current project when it still exists', () =>
    expect(resolveProjectSelection([project('a'), project('b')], 'b')?.id).toBe('b'));

  it('does not bounce to first project when the poll returns an empty list', () =>
    expect(resolveProjectSelection([], 'b')).toBeNull());

  it('falls back to first project when currentId is gone from a non-empty catalog', () =>
    expect(resolveProjectSelection([project('a')], 'missing')?.id).toBe('a'));

  it('falls back to first project when no currentId is set', () =>
    expect(resolveProjectSelection([project('a')], '')?.id).toBe('a'));

  it('returns null for empty list without currentId', () =>
    expect(resolveProjectSelection([], '')).toBeNull());
});

describe('stale project route', () => {
  it('keeps a route whose project still exists', () =>
    expect(resolveStaleProjectRoute([project('a'), project('b')], 'b')).toEqual({ kind: 'keep' }));

  it('waits while the catalog is still empty', () =>
    expect(resolveStaleProjectRoute([], 'missing')).toEqual({ kind: 'wait' }));

  it('redirects an unknown id onto the first catalog project', () =>
    expect(resolveStaleProjectRoute([project('a'), project('b')], 'missing')).toEqual({
      kind: 'redirect',
      projectId: 'a',
    }));

  it('keeps routes without a project id', () =>
    expect(resolveStaleProjectRoute([project('a')], '')).toEqual({ kind: 'keep' }));
});
