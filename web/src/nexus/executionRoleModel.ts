import type { RuntimeExecutionRoleView } from '../types';

export type ExecutionRoleTone = 'positive' | 'warning' | 'neutral' | 'danger';

export function executionRoleTone(view?: RuntimeExecutionRoleView | null): ExecutionRoleTone {
  const enforcement = String(view?.binding?.enforcement || '').toUpperCase();
  switch (enforcement) {
    case 'ENFORCED':
      return 'positive';
    case 'BEST_EFFORT':
      return 'warning';
    case 'UNSUPPORTED':
      return 'danger';
    default:
      return 'neutral';
  }
}

export function executionRoleSummary(view?: RuntimeExecutionRoleView | null): string {
  if (!view?.role) return 'Role unavailable';
  const enforcement = String(view.binding?.enforcement || 'UNVERIFIED').replaceAll('_', ' ');
  const strategy = String(view.binding?.activation_strategy || 'NEXUS_OVERLAY').replaceAll('_', ' ');
  return `${view.role.label} · ${enforcement} · ${strategy}`;
}
