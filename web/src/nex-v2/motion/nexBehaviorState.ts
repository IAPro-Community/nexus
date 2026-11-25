import type { NexExpression } from './nexEnvelope';

export type NexOperationalMood = 'idle' | 'working' | 'attention' | 'critical' | 'offline';
export type NexBehavior =
  'ambient' | 'greet' | 'look' | 'thinking' | 'attention' | 'error' | 'resolved' | 'rest';

export type NexIdleActionId =
  | 'idle_hover'
  | 'idle_blink'
  | 'idle_glance'
  | 'idle_head_tilt'
  | 'idle_orbit_breathe'
  | 'idle_fin_twitch'
  | 'idle_drowse'
  | 'idle_wake';

export interface NexBehaviorInput {
  mood: NexOperationalMood;
  motionEnabled: boolean;
  reducedMotion: boolean;
  visible: boolean;
  localInteraction: 'none' | 'greet' | 'look';
  idleAction: NexIdleActionId | null;
  resolvedActive: boolean;
}

export interface NexBehaviorResolution {
  behavior: NexBehavior;
  expression: NexExpression;
  orbitMode: 'idle' | 'orbit';
  animated: boolean;
}

/**
 * Resolves the single authoritative Nex behavior and rig expression from
 * operational state and local hitbox interaction.
 *
 * Precedence: offline > critical > attention > working > transient interaction > rest > ambient.
 * Reduced-motion, hidden, or offline states freeze motion immediately ('still').
 */
export function resolveNexBehaviorState(input: NexBehaviorInput): NexBehaviorResolution {
  const motionAllowed =
    input.motionEnabled && !input.reducedMotion && input.visible && input.mood !== 'offline';

  let behavior: NexBehavior = 'ambient';
  if (input.mood === 'offline') {
    behavior = 'rest';
  } else if (input.mood === 'critical') {
    behavior = 'error';
  } else if (input.mood === 'attention') {
    behavior = 'attention';
  } else if (input.mood === 'working') {
    behavior = 'thinking';
  } else if (input.resolvedActive) {
    behavior = 'resolved';
  } else if (input.localInteraction === 'greet') {
    behavior = 'greet';
  } else if (input.localInteraction === 'look') {
    behavior = 'look';
  } else if (input.idleAction === 'idle_drowse') {
    behavior = 'rest';
  }

  if (!motionAllowed) {
    return {
      behavior,
      expression: 'still',
      orbitMode: input.mood === 'working' ? 'orbit' : 'idle',
      animated: false,
    };
  }

  const expressionMap: Record<NexBehavior, NexExpression> = {
    ambient: 'ambient',
    greet: 'greet',
    look: 'look',
    thinking: 'thinking',
    attention: 'attention',
    error: 'error',
    resolved: 'resolved',
    rest: 'rest',
  };

  return {
    behavior,
    expression: expressionMap[behavior],
    orbitMode: input.mood === 'working' ? 'orbit' : 'idle',
    animated: true,
  };
}

/**
 * Deduplicates one-shot verified completion celebration using a stable event ID.
 * Never infers completion from counter drops or absent data.
 */
export class VerifiedCompletionTracker {
  private readonly seen = new Set<string>();
  private initialized = false;

  consume(completedIds: readonly string[]): string | null {
    if (!this.initialized) {
      for (const id of completedIds) {
        if (id) this.seen.add(id);
      }
      this.initialized = true;
      return null;
    }
    let newlyObserved: string | null = null;
    for (const id of completedIds) {
      if (!id || this.seen.has(id)) continue;
      this.seen.add(id);
      newlyObserved = id;
    }
    while (this.seen.size > 128) {
      const oldest = this.seen.values().next().value;
      if (oldest === undefined) break;
      this.seen.delete(oldest);
    }
    return newlyObserved;
  }

  reset(): void {
    this.seen.clear();
    this.initialized = false;
  }
}
