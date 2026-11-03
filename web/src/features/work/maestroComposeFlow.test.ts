import { describe, expect, it } from 'vitest';
import type { MaestroComposePreviewResult, WorkPlan } from '../../types';
import { flowFromWorkPlan } from './flowModel';
import { draftFromMaestroComposition } from './maestroComposeFlow';

const baseline = flowFromWorkPlan({
  id: 'flow-1',
  project_id: 'project-1',
  title: 'Existing Flow',
  description: 'Existing',
  status: 'READY',
  current_revision: 4,
  structured_facts: { 'nexus.source_artifact_id': 'artifact-1' },
  phases: [{ id: 'phase-1', title: 'Old', order: 1, packages: [] }],
  created_at: '2026-11-03T00:00:00Z',
  updated_at: '2026-11-03T00:00:00Z',
} satisfies WorkPlan);

const preview: MaestroComposePreviewResult = {
  schemaVersion: 1,
  authority: 'MAESTRO',
  source: 'maestro-semantic-planner',
  providerId: 'codex',
  reviewOnly: true,
  executable: false,
  requiresApproval: true,
  artifact: { id: 'artifact-1', version: 1, sha256: 'a'.repeat(64) },
  context: { contentDigest: 'b'.repeat(64) },
  validation: { valid: true, blockers: [], warnings: [] },
  tasks: [
    {
      id: 'api',
      title: 'Implement API',
      objective: 'Build API',
      dependsOn: [],
      acceptanceCriteria: ['API handles invalid requests'],
    },
    {
      id: 'verify',
      title: 'Verify',
      objective: 'Test API',
      dependsOn: ['api'],
      acceptanceCriteria: ['Tests pass'],
    },
  ],
};

describe('Maestro composition into Flow draft', () => {
  it('imports approved DAG without launching or pinning a provider', () => {
    const draft = draftFromMaestroComposition(baseline, preview, 'Maestro review');
    expect(draft.status).toBe('DRAFT');
    expect(draft.policy).toBe('GUIDED');
    expect(draft.steps.map((step) => [step.id, step.dependencies])).toEqual([
      ['api', []],
      ['verify', ['api']],
    ]);
    expect(draft.steps.every((step) => step.provider === undefined)).toBe(true);
    expect(draft.structuredFacts?.['nexus.source_artifact_id']).toBe('artifact-1');
    expect(draft.structuredFacts?.['nexus.maestro.composition_context_digest']).toBe('b'.repeat(64));
    expect(baseline.steps).toHaveLength(0);
  });

  it('refuses invalid graph and executable response', () => {
    expect(() => draftFromMaestroComposition(baseline, {
      ...preview, executable: true,
    } as unknown as MaestroComposePreviewResult, 'Maestro review')).toThrow();
    expect(() => draftFromMaestroComposition(baseline, {
      ...preview, validation: { valid: false }, tasks: null,
    }, 'Maestro review')).toThrow();
    expect(() => draftFromMaestroComposition(baseline, {
      ...preview, tasks: [{ ...preview.tasks![1], dependsOn: ['missing'] }],
    }, 'Maestro review')).toThrow();
  });
});
