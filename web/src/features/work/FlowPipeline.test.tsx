import { describe, expect, it } from 'vitest';
import { groupStepsByPhase } from './FlowPipeline';
import type { FlowStepModel } from './flowModel';

const step = (
  partial: Partial<FlowStepModel> & Pick<FlowStepModel, 'id' | 'phaseId' | 'order'>,
): FlowStepModel => ({
  title: partial.title || partial.id,
  goal: '',
  priority: 'NORMAL',
  status: 'READY',
  dependencies: [],
  role: 'planner',
  assignmentStrategy: 'AUTO',
  skillIds: [],
  maestroGates: [],
  maestroSkills: [],
  relevantPaths: [],
  acceptanceCriteria: [],
  verificationRequirements: [],
  sharedArtifacts: [],
  ...partial,
});

describe('FlowPipeline helpers', () => {
  it('groups and orders step cards by phase', () => {
    const map = groupStepsByPhase([
      step({ id: 's2', phaseId: 'p1', order: 2, title: 'Second' }),
      step({ id: 's1', phaseId: 'p1', order: 1, title: 'First' }),
      step({ id: 's3', phaseId: 'p2', order: 0, title: 'Build' }),
    ]);
    expect([...map.keys()]).toEqual(['p1', 'p2']);
    expect(map.get('p1')?.map((item) => item.id)).toEqual(['s1', 's2']);
    expect(map.get('p2')?.[0]?.title).toBe('Build');
  });

  it('returns empty map for empty pipeline', () => {
    expect(groupStepsByPhase([]).size).toBe(0);
  });
});
