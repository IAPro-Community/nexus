import { describe, expect, it } from 'vitest';
import {
  hudOtherAccounts,
  hudPercentage,
  hudPercentageFor,
  hudQuotaEvidence,
  hudRemaining,
  hudVisibleMission,
  selectHudFocus,
} from './hudMiniModel';
import type { HudProvider, HudState } from './hudModel';

const provider = (patch: Partial<HudProvider> = {}): HudProvider => ({
  provider: 'codex',
  profile: 'primary',
  health: 'OK',
  availability: 'AVAILABLE',
  capacity_state: 'OK',
  freshness: 'LIVE',
  rate_limited: false,
  limiting_window: {
    kind: '5h',
    remaining_percent: 68,
    status: 'LIVE',
    reset_description: '1h42',
  },
  windows: [],
  ...patch,
});

const state = (providers: HudProvider[]): HudState => ({
  generated_at: new Date(0).toISOString(),
  providers,
  agents: [],
  missions: [],
  alerts: [],
  primary: { kind: 'CAPACITY' },
  system: {
    capacity: 'AVAILABLE',
    resources: 'AVAILABLE',
    agents: 'AVAILABLE',
    missions: 'AVAILABLE',
    cost: 'AVAILABLE',
    cache_ttl_millis: 15000,
  },
});

