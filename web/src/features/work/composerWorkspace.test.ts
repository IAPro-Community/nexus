import { describe, expect, it } from 'vitest';
import { composerActionPhase, composerBriefStats, composerReadinessDisplay } from './composerModel';
import type { ComposerSessionView } from '../../types';

describe('composer workspace surfaces', () => {
  it('surfaces pending decisions without inventing a readiness percentage', () => {
    const view = {
      session: { state: 'EXPLORING' },
      brief: {
        decisions: ['Keep Go'],
        readiness: { state: 'NEEDS_INFORMATION', score: 0 },
        unknowns: [{ id: 'u1', question: 'Block merge?', status: 'OPEN', severity: 'BLOCKING' }],
      },
    } as unknown as ComposerSessionView;

    expect(composerActionPhase(view, false)).toBe('elaborating');
    expect(composerReadinessDisplay(view.brief.readiness).mode).toBe('semantic');
    expect(composerBriefStats(view).decisionsPending).toBe(1);
  });

  it('moves to review when readiness is actually ready', () => {
    const view = {
      session: { state: 'EXPLORING' },
      brief: { readiness: { state: 'READY', score: 41 } },
    } as unknown as ComposerSessionView;
    expect(composerActionPhase(view, false)).toBe('ready_review');
    expect(composerReadinessDisplay(view.brief.readiness)).toEqual({
      mode: 'score',
      score: 41,
      state: 'READY',
    });
  });
});
