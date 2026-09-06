import type { ContextReadinessState } from '../../types';

export type ComposerGateAction = 'NONE' | 'PREPARE' | 'WAIT' | 'REFRESH' | 'RETRY';
export interface ComposerGate {
  canCompose: boolean;
  action: ComposerGateAction;
  reason: string;
}

export function composerNeedsGapConfirmation(brief: {
  open_questions?: string[];
  readiness?: { state?: string };
}): boolean {
  return brief.readiness?.state === 'BLOCKED' || (brief.open_questions?.length ?? 0) > 0;
}

export function composerGateForReadiness(state: ContextReadinessState): ComposerGate {
  switch (state) {
    case 'READY':
      return {
        canCompose: true,
        action: 'NONE',
        reason: 'Durable project context matches the current source fingerprint.',
      };
    case 'MISSING':
      return {
        canCompose: false,
        action: 'PREPARE',
        reason: 'No durable context readiness checkpoint exists yet.',
      };
    case 'HYDRATING':
      return { canCompose: false, action: 'WAIT', reason: 'Context readiness is being evaluated.' };
    case 'STALE':
      return {
        canCompose: false,
        action: 'REFRESH',
        reason: 'Branch, HEAD, dirty state or Maestro version changed.',
      };
    case 'FAILED':
    default:
      return {
        canCompose: false,
        action: 'RETRY',
        reason: 'Context readiness could not be established.',
      };
  }
}
