import { describe, expect, it } from 'vitest';
import { bindStructuredHarnessProfile, isStructuredProtocolAllowed, structuredProtocolsForProvider } from './structuredHarnessOptions';

describe('experimental structured harness picker', () => {
  it('preserves terminal mode as the default for every provider', () => {
    expect(isStructuredProtocolAllowed('codex', '', false)).toBe(true);
  });
  it('does not offer a missing/unregistered provider as working ACP', () => {
    expect(structuredProtocolsForProvider('mimo', false)).toEqual([]);
    expect(isStructuredProtocolAllowed('mimo', 'acp', false)).toBe(false);
  });
  it('matches the server launch matrix, without inventing extra transports', () => {
    expect(structuredProtocolsForProvider('omp', true)).toEqual(['acp', 'pi_rpc']);
    expect(structuredProtocolsForProvider('pi', true)).toEqual(['pi_rpc']);
    expect(isStructuredProtocolAllowed('codex', 'acp', true)).toBe(false);
    expect(isStructuredProtocolAllowed('pi', 'acp', true)).toBe(false);
  });
});

describe('structured agent configuration identity', () => {
  it('pins one selected provider and profile without scheduler failover', () => {
    const original = {
      provider: '', profile: '', allocation: { prefer_provider: 'opencode' },
      isolation: 'project',
    };
    const configured = bindStructuredHarnessProfile(original, 'omp', 'connected-profile');
    expect(configured.provider).toBe('omp');
    expect(configured.profile).toBe('connected-profile');
    expect(configured.allocation).toBeUndefined();
    expect(original.provider).toBe('');
  });
  it('refuses an unselected profile', () => {
    expect(() => bindStructuredHarnessProfile(
      { provider: '', profile: '', isolation: 'project' }, 'mimo', '',
    )).toThrow(/explicit registered provider profile/);
  });
});
