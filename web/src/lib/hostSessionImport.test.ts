import { describe, expect, it } from 'vitest';
import { summarizeHostSessionImports } from './hostSessionImport';

describe('host session import summary', () => {
  it('reports outcomes without including provider or credential data', () => {
    const summary = summarizeHostSessionImports([
      { provider: 'codex', profile: 'personal', status: 'IMPORTED' },
      { provider: 'claude', status: 'ALREADY_AUTHENTICATED' },
      { provider: 'gemini', status: 'NO_SESSION' },
      { provider: 'agy', status: 'LOGIN_REQUIRED' },
      { provider: 'cursor', status: 'UNSUPPORTED' },
      { provider: 'opencode', status: 'NOT_INSTALLED' },
      { provider: 'codex', status: 'FAILED' },
    ]);

    expect(summary).toEqual({
      imported: 1,
      alreadyAuthenticated: 1,
      notImported: 2,
      unsupported: 2,
      failed: 1,
    });
    expect(JSON.stringify(summary)).not.toMatch(/codex|personal|credential|token/i);
  });
});
