import { asArray } from '../../lib/safeArray';
import type { AttentionGroup } from '../../types';

export type HudMode = 'AUTO' | 'CAPACITY' | 'RESOURCE' | 'AGENT' | 'MISSION' | 'ALERT' | 'MINIMAL';

export type HudSurfaceMode = 'docked' | 'floating';
export type HudDock = 'top' | 'bottom';

export interface HudWindow {
  group?: string;
  kind: string;
  label?: string;
  remaining_percent?: number;
  reset_description?: string;
  reset_at?: string;
  status: string;
}

export interface HudProvider {
  provider: string;
  profile: string;
  display_name?: string;
  health: string;
  availability: string;
  capacity_state: string;
  source?: string;
  freshness: string;
  last_updated?: string;
  rate_limited: boolean;
  limiting_window?: HudWindow;
  windows: HudWindow[];
}

export interface HudRecommendation {
  provider: string;
  profile: string;
  display_name?: string;
  score: number;
  confidence: string;
  explanation: string;
  alternatives?: string[];
  context: string;
}

export interface HudAgent {
  id: string;
  project_id: string;
  name: string;
  role?: string;
  status: string;
  provider?: string;
  profile?: string;
  model?: string;
  runtime_id?: string;
  started_at?: string;
}

export interface HudMission {
  run_id: string;
  project_id: string;
  state: string;
  current_stage: string;
  current_task_id?: string;
  current_task_title?: string;
  provider?: string;
  profile?: string;
  completed_packages: number;
  total_packages: number;
  needs_user: boolean;
  intervention_reason?: string;
  updated_at: string;
}

export interface HudAlert {
  fingerprint: string;
  category: string;
  severity: 'INFO' | 'WARNING' | 'CRITICAL' | string;
  provider?: string;
  profile?: string;
  reason?: string;
  remaining_percent?: number;
  acked: boolean;
  snoozed_until?: string;
  last_observed_at: string;
}

export interface HudRuntimeAttention {
  runtime_id: string;
  agent_id?: string;
  project_id?: string;
  project_name?: string;
  provider: string;
  reason: string;
  prompt_kind: 'yn' | 'choice' | 'free_text' | string;
  fingerprint: string;
  updated_at: string;
}

export interface HudCostKindTotal {
  kind: 'PROVIDER_REPORTED' | 'API_EQUIVALENT_ESTIMATE' | 'INFERRED' | 'UNKNOWN' | string;
  microusd: number;
  count: number;
}

export interface HudCostSummary {
  evidence: 'EMPTY' | 'ATTRIBUTION_ONLY' | 'PARTIAL' | 'MEASURED' | string;
  token_evidence: 'EMPTY' | 'ATTRIBUTION_ONLY' | 'PARTIAL' | 'MEASURED' | string;
  known_microusd: number;
  unknown_observation_count: number;
  budget_level: string;
  by_kind: HudCostKindTotal[];
  generated_at?: string;
}

export interface HudState {
  generated_at: string;
  project_id?: string;
  providers: HudProvider[];
  recommendation?: HudRecommendation;
  agents: HudAgent[];
  missions: HudMission[];
  alerts: HudAlert[];
  runtime_attention?: HudRuntimeAttention[];
  cost?: HudCostSummary;
  primary: { kind: string; id?: string };
  system: {
    capacity: string;
    resources: string;
    agents: string;
    missions: string;
    cost: string;
    cache_ttl_millis: number;
  };
}

export interface HudResponse {
  enabled: boolean;
  rollout: 'experimental' | 'beta' | 'stable' | string;
  state?: HudState;
}

/** Normalize nullable Go slices once at the desktop HUD network boundary. */
export function normalizeHudResponse(response: HudResponse): HudResponse {
  if (!response.state) return { ...response, state: undefined };
  return {
    ...response,
    state: {
      ...response.state,
      providers: asArray(response.state.providers),
      agents: asArray(response.state.agents),
      missions: asArray(response.state.missions),
      alerts: asArray(response.state.alerts),
      runtime_attention: asArray(response.state.runtime_attention),
      system: {
        capacity: response.state.system?.capacity || 'UNKNOWN',
        resources: response.state.system?.resources || 'UNKNOWN',
        agents: response.state.system?.agents || 'UNKNOWN',
        missions: response.state.system?.missions || 'UNKNOWN',
        cost: response.state.system?.cost || 'UNKNOWN',
        cache_ttl_millis: response.state.system?.cache_ttl_millis || 15_000,
      },
    },
  };
}

export function normalizeAttentionGroup(
  group: AttentionGroup | undefined,
): AttentionGroup | undefined {
  if (!group) return undefined;
  const needsYou = asArray<AttentionGroup['needs_you'][number]>(group.needs_you);
  return {
    ...group,
    needs_you: needsYou,
    completed: asArray(group.completed),
    failed: asArray(group.failed),
    all_items: asArray(group.all_items),
    total_needs:
      typeof group.total_needs === 'number' && Number.isFinite(group.total_needs)
        ? group.total_needs
        : needsYou.length,
  };
}

export type NativeHudStatus = 'loading' | 'offline' | 'disabled' | 'ready';

