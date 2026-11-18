import { describe, expect, it } from 'vitest';
import type { CapacityResourceStatus, MaestroStatus, ProviderAccount } from '../../types';
import {
  collectObservations,
  deriveDiagnosticsProblems,
  isObservationStale,
  normalizeObservationState,
  OBSERVATION_TTL_MS,
  type UsageObservation,
} from './usageDiagnosticsModel';
import {
  normalizeUsageArea,
  readStoredUsageArea,
  writeStoredUsageArea,
  USAGE_AREA_STORAGE_KEY,
} from './usageModel';

const NOW = Date.parse('2026-11-18T12:00:00Z');

function account(overrides: Partial<ProviderAccount>): ProviderAccount {
  return {
    id: 'a1',
    provider: 'openai',
    profile: 'default',
    display_name: 'OpenAI (default)',
    authenticated: true,
    is_default: true,
    available: true,
    quota_remaining: 50,
    quota_total: 100,
    rate_limited: false,
    health: 'healthy',
    last_checked: '2026-11-18T11:00:00Z',
    ...overrides,
  };
}

function monitor(
  state: string,
  overrides: Partial<CapacityResourceStatus> = {},
): CapacityResourceStatus {
  return {
    key: 'openai/default',
    provider: 'openai',
    profile: 'default',
    quota_group: '',
    window_kind: '',
    state,
    fingerprint: 'fp-1',
    transitioned_at: '2026-11-18T11:30:00Z',
    last_observed_at: '2026-11-18T11:30:00Z',
    acked: false,
    ...overrides,
  };
}

describe('usage area persistence', () => {
  it('normalizes unknown stored values back to the accounts area', () => {
    expect(normalizeUsageArea('consumption')).toBe('consumption');
    expect(normalizeUsageArea('diagnostics')).toBe('diagnostics');
    expect(normalizeUsageArea('accounts')).toBe('accounts');
    expect(normalizeUsageArea('whatever')).toBe('accounts');
    expect(normalizeUsageArea(null)).toBe('accounts');
  });

  it('reads and writes through a storage double without throwing', () => {
    const backing = new Map<string, string>();
    const storage = {
      getItem: (key: string) => backing.get(key) ?? null,
      setItem: (key: string, value: string) => void backing.set(key, value),
    };
    expect(readStoredUsageArea(storage)).toBe('accounts');
    writeStoredUsageArea(storage, 'diagnostics');
    expect(backing.get(USAGE_AREA_STORAGE_KEY)).toBe('diagnostics');
    expect(readStoredUsageArea(storage)).toBe('diagnostics');
    expect(readStoredUsageArea(null)).toBe('accounts');
  });
});

describe('normalizeObservationState', () => {
  it('trusts the reported quota_view status', () => {
    expect(normalizeObservationState(account({ quota_view: { status: 'CACHED' } }))).toBe('CACHED');
    expect(normalizeObservationState(account({ quota_view: { status: ' live ' } }))).toBe('LIVE');
    expect(normalizeObservationState(account({ quota_view: { status: 'UNSUPPORTED' } }))).toBe(
      'UNSUPPORTED',
    );
  });

  it('falls back without inventing evidence', () => {
    expect(normalizeObservationState(account({}))).toBe('UNKNOWN');
    expect(normalizeObservationState(account({ network_error: true }))).toBe('UNAVAILABLE');
    expect(
      normalizeObservationState(account({ rate_limited: true, quota_view: { status: 'LIVE' } })),
    ).toBe('LIVE');
  });
});

describe('isObservationStale', () => {
  const cached: UsageObservation = {
    accountId: 'a1',
    provider: 'openai',
    profile: 'default',
    source: null,
    state: 'CACHED',
    fetchedAt: new Date(NOW - OBSERVATION_TTL_MS - 1_000).toISOString(),
    ageSeconds: null,
    monitorState: null,
  };

  it('flags CACHED evidence older than the routing TTL', () => {
    expect(isObservationStale(cached, NOW)).toBe(true);
  });

  it('keeps fresh CACHED evidence and never flags other states', () => {
    const fresh: UsageObservation = {
      ...cached,
      fetchedAt: new Date(NOW - 60_000).toISOString(),
    };
    expect(isObservationStale(fresh, NOW)).toBe(false);
    expect(isObservationStale({ ...cached, state: 'LIVE' }, NOW)).toBe(false);
    expect(isObservationStale({ ...cached, fetchedAt: null }, NOW)).toBe(false);
  });
});

describe('collectObservations', () => {
  it('collects one row per account sorted by provider/profile', () => {
    const rows = collectObservations(
      [
        account({ id: 'b', provider: 'zai', profile: 'work' }),
        account({ id: 'c', provider: 'zai', profile: 'personal' }),
        account({
          id: 'a',
          provider: 'anthropic',
          profile: 'main',
          quota_view: { status: 'LIVE', source: 'cli', fetched_at: '2026-11-18T11:59:00Z' },
        }),
      ],
      { 'zai/personal': monitor('HEALTHY') },
      NOW,
    );
    expect(rows.map((row) => `${row.provider}/${row.profile}`)).toEqual([
      'anthropic/main',
      'zai/personal',
      'zai/work',
    ]);
    expect(rows[0]).toMatchObject({ state: 'LIVE', source: 'cli', ageSeconds: 60 });
    expect(rows[1]).toMatchObject({ state: 'UNKNOWN', source: null, monitorState: 'HEALTHY' });
    expect(rows[2].ageSeconds).toBeNull();
  });
});

