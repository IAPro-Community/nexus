import { describe, expect, it } from 'vitest';
import type { WorkPlan } from '../../types';
import { executionWaves, expandStepLocalized, flowFromWorkPlan, removeStepLocalized, splitStepLocalized, topologicalOrder, updateStepLocalized, validateFlow, workPlanFromFlow } from './flowModel';

const fixture = (): WorkPlan => ({
  id: 'plan', project_id: 'project', title: 'Flow', description: 'desc', status: 'DRAFT', current_revision: 4,
  structured_facts: { 'existing.fact': 'keep', 'nexus.flow_policy': 'AUTONOMOUS' }, created_at: '2026-09-01T00:00:00Z', updated_at: '2026-09-01T00:00:00Z',
  phases: [
    { id:'p1', title:'Build', order:1, packages:[
      { id:'A', title:'A', goal:'A', priority:'NORMAL', status:'READY', dependencies:[], role:'architect', agent_allocation:'agent-a', assignment_strategy:'EXISTING', resource_policy:'BALANCED', acceptance_criteria:['a'] },
      { id:'B', title:'B', goal:'B', priority:'HIGH', status:'READY', dependencies:['A'], parallel_group:'impl', role:'implementer', assignment_strategy:'CREATE', maestro_skills:['backend'], relevant_paths:['internal'], acceptance_criteria:['b'], verification_requirements:['go test ./internal/...'] },
      { id:'C', title:'C', goal:'C', priority:'HIGH', status:'READY', dependencies:['A'], parallel_group:'impl', role:'implementer', assignment_strategy:'AUTO', provider:'codex', profile:'fast', relevant_paths:['web/src'], acceptance_criteria:['c'], verification_requirements:['npm test'] },
    ]},
    { id:'p2', title:'Verify', order:2, packages:[
      { id:'D', title:'D', goal:'D', priority:'CRITICAL', status:'READY', dependencies:['B','C'], role:'tester', assignment_strategy:'AUTO', maestro_skills:['verification'], acceptance_criteria:['d'], verification_requirements:['go test ./...','npm test'] },
    ]},
  ],
});

describe('Flow editor compatibility model', () => {
  it('round-trips WorkPlan fields used by Flow', () => {
    const plan = fixture();
    const flow = flowFromWorkPlan(plan);
    expect(flow.policy).toBe('AUTONOMOUS');
    expect(flowFromWorkPlan(workPlanFromFlow(flow, plan))).toEqual(flow);
  });

  it('fails closed to an empty draft when a persisted plan has null collections', () => {
    const malformed = { ...fixture(), phases: null as unknown as WorkPlan['phases'] };
    expect(flowFromWorkPlan(malformed)).toMatchObject({ phases: [], steps: [] });

    const malformedPhase = {
      ...fixture(),
      phases: [{ ...fixture().phases[0], packages: null as unknown as WorkPlan['phases'][number]['packages'] }],
    };
    expect(flowFromWorkPlan(malformedPhase)).toMatchObject({ phases: [{ id: 'p1' }], steps: [] });
  });

  it('computes deterministic A -> B||C -> D waves', () => {
    const flow = flowFromWorkPlan(fixture());
    expect(topologicalOrder(flow)).toEqual(['A','B','C','D']);
    expect(executionWaves(flow)).toEqual([['A'],['B','C'],['D']]);
  });

  it('rejects cycles and unknown dependencies', () => {
    let flow = flowFromWorkPlan(fixture());
    flow = updateStepLocalized(flow, 'A', { dependencies:['D'] });
    expect(validateFlow(flow).some((item) => item.includes('cycle'))).toBe(true);
    flow = flowFromWorkPlan(fixture());
    flow = updateStepLocalized(flow, 'B', { dependencies:['missing'] });
    expect(validateFlow(flow).some((item) => item.includes('unknown'))).toBe(true);
  });

  it('edits only the selected Step', () => {
    const flow = flowFromWorkPlan(fixture());
    const beforeA = JSON.stringify(flow.steps.find((step) => step.id==='A'));
    const beforeC = JSON.stringify(flow.steps.find((step) => step.id==='C'));
    const next = updateStepLocalized(flow,'B',{goal:'refined B',acceptanceCriteria:['b','extra']});
    expect(JSON.stringify(next.steps.find((step) => step.id==='A'))).toBe(beforeA);
    expect(JSON.stringify(next.steps.find((step) => step.id==='C'))).toBe(beforeC);
    expect(next.steps.find((step) => step.id==='B')?.goal).toBe('refined B');
  });

  it('split redirects dependents and remove only normalizes direct references', () => {
    const flow = flowFromWorkPlan(fixture());
    const split = splitStepLocalized(flow,'B',['B1','B2']);
    expect(split.steps.find((step) => step.id==='B2')?.dependencies).toEqual(['B1']);
    expect(split.steps.find((step) => step.id==='D')?.dependencies).toEqual(['B2','C']);
    const removed = removeStepLocalized(flow,'B');
    expect(removed.steps.some((step) => step.id==='B')).toBe(false);
    expect(removed.steps.find((step) => step.id==='D')?.dependencies).toEqual(['C']);
  });

  it('validates explicit assignment contracts', () => {
    let flow=flowFromWorkPlan(fixture());
    flow=updateStepLocalized(flow,'A',{agentId:undefined});
    expect(validateFlow(flow).some((item)=>item.includes('EXISTING'))).toBe(true);
    flow=flowFromWorkPlan(fixture());
    flow=updateStepLocalized(flow,'B',{assignmentStrategy:'AUTO',agentId:'agent-x'});
    expect(validateFlow(flow).some((item)=>item.includes('AUTO'))).toBe(true);
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
    expect(expanded.steps.find((step) => step.id === 'B')!.acceptanceCriteria.length).toBeGreaterThan(flow.steps.find((step) => step.id === 'B')!.acceptanceCriteria.length);
  });
});
