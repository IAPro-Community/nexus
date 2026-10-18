import { isWorkingAgent, type HudMission, type HudProvider, type HudState } from './hudModel';

export type HudTone = 'normal' | 'warning' | 'critical' | 'unknown';

export interface HudFocus {
  provider?: HudProvider;
  active: boolean;
  remaining: number | null;
  tone: HudTone;
}

const usableRemaining = (provider?: HudProvider): number | null => {
  // Only known evidence classes may display a capacity percentage. A provider
  // can still report a stale numeric value when the observation is unavailable.
  if (!provider || !['LIVE', 'CACHED', 'ESTIMATED'].includes(provider.freshness.toUpperCase())) {
    return null;
  }
  const value = provider.limiting_window?.remaining_percent;
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
  const agents = state.agents.filter(isWorkingAgent);
  const mission = hudVisibleMission(state.missions);
  const exact = agents
    .map((agent) => findProvider(state.providers, agent.provider, agent.profile))
    .find((provider) => provider !== undefined);
  const missionProvider = mission
    ? findProvider(state.providers, mission.provider, mission.profile)
    : undefined;
  // Without runtime identity the first available measured resource is only a
  // highlighted account, NEVER asserted to be the active routing target.
  const fallback =
    state.providers.find(
      (provider) => usableRemaining(provider) !== null && provider.availability === 'AVAILABLE',
    ) ||
    state.providers.find((provider) => usableRemaining(provider) !== null) ||
    state.providers[0];
  const provider = exact || missionProvider || fallback;
  const remaining = hudRemaining(provider, offline);
  const stateName = provider?.capacity_state?.toUpperCase();
  const alert = state.alerts.some(
    (item) =>
      item.provider?.toLowerCase() === provider?.provider.toLowerCase() &&
      (!item.profile || item.profile.toLowerCase() === provider?.profile.toLowerCase()) &&
      ['CRITICAL', 'EXHAUSTED'].includes(item.severity.toUpperCase()),
  );
  const tone: HudTone =
    remaining === null
      ? 'unknown'
      : alert || stateName === 'CRITICAL' || stateName === 'EXHAUSTED'
        ? 'critical'
        : stateName === 'WARNING'
          ? 'warning'
          : 'normal';
  // A Mission can highlight its configured resource but is not evidence that
  // this profile currently owns a running provider session.
  return { provider, active: Boolean(exact), remaining, tone };
};

export const hudPercentage = (value: number | null) =>
  value === null ? '—' : `${Math.round(value)}%`;

export type HudQuotaEvidence = 'live' | 'cached' | 'estimated' | 'unknown';

export const hudQuotaEvidence = (provider?: HudProvider, offline = false): HudQuotaEvidence => {
  if (hudRemaining(provider, offline) === null) return 'unknown';
  switch (provider?.freshness.toUpperCase()) {
    case 'LIVE':
      return 'live';
    case 'CACHED':
      return 'cached';
    case 'ESTIMATED':
      return 'estimated';
    default:
      return 'unknown';
  }
};

export const hudPercentageFor = (provider?: HudProvider, offline = false): string => {
  const remaining = hudRemaining(provider, offline);
  if (remaining === null) return '—';
  return `${hudQuotaEvidence(provider, offline) === 'estimated' ? '~' : ''}${Math.round(remaining)}%`;
};

// Do not surface a completed/failed historical Mission as a current execution.
// The projection is ordered by update time; human intervention takes priority.
export const hudVisibleMission = (missions: HudMission[]): HudMission | undefined =>
  missions.find((mission) => mission.needs_user) ||
  missions.find(
    (mission) => !/^(COMPLETED|CANCELED|CANCELLED|FAILED|STOPPED)(_|$)/i.test(mission.state),
  );

export const hudOtherAccounts = (state: HudState, focused?: HudProvider) =>
  state.providers
    .filter((account) => account !== focused)
    .sort((a, b) => {
      const aRemaining = usableRemaining(a);
      const bRemaining = usableRemaining(b);
      if (aRemaining === null) return bRemaining === null ? 0 : 1;
      if (bRemaining === null) return -1;
      return bRemaining - aRemaining;
    });
