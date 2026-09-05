import { describe, expect, it } from 'vitest';
import {
  notificationTitle,
  capacityAlertFromEvent,
  formatCapacityRemaining,
  capacitySeverityTone,
} from './notificationModel';

describe('notificationTitle', () => {
  it('uses the provider/runtime dynamic title when available', () => {
    expect(notificationTitle('proxy-nginx', 'QUESTION', 'Deploy requires approval')).toBe(
      '[Nexus | proxy-nginx] Deploy requires approval',
    );
  });

  it('keeps semantic fallback titles for attention reasons', () => {
    expect(notificationTitle(undefined, 'APPROVAL')).toContain('Aprovação Necessária');
  });
});

describe('capacityAlertFromEvent', () => {
  it('parses structured payload_json into CanonicalAlert shape', () => {
    const rawEvent = {
      id: 'evt-1',
      fingerprint: 'fp-1',
      event_type: 'capacity.alert',
      provider: 'claude',
      profile: 'default',
      quota_group: 'requests',
      window_kind: 'sliding_5h',
      severity: 'WARNING',
      payload_json: JSON.stringify({
        id: 'alt-1',
        fingerprint: 'fp-1',
        provider: 'claude',
        profile: 'default',
        group: 'requests',
        window: 'sliding_5h',
        severity: 'WARNING',
        remaining_percent: 18.5,
        subject: 'Claude capacity warning',
        recommendation: {
          provider: 'codex',
          profile: 'prod',
          display_name: 'Codex Prod',
          score: 0.95,
          fingerprint: 'rec-fp-1',
        },
      }),
      emitted_at: '2026-09-05T12:00:00Z',
    };

    const alert = capacityAlertFromEvent(rawEvent);
    expect(alert).not.toBeNull();
    expect(alert?.provider).toBe('claude');
    expect(alert?.remaining_percent).toBe(18.5);
    expect(alert?.recommendation?.provider).toBe('codex');
  });

  it('returns null for invalid or corrupted event payloads', () => {
    expect(capacityAlertFromEvent(null as any)).toBeNull();
    expect(capacityAlertFromEvent({} as any)).toBeNull();
    expect(capacityAlertFromEvent({ payload_json: 'invalid-json' })).toBeNull();
  });
});

describe('formatCapacityRemaining', () => {
  it('formats percentages clearly', () => {
    expect(formatCapacityRemaining(18.5)).toBe('18.5%');
    expect(formatCapacityRemaining(0)).toBe('0.0%');
    expect(formatCapacityRemaining(null)).toBe('N/A');
    expect(formatCapacityRemaining(undefined)).toBe('N/A');
  });
});

describe('capacitySeverityTone', () => {
  it('maps severity to UI tone tokens', () => {
    expect(capacitySeverityTone('WARNING')).toBe('warning');
    expect(capacitySeverityTone('CRITICAL')).toBe('danger');
    expect(capacitySeverityTone('EXHAUSTED')).toBe('danger');
    expect(capacitySeverityTone('RECOVERED')).toBe('success');
    expect(capacitySeverityTone('UNKNOWN')).toBe('default');
  });
});
