import { describe, expect, it } from 'vitest';
import { companionAttentionRoute, hudAttentionRoute } from './attentionNavigation';
import type { CompanionAttention } from './companionModel';
import type { HudRuntimeAttention } from './hudModel';

const wait: HudRuntimeAttention = {
  runtime_id: 'rt-agy/one',
  project_id: 'project space',
  provider: 'agy',
  fingerprint: 'f1',
  prompt_kind: 'choice',
  reason: 'QUESTION',
  updated_at: '2026-10-22T00:00:00Z',
};

describe('HUD runtime attention navigation', () => {
  it('opens the exact supervised terminal when no persistent agent is bound', () => {
    expect(hudAttentionRoute(wait)).toBe('/p/project%20space/terminals/rt-agy%2Fone');
  });

  it('opens the project-bound persistent agent terminal for Claude/Codex', () => {
    expect(hudAttentionRoute({ ...wait, agent_id: 'agent/x' })).toBe(
      '/p/project%20space/agents/agent%2Fx',
    );
  });

  it('fails closed to projects when the source cannot identify project ownership', () => {
    expect(hudAttentionRoute({ ...wait, project_id: '' })).toBe('/projects');
  });
});

describe('Companion attention navigation', () => {
  const item: CompanionAttention = {
    id: 'mission:project space:run/1',
    titleKey: 'hud.companion.missionTitle',
    detailKey: 'hud.companion.missionDetail',
    severity: 'critical',
    target: 'missions',
    needsUser: true,
    projectId: 'project space',
    recordId: 'run/1',
  };

  it('opens the exact Mission, Agent, or project Resources route', () => {
    expect(companionAttentionRoute(item)).toBe('/p/project%20space/missions/run%2F1');
    expect(companionAttentionRoute({ ...item, target: 'agents', recordId: 'agent/2' })).toBe(
      '/p/project%20space/agents/agent%2F2',
    );
    expect(companionAttentionRoute({ ...item, target: 'usage' })).toBe(
      '/p/project%20space/resources',
    );
  });

  it('falls back to Projects if verified project context is missing', () => {
    expect(companionAttentionRoute({ ...item, projectId: undefined })).toBe('/projects');
  });
});
