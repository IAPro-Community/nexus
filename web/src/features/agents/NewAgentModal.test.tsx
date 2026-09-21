import { describe, expect, it } from 'vitest';
import { CANONICAL_AGENT_TYPES, DEFAULT_AGENT_PRESET } from './NewAgentModal';

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
  it('includes the Nexus 1.0 specialist modes', () => {
    const ids = new Set(CANONICAL_AGENT_TYPES.map((preset) => preset.id));
    for (const id of [
      'senior-developer',
      'investigator',
      'architect',
      'security-reviewer',
      'performance-investigator',
    ]) {
      expect(ids.has(id)).toBe(true);
    }
  });

  it('keeps review and investigation specialties conservative by default', () => {
    for (const id of [
      'code-reviewer',
      'investigator',
      'security-reviewer',
      'performance-investigator',
    ]) {
      const preset = CANONICAL_AGENT_TYPES.find((candidate) => candidate.id === id);
      expect(preset?.defaultMode).toBe('Safe');
      expect(
        preset?.agentSpec.constraints?.some((constraint) =>
          constraint.toLowerCase().includes('read-only'),
        ),
      ).toBe(true);
    }
  });
  it('preserves the established backend default independent of preset ordering', () => {
    expect(DEFAULT_AGENT_PRESET.id).toBe('backend-engineer');
  });
});
