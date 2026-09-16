import { describe, expect, it } from 'vitest';
import type { WorkPlan } from '../../types';
import {
  executionWaves,
  expandStepLocalized,
  flowFromWorkPlan,
  flowGraphSignature,
  removeStepLocalized,
  splitStepLocalized,
  topologicalOrder,
  updateStepLocalized,
  validateFlow,
  workPlanFromFlow,
} from './flowModel';

const fixture = (): WorkPlan => ({
  id: 'plan',
  project_id: 'project',
  title: 'Flow',
  description: 'desc',
  status: 'DRAFT',
  current_revision: 4,
  structured_facts: { 'existing.fact': 'keep', 'nexus.flow_policy': 'AUTONOMOUS' },
  created_at: '2026-09-01T00:00:00Z',
  updated_at: '2026-09-01T00:00:00Z',
  phases: [
    {
      id: 'p1',
      title: 'Build',
      order: 1,
      packages: [
        {
          id: 'A',
          title: 'A',
          goal: 'A',
          priority: 'NORMAL',
          status: 'READY',
          dependencies: [],
          role: 'architect',
          agent_allocation: 'agent-a',
          assignment_strategy: 'EXISTING',
          resource_policy: 'BALANCED',
          acceptance_criteria: ['a'],
        },
        {
          id: 'B',
          title: 'B',
          goal: 'B',
          priority: 'HIGH',
          status: 'READY',
          dependencies: ['A'],
          parallel_group: 'impl',
          role: 'implementer',
          assignment_strategy: 'CREATE',
          skill_ids: ['backend'],
          maestro_skills: ['backend'],
          relevant_paths: ['internal'],
          acceptance_criteria: ['b'],
          verification_requirements: ['go test ./internal/...'],
        },
        {
          id: 'C',
          title: 'C',
          goal: 'C',
          priority: 'HIGH',
          status: 'READY',
          dependencies: ['A'],
          parallel_group: 'impl',
          role: 'implementer',
          assignment_strategy: 'AUTO',
          provider: 'codex',
          profile: 'fast',
          relevant_paths: ['web/src'],
          acceptance_criteria: ['c'],
          verification_requirements: ['npm test'],
        },
      ],
    },
    {
      id: 'p2',
      title: 'Verify',
      order: 2,
      packages: [
        {
          id: 'D',
          title: 'D',
          goal: 'D',
          priority: 'CRITICAL',
          status: 'READY',
          dependencies: ['B', 'C'],
          role: 'tester',
          assignment_strategy: 'AUTO',
          maestro_skills: ['verification'],
          acceptance_criteria: ['d'],
          verification_requirements: ['go test ./...', 'npm test'],
        },
      ],
    },
  ],
});

