import { describe, expect, it } from 'vitest';
import {
  composerActionPhase,
  composerBriefStats,
  composerConstraintLines,
  composerExpectedOutcome,
  composerReadinessDisplay,
  composerScopeLines,
  formatComposerRelativeTime,
} from './composerModel';
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

  it('treats a materialized artifact as the finalized action phase', () => {
    const view = {
      session: { state: 'EXPLORING' },
      brief: { readiness: { state: 'EXPLORING' } },
    } as unknown as ComposerSessionView;
    expect(composerActionPhase(view, true)).toBe('finalized');
    expect(
      composerActionPhase(
        { session: { state: 'FINALIZED' }, brief: {} } as unknown as ComposerSessionView,
        false,
      ),
    ).toBe('finalized');
  });

  it('reads overview fields from structured brief data', () => {
    const view = {
      brief: {
        quality: { definition_of_done: ['Green CI'] },
        constraints: { must: ['Keep Go'], extra: 'No rewrite' },
        context: { existing_state: ['CLI'], evidence: ['Makefile'] },
      },
    } as unknown as ComposerSessionView;
    expect(composerExpectedOutcome(view)).toBe('Green CI');
    expect(composerConstraintLines(view)).toEqual(['Keep Go', 'No rewrite']);
    expect(composerScopeLines(view)).toEqual(['CLI', 'Makefile']);
  });

  it('formats relative time in the active locale', () => {
    const now = Date.parse('2026-09-15T12:00:00Z');
    const pt = formatComposerRelativeTime('2026-09-15T11:48:00Z', now, 'pt-BR');
    expect(pt).toMatch(/minuto|minutos|há/i);
  });
});
