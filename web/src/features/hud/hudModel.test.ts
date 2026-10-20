import { describe, expect, it } from 'vitest';
import {
  defaultHudPreferences,
  formatHudCost,
  normalizeHudPreferences,
  providerCapacityLabel,
  resolveNativeHudStatus,
  type HudCostSummary,
  type HudProvider,
} from './hudModel';

const cost = (patch: Partial<HudCostSummary>): HudCostSummary => ({
  evidence: 'MEASURED',
  token_evidence: 'MEASURED',
  known_microusd: 4_200_000,
  unknown_observation_count: 0,
  budget_level: 'OK',
  by_kind: [],
  ...patch,
});

describe('HUD presentation model', () => {
  it('distinguishes loading, offline, disabled, and ready native HUD states', () => {
    expect(resolveNativeHudStatus(undefined, false, false)).toBe('loading');
    expect(resolveNativeHudStatus(undefined, false, true)).toBe('offline');
    expect(resolveNativeHudStatus(false, false, false)).toBe('disabled');
    expect(resolveNativeHudStatus(true, false, false)).toBe('offline');
    expect(resolveNativeHudStatus(true, true, false)).toBe('ready');
    expect(resolveNativeHudStatus(true, true, true)).toBe('offline');
  });

  it('labels measured provider-reported usage without claiming actual billing', () => {
    expect(
      formatHudCost(
        cost({
          by_kind: [{ kind: 'PROVIDER_REPORTED', microusd: 4_200_000, count: 1 }],
        }),
      ).semantics,
    ).toBe('reported');
  });

  it('marks inferred/API-equivalent cost as estimated', () => {
    const display = formatHudCost(
      cost({
        by_kind: [{ kind: 'API_EQUIVALENT_ESTIMATE', microusd: 4_200_000, count: 1 }],
      }),
    );
    expect(display.semantics).toBe('estimated');
    expect(display.label.startsWith('~')).toBe(true);
  });

  it('does not present mixed or incomplete evidence as a total actual cost', () => {
    const display = formatHudCost(
      cost({
        evidence: 'PARTIAL',
        unknown_observation_count: 2,
        by_kind: [{ kind: 'PROVIDER_REPORTED', microusd: 4_200_000, count: 1 }],
      }),
    );
    expect(display.semantics).toBe('known-partial');
    expect(display.label).toContain('known');
  });

  it('never treats incomplete provider reports or subscription equivalents as an invoice', () => {
    const incomplete = formatHudCost(
      cost({
        evidence: 'PARTIAL',
        unknown_observation_count: 1,
        by_kind: [{ kind: 'PROVIDER_REPORTED', microusd: 4_200_000, count: 1 }],
      }),
    );
    expect(incomplete.semantics).toBe('known-partial');

    const subscriptionEquivalent = formatHudCost(
      cost({
        by_kind: [{ kind: 'API_EQUIVALENT_ESTIMATE', microusd: 4_200_000, count: 1 }],
      }),
    );
    expect(subscriptionEquivalent.semantics).toBe('estimated');
  });

  it('preserves UNKNOWN capacity instead of coercing it to zero', () => {
    const provider = {
      provider: 'codex',
      profile: 'default',
      health: 'UNKNOWN',
      availability: 'UNKNOWN',
      capacity_state: 'UNKNOWN',
      freshness: 'UNKNOWN',
      rate_limited: false,
      windows: [],
    } satisfies HudProvider;
    expect(providerCapacityLabel(provider)).toBe('UNKNOWN');
  });

  it('normalizes invalid preferences without mixing them with Capacity preferences', () => {
    const normalized = normalizeHudPreferences({
      ...defaultHudPreferences,
      mode: 'NOT_A_MODE' as never,
      surfaceMode: 'bad' as never,
    });
    expect(normalized.mode).toBe('AUTO');
    expect(normalized.surfaceMode).toBe('docked');
    expect(normalized.version).toBe(1);
    expect(defaultHudPreferences.enabled).toBe(false);
  });
});
