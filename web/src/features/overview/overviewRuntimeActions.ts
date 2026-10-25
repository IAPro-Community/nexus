import type { Agent, RuntimeSession } from '../../types';

export interface OverviewRuntimeActions {
  onOpenAgent: (agent: Agent, runtimeId?: string) => void;
  onAttachRuntime: (runtime: RuntimeSession) => void;
}

export function attachOverviewRuntime(
  runtime: RuntimeSession,
  matchedAgent: Agent | undefined,
  actions: OverviewRuntimeActions,
): void {
  if (matchedAgent) {
    actions.onOpenAgent(matchedAgent, runtime.runtime_id);
    return;
  }

  actions.onAttachRuntime(runtime);
}
