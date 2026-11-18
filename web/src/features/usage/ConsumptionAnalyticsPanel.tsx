import { useEffect, useMemo, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Activity,
  ArrowDownToLine,
  CircleDollarSign,
  Database,
  Download,
  RefreshCw,
} from 'lucide-react';
import { Button } from '../../design-system';
import { downloadAuthenticatedCsv } from '../../lib/downloadAuthenticatedCsv';
import { nexus } from '../../nexus/api';
import type { CostAnalyticsReport, CostAnalyticsTotals } from '../../types';
import {
  cacheRate,
  costEvidenceLabel,
  dailyMetricAvailable,
  dailyMetricMax,
  dailyMetricValue,
  formatAmount,
  formatMicroUSD,
  type DailyMetric,
} from './consumptionAnalyticsModel';
import styles from './ConsumptionAnalyticsPanel.module.scss';

type Period = 7 | 30 | 90;
const periods: readonly Period[] = [7, 30, 90];
const refreshMs = 60_000;
const metricOrder: readonly DailyMetric[] = ['invocations', 'tokens', 'cost'];

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

function observedUSD(totals: CostAnalyticsTotals, kind: 'reported' | 'equivalent'): string {
  const count =
    kind === 'reported' ? totals.reported_cost_observations : totals.equivalent_cost_observations;
  if (count === 0) return '—';
  const value =
    kind === 'reported' ? totals.provider_reported_microusd : totals.api_equivalent_microusd;
  return money(value);
}

function formatMetricValue(metric: DailyMetric, value: number): string {
  if (metric === 'cost') return formatMicroUSD(value);
  return formatAmount(value);
}

