import React from 'react';
import { useTranslation } from 'react-i18next';
import type { NexV2HudState, NexV2ModelCatalogWatchSnapshot } from './nexV2Types';
import styles from './NexResourcesTab.module.scss';

export interface NexResourcesTabProps {
  hudState: NexV2HudState | null;
  modelWatch: NexV2ModelCatalogWatchSnapshot | null;
  offline: boolean;
  privacyMode: boolean;
}

function resolveFreshnessBadge(freshness: string | undefined, t: (k: string) => string): string {
  const norm = (freshness || '').toUpperCase();
  if (norm === 'STALE') return t('nexV2.resourcesStatusStale');
  if (norm === 'UNAVAILABLE') return t('nexV2.resourcesStatusUnavailable');
  if (!norm || norm === 'UNKNOWN') return t('nexV2.resourcesStatusUnknown');
  return norm;
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
  const watchEntries = modelWatch?.entries ?? [];
  const watchChanges = modelWatch?.changes ?? [];

  return (
    <div className={styles.stack} data-nex-resources>
      {offline || providers.length === 0 ? (
        <div className={styles.card}>
          <span className={styles.statusBadge}>{t('nexV2.resourcesStatusUnknown')}</span>
          <p className={styles.emptyText}>{t('nexV2.resourcesEmpty')}</p>
        </div>
      ) : (
        providers.map((provider) => {
          const freshness = resolveFreshnessBadge(provider.freshness, t);
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
                <strong>
                  {provider.display_name || provider.provider} · {profileLabel}
                </strong>
                <div className={styles.badges}>
                  {provider.rate_limited && (
                    <span className={styles.alertBadge}>{t('nexV2.rateLimitedBadge')}</span>
                  )}
                  <span className={styles.statusBadge}>{freshness}</span>
                </div>
              </div>

              {windows.length === 0 ? (
                <p className={styles.cardMeta}>{t('nexV2.resourcesQuotaUnknown')}</p>
              ) : (
                <ul className={styles.windowList}>
                  {windows.map((win, idx) => {
                    const resetLabel = formatResetText(win.reset_description, win.reset_at);
                    return (
                      <li key={`${win.kind}-${win.group || idx}`} className={styles.windowItem}>
                        <span className={styles.windowName}>
                          {win.label || win.kind}
                          {win.group
                            ? ` (${t('nexV2.sharedPoolLabel', { group: win.group })})`
                            : ''}
                        </span>
                        <span className={styles.windowValue}>
                          {typeof win.remaining_percent === 'number'
                            ? t('nexV2.resourcesRemaining', {
                                percent: Math.round(win.remaining_percent),
                              })
                            : t('nexV2.resourcesQuotaUnknown')}
                        </span>
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

      {hudState?.cost && (
        <section className={styles.card} data-nex-cost-card>
          <div className={styles.cardHeader}>
            <strong>
              {t('nexV2.resourcesCostTitle', {
                level: hudState.cost.budget_level || t('nexV2.resourcesStatusUnknown'),
              })}
            </strong>
            <span className={styles.statusBadge}>{hudState.cost.evidence}</span>
          </div>
          <p className={styles.cardMeta}>
            {t('nexV2.costKnownUsdLabel', {
              amount: privacyMode
                ? t('nexV2.privacyMasked')
                : formatUsdFromMicro(hudState.cost.known_microusd),
            })}
          </p>
        </section>
      )}

      <div className={styles.summaryRow}>
        <span className={styles.summaryChip}>
          {t('nexV2.resourcesAgentsTitle', { count: hudState?.agents.length ?? 0 })}
        </span>
        <span className={styles.summaryChip}>
          {t('nexV2.resourcesMissionsTitle', { count: hudState?.missions.length ?? 0 })}
        </span>
      </div>

      <section className={styles.card} data-nex-model-watch>
        <div className={styles.cardHeader}>
          <strong>{t('nexV2.modelWatchTitle')}</strong>
          <span className={styles.statusBadge}>
            {resolveFreshnessBadge(modelWatch?.freshness, t)}
          </span>
        </div>
        {watchEntries.length === 0 && watchChanges.length === 0 ? (
          <p className={styles.emptyText}>{t('nexV2.modelWatchEmpty')}</p>
        ) : (
          <ul className={styles.windowList}>
            {watchChanges.slice(-3).map((ch) => (
              <li key={ch.fingerprint} className={styles.windowItem}>
                <span className={styles.statusBadge}>{ch.kind}</span>
                <span className={styles.windowValue}>
                  {ch.provider} · {ch.model_id}
                </span>
              </li>
            ))}
            {watchChanges.length === 0 &&
              watchEntries.slice(0, 4).map((entry) => (
                <li
                  key={`${entry.provider}-${entry.profile}-${entry.model_id}`}
                  className={styles.windowItem}
                >
                  <span className={styles.windowName}>{entry.provider}</span>
                  <span className={styles.windowValue}>{entry.model_id}</span>
                </li>
              ))}
          </ul>
        )}
      </section>
    </div>
  );
};
