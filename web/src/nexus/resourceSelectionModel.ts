import type { ProviderAccount } from '../types';

const hardIneligibilityReasons = new Set([
  'EXHAUSTED',
  'RATE_LIMITED',
  'AUTH_REQUIRED',
  'UNAVAILABLE',
  'NETWORK_ERROR',
  'COOLDOWN',
  'NOT_AUTHENTICATED',
]);

const unavailableHealthStates = new Set(['auth_required', 'unavailable', 'unhealthy']);

/**
 * Mirrors the backend's explicit-selection policy for a manually chosen
 * provider account. Automatic eligibility is intentionally not reused here.
 */
export function canManuallySelectResourceAccount(account: ProviderAccount): boolean {
  const reason = account.eligibility?.reason?.toUpperCase();
  const health = account.health.trim().toLowerCase();

  if (
    !account.authenticated ||
    account.rate_limited ||
    account.network_error ||
    account.avail_reasons?.rate_limited ||
    account.avail_reasons?.auth_required ||
    unavailableHealthStates.has(health) ||
    (reason && hardIneligibilityReasons.has(reason))
  ) {
    return false;
  }

  if (reason === 'UNSUPPORTED_QUOTA') return true;
  if (reason === 'UNKNOWN_QUOTA') return account.available;

  return account.available && account.eligibility?.eligible !== false;
}
