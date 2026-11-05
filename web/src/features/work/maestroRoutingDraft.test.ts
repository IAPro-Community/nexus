import { describe, expect, it } from 'vitest';
import type { WorkPlan } from '../../types';
import { flowFromWorkPlan, workPlanFromFlow, type FlowDraftModel } from './flowModel';
import {
  MAESTRO_ROUTE_ASSIGNMENTS_FACT,
  readMaestroRoutingAssignments,
  updateMaestroRoutingAssignment,
} from './maestroRoutingDraft';

const step = (id: string): FlowDraftModel['steps'][number] => ({
  id,
  phaseId: 'phase-1',
  order: id === 'frontend' ? 0 : 1,
  title: id,
  goal: 'Deliver ' + id,
  priority: 'NORMAL',
  status: 'PENDING',
  dependencies: [],
  role: id,
  assignmentStrategy: 'AUTO',
  skillIds: [],
  maestroGates: [],
  maestroSkills: [],
  relevantPaths: [],
  acceptanceCriteria: ['Review and verify ' + id],
  verificationRequirements: [],
  sharedArtifacts: [],
});

const baseline = (): FlowDraftModel => ({
  id: 'flow-1',
  projectId: 'project-1',
  title: 'Product',
  description: 'Deliver product',
  status: 'DRAFT',
  revision: 3,
  policy: 'GUIDED',
  policyStored: true,
  phases: [{ id: 'phase-1', title: 'Build', order: 1 }],
  steps: [step('frontend'), step('backend')],
  structuredFacts: { 'nexus.source_artifact_id': 'artifact-1' },
});

describe('Maestro routing preferences stored with Flow', () => {
  it('survives a server roundtrip without inventing an executable Run', () => {
    const original = baseline();
    const updated = updateMaestroRoutingAssignment(original, 'frontend', {
      providerId: 'agy',
      model: 'frontend-model',
      sandbox: 'enabled',
      worktreePath: '/repo/agy',
    });
    expect(original.steps[0].provider).toBeUndefined();
    expect(updated.steps[0].provider).toBe('agy');
    expect(updated.structuredFacts?.['nexus.source_artifact_id']).toBe('artifact-1');
    expect(updated.structuredFacts?.[MAESTRO_ROUTE_ASSIGNMENTS_FACT]).toContain('frontend-model');

    const roundtrip = JSON.parse(JSON.stringify(updated)) as FlowDraftModel;
    expect(readMaestroRoutingAssignments(roundtrip).frontend).toEqual({
      providerId: 'agy',
      model: 'frontend-model',
      sandbox: 'enabled',
      worktreePath: '/repo/agy',
    });
  });

  it('preserves independent AGY and OpenCode selections when one is edited', () => {
    let draft = updateMaestroRoutingAssignment(baseline(), 'frontend', {
      providerId: 'agy',
      model: 'frontend-model',
      sandbox: 'enabled',
      worktreePath: '/repo/agy',
    });
    draft = updateMaestroRoutingAssignment(draft, 'backend', {
      providerId: 'opencode',
      model: '',
      sandbox: 'none',
      worktreePath: '/repo/opencode',
    });
    draft = updateMaestroRoutingAssignment(draft, 'frontend', {
      providerId: 'codex',
      model: '',
      sandbox: 'read-only',
      worktreePath: '/repo/codex',
    });
    const saved = readMaestroRoutingAssignments(draft);
    expect(saved.frontend.providerId).toBe('codex');
    expect(saved.backend).toEqual({
      providerId: 'opencode',
      model: '',
      sandbox: 'none',
      worktreePath: '/repo/opencode',
    });
    expect(draft.steps.map((item) => item.provider)).toEqual(['codex', 'opencode']);
  });

  it('refuses unknown tasks and unsupported sandbox/provider combinations', () => {
    expect(() =>
      updateMaestroRoutingAssignment(baseline(), 'foreign', {
        providerId: 'agy',
        model: '',
        sandbox: 'enabled',
        worktreePath: '/repo/agy',
      }),
    ).toThrow('Unknown Maestro routing task');
    expect(() =>
      updateMaestroRoutingAssignment(baseline(), 'backend', {
        providerId: 'opencode',
        model: '',
        sandbox: 'workspace-write',
        worktreePath: '/repo/opencode',
      }),
    ).toThrow('Invalid Maestro routing assignment');
  });


  it('serializes routing choices into a revisioned WorkPlan and restores them for another client', () => {
    const edited = updateMaestroRoutingAssignment(baseline(), 'frontend', {
      providerId: 'agy',
      model: 'frontend-model',
      sandbox: 'enabled',
      worktreePath: '/repo/agy',
    });
    const baselinePlan = {
      id: 'flow-1',
      project_id: 'project-1',
      title: 'Product',
      description: 'Deliver product',
      status: 'READY',
      current_revision: 3,
      phases: [],
      created_at: '2026-11-04T00:00:00Z',
      updated_at: '2026-11-04T00:00:00Z',
    } satisfies WorkPlan;
    const saved = workPlanFromFlow(
      { ...edited, status: 'DRAFT', policyStored: true },
      baselinePlan,
    );
    expect(saved.status).toBe('DRAFT');
    expect(saved.structured_facts?.['nexus.source_artifact_id']).toBe('artifact-1');
    const serverRevision = { ...saved, current_revision: 4 };
    const reopened = flowFromWorkPlan(serverRevision);
    expect(reopened.revision).toBe(4);
    expect(readMaestroRoutingAssignments(reopened).frontend).toEqual({
      providerId: 'agy',
      model: 'frontend-model',
      sandbox: 'enabled',
      worktreePath: '/repo/agy',
    });
  });

  it('keeps object prototypes intact for unusual task identifiers', () => {
    const draft = { ...baseline(), steps: [step('__proto__')] };
    const updated = updateMaestroRoutingAssignment(draft, '__proto__', {
      providerId: 'agy',
      model: '',
      sandbox: 'enabled',
      worktreePath: '/repo/agy',
    });
    const restored = readMaestroRoutingAssignments(updated);
    expect(Object.getPrototypeOf(restored)).toBe(Object.prototype);
    expect(Object.hasOwn(restored, '__proto__')).toBe(true);
    expect(restored.__proto__.providerId).toBe('agy');
  });

  it('ignores tampered data and never restores an assignment for a different provider', () => {
    const draft = updateMaestroRoutingAssignment(baseline(), 'frontend', {
      providerId: 'agy',
      model: '',
      sandbox: 'enabled',
      worktreePath: '/repo/agy',
    });
    expect(
      readMaestroRoutingAssignments({
        ...draft,
        steps: draft.steps.map((item) =>
          item.id === 'frontend' ? { ...item, provider: 'opencode' } : item,
        ),
      }).frontend,
    ).toBeUndefined();
    expect(
      readMaestroRoutingAssignments({
        ...draft,
        structuredFacts: {
          [MAESTRO_ROUTE_ASSIGNMENTS_FACT]: '{"schemaVersion":1,"assignments":"invalid"}',
        },
      }),
    ).toEqual({});
  });
});
