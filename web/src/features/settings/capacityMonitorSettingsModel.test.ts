import { describe, expect, it } from 'vitest';
import { validateCapacityThresholds } from './capacityMonitorSettingsModel';
export { validateCapacityThresholds } from './capacityMonitorSettingsModel';

describe('validateCapacityThresholds', () => {
  it('accepts valid defaults', () => {
    expect(
      validateCapacityThresholds({
        warning_threshold: 20,
        critical_threshold: 10,
        exhausted_threshold: 0,
        recovery_threshold: 25,
      }),
    ).toBeNull();
  });

  it('rejects invalid ordering (critical >= warning)', () => {
    expect(
      validateCapacityThresholds({
        warning_threshold: 15,
        critical_threshold: 15,
        exhausted_threshold: 0,
        recovery_threshold: 25,
      }),
    ).toBe('Critical threshold must be < warning');
  });

  it('rejects recovery exceeding 100', () => {
    expect(
      validateCapacityThresholds({
        warning_threshold: 20,
        critical_threshold: 10,
        exhausted_threshold: 0,
        recovery_threshold: 105,
      }),
    ).toBe('Recovery threshold must be <= 100');
  });
});
