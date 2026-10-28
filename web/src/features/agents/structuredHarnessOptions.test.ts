import { describe, expect, it } from 'vitest';
import {
  bindStructuredHarnessProfile,
  isStructuredProtocolAllowed,
  structuredProtocolsForProvider,
} from './structuredHarnessOptions';

describe('experimental structured harness picker', () => {
  it('keeps terminal as the default for older or unavailable Core versions', () => {
    expect(isStructuredProtocolAllowed(undefined, '')).toBe(true);
    expect(structuredProtocolsForProvider(undefined)).toEqual([]);
    expect(structuredProtocolsForProvider({ installed: true })).toEqual([]);
  });

  it('does not offer protocols when the provider is unavailable', () => {
    const unavailable = { installed: false, structured_modes: ['acp'] };
    expect(structuredProtocolsForProvider(unavailable)).toEqual([]);
    expect(isStructuredProtocolAllowed(unavailable, 'acp')).toBe(false);
  });

  it('only trusts explicit Core entrypoint declarations, not a local provider matrix', () => {
    const pi = { installed: true, structured_modes: ['pi_rpc'] };
    const omp = { installed: true, structured_modes: ['acp', 'pi_rpc'] };
    expect(structuredProtocolsForProvider(omp)).toEqual(['acp', 'pi_rpc']);
    expect(structuredProtocolsForProvider(pi)).toEqual(['pi_rpc']);
    expect(isStructuredProtocolAllowed(pi, 'acp')).toBe(false);
    expect(isStructuredProtocolAllowed(omp, 'acp')).toBe(true);
  });

  it('rejects unknown or malformed backend protocol names and de-duplicates modes', () => {
    expect(
      structuredProtocolsForProvider({
        installed: true,
        structured_modes: ['experimental', 'acp', 'acp', 'pi_rpc', 'terminal'],
      }),
    ).toEqual(['acp', 'pi_rpc']);
    expect(
      structuredProtocolsForProvider({
        installed: true,
        structured_modes: ['unknown'],
      }),
    ).toEqual([]);
  });
});

describe('structured agent configuration identity', () => {
  it('pins one selected provider and profile without scheduler failover', () => {
    const original = {
      provider: '',
      profile: '',
      allocation: { prefer_provider: 'opencode' },
      isolation: 'project',
    };
    const configured = bindStructuredHarnessProfile(original, 'omp', 'connected-profile');
    expect(configured.provider).toBe('omp');
    expect(configured.profile).toBe('connected-profile');
    expect(configured.allocation).toBeUndefined();
    expect(original.provider).toBe('');
  });
  it('refuses an unselected profile', () => {
    expect(() =>
      bindStructuredHarnessProfile({ provider: '', profile: '', isolation: 'project' }, 'mimo', ''),
    ).toThrow(/explicit registered provider profile/);
  });
});
