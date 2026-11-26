import React from 'react';
import { useTranslation } from 'react-i18next';
import { ChevronDown } from 'lucide-react';
import { resolveQuotaTone } from './nexV2HudHelpers';
import type { NexV2HudState, NexV2ModelCatalogWatchSnapshot } from './nexV2Types';
import styles from './NexResourcesTab.module.scss';

export interface NexResourcesTabProps {
  hudState: NexV2HudState | null;
  modelWatch: NexV2ModelCatalogWatchSnapshot | null;
  offline: boolean;
  privacyMode: boolean;
}

function resolveFreshnessInfo(
  freshness: string | undefined,
  t: (k: string) => string,
): { label: string; tone: 'fresh' | 'stale' | 'unavailable' | 'unknown'; raw: string } {
  const norm = (freshness || 'UNKNOWN').toUpperCase();
  if (norm === 'FRESH') {
    return { label: t('nexV2.resourcesStatusFresh'), tone: 'fresh', raw: norm };
  }
  if (norm === 'STALE') {
    return { label: t('nexV2.resourcesStatusStale'), tone: 'stale', raw: norm };
  }
  if (norm === 'UNAVAILABLE') {
    return { label: t('nexV2.resourcesStatusUnavailable'), tone: 'unavailable', raw: norm };
  }
  return { label: t('nexV2.resourcesStatusUnknown'), tone: 'unknown', raw: norm };
}

function resolveEvidenceLabel(evidence: string | undefined, t: (k: string) => string): string {
  const norm = (evidence || 'UNKNOWN').toUpperCase();
  if (norm === 'COMPLETE' || norm === 'FULL' || norm === 'EXACT') {
    return t('nexV2.evidenceComplete');
  }
  if (norm === 'PARTIAL') return t('nexV2.evidencePartial');
  return t('nexV2.evidenceUnknown');
}

function resolveChangeKindLabel(kind: string, t: (k: string) => string): string {
  switch (kind.toUpperCase()) {
    case 'ADDED':
      return t('nexV2.modelChangeAdded');
    case 'DEPRECATED':
      return t('nexV2.modelChangeDeprecated');
    case 'REMOVED':
      return t('nexV2.modelChangeRemoved');
    default:
      return kind;
  }
}

function formatResetText(
  resetDescription: string | undefined,
  resetAt: string | undefined,
): string | null {
  if (resetDescription && resetDescription.trim()) return resetDescription;
  if (!resetAt) return null;
  const parsed = new Date(resetAt);
  if (Number.isNaN(parsed.getTime())) return null;
  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
  }).format(parsed);
}

function formatUsdFromMicro(microusd: number): string {
  const usd = microusd / 1_000_000;
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(usd);
}

