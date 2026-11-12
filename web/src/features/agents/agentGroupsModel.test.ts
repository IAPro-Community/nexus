import { describe, expect, it } from 'vitest';
import type { Agent, AgentGroup } from '../../types';
import {
  filterAgentsByGroup,
  groupForAgent,
  moveAgentToGroup,
  reorderGroupAgent,
} from './agentGroupsModel';

const agents = [
  { id: 'agt_a', name: 'A' },
  { id: 'agt_b', name: 'B' },
  { id: 'agt_c', name: 'C' },
] as Agent[];
const groups = [
  { id: 'grp_one', agent_ids: ['agt_b', 'agt_a'] },
  { id: 'grp_two', agent_ids: ['agt_c'] },
] as AgentGroup[];

describe('Agent group identity, filtering, and order', () => {
  it('filters members while retaining the original Agent records and stable IDs', () => {
    const filtered = filterAgentsByGroup(agents, groups, 'grp_one');
    expect(filtered.map((agent) => agent.id)).toEqual(['agt_a', 'agt_b']);
    expect(filtered[0]).toBe(agents[0]);
    expect(groupForAgent(groups, 'agt_b')).toBe('grp_one');
    expect(groupForAgent(groups, 'agt_c')).toBe('grp_two');
  });

  it('keeps filtering independent of membership and reports ungrouped Agents', () => {
    expect(filterAgentsByGroup(agents, groups, 'all')).toEqual(agents);
    expect(filterAgentsByGroup(agents, groups, '')).toEqual([]);
    expect(filterAgentsByGroup(agents, [groups[0]], '')).toEqual([agents[2]]);
  });

  it('moves and reorders membership by stable Agent ID without changing Agent records', () => {
    const moved = moveAgentToGroup(groups, 'agt_a', 'grp_two');
    expect(moved[0].agent_ids).toEqual(['agt_b']);
    expect(moved[1].agent_ids).toEqual(['agt_c', 'agt_a']);
    expect(agents[0].id).toBe('agt_a');
    const reordered = reorderGroupAgent(moved[0], 'agt_b', 1);
    expect(reordered.agent_ids).toEqual(['agt_b']);
    expect(
      reorderGroupAgent({ ...moved[1], agent_ids: ['agt_a', 'agt_c'] }, 'agt_a', 1).agent_ids,
    ).toEqual(['agt_c', 'agt_a']);
  });
});
