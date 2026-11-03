import type { MaestroComposePreviewResult } from '../../types';
import type { FlowDraftModel, FlowStepModel } from './flowModel';

// The operator explicitly imports a non-executable Maestro DAG as an editable
// Nexus Flow draft. No provider or agent is automatically assigned or started.
export function draftFromMaestroComposition(
  baseline: FlowDraftModel,
  preview: MaestroComposePreviewResult,
): FlowDraftModel {
  if (
    preview.authority !== 'MAESTRO' ||
    !preview.reviewOnly ||
    preview.executable ||
    !preview.requiresApproval ||
    !preview.validation.valid ||
    !preview.tasks?.length ||
    preview.tasks.length > 32
  ) {
    throw new Error('Maestro returned a non-adoptable composition');
  }
  const ids = new Set(preview.tasks.map((task) => task.id));
  if (
    ids.size !== preview.tasks.length ||
    preview.tasks.some(
      (task) =>
        !task.id ||
        !task.title.trim() ||
        !task.objective.trim() ||
        !task.acceptanceCriteria.length ||
        task.dependsOn.some((id) => id === task.id || !ids.has(id)),
    )
  ) {
    throw new Error('Maestro returned an invalid task graph');
  }

  const phaseId = baseline.phases[0]?.id || `phase-maestro-${baseline.id}`;
  const steps: FlowStepModel[] = preview.tasks.map((task, index) => ({
    id: task.id,
    phaseId,
    order: index,
    title: task.title,
    goal: task.objective,
    priority: 'NORMAL',
    status: 'PENDING',
    dependencies: [...task.dependsOn],
    role: baseline.steps.find((step) => step.id === task.id)?.role || 'generalist',
    assignmentStrategy: 'AUTO',
    skillIds: [],
    maestroGates: [],
    maestroSkills: [],
    relevantPaths: [],
    acceptanceCriteria: [...task.acceptanceCriteria],
    verificationRequirements: [],
    sharedArtifacts: [],
  }));

  return {
    ...baseline,
    policy: 'GUIDED',
    policyStored: true,
    status: 'DRAFT',
    phases: [{ id: phaseId, title: 'Maestro · plano para revisão', order: 1 }],
    steps,
    structuredFacts: {
      ...baseline.structuredFacts,
      'nexus.maestro.composition_artifact_sha256': preview.artifact.sha256,
      'nexus.maestro.composition_context_digest': preview.context.contentDigest,
    },
  };
}
