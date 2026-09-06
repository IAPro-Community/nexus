import { describe, expect, it } from 'vitest';
import { composerGateForReadiness, composerNeedsGapConfirmation } from './composerModel';
import type { ContextReadinessState } from '../../types';

describe('Composer Context Readiness gate', () => {
  const states: ContextReadinessState[] = ['MISSING', 'HYDRATING', 'READY', 'STALE', 'FAILED'];
  it('allows planning/refinement only when READY', () => {
    for (const state of states)
      expect(composerGateForReadiness(state).canCompose).toBe(state === 'READY');
  });
  it('maps non-ready states to explicit actions', () => {
    expect(composerGateForReadiness('MISSING').action).toBe('PREPARE');
    expect(composerGateForReadiness('HYDRATING').action).toBe('WAIT');
    expect(composerGateForReadiness('STALE').action).toBe('REFRESH');
    expect(composerGateForReadiness('FAILED').action).toBe('RETRY');
  });
});

describe('Composer finalization guard', () => {
  it('detects blocking readiness or open questions before a finalize request', () => {
    expect(
      composerNeedsGapConfirmation({
        open_questions: [],
        readiness: { state: 'BLOCKED' },
      }),
    ).toBe(true);
    expect(
      composerNeedsGapConfirmation({
        open_questions: ['Which provider?'],
        readiness: { state: 'READY' },
      }),
    ).toBe(true);
  });

  it('allows finalization when readiness is clear and no questions remain', () => {
    expect(
      composerNeedsGapConfirmation({
        open_questions: [],
        readiness: { state: 'READY' },
      }),
    ).toBe(false);
  });
});
