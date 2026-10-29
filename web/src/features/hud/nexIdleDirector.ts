/**
 * NEX animation choreography: visual-only; cannot infer Core status, progress or success.
 *
 * A director owns one timer chain. Site and native HUD renderers may subscribe
 * without requiring another LLM request or changing core telemetry.
 * The 12 ids describe the approved target choreography, not completed art.
 */
export const NEX_IDLE_ACTIONS = [
  'idle_hover',
  'idle_blink',
  'idle_glance',
  'idle_orbit_breathe',
  'idle_head_tilt',
  'idle_fin_twitch',
  'idle_module_inspect',
  'idle_orbit_shuffle',
  'idle_curiosity',
  'idle_micro_stretch',
  'idle_drowse',
  'idle_wake',
] as const;

export type NexIdleAction = (typeof NEX_IDLE_ACTIONS)[number];
export type NexIdleMood = 'idle' | 'working' | 'warning' | 'critical' | 'offline';

export interface NexIdleEnvironment {
  mood: NexIdleMood;
  motionEnabled: boolean;
  visible: boolean;
  reducedMotion: boolean;
}

export interface NexIdleTiming {
  setTimeout(callback: () => void, delayMs: number): ReturnType<typeof setTimeout>;
  clearTimeout(handle: ReturnType<typeof setTimeout>): void;
  now(): number;
}

const DEFAULT_TIMING: NexIdleTiming = {
  setTimeout: (callback, ms) => setTimeout(callback, ms),
  clearTimeout: (handle) => clearTimeout(handle),
  now: () => Date.now(),
};

const STANDARD_ACTIONS: readonly NexIdleAction[] = [
  'idle_blink',
  'idle_glance',
  'idle_orbit_breathe',
  'idle_head_tilt',
  'idle_fin_twitch',
  'idle_module_inspect',
  'idle_orbit_shuffle',
  'idle_curiosity',
  'idle_micro_stretch',
];

export const NEX_IDLE_DURATION_MS: Readonly<Record<NexIdleAction, number>> = {
  idle_hover: 2400,
  idle_blink: 650,
  idle_glance: 1700,
  idle_orbit_breathe: 2100,
  idle_head_tilt: 2200,
  idle_fin_twitch: 1500,
  idle_module_inspect: 2900,
  idle_orbit_shuffle: 3100,
  idle_curiosity: 2400,
  idle_micro_stretch: 2600,
  idle_drowse: 3800,
  idle_wake: 1700,
};

const DEFAULT_ENV: NexIdleEnvironment = {
  mood: 'offline',
  motionEnabled: false,
  visible: false,
  reducedMotion: true,
};

/** Deterministic, session-local PRNG for reproducible animation schedules. */
export function createNexRandom(seed: number): () => number {
  let state = seed >>> 0;
  return () => {
    state = (Math.imul(1664525, state) + 1013904223) >>> 0;
    return state / 4294967296;
  };
}

export function chooseNexIdleAction(
  random: () => number,
  previous: NexIdleAction | null,
): NexIdleAction {
  const choices = STANDARD_ACTIONS.filter((action) => action !== previous);
  const value = random();
  const index = Math.min(choices.length - 1, Math.floor(Math.max(0, value) * choices.length));
  return choices[index];
}

export class NexIdleDirector {
  private environment: NexIdleEnvironment = DEFAULT_ENV;
  private readonly timing: NexIdleTiming;
  private readonly random: () => number;
  private readonly onChange: (action: NexIdleAction | null) => void;
  private timer: ReturnType<typeof setTimeout> | null = null;
  private endTimer: ReturnType<typeof setTimeout> | null = null;
  private running = false;
  private current: NexIdleAction | null = null;
  private previous: NexIdleAction | null = null;
  private lastInteraction: number;
  private drowsed = false;

  constructor(
    onChange: (action: NexIdleAction | null) => void,
    options: { seed?: number; timing?: NexIdleTiming } = {},
  ) {
    this.onChange = onChange;
    this.timing = options.timing ?? DEFAULT_TIMING;
    this.random = createNexRandom(options.seed ?? 1729);
    this.lastInteraction = this.timing.now();
  }

  get currentAction(): NexIdleAction | null {
    return this.current;
  }

  start(): void {
    if (this.running) return;
    this.running = true;
    this.lastInteraction = this.timing.now();
    this.schedule();
  }

  stop(): void {
    this.running = false;
    this.cancel();
    this.previous = null;
    this.drowsed = false;
  }

  updateEnvironment(next: NexIdleEnvironment): void {
    const wasAllowed = this.allowed();
    this.environment = next;
    if (!this.allowed()) {
      this.cancel();
      this.drowsed = false;
    } else if (!wasAllowed && this.running) {
      this.lastInteraction = this.timing.now();
      this.schedule();
    }
  }

  /** Called for actual user interaction, not a guessed Core event. */
  notifyInteraction(): void {
    this.lastInteraction = this.timing.now();
    if (!this.allowed()) return;
    const shouldWake = this.drowsed;
    this.cancel();
    this.drowsed = false;
    if (shouldWake) this.play('idle_wake');
    else this.schedule();
  }

  private allowed(): boolean {
    return (
      this.running &&
      this.environment.mood === 'idle' &&
      this.environment.motionEnabled &&
      this.environment.visible &&
      !this.environment.reducedMotion
    );
  }

  private cancel(): void {
    if (this.timer !== null) this.timing.clearTimeout(this.timer);
    if (this.endTimer !== null) this.timing.clearTimeout(this.endTimer);
    this.timer = null;
    this.endTimer = null;
    this.setCurrent(null);
  }

  private setCurrent(action: NexIdleAction | null): void {
    if (this.current === action) return;
    this.current = action;
    this.onChange(action);
  }

  private play(action: NexIdleAction): void {
    if (!this.allowed()) return;
    this.previous = action;
    this.setCurrent(action);
    this.endTimer = this.timing.setTimeout(() => {
      this.endTimer = null;
      this.setCurrent(null);
      this.schedule();
    }, NEX_IDLE_DURATION_MS[action]);
  }

  private schedule(): void {
    if (!this.allowed() || this.timer !== null || this.endTimer !== null) return;
    // Keep the HUD calm. Do not exceed roughly 2 special gestures/minute.
    const delayMs = 24000 + Math.floor(this.random() * 12000);
    this.timer = this.timing.setTimeout(() => {
      this.timer = null;
      if (!this.allowed()) return;
      if (this.timing.now() - this.lastInteraction > 180000 && !this.drowsed) {
        this.drowsed = true;
        this.play('idle_drowse');
      } else if (!this.drowsed) {
        this.play(chooseNexIdleAction(this.random, this.previous));
      }
    }, delayMs);
  }
}
