import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  chooseNexIdleAction,
  createNexRandom,
  NEX_IDLE_ACTIONS,
  NexIdleDirector,
  type NexIdleAction,
  type NexIdleEnvironment,
} from './nexIdleDirector';

const IDLE: NexIdleEnvironment = {
  mood: 'idle',
  visible: true,
  motionEnabled: true,
  reducedMotion: false,
};

afterEach(() => {
  vi.useRealTimers();
});

describe('NEX IdleDirector', () => {
  it('enumerates 12 canonical idle actions without duplicates', () => {
    expect(NEX_IDLE_ACTIONS).toHaveLength(12);
    expect(new Set(NEX_IDLE_ACTIONS).size).toBe(12);
  });

  it('chooses deterministically and avoids immediate repeats', () => {
    const randomA = createNexRandom(1729);
    const randomB = createNexRandom(1729);
    expect([randomA(), randomA(), randomA()]).toEqual([
      randomB(),
      randomB(),
      randomB(),
    ]);
    expect(chooseNexIdleAction(() => 0, 'idle_blink')).not.toBe('idle_blink');
  });

  it('keeps one timer chain and emits no synthetic operational state', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-10-29T10:00:00Z'));
    const actions: Array<NexIdleAction | null> = [];
    const director = new NexIdleDirector((action) => actions.push(action));
    director.updateEnvironment(IDLE);
    director.start();
    director.start();
    expect(vi.getTimerCount()).toBe(1);
    vi.advanceTimersByTime(45000);
    expect(actions.some((action) => action?.startsWith('idle_'))).toBe(true);
    director.stop();
    expect(director.currentAction).toBeNull();
    expect(vi.getTimerCount()).toBe(0);
  });

  it('preempts idle immediately on working, warning, critical and offline', () => {
    for (const mood of ['working', 'warning', 'critical', 'offline'] as const) {
      vi.useFakeTimers();
      const director = new NexIdleDirector(() => undefined);
      director.updateEnvironment(IDLE);
      director.start();
      vi.advanceTimersByTime(30000);
      director.updateEnvironment({ ...IDLE, mood });
      expect(director.currentAction).toBeNull();
      expect(vi.getTimerCount()).toBe(0);
      director.stop();
      vi.clearAllTimers();
    }
  });

  it('pauses on reduced-motion, invisibility and disabled motion', () => {
    for (const change of [
      { reducedMotion: true },
      { visible: false },
      { motionEnabled: false },
    ]) {
      vi.useFakeTimers();
      const director = new NexIdleDirector(() => undefined);
      director.updateEnvironment(IDLE);
      director.start();
      director.updateEnvironment({ ...IDLE, ...change });
      vi.advanceTimersByTime(300000);
      expect(director.currentAction).toBeNull();
      expect(vi.getTimerCount()).toBe(0);
      director.stop();
      vi.clearAllTimers();
    }
  });

  it('allows waking only after an actual drowse and user interaction', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-10-29T10:00:00Z'));
    const actions: Array<NexIdleAction | null> = [];
    const director = new NexIdleDirector((action) => actions.push(action));
    director.updateEnvironment(IDLE);
    director.start();
    vi.advanceTimersByTime(230000);
    expect(actions).toContain('idle_drowse');
    director.notifyInteraction();
    expect(director.currentAction).toBe('idle_wake');
    vi.advanceTimersByTime(1700);
    expect(director.currentAction).toBeNull();
    director.stop();
  });

  it('cleans its timer on destroy even if no action has played yet', () => {
    vi.useFakeTimers();
    const director = new NexIdleDirector(() => undefined);
    director.updateEnvironment(IDLE);
    director.start();
    expect(vi.getTimerCount()).toBe(1);
    director.stop();
    expect(vi.getTimerCount()).toBe(0);
  });
});
