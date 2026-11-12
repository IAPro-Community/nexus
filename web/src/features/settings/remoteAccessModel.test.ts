import { describe, expect, it } from 'vitest';
import { formatAccessCode } from './remoteAccessModel';

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
