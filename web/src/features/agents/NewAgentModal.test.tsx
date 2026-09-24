import { describe, expect, it } from 'vitest';
import { AGENT_MODE_PRESENTATION, NATIVE_PROVIDERS } from './NewAgentModal';

describe('Agent Mode presentation', () => {
  it('covers every required Nexus 1.0 specialist with user-facing copy', () => {
    for (const id of [
      'developer',
      'senior-developer',
      'reviewer',
      'investigator',
      'architect',
      'security-reviewer',
      'qa-engineer',
      'performance-investigator',
      'documentation-specialist',
    ]) {
      const presentation = AGENT_MODE_PRESENTATION[id];
      expect(presentation?.name).toBeTruthy();
      expect(presentation?.description).toBeTruthy();
      expect(presentation?.icon).toBeTruthy();
    }
  });

  it('contains presentation only, not duplicated AgentSpec policy', () => {
    for (const value of Object.values(AGENT_MODE_PRESENTATION)) {
      expect(value).not.toHaveProperty('agentSpec');
      expect(value).not.toHaveProperty('agent_spec');
      expect(value).not.toHaveProperty('capabilities');
      expect(value).not.toHaveProperty('constraints');
    }
  });

  it('keeps AGY and OpenCode as distinct native provider identities', () => {
    const ids = NATIVE_PROVIDERS.map((provider) => provider.id);
    expect(ids).toContain('agy');
    expect(ids).toContain('opencode');
    expect(new Set(ids).size).toBe(ids.length);

    const agy = NATIVE_PROVIDERS.find((provider) => provider.id === 'agy');
    const opencode = NATIVE_PROVIDERS.find((provider) => provider.id === 'opencode');
    expect(agy?.alias).toBe('nexus agy');
    expect(agy?.label).not.toContain('OpenCode');
    expect(opencode?.alias).toBe('nexus opencode');
    expect(opencode?.label).toContain('OpenCode');
  });
});
