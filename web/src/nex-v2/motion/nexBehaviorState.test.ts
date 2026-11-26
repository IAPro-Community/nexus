import { describe, expect, it } from 'vitest';
import { VerifiedCompletionTracker, resolveNexBehaviorState } from './nexBehaviorState';

describe('Nex 2.0 behavior precedence and completion tracking', () => {
  it('enforces offline > critical > attention > working > transient interaction > rest > ambient', () => {
    const base = {
      motionEnabled: true,
      reducedMotion: false,
      visible: true,
      localInteraction: 'greet' as const,
      idleAction: 'idle_drowse' as const,
      resolvedActive: true,
    };

    const offline = resolveNexBehaviorState({ ...base, mood: 'offline' });
    expect(offline.expression).toBe('still');
    expect(offline.animated).toBe(false);

    const critical = resolveNexBehaviorState({ ...base, mood: 'critical' });
    expect(critical.behavior).toBe('error');
    expect(critical.expression).toBe('error');

    const attention = resolveNexBehaviorState({ ...base, mood: 'attention' });
    expect(attention.behavior).toBe('attention');

    const working = resolveNexBehaviorState({ ...base, mood: 'working' });
    expect(working.behavior).toBe('thinking');
    expect(working.orbitMode).toBe('orbit');

    const resolved = resolveNexBehaviorState({ ...base, mood: 'idle' });
    expect(resolved.behavior).toBe('resolved');

    const greet = resolveNexBehaviorState({
      ...base,
      mood: 'idle',
      resolvedActive: false,
    });
    expect(greet.behavior).toBe('greet');

    const rest = resolveNexBehaviorState({
      ...base,
      mood: 'idle',
      resolvedActive: false,
      localInteraction: 'none',
    });
    expect(rest.behavior).toBe('rest');

    const chatThinking = resolveNexBehaviorState({
      ...base,
      mood: 'idle',
      thinkingActive: true,
    });
    expect(chatThinking.behavior).toBe('thinking');
    expect(chatThinking.orbitMode).toBe('orbit');

    const idleGlance = resolveNexBehaviorState({
      ...base,
      mood: 'idle',
      resolvedActive: false,
      localInteraction: 'none',
      idleAction: 'idle_glance',
    });
    expect(idleGlance.behavior).toBe('look');
  });

  it('freezes motion immediately under reduced-motion or hidden state', () => {
    const reduced = resolveNexBehaviorState({
      mood: 'working',
      motionEnabled: true,
      reducedMotion: true,
      visible: true,
      localInteraction: 'none',
      idleAction: null,
      resolvedActive: false,
    });
    expect(reduced.expression).toBe('still');
    expect(reduced.animated).toBe(false);

    const hidden = resolveNexBehaviorState({
      mood: 'idle',
      motionEnabled: true,
      reducedMotion: false,
      visible: false,
      localInteraction: 'greet',
      idleAction: null,
      resolvedActive: false,
    });
    expect(hidden.expression).toBe('still');
    expect(hidden.animated).toBe(false);
  });

  it('deduplicates verified completions using initial baseline', () => {
    const tracker = new VerifiedCompletionTracker();
    expect(tracker.consume(['evt-1'])).toBeNull();
    expect(tracker.consume(['evt-1'])).toBeNull();
    expect(tracker.consume(['evt-1', 'evt-2'])).toBe('evt-2');
    expect(tracker.consume(['evt-1', 'evt-2'])).toBeNull();
  });
});
