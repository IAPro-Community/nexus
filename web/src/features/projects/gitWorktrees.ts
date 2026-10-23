import type { GitWorktree } from '../../types';

export const linkedGitWorktrees = (worktrees: readonly GitWorktree[] | undefined): GitWorktree[] =>
  (worktrees ?? []).filter((worktree) => !worktree.is_primary);

// A branch checked out in another worktree cannot be safely checked out in
// this project's canonical checkout. No branch-switching action is performed.
export const linkedWorktreeForBranch = (
  worktrees: readonly GitWorktree[],
  branch: string,
): GitWorktree | undefined =>
  worktrees.find((worktree) => !worktree.detached && worktree.branch === branch);
