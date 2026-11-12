import type { ChangeCollision, ChangeCollisionsReport } from '../../types';

/**
 * Presentation model for the read-only change-collision detector
 * (capability change-collision-detector). The detector never blocks a
 * merge; this model only shapes what the UI shows, keeping the
 * detector's uncertainty explicit instead of converting it into
 * false confidence.
 */

export interface ChangeCollisionsSummary {
  /** Worktrees actually scanned (with uncommitted changes). */
  scannedWorktrees: number;
  /** Files changed by more than one worktree. */
  total: number;
  /** Intersecting hunks: merge will very likely conflict. */
  hunkOverlaps: number;
  /** Same file, disjoint hunks: candidate only. */
  sameFileCandidates: number;
}

export const summarizeCollisions = (report: ChangeCollisionsReport): ChangeCollisionsSummary => {
  const collisions = report.collisions || [];
  return {
    scannedWorktrees: (report.worktrees || []).length,
    total: collisions.length,
    hunkOverlaps: collisions.filter((c) => c.confidence === 'hunk-overlap').length,
    sameFileCandidates: collisions.filter((c) => c.confidence === 'same-file').length,
  };
};

export const hasPartialCoverage = (report: ChangeCollisionsReport): boolean =>
  (report.scan_errors || []).length > 0;

/** Worktree label for the UI: branch when known, path as fallback. */
export const worktreeLabel = (path: string, branch?: string): string => branch || path;

/** i18n key for the confidence label shown as the collision badge. */
export const confidenceLabelKey = (collision: ChangeCollision): string =>
  collision.confidence === 'hunk-overlap'
    ? 'agents.collisions.conflictLikely'
    : 'agents.collisions.uncertainCandidate';

/** Sort collisions so the most certain evidence comes first. */
export const sortCollisions = (collisions: ChangeCollision[]): ChangeCollision[] =>
  [...collisions].sort((a, b) => {
    if (a.confidence !== b.confidence) {
      return a.confidence === 'hunk-overlap' ? -1 : 1;
    }
    if (a.file !== b.file) {
      return a.file < b.file ? -1 : 1;
    }
    return a.worktrees.join() < b.worktrees.join() ? -1 : 1;
  });
