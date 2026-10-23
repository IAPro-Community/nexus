import { describe, expect, it } from 'vitest';
import type { GitWorktree } from '../../types';
import { linkedGitWorktrees, linkedWorktreeForBranch } from './gitWorktrees';

const primary: GitWorktree = {
  path: '/repo',
  head: 'abc',
  branch: 'main',
  is_primary: true,
  detached: false,
  locked: false,
  prunable: false,
};
const linked: GitWorktree = {
  ...primary,
  path: '/repo-feature',
  branch: 'feature/a',
  is_primary: false,
};

describe('linked Git worktree visibility', () => {
  it('keeps the canonical checkout separate from linked worktrees', () => {
    expect(linkedGitWorktrees([primary, linked])).toEqual([linked]);
    expect(linkedGitWorktrees(undefined)).toEqual([]);
  });

  it('identifies branches already checked out outside the canonical checkout', () => {
    expect(linkedWorktreeForBranch([linked], 'feature/a')).toBe(linked);
    expect(linkedWorktreeForBranch([linked], 'main')).toBeUndefined();
  });

  it('does not reserve a branch for a detached worktree', () => {
    expect(
      linkedWorktreeForBranch([{ ...linked, detached: true, branch: '' }], 'feature/a'),
    ).toBeUndefined();
  });
});
