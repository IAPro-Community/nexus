import { describe, expect, it } from 'vitest';
import {
  composerActionPhase,
  composerBriefStats,
  composerContextBudgetPercent,
  composerContextItems,
  composerExpectedOutcome,
  composerGateForReadiness,
  composerNeedsGapConfirmation,
  composerReadinessDisplay,
  selectComposerNextUnknown,
  openComposerUnknowns,
  formatComposerRelativeTime,
  formatComposerSessionTitle,
} from './composerModel';
import type { ComposerSessionView } from '../../types';

describe('Composer Context Readiness gate', () => {
  it('keeps destination permissions separated from Composer editability', () => {
    expect(composerGateForReadiness('READY').canMaterialize).toBe(true);
    expect(composerGateForReadiness('MISSING').canMaterialize).toBe(false);
    expect(composerGateForReadiness('MISSING').canExecute).toBe(false);
    expect(composerGateForReadiness('STALE').canMaterialize).toBe(true);
    expect(composerGateForReadiness('STALE').canExecute).toBe(true);
    expect(composerGateForReadiness('FAILED').canExecute).toBe(false);
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
  it('requires confirmation for genuinely blocking gaps', () => {
    expect(
      composerNeedsGapConfirmation({
        open_questions: [],
        readiness: { state: 'BLOCKED' },
      }),
    ).toBe(true);
    expect(
      composerNeedsGapConfirmation({
        open_questions: ['Which provider?'],
        readiness: { state: 'READY_WITH_ASSUMPTIONS' },
        unknowns: [{ severity: 'BLOCKING', status: 'OPEN' }],
      }),
    ).toBe(true);
  });

  it('does not interrupt a complete prompt for recommended open details', () => {
    expect(
      composerNeedsGapConfirmation({
        open_questions: ['Do you want an additional screenshot?'],
        readiness: { state: 'READY_WITH_ASSUMPTIONS' },
        unknowns: [{ severity: 'RECOMMENDED', status: 'OPEN' }],
      }),
    ).toBe(false);
  });

  it('allows finalization when blocking unknowns were resolved', () => {
    expect(
      composerNeedsGapConfirmation({
        open_questions: [],
        readiness: { state: 'READY' },
        unknowns: [{ severity: 'BLOCKING', status: 'ANSWERED' }],
      }),
    ).toBe(false);
  });

  it('keeps the legacy open-question fallback when structured unknowns are absent', () => {
    expect(
      composerNeedsGapConfirmation({
        open_questions: ['Legacy question'],
        readiness: { state: 'READY' },
      }),
    ).toBe(true);
  });
});

describe('composerReadinessDisplay', () => {
  it('uses a real positive score when the backend provided one', () => {
    expect(composerReadinessDisplay({ score: 82, state: 'READY_WITH_ASSUMPTIONS' })).toEqual({
      mode: 'score',
      score: 82,
      state: 'READY_WITH_ASSUMPTIONS',
    });
  });

  it('does not invent a percentage for zero or missing scores', () => {
    expect(composerReadinessDisplay({ score: 0, state: 'EXPLORING' })).toEqual({
      mode: 'semantic',
      state: 'EXPLORING',
    });
    expect(composerReadinessDisplay(null)).toEqual({ mode: 'semantic', state: 'UNKNOWN' });
  });
});

describe('composerBriefStats', () => {
  it('counts decisions, inferred premises, risks and open unknowns from the view', () => {
    const view = {
      brief: {
        goal: 'Ship',
        decisions: ['Use Go'],
        risks: ['Quota'],
        open_questions: ['Security gate?'],
        unknowns: [
          { id: 'u1', question: 'Gate?', severity: 'BLOCKING', status: 'OPEN' },
          { id: 'u2', question: 'Stack?', severity: 'NICE_TO_HAVE', status: 'INFERRED' },
        ],
      },
    } as unknown as ComposerSessionView;
    expect(composerBriefStats(view)).toEqual({
      decisionsDefined: 1,
      decisionsPending: 1,
      premisesInferred: 1,
      risksIdentified: 1,
      openDoubts: 1,
    });
  });
});

describe('formatComposerRelativeTime', () => {
  it('formats a past timestamp relative to now', () => {
    const now = Date.parse('2026-09-15T12:00:00Z');
    const label = formatComposerRelativeTime('2026-09-15T11:48:00Z', now);
    expect(label.length).toBeGreaterThan(0);
    expect(label).not.toMatch(/82%/);
  });
});

describe('composerActionPhase', () => {
  it('keeps elaboration until readiness or an artifact exists', () => {
    const exploring = {
      session: { state: 'EXPLORING' },
      brief: { readiness: { state: 'NEEDS_INFORMATION' } },
    } as unknown as ComposerSessionView;
    expect(composerActionPhase(exploring, false)).toBe('elaborating');
    expect(composerActionPhase(exploring, true)).toBe('finalized');
    expect(
      composerActionPhase(
        {
          session: { state: 'EXPLORING' },
          brief: { readiness: { state: 'READY' } },
        } as unknown as ComposerSessionView,
        false,
      ),
    ).toBe('ready_review');
  });
});

describe('composerExpectedOutcome', () => {
  it('prefers definition of done over acceptance criteria', () => {
    expect(
      composerExpectedOutcome({
        brief: {
          quality: {
            definition_of_done: ['Merged to main'],
            acceptance_criteria: ['Tests pass'],
          },
        },
      } as unknown as ComposerSessionView),
    ).toBe('Merged to main');
  });
});

describe('composer context manifest projection', () => {
  it('separates kept, referenced and dropped context without inventing entries', () => {
    const view = {
      context_manifest: {
        version: '1',
        project_id: 'p1',
        fingerprint: 'fp',
        generated_at: '2026-09-21T00:00:00Z',
        budget: {
          max_estimated_tokens: 100,
          used_estimated_tokens: 40,
          kept_items: 1,
          referenced_items: 1,
          dropped_items: 1,
        },
        items: [
          {
            id: 'keep',
            kind: 'PROJECT_FACT',
            source: 'project',
            summary: 'fact',
            decision: 'KEEP',
            reason: 'relevant',
            estimated_tokens: 40,
          },
          {
            id: 'ref',
            kind: 'MISSION_REFERENCE',
            source: 'mission',
            summary: 'mission',
            decision: 'REFERENCE_ONLY',
            reason: 'reference',
            estimated_tokens: 10,
          },
          {
            id: 'drop',
            kind: 'PROJECT_FACT',
            source: 'project',
            summary: 'noise',
            decision: 'DROP',
            reason: 'budget',
            estimated_tokens: 70,
          },
        ],
      },
    } as unknown as ComposerSessionView;

    expect(composerContextItems(view, 'KEEP').map((item) => item.id)).toEqual(['keep']);
    expect(composerContextItems(view, 'REFERENCE_ONLY').map((item) => item.id)).toEqual(['ref']);
    expect(composerContextItems(view, 'DROP').map((item) => item.id)).toEqual(['drop']);
    expect(composerContextBudgetPercent(view)).toBe(40);
  });
});

describe('Composer Interview Engine question selection', () => {
  it('does not re-ask inferred repository or memory facts', () => {
    const view = {
      brief: {
        goal: 'Improve routing',
        unknowns: [
          {
            id: 'stack',
            question: 'Which stack?',
            severity: 'RECOMMENDED',
            status: 'INFERRED',
            source: 'PROJECT_CONTEXT',
            priority_score: 99,
          },
          {
            id: 'acceptance',
            question: 'What proves success?',
            severity: 'RECOMMENDED',
            status: 'OPEN',
            source: 'INTERVIEW',
            priority_score: 70,
          },
        ],
      },
    } as unknown as ComposerSessionView;

    expect(openComposerUnknowns(view).map((item) => item.id)).toEqual(['acceptance']);
    expect(selectComposerNextUnknown(view)?.id).toBe('acceptance');
  });

  it('prefers expected information gain over checklist order', () => {
    const view = {
      brief: {
        goal: 'Improve routing',
        unknowns: [
          {
            id: 'scope',
            question: 'Scope?',
            severity: 'RECOMMENDED',
            status: 'OPEN',
            priority_score: 45,
          },
          {
            id: 'acceptance',
            question: 'Success?',
            severity: 'RECOMMENDED',
            status: 'OPEN',
            priority_score: 88,
          },
        ],
      },
    } as unknown as ComposerSessionView;

    expect(selectComposerNextUnknown(view)?.id).toBe('acceptance');
  });
});
