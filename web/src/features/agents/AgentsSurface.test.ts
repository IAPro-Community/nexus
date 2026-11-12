import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const source = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), 'AgentsSurface.tsx'),
  'utf8',
);

describe('Agents grouping surface contract', () => {
  it('exposes persistent task, worktree, and feature groups in a scoped UI', () => {
    expect(source).toContain('data-feature-tour-scope="agent-groups"');
    expect(source).toContain('data-tour="agent-groups"');
    expect(source).toMatch(/listAgentGroups\(project\.id\)/);
    expect(source).toMatch(/createAgentGroup\(project\.id/);
    expect(source).toContain('groupTask.trim()');
    expect(source).toContain('groupWorktree.trim()');
    expect(source).toContain('groupFeature.trim()');
  });

  it('filters and manually orders group membership while rendering each stable Agent identity', () => {
    expect(source).toContain('filterAgentsByGroup(agents, agentGroups, groupFilter)');
    expect(source).toContain('reorderAgent(assignedGroupEntity, agent.id');
    expect(source).toContain('key={agent.id}');
    expect(source).toContain('data-agent-id={agent.id}');
    expect(source).toContain('onTerminal(agent)');
  });
});
