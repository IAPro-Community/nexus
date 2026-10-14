import type { ProviderAccount } from "../types";

const hardIneligibilityReasons = new Set([
  "EXHAUSTED",
  "RATE_LIMITED",
  "AUTH_REQUIRED",
  "UNAVAILABLE",
  "NETWORK_ERROR",
  "COOLDOWN",
  "NOT_AUTHENTICATED",
]);

const unavailableHealthStates = new Set([
  "auth_required",
  "unavailable",
  "unhealthy",
]);

/**
 * Mirrors the backend's explicit-selection policy for a manually chosen
 * provider account. Automatic eligibility is intentionally not reused here.
 * Allow optional discovery fields for older UI projections, while the backend
 * remains the final authority at selection time.
 */
export type ManualResourceCandidate = Pick<
  ProviderAccount,
  "provider" | "profile" | "authenticated" | "available"
> &
  Partial<
    Pick<
      ProviderAccount,
      | "health"
      | "rate_limited"
      | "network_error"
      | "avail_reasons"
      | "eligibility"
      | "cooldown_until"
    >
  >;

export function canManuallySelectResourceAccount(
  account: ManualResourceCandidate,
): boolean {
  const reason = account.eligibility?.reason?.toUpperCase();
  const health = (account.health || "").trim().toLowerCase();
  const cooldownUntil = account.cooldown_until
    ? Date.parse(account.cooldown_until)
    : NaN;

  if (
    !account.provider.trim() ||
    !account.profile.trim() ||
    !account.authenticated ||
    (Number.isFinite(cooldownUntil) && cooldownUntil > Date.now()) ||
    account.rate_limited ||
    account.network_error ||
    account.avail_reasons?.rate_limited ||
    account.avail_reasons?.auth_required ||
    unavailableHealthStates.has(health) ||
    (reason && hardIneligibilityReasons.has(reason))
  ) {
    return false;
  }

  if (reason === "UNSUPPORTED_QUOTA") return true;
  if (reason === "UNKNOWN_QUOTA") return account.available;

  return account.available && account.eligibility?.eligible !== false;
}
