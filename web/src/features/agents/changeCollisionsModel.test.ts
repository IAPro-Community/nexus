import { describe, expect, it } from 'vitest';
import type { ChangeCollisionsReport } from '../../types';
import {
  confidenceLabelKey,
  hasPartialCoverage,
  sortCollisions,
  summarizeCollisions,
  worktreeLabel,
} from './changeCollisionsModel';

const report = (overrides: Partial<ChangeCollisionsReport> = {}): ChangeCollisionsReport => ({
  project_id: 'p1',
  scanned_at: '2026-11-10T00:00:00Z',
  worktrees: [
    { path: '/wt/a', branch: 'main', changed_files: ['shared.go'] },
    { path: '/wt/b', branch: 'feature/x', changed_files: ['shared.go'] },
  ],
  collisions: [
    {
      file: 'shared.go',
      worktrees: ['/wt/a', '/wt/b'],
      confidence: 'same-file',
      uncertain: true,
      reason: 'disjoint hunks',
    },
  ],
  uncertain_count: 1,
  scan_errors: [],
  ...overrides,
});

describe('summarizeCollisions', () => {
  it('counts overlaps and same-file candidates separately', () => {
    const summary = summarizeCollisions(
      report({
        collisions: [
          {
            file: 'a.go',
            worktrees: ['/wt/a', '/wt/b'],
            confidence: 'hunk-overlap',
            uncertain: false,
            reason: 'r',
          },
          {
            file: 'b.go',
            worktrees: ['/wt/a', '/wt/b'],
            confidence: 'same-file',
            uncertain: true,
            reason: 'r',
          },
        ],
      }),
    );
    expect(summary).toEqual({
      scannedWorktrees: 2,
      total: 2,
      hunkOverlaps: 1,
      sameFileCandidates: 1,
    });
  });

  it('never crashes on null arrays from the wire', () => {
    const summary = summarizeCollisions(
      report({ collisions: null as unknown as [], worktrees: undefined as unknown as [] }),
    );
    expect(summary.total).toBe(0);
    expect(summary.scannedWorktrees).toBe(0);
  });
});

describe('hasPartialCoverage', () => {
  it('reports scan errors honestly', () => {
    expect(hasPartialCoverage(report())).toBe(false);
    expect(hasPartialCoverage(report({ scan_errors: [{ path: '/wt/c', reason: 'gone' }] }))).toBe(
      true,
    );
  });
});

describe('worktreeLabel', () => {
  it('prefers branch and falls back to path', () => {
    expect(worktreeLabel('/wt/a', 'main')).toBe('main');
    expect(worktreeLabel('/wt/a')).toBe('/wt/a');
  });
});

describe('confidenceLabelKey', () => {
  it('maps confidence levels to i18n keys', () => {
    expect(
      confidenceLabelKey({
        file: 'a.go',
        worktrees: ['/wt/a'],
        confidence: 'hunk-overlap',
        uncertain: false,
        reason: 'r',
      }),
    ).toBe('agents.collisions.conflictLikely');
    expect(
      confidenceLabelKey({
        file: 'a.go',
        worktrees: ['/wt/a'],
        confidence: 'same-file',
        uncertain: true,
        reason: 'r',
      }),
    ).toBe('agents.collisions.uncertainCandidate');
  });
});

describe('sortCollisions', () => {
  it('puts hunk overlaps first and is stable by file', () => {
    const sorted = sortCollisions([
      {
        file: 'z.go',
        worktrees: ['/wt/a'],
        confidence: 'same-file',
        uncertain: true,
        reason: 'r',
      },
      {
        file: 'm.go',
        worktrees: ['/wt/a'],
        confidence: 'hunk-overlap',
        uncertain: false,
        reason: 'r',
      },
    ]);
    expect(sorted.map((c) => c.file)).toEqual(['m.go', 'z.go']);
  });
});
