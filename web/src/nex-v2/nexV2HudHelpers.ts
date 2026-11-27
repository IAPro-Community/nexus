import type { NexOperationalMood } from './motion/nexBehaviorState';
import type { NexV2HudAlert, NexV2HudState, NexV2TabId } from './nexV2Types';

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

export function resolveTopNexBubbleAlert(
  hudState: NexV2HudState | null,
  dismissedAlerts: Record<string, boolean>,
  labels?: { missionNeedsUserFallback?: string },
): { alert: NexV2HudAlert | null; totalPending: number } {
  if (!hudState) return { alert: null, totalPending: 0 };
  const alerts = (hudState.alerts ?? []).filter((a) => !a.acked && !dismissedAlerts[a.fingerprint]);
  const criticalAlerts = alerts.filter((a) => (a.severity || '').toUpperCase() === 'CRITICAL');
  const standardAlerts = alerts.filter((a) => (a.severity || '').toUpperCase() !== 'CRITICAL');

  const runtimeAlerts: NexV2HudAlert[] = (hudState.runtime_attention ?? [])
    .map((r) => ({
      fingerprint: `runtime:${r.runtime_id}:${r.fingerprint}`,
      category: 'runtime_attention',
      severity: (r.reason || '').toUpperCase() === 'ERROR' ? 'CRITICAL' : 'WARNING',
      provider: r.provider,
      profile: r.project_name || r.project_id,
      reason: [r.reason, r.prompt_kind].filter(Boolean).join(' · ') || undefined,
      acked: false,
      last_observed_at: r.updated_at || hudState.generated_at,
    }))
    .filter((a) => !dismissedAlerts[a.fingerprint]);

  const missionAlerts: NexV2HudAlert[] = (hudState.missions ?? [])
    .filter((m) => m.needs_user)
    .map((m) => ({
      fingerprint: `mission:${m.run_id}:${m.current_stage}`,
      category: 'mission_attention',
      severity: 'WARNING',
      provider: m.provider || 'Mission',
      profile: m.current_task_title || m.run_id,
      reason: m.intervention_reason || labels?.missionNeedsUserFallback || 'Intervenção necessária',
      acked: false,
      last_observed_at: m.updated_at || hudState.generated_at,
    }))
    .filter((a) => !dismissedAlerts[a.fingerprint]);

  const queue = [...criticalAlerts, ...runtimeAlerts, ...missionAlerts, ...standardAlerts];
  return { alert: queue[0] ?? null, totalPending: queue.length };
}
