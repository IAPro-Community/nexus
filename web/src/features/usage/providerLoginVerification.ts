import type { ProviderAccount } from '../../types';

// A quota refresh is not a successful login; only the backend's exact account
// identity and authenticated flag can complete this onboarding step.
export function isProviderLoginVerified(
  accounts: Array<Pick<ProviderAccount, 'provider' | 'profile' | 'authenticated'>>,
  provider: string,
  profile: string,
): boolean {
  return accounts.some(
    (account) =>
      account.provider === provider &&
      account.profile === profile &&
      account.authenticated === true,
  );
}
