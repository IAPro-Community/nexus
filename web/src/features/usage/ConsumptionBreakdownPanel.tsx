import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ArrowLeft, Layers, RefreshCw } from 'lucide-react';
import { Button } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { CostObservation } from '../../types';
import {
  byRecencyDesc,
  costEvidenceLabel,
  formatDuration,
  formatMicroUSD,
  formatThroughput,
  formatTokens,
  groupObservations,
  type BreakdownGroup,
  type BreakdownGroupKind,
} from './consumptionAnalyticsModel';
import styles from './ConsumptionBreakdownPanel.module.scss';

type Period = 7 | 30 | 90;
const periods: readonly Period[] = [7, 30, 90];
const refreshMs = 60_000;
const observationLimit = 50;

interface DrillTarget {
  kind: BreakdownGroupKind;
  key: string;
}

/** A measured token amount, or null when the observation did not report it. */
function measured(value: number | undefined): string {
  return typeof value === 'number' ? formatTokens(value) : '—';
}

function ObservationRow({ observation }: { observation: CostObservation }) {
  const { t } = useTranslation();
  const costEvidence = costEvidenceLabel(observation.cost_microusd === undefined ? 0 : 1, 1);
  return (
    <li className={styles.row}>
      <div className={styles.rowHead}>
        <span className={styles.model}>
          {observation.model || t('consumption.unattributed')}
          <small>{observation.provider || t('consumption.unattributed')}</small>
        </span>
        {observation.status ? (
          <span className={styles.status}>{observation.status}</span>
        ) : (
          <span className={styles.status} data-absent="true">
            {t('consumption.breakdown.statusUnknown')}
          </span>
        )}
        <time dateTime={observation.observed_at} className={styles.observedAt}>
          {new Date(observation.observed_at).toLocaleString(undefined, {
            dateStyle: 'short',
            timeStyle: 'short',
          })}
        </time>
      </div>
      <dl className={styles.facts}>
        <div className={styles.fact}>
          <dt>{t('consumption.breakdown.tokensIn')}</dt>
          <dd>{measured(observation.input_tokens)}</dd>
        </div>
        <div className={styles.fact}>
          <dt>{t('consumption.breakdown.tokensOut')}</dt>
          <dd>{measured(observation.output_tokens)}</dd>
        </div>
        <div className={styles.fact}>
          <dt>{t('consumption.breakdown.tokensCache')}</dt>
          <dd>{measured(observation.cached_input_tokens)}</dd>
        </div>
        <div className={styles.fact}>
          <dt>{t('consumption.breakdown.cost')}</dt>
          <dd>
            {observation.cost_microusd === undefined
              ? '—'
              : formatMicroUSD(observation.cost_microusd)}{' '}
            <small>{t(`consumption.evidenceStates.${costEvidence}`)}</small>
          </dd>
        </div>
        <div className={styles.fact}>
          <dt>{t('consumption.breakdown.duration')}</dt>
          <dd>{formatDuration(observation.duration_ms)}</dd>
        </div>
        <div className={styles.fact}>
          <dt>{t('consumption.breakdown.ttft')}</dt>
          <dd>
            {observation.ttft_ms === undefined ? '—' : `${formatTokens(observation.ttft_ms)} ms`}
          </dd>
        </div>
        <div className={styles.fact}>
          <dt>{t('consumption.breakdown.tps')}</dt>
          <dd>
            {observation.output_tokens_per_second === undefined
              ? '—'
              : formatThroughput(observation.output_tokens_per_second)}
          </dd>
        </div>
        {observation.session_id ? (
          <div className={styles.fact}>
            <dt>{t('consumption.breakdown.session')}</dt>
            <dd className={styles.mono}>{observation.session_id}</dd>
          </div>
        ) : null}
      </dl>
    </li>
  );
}

