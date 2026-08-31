import { describe, expect, it } from 'vitest';
import { canPlan, composerActions } from './composerModel';

describe('Composer readiness gate', () => {
  it.each(['MISSING', 'HYDRATING', 'STALE', 'FAILED'] as const)('%s blocks planning but keeps direct actions', (state) => {
    expect(canPlan(state)).toBe(false);
    expect(composerActions(state)).toMatchObject({ sendToAgent: true, copyPrompt: true, turnIntoFlow: false });
  });
  it('READY enables Flow planning', () => {
    expect(composerActions('READY')).toEqual({ sendToAgent: true, copyPrompt: true, turnIntoFlow: true });
  });
});
