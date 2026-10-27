import { describe, expect, it } from 'vitest';
import type { WorkPlan } from '../../types';
import { countComposerBlockingDecisions, readComposerHandoff } from './ComposerHandoffSummary';

const planWithFacts = (facts: Record<string, string>): WorkPlan =>
  ({ id: 'plan-test', structured_facts: facts }) as WorkPlan;

describe('Composer → Flow immutable handoff', () => {
  it('remains absent for manual and legacy WorkPlans', () => {
    expect(readComposerHandoff(planWithFacts({}))).toBeNull();
    expect(readComposerHandoff(null)).toBeNull();
  });

  it('rejects malformed and unsupported contracts without inventing approvals', () => {
    expect(readComposerHandoff(planWithFacts({ 'composer.handoff.v1': '{invalid' }))).toBeNull();
    expect(
      readComposerHandoff(planWithFacts({ 'composer.handoff.v1': '{"version":2}' })),
    ).toBeNull();
  });

  it('projects immutable lineage and differentiates proposals from decisions', () => {
    const raw = {
      version: 1,
      session_id: 'cmp_123',
      artifact_id: 'art_456',
      artifact_version: 2,
      artifact_hash: 'sha-example',
      goal: 'Criar aplicativo de comunicação escolar',
      approved_answers: { 'context.audience': 'Famílias e docentes' },
      recommendations: [{ value: 'Começar pelos comunicados', status: 'PROPOSED' }],
      unresolved: [
        { question: 'Posso alterar o banco de produção?', status: 'OPEN', severity: 'BLOCKING' },
        { question: 'Qual tema visual?', status: 'OPEN', severity: 'OPTIONAL' },
      ],
    };
    const parsed = readComposerHandoff(
      planWithFacts({
        'composer.handoff.v1': JSON.stringify(raw),
      }),
    );
    expect(parsed?.artifact_version).toBe(2);
    expect(parsed?.approved_answers?.['context.audience']).toBe('Famílias e docentes');
    expect(parsed?.recommendations?.[0].status).toBe('PROPOSED');
    expect(countComposerBlockingDecisions(parsed!)).toBe(1);
  });

  it('does not mark resolved blocking questions as outstanding', () => {
    const raw = {
      version: 1,
      artifact_id: 'a',
      goal: 'Tarefa',
      unresolved: [{ question: 'Definição', status: 'ANSWERED', severity: 'BLOCKING' }],
    };
    const parsed = readComposerHandoff(
      planWithFacts({
        'composer.handoff.v1': JSON.stringify(raw),
      }),
    );
    expect(countComposerBlockingDecisions(parsed!)).toBe(0);
  });
});
