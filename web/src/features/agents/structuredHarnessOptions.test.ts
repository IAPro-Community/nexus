import { describe, expect, it } from 'vitest';
import { isStructuredProtocolAllowed, structuredProtocolsForProvider } from './structuredHarnessOptions';

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
