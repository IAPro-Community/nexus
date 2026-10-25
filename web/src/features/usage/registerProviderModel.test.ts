import { describe, expect, it } from 'vitest';
import {
  preferredRegistrationProvider,
  requiresExplicitAdditionalProfileName,
} from './registerProviderModel';

describe('register provider safety', () => {
  const unavailable = { id: 'claude', installed: false, registration_state: '' };
  const pending = { id: 'codex', installed: true, registration_state: 'PENDING_AUTH' };
  const unregistered = {
    id: 'opencode',
    installed: true,
    registration_state: 'INSTALLED_UNREGISTERED',
  };
  const unverified = { id: 'cursor', installed: true, registration_state: 'REGISTRATION_UNKNOWN' };

  it('never selects an unavailable binary', () => {
    expect(preferredRegistrationProvider([unavailable], 'claude')).toBe('');
    expect(preferredRegistrationProvider([unavailable, unregistered], 'claude')).toBe('opencode');
  });

  it('prefers a CLI without an existing profile', () => {
    expect(preferredRegistrationProvider([pending, unregistered])).toBe('opencode');
    expect(preferredRegistrationProvider([pending, unregistered], 'codex')).toBe('codex');
  });

  it('does not treat an unverified existing profile as unregistered', () => {
    expect(preferredRegistrationProvider([unverified, unregistered])).toBe('opencode');
    expect(requiresExplicitAdditionalProfileName(unverified, '')).toBe(true);
  });

  it('prevents accidental duplicate but permits explicit multi-account', () => {
    expect(requiresExplicitAdditionalProfileName(pending, '')).toBe(true);
    expect(requiresExplicitAdditionalProfileName(pending, 'work')).toBe(false);
    expect(requiresExplicitAdditionalProfileName(unregistered, '')).toBe(false);
  });
});
