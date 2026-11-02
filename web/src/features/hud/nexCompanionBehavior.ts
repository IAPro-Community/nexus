import type { MascotMood } from './mascotModel';
import type { NexIdleAction } from './nexIdleDirector';

/**
 * Lightweight Clippy-like presentation projection.
 * This never infers a Core event, controls a mission, or starts an animation loop.
 * The existing mascotMood and NexIdleDirector remain the only authorities.
 */
export type NexCompanionBehavior =
  | 'still'
  | 'ambient'
  | 'greet'
  | 'look'
  | 'thinking'
  | 'attention'
  | 'error'
  | 'resolved'
  | 'rest';

export type NexCompanionInteraction = 'none' | 'greet' | 'look';

export interface NexCompanionInput {
  mood: MascotMood;
  idleAction: NexIdleAction | null;
  interaction: NexCompanionInteraction;
  /** True only for a newly observed, deduplicated authoritative completion. */
  resolved: boolean;
  motionAllowed: boolean;
}

export function resolveNexCompanionBehavior(input: NexCompanionInput): NexCompanionBehavior {
  if (!input.motionAllowed || input.mood === 'offline') return 'still';
  if (input.mood === 'critical') return 'error';
  if (input.mood === 'attention') return 'attention';
  if (input.mood === 'working') return 'thinking';
  if (input.resolved) return 'resolved';
  if (input.idleAction === 'idle_drowse' || input.idleAction === 'idle_wake') return 'rest';
  if (input.interaction === 'look') return 'look';
  if (input.interaction === 'greet') return 'greet';
  return 'ambient';
}

/** Stable key based on the canonical Attention Center completed bucket. */
export interface NexCompletedRecord {
  project_id: string;
  mission_id: string;
  state: string;
  age_seconds: number;
}

export function nexCompletedEventId(record: NexCompletedRecord): string | null {
  const state = typeof record.state === 'string' ? record.state.toUpperCase() : '';
  if (!record.project_id ||
    !record.mission_id ||
    !/^(COMPLETED_VERIFIED|COMPLETED|SUCCEEDED|SUCCESS)$/.test(state)) {
    return null;
  }
  // An old completed record must not trigger a new celebration.
  if (!Number.isFinite(record.age_seconds) || record.age_seconds < 0 || record.age_seconds > 90) {
    return null;
  }
  return `${record.project_id}:${record.mission_id}:${state}`;
}

/**
 * Baseline on first observation; afterwards emit only genuinely new IDs.
 * caller owns the seen Set (per project) and does not replay stale records.
 */
export function consumeNexNewCompletion(
  records: readonly NexCompletedRecord[],
  seen: Set<string>,
  initialized: boolean,
): string | null {
  let newest: string | null = null;
  for (const record of records) {
    const id = nexCompletedEventId(record);
    if (!id || seen.has(id)) continue;
    seen.add(id);
    if (initialized) newest = id;
  }
  return newest;
}
