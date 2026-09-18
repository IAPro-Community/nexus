import { describe, expect, it } from 'vitest';
import { CANONICAL_AGENT_TYPES } from './NewAgentModal';

describe('canonical persistent agent presets', () => {
  it('defines behavioral specialization for every specialty', () => {
    expect(CANONICAL_AGENT_TYPES).not.toHaveLength(0);

    for (const preset of CANONICAL_AGENT_TYPES) {
      expect(preset.agentSpec.role).toBe(preset.role);
      expect(preset.agentSpec.instructions?.length).toBeGreaterThan(0);
      expect(preset.agentSpec.responsibilities?.length).toBeGreaterThan(0);
      expect(preset.agentSpec.capabilities?.length).toBeGreaterThan(0);
    }
  });
});
