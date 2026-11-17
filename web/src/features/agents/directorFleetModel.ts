import type { Agent, DirectorAgentProjection } from '../../types';

export type FleetStateFilter = 'all' | 'blocked' | 'working' | 'idle' | 'done' | 'offline';

function isOffline(agent: DirectorAgentProjection): boolean {
  return agent.process_state !== 'LIVE';
}

function priority(agent: DirectorAgentProjection): number {
  if (isOffline(agent)) return 6;
  switch (agent.observed_state) {
    case 'blocked':
      return 0;
    case 'error':
      return 1;
    case 'working':
      return 2;
    case 'idle':
      return 3;
    case 'done':
      return 4;
    default:
      return 5;
  }
}

/** Client-side ordering only; never infers completion or mutates the server fleet. */
export function filterDirectorFleet(
  agents: readonly DirectorAgentProjection[],
  knownAgents: ReadonlyMap<string, Pick<Agent, 'name' | 'role'>>,
  filter: FleetStateFilter,
  search: string,
): DirectorAgentProjection[] {
  const needle = search.trim().toLocaleLowerCase();
  return agents
    .map((agent, index) => ({ agent, index }))
    .filter(({ agent }) => {
      if (filter === 'offline') {
        if (!isOffline(agent)) return false;
      } else if (filter !== 'all' && (isOffline(agent) || agent.observed_state !== filter)) {
        return false;
      }
      if (!needle) return true;
      const identity = knownAgents.get(agent.agent_id);
      return [agent.agent_id, identity?.name, identity?.role, agent.provider]
        .filter(Boolean)
        .join(' ')
        .toLocaleLowerCase()
        .includes(needle);
    })
    .sort((left, right) => priority(left.agent) - priority(right.agent) || left.index - right.index)
    .map(({ agent }) => agent);
}
