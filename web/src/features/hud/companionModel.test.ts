import { describe, expect, it } from 'vitest';
import {
  companionMood,
  nextCompanionBubble,
  projectCompanionAttention,
  readCompanionSnoozes,
  snoozeCompanionAttention,
} from './companionModel';
import type { HudState } from './hudModel';

const snapshot = (changes: Partial<HudState> = {}): HudState => ({
  generated_at: '2026-10-21T00:00:00Z',
  providers: [],
  missions: [],
  agents: [],
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
  ...changes,
});

describe('Nexus Companion attention projection', () => {
  it('announces a real user intervention without inventing an action for completed missions', () => {
    const state = snapshot({
      missions: [
        {
          run_id: 'one',
          project_id: 'a',
          state: 'BLOCKED_NEEDS_USER',
          current_stage: 'execution',
          completed_packages: 2,
          total_packages: 5,
          needs_user: true,
          intervention_reason: 'Aprovação aguardando.',
          updated_at: '2026-10-21T00:00:00Z',
        },
        {
          run_id: 'done',
          project_id: 'a',
          state: 'COMPLETED_VERIFIED',
          current_stage: 'complete',
          completed_packages: 5,
          total_packages: 5,
          needs_user: true,
          updated_at: '2026-10-21T00:00:00Z',
        },
      ],
    });
    const attention = projectCompanionAttention(state);
    expect(attention).toHaveLength(1);
    expect(attention[0].target).toBe('missions');
    expect(attention[0].needsUser).toBe(true);
    expect(companionMood(state, attention, false)).toBe('critical');
  });

  it('ignores acknowledged and snoozed provider events', () => {
    const alert = (fingerprint: string, acked = false, snoozed_until?: string) => ({
      fingerprint,
      category: 'CAPACITY',
      severity: 'CRITICAL',
      acked,
      snoozed_until,
      last_observed_at: '2026-10-21T00:00:00Z',
    });
    const state = snapshot({
      alerts: [
        alert('read', true),
        alert('later', false, '2026-10-22T00:00:00Z'),
        alert('live'),
      ],
    });
    expect(projectCompanionAttention(state, Date.parse('2026-10-21T00:00:00Z'))).toHaveLength(1);
    expect(projectCompanionAttention(state, Date.parse('2026-10-23T00:00:00Z'))).toHaveLength(2);
  });

  it('distinguishes working, idle and offline without inventing quotas', () => {
    const working = snapshot({
      agents: [{ id: '1', project_id: 'p', name: 'Coder', status: 'WORKING' }],
    });
    expect(companionMood(working, [], false)).toBe('working');
    expect(companionMood(snapshot(), [], false)).toBe('idle');
    expect(companionMood(working, [], true)).toBe('offline');
  });

  it('bounds snoozes and never acknowledges the original event', () => {
    const values = new Map<string, string>();
    const storage = {
      getItem: (key: string) => values.get(key) || null,
      setItem: (key: string, value: string) => { values.set(key, value); },
    };
    const now = Date.parse('2026-10-21T10:00:00Z');
    const item = {
      id: 'mission:p:42',
      title: 'Aguarda você',
      detail: 'Revisão',
      severity: 'critical' as const,
      target: 'missions' as const,
      needsUser: true,
    };
    snoozeCompanionAttention(storage, item.id, now);
    expect(nextCompanionBubble([item], readCompanionSnoozes(storage, now), now)).toBeUndefined();
    expect(nextCompanionBubble([item], readCompanionSnoozes(storage, now + 3600001), now + 3600001)).toEqual(item);
  });
});
