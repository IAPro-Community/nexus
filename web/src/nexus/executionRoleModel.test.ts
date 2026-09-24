import { describe, expect, it } from 'vitest';
import type { RuntimeExecutionRoleView } from '../types';
import { executionRoleSummary, executionRoleTone } from './executionRoleModel';

function view(enforcement: string): RuntimeExecutionRoleView {
  return {
    persisted: true,
    role: {
      id: 'review',
      label: 'Review',
      base_agent_mode_id: 'reviewer',
      access_intent: 'READ_ONLY',
      default_scope: 'RUNTIME',
    },
    binding: {
      runtime_generation_id: 'gen_1',
      agent_id: 'agt_1',
      role_id: 'review',
      scope: 'RUNTIME',
      activation_strategy: 'NEXUS_OVERLAY',
      enforcement,
    },
  };
}

describe('executionRoleModel', () => {
  it('does not present best-effort as enforced', () => {
    expect(executionRoleTone(view('BEST_EFFORT'))).toBe('warning');
    expect(executionRoleSummary(view('BEST_EFFORT'))).toContain('BEST EFFORT');
  });

  it('distinguishes enforced and unsupported states', () => {
    expect(executionRoleTone(view('ENFORCED'))).toBe('positive');
    expect(executionRoleTone(view('UNSUPPORTED'))).toBe('danger');
  });
});