export function ConsumptionAnalyticsPanel({ projectId }: { projectId?: string }) {
  const { t, i18n } = useTranslation();
  const [period, setPeriod] = useState<Period>(7);
  const [data, setData] = useState<CostAnalyticsReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');
  const [revision, setRevision] = useState(0);
  const [preferredMetric, setPreferredMetric] = useState<DailyMetric>('invocations');
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState('');

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
  const byDay = useMemo(() => data?.by_day ?? [], [data]);
  // A selected metric without measured evidence in the current report falls
  // back to invocations instead of plotting fabricated zeros.
  const activeMetric: DailyMetric = useMemo(
    () => (dailyMetricAvailable(byDay, preferredMetric) ? preferredMetric : 'invocations'),
    [byDay, preferredMetric],
  );
  const maxDaily = useMemo(() => dailyMetricMax(byDay, activeMetric), [byDay, activeMetric]);

  const exportCsv = () => {
    if (!data || data.total.observation_count === 0 || exporting) return;
    setExporting(true);
    setExportError('');
    downloadAuthenticatedCsv(
      `/api/v1/cost/observations.csv?days=${period}`,
      `nexus-cost-observations-${period}d.csv`,
    )
      .catch((reason: unknown) => {
        setExportError(reason instanceof Error ? reason.message : t('consumption.exportCsvError'));
      })
      .finally(() => setExporting(false));
  };

  const onMetricKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    const index = metricOrder.indexOf(activeMetric);
    const delta =
      event.key === 'ArrowRight' || event.key === 'ArrowDown'
        ? 1
        : event.key === 'ArrowLeft' || event.key === 'ArrowUp'
          ? -1
          : 0;
    if (delta === 0) return;
    event.preventDefault();
    for (let step = 1; step <= metricOrder.length; step += 1) {
      const next =
        metricOrder[(index + delta * step + metricOrder.length * 2) % metricOrder.length];
      if (dailyMetricAvailable(byDay, next)) {
        setPreferredMetric(next);
        return;
      }
    }
  };

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
          <Button
            size="sm"
            tone="ghost"
            disabled={loading || exporting || !data || data.total.observation_count === 0}
            onClick={exportCsv}
          >
            <Download size={14} aria-hidden="true" />
            {exporting ? t('consumption.exportCsvBusy') : t('consumption.exportCsv')}
          </Button>
        </div>
      </header>
      {exportError && (
        <p role="alert" className={styles.error}>
          {exportError}
        </p>
      )}

      {loading && (
        <p role="status" className={styles.note}>
          {t('consumption.loading')}
        </p>
      )}
      {error && (
        <p role="alert" className={styles.error}>
          {error}
        </p>
      )}
      {data && !loading && (
        <>
          <div className={styles.metrics}>
            <div className={styles.metric}>
              <span>
                <CircleDollarSign size={14} /> {t('consumption.reportedCost')}
              </span>
              <strong>{observedUSD(data.total, 'reported')}</strong>
              <small>{t('consumption.reportedDisclaimer')}</small>
            </div>
            <div className={styles.metric}>
              <span>
                <CircleDollarSign size={14} /> {t('consumption.apiEstimate')}
              </span>
              <strong>{observedUSD(data.total, 'equivalent')}</strong>
              <small>{t('consumption.estimateDisclaimer')}</small>
            </div>
            <div className={styles.metric}>
              <span>
                <Database size={14} /> {t('consumption.observations')}
              </span>
              <strong>{amount(data.total.observation_count)}</strong>
              <small>
                {t('consumption.unknownCount', {
                  count: data.total.unknown_cost_observations,
                })}
              </small>
            </div>
            <div className={styles.metric}>
              <span>
                <ArrowDownToLine size={14} /> {t('consumption.cacheRate')}
              </span>
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
          {data.total.inferred_cost_observations > 0 && (
            <p className={styles.note}>
              {t('consumption.inferred', { value: money(data.total.inferred_microusd) })}
            </p>
          )}

          {data.total.observation_count === 0 ? (
            <p className={styles.empty}>{t('consumption.empty')}</p>
          ) : (
            <div className={styles.layout}>
              <div className={styles.section}>
                <h3>{t('consumption.daily')}</h3>
                <div
                  className={styles.metricToggle}
                  role="radiogroup"
                  aria-label={t('consumption.metric')}
                  onKeyDown={onMetricKeyDown}
                >
                  {metricOrder.map((metric) => {
                    const available = dailyMetricAvailable(byDay, metric);
                    const checked = activeMetric === metric;
                    return (
                      <button
                        key={metric}
                        type="button"
                        role="radio"
                        aria-checked={checked}
                        aria-disabled={!available}
                        tabIndex={checked ? 0 : -1}
                        className={styles.metricOption}
                        onClick={() => {
                          if (available) setPreferredMetric(metric);
                        }}
                      >
                        {t(`consumption.metricOptions.${metric}`)}
                      </button>
                    );
                  })}
                </div>
                <p className={styles.note}>
                  {activeMetric === 'invocations'
                    ? t('consumption.dailyExplanation')
                    : t(`consumption.dailyExplanationFor.${activeMetric}`)}
                </p>
                <ol className={styles.timeline}>
                  {data.by_day.map((bucket) => {
                    const value = dailyMetricValue(bucket, activeMetric);
                    const hasEvidence = value !== null;
                    return (
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
                          aria-label={
                            hasEvidence
                              ? t(`consumption.dayMetric.${activeMetric}`, {
                                  date: bucket.date,
                                  value: formatMetricValue(activeMetric, value),
                                })
                              : t('consumption.dayNoEvidence', {
                                  date: bucket.date,
                                  metric: t(`consumption.metricOptions.${activeMetric}`),
                                })
                          }
                          data-missing={hasEvidence ? undefined : 'true'}
                        >
                          <span
                            className={styles.bar}
                            style={{
                              width: hasEvidence ? `${(value / maxDaily) * 100}%` : undefined,
                            }}
                          />
                        </div>
                        <strong>
                          {hasEvidence ? formatMetricValue(activeMetric, value) : '—'}
                        </strong>
                      </li>
                    );
                  })}
                </ol>
                <p className={styles.note}>{t('consumption.dayMissingHint')}</p>
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