describe('Flow editor compatibility model', () => {
  it('round-trips WorkPlan fields used by Flow', () => {
    const plan = fixture();
    const flow = flowFromWorkPlan(plan);
    expect(flow.policy).toBe('AUTONOMOUS');
    expect(flowFromWorkPlan(workPlanFromFlow(flow, plan))).toEqual(flow);
  });

  it('round-trips canonical skill_ids while retaining legacy aliases', () => {
    const plan = fixture();
    const flow = flowFromWorkPlan(plan);
    expect(flow.steps.find((step) => step.id === 'B')?.skillIds).toEqual(['backend']);
    const roundTrip = workPlanFromFlow(flow, plan);
    expect(roundTrip.phases[0].packages[1].skill_ids).toEqual(['backend']);
    expect(roundTrip.phases[0].packages[1].maestro_skills).toEqual(['backend']);
  });

  it('treats nullable API phase collections as empty', () => {
    const malformed = {
      ...fixture(),
      phases: [
        { ...fixture().phases[0], packages: null },
        { ...fixture().phases[1], packages: null },
      ],
    } as unknown as WorkPlan;

    expect(flowFromWorkPlan(malformed)).toMatchObject({
      phases: [
        { id: 'p1', title: 'Build', order: 1 },
        { id: 'p2', title: 'Verify', order: 2 },
      ],
      steps: [],
    });
  });

  it('normalizes null package dependencies so canvas render does not crash', () => {
    const malformed = {
      ...fixture(),
      phases: [
        {
          ...fixture().phases[0],
          packages: [
            { ...fixture().phases[0].packages[0], dependencies: null },
            { ...fixture().phases[0].packages[1], dependencies: null },
          ],
        },
      ],
    } as unknown as WorkPlan;
    const flow = flowFromWorkPlan(malformed);
    expect(flow.steps.every((step) => Array.isArray(step.dependencies))).toBe(true);
    expect(() => executionWaves(flow)).not.toThrow();
    for (const step of flow.steps) {
      expect((step.dependencies || []).length).toBeGreaterThanOrEqual(0);
    }
  });

  it('normalizes null package dependency arrays so canvas .length is safe', () => {
    const malformed = {
      ...fixture(),
      phases: [
        {
          id: 'p1',
          title: 'Build',
          order: 1,
          packages: [
            {
              id: 'A',
              title: 'A',
              goal: 'A',
              priority: 'NORMAL',
              status: 'READY',
              dependencies: null,
              role: 'architect',
              assignment_strategy: 'AUTO',
              acceptance_criteria: null,
            },
          ],
        },
      ],
    } as unknown as WorkPlan;

    const flow = flowFromWorkPlan(malformed);
    expect(flow.steps[0].dependencies).toEqual([]);
    expect(flow.steps[0].acceptanceCriteria).toEqual([]);
    expect(() => executionWaves(flow)).not.toThrow();
    expect((flow.steps[0].dependencies || []).length).toBe(0);
  });

  it('computes deterministic A -> B||C -> D waves', () => {
    const flow = flowFromWorkPlan(fixture());
    expect(topologicalOrder(flow)).toEqual(['A', 'B', 'C', 'D']);
    expect(executionWaves(flow)).toEqual([['A'], ['B', 'C'], ['D']]);
  });

  it('drops schema placeholders so a copied CLI outline still renders', () => {
    const plan: WorkPlan = {
      ...fixture(),
      phases: [
        {
          id: 'p1',
          title: 'Execution Phase',
          order: 1,
          packages: [
            {
              id: 'pkg_1',
              title: '...',
              goal: '...',
              priority: 'HIGH',
              status: 'READY',
              dependencies: ['package title'],
              role: 'implementer',
              assignment_strategy: 'AUTO',
              acceptance_criteria: ['measurable criterion'],
            },
          ],
        },
      ],
    };
    const flow = flowFromWorkPlan(plan);
    expect(flow.steps[0].dependencies).toEqual([]);
    expect(flow.steps[0].title).not.toBe('...');
    expect(flow.steps[0].goal).not.toBe('...');
    expect(() => executionWaves(flow)).not.toThrow();
    expect(validateFlow(flow).some((item) => item.includes('unknown dependency'))).toBe(false);
  });

  it('resolves dependency titles to package ids', () => {
    const plan: WorkPlan = {
      ...fixture(),
      phases: [
        {
          id: 'p1',
          title: 'Build',
          order: 1,
          packages: [
            {
              id: 'pkg_a',
              title: 'Foundation',
              goal: 'schema',
              priority: 'HIGH',
              status: 'READY',
              dependencies: [],
              role: 'architect',
              assignment_strategy: 'AUTO',
              acceptance_criteria: ['ok'],
            },
            {
              id: 'pkg_b',
              title: 'API',
              goal: 'http',
              priority: 'HIGH',
              status: 'READY',
              dependencies: ['Foundation'],
              role: 'implementer',
              assignment_strategy: 'AUTO',
              acceptance_criteria: ['ok'],
            },
          ],
        },
      ],
    };
    const flow = flowFromWorkPlan(plan);
    expect(flow.steps.find((step) => step.id === 'pkg_b')?.dependencies).toEqual(['pkg_a']);
    expect(() => executionWaves(flow)).not.toThrow();
  });

  it('rejects cycles and unknown dependencies', () => {
    let flow = flowFromWorkPlan(fixture());
    flow = updateStepLocalized(flow, 'A', { dependencies: ['D'] });
    expect(validateFlow(flow).some((item) => item.includes('cycle'))).toBe(true);
    flow = flowFromWorkPlan(fixture());
    flow = updateStepLocalized(flow, 'B', { dependencies: ['missing'] });
    expect(validateFlow(flow).some((item) => item.includes('unknown'))).toBe(true);
  });

  it('edits only the selected Step', () => {
    const flow = flowFromWorkPlan(fixture());
    const beforeA = JSON.stringify(flow.steps.find((step) => step.id === 'A'));
    const beforeC = JSON.stringify(flow.steps.find((step) => step.id === 'C'));
    const next = updateStepLocalized(flow, 'B', {
      goal: 'refined B',
      acceptanceCriteria: ['b', 'extra'],
    });
    expect(JSON.stringify(next.steps.find((step) => step.id === 'A'))).toBe(beforeA);
    expect(JSON.stringify(next.steps.find((step) => step.id === 'C'))).toBe(beforeC);
    expect(next.steps.find((step) => step.id === 'B')?.goal).toBe('refined B');
  });

  it('split redirects dependents and remove only normalizes direct references', () => {
    const flow = flowFromWorkPlan(fixture());
    const split = splitStepLocalized(flow, 'B', ['B1', 'B2']);
    expect(split.steps.find((step) => step.id === 'B2')?.dependencies).toEqual(['B1']);
    expect(split.steps.find((step) => step.id === 'D')?.dependencies).toEqual(['B2', 'C']);
    const removed = removeStepLocalized(flow, 'B');
    expect(removed.steps.some((step) => step.id === 'B')).toBe(false);
    expect(removed.steps.find((step) => step.id === 'D')?.dependencies).toEqual(['C']);
  });

  it('validates explicit assignment contracts', () => {
    let flow = flowFromWorkPlan(fixture());
    flow = updateStepLocalized(flow, 'A', { agentId: undefined });
    expect(validateFlow(flow).some((item) => item.includes('EXISTING'))).toBe(true);
    flow = flowFromWorkPlan(fixture());
    flow = updateStepLocalized(flow, 'B', { assignmentStrategy: 'AUTO', agentId: 'agent-x' });
    expect(validateFlow(flow).some((item) => item.includes('AUTO'))).toBe(true);
  });
});

