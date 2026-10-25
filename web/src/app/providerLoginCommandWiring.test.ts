import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const workspaceSurfaceHost = readFileSync(
  new URL('./WorkspaceSurfaceHost.tsx', import.meta.url),
  'utf8',
);
const accountLoginModal = readFileSync(
  new URL('../features/usage/AccountLoginModal.tsx', import.meta.url),
  'utf8',
);

describe('provider login command wiring', () => {
  it('uses the shared formatter in the resource panel terminal flow', () => {
    expect(workspaceSurfaceHost).toContain('providerLoginCommand(provider, profile)');
  });

  it('uses the shared formatter in the account login dialog', () => {
    expect(accountLoginModal).toContain('providerLoginCommand(provider, profile)');
  });
});
