import { describe, expect, it } from 'vitest';
import {
  bestGroupRemainingFromQuotaView,
  buildDirectAgentName,
  compactQuotaLabel,
  directAccountTitle,
  directQuotaDisplay,
  directQuotaPercent,
  eligibleDirectResources,
  quotaTruthState,
} from './directSessionModel';

describe('direct session model', () => {
  it('builds a short useful Agent name from the prompt', () => {
    expect(buildDirectAgentName('Corrigir autenticação Microsoft no backend', 'codex')).toBe(
      'Codex · Corrigir autenticação Microsoft',
    );
  });

  it('falls back to provider session when prompt is empty', () => {
    expect(buildDirectAgentName('', 'claude')).toBe('Claude · Direct Session');
  });

  it('only offers authenticated and available resources and prefers the requested provider', () => {
    const accounts = [
      { id: '1', provider: 'codex', profile: 'work', authenticated: true, available: true },
      { id: '2', provider: 'claude', profile: 'personal', authenticated: true, available: true },
      { id: '3', provider: 'codex', profile: 'blocked', authenticated: true, available: false },
      { id: '4', provider: 'gemini', profile: 'anon', authenticated: false, available: true },
    ];
    expect(eligibleDirectResources(accounts, 'claude').map((item) => item.id)).toEqual(['2', '1']);
  });

  it('prefers the account display name and skips duplicate provider chips', () => {
    expect(
      directAccountTitle({
        display_name: 'codex (kiver.omegasistemas)',
        provider: 'codex',
        profile: 'kiver.omegasistemas',
      }),
    ).toBe('codex (kiver.omegasistemas)');
    expect(directAccountTitle({ provider: 'claude', profile: 'default' })).toBe('claude');
  });

  it('formats known quota as a percent and keeps unknown as null', () => {
    expect(directQuotaPercent({ quota_remaining: 0.7, quota_view: { status: 'LIVE' } })).toBe(70);
    expect(directQuotaPercent({ quota_remaining: 40, quota_view: { status: 'LIVE' } })).toBe(40);
    expect(directQuotaPercent({ quota_remaining: 40, quota_view: { status: 'OK' } })).toBeNull();
    expect(
      directQuotaPercent({ quota_remaining: 0.7, avail_reasons: { unknown_quota: true } }),
    ).toBeNull();
  });

  it('does not paint unknown or stale observations green', () => {
    expect(quotaTruthState({ available: true, quota_view: { status: 'UNKNOWN' } })).toBe('unknown');
    expect(quotaTruthState({ available: true, quota_view: { status: 'CACHED' } })).toBe('stale');
    expect(
      quotaTruthState({
        available: false,
        avail_reasons: { exhausted_windows: ['5h'] },
        quota_view: { status: 'LIVE' },
      }),
    ).toBe('exhausted');
  });

  it('scores the best usable group instead of the global bottleneck for AGY', () => {
    expect(
      directQuotaPercent({
        quota_remaining: 0,
        quota_view: {
          status: 'CACHED',
          model_groups: [
            {
              windows: [
                { kind: '5h', remaining: 100 },
                { kind: 'weekly', remaining: 100 },
              ],
            },
          ],
        },
      }),
    ).toBe(100);
    const agyView = {
      status: 'CACHED',
      model_groups: [
        {
          name: 'Gemini Models',
          windows: [
            { kind: '5h', remaining: 0 },
            { kind: 'weekly', remaining: 66 },
          ],
        },
        {
          name: 'Claude & GPT Models',
          windows: [
            { kind: 'claude_5h', remaining: 100 },
            { kind: 'claude_weekly', remaining: 100 },
          ],
        },
      ],
    };
    expect(bestGroupRemainingFromQuotaView(agyView)).toBe(100);
    expect(directQuotaPercent({ quota_remaining: 0, quota_view: agyView })).toBe(100);
    expect(compactQuotaLabel(agyView)).toBe('Gemini 0% · Claude 100%');
    expect(directQuotaDisplay({ quota_remaining: 0, quota_view: agyView })).toBe(
      'Gemini 0% · Claude 100%',
    );
  });

  it('keeps Codex 5h and weekly in the same pool', () => {
    const view = {
      status: 'CACHED',
      model_groups: [
        {
          name: 'Claude & GPT Models',
          windows: [
            { kind: '5h', remaining: 0 },
            { kind: 'weekly', remaining: 90 },
          ],
        },
      ],
    };
    expect(directQuotaPercent({ quota_remaining: 0.9, quota_view: view })).toBe(0);
    expect(compactQuotaLabel(view)).toBe('5h 0% · weekly 90%');
  });
});
