import type { AttentionGroup, AttentionItem } from '../../types';
import { asArray } from '../../lib/safeArray';
import type { HudAlert, HudMission, HudProvider, HudRuntimeAttention, HudState } from './hudModel';

export type MascotMood = 'idle' | 'working' | 'attention' | 'critical' | 'offline';
export type MascotTarget = 'missions' | 'agents' | 'usage' | 'nexus';

export interface MascotAttention {
  id: string;
  titleKey: string;
  detailKey: string;
  severity: 'warning' | 'critical';
  target: MascotTarget;
  needsUser: boolean;
  projectId?: string;
  recordId?: string;
}

const TERMINAL_MISSION = /^(COMPLETED|CANCELED|CANCELLED|FAILED|STOPPED)(_|$)/i;
const ACTIVE_MISSION =
  /^(ALLOCATING|COMPILING|EXECUTING|TESTING|REVIEWING|VERIFYING|REMEDIATING)$/i;
const AWAITING_AGENT = /^(WAITING_INPUT|AWAITING_INPUT|NEEDS_USER|BLOCKED_NEEDS_USER)$/i;
const FAILED_AGENT = /^(FAILED|ERROR|RECOVERABLE)$/i;
// Keep the v1 storage key so existing dismissed attention items remain snoozed.
const MASCOT_SNOOZE_STORAGE_KEY = 'iapro:nexus:companion:snooze:v1';
const MAX_SNOOZED_ITEMS = 80;

const alertIsSnoozed = (alert: HudAlert, now: number): boolean => {
  if (!alert.snoozed_until) return false;
  const until = Date.parse(alert.snoozed_until);
  return Number.isFinite(until) && until > now;
};

const missionAttention = (item: AttentionItem): MascotAttention => ({
  id: `mission:${item.project_id}:${item.mission_id}:${item.intervention?.id || 'state'}:v${item.intervention?.version ?? 0}`,
  titleKey: 'hud.mascot.missionTitle',
  detailKey: 'hud.mascot.missionDetail',
  severity: 'critical',
  target: 'missions',
  needsUser: item.level === 'REQUIRE_USER',
  projectId: item.project_id,
  recordId: item.mission_id,
});

const legacyMissionAttention = (mission: HudMission): MascotAttention => ({
  id: `mission:${mission.project_id}:${mission.run_id}`,
  titleKey: 'hud.mascot.missionTitle',
  detailKey: 'hud.mascot.missionDetail',
  severity: 'critical',
  target: 'missions',
  needsUser: true,
  projectId: mission.project_id,
  recordId: mission.run_id,
});

const runtimeAttention = (runtime: HudRuntimeAttention): MascotAttention | undefined => {
  const fingerprint = runtime.fingerprint || runtime.runtime_id;
  if (!fingerprint) return undefined;
  const approval = (runtime.reason || '').toUpperCase() === 'APPROVAL';
  return {
    id: `runtime:${fingerprint}`,
    titleKey: approval ? 'hud.mascot.runtimeApprovalTitle' : 'hud.mascot.agentTitle',
    detailKey: approval ? 'hud.mascot.runtimeApprovalDetail' : 'hud.mascot.agentDetail',
    severity: 'warning',
    target: 'agents',
    needsUser: true,
    projectId: runtime.project_id,
    recordId: runtime.runtime_id,
  };
};

const capacityAttention = (alert: HudAlert, projectId?: string): MascotAttention | undefined => {
  const severity = (alert.severity || '').toUpperCase();
  if (alert.acked || severity === 'INFO') return undefined;
  const critical = severity === 'CRITICAL';
  return {
    id: `capacity:${alert.fingerprint}`,
    titleKey: critical ? 'hud.mascot.capacityCriticalTitle' : 'hud.mascot.capacityWarningTitle',
    detailKey: 'hud.mascot.capacityDetail',
    severity: critical ? 'critical' : 'warning',
    target: 'usage',
    needsUser: false,
    projectId,
    recordId: alert.fingerprint,
  };
};

