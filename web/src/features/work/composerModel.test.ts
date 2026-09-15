import { describe, expect, it } from 'vitest';
import {
  composerGateForReadiness,
  composerNeedsGapConfirmation,
  formatComposerSessionTitle,
} from './composerModel';

describe('Composer Context Readiness gate', () => {
  it('keeps destination permissions separated from Composer editability', () => {
    expect(composerGateForReadiness('READY').canMaterialize).toBe(true);
    expect(composerGateForReadiness('MISSING').canMaterialize).toBe(false);
    expect(composerGateForReadiness('STALE').canExecute).toBe(false);
  });
  it('maps non-ready states to explicit actions', () => {
    expect(composerGateForReadiness('MISSING').action).toBe('PREPARE');
    expect(composerGateForReadiness('HYDRATING').action).toBe('WAIT');
    expect(composerGateForReadiness('STALE').action).toBe('REFRESH');
    expect(composerGateForReadiness('FAILED').action).toBe('RETRY');
  });
});

describe('formatComposerSessionTitle', () => {
  it('replaces keyboard-spam titles with a compact fallback', () => {
    expect(formatComposerSessionTitle('çlkkkkkkkkkkk', 'cmp_ABCDEF123456')).toBe('EF123456');
  });

  it('keeps meaningful titles and trims long ones', () => {
    expect(formatComposerSessionTitle('Auth JWT', 'cmp_x')).toBe('Auth JWT');
    expect(formatComposerSessionTitle('a'.repeat(60), 'cmp_x').endsWith('…')).toBe(true);
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
