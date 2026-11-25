import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { NexMotionClock } from './nexMotionClock';

describe('NexMotionClock single animation clock', () => {
  beforeEach(() => {
    vi.stubGlobal('requestAnimationFrame', () => 42);
    vi.stubGlobal('cancelAnimationFrame', vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('paces idle mode at ~20 FPS (50ms interval) and stops RAF when paused', () => {
    let draws = 0;
    const clock = new NexMotionClock(() => {
      draws += 1;
    });

    clock.setMode('idle');
    expect(clock.isRunning()).toBe(true);

    // Step through 200ms at 16ms increments (~12 RAF ticks -> ~4-5 draws at 50ms).
    for (let ts = 0; ts <= 200; ts += 16) {
      clock.step(ts);
    }
    expect(draws).toBeGreaterThanOrEqual(4);
    expect(draws).toBeLessThanOrEqual(6);
    expect(clock.getMetrics().targetFps).toBe(20);

    clock.setMode('paused');
    expect(clock.isRunning()).toBe(false);
    const frozenDraws = draws;
    clock.step(300);
    expect(draws).toBe(frozenDraws);
  });

  it('paces interactive mode up to 60 FPS and tracks frame percentiles', () => {
    let draws = 0;
    const clock = new NexMotionClock(() => {
      draws += 1;
    });

    clock.setMode('interactive');
    for (let ts = 0; ts <= 320; ts += 16) {
      clock.step(ts);
    }
    const metrics = clock.getMetrics();
    expect(draws).toBeGreaterThanOrEqual(16);
    expect(metrics.targetFps).toBe(60);
    expect(metrics.p95FrameMs).toBeLessThanOrEqual(33);
    expect(metrics.longStuttersOver100ms).toBe(0);
    clock.destroy();
  });
});
