import type { HudAgent, HudAlert, HudMission, HudState } from './hudModel';

export type CompanionMood = 'idle' | 'working' | 'attention' | 'critical' | 'offline' | 'success';
export type CompanionTarget = 'missions' | 'agents' | 'usage' | 'nexus';
export interface CompanionAttention {
  id: string;
  title: string;
  detail: string;
  severity: 'info' | 'warning' | 'critical';
  target: CompanionTarget;
  needsUser: boolean;
}

const TERMINAL = /^(COMPLETED|CANCELED|CANCELLED|FAILED|STOPPED)(_|$)/i;
const AWAITING = /^(WAITING_INPUT|AWAITING_INPUT|NEEDS_USER|BLOCKED_NEEDS_USER)$/i;
const isSnoozed = (alert: HudAlert, now: number) => {
  if (!alert.snoozed_until) return false;
  const until = Date.parse(alert.snoozed_until);
  return Number.isFinite(until) && until > now;
};

export function projectCompanionAttention(
  state: HudState | undefined,
  now = Date.now(),
): CompanionAttention[] {
  if (!state) return [];
  const items: CompanionAttention[] = [];

  for (const mission of state.missions) {
    if (!mission.needs_user || TERMINAL.test(mission.state)) continue;
    items.push({
      id: `mission:${mission.project_id}:${mission.run_id}`,
      title: 'Uma missão precisa de resposta',
      detail: mission.intervention_reason?.trim() || 'Há uma intervenção aguardando sua decisão.',
      severity: 'critical',
      target: 'missions',
      needsUser: true,
    });
  }
  for (const agent of state.agents) {
    if (!AWAITING.test(agent.status)) continue;
    items.push({
      id: `agent:${agent.project_id}:${agent.id}:${agent.status}`,
      title: 'Um agente aguarda você',
      detail: 'Há uma solicitação de entrada para continuar.',
      severity: 'warning',
      target: 'agents',
      needsUser: true,
    });
  }
  for (const alert of state.alerts) {
    if (alert.acked || isSnoozed(alert, now)) continue;
    if (alert.severity.toUpperCase() === 'INFO') continue;
    const critical = alert.severity.toUpperCase() === 'CRITICAL';
    items.push({
      id: `capacity:${alert.fingerprint}`,
      title: critical ? 'Capacidade crítica detectada' : 'Capacidade precisa de atenção',
      detail: 'Consulte as contas e janelas disponíveis em Usage.',
      severity: critical ? 'critical' : 'warning',
      target: 'usage',
      needsUser: false,
    });
  }
  const order = { critical: 0, warning: 1, info: 2 };
  return items.sort((a, b) => order[a.severity] - order[b.severity] || a.id.localeCompare(b.id));
}

export function companionMood(
  state: HudState | undefined,
  attention: CompanionAttention[],
  offline: boolean,
): CompanionMood {
  if (offline || !state) return 'offline';
  if (attention.some((item) => item.severity === 'critical')) return 'critical';
  if (attention.length > 0) return 'attention';
  if (state.agents.some((agent) => agent.status.toUpperCase() === 'WORKING') ||
      state.missions.some((mission) => !TERMINAL.test(mission.state))) return 'working';
  return 'idle';
}

export const COMPANION_DISMISS_KEY = 'iapro:nexus:companion:snooze:v1';
const MAX_SNOOZED = 80;
export function readCompanionSnoozes(storage: Pick<Storage, 'getItem'>, now = Date.now()): Record<string, number> {
  try {
    const raw = JSON.parse(storage.getItem(COMPANION_DISMISS_KEY) || '{}') as unknown;
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return {};
    return Object.fromEntries(
      Object.entries(raw).filter(([key, until]) =>
        key.length <= 200 && typeof until === 'number' &&
        Number.isFinite(until) && until > now && until < now + 31 * 86400000,
      ).slice(-MAX_SNOOZED),
    );
  } catch {
    return {};
  }
}

export function snoozeCompanionAttention(
  storage: Pick<Storage, 'getItem' | 'setItem'>,
  id: string,
  now = Date.now(),
  durationMs = 60 * 60 * 1000,
): void {
  if (id.length > 200) return;
  const entries = readCompanionSnoozes(storage, now);
  entries[id] = now + durationMs;
  storage.setItem(COMPANION_DISMISS_KEY, JSON.stringify(
    Object.fromEntries(Object.entries(entries).slice(-MAX_SNOOZED)),
  ));
}

export function nextCompanionBubble(
  items: CompanionAttention[],
  snoozed: Record<string, number>,
  now = Date.now(),
): CompanionAttention | undefined {
  return items.find((item) => !snoozed[item.id] || snoozed[item.id] <= now);
}
