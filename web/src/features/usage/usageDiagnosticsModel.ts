import type {
  CapacityResourceStatus,
  MaestroStatus,
  ProviderAccount,
  ProviderInfo,
} from '../../types';
import { isAccountAuthRequired } from './usageModel';

/**
 * Mirrors internal/core/quota DefaultTTL (5 minutes): the routing layer
 * already treats observations older than this as stale evidence.
 */
export const OBSERVATION_TTL_MS = 5 * 60_000;

export type ObservationState =
  'LIVE' | 'CACHED' | 'ESTIMATED' | 'UNKNOWN' | 'UNSUPPORTED' | 'RATE_LIMITED' | 'UNAVAILABLE';

const REPORTED_OBSERVATION_STATES: readonly string[] = [
  'LIVE',
  'CACHED',
  'ESTIMATED',
  'UNSUPPORTED',
  'RATE_LIMITED',
  'UNAVAILABLE',
];

/**
 * Normalize the observation state of an account without inventing evidence.
 * The quota_view status is the authority; UNKNOWN is the explicit fallback
 * when no usable status was reported.
 */
export function normalizeObservationState(account: ProviderAccount): ObservationState {
  const status = String(account.quota_view?.status || '')
    .trim()
    .toUpperCase();
  if (REPORTED_OBSERVATION_STATES.includes(status)) {
    return status as ObservationState;
  }
  if (account.network_error) return 'UNAVAILABLE';
  if (account.rate_limited) return 'RATE_LIMITED';
  return 'UNKNOWN';
}

export interface UsageObservation {
  accountId: string;
  provider: string;
  profile: string;
  source: string | null;
  state: ObservationState;
  fetchedAt: string | null;
  ageSeconds: number | null;
  monitorState: string | null;
}

export function collectObservations(
  accounts: ProviderAccount[],
  monitorByAccount: Record<string, CapacityResourceStatus>,
  now = Date.now(),
): UsageObservation[] {
  const rows: UsageObservation[] = [];
  for (const account of accounts) {
    const fetchedAt = account.quota_view?.fetched_at || null;
    let ageSeconds: number | null = null;
    if (fetchedAt) {
      const ts = Date.parse(fetchedAt);
      if (Number.isFinite(ts)) {
        ageSeconds = Math.max(0, Math.round((now - ts) / 1000));
      }
    }
    const monitor = monitorByAccount[`${account.provider}/${account.profile}`] || null;
    rows.push({
      accountId: account.id,
      provider: account.provider,
      profile: account.profile,
      source: String(account.quota_view?.source || '').trim() || null,
      state: normalizeObservationState(account),
      fetchedAt,
      ageSeconds,
      monitorState: monitor?.state || null,
    });
  }
  return rows.sort(
    (a, b) => a.provider.localeCompare(b.provider) || a.profile.localeCompare(b.profile),
  );
}

/** CACHED evidence older than the routing TTL is stale and must be refreshed. */
export function isObservationStale(observation: UsageObservation, now = Date.now()): boolean {
  if (observation.state !== 'CACHED' || !observation.fetchedAt) return false;
  const ts = Date.parse(observation.fetchedAt);
  if (!Number.isFinite(ts)) return false;
  return now - ts > OBSERVATION_TTL_MS;
}

export type DiagnosticProblemSeverity = 'critical' | 'warning' | 'info';

export type DiagnosticAction =
  | { kind: 'refresh'; provider: string; profile: string }
  | { kind: 'login'; accountId: string }
  | { kind: 'terminal'; provider: string; profile: string }
  | { kind: 'register'; provider: string };

export type DiagnosticProblemKind =
  | 'authRequired'
  | 'noLiveObservation'
  | 'staleObservation'
  | 'monitorCritical'
  | 'maestroUnavailable'
  | 'installedWithoutProfile';

export interface DiagnosticProblem {
  id: string;
  kind: DiagnosticProblemKind;
  severity: DiagnosticProblemSeverity;
  accountId?: string;
  provider?: string;
  profile?: string;
  /** i18n key under usage.diagnostics.problems. */
  reasonKey: string;
  params: Record<string, string>;
  action?: DiagnosticAction;
}

const SEVERITY_RANK: Record<DiagnosticProblemSeverity, number> = {
  critical: 3,
  warning: 2,
  info: 1,
};

