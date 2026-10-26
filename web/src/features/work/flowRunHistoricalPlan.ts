import type { PlanRevision, WorkPlan } from '../../types';
import { normalizeWorkPlan } from '../../nexus/workPlan';

/**
 * A Mission is bound to a frozen WorkPlan revision. A later edit of the same
 * WorkPlan must never rewrite historical Composer decisions in the run UI.
 * If the original revision cannot be loaded, return null rather than showing
 * current data as approved history.
 */
export function selectRunHistoricalPlan(
  detail: { plan: WorkPlan; revisions: PlanRevision[] },
  runRevision: number,
): WorkPlan | null {
  if (runRevision <= 0) return null;
  if (detail.plan.current_revision === runRevision) return detail.plan;
  const frozen = (detail.revisions || []).find((revision) => revision.revision === runRevision);
  if (!frozen?.snapshot_json) return null;
  try {
    const plan = normalizeWorkPlan(JSON.parse(frozen.snapshot_json));
    if (plan.id !== detail.plan.id || plan.project_id !== detail.plan.project_id ||
        plan.current_revision !== runRevision) return null;
    return plan;
  } catch {
    return null;
  }
}
