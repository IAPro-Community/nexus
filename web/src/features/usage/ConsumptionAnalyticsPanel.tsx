import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Activity, ArrowDownToLine, CircleDollarSign, Database, RefreshCw } from 'lucide-react';
import { Button } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { CostAnalyticsReport, CostAnalyticsTotals } from '../../types';
import { cacheRate, costEvidenceLabel } from './consumptionAnalyticsModel';
import styles from './ConsumptionAnalyticsPanel.module.scss';

type Period = 7 | 30 | 90;
const periods: readonly Period[] = [7, 30, 90];
const refreshMs = 60_000;

function money(microUSD: number): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 4,
  }).format(microUSD / 1_000_000);
}

function amount(value: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(value);
}

function observedUSD(
  totals: CostAnalyticsTotals,
  kind: 'reported' | 'equivalent',
): string {
  const count =
    kind === 'reported'
      ? totals.reported_cost_observations
      : totals.equivalent_cost_observations;
  if (count === 0) return '—';
  return money(
    kind === 'reported'
      ? totals.provider_reported_microusd
      : totals.api_equivalent_microusd,
  );
}

export function ConsumptionAnalyticsPanel({ projectId }: { projectId?: string }) {
  const { t, i18n } = useTranslation();
  const [period, setPeriod] = useState<Period>(7);
  const [data, setData] = useState<CostAnalyticsReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    let active = true;
    const load = async (quiet = false) => {
      if (!quiet) {
        setLoading(true);
        setData(null);
      } else {
        setRefreshing(true);
      }
      setError('');
      try {
        const result = await nexus.getCostAnalytics(period, projectId);
        if (active) setData(result);
      } catch (reason) {
        if (active) setError(reason instanceof Error ? reason.message : t('consumption.loadError'));
      } finally {
        if (active) {
          setLoading(false);
          setRefreshing(false);
        }
      }
    };

    void load();
    const timer = window.setInterval(() => {
      if (!document.hidden) void load(true);
    }, refreshMs);
    const onVisible = () => {
      if (!document.hidden) void load(true);
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      active = false;
      window.clearInterval(timer);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [period, projectId, revision, t]);

  const cache = data ? cacheRate(data.total) : null;
  const maxDaily = useMemo(
    () => Math.max(1, ...(data?.by_day.map((bucket) => bucket.observation_count) || [])),
    [data],
  );

  return (
    <section className={styles.root} aria-label={t('consumption.title')}>
      <header className={styles.header}>
        <div className={styles.heading}>
          <Activity size={18} aria-hidden="true" />
          <div>
            <h2>{t('consumption.title')}</h2>
            <p>{t('consumption.subtitle')}</p>
          </div>
        </div>
        <div className={styles.actions}>
          <div className={styles.periods} role="group" aria-label={t('consumption.period')}>
            {periods.map((days) => (
              <button
                key={days}
                type="button"
                aria-pressed={days === period}
                className={styles.period}
                onClick={() => setPeriod(days)}
              >
                {t('consumption.days', { count: days })}
              </button>
            ))}
          </div>
          <Button
            size="sm"
            tone="ghost"
            disabled={loading || refreshing}
            onClick={() => setRevision((value) => value + 1)}
          >
            <RefreshCw size={14} aria-hidden="true" />
            {t('consumption.refresh')}
          </Button>
        </div>
      </header>

      {loading && <p role="status" className={styles.note}>{t('consumption.loading')}</p>}
      {error && <p role="alert" className={styles.error}>{error}</p>}
      {data && !loading && (
        <>
          <div className={styles.metrics}>
            <div className={styles.metric}>
              <span><CircleDollarSign size={14} /> {t('consumption.reportedCost')}</span>
              <strong>{observedUSD(data.total, 'reported')}</strong>
              <small>{t('consumption.reportedDisclaimer')}</small>
            </div>
            <div className={styles.metric}>
              <span><CircleDollarSign size={14} /> {t('consumption.apiEstimate')}</span>
              <strong>{observedUSD(data.total, 'equivalent')}</strong>
              <small>{t('consumption.estimateDisclaimer')}</small>
            </div>
            <div className={styles.metric}>
              <span><Database size={14} /> {t('consumption.observations')}</span>
              <strong>{amount(data.total.observation_count)}</strong>
              <small>
                {t('consumption.unknownCount', {
                  count: data.total.unknown_cost_observations,
                })}
              </small>
            </div>
            <div className={styles.metric}>
              <span><ArrowDownToLine size={14} /> {t('consumption.cacheRate')}</span>
              <strong>{cache === null ? '—' : `${(cache * 100).toFixed(1)}%`}</strong>
              <small>
                {t('consumption.cacheCoverage', {
                  measured: data.total.known_cache_observations,
                  total: data.total.observation_count,
                })}
              </small>
            </div>
          </div>

          <p className={styles.note}>{t('consumption.disclaimer')}</p>

          {data.total.observation_count === 0 ? (
            <p className={styles.empty}>{t('consumption.empty')}</p>
          ) : (
            <div className={styles.layout}>
              <div className={styles.section}>
                <h3>{t('consumption.daily')}</h3>
                <p className={styles.note}>{t('consumption.dailyExplanation')}</p>
                <ol className={styles.timeline}>
                  {data.by_day.map((bucket) => (
                    <li key={bucket.date} className={styles.day}>
                      <time dateTime={bucket.date}>
                        {new Date(`${bucket.date}T12:00:00Z`).toLocaleDateString(i18n.language, {
                          month: 'short',
                          day: '2-digit',
                        })}
                      </time>
                      <div
                        className={styles.barTrack}
                        role="img"
                        aria-label={t('consumption.dayObservations', {
                          date: bucket.date,
                          count: bucket.observation_count,
                        })}
                      >
                        <span
                          className={styles.bar}
                          style={{ width: `${(bucket.observation_count / maxDaily) * 100}%` }}
                        />
                      </div>
                      <strong>{amount(bucket.observation_count)}</strong>
                    </li>
                  ))}
                </ol>
              </div>
              <div className={styles.section}>
                <h3>{t('consumption.models')}</h3>
                <p className={styles.note}>{t('consumption.modelsExplanation')}</p>
                <div className={styles.tableScroll}>
                  <table className={styles.table}>
                    <thead>
                      <tr>
                        <th>{t('consumption.model')}</th>
                        <th>{t('consumption.observations')}</th>
                        <th>{t('consumption.reportedCost')}</th>
                        <th>{t('consumption.apiEstimate')}</th>
                        <th>{t('consumption.evidence')}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.by_model.map((item) => (
                        <tr key={`${item.provider}:${item.model}`}>
                          <th scope="row">
                            {item.model || t('consumption.unattributed')}
                            <small>{item.provider || t('consumption.unattributed')}</small>
                          </th>
                          <td>{amount(item.observation_count)}</td>
                          <td>{observedUSD(item, 'reported')}</td>
                          <td>{observedUSD(item, 'equivalent')}</td>
                          <td>
                            {t(
                              'consumption.evidenceStates.' +
                                costEvidenceLabel(
                                  item.known_cost_observations,
                                  item.observation_count,
                                ),
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {data.models_truncated && (
                  <p className={styles.note}>{t('consumption.truncated')}</p>
                )}
              </div>
            </div>
          )}

          <footer className={styles.footer}>
            <span>{t('consumption.source')}</span>
            <time dateTime={data.generated_at}>
              {new Date(data.generated_at).toLocaleString(i18n.language)}
            </time>
            {refreshing && <span role="status">{t('consumption.refreshing')}</span>}
          </footer>
        </>
      )}
    </section>
  );
}
