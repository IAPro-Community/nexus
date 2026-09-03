import { describe, expect, it } from 'vitest';
import { globalSearch } from './globalSearch';

describe('globalSearch', () => {
  const projects = [
    { id: 'proj-1', name: 'Nexus' },
    { id: 'proj-2', name: 'Omega' },
  ];

  const agents = [
    { id: 'agt-1', title: 'Backend Developer', projectId: 'proj-1' },
    { id: 'agt-2', title: 'Frontend Developer', projectId: 'proj-1' },
  ];

  const providerSessions = [
    { id: 'sess-1', title: 'Auth refactor', projectId: 'proj-1', provider: 'claude' },
  ];

  const flowRuns = [
    { id: 'run-1', title: 'Build pipeline', projectId: 'proj-1' },
  ];

  it('returns empty for empty query', () => {
    expect(globalSearch('', projects, agents, providerSessions, flowRuns)).toEqual([]);
  });

  it('finds projects by exact name', () => {
    const results = globalSearch('Nexus', projects, agents, providerSessions, flowRuns);
    expect(results.length).toBeGreaterThan(0);
    expect(results[0].title).toBe('Nexus');
    expect(results[0].type).toBe('project');
  });

  it('finds agents by fuzzy match', () => {
    const results = globalSearch('bkend', projects, agents, providerSessions, flowRuns);
    expect(results.length).toBeGreaterThan(0);
    expect(results.some((r) => r.type === 'agent' && r.title === 'Backend Developer')).toBe(true);
  });

  it('finds provider sessions', () => {
    const results = globalSearch('Auth', projects, agents, providerSessions, flowRuns);
    expect(results.length).toBeGreaterThan(0);
    expect(results.some((r) => r.type === 'provider-session')).toBe(true);
  });

  it('finds flow runs', () => {
    const results = globalSearch('Build', projects, agents, providerSessions, flowRuns);
    expect(results.length).toBeGreaterThan(0);
    expect(results.some((r) => r.type === 'flow-run')).toBe(true);
  });

  it('sorts by relevance', () => {
    const results = globalSearch('Nexus', projects, agents, providerSessions, flowRuns);
    // Exact match should be first
    expect(results[0].title).toBe('Nexus');
  });
});
