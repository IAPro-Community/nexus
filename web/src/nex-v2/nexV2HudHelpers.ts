import type { NexOperationalMood } from './motion/nexBehaviorState';
import type { NexV2HudState, NexV2TabId } from './nexV2Types';

export function countAttentionItems(hudState: NexV2HudState | null): number {
  if (!hudState) return 0;
  const activeAlerts = (hudState.alerts ?? []).filter((a) => !a.acked).length;
  const runtimeAttention = (hudState.runtime_attention ?? []).length;
  const blockedMissions = (hudState.missions ?? []).filter((m) => m.needs_user).length;
  return activeAlerts + runtimeAttention + blockedMissions;
}

export function hasCriticalAttention(hudState: NexV2HudState | null): boolean {
  if (!hudState) return false;
  return (hudState.alerts ?? []).some(
    (a) => !a.acked && (a.severity || '').toUpperCase() === 'CRITICAL',
  );
}

export function resolveLowestQuotaPercent(hudState: NexV2HudState | null): number | null {
  if (!hudState) return null;
  let minVal: number | null = null;
  for (const provider of hudState.providers ?? []) {
    if (
      provider.limiting_window &&
      typeof provider.limiting_window.remaining_percent === 'number'
    ) {
      const pct = Math.round(provider.limiting_window.remaining_percent);
      minVal = minVal === null ? pct : Math.min(minVal, pct);
    }
    for (const win of provider.windows ?? []) {
      if (typeof win.remaining_percent === 'number') {
        const pct = Math.round(win.remaining_percent);
        minVal = minVal === null ? pct : Math.min(minVal, pct);
      }
    }
  }
  return minVal;
}

export function resolveDefaultNexV2Tab(hudState: NexV2HudState | null): NexV2TabId {
  return countAttentionItems(hudState) > 0 ? 'attention' : 'resources';
}

export type QuotaTone = 'ok' | 'warning' | 'critical' | 'unknown';

export function resolveQuotaTone(percent: number | undefined | null): QuotaTone {
  if (typeof percent !== 'number' || !Number.isFinite(percent)) return 'unknown';
  if (percent < 20) return 'critical';
  if (percent <= 50) return 'warning';
  return 'ok';
}

export function resolveMoodTranslationKey(mood: NexOperationalMood): string {
  switch (mood) {
    case 'working':
      return 'nexV2.moodWorking';
    case 'attention':
      return 'nexV2.moodAttention';
    case 'critical':
      return 'nexV2.moodCritical';
    case 'offline':
      return 'nexV2.moodOffline';
    default:
      return 'nexV2.moodIdle';
  }
}
