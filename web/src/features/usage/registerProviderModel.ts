import type { ProviderInfo } from '../../types';

type RegistrationCandidate = Pick<ProviderInfo, 'id' | 'installed' | 'registration_state'>;

const hasProfile = (provider: RegistrationCandidate) =>
  provider.registration_state === 'REGISTRATION_UNKNOWN' ||
  provider.registration_state === 'PENDING_AUTH' ||
  provider.registration_state === 'REGISTERED_AUTHENTICATED' ||
  provider.registration_state === 'REGISTERED_AUTH_EXPIRED';

export const preferredRegistrationProvider = (
  providers: RegistrationCandidate[],
  initialProvider?: string,
): string => {
  const installed = providers.filter((provider) => provider.installed);
  const requested = installed.find((provider) => provider.id === initialProvider);
  if (requested) return requested.id;
  // Prefer an unregistered CLI. Never select a missing CLI by default.
  return (installed.find((provider) => !hasProfile(provider)) || installed[0])?.id || '';
};

// A pre-existing profile is not a reason to block additional accounts. Require
// the user to intentionally name the new profile instead of silently retrying
// the default provider name, which would produce a duplicate 409.
export const requiresExplicitAdditionalProfileName = (
  provider: RegistrationCandidate | undefined,
  profileName: string,
): boolean => Boolean(provider && hasProfile(provider) && !profileName.trim());
