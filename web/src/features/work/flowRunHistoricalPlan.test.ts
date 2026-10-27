import { describe, expect, it } from 'vitest';
import type { PlanRevision, WorkPlan } from '../../types';
import { selectRunHistoricalPlan } from './flowRunHistoricalPlan';

const plan = (revision: number, title: string): WorkPlan => ({
  id: 'plan-1',
  project_id: 'project-1',
  title,
  description: 'Descrição',
  status: 'READY',
  current_revision: revision,
  phases: [],
  created_at: '',
  updated_at: '',
});

const revision = (number: number, snapshot: string): PlanRevision => ({
  id: `rev-${number}`,
  plan_id: 'plan-1',
  revision: number,
  snapshot_json: snapshot,
  change_summary: '',
  created_at: '',
});

describe('Flow Run approved historical WorkPlan', () => {
  it('uses the current plan only when the run approved the same revision', () => {
    const current = plan(2, 'Revisão atual');
    expect(selectRunHistoricalPlan({ plan: current, revisions: [] }, 2)?.title).toBe(
      'Revisão atual',
    );
  });

  it('uses frozen revision instead of later Composer or Flow edits', () => {
    const first = plan(1, 'Planejamento aprovado originalmente');
    const current = plan(3, 'Planejamento editado posteriormente');
    const selected = selectRunHistoricalPlan(
      {
        plan: current,
        revisions: [revision(1, JSON.stringify(first))],
      },
      1,
    );
    expect(selected?.title).toBe('Planejamento aprovado originalmente');
    expect(selected?.current_revision).toBe(1);
  });

  it('never silently substitutes latest plan for missing or invalid evidence', () => {
    const current = plan(3, 'Última revisão');
    expect(selectRunHistoricalPlan({ plan: current, revisions: [] }, 1)).toBeNull();
    expect(
      selectRunHistoricalPlan(
        {
          plan: current,
          revisions: [revision(1, '{invalid')],
        },
        1,
      ),
    ).toBeNull();
    expect(
      selectRunHistoricalPlan(
        {
          plan: current,
          revisions: [revision(1, JSON.stringify(plan(2, 'Revisão errada')))],
        },
        1,
      ),
    ).toBeNull();
    expect(
      selectRunHistoricalPlan(
        {
          plan: current,
          revisions: [
            revision(1, JSON.stringify({ ...plan(1, 'Outro projeto'), project_id: 'other' })),
          ],
        },
        1,
      ),
    ).toBeNull();
  });
});
