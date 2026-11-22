import { describe, expect, it } from 'vitest';
import { formatAccessCode, remoteAccessMode, tunnelOriginKind } from './remoteAccessModel';

describe('formatAccessCode', () => {
  it('groups an 8-character code as ABCD-EFGH', () => {
    expect(formatAccessCode('A1B0CDEF')).toBe('A1B0-CDEF');
  });

  it('trims and uppercases typed input', () => {
    expect(formatAccessCode('  a1b0cdef ')).toBe('A1B0-CDEF');
  });

  it('returns unexpected lengths unchanged', () => {
    expect(formatAccessCode('ABC')).toBe('ABC');
    expect(formatAccessCode('')).toBe('');
  });
});

describe('tunnelOriginKind', () => {
  it('reports a named tunnel origin as stable', () => {
    expect(tunnelOriginKind(true)).toBe('stable');
  });

  it('reports a quick tunnel origin as ephemeral', () => {
    expect(tunnelOriginKind(false)).toBe('ephemeral');
  });

  it('treats an absent flag as ephemeral', () => {
    // The server that predates the stable-url contract must not make the
    // operator believe an installed app survives a restart.
    expect(tunnelOriginKind(undefined)).toBe('ephemeral');
  });
});

describe('remoteAccessMode', () => {
  it('reports the private mode when a remote origin is published', () => {
    expect(remoteAccessMode(false, 'https://nexus.tail1234.ts.net')).toBe('private');
  });

  it('reports the tunnel mode when a public tunnel is active', () => {
    expect(remoteAccessMode(true, '')).toBe('tunnel');
    expect(remoteAccessMode(true, undefined)).toBe('tunnel');
  });

  it('reports no mode when nothing is published', () => {
    expect(remoteAccessMode(false, '')).toBe('none');
    expect(remoteAccessMode(false, undefined)).toBe('none');
  });

  it('prefers the active tunnel over the private origin when both exist', () => {
    // The server binds the invitation to the live tunnel URL while a tunnel
    // runs, so the address shown must be the one the invitation uses.
    expect(remoteAccessMode(true, 'https://nexus.tail1234.ts.net')).toBe('tunnel');
  });
});