export interface DiagnosticsInput {
  accounts: ProviderAccount[];
  monitorByAccount: Record<string, CapacityResourceStatus>;
  maestroStatus: MaestroStatus | null;
  maestroError: string;
  /** Only the id is consumed; ProviderInfo[] from installedProvidersWithoutAccounts fits. */
  installedWithoutAccounts: Array<Pick<ProviderInfo, 'id'>>;
  now?: number;
}

/**
 * Deduce read-only problems from existing evidence only. Nothing here invents
 * state: an account without a usable observation surfaces as "no live
 * observation", never as a healthy count.
 */
export function deriveDiagnosticsProblems(input: DiagnosticsInput): DiagnosticProblem[] {
  const now = input.now ?? Date.now();
  const problems: DiagnosticProblem[] = [];

  for (const account of input.accounts) {
    const state = normalizeObservationState(account);
    const label = {
      provider: account.provider,
      profile: account.profile,
    };
    if (isAccountAuthRequired(account)) {
      problems.push({
        id: `authRequired:${account.id}`,
        kind: 'authRequired',
        severity: 'warning',
        accountId: account.id,
        ...label,
        reasonKey: 'usage.diagnostics.problems.authRequired',
        params: label,
        action: { kind: 'login', accountId: account.id },
      });
    }
    if (state === 'UNKNOWN' || state === 'UNAVAILABLE') {
      problems.push({
        id: `noLiveObservation:${account.id}`,
        kind: 'noLiveObservation',
        severity: 'warning',
        accountId: account.id,
        ...label,
        reasonKey: 'usage.diagnostics.problems.noLiveObservation',
        params: { ...label, state },
        // No usable evidence at all: refreshing may recover it. When the
        // provider itself is unreachable, the terminal is the diagnostic path.
        action:
          state === 'UNAVAILABLE'
            ? { kind: 'terminal', provider: account.provider, profile: account.profile }
            : { kind: 'refresh', provider: account.provider, profile: account.profile },
      });
    }
    const observation: UsageObservation | null = account.quota_view?.fetched_at
      ? {
          accountId: account.id,
          provider: account.provider,
          profile: account.profile,
          source: null,
          state,
          fetchedAt: account.quota_view.fetched_at,
          ageSeconds: null,
          monitorState: null,
        }
      : null;
    if (observation && isObservationStale(observation, now)) {
      problems.push({
        id: `staleObservation:${account.id}`,
        kind: 'staleObservation',
        severity: 'warning',
        accountId: account.id,
        ...label,
        reasonKey: 'usage.diagnostics.problems.staleObservation',
        params: label,
        action: { kind: 'refresh', provider: account.provider, profile: account.profile },
      });
    }
    // Monitor UNKNOWN adds no separate problem: the observation-level check
    // above already flags "no usable evidence" without double reporting.
    const monitor = input.monitorByAccount[`${account.provider}/${account.profile}`];
    if (monitor && (monitor.state === 'EXHAUSTED' || monitor.state === 'CRITICAL')) {
      problems.push({
        id: `monitorCritical:${account.id}`,
        kind: 'monitorCritical',
        severity: 'critical',
        accountId: account.id,
        ...label,
        reasonKey: 'usage.diagnostics.problems.monitorCritical',
        params: { ...label, state: monitor.state, reason: monitor.reason || '' },
        action: { kind: 'refresh', provider: account.provider, profile: account.profile },
      });
    }
  }

  const maestroDown =
    Boolean(input.maestroError) ||
    (input.maestroStatus ? input.maestroStatus.available === false : false);
  if (maestroDown) {
    problems.push({
      id: 'maestroUnavailable',
      kind: 'maestroUnavailable',
      severity: 'info',
      reasonKey: 'usage.diagnostics.problems.maestroUnavailable',
      params: {
        error: input.maestroError || String(input.maestroStatus?.error || ''),
      },
    });
  }

  for (const provider of input.installedWithoutAccounts) {
    problems.push({
      id: `installedWithoutProfile:${provider.id}`,
      kind: 'installedWithoutProfile',
      severity: 'info',
      provider: provider.id,
      reasonKey: 'usage.diagnostics.problems.installedWithoutProfile',
      params: { provider: provider.id },
      action: { kind: 'register', provider: provider.id },
    });
  }

  return problems.sort((a, b) => {
    const bySeverity = SEVERITY_RANK[b.severity] - SEVERITY_RANK[a.severity];
    if (bySeverity !== 0) return bySeverity;
    return (a.provider || '').localeCompare(b.provider || '');
  });
}
