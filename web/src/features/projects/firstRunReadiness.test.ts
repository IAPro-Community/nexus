import { describe, expect, it } from 'vitest';
import { deriveFirstRunReadiness, summarizeHostSessionImport } from './firstRunReadiness';

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

  it('prompts login for an expired profile instead of creating a duplicate', () => {
    const status = deriveFirstRunReadiness(
      [{ id: 'codex', installed: true, registration_state: 'REGISTERED_AUTH_EXPIRED' }],
      'loaded',
    );
    expect(status.status).toBe('needs-login');
    expect(status.suggestedProvider).toBe('codex');
  });

  it('does not ask for a new account when authentication could not be verified', () => {
    const status = deriveFirstRunReadiness(
      [{ id: 'cursor', installed: true, registration_state: 'REGISTRATION_UNKNOWN' }],
      'loaded',
    );
    expect(status.status).toBe('verification-unknown');
    expect(status.suggestedProvider).toBe('cursor');
    expect(status.canImportSessions).toBe(true);
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

  it('offers host-session import while any installed provider lacks an authenticated profile', () => {
    const status = deriveFirstRunReadiness(
      [
        { id: 'codex', installed: true, registration_state: 'INSTALLED_UNREGISTERED' },
        { id: 'claude', installed: true, registration_state: 'REGISTERED_AUTHENTICATED' },
      ],
      'loaded',
    );
    expect(status.canImportSessions).toBe(true);
  });

  it('does not offer import after all installed providers are authenticated', () => {
    const status = deriveFirstRunReadiness(
      [{ id: 'codex', installed: true, registration_state: 'REGISTERED_AUTHENTICATED' }],
      'loaded',
    );
    expect(status.canImportSessions).toBe(false);
  });

  it('summarizes imports without exposing account or credential data', () => {
    expect(
      summarizeHostSessionImport([
        { status: 'IMPORTED' },
        { status: 'ALREADY_AUTHENTICATED' },
        { status: 'NO_SESSION' },
        { status: 'LOGIN_REQUIRED' },
        { status: 'UNSUPPORTED' },
        { status: 'FAILED' },
      ]),
    ).toEqual({
      imported: 1,
      alreadyAuthenticated: 1,
      notImported: 2,
      unsupported: 1,
      failed: 1,
    });
  });

  it('does not block project creation when no CLI is installed', () => {
    const status = deriveFirstRunReadiness([], 'loaded');
    expect(status.status).toBe('needs-provider');
    expect(status.suggestedProvider).toBeUndefined();
  });
});