describe('Flow phase and localized expansion contracts', () => {
  it('rejects a Step that references a phase that does not exist', () => {
    const flow = flowFromWorkPlan(fixture());
    flow.steps[1] = { ...flow.steps[1], phaseId: 'missing-phase' };
    expect(validateFlow(flow).some((error) => error.includes('unknown phase'))).toBe(true);
  });

  it('expands only the selected Step without rewriting unrelated Steps', () => {
    const flow = flowFromWorkPlan(fixture());
    const beforeA = JSON.stringify(flow.steps.find((step) => step.id === 'A')!);
    const beforeC = JSON.stringify(flow.steps.find((step) => step.id === 'C')!);
    const beforeD = JSON.stringify(flow.steps.find((step) => step.id === 'D')!);
    const expanded = expandStepLocalized(flow, 'B');
    expect(JSON.stringify(expanded.steps.find((step) => step.id === 'A')!)).toBe(beforeA);
    expect(JSON.stringify(expanded.steps.find((step) => step.id === 'C')!)).toBe(beforeC);
    expect(JSON.stringify(expanded.steps.find((step) => step.id === 'D')!)).toBe(beforeD);
    expect(
      expanded.steps.find((step) => step.id === 'B')!.acceptanceCriteria.length,
    ).toBeGreaterThan(flow.steps.find((step) => step.id === 'B')!.acceptanceCriteria.length);
  });
});

describe('flowGraphSignature', () => {
  it('stays stable across cloned flow objects with the same steps', () => {
    const flow = flowFromWorkPlan(fixture());
    expect(flowGraphSignature({ ...flow, title: 'Other' })).toBe(flowGraphSignature(flow));
  });

  it('changes when a dependency is added', () => {
    const flow = flowFromWorkPlan(fixture());
    const next = updateStepLocalized(flow, 'C', { dependencies: ['A', 'B'] });
    expect(flowGraphSignature(next)).not.toBe(flowGraphSignature(flow));
  });
});
