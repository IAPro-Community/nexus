import type { CostBudgetAssessment } from '../../types';

// Observed zero is meaningful, but a missing/estimated amount is UNKNOWN.
// Never convert the legacy mixed aggregate into an apparent billed amount.
export function reportedCostForDisplay(
  budget: Pick<CostBudgetAssessment, 'reported_cost_observations' | 'reported_cost_microusd'>,
): number | null {
  const value = budget.reported_cost_microusd;
  if (budget.reported_cost_observations <= 0 || !Number.isSafeInteger(value) || value < 0) {
    return null;
  }
  return value;
}
