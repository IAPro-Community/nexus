import { describe, expect, it } from 'vitest';
import {
  consumeNexNewCompletion,
  nexCompletedEventId,
  resolveNexCompanionBehavior,
  type NexCompanionInput,
} from './nexCompanionBehavior';

const base: NexCompanionInput = {
  mood: 'idle',
  idleAction: null,
  interaction: 'none',
  resolved: false,
  motionAllowed: true,
};

describe('NEX Companion 2D', () => {
  it('maps the eight contextual behaviors using the existing five operational moods', () => {
    expect(resolveNexCompanionBehavior(base)).toBe('ambient');
    expect(resolveNexCompanionBehavior({ ...base, interaction: 'greet' })).toBe('greet');
    expect(resolveNexCompanionBehavior({ ...base, interaction: 'look' })).toBe('look');
    expect(resolveNexCompanionBehavior({ ...base, mood: 'working' })).toBe('thinking');
    expect(resolveNexCompanionBehavior({ ...base, mood: 'attention' })).toBe('attention');
    expect(resolveNexCompanionBehavior({ ...base, mood: 'critical' })).toBe('error');
    expect(resolveNexCompanionBehavior({ ...base, resolved: true })).toBe('resolved');
    expect(resolveNexCompanionBehavior({ ...base, idleAction: 'idle_drowse' })).toBe('rest');
  });

  it('preempts user gestures with canonical operational status', () => {
    expect(resolveNexCompanionBehavior({ ...base, mood: 'critical', resolved: true, interaction: 'look' }))
      .toBe('error');
    expect(resolveNexCompanionBehavior({ ...base, mood: 'attention', resolved: true, interaction: 'greet' }))
      .toBe('attention');
    expect(resolveNexCompanionBehavior({ ...base, mood: 'working', interaction: 'look' }))
      .toBe('thinking');
    expect(resolveNexCompanionBehavior({ ...base, mood: 'offline' })).toBe('still');
    expect(resolveNexCompanionBehavior({ ...base, motionAllowed: false })).toBe('still');
  });

  it('refuses to infer a success from missing IDs, failures or stale history', () => {
    expect(nexCompletedEventId({ project_id: '', mission_id: 'a', state: 'COMPLETED', age_seconds: 2 }))
      .toBeNull();
    expect(nexCompletedEventId({ project_id: 'p', mission_id: 'a', state: 'FAILED', age_seconds: 2 }))
      .toBeNull();
    expect(nexCompletedEventId({ project_id: 'p', mission_id: 'a', state: 'COMPLETED', age_seconds: 3600 }))
      .toBeNull();
  });

  it('baselines first fetch, deduplicates refreshing and emits only newly completed missions', () => {
    const seen = new Set<string>();
    const first = [{ project_id: 'p', mission_id: 'old', state: 'COMPLETED', age_seconds: 3 }];
    const added = [...first, { project_id: 'p', mission_id: 'new', state: 'COMPLETED', age_seconds: 1 }];
    expect(consumeNexNewCompletion(first, seen, false)).toBeNull();
    expect(consumeNexNewCompletion(added, seen, true)).toBe('p:new:COMPLETED');
    expect(consumeNexNewCompletion(added, seen, true)).toBeNull();
  });
});
