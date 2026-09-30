import type { ProviderAccount } from '../../types';
import { asArray } from '../../lib/safeArray';
import {
  bestGroupRemainingFromQuotaView,
  compactQuotaLabel,
  isMeasuredQuotaStatus,
} from '../work/directSessionModel';

function cap(account: ProviderAccount, key: string): string {
  return String(account.capabilities?.[key] || '').toUpperCase();
}

export function accountKey(account: Pick<ProviderAccount, 'provider' | 'profile'>): string {
  return `${account.provider}:${account.profile}`;
}

export function isIntelligenceCLICapable(account: ProviderAccount): boolean {
  return cap(account, 'headless') === 'SUPPORTED' && cap(account, 'submit_prompt') === 'SUPPORTED';
}

function isTerminalOnly(account: ProviderAccount): boolean {
  const provider = String(account.provider || '').toLowerCase();
  return provider === 'shell' || provider === 'fake';
}

export function quotaPercent(account: ProviderAccount): number | null {
  if (account.avail_reasons?.unknown_quota || !isMeasuredQuotaStatus(account.quota_view?.status))
    return null;
  const fromWindows = bestGroupRemainingFromQuotaView(account.quota_view);
  if (fromWindows != null) return fromWindows;
  const remaining = account.quota_remaining;
  if (!Number.isFinite(remaining)) return null;
  return Math.round(remaining <= 1 ? remaining * 100 : remaining);
}

export function quotaLabel(account: ProviderAccount): string | null {
  if (account.avail_reasons?.unknown_quota || !isMeasuredQuotaStatus(account.quota_view?.status))
    return null;
  const label = compactQuotaLabel(account.quota_view);
  if (label) return label;
  const pct = quotaPercent(account);
  return pct == null ? null : `${pct}%`;
}

export function intelligenceModelChoices(
  account?: ProviderAccount | null,
): Array<{ id: string; label: string; remaining: number | null }> {
  const groups = asArray<{ name?: string; windows?: Array<{ kind?: string; remaining?: number }> }>(
    account?.quota_view?.model_groups,
  );
  return groups
    .map((group) => {
      const id = String(group.name || '').trim();
      const windows = asArray<{ kind?: string; remaining?: number }>(group.windows).filter(
        (w) => w.kind !== 'unknown',
      );
      const remaining = windows.length
        ? Math.min(...windows.map((w) => (Number.isFinite(w.remaining) ? Number(w.remaining) : 0)))
        : null;
      return { id, label: id || 'default', remaining };
    })
    .filter((choice) => choice.id);
}

/** All discovered provider profiles, with the current Intelligence selection kept visible. */
export function intelligenceCLIProfiles(
  accounts: ProviderAccount[] | null | undefined,
  current?: { provider?: string; profile?: string },
): ProviderAccount[] {
  const list = asArray<ProviderAccount>(accounts).filter((account) => !isTerminalOnly(account));
  const provider = current?.provider?.trim();
  const profile = current?.profile?.trim();
  let merged = list;
  if (provider && profile) {
    const key = `${provider}:${profile}`;
    if (!merged.some((account) => accountKey(account) === key)) {
      const existing = asArray<ProviderAccount>(accounts).find(
        (account) => accountKey(account) === key,
      );
      merged = [
        existing || {
          id: key,
          provider,
          profile,
          display_name: `${provider} · ${profile}`,
          authenticated: true,
          is_default: false,
          available: true,
          quota_remaining: 0,
          quota_total: 0,
          rate_limited: false,
          health: 'unknown',
          last_checked: '',
        },
        ...merged,
      ];
    }
  }
  return [...merged].sort((a, b) => {
    const rank = (account: ProviderAccount) =>
      (isIntelligenceCLICapable(account) ? 0 : 2) + (account.authenticated ? 0 : 1);
    const delta = rank(a) - rank(b);
    if (delta !== 0) return delta;
    return accountKey(a).localeCompare(accountKey(b));
  });
}