describe('deriveDiagnosticsProblems', () => {
  const maestroOk: MaestroStatus = { available: true, mode: 'full' };

  it('deduces no-live-observation and auth-required problems with actions', () => {
    const problems = deriveDiagnosticsProblems({
      accounts: [
        account({ authenticated: false, available: false }),
        account({ id: 'a2', provider: 'zai', profile: 'work' }),
      ],
      monitorByAccount: {},
      maestroStatus: maestroOk,
      maestroError: '',
      installedWithoutAccounts: [],
      now: NOW,
    });
    expect(problems.map((problem) => problem.kind)).toEqual([
      'authRequired',
      'noLiveObservation',
      'noLiveObservation',
    ]);
    expect(problems[0].action).toEqual({ kind: 'login', accountId: 'a1' });
    expect(problems[1].action).toEqual({ kind: 'refresh', provider: 'openai', profile: 'default' });
    expect(problems[1].params.state).toBe('UNKNOWN');
  });

  it('flags monitor critical states with reason and stale CACHED evidence', () => {
    const problems = deriveDiagnosticsProblems({
      accounts: [
        account({
          quota_view: {
            status: 'CACHED',
            fetched_at: new Date(NOW - OBSERVATION_TTL_MS - 60_000).toISOString(),
          },
        }),
      ],
      monitorByAccount: {
        'openai/default': monitor('EXHAUSTED', { reason: '5h window drained' }),
      },
      maestroStatus: maestroOk,
      maestroError: '',
      installedWithoutAccounts: [],
      now: NOW,
    });
    expect(problems.map((problem) => problem.kind)).toEqual([
      'monitorCritical',
      'staleObservation',
    ]);
    expect(problems[0].severity).toBe('critical');
    expect(problems[0].params).toMatchObject({
      state: 'EXHAUSTED',
      reason: '5h window drained',
      provider: 'openai',
    });
    expect(problems[0].action).toEqual({ kind: 'refresh', provider: 'openai', profile: 'default' });
  });

  it('points unavailable providers to the terminal instead of a blind refresh', () => {
    const problems = deriveDiagnosticsProblems({
      accounts: [account({ network_error: true })],
      monitorByAccount: {},
      maestroStatus: maestroOk,
      maestroError: '',
      installedWithoutAccounts: [],
      now: NOW,
    });
    expect(problems).toHaveLength(1);
    expect(problems[0].kind).toBe('noLiveObservation');
    expect(problems[0].params.state).toBe('UNAVAILABLE');
    expect(problems[0].action).toEqual({
      kind: 'terminal',
      provider: 'openai',
      profile: 'default',
    });
  });

  it('reports maestro unavailability without promising execution', () => {
    const fromStatus = deriveDiagnosticsProblems({
      accounts: [],
      monitorByAccount: {},
      maestroStatus: { available: false, mode: 'off', error: 'not running' },
      maestroError: '',
      installedWithoutAccounts: [],
      now: NOW,
    });
    const fromError = deriveDiagnosticsProblems({
      accounts: [],
      monitorByAccount: {},
      maestroStatus: maestroOk,
      maestroError: 'fetch failed',
      installedWithoutAccounts: [],
      now: NOW,
    });
    expect(fromStatus).toHaveLength(1);
    expect(fromStatus[0].kind).toBe('maestroUnavailable');
    expect(fromStatus[0].severity).toBe('info');
    expect(fromStatus[0].params.error).toBe('not running');
    expect(fromStatus[0].action).toBeUndefined();
    expect(fromError[0].params.error).toBe('fetch failed');
  });

  it('suggests registering profiles for installed providers without accounts', () => {
    const problems = deriveDiagnosticsProblems({
      accounts: [],
      monitorByAccount: {},
      maestroStatus: maestroOk,
      maestroError: '',
      installedWithoutAccounts: [{ id: 'codex' }],
      now: NOW,
    });
    expect(problems).toHaveLength(1);
    expect(problems[0].kind).toBe('installedWithoutProfile');
    expect(problems[0].action).toEqual({ kind: 'register', provider: 'codex' });
  });

  it('stays silent when every account has LIVE evidence and maestro is up', () => {
    const problems = deriveDiagnosticsProblems({
      accounts: [
        account({
          quota_view: { status: 'LIVE', fetched_at: new Date(NOW - 30_000).toISOString() },
        }),
      ],
      monitorByAccount: { 'openai/default': monitor('HEALTHY') },
      maestroStatus: maestroOk,
      maestroError: '',
      installedWithoutAccounts: [],
      now: NOW,
    });
    expect(problems).toEqual([]);
  });

  it('sorts critical problems before informational ones', () => {
    const problems = deriveDiagnosticsProblems({
      accounts: [
        account({
          quota_view: {
            status: 'CACHED',
            fetched_at: new Date(NOW - OBSERVATION_TTL_MS - 60_000).toISOString(),
          },
        }),
      ],
      monitorByAccount: { 'openai/default': monitor('CRITICAL') },
      maestroStatus: { available: false, mode: 'off' },
      maestroError: '',
      installedWithoutAccounts: [{ id: 'codex' }],
      now: NOW,
    });
    expect(problems.map((problem) => problem.severity)).toEqual([
      'critical',
      'warning',
      'info',
      'info',
    ]);
  });
});
