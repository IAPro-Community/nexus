import type { RuntimeSession } from '../types';

const ACTIVE_STATES = new Set<RuntimeSession['state']>([
  'STARTING', 'RUNNING', 'HANDOFF', 'WAITING', 'APPROVAL', 'DETACHED',
]);

export function isVisibleRuntimeAttention(runtime: Pick<RuntimeSession, 'state' | 'attention_reason'>): boolean {
  if (!ACTIVE_STATES.has(runtime.state)) return false;
  return runtime.attention_reason === 'QUESTION'
    || runtime.attention_reason === 'APPROVAL'
    || runtime.attention_reason === 'TASK_COMPLETED'
    || runtime.state === 'WAITING'
    || runtime.state === 'APPROVAL';
}