export function projectMascotAttention(
  state: HudState | undefined,
  attentionGroup?: AttentionGroup,
  now = Date.now(),
): MascotAttention[] {
  if (!state && !attentionGroup) return [];
  const items: MascotAttention[] = [];

  // The Attention Center is the authority for Mission intervention and failure.
  if (attentionGroup) {
    items.push(
      ...asArray<AttentionItem>(attentionGroup.needs_you)
        .filter((item) => item.level === 'REQUIRE_USER')
        .map(missionAttention),
    );
    items.push(
      ...asArray<AttentionItem>(attentionGroup.failed).map((item) => ({
        ...missionAttention(item),
        id: `failed:${item.project_id}:${item.mission_id}`,
        titleKey: 'hud.mascot.missionFailedTitle',
        detailKey: 'hud.mascot.missionFailedDetail',
        severity: 'critical' as const,
        needsUser: false,
      })),
    );
  } else {
    items.push(
      ...asArray<HudMission>(state?.missions)
        .filter((mission) => mission.needs_user && !TERMINAL_MISSION.test(mission.state))
        .map(legacyMissionAttention),
    );
  }

  const runtimeAgentIds = new Set<string>();
  for (const runtime of asArray<HudRuntimeAttention>(state?.runtime_attention)) {
    const item = runtimeAttention(runtime);
    if (!item) continue;
    if (runtime.agent_id) runtimeAgentIds.add(`${runtime.project_id || ''}:${runtime.agent_id}`);
    items.push(item);
  }

  for (const agent of asArray<HudState['agents'][number]>(state?.agents)) {
    const status = (agent.status || '').toUpperCase();
    const runtimeKey = `${agent.project_id}:${agent.id}`;
    if (runtimeAgentIds.has(runtimeKey)) continue;
    if (AWAITING_AGENT.test(status)) {
      items.push({
        id: `agent:${agent.project_id}:${agent.id}:${status}`,
        titleKey: 'hud.mascot.agentTitle',
        detailKey: 'hud.mascot.agentDetail',
        severity: 'warning',
        target: 'agents',
        needsUser: true,
        projectId: agent.project_id,
        recordId: agent.id,
      });
    } else if (FAILED_AGENT.test(status)) {
      items.push({
        id: `agent-error:${agent.project_id}:${agent.id}:${status}`,
        titleKey: 'hud.mascot.agentFailedTitle',
        detailKey: 'hud.mascot.agentFailedDetail',
        severity: 'critical',
        target: 'agents',
        needsUser: false,
        projectId: agent.project_id,
        recordId: agent.id,
      });
    }
  }

  for (const alert of asArray<HudAlert>(state?.alerts)) {
    if (alertIsSnoozed(alert, now)) continue;
    const item = capacityAttention(alert, state?.project_id);
    if (item) items.push(item);
  }

  const priority = { critical: 0, warning: 1 };
  return [...new Map(items.map((item) => [item.id, item])).values()].sort(
    (left, right) =>
      priority[left.severity] - priority[right.severity] || left.id.localeCompare(right.id),
  );
}

export function mascotMood(
  state: HudState | undefined,
  attention: MascotAttention[],
  offline: boolean,
): MascotMood {
  if (offline || !state) return 'offline';
  const providers = asArray<HudProvider>(state.providers);
  if (
    providers.length > 0 &&
    providers.every((provider) => (provider.availability || '').toUpperCase() === 'UNAVAILABLE')
  ) {
    return 'offline';
  }
  if (attention.some((item) => item.severity === 'critical')) return 'critical';
  if (attention.length > 0) return 'attention';
  if (
    asArray<HudState['agents'][number]>(state.agents).some(
      (agent) => (agent.status || '').toUpperCase() === 'WORKING',
    ) ||
    asArray<HudMission>(state.missions).some((mission) => ACTIVE_MISSION.test(mission.state || ''))
  ) {
    return 'working';
  }
  return 'idle';
}

export function mascotMotionAllowed(mood: MascotMood, requested: boolean): boolean {
  return requested && mood !== 'offline';
}

export function readMascotSnoozes(
  storage: Pick<Storage, 'getItem'>,
  now = Date.now(),
): Record<string, number> {
  try {
    const raw = JSON.parse(storage.getItem(MASCOT_SNOOZE_STORAGE_KEY) || '{}') as unknown;
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return {};
    return Object.fromEntries(
      Object.entries(raw)
        .filter(
          ([key, until]) =>
            key.length <= 200 &&
            typeof until === 'number' &&
            Number.isFinite(until) &&
            until > now &&
            until < now + 31 * 24 * 60 * 60 * 1000,
        )
        .slice(-MAX_SNOOZED_ITEMS),
    );
  } catch {
    return {};
  }
}

export function snoozeMascotAttention(
  storage: Pick<Storage, 'getItem' | 'setItem'>,
  id: string,
  now = Date.now(),
  durationMs = 60 * 60 * 1000,
): void {
  if (!id || id.length > 200 || !Number.isFinite(durationMs) || durationMs <= 0) return;
  try {
    const entries = readMascotSnoozes(storage, now);
    entries[id] = now + durationMs;
    storage.setItem(
      MASCOT_SNOOZE_STORAGE_KEY,
      JSON.stringify(Object.fromEntries(Object.entries(entries).slice(-MAX_SNOOZED_ITEMS))),
    );
  } catch {
    // The caller keeps the current window's snooze in memory when storage is unavailable.
  }
}

export function nextMascotBubble(
  items: MascotAttention[],
  snoozed: Record<string, number>,
  now = Date.now(),
  dismissed: Record<string, boolean> = {},
): MascotAttention | undefined {
  return items.find(
    (item) => !dismissed[item.id] && (!snoozed[item.id] || snoozed[item.id] <= now),
  );
}
