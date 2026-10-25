import { isWorkingAgent, type HudMission, type HudProvider, type HudState } from './hudModel';
import { asArray } from '../../lib/safeArray';

export type HudTone = 'normal' | 'warning' | 'critical' | 'unknown';

export interface HudFocus {
  provider?: HudProvider;
  active: boolean;
  remaining: number | null;
  tone: HudTone;
}

const usableRemaining = (provider?: HudProvider): number | null => {
  // RATE_LIMITED can include measured percentages: never confuse a blocked
  // account with an unknown measurement, or a percentage with eligibility.
  if (
    !provider ||
    !['LIVE', 'CACHED', 'ESTIMATED', 'RATE_LIMITED'].includes(provider.freshness.toUpperCase())
  ) {
    return null;
  }
  const window = provider.limiting_window;
  if (!window || window.status.toUpperCase() === 'UNKNOWN') return null;
  // Once a quota window has rolled over its last percentage is historical.
  // The provider must be measured again before the capsule reports capacity.
  if (window.reset_at) {
    const resetAt = Date.parse(window.reset_at);
    if (!Number.isFinite(resetAt) || resetAt <= Date.now()) return null;
  }
  const value = window.remaining_percent;
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 && value <= 100
    ? value
    : null;
};

export const hudRemaining = (provider?: HudProvider, offline = false): number | null =>
  offline ? null : usableRemaining(provider);

const findProvider = (providers: HudProvider[], provider?: string, profile?: string) => {
  // A provider name without the profile is not a runtime account identity.
  // Never select the first of several linked Codex/Claude accounts by accident.
  if (!provider?.trim() || !profile?.trim()) return undefined;
  return providers.find(
    (candidate) =>
      candidate.provider.toLowerCase() === provider.toLowerCase() &&
      candidate.profile.toLowerCase() === profile.toLowerCase(),
  );
};

export const selectHudFocus = (state: HudState, offline = false): HudFocus => {
  const providers = asArray<HudProvider>(state.providers);
  const agents = asArray<HudState['agents'][number]>(state.agents).filter(isWorkingAgent);
  const missions = asArray<HudMission>(state.missions);
  const alerts = asArray<HudState['alerts'][number]>(state.alerts);
  const mission = hudVisibleMission(missions);
  const executingAccounts = agents
    .map((agent) => findProvider(providers, agent.provider, agent.profile))
    .filter((provider): provider is HudProvider => provider !== undefined);
  const distinctAccounts = new Set(
    executingAccounts.map((provider) =>
      JSON.stringify([provider.provider.toLowerCase(), provider.profile.toLowerCase()]),
    ),
  );
  // Multiple running identities cannot truthfully be described as one
  // "active account". Use the neutral highlighted-account presentation.
  const exact = distinctAccounts.size === 1 ? executingAccounts[0] : undefined;
  const missionProvider = mission
    ? findProvider(providers, mission.provider, mission.profile)
    : undefined;
  // Without runtime identity the first available measured resource is only a
  // highlighted account, NEVER asserted to be the active routing target.
  const fallback =
    providers.find(
      (provider) => usableRemaining(provider) !== null && provider.availability === 'AVAILABLE',
    ) ||
    providers.find((provider) => usableRemaining(provider) !== null) ||
    providers[0];
  const provider = exact || missionProvider || fallback;
  const remaining = hudRemaining(provider, offline);
  const stateName = provider?.capacity_state?.toUpperCase();
  const alert = alerts.some(
    (item) =>
      item.provider?.toLowerCase() === provider?.provider.toLowerCase() &&
      (!item.profile || item.profile.toLowerCase() === provider?.profile.toLowerCase()) &&
      ['CRITICAL', 'EXHAUSTED'].includes(item.severity.toUpperCase()),
  );
  const blocked = provider?.rate_limited || provider?.freshness.toUpperCase() === 'RATE_LIMITED';
  const exhausted = alert || stateName === 'CRITICAL' || stateName === 'EXHAUSTED';
  const tone: HudTone =
    blocked || (remaining !== null && exhausted)
      ? 'critical'
      : remaining === null
        ? 'unknown'
        : stateName === 'WARNING'
          ? 'warning'
          : 'normal';
  // A Mission can highlight its configured resource but is not evidence that
  // this profile currently owns a running provider session.
  return { provider, active: Boolean(exact), remaining, tone };
};

export const hudPercentage = (value: number | null) =>
  value === null ? '—' : `${Math.round(value)}%`;

export type HudQuotaEvidence = 'live' | 'cached' | 'estimated' | 'rate-limited' | 'unknown';

export const hudQuotaEvidence = (provider?: HudProvider, offline = false): HudQuotaEvidence => {
  if (hudRemaining(provider, offline) === null) return 'unknown';
  switch (provider?.freshness.toUpperCase()) {
    case 'LIVE':
      return 'live';
    case 'CACHED':
      return 'cached';
    case 'ESTIMATED':
      return 'estimated';
    case 'RATE_LIMITED':
      return 'rate-limited';
    default:
      return 'unknown';
  }
};

export const hudPercentageFor = (provider?: HudProvider, offline = false): string => {
  const remaining = hudRemaining(provider, offline);
  if (remaining === null) return '—';
  const prefix = hudQuotaEvidence(provider, offline) === 'estimated' ? '~' : '';
  return `${prefix}${Math.round(remaining)}%`;
};

// Do not surface a completed/failed historical Mission as a current execution.
// The projection is ordered by update time; human intervention takes priority.
export const hudVisibleMission = (missions: HudMission[]): HudMission | undefined =>
  missions.find((mission) => mission.needs_user) ||
  missions.find(
    (mission) => !/^(COMPLETED|CANCELED|CANCELLED|FAILED|STOPPED)(_|$)/i.test(mission.state),
  );

export const hudOtherAccounts = (state: HudState, focused?: HudProvider) =>
  asArray<HudProvider>(state.providers)
    .filter((account) => account !== focused)
    .sort((a, b) => {
      const aRemaining = usableRemaining(a);
      const bRemaining = usableRemaining(b);
      if (aRemaining === null) return bRemaining === null ? 0 : 1;
      if (bRemaining === null) return -1;
      return bRemaining - aRemaining;
    });
