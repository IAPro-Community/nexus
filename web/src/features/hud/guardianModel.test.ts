import { describe, expect, it } from 'vitest';
import { guardianMood, normalizeHudAppearance } from './guardianModel';

describe('Nexus Guardian presentation', () => {
  it('recovers invalid or legacy settings to default mascot mode', () => {
    expect(normalizeHudAppearance(undefined)).toBe('MASCOT');
    expect(normalizeHudAppearance('AUTO')).toBe('MASCOT');
    expect(normalizeHudAppearance('BAR')).toBe('BAR');
    expect(normalizeHudAppearance('MASCOT')).toBe('MASCOT');
    expect(normalizeHudAppearance('TRAY')).toBe('TRAY');
  });

  it('prioritizes offline and real interventions above routine work', () => {
    expect(guardianMood(true, true, 4)).toBe('offline');
    expect(guardianMood(false, true, 4)).toBe('warning');
    expect(guardianMood(false, false, 2)).toBe('working');
    expect(guardianMood(false, false, 0)).toBe('idle');
  });

  it('never announces a warning merely because no agents are active', () => {
    expect(guardianMood(false, false, 0)).not.toBe('warning');
  });
});
