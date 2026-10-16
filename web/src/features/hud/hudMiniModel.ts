import {
  activeMission,
  isWorkingAgent,
  type HudProvider,
  type HudState,
} from './hudModel';

export type HudTone = 'normal' | 'warning' | 'critical' | 'unknown';

export interface HudFocus {
  provider?: HudProvider;
  active: boolean;
  remaining: number | null;
  tone: HudTone;
}

const usableRemaining = (provider?: HudProvider): number | null => {
  if (!provider || ['UNKNOWN', 'UNAVAILABLE', 'UNSUPPORTED'].includes(provider.freshness.toUpperCase())) {
    return null;
  }
  const value = provider.limiting_window?.remaining_percent;
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 && value <= 100
    ? value
    : null;
};

export const hudRemaining = (provider?: HudProvider, offline = false): number | null =>
  offline ? null : usableRemaining(provider);

const findProvider = (providers: HudProvider[], provider?: string, profile?: string) =>
  providers.find(
    (candidate) =>
      candidate.provider.toLowerCase() === provider?.toLowerCase() &&
      (!profile || candidate.profile.toLowerCase() === profile.toLowerCase()),
  );

export const selectHudFocus = (state: HudState, offline = false): HudFocus => {
  const agents = state.agents.filter(isWorkingAgent);
  const mission = activeMission(state.missions);
  const exact = agents
    .map((agent) => findProvider(state.providers, agent.provider, agent.profile))
    .find((provider) => provider !== undefined);
  const missionProvider = mission ? findProvider(state.providers, mission.provider, mission.profile) : undefined;
  // Without runtime identity the first available measured resource is only a
  // highlighted account, NEVER asserted to be the active routing target.
  const fallback =
    state.providers.find((provider) => usableRemaining(provider) !== null && provider.availability === 'AVAILABLE') ||
    state.providers.find((provider) => usableRemaining(provider) !== null) ||
    state.providers[0];
  const provider = exact || missionProvider || fallback;
  const remaining = hudRemaining(provider, offline);
  const stateName = provider?.capacity_state?.toUpperCase();
  const alert = state.alerts.some(
    (item) =>
      item.provider.toLowerCase() === provider?.provider.toLowerCase() &&
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
  return { provider, active: Boolean(exact || missionProvider), remaining, tone };
};

export const hudPercentage = (value: number | null) =>
  value === null ? '—' : `${Math.round(value)}%`;

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
