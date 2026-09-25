import { describe, expect, it } from 'vitest';
import type { ProviderAccount, ProviderInfo } from '../../types';
import {
  accountQuotaState,
  capabilityLabelKey,
  displayLabelKey,
  filterAccounts,
  formatResetDescription,
  getQuotaProgressTone,
  groupHasCapacity,
  installedProvidersWithoutAccounts,
  isAccountAuthRequired,
  parseResetDescription,
  relativeAge,
  summarizeAccounts,
  windowKindLabelKey,
} from './usageModel';

const base = (partial: Partial<ProviderAccount>): ProviderAccount => ({
  id: 'agy:test',
  provider: 'agy',
  profile: 'test',
  display_name: 'agy test',
  authenticated: true,
  is_default: false,
  available: true,
  quota_remaining: 0.5,
  quota_total: 1,
  rate_limited: false,
  health: 'healthy',
  last_checked: new Date().toISOString(),
  ...partial,
});

describe('usageModel', () => {
  it('lists installed OpenCode even when lifecycle is pending auth and no resource profile exists', () => {
    const providers: ProviderInfo[] = [
      {
        id: 'opencode',
        installed: true,
        version: '1.18.30',
        registration_state: 'PENDING_AUTH',
        control_level: 'TERMINAL',
        capabilities: {} as ProviderInfo['capabilities'],
      },
      {
        id: 'codex',
        installed: true,
        version: '0.154.0',
        registration_state: 'REGISTERED_AUTHENTICATED',
        control_level: 'TERMINAL',
        capabilities: {} as ProviderInfo['capabilities'],
      },
    ];
    const accounts = [
      base({
        id: 'codex:default',
        provider: 'codex',
        profile: 'default',
      }),
    ];

    expect(
      installedProvidersWithoutAccounts(providers, accounts).map((provider) => provider.id),
    ).toEqual(['opencode']);
  });

  it('does not mark account exhausted when another group still has capacity', () => {
    const account = base({
      available: true,
      avail_reasons: { exhausted_windows: ['gemini:weekly'] },
      quota_view: {
        status: 'LIVE',
        model_groups: [
          {
            name: 'Gemini',
            windows: [{ kind: 'weekly', remaining: 0 }],
          },
          {
            name: 'Claude',
            windows: [{ kind: '5h', remaining: 100 }],
          },
        ],
      },
    });
    expect(accountQuotaState(account)).toBe('confirmed');
    expect(groupHasCapacity(account.quota_view?.model_groups?.[1])).toBe(true);
  });

  it('summarizes fleet counts', () => {
    const summary = summarizeAccounts([
      base({
        available: true,
        quota_view: {
          status: 'LIVE',
          model_groups: [{ windows: [{ kind: '5h', remaining: 40 }] }],
        },
      }),
      base({
        id: 'b',
        available: false,
        avail_reasons: { exhausted_windows: ['all'] },
        quota_view: { status: 'LIVE', model_groups: [{ windows: [{ kind: '5h', remaining: 0 }] }] },
      }),
      base({
        id: 'c',
        available: false,
        avail_reasons: { unknown_quota: true },
        quota_view: { status: 'UNKNOWN' },
      }),
    ]);
    expect(summary.usable).toBe(1);
    expect(summary.exhausted).toBe(1);
    expect(summary.unknown).toBe(1);
  });

  it('returns relative age without mixing language into the status code', () => {
    const now = Date.parse('2026-09-10T20:00:00Z');
    expect(relativeAge('2026-09-10T19:59:52Z', now)).toEqual({ unit: 'seconds', count: 8 });
  });

  it('maps known English API labels and window kinds to i18n keys', () => {
    expect(displayLabelKey('OpenCode Provider')).toBe('usage.plan.opencode');
    expect(displayLabelKey('Gemini Models', 'gemini')).toBe('usage.group.gemini');
    expect(displayLabelKey(undefined, 'claude_gpt')).toBe('usage.group.claudeGpt');
    expect(displayLabelKey('ChatGPT Plus')).toBeNull();
    expect(windowKindLabelKey('weekly')).toBe('usage.window.weekly');
    expect(windowKindLabelKey('5h')).toBe('usage.window.fiveHour');
    expect(windowKindLabelKey('unknown')).toBe('usage.window.unknown');
    expect(capabilityLabelKey('cancel_turn')).toBe('usage.capability.cancel_turn');
  });

  it('parses English Codex reset strings for localized formatting', () => {
    expect(parseResetDescription('resets 03:48')).toEqual({ kind: 'sameDay', time: '03:48' });
    expect(parseResetDescription('resets 05:35 on 15 Sep')).toEqual({
      kind: 'crossDay',
      time: '05:35',
      day: 15,
      monthIndex: 8,
    });
    expect(
      formatResetDescription('resets 05:35 on 15 Sep', 'pt-BR', (key, options) =>
        key === 'usage.resetOn' ? `renova às ${options?.time} em ${options?.date}` : key,
      ),
    ).toMatch(/^renova às 05:35 em /);
  });

  it('determines if an account requires authentication', () => {
    expect(isAccountAuthRequired(base({ authenticated: true, health: 'healthy' }))).toBe(false);
    expect(isAccountAuthRequired(base({ authenticated: false, health: 'AUTH_REQUIRED' }))).toBe(
      true,
    );
    expect(
      isAccountAuthRequired(
        base({
          authenticated: true,
          eligibility: { eligible: false, reason: 'NOT_AUTHENTICATED', confidence: 'ESTIMATED' },
        }),
      ),
    ).toBe(true);
  });

  it('returns appropriate progress bar tones', () => {
    expect(getQuotaProgressTone(95)).toBe('success');
    expect(getQuotaProgressTone(50)).toBe('info');
    expect(getQuotaProgressTone(15)).toBe('warning');
    expect(getQuotaProgressTone(5)).toBe('danger');
  });

  it('filters accounts by query, provider, and status', () => {
    const fleet = [
      base({
        id: 'agy:work',
        provider: 'agy',
        profile: 'work',
        display_name: 'Work Account',
        quota_view: { account: 'work@gmail.com' },
        available: true,
        authenticated: true,
      }),
      base({
        id: 'codex:stale',
        provider: 'codex',
        profile: 'stale',
        display_name: 'Codex Work',
        quota_view: { account: 'codex@omega.net' },
        available: false,
        authenticated: false,
      }),
      base({
        id: 'claude:personal',
        provider: 'claude',
        profile: 'personal',
        available: true,
        authenticated: true,
      }),
    ];

    expect(filterAccounts(fleet, { search: 'omega' })).toHaveLength(1);
    expect(filterAccounts(fleet, { provider: 'agy' })).toHaveLength(1);
    expect(filterAccounts(fleet, { status: 'auth_required' })).toHaveLength(1);
    expect(filterAccounts(fleet, { status: 'usable' })).toHaveLength(2);
  });
});
