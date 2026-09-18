import { asArray } from '../../lib/safeArray';

export interface DirectResourceLike {
  id: string;
  provider: string;
  profile: string;
  authenticated: boolean;
  available: boolean;
}

export type QuotaTruthState =
  'confirmed' | 'exhausted' | 'stale' | 'unknown' | 'blocked' | 'unauthenticated';

export function quotaTruthState(resource: {
  available: boolean;
  authenticated?: boolean;
  rate_limited?: boolean;
  avail_reasons?: { unknown_quota?: boolean; exhausted_windows?: string[]; rate_limited?: boolean };
  quota_view?: { status?: string; model_groups?: QuotaGroupLike[] };
}): QuotaTruthState {
  // Unauthenticated accounts (e.g. expired token) must never show confirmed quota.
  if (resource.authenticated === false) return 'unauthenticated';
  if (resource.rate_limited || resource.avail_reasons?.rate_limited) return 'blocked';
  if ((resource.avail_reasons?.exhausted_windows || []).length > 0) return 'exhausted';
  const status = String(resource.quota_view?.status || '').toUpperCase();
  const remaining = bestGroupRemainingFromQuotaView(resource.quota_view);
  if (remaining != null && remaining <= 0) return 'exhausted';
  if (status === 'LIVE' && remaining != null && resource.available) return 'confirmed';
  if (status === 'CACHED' || status === 'ESTIMATED') return 'stale';
  return 'unknown';
}

export interface QuotaWindowLike {
  kind?: string;
  remaining?: number;
}

export interface QuotaGroupLike {
  key?: string;
  name?: string;
  windows?: QuotaWindowLike[];
}

const providerLabel = (provider: string) =>
  provider.trim().replace(/(^|[-_\s])\w/g, (part) => part.toUpperCase());

export function buildDirectAgentName(prompt: string, provider: string): string {
  const label = providerLabel(provider || 'AI');
  const cleaned = (prompt || '').trim().replace(/\s+/g, ' ');
  if (!cleaned) return `${label} · Direct Session`;
  const words = cleaned.split(' ');
  const short = words.slice(0, 3).join(' ');
  return `${label} · ${short}`;
}

export function directAccountTitle(resource: {
  display_name?: string;
  provider: string;
  profile: string;
}): string {
  const name = (resource.display_name || '').trim();
  if (name) return name;
  const profile = (resource.profile || '').trim();
  if (!profile || profile === 'default') return resource.provider;
  return `${resource.provider} (${profile})`;
}

function groupRemaining(group?: QuotaGroupLike): number | null {
  let min: number | null = null;
  for (const window of asArray<QuotaWindowLike>(group?.windows)) {
    if (!window || window.kind === 'unknown') continue;
    const remaining = Number(window.remaining);
    if (!Number.isFinite(remaining)) continue;
    min = min == null ? remaining : Math.min(min, remaining);
  }
  return min;
}

/** Min across all windows — warning bottleneck, not account score. */
export function bottleneckRemainingFromQuotaView(view?: {
  model_groups?: QuotaGroupLike[];
}): number | null {
  let min: number | null = null;
  for (const group of asArray<QuotaGroupLike>(view?.model_groups)) {
    const rem = groupRemaining(group);
    if (rem == null) continue;
    min = min == null ? rem : Math.min(min, rem);
  }
  return min == null ? null : Math.round(min);
}

/** Max of per-group mins — independent pools (AGY) do not cancel each other. */
export function bestGroupRemainingFromQuotaView(view?: {
  model_groups?: QuotaGroupLike[];
}): number | null {
  let best: number | null = null;
  for (const group of asArray<QuotaGroupLike>(view?.model_groups)) {
    const rem = groupRemaining(group);
    if (rem == null) continue;
    best = best == null ? rem : Math.max(best, rem);
  }
  return best == null ? null : Math.round(best);
}

function shortGroupLabel(group: QuotaGroupLike): string {
  const key = String(group.key || group.name || '').toLowerCase();
  if (key.includes('gemini')) return 'Gemini';
  if (key.includes('claude') || key.includes('gpt')) return 'Claude';
  const name = String(group.name || '').trim();
  return name || 'Quota';
}

function shortWindowLabel(kind?: string): string {
  switch (kind) {
    case '5h':
    case 'daily':
    case 'claude_5h':
    case 'claude_five_hour':
      return '5h';
    case 'weekly':
    case 'claude_weekly':
      return 'weekly';
    default:
      return kind || 'quota';
  }
}

/** Compact label: "Gemini 0% · Claude 100%" or "5h 70% · weekly 95%". */
export function compactQuotaLabel(view?: { model_groups?: QuotaGroupLike[] }): string | null {
  const groups = asArray<QuotaGroupLike>(view?.model_groups).filter(
    (group) => groupRemaining(group) != null,
  );
  if (groups.length === 0) return null;
  if (groups.length > 1) {
    return groups
      .map((group) => `${shortGroupLabel(group)} ${Math.round(groupRemaining(group) as number)}%`)
      .join(' · ');
  }
  const windows = asArray<QuotaWindowLike>(groups[0].windows).filter(
    (w) => w && w.kind !== 'unknown' && Number.isFinite(Number(w.remaining)),
  );
  if (windows.length === 0) return null;
  return windows
    .map((w) => `${shortWindowLabel(w.kind)} ${Math.round(Number(w.remaining))}%`)
    .join(' · ');
}

export function directQuotaPercent(resource: {
  quota_remaining?: number;
  avail_reasons?: { unknown_quota?: boolean };
  quota_view?: { status?: string; model_groups?: QuotaGroupLike[] };
}): number | null {
  const status = String(resource.quota_view?.status || '').toUpperCase();
  const known =
    ['LIVE', 'CACHED', 'ESTIMATED'].includes(status) && !resource.avail_reasons?.unknown_quota;
  if (!known) return null;
  const fromWindows = bestGroupRemainingFromQuotaView(resource.quota_view);
  if (fromWindows != null) return fromWindows;
  if (!Number.isFinite(resource.quota_remaining)) return null;
  const raw = resource.quota_remaining ?? 0;
  return Math.round(raw * (raw <= 1 ? 100 : 1));
}

export function directQuotaDisplay(resource: {
  quota_remaining?: number;
  avail_reasons?: { unknown_quota?: boolean };
  quota_view?: { status?: string; model_groups?: QuotaGroupLike[]; fetched_at?: string };
}): string | null {
  const known = !resource.avail_reasons?.unknown_quota && resource.quota_view?.status !== 'UNKNOWN';
  if (!known) return null;
  const label = compactQuotaLabel(resource.quota_view);
  if (label) return label;
  const pct = directQuotaPercent(resource);
  return pct == null ? null : `${pct}%`;
}

export function eligibleDirectResources<T extends DirectResourceLike>(
  accounts: T[],
  preferProvider = '',
): T[] {
  const preferred = preferProvider.trim().toLowerCase();
  return accounts
    .filter((account) => account.authenticated && account.available)
    .sort((a, b) => {
      const ap = a.provider.toLowerCase() === preferred ? 1 : 0;
      const bp = b.provider.toLowerCase() === preferred ? 1 : 0;
      if (ap !== bp) return bp - ap;
      return `${a.provider}:${a.profile}`.localeCompare(`${b.provider}:${b.profile}`);
    });
}