describe('Nexus HUD compact state', () => {
  it('never treats a missing or stale measurement as zero quota', () => {
    expect(hudRemaining(provider({ freshness: 'UNKNOWN' }))).toBeNull();
    expect(hudRemaining(provider({ limiting_window: undefined }))).toBeNull();
    expect(hudRemaining(provider(), true)).toBeNull();
    expect(hudPercentage(null)).toBe('—');
    expect(hudPercentage(0)).toBe('0%');
  });

  it('shows the executing account when an Agent provides runtime identity', () => {
    const a = provider({ profile: 'primary' });
    const b = provider({
      profile: 'secondary',
      limiting_window: { kind: '5h', status: 'LIVE', remaining_percent: 34 },
    });
    const input = state([a, b]);
    input.agents = [
      {
        id: 'a',
        project_id: 'p',
        name: 'Coder',
        status: 'WORKING',
        provider: 'codex',
        profile: 'secondary',
      },
    ];
    const focus = selectHudFocus(input);
    expect(focus.provider).toBe(b);
    expect(focus.active).toBe(true);
    expect(focus.remaining).toBe(34);
  });

  it('does not falsely claim that the highlighted account is active', () => {
    const focus = selectHudFocus(state([provider()]));
    expect(focus.active).toBe(false);
    expect(focus.remaining).toBe(68);
    expect(selectHudFocus(state([])).remaining).toBeNull();
  });

  it('shows canonical warning and critical states without inventing them from UNKNOWN', () => {
    const warning = selectHudFocus(state([provider({ capacity_state: 'WARNING' })]));
    expect(warning.tone).toBe('warning');
    const critical = selectHudFocus(state([provider({ capacity_state: 'EXHAUSTED' })]));
    expect(critical.tone).toBe('critical');
    const unknown = selectHudFocus(
      state([provider({ freshness: 'UNKNOWN', capacity_state: 'WARNING' })]),
    );
    expect(unknown.tone).toBe('unknown');
  });

  it('keeps multiple Codex profiles distinct and orders alternatives by measured remaining quota', () => {
    const a = provider();
    const b = provider({
      profile: 'b',
      limiting_window: { kind: '5h', status: 'LIVE', remaining_percent: 23 },
    });
    const c = provider({
      profile: 'c',
      limiting_window: { kind: '5h', status: 'LIVE', remaining_percent: 91 },
    });
    const d = provider({ profile: 'd', freshness: 'UNKNOWN' });
    expect(hudOtherAccounts(state([a, b, c, d]), a)).toEqual([c, b, d]);
  });

  it('does not invent a runtime account for multiple profiles with missing identity', () => {
    const a = provider({ profile: 'alpha' });
    const b = provider({ profile: 'beta' });
    const input = state([a, b]);
    input.agents = [
      { id: 'a', project_id: 'p', name: 'Coder', status: 'WORKING', provider: 'codex' },
    ];
    expect(selectHudFocus(input).active).toBe(false);
    // A known provider alone cannot identify the active subscription account.
    expect(selectHudFocus(input).provider).toBe(a);
  });

  it('labels a measured estimate and distinguishes cached quota from live', () => {
    const estimated = provider({
      freshness: 'ESTIMATED',
      limiting_window: {
        kind: '5h',
        status: 'ESTIMATED',
        remaining_percent: 17.7,
      },
    });
    expect(hudQuotaEvidence(estimated)).toBe('estimated');
    expect(hudPercentageFor(estimated)).toBe('~18%');
    const cached = provider({ freshness: 'CACHED' });
    expect(hudQuotaEvidence(cached)).toBe('cached');
    expect(hudPercentageFor(cached)).toBe('68%');
    const unavailable = provider({ freshness: 'STALE' });
    expect(hudQuotaEvidence(unavailable)).toBe('unknown');
    expect(hudPercentageFor(unavailable)).toBe('—');
  });

  it('displays active Missions but not terminal history, prioritizing intervention', () => {
    const mission = (stateName: string, needsUser = false) => ({
      run_id: stateName,
      project_id: 'p',
      state: stateName,
      current_stage: 'execution',
      completed_packages: 2,
      total_packages: 5,
      needs_user: needsUser,
      updated_at: new Date(0).toISOString(),
    });
    const running = mission('RUNNING');
    const intervention = mission('BLOCKED_NEEDS_USER', true);
    expect(hudVisibleMission([mission('COMPLETED_VERIFIED'), running])).toEqual(running);
    expect(hudVisibleMission([running, intervention])).toEqual(intervention);
    expect(hudVisibleMission([mission('FAILED_BUDGET_EXCEEDED')])).toBeUndefined();
  });

  it('retains measured quota when a provider reports RATE_LIMITED', () => {
    const limited = provider({
      freshness: 'RATE_LIMITED',
      rate_limited: true,
      availability: 'UNAVAILABLE',
      limiting_window: { kind: '5h', status: 'RATE_LIMITED', remaining_percent: 44 },
    });
    expect(hudRemaining(limited)).toBe(44);
    expect(hudQuotaEvidence(limited)).toBe('rate-limited');
    expect(hudPercentageFor(limited)).toBe('44%');
    expect(selectHudFocus(state([limited])).tone).toBe('critical');
  });

  it('expires quota after rollover even if cached percentage is present', () => {
    const expired = provider({
      limiting_window: {
        kind: '5h',
        status: 'LIVE',
        remaining_percent: 0,
        reset_at: '2020-01-01T00:00:00Z',
      },
    });
    expect(hudRemaining(expired)).toBeNull();
    expect(hudPercentageFor(expired)).toBe('—');
    expect(selectHudFocus(state([expired])).tone).toBe('unknown');
  });

  it('avoids falsely declaring one active account when agents use different profiles', () => {
    const a = provider({ profile: 'alpha' });
    const b = provider({ profile: 'beta' });
    const snapshot = state([a, b]);
    snapshot.agents = [
      {
        id: 'a',
        project_id: 'p',
        name: 'Architect',
        status: 'WORKING',
        provider: 'codex',
        profile: 'alpha',
      },
      {
        id: 'b',
        project_id: 'p',
        name: 'Coder',
        status: 'WORKING',
        provider: 'codex',
        profile: 'beta',
      },
    ];
    expect(selectHudFocus(snapshot).active).toBe(false);
    snapshot.agents[1].profile = 'alpha';
    expect(selectHudFocus(snapshot).active).toBe(true);
  });

  it('does not promote an offline measurement to a current value', () => {
    const focus = selectHudFocus(state([provider()]), true);
    expect(focus.remaining).toBeNull();
    expect(focus.tone).toBe('unknown');
  });
});