export function ConsumptionBreakdownPanel({ projectId }: { projectId?: string }) {
  const { t } = useTranslation();
  const [period, setPeriod] = useState<Period>(7);
  const [observations, setObservations] = useState<CostObservation[] | null>(null);
  const [detail, setDetail] = useState<CostObservation[] | null>(null);
  const [drill, setDrill] = useState<DrillTarget | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [error, setError] = useState('');
  const [detailError, setDetailError] = useState('');
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    let active = true;
    const load = async (quiet = false) => {
      setError('');
      setDetailError('');
      if (drill) {
        if (!quiet) {
          setDetailLoading(true);
          setDetail(null);
        }
        try {
          const rows = await nexus.getCostObservations({
            limit: observationLimit,
            days: period,
            projectId,
            missionRunId: drill.kind === 'mission' ? drill.key || undefined : undefined,
            agentId: drill.kind === 'agent' ? drill.key || undefined : undefined,
          });
          if (active) setDetail(rows.slice().sort(byRecencyDesc));
        } catch (reason) {
          if (active) {
            setDetailError(
              reason instanceof Error ? reason.message : t('consumption.breakdown.loadError'),
            );
          }
        } finally {
          if (active) setDetailLoading(false);
        }
        return;
      }
      if (!quiet) {
        setLoading(true);
        setObservations(null);
      } else {
        setRefreshing(true);
      }
      try {
        const rows = await nexus.getCostObservations({
          limit: observationLimit,
          days: period,
          projectId,
        });
        if (active) setObservations(rows.slice().sort(byRecencyDesc));
      } catch (reason) {
        if (active) {
          setError(reason instanceof Error ? reason.message : t('consumption.breakdown.loadError'));
        }
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
  }, [period, projectId, drill, revision, t]);

  const missionGroups = useMemo(
    () => (observations ? groupObservations(observations, 'mission') : []),
    [observations],
  );
  const agentGroups = useMemo(
    () => (observations ? groupObservations(observations, 'agent') : []),
    [observations],
  );

  const groupLabel = (group: BreakdownGroup): string => group.key || t('consumption.unattributed');

  const openGroup = (group: BreakdownGroup) => {
    setDrill({ kind: group.kind, key: group.key });
  };

  const renderGroups = (kind: BreakdownGroupKind, groups: readonly BreakdownGroup[]) => (
    <div className={styles.groupBlock}>
      <h3>
        {t(
          kind === 'mission' ? 'consumption.breakdown.byMission' : 'consumption.breakdown.byAgent',
        )}
      </h3>
      {groups.length === 0 ? (
        <p className={styles.empty}>{t('consumption.breakdown.noGroups')}</p>
      ) : (
        <ul className={styles.groups}>
          {groups.map((group) => (
            <li key={`${group.kind}:${group.key}`}>
              <button
                type="button"
                className={styles.groupButton}
                aria-label={t('consumption.breakdown.openGroup', {
                  kind: t(
                    group.kind === 'mission'
                      ? 'consumption.breakdown.kindMission'
                      : 'consumption.breakdown.kindAgent',
                  ),
                  id: groupLabel(group),
                  count: group.count,
                })}
                onClick={() => openGroup(group)}
              >
                <span className={styles.groupId}>{groupLabel(group)}</span>
                <span className={styles.groupCount}>
                  {t('consumption.breakdown.groupCount', { count: group.count })}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );

  return (
    <section className={styles.root} aria-label={t('consumption.breakdown.title')}>
      <header className={styles.header}>
        <div className={styles.heading}>
          <Layers size={18} aria-hidden="true" />
          <div>
            <h2>{t('consumption.breakdown.title')}</h2>
            <p>{t('consumption.breakdown.subtitle')}</p>
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
                onClick={() => {
                  setPeriod(days);
                  setDrill((current) => (current ? { ...current } : null));
                }}
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

      {drill ? (
        <div className={styles.detailArea}>
          <div className={styles.detailHead}>
            <Button size="sm" tone="ghost" onClick={() => setDrill(null)}>
              <ArrowLeft size={14} aria-hidden="true" />
              {t('consumption.breakdown.back')}
            </Button>
            <h3>
              {t('consumption.breakdown.detailFor', {
                kind: t(
                  drill.kind === 'mission'
                    ? 'consumption.breakdown.kindMission'
                    : 'consumption.breakdown.kindAgent',
                ),
                id: drill.key || t('consumption.unattributed'),
              })}
            </h3>
          </div>
          <p role="status" aria-live="polite" className={styles.note}>
            {detailLoading ? t('consumption.breakdown.detailLoading') : ''}
          </p>
          {detailError && (
            <p role="alert" className={styles.error}>
              {detailError}
            </p>
          )}
          {!detailLoading && !detailError && detail && detail.length === 0 && (
            <p className={styles.empty}>{t('consumption.breakdown.detailEmpty')}</p>
          )}
          {!detailLoading && detail && detail.length > 0 && (
            <ul className={styles.rows} aria-label={t('consumption.breakdown.detailTitle')}>
              {detail.map((observation) => (
                <ObservationRow key={observation.id} observation={observation} />
              ))}
            </ul>
          )}
        </div>
      ) : (
        <>
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
          {observations && !loading && !error && (
            <>
              {missionGroups.length === 0 && agentGroups.length === 0 ? (
                <p className={styles.empty}>{t('consumption.breakdown.empty')}</p>
              ) : (
                <>
                  {renderGroups('mission', missionGroups)}
                  {renderGroups('agent', agentGroups)}
                </>
              )}
              {refreshing && <span role="status">{t('consumption.refreshing')}</span>}
            </>
          )}
        </>
      )}
    </section>
  );
}