export function resolveNativeHudStatus(
  enabled: boolean | undefined,
  hasState: boolean,
  offline: boolean,
): NativeHudStatus {
  if (enabled === false) return 'disabled';
  // A previous HTTP snapshot is not proof that the native HUD is online.
  if (offline) return 'offline';
  if (enabled && hasState) return 'ready';
  if (enabled === true) return 'offline';
  return 'loading';
}

export interface HudPresentationPreferences {
  version: 1;
  enabled: boolean;
  mode: HudMode;
  surfaceMode: HudSurfaceMode;
  dock: HudDock;
  compact: boolean;
  showCapacity: boolean;
  showAgents: boolean;
  showMissions: boolean;
  showCost: boolean;
  showRecommendation: boolean;
  companionMotion: boolean;
  privacyMode: boolean;
  floatingPosition: { x: number; y: number };
}

export const HUD_PREFERENCES_KEY = 'iapro:nexus:hud:presentation:v1';

export const defaultHudPreferences: HudPresentationPreferences = {
  version: 1,
  enabled: false,
  mode: 'AUTO',
  surfaceMode: 'docked',
  dock: 'bottom',
  compact: true,
  showCapacity: true,
  showAgents: true,
  showMissions: true,
  showCost: true,
  showRecommendation: true,
  companionMotion: true,
  privacyMode: false,
  floatingPosition: { x: 24, y: 24 },
};

const modes = new Set<HudMode>([
  'AUTO',
  'CAPACITY',
  'RESOURCE',
  'AGENT',
  'MISSION',
  'ALERT',
  'MINIMAL',
]);

export const normalizeHudPreferences = (
  raw: Partial<HudPresentationPreferences> | null | undefined,
): HudPresentationPreferences => ({
  ...defaultHudPreferences,
  ...(raw || {}),
  version: 1,
  mode: raw?.mode && modes.has(raw.mode) ? raw.mode : defaultHudPreferences.mode,
  companionMotion:
    typeof raw?.companionMotion === 'boolean'
      ? raw.companionMotion
      : defaultHudPreferences.companionMotion,
  surfaceMode:
    raw?.surfaceMode === 'floating' || raw?.surfaceMode === 'docked'
      ? raw.surfaceMode
      : defaultHudPreferences.surfaceMode,
  dock: raw?.dock === 'top' || raw?.dock === 'bottom' ? raw.dock : defaultHudPreferences.dock,
  floatingPosition: {
    ...defaultHudPreferences.floatingPosition,
    ...(raw?.floatingPosition || {}),
  },
});

export const loadHudPreferences = (): HudPresentationPreferences => {
  if (typeof window === 'undefined') return defaultHudPreferences;
  try {
    const raw = window.localStorage.getItem(HUD_PREFERENCES_KEY);
    return normalizeHudPreferences(
      raw ? (JSON.parse(raw) as Partial<HudPresentationPreferences>) : null,
    );
  } catch {
    return defaultHudPreferences;
  }
};

export const saveHudPreferences = (prefs: HudPresentationPreferences) => {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(HUD_PREFERENCES_KEY, JSON.stringify(normalizeHudPreferences(prefs)));
  window.dispatchEvent(new CustomEvent('nexus:hud-preferences'));
};

export type HudCostDisplay = {
  label: string;
  semantics: 'reported' | 'estimated' | 'known-partial' | 'unknown';
};

const formatUSD = (microusd: number) =>
  new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(microusd / 1_000_000);

export const formatHudCost = (cost?: HudCostSummary): HudCostDisplay => {
  if (!cost || cost.known_microusd <= 0 || cost.by_kind.length === 0) {
    return { label: '—', semantics: 'unknown' };
  }
  const kinds = new Set(
    cost.by_kind.filter((entry) => entry.microusd > 0).map((entry) => entry.kind),
  );
  const value = formatUSD(cost.known_microusd);
  if (
    kinds.size === 1 &&
    kinds.has('PROVIDER_REPORTED') &&
    cost.unknown_observation_count === 0 &&
    cost.evidence === 'MEASURED'
  ) {
    // Provider-reported usage is not proof of an invoiced API charge.
    // Subscription plans can report equivalent token costs without billing.
    return { label: value, semantics: 'reported' };
  }
  if (
    cost.unknown_observation_count === 0 &&
    kinds.size > 0 &&
    !kinds.has('PROVIDER_REPORTED') &&
    !kinds.has('UNKNOWN')
  ) {
    return { label: `~${value}`, semantics: 'estimated' };
  }
  return { label: `${value} known`, semantics: 'known-partial' };
};

export const providerCapacityLabel = (provider: HudProvider): string => {
  const remaining = provider.limiting_window?.remaining_percent;
  if (remaining === undefined || provider.freshness === 'UNKNOWN') return 'UNKNOWN';
  return `${Math.round(remaining)}%`;
};

export const isWorkingAgent = (agent: HudAgent) => agent.status.toUpperCase() === 'WORKING';

export const activeMission = (missions: HudMission[]) =>
  missions.find((mission) => mission.needs_user) ||
  missions.find(
    (mission) =>
      !['COMPLETED_VERIFIED', 'CANCELED_BY_USER', 'FAILED_BUDGET_EXCEEDED'].includes(mission.state),
  );
