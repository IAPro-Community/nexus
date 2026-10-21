import type { HudAlert, HudState } from './hudModel';
import type { AttentionGroup } from '../../types';

export type CompanionMood = 'idle' | 'working' | 'attention' | 'critical' | 'offline' | 'success';
export type CompanionTarget = 'missions' | 'agents' | 'usage' | 'nexus';
export interface CompanionAttention {
  id: string;
  title: string;
  detail: string;
  severity: 'info' | 'warning' | 'critical';
  target: CompanionTarget;
  needsUser: boolean;
  projectId?: string;
  recordId?: string;
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
  attentionGroup?: AttentionGroup,
): CompanionAttention[] {
  if (!state && !attentionGroup) return [];
  const items: CompanionAttention[] = [];

  // The Attention Center owns human-intervention classification. If it is
  // reachable, do not override it with older Mission projection heuristics.
  for (const input of attentionGroup?.needs_you || []) {
    if (input.level !== 'REQUIRE_USER') continue;
    items.push({
      id: `mission:${input.project_id}:${input.mission_id}:${input.intervention?.id || 'state'}:v${input.intervention?.version ?? 0}`,
      title: 'Uma missão precisa de resposta',
      detail: 'Há uma intervenção aguardando sua decisão.',
      severity: 'critical',
      target: 'missions',
      needsUser: true,
      projectId: input.project_id,
      recordId: input.mission_id,
    });
  }
  for (const mission of attentionGroup ? [] : state?.missions || []) {
    if (!mission.needs_user || TERMINAL.test(mission.state)) continue;
    items.push({
      id: `mission:${mission.project_id}:${mission.run_id}`,
      title: 'Uma missão precisa de resposta',
      detail: mission.intervention_reason?.trim() || 'Há uma intervenção aguardando sua decisão.',
      severity: 'critical',
      target: 'missions',
      needsUser: true,
      projectId: mission.project_id,
      recordId: mission.run_id,
    });
  }
  for (const agent of state?.agents || []) {
    if (!AWAITING.test(agent.status)) continue;
    items.push({
      id: `agent:${agent.project_id}:${agent.id}:${agent.status}`,
      title: 'Um agente aguarda você',
      detail: 'Há uma solicitação de entrada para continuar.',
      severity: 'warning',
      target: 'agents',
      needsUser: true,
      projectId: agent.project_id,
      recordId: agent.id,
    });
  }
  for (const alert of state?.alerts || []) {
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
      projectId: state?.project_id,
    });
  }
  const order = { critical: 0, warning: 1, info: 2 };
  const unique = [...new Map(items.map((item) => [item.id, item])).values()];
  return unique.sort(
    (a, b) => order[a.severity] - order[b.severity] || a.id.localeCompare(b.id),
  );
}

export function companionMood(
  state: HudState | undefined,
  attention: CompanionAttention[],
  offline: boolean,
): CompanionMood {
  if (offline || !state) return 'offline';
  if (attention.some((item) => item.severity === 'critical')) return 'critical';
  if (attention.length > 0) return 'attention';
  if (
    state.agents.some((agent) => agent.status.toUpperCase() === 'WORKING') ||
    state.missions.some((mission) => !TERMINAL.test(mission.state))
  ) {
    return 'working';
  }
  return 'idle';
}

export const COMPANION_DISMISS_KEY = 'iapro:nexus:companion:snooze:v1';
const MAX_SNOOZED = 80;
export function readCompanionSnoozes(
  storage: Pick<Storage, 'getItem'>,
  now = Date.now(),
): Record<string, number> {
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
  storage.setItem(
    COMPANION_DISMISS_KEY,
    JSON.stringify(Object.fromEntries(Object.entries(entries).slice(-MAX_SNOOZED))),
  );
}

export function nextCompanionBubble(
  items: CompanionAttention[],
  snoozed: Record<string, number>,
  now = Date.now(),
): CompanionAttention | undefined {
  return items.find((item) => !snoozed[item.id] || snoozed[item.id] <= now);
}
