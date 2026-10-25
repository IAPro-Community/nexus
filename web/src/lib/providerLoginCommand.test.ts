import { describe, expect, it } from 'vitest';
import { providerLoginCommand } from './providerLoginCommand';

describe('providerLoginCommand', () => {
  it('omits the redundant profile argument when the profile uses the provider name', () => {
    expect(providerLoginCommand('codex', 'codex')).toBe('nexus login codex');
  });

  it('keeps explicit profile names in the command', () => {
    expect(providerLoginCommand('codex', 'work')).toBe('nexus login codex work');
  });
});
