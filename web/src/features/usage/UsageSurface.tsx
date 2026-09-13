import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Gauge, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button, EmptyState, Spinner } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ProviderAccount, RecommendationResult } from '../../types';
import { asArray } from '../../lib/safeArray';
import { UsageAccountCard } from './UsageAccountCard';
import { UsagePolicyPanel } from './UsagePolicyPanel';
import { UsageRecommendRail } from './UsageRecommendRail';
import { UsageSummaryStrip } from './UsageSummaryStrip';
import { groupAccountsByProvider, summarizeAccounts } from './usageModel';
import { FRESHNESS_TICK_MS, USAGE_POLL_MS } from './usageRefresh';
import styles from './UsageSurface.module.scss';

export const UsageSurface: React.FC = () => {
  const { t } = useTranslation();
  const [accounts, setAccounts] = useState<ProviderAccount[]>([]);
  const [policy, setPolicy] = useState('BALANCED');
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');
  const [loadedAt, setLoadedAt] = useState<number | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const [recommend, setRecommend] = useState<RecommendationResult | null>(null);
  const [recommendLoading, setRecommendLoading] = useState(false);
  const [recommendError, setRecommendError] = useState('');

  const loadRecommend = useCallback(async (activePolicy: string) => {
    setRecommendLoading(true);
    setRecommendError('');
    try {
      const result = await nexus.recommendResources(
        { task_kind: 'general', role: 'developer' },
        activePolicy || 'BALANCED',
      );
      setRecommend(result);
    } catch (err) {
      setRecommend(null);
      setRecommendError(err instanceof Error ? err.message : String(err));
    } finally {
      setRecommendLoading(false);
    }
  }, []);

  const applyPayload = useCallback(
    (
      data: { accounts?: ProviderAccount[]; policy?: string },
      options?: { recommend?: boolean },
    ) => {
      const nextAccounts = asArray<ProviderAccount>(data.accounts);
      const nextPolicy = data.policy || 'BALANCED';
      setAccounts(nextAccounts);
      setPolicy(nextPolicy);
      setLoadedAt(Date.now());
      setNow(Date.now());
      if (options?.recommend !== false) {
        void loadRecommend(nextPolicy);
      }
    },
    [loadRecommend],
  );

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      applyPayload(await nexus.listResources());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setAccounts([]);
    } finally {
      setLoading(false);
    }
  }, [applyPayload]);

  const softReload = useCallback(async () => {
    if (document.hidden) return;
    try {
      // Cache read only — QuotaMonitor may have written fresher usage.json.
      applyPayload(await nexus.listResources(), { recommend: false });
    } catch {
      // Keep the last good snapshot visible; the next tick retries.
    }
  }, [applyPayload]);

  const refresh = useCallback(async () => {
    setRefreshing(true);
    setError('');
    try {
      applyPayload(await nexus.refreshResources());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setRefreshing(false);
    }
  }, [applyPayload]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    const onVisible = () => {
      if (!document.hidden) void softReload();
    };
    const poll = window.setInterval(() => {
      void softReload();
    }, USAGE_POLL_MS);
    const tick = window.setInterval(() => setNow(Date.now()), FRESHNESS_TICK_MS);
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      window.clearInterval(poll);
      window.clearInterval(tick);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [softReload]);

  const summary = useMemo(() => summarizeAccounts(accounts), [accounts]);
  const groups = useMemo(() => groupAccountsByProvider(accounts), [accounts]);
  const bestLabel = recommend?.recommended
    ? recommend.recommended.account.display_name ||
      `${recommend.recommended.account.provider}:${recommend.recommended.account.profile}`
    : null;

  return (
    <div className={`nx-surface-scroll ${styles.root}`}>
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">{t('surfaces.resourcesEyebrow')}</span>
          <h1>{t('surfaces.resourcesTitle')}</h1>
          <p>{t('surfaces.resourcesIntro')}</p>
        </div>
        <div className={styles.headerActions}>
          {loadedAt ? (
            <small className={styles.loadedAt}>
              {t('usage.loadedAt', 'Atualizado')}: {new Date(loadedAt).toLocaleTimeString()}
            </small>
          ) : null}
          <Button
            size="sm"
            tone="brand"
            disabled={loading || refreshing}
            onClick={() => void refresh()}
          >
            <RefreshCw size={13} className={refreshing ? 'nx-spin' : ''} />
            {refreshing
              ? t('usage.refreshing', 'Atualizando…')
              : t('usage.refresh', 'Atualizar cota')}
          </Button>
        </div>
      </div>

      {error ? <div className="nx-inline-error">{error}</div> : null}
      {loading ? <Spinner label={t('resources.loading')} /> : null}

      {!loading && accounts.length === 0 ? (
        <EmptyState
          icon={<Gauge size={20} />}
          title={t('resources.empty')}
          hint={t('resources.emptyHint')}
        />
      ) : null}

      {!loading && accounts.length > 0 ? (
        <div className={styles.layout}>
          <div className={styles.main}>
            <UsageSummaryStrip summary={summary} bestLabel={bestLabel} />
            <div className={styles.fleet}>
              {groups.map((group) => (
                <section key={group.provider} className={styles.providerGroup}>
                  <h2 className={styles.providerHeading}>
                    <span>{group.provider}</span>
                    <small>{t('resources.accountCount', { count: group.accounts.length })}</small>
                  </h2>
                  <div className={styles.accountGrid}>
                    {group.accounts.map((account) => (
                      <UsageAccountCard key={account.id} account={account} now={now} />
                    ))}
                  </div>
                </section>
              ))}
            </div>
          </div>
          <aside className={styles.rail}>
            <UsageRecommendRail
              loading={recommendLoading}
              result={recommend}
              error={recommendError}
            />
            <UsagePolicyPanel policy={policy} />
          </aside>
        </div>
      ) : null}
    </div>
  );
};
