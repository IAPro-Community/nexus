import { describe, expect, it } from 'vitest';
import type { DirectorAgentProjection } from '../../types';
import { filterDirectorFleet } from './directorFleetModel';

function member(
  id: string,
  observed: DirectorAgentProjection['observed_state'],
  process: DirectorAgentProjection['process_state'] = 'LIVE',
): DirectorAgentProjection {
  return {
    schema_version: 1,
    execution_owner: 'NEXUS',
    agent_id: id,
    project_id: 'project-1',
    agent_status: 'WORKING',
    process_state: process,
    process_evidence: 'SESSION_HOST_LIVE',
    observed_state: observed,
    actions: [],
  };
}

describe('Director fleet filtering and ordering', () => {
  const agents = new Map([
    ['reviewer', { name: 'Reviewer', role: 'review' }],
    ['builder', { name: 'Builder', role: 'developer' }],
  ]);
  const fleet = [
    member('offline', 'blocked', 'NOT_LIVE'),
    member('builder', 'working'),
    member('reviewer', 'blocked'),
    member('idle', 'idle'),
    member('pending', 'unknown'),
  ];

  it('prioritizes real live blocks without promoting stale sessions', () => {
    expect(filterDirectorFleet(fleet, agents, 'all', '').map((agent) => agent.agent_id)).toEqual([
      'reviewer',
      'builder',
      'idle',
      'pending',
      'offline',
    ]);
    expect(
      filterDirectorFleet(fleet, agents, 'blocked', '').map((agent) => agent.agent_id),
    ).toEqual(['reviewer']);
    expect(
      filterDirectorFleet(fleet, agents, 'offline', '').map((agent) => agent.agent_id),
    ).toEqual(['offline']);
  });

  it('matches known display names and roles without modifying the original ordering', () => {
    expect(filterDirectorFleet(fleet, agents, 'all', 'review').map((a) => a.agent_id)).toEqual([
      'reviewer',
    ]);
    expect(filterDirectorFleet(fleet, agents, 'all', 'developer').map((a) => a.agent_id)).toEqual([
      'builder',
    ]);
    expect(fleet.map((agent) => agent.agent_id)[0]).toBe('offline');
  });
});
