import type { MaestroComposePreviewResult } from '../../types';
import type { FlowDraftModel, FlowStepModel } from './flowModel';

// The operator explicitly imports a non-executable Maestro DAG as an editable
// Nexus Flow draft. No provider or agent is automatically assigned or started.
export function draftFromMaestroComposition(
  baseline: FlowDraftModel,
  preview: MaestroComposePreviewResult,
  phaseTitle: string,
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
  // A valid Maestro DAG may still belong to a different or older artifact.
  // Compare it with the frozen Composer handoff before replacing local edits.
  const facts = baseline.structuredFacts || {};
  if (
    facts['nexus.source_artifact_id'] !== preview.artifact.id ||
    !Number.isInteger(preview.artifact.version) ||
    preview.artifact.version < 1 ||
    !/^[a-f0-9]{64}$/i.test(preview.artifact.sha256)
  ) {
    throw new Error('Maestro composition does not match the source PromptArtifact');
  }
  const sourceRevision = facts['nexus.source_artifact_revision'];
  if (sourceRevision && sourceRevision !== String(preview.artifact.version)) {
    throw new Error('Maestro composition uses a stale PromptArtifact revision');
  }
  const handoffJSON = facts['composer.handoff.v1'];
  if (handoffJSON) {
    let handoff: unknown;
    try {
      handoff = JSON.parse(handoffJSON);
    } catch {
      throw new Error('Invalid immutable Composer handoff');
    }
    if (!handoff || typeof handoff !== 'object' || Array.isArray(handoff)) {
      throw new Error('Invalid immutable Composer handoff');
    }
    const frozen = handoff as Record<string, unknown>;
    if (
      frozen.artifact_id !== preview.artifact.id ||
      frozen.artifact_version !== preview.artifact.version ||
      typeof frozen.artifact_hash !== 'string' ||
      frozen.artifact_hash.toLowerCase() !== preview.artifact.sha256.toLowerCase()
    ) {
      throw new Error('Maestro composition conflicts with immutable Composer provenance');
    }
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
    phases: [{ id: phaseId, title: phaseTitle, order: 1 }],
    steps,
    structuredFacts: {
      ...baseline.structuredFacts,
      'nexus.maestro.composition_artifact_sha256': preview.artifact.sha256,
      'nexus.maestro.composition_context_digest': preview.context.contentDigest,
    },
  };
}
