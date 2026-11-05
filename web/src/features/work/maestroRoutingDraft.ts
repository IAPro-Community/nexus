import type { MaestroRoutingProvider, MaestroWorkAssignmentInput } from '../../types';
import type { FlowDraftModel } from './flowModel';

// This is UI plan state, not a Maestro execution ledger or semantic memory.
// Persist alongside the WorkPlan so another Nexus client can review the same
// provider/model/sandbox/worktree choices. Maestro revalidates on every preview.
export const MAESTRO_ROUTE_ASSIGNMENTS_FACT = 'nexus.maestro.route_assignments.v1';

export type MaestroRouteAssignmentDraft = {
  providerId: MaestroRoutingProvider | '';
  model: string;
  sandbox: MaestroWorkAssignmentInput['sandbox'];
  worktreePath: string;
};

const PROVIDERS = new Set(['agy', 'opencode', 'codex', 'claude']);
const SANDBOXES = new Set(['none', 'enabled', 'read-only', 'workspace-write']);
const SAFE_ID = /^[A-Za-z0-9._:-]{1,128}$/;

function validAssignment(value: unknown): value is MaestroRouteAssignmentDraft {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false;
  const item = value as Partial<MaestroRouteAssignmentDraft>;
  if (
    typeof item.providerId !== 'string' ||
    (item.providerId !== '' && !PROVIDERS.has(item.providerId)) ||
    typeof item.model !== 'string' ||
    item.model.length > 128 ||
    typeof item.sandbox !== 'string' ||
    !SANDBOXES.has(item.sandbox) ||
    typeof item.worktreePath !== 'string' ||
    item.worktreePath.length > 4096 ||
    item.worktreePath.includes('\0')
  ) {
    return false;
  }
  if (item.providerId === 'agy' && !['none', 'enabled'].includes(item.sandbox)) return false;
  if (item.providerId === 'opencode' || item.providerId === 'claude') {
    return item.sandbox === 'none';
  }
  if (item.providerId === '' && item.sandbox !== 'none') return false;
  return true;
}

export function readMaestroRoutingAssignments(
  draft: FlowDraftModel,
): Record<string, MaestroRouteAssignmentDraft> {
  const raw = draft.structuredFacts?.[MAESTRO_ROUTE_ASSIGNMENTS_FACT];
  if (!raw || raw.length > 150000) return {};
  try {
    const parsed = JSON.parse(raw) as {
      schemaVersion?: unknown;
      assignments?: unknown;
    };
    if (!parsed || parsed.schemaVersion !== 1 || !Array.isArray(parsed.assignments)) return {};
    if (parsed.assignments.length > 32) return {};
    const known = new Map(draft.steps.map((step) => [step.id, step.provider || '']));
    const result: Record<string, MaestroRouteAssignmentDraft> = {};
    for (const entry of parsed.assignments) {
      if (!entry || typeof entry !== 'object' || Array.isArray(entry)) return {};
      const item = entry as { taskId?: unknown } & Record<string, unknown>;
      const taskId = item.taskId;
      if (typeof taskId !== 'string' || !SAFE_ID.test(taskId)) return {};
      if (!known.has(taskId) || known.get(taskId) !== item.providerId) continue;
      if (!validAssignment(item) || Object.hasOwn(result, taskId)) return {};
      result[taskId] = {
        providerId: item.providerId,
        model: item.model,
        sandbox: item.sandbox,
        worktreePath: item.worktreePath,
      };
    }
    return result;
  } catch {
    return {};
  }
}

export function updateMaestroRoutingAssignment(
  draft: FlowDraftModel,
  taskId: string,
  assignment: MaestroRouteAssignmentDraft,
): FlowDraftModel {
  if (!SAFE_ID.test(taskId) || !draft.steps.some((step) => step.id === taskId)) {
    throw new Error('Unknown Maestro routing task');
  }
  if (!validAssignment(assignment)) throw new Error('Invalid Maestro routing assignment');
  const current = readMaestroRoutingAssignments(draft);
  current[taskId] = { ...assignment };
  const assignments = draft.steps
    .filter((step) => current[step.id] && SAFE_ID.test(step.id))
    .map((step) => ({ taskId: step.id, ...current[step.id] }));
  return {
    ...draft,
    steps: draft.steps.map((step) =>
      step.id === taskId ? { ...step, provider: assignment.providerId || undefined } : step,
    ),
    structuredFacts: {
      ...draft.structuredFacts,
      [MAESTRO_ROUTE_ASSIGNMENTS_FACT]: JSON.stringify({ schemaVersion: 1, assignments }),
    },
  };
}
