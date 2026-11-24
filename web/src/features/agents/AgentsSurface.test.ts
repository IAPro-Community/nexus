import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const source = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), 'AgentsSurface.tsx'),
  'utf8',
);
const directorSource = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), 'DirectorAgentPanel.tsx'),
  'utf8',
);
const dataSource = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), '../../app/useNexusData.ts'),
  'utf8',
);

describe('Agent command center contract', () => {
  it('lists every Agent of the selected project and renders observed status', () => {
    expect(dataSource).toContain('nexus.listAgents(projectId)');
    expect(source).toContain('data-agent-id={agent.id}');
    expect(source).toContain(
      '<Badge tone={tone(agent.status)}>{translateStatus(agent.status)}</Badge>',
    );
  });

  it('derives the card status only from the structured Agent field', () => {
    expect(source).not.toMatch(/output|stdout|last_message|attention_reason/i);
  });

  it('selects the exact Agent directly for terminal and Director', () => {
    expect(source).toContain('onClick={() => onTerminal(agent)}');
    expect(source).toContain('onClick={() => setDirectorAgent(agent)}');
  });
});

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

describe('Agent Inspector contract', () => {
  it('projects observed status, identity, evidence, quota and lineage', () => {
    expect(directorSource).toContain("t('director.quotaStatus')");
    expect(directorSource).toContain("t('director.quotaSource')");
    expect(directorSource).toContain("t('director.quotaRemaining')");
    expect(directorSource).toContain("t('director.lineageTitle')");
    expect(directorSource).toContain('view.workspace');
    expect(directorSource).toContain('view.worktree');
    expect(directorSource).toContain("t('director.observedState')");
  });

  it('never projects provider session identifiers in the Inspector', () => {
    expect(directorSource).not.toContain('source_session');
    expect(directorSource).not.toContain('target_session');
    expect(directorSource).not.toContain('provider_session');
  });
});
