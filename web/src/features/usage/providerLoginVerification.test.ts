import { describe, expect, it } from 'vitest';
import { isProviderLoginVerified } from './providerLoginVerification';

describe('provider login verification', () => {
  const accounts = [
    { provider: 'codex', profile: 'personal', authenticated: true },
    { provider: 'codex', profile: 'work', authenticated: false },
    { provider: 'claude', profile: 'default', authenticated: true },
  ];

  it('confirms only a matching authenticated profile', () => {
    expect(isProviderLoginVerified(accounts, 'codex', 'personal')).toBe(true);
    expect(isProviderLoginVerified(accounts, 'codex', 'work')).toBe(false);
    expect(isProviderLoginVerified(accounts, 'codex', 'default')).toBe(false);
    expect(isProviderLoginVerified(accounts, 'claude', 'default')).toBe(true);
  });

  it('cannot claim successful login when no resource evidence exists', () => {
    expect(isProviderLoginVerified([], 'codex', 'personal')).toBe(false);
    expect(isProviderLoginVerified(accounts, 'opencode', 'personal')).toBe(false);
  });
});
