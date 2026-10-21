export type HudMode = 'AUTO' | 'CAPACITY' | 'RESOURCE' | 'AGENT' | 'MISSION' | 'ALERT' | 'MINIMAL';

export type HudSurfaceMode = 'docked' | 'floating';
export type HudDock = 'top' | 'bottom';
export type HudAppearance = 'bar' | 'companion';

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
  appearance: HudAppearance;
  companionMotion: boolean;
  attentionBubbles: boolean;
  showCapacity: boolean;
  showAgents: boolean;
  showMissions: boolean;
  showCost: boolean;
  showRecommendation: boolean;
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
  appearance: 'bar',
  companionMotion: true,
  attentionBubbles: true,
  showCapacity: true,
  showAgents: true,
  showMissions: true,
  showCost: true,
  showRecommendation: true,
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
  appearance: raw?.appearance === 'companion' ? 'companion' : 'bar',
  companionMotion: raw?.companionMotion !== false,
  attentionBubbles: raw?.attentionBubbles !== false,
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
