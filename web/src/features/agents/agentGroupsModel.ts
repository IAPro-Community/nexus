import type { Agent, AgentGroup } from '../../types';

export function groupForAgent(groups: AgentGroup[], agentId: string): string {
  return groups.find((group) => group.agent_ids.includes(agentId))?.id ?? '';
}

export function moveAgentToGroup(
  groups: AgentGroup[],
  agentId: string,
  targetGroupId: string,
): AgentGroup[] {
  return groups.map((group) => ({
    ...group,
    agent_ids:
      group.id === targetGroupId
        ? [...group.agent_ids.filter((id) => id !== agentId), agentId]
        : group.agent_ids.filter((id) => id !== agentId),
  }));
}

export function reorderGroupAgent(group: AgentGroup, agentId: string, delta: -1 | 1): AgentGroup {
  const index = group.agent_ids.indexOf(agentId);
  const target = index + delta;
  if (index < 0 || target < 0 || target >= group.agent_ids.length) return group;
  const agent_ids = [...group.agent_ids];
  [agent_ids[index], agent_ids[target]] = [agent_ids[target], agent_ids[index]];
  return { ...group, agent_ids };
}

export function filterAgentsByGroup(
  agents: Agent[],
  groups: AgentGroup[],
  groupId: string,
): Agent[] {
  if (groupId === 'all') return agents;
  if (groupId === '') {
    const groupedIds = new Set(groups.flatMap((group) => group.agent_ids));
    return agents.filter((agent) => !groupedIds.has(agent.id));
  }
  const group = groups.find((candidate) => candidate.id === groupId);
  const memberIds = new Set(group?.agent_ids ?? []);
  return agents.filter((agent) => memberIds.has(agent.id));
}
