import type { AttentionGroup } from '../../types';
import type { HudState } from './hudModel';
import {
  type MascotAttention,
  type MascotTarget,
  mascotMood,
  mascotMotionAllowed,
  nextMascotBubble,
  projectMascotAttention,
  readMascotSnoozes,
  snoozeMascotAttention,
} from './mascotModel';

export type CompanionMood = 'idle' | 'working' | 'attention' | 'critical' | 'offline';
export type CompanionTarget = MascotTarget;
export type CompanionAttention = MascotAttention;

export const projectCompanionAttention = (
  state: HudState | undefined,
  attentionGroup?: AttentionGroup,
  now = Date.now(),
): CompanionAttention[] => projectMascotAttention(state, attentionGroup, now);

export const companionMood = (
  state: HudState | undefined,
  attention: CompanionAttention[],
  offline: boolean,
): CompanionMood => mascotMood(state, attention, offline);

export const companionMotionAllowed = (mood: CompanionMood, requested: boolean): boolean =>
  mascotMotionAllowed(mood, requested);

export const readCompanionSnoozes = (
  storage: Pick<Storage, 'getItem'>,
  now = Date.now(),
): Record<string, number> => readMascotSnoozes(storage, now);

export const snoozeCompanionAttention = (
  storage: Pick<Storage, 'getItem' | 'setItem'>,
  id: string,
  now = Date.now(),
  durationMs = 60 * 60 * 1000,
): void => snoozeMascotAttention(storage, id, now, durationMs);

export const nextCompanionBubble = (
  items: CompanionAttention[],
  snoozed: Record<string, number>,
  now = Date.now(),
  dismissed: Record<string, boolean> = {},
): CompanionAttention | undefined => nextMascotBubble(items, snoozed, now, dismissed);

export * from './mascotModel';
