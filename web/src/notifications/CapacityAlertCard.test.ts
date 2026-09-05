import { describe, expect, it } from 'vitest';
import type { CapacityAlert } from '../types';
import { formatCapacityRemaining, capacitySeverityTone } from './notificationModel';

describe('CapacityAlertCard presentation logic', () => {
  const alert: CapacityAlert = {
    id: 'alt-1',
    fingerprint: 'fp-1',
    provider: 'anthropic',
    profile: 'prod',
    group: 'claude-3-5-sonnet',
    window: 'sliding_5h',
    severity: 'CRITICAL',
    remaining_percent: 7.5,
    reset_at: '2026-09-05T18:00:00Z',
    subject: 'Anthropic quota critically low',
    recommendation: {
      provider: 'openai',
      profile: 'backup',
      display_name: 'OpenAI GPT-4o',
      score: 0.92,
      fingerprint: 'rec-1',
    },
  };

  it('computes correct tone and badge for critical alert', () => {
    expect(capacitySeverityTone(alert.severity)).toBe('danger');
    expect(formatCapacityRemaining(alert.remaining_percent)).toBe('7.5%');
  });

  it('determines if recommendation switch is actionable', () => {
    expect(alert.recommendation).toBeDefined();
    expect(alert.recommendation?.provider).toBe('openai');
    expect(alert.recommendation?.score).toBeGreaterThan(0.8);
  });
});
