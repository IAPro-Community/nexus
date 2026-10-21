export type HudAppearance = 'BAR' | 'MASCOT' | 'TRAY';
export type GuardianMood = 'idle' | 'working' | 'warning' | 'offline';

export function normalizeHudAppearance(value: unknown): HudAppearance {
  return value === 'BAR' || value === 'MASCOT' || value === 'TRAY' ? value : 'BAR';
}

/** Present only canonical HUD evidence, never infer an alarm from stale quota. */
export function guardianMood(
  offline: boolean,
  attention: boolean,
  workingAgents: number,
): GuardianMood {
  if (offline) return 'offline';
  if (attention) return 'warning';
  if (workingAgents > 0) return 'working';
  return 'idle';
}
