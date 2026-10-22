import { describe, expect, it } from 'vitest';
import type { ProviderAccount } from '../types';
import { canManuallySelectResourceAccount } from './resourceSelectionModel';

function account(overrides: Partial<ProviderAccount> = {}): ProviderAccount {
  return {
    id: 'opencode:bolzan',
    provider: 'opencode',
    profile: 'bolzan',
    display_name: 'opencode (bolzan)',
    authenticated: true,
    is_default: false,
    available: false,
    quota_remaining: 0,
    quota_total: 1,
    rate_limited: false,
    health: 'healthy',
    last_checked: '',
    ...overrides,
  };
}

describe('manual resource selection', () => {
  it('allows an authenticated unsupported-quota account with no balance API', () => {
    expect(
      canManuallySelectResourceAccount(
        account({
          eligibility: {
            eligible: false,
            reason: 'UNSUPPORTED_QUOTA',
            confidence: 'UNSUPPORTED',
          },
          avail_reasons: { unknown_quota: true },
        }),
      ),
    ).toBe(true);
  });

  it('allows UNKNOWN quota only when the account is otherwise available', () => {
    const eligibility = { eligible: false, reason: 'UNKNOWN_QUOTA', confidence: 'UNKNOWN' };
    expect(canManuallySelectResourceAccount(account({ available: true, eligibility }))).toBe(true);
    expect(canManuallySelectResourceAccount(account({ available: false, eligibility }))).toBe(
      false,
    );
  });

  it.each([
    ['authentication is missing', { authenticated: false }],
    ['the provider is unavailable', { health: 'unavailable' }],
    ['the account needs authentication', { health: 'auth_required' }],
    ['the provider is rate limited', { rate_limited: true }],
    ['the quota source reports rate limit', { avail_reasons: { rate_limited: true } }],
    ['the probe hit a network error', { network_error: true }],
  ])('does not bypass hard gates when %s', (_name, override) => {
    expect(
      canManuallySelectResourceAccount(
        account({
          eligibility: {
            eligible: false,
            reason: 'UNSUPPORTED_QUOTA',
            confidence: 'UNSUPPORTED',
          },
          ...override,
        }),
      ),
    ).toBe(false);
  });

  it('does not offer an opaque-quota account while its cooldown is active', () => {
    const eligibility = {
      eligible: true,
      reason: 'UNSUPPORTED_QUOTA',
      confidence: 'UNSUPPORTED',
    };
    expect(
      canManuallySelectResourceAccount(
        account({
          eligibility,
          cooldown_until: new Date(Date.now() + 60_000).toISOString(),
        }),
      ),
    ).toBe(false);
    expect(
      canManuallySelectResourceAccount(
        account({
          eligibility,
          cooldown_until: new Date(Date.now() - 60_000).toISOString(),
        }),
      ),
    ).toBe(true);
  });

  it('requires a concrete provider and profile even for opaque quota', () => {
    const eligibility = {
      eligible: true,
      reason: 'UNSUPPORTED_QUOTA',
      confidence: 'UNSUPPORTED',
    };
    expect(canManuallySelectResourceAccount(account({ eligibility, provider: '' }))).toBe(false);
    expect(canManuallySelectResourceAccount(account({ eligibility, profile: '' }))).toBe(false);
  });

  it('keeps exhausted quota accounts disabled', () => {
    expect(
      canManuallySelectResourceAccount(
        account({
          available: false,
          eligibility: { eligible: false, reason: 'EXHAUSTED', confidence: 'LIVE' },
        }),
      ),
    ).toBe(false);
  });
});
