import { describe, expect, it } from 'vitest';
import { AGENT_MODE_PRESENTATION } from './NewAgentModal';

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
});
