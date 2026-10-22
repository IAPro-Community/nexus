import { describe, expect, it } from 'vitest';
import {
  bestGroupRemainingFromQuotaView,
  buildDirectAgentName,
  compactQuotaLabel,
  directAccountTitle,
  directQuotaDisplay,
  directQuotaPercent,
  eligibleDirectResources,
  recommendedDirectAccountId,
  directResourcePolicy,
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

  it('offers operational manual resources, including opaque quota, and prefers the requested provider', () => {
    const accounts = [
      { id: '1', provider: 'codex', profile: 'work', authenticated: true, available: true },
      { id: '2', provider: 'claude', profile: 'personal', authenticated: true, available: true },
      { id: '3', provider: 'codex', profile: 'blocked', authenticated: true, available: false },
      { id: '4', provider: 'gemini', profile: 'anon', authenticated: false, available: true },
      {
        id: '5',
        provider: 'opencode',
        profile: 'manual',
        authenticated: true,
        available: false,
        eligibility: { eligible: true, reason: 'UNSUPPORTED_QUOTA', confidence: 'UNSUPPORTED' },
      },
      {
        id: '6',
        provider: 'agy',
        profile: 'cooldown',
        authenticated: true,
        available: false,
        eligibility: { eligible: false, reason: 'COOLDOWN', confidence: 'UNKNOWN' },
      },
    ];
    expect(eligibleDirectResources(accounts, 'claude').map((item) => item.id)).toEqual([
      '2',
      '1',
      '5',
    ]);
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

  it.each(['ERROR', 'UNSUPPORTED'])(
    'does not render labels for error or unsupported quota evidence: %s',
    (status) => {
      const view = {
        status,
        model_groups: [
          {
            name: 'Codex',
            windows: [{ kind: '5h', remaining: 0 }],
          },
        ],
      };
      expect(directQuotaPercent({ quota_remaining: 0, quota_view: view })).toBeNull();
      expect(directQuotaDisplay({ quota_remaining: 0, quota_view: view })).toBeNull();
    },
  );
});

describe('scheduler-authoritative Direct selection', () => {
  const accounts = [
    {
      id: 'codex-default',
      provider: 'codex',
      profile: 'default',
      authenticated: true,
      available: true,
    },
    {
      id: 'claude-work',
      provider: 'claude',
      profile: 'work',
      authenticated: true,
      available: true,
    },
  ];
  it('uses backend recommendation rather than first/alphabetical account', () => {
    expect(
      recommendedDirectAccountId(accounts, { recommended: { account: { id: 'claude-work' } } }),
    ).toBe('claude-work');
  });
  it('does not invent a selection for missing or ineligible recommendation', () => {
    expect(
      recommendedDirectAccountId(accounts, { recommended: { account: { id: 'missing' } } }),
    ).toBe('');
    expect(recommendedDirectAccountId(accounts, null)).toBe('');
  });
});

describe('project resource policy normalization', () => {
  it('supports a legacy enum and a project JSON configuration', () => {
    expect(directResourcePolicy('PRESERVE_QUOTA')).toBe('PRESERVE_QUOTA');
    expect(directResourcePolicy('{"policy":"PREFER_PROVIDER"}')).toBe('PREFER_PROVIDER');
    expect(directResourcePolicy('{"scheduler_policy":"PRESERVE_QUOTA"}')).toBe('PRESERVE_QUOTA');
  });
  it('fails to BALANCED on unknown, malformed or empty policy', () => {
    expect(directResourcePolicy('{}')).toBe('BALANCED');
    expect(directResourcePolicy('{broken')).toBe('BALANCED');
    expect(directResourcePolicy('CUSTOM')).toBe('BALANCED');
  });
});