export const NexResourcesTab: React.FC<NexResourcesTabProps> = ({
  hudState,
  modelWatch,
  offline,
  privacyMode,
}) => {
  const { t } = useTranslation();
  const providers = hudState?.providers ?? [];
  const agents = hudState?.agents ?? [];
  const missions = hudState?.missions ?? [];
  const workingAgents = agents.filter((a) =>
    ['WORKING', 'RUNNING'].includes((a.status || '').toUpperCase()),
  ).length;
  const blockedMissions = missions.filter((m) => m.needs_user).length;
  const watchEntries = modelWatch?.entries ?? [];
  const watchChanges = modelWatch?.changes ?? [];
  const watchFreshness = resolveFreshnessInfo(modelWatch?.freshness, t);

  const spendDisplay = hudState?.cost
    ? privacyMode
      ? t('nexV2.privacyMasked')
      : formatUsdFromMicro(hudState.cost.known_microusd)
    : t('nexV2.resourcesQuotaUnknown');

  const spendSub = hudState?.cost
    ? resolveEvidenceLabel(hudState.cost.evidence, t)
    : t('nexV2.evidenceUnknown');

  return (
    <div className={styles.stack} data-nex-resources>
      <div className={styles.kpiGrid} data-nex-kpi-strip>
        <div className={styles.kpiTile}>
          <span className={styles.kpiLabel}>{t('nexV2.kpiAgents')}</span>
          <strong className={styles.kpiValue}>
            {t('nexV2.kpiAgentsActive', { working: workingAgents, total: agents.length })}
          </strong>
        </div>
        <div className={styles.kpiTile} data-tone={blockedMissions > 0 ? 'warning' : 'neutral'}>
          <span className={styles.kpiLabel}>{t('nexV2.kpiMissions')}</span>
          <strong className={styles.kpiValue}>{missions.length}</strong>
          {blockedMissions > 0 && (
            <span className={styles.kpiSubWarn}>
              {t('nexV2.kpiMissionsBlocked', { count: blockedMissions })}
            </span>
          )}
        </div>
        <div
          className={styles.kpiTile}
          data-nex-cost-card
          data-evidence={(hudState?.cost?.evidence || 'UNKNOWN').toUpperCase()}
        >
          <span className={styles.kpiLabel}>{t('nexV2.kpiSpend')}</span>
          <strong className={styles.kpiValue}>{spendDisplay}</strong>
          <span className={styles.kpiSub}>{spendSub}</span>
        </div>
      </div>

      {offline || providers.length === 0 ? (
        <div className={styles.emptyCard} data-freshness="UNKNOWN">
          <strong className={styles.emptyTitle}>{t('nexV2.resourcesStatusUnknown')}</strong>
          <p className={styles.emptyText}>{t('nexV2.resourcesEmpty')}</p>
        </div>
      ) : (
        providers.map((provider) => {
          const freshness = resolveFreshnessInfo(provider.freshness, t);
          const windows =
            provider.windows.length > 0
              ? provider.windows
              : provider.limiting_window
                ? [provider.limiting_window]
                : [];
          const profileLabel = privacyMode ? t('nexV2.privacyMasked') : provider.profile;

          return (
            <article key={`${provider.provider}-${provider.profile}`} className={styles.card}>
              <div className={styles.cardHeader}>
                <div className={styles.providerTitle}>
                  <strong>{provider.display_name || provider.provider}</strong>
                  {profileLabel && <span className={styles.profileTag}>· {profileLabel}</span>}
                </div>
                <div className={styles.badges}>
                  {provider.rate_limited && (
                    <span className={styles.limitBadge}>{t('nexV2.rateLimitedBadge')}</span>
                  )}
                  <span
                    className={styles.freshnessTag}
                    data-tone={freshness.tone}
                    data-freshness={freshness.raw}
                  >
                    <span className={styles.freshnessDot} aria-hidden="true" />
                    {freshness.label}
                  </span>
                </div>
              </div>

              {windows.length === 0 ? (
                <p className={styles.cardMeta} data-quota-state="UNKNOWN">
                  {t('nexV2.resourcesQuotaUnknown')}
                </p>
              ) : (
                <ul className={styles.windowList}>
                  {windows.map((win, idx) => {
                    const resetLabel = formatResetText(win.reset_description, win.reset_at);
                    const hasPercent = typeof win.remaining_percent === 'number';
                    const pct = hasPercent
                      ? Math.max(0, Math.min(100, Math.round(win.remaining_percent!)))
                      : null;
                    const tone = resolveQuotaTone(pct);
                    return (
                      <li key={`${win.kind}-${win.group || idx}`} className={styles.windowItem}>
                        <div className={styles.windowRow}>
                          <span className={styles.windowName}>
                            {win.label || win.kind}
                            {win.group ? (
                              <span className={styles.poolName}>
                                {' '}
                                · {t('nexV2.sharedPoolLabel', { group: win.group })}
                              </span>
                            ) : null}
                          </span>
                          <span
                            className={styles.windowValue}
                            data-tone={tone}
                            data-quota-state={hasPercent ? 'KNOWN' : 'UNKNOWN'}
                          >
                            {pct !== null
                              ? t('nexV2.resourcesRemaining', { percent: pct })
                              : t('nexV2.resourcesQuotaUnknown')}
                          </span>
                        </div>
                        <div
                          className={styles.progressTrack}
                          role="progressbar"
                          aria-label={win.label || win.kind}
                          aria-valuemin={0}
                          aria-valuemax={100}
                          aria-valuenow={pct ?? undefined}
                          data-tone={tone}
                        >
                          {pct !== null && (
                            <div
                              className={styles.progressFill}
                              data-tone={tone}
                              style={{ width: `${pct}%` }}
                            />
                          )}
                        </div>
                        {resetLabel && (
                          <span className={styles.windowReset}>
                            {t('nexV2.resetWindowLabel', { reset: resetLabel })}
                          </span>
                        )}
                      </li>
                    );
                  })}
                </ul>
              )}
            </article>
          );
        })
      )}

      <details className={styles.detailsCard} data-nex-model-watch>
        <summary className={styles.detailsSummary}>
          <div className={styles.detailsTitleGroup}>
            <ChevronDown size={14} className={styles.chevron} aria-hidden="true" />
            <strong>{t('nexV2.modelWatchTitle')}</strong>
            <span className={styles.summaryMeta}>
              {watchChanges.length > 0
                ? t('nexV2.modelWatchSummaryChanges', { count: watchChanges.length })
                : t('nexV2.modelWatchSummaryCount', { count: watchEntries.length })}
            </span>
          </div>
          <span
            className={styles.freshnessTag}
            data-tone={watchFreshness.tone}
            data-freshness={watchFreshness.raw}
          >
            <span className={styles.freshnessDot} aria-hidden="true" />
            {watchFreshness.label}
          </span>
        </summary>
        <div className={styles.detailsBody}>
          {watchEntries.length === 0 && watchChanges.length === 0 ? (
            <p className={styles.emptyText}>{t('nexV2.modelWatchEmpty')}</p>
          ) : (
            <ul className={styles.modelList}>
              {watchChanges.slice(-3).map((ch) => (
                <li key={ch.fingerprint} className={styles.modelItem}>
                  <span className={styles.changePill} data-change-kind={ch.kind}>
                    {resolveChangeKindLabel(ch.kind, t)}
                  </span>
                  <span className={styles.modelText}>
                    {ch.provider} · {ch.model_id}
                  </span>
                </li>
              ))}
              {watchChanges.length === 0 &&
                watchEntries.slice(0, 4).map((entry) => (
                  <li
                    key={`${entry.provider}-${entry.profile}-${entry.model_id}`}
                    className={styles.modelItem}
                  >
                    <span className={styles.windowName}>{entry.provider}</span>
                    <span className={styles.modelText}>{entry.model_id}</span>
                  </li>
                ))}
            </ul>
          )}
        </div>
      </details>
    </div>
  );
};
