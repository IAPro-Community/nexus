import type { Agent, RuntimeSession } from '../types';
import { isLiveRuntimeState } from '../nexus/agentTerminalModel';

export function agentForRuntime(runtime: RuntimeSession, agents: Agent[]): Agent | undefined {
  if (!runtime.agent_id) return undefined;
  return agents.find((agent) => agent.id === runtime.agent_id);
}

export function terminalSurfaceIDForRuntime(runtime: RuntimeSession): string | undefined {
  return runtime.agent_id ? `agent:${runtime.agent_id}:terminal` : undefined;
}

/** Returns the newest live runtime generation for an Agent.
 *
 * Registry/API order is intentionally not trusted: the registry is map-backed
 * and handoff/recovery can briefly expose sibling generations simultaneously.
 */
export function currentRuntimeForAgent(
  runtimes: RuntimeSession[],
  agentId?: string,
): RuntimeSession | undefined {
  if (!agentId) return undefined;
  return runtimes
    .filter((runtime) => runtime.agent_id === agentId && isLiveRuntimeState(runtime.state))
    .sort((left, right) => {
      const byStarted = Date.parse(right.started_at || '') - Date.parse(left.started_at || '');
      if (Number.isFinite(byStarted) && byStarted !== 0) return byStarted;
      const byProviderGeneration =
        (right.provider_generation || 0) - (left.provider_generation || 0);
      if (byProviderGeneration !== 0) return byProviderGeneration;
      return (right.host_generation || 0) - (left.host_generation || 0);
    })[0];
}
