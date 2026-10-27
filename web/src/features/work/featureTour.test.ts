import { describe, expect, it } from 'vitest';
import { availableTourSteps } from '../../app/tour/tour';
import { featureTourSteps, featureTourStorageKey, parseFeatureTourProgress } from './featureTour';

describe('Composer and Flow contextual tours', () => {
  it('provides real distinct step identifiers per surface', () => {
    expect(Object.keys(featureTourSteps)).toEqual([
      'composer-start', 'composer-session', 'composer-offline',
      'flow-library', 'flow-run', 'flow-editor',
    ]);
    for (const [surface, steps] of Object.entries(featureTourSteps)) {
      expect(steps.length, surface).toBeGreaterThan(0);
      const ids = new Set<string>();
      for (const step of steps) {
        expect(step.target).toMatch(/^\[data-tour="[a-z0-9-]+"\]$/);
        expect(step.title).toMatch(/^tour\./);
        expect(step.body).toMatch(/^tour\./);
        expect(ids.has(step.id), `${surface} contains duplicate step ${step.id}`).toBe(false);
        ids.add(step.id);
      }
    }
  });

  it('skips absent or hidden targets without inventing features', () => {
    const steps = featureTourSteps['flow-editor'];
    const visible = availableTourSteps(steps, (selector) =>
      selector === '[data-tour="flow-goal"]' || selector === '[data-tour="flow-canvas"]');
    expect(visible.map((step) => step.id)).toEqual(['goal', 'canvas']);
  });

  it('persists independent progress for each project and area', () => {
    const key = featureTourStorageKey('client/alpha', 'composer-start');
    expect(key).toContain('client%2Falpha');
    expect(key).not.toEqual(featureTourStorageKey('client/beta', 'composer-start'));
    expect(key).not.toEqual(featureTourStorageKey('client/alpha', 'flow-editor'));
  });

  it('restores only safe, bounded, numeric progress', () => {
    expect(parseFeatureTourProgress(null, 5)).toBe(0);
    expect(parseFeatureTourProgress('', 5)).toBe(0);
    expect(parseFeatureTourProgress('2', 5)).toBe(2);
    expect(parseFeatureTourProgress('999', 5)).toBe(4);
    expect(parseFeatureTourProgress('-10', 5)).toBe(0);
    expect(parseFeatureTourProgress('NaN', 5)).toBe(0);
    expect(parseFeatureTourProgress('2.5', 5)).toBe(0);
    expect(parseFeatureTourProgress('2', 0)).toBe(0);
  });
});
