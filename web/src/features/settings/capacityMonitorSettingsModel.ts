import type { CapacityPreferences } from '../../types';

export function validateCapacityThresholds(prefs: Partial<CapacityPreferences>): string | null {
  const warning = prefs.warning_threshold ?? 20;
  const critical = prefs.critical_threshold ?? 10;
  const exhausted = prefs.exhausted_threshold ?? 0;
  const recovery = prefs.recovery_threshold ?? 25;

  if (exhausted < 0 || exhausted >= critical) {
    return 'Exhausted threshold must be >= 0 and < critical';
  }
  if (critical >= warning) {
    return 'Critical threshold must be < warning';
  }
  if (warning >= recovery) {
    return 'Warning threshold must be < recovery';
  }
  if (recovery > 100) {
    return 'Recovery threshold must be <= 100';
  }
  return null;
}
