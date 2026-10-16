import { describe, expect, it } from 'vitest';
import {
  hudOtherAccounts,
  hudPercentage,
  hudRemaining,
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
    const b = provider({ profile: 'secondary', limiting_window: { kind: '5h', status: 'LIVE', remaining_percent: 34 } });
    const input = state([a, b]);
    input.agents = [{
      id: 'a', project_id: 'p', name: 'Coder', status: 'WORKING',
      provider: 'codex', profile: 'secondary',
    }];
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
    const unknown = selectHudFocus(state([provider({ freshness: 'UNKNOWN', capacity_state: 'WARNING' })]));
    expect(unknown.tone).toBe('unknown');
  });

  it('keeps multiple Codex profiles distinct and orders alternatives by measured remaining quota', () => {
    const a = provider();
    const b = provider({ profile: 'b', limiting_window: { kind: '5h', status: 'LIVE', remaining_percent: 23 } });
    const c = provider({ profile: 'c', limiting_window: { kind: '5h', status: 'LIVE', remaining_percent: 91 } });
    const d = provider({ profile: 'd', freshness: 'UNKNOWN' });
    expect(hudOtherAccounts(state([a, b, c, d]), a)).toEqual([c, b, d]);
  });

  it('does not promote an offline measurement to a current value', () => {
    const focus = selectHudFocus(state([provider()]), true);
    expect(focus.remaining).toBeNull();
    expect(focus.tone).toBe('unknown');
  });
});
