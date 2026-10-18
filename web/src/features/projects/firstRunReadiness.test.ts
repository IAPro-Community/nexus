import { describe, expect, it } from 'vitest';
import { deriveFirstRunReadiness } from './firstRunReadiness';

describe('first run detection', () => {
  it('does not fabricate a ready account while checks are running or unavailable', () => {
    expect(deriveFirstRunReadiness([], 'checking').status).toBe('checking');
    expect(deriveFirstRunReadiness([], 'unavailable').status).toBe('unavailable');
  });

  it('treats an installed CLI as needing authentication, not ready', () => {
    const status = deriveFirstRunReadiness(
      [{ id: 'codex', installed: true, registration_state: 'PENDING_AUTH' }],
      'loaded',
    );
    expect(status.status).toBe('needs-login');
    expect(status.suggestedProvider).toBe('codex');
  });

  it('selects only installed providers and does not mistake registration for login', () => {
    const status = deriveFirstRunReadiness(
      [
        { id: 'claude', installed: false, registration_state: 'REGISTERED_AUTHENTICATED' },
        { id: 'codex', installed: true, registration_state: 'INSTALLED_UNREGISTERED' },
      ],
      'loaded',
    );
    expect(status.status).toBe('needs-account');
    expect(status.suggestedProvider).toBe('codex');
  });

  it('does not offer duplicate registration for an existing pending account', () => {
    const state = deriveFirstRunReadiness(
      [
        { id: 'codex', installed: true, registration_state: 'PENDING_AUTH' },
        { id: 'claude', installed: true, registration_state: 'INSTALLED_UNREGISTERED' },
      ],
      'loaded',
    );
    expect(state.status).toBe('needs-login');
  });

  it('reports readiness only when backend confirms authentication', () => {
    const status = deriveFirstRunReadiness(
      [
        { id: 'codex', installed: true, registration_state: 'REGISTERED_AUTHENTICATED' },
        { id: 'claude', installed: true, registration_state: 'PENDING_AUTH' },
      ],
      'loaded',
    );
    expect(status.status).toBe('ready');
  });

  it('does not block project creation when no CLI is installed', () => {
    const status = deriveFirstRunReadiness([], 'loaded');
    expect(status.status).toBe('needs-provider');
    expect(status.suggestedProvider).toBeUndefined();
  });
});
