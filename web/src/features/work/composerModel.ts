export type ContextReadiness = 'MISSING' | 'HYDRATING' | 'READY' | 'STALE' | 'FAILED';

export function canPlan(readiness: ContextReadiness): boolean {
  return readiness === 'READY';
}

export function composerActions(readiness: ContextReadiness) {
  return {
    sendToAgent: true,
    copyPrompt: true,
    turnIntoFlow: canPlan(readiness),
  };
}
