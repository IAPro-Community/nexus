import { describe, expect, it } from 'vitest';
import { NEX_IDLE_ACTIONS } from './nexIdleDirector';
import { resolveNexRigPresentation } from './nexRigProjection';

describe('NEX procedural rig projection', () => {
  it('keeps the operational and completion expressions authoritative', () => {
    for (const behavior of ['thinking', 'attention', 'error', 'resolved'] as const) {
      expect(resolveNexRigPresentation(behavior, 'idle_orbit_shuffle', true))
        .toEqual({ mode: 'idle', expression: behavior });
    }
  });

  it('uses the existing true orbit only for approved idle orbit gestures', () => {
    expect(resolveNexRigPresentation('ambient', 'idle_orbit_shuffle', true))
      .toEqual({ mode: 'orbit', expression: 'ambient' });
    expect(resolveNexRigPresentation('ambient', 'idle_orbit_breathe', true).mode).toBe('orbit');
    expect(resolveNexRigPresentation('ambient', null, true).mode).toBe('idle');
  });

  it('uses only expressions available in the approved rig', () => {
    for (const action of NEX_IDLE_ACTIONS) {
      const presentation = resolveNexRigPresentation('ambient', action, true);
      expect(['ambient', 'look', 'greet', 'rest']).toContain(presentation.expression);
    }
    expect(resolveNexRigPresentation('rest', 'idle_wake', true).expression).toBe('greet');
  });

  it('disables orbit and decorative expression when motion is unavailable', () => {
    expect(resolveNexRigPresentation('ambient', 'idle_orbit_shuffle', false))
      .toEqual({ mode: 'idle', expression: 'still' });
    expect(resolveNexRigPresentation('still', 'idle_orbit_shuffle', true))
      .toEqual({ mode: 'idle', expression: 'still' });
  });
});
