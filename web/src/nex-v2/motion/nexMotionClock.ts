export type NexPacingMode = 'paused' | 'idle' | 'interactive';

export interface NexClockMetrics {
  mode: NexPacingMode;
  targetFps: number;
  frameCount: number;
  p50FrameMs: number;
  p95FrameMs: number;
  maxFrameMs: number;
  longStuttersOver100ms: number;
}

const IDLE_INTERVAL_MS = 50; // 20 FPS within 15-30 FPS budget
const INTERACTIVE_INTERVAL_MS = 16; // 60 FPS target (matches 16ms/16.67ms vsync ticks)
const MAX_DELTA_SEC = 0.05;
const SAMPLE_CAPACITY = 120;

/**
 * Single animation clock for Nex 2.0.
 * - Pauses completely (0 RAF callbacks) when hidden, offline, or reduced-motion.
 * - Paces idle rendering at 20 FPS to keep CPU <= 1% on Linux/Windows.
 * - Paces interactive/working expressions up to 60 FPS with rolling frame stats.
 */
export class NexMotionClock {
  private rafId: number | null = null;
  private mode: NexPacingMode = 'paused';
  private lastTimestampMs: number | null = null;
  private lastDrawMs: number | null = null;
  private simTimeSec = 0;
  private readonly frameDeltas: number[] = [];
  private totalFrames = 0;
  private stutters = 0;
  private readonly onTick: (simTimeSec: number, dtSec: number) => void;

  constructor(onTick: (simTimeSec: number, dtSec: number) => void) {
    this.onTick = onTick;
  }

  setMode(next: NexPacingMode): void {
    if (this.mode === next) return;
    this.mode = next;
    if (next === 'paused') {
      this.stopRaf();
      this.lastTimestampMs = null;
      this.lastDrawMs = null;
      return;
    }
    if (this.rafId === null && typeof requestAnimationFrame === 'function') {
      this.lastTimestampMs = null;
      this.lastDrawMs = null;
      this.rafId = requestAnimationFrame((ts) => this.step(ts));
    }
  }

  getMode(): NexPacingMode {
    return this.mode;
  }

  isRunning(): boolean {
    return this.rafId !== null;
  }

  /**
   * Advances the clock deterministically for unit tests or RAF callbacks.
   */
  step(timestampMs: number): void {
    this.rafId = null;
    if (this.mode === 'paused') return;

    const dtSec =
      this.lastTimestampMs !== null
        ? Math.max(0, Math.min(MAX_DELTA_SEC, (timestampMs - this.lastTimestampMs) / 1000))
        : 0;
    this.lastTimestampMs = timestampMs;
    this.simTimeSec += dtSec;

    const minIntervalMs = this.mode === 'idle' ? IDLE_INTERVAL_MS : INTERACTIVE_INTERVAL_MS;
    const elapsedSinceDraw =
      this.lastDrawMs !== null ? timestampMs - this.lastDrawMs : minIntervalMs;

    if (elapsedSinceDraw >= minIntervalMs - 0.5) {
      if (this.lastDrawMs !== null) {
        this.recordFrameInterval(elapsedSinceDraw);
      }
      this.lastDrawMs = timestampMs;
      this.onTick(this.simTimeSec, dtSec);
    }

    if (!this.isPaused() && typeof requestAnimationFrame === 'function') {
      this.rafId = requestAnimationFrame((ts) => this.step(ts));
    }
  }

  private isPaused(): boolean {
    return this.mode === 'paused';
  }

  private recordFrameInterval(ms: number): void {
    this.totalFrames += 1;
    if (ms > 100) this.stutters += 1;
    this.frameDeltas.push(ms);
    if (this.frameDeltas.length > SAMPLE_CAPACITY) {
      this.frameDeltas.shift();
    }
  }

  getMetrics(): NexClockMetrics {
    const sorted = [...this.frameDeltas].sort((a, b) => a - b);
    const percentile = (p: number) => {
      if (sorted.length === 0) return 0;
      const idx = Math.min(sorted.length - 1, Math.floor(sorted.length * p));
      return Number(sorted[idx].toFixed(2));
    };
    const targetFps = this.mode === 'paused' ? 0 : this.mode === 'idle' ? 20 : 60;
    return {
      mode: this.mode,
      targetFps,
      frameCount: this.totalFrames,
      p50FrameMs: percentile(0.5),
      p95FrameMs: percentile(0.95),
      maxFrameMs: sorted.length > 0 ? Number(sorted[sorted.length - 1].toFixed(2)) : 0,
      longStuttersOver100ms: this.stutters,
    };
  }

  destroy(): void {
    this.mode = 'paused';
    this.stopRaf();
  }

  private stopRaf(): void {
    if (this.rafId !== null && typeof cancelAnimationFrame === 'function') {
      cancelAnimationFrame(this.rafId);
    }
    this.rafId = null;
  }
}
