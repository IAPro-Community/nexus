import { describe, expect, it } from 'vitest';
import { capacityAlertFromEvent, type CapacityAlertCopy } from './inAppNotificationModel';
import type { EventRecord } from '../types';

const copy: CapacityAlertCopy = {
  lowTitle: (provider, remaining) => `Consumo: ${provider} (${remaining})`,
  exhaustedTitle: (provider) => `Esgotada: ${provider}`,
  recoveredTitle: (provider) => `Recuperada: ${provider}`,
  stateLabel: (state) => state,
  resetLabel: (reset) => `reset ${reset}`,
  freshnessLabel: (age) => `leitura ${age}`,
  recommendationLabel: (name, reason) => (reason ? `${name} — ${reason}` : name),
};

const base: EventRecord = {
  id: 'evt-1',
  runtime_id: '',
  type: 'QUOTA_LOW',
  summary: '',
  data: {},
  timestamp: new Date().toISOString(),
};

describe('capacityAlertFromEvent', () => {
  it('returns null for legacy events without monitor state', () => {
    expect(capacityAlertFromEvent(base, copy)).toBeNull();
    expect(capacityAlertFromEvent({ ...base, type: 'QUOTA_MONITOR_DEGRADED' }, copy)).toBeNull();
  });

  it('keys the id by fingerprint so repeats collapse', () => {
    const data = {
      state: 'WARNING',
      fingerprint: 'alert-abc',
      provider: 'codex',
      remaining_percent: 15,
    };
    const first = capacityAlertFromEvent({ ...base, data }, copy);
    const second = capacityAlertFromEvent({ ...base, id: 'evt-2', data }, copy);
    expect(first).not.toBeNull();
    expect(second).not.toBeNull();
    expect(first!.id).toBe('capacity:alert-abc');
    expect(second!.id).toBe(first!.id);
    expect(first!.tone).toBe('warning');
  });

  it('maps exhausted to danger with handoff action when recommended', () => {
    const notif = capacityAlertFromEvent(
      {
        ...base,
        type: 'QUOTA_EXHAUSTED',
        data: {
          state: 'EXHAUSTED',
          fingerprint: 'alert-exh',
          provider: 'codex',
          profile: 'work',
          recommended_provider: 'agy',
          recommended_profile: 'main',
          recommended_display_name: 'AGY Main',
        },
      },
      copy,
    );
    expect(notif).toMatchObject({ tone: 'danger', title: 'Esgotada: codex' });
    expect(notif!.action).toMatchObject({
      type: 'context_continue',
      targetProvider: 'agy',
      targetProfile: 'main',
      isCrossProvider: true,
    });
  });

  it('maps recovery to success tone', () => {
    const notif = capacityAlertFromEvent(
      {
        ...base,
        type: 'QUOTA_MONITOR_RECOVERED',
        data: { state: 'RECOVERED', fingerprint: 'alert-rec', provider: 'codex' },
      },
      copy,
    );
    expect(notif).toMatchObject({ tone: 'success', title: 'Recuperada: codex' });
    expect(notif!.action).toBeUndefined();
  });

  it('composes reset, freshness and recommendation into the message', () => {
    const notif = capacityAlertFromEvent(
      {
        ...base,
        data: {
          state: 'CRITICAL',
          fingerprint: 'alert-crit',
          provider: 'codex',
          remaining_percent: 8,
          reset_desc: '2h',
          data_age: '1m',
          recommended_display_name: 'AGY Main',
          recommendation_reason: 'healthy window',
        },
      },
      copy,
    );
    expect(notif!.message).toContain('reset 2h');
    expect(notif!.message).toContain('leitura 1m');
    expect(notif!.message).toContain('AGY Main — healthy window');
  });
});
