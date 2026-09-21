import React, { useMemo } from 'react';
import { CircleDollarSign, Clock3, Database, TriangleAlert } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { CostSummary } from '../../types';
import styles from './CostSummaryStrip.module.scss';

const formatUSD = (microUSD: number): string =>
  new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 4,
  }).format(microUSD / 1_000_000);

const formatNumber = (value: number): string =>
  new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(value);

const formatDuration = (durationMS: number): string => {
  const seconds = Math.max(0, Math.round(durationMS / 1000));
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  return rest ? `${minutes}m ${rest}s` : `${minutes}m`;
};

export const CostSummaryStrip: React.FC<{ summary: CostSummary | null }> = ({ summary }) => {
  const { t } = useTranslation();

  const knownTokens = useMemo(() => {
    if (!summary) return 0;
    const aggregate = summary.aggregate;
    return (
      aggregate.input_tokens +
      aggregate.cached_input_tokens +
      aggregate.output_tokens +
      aggregate.reasoning_tokens
    );
  }, [summary]);

  if (!summary || summary.aggregate.observation_count === 0) {
    return (
      <section className={styles.root} aria-label={t('usage.costTitle', 'Cost Intelligence')}>
        <div className={styles.heading}>
          <CircleDollarSign size={16} />
          <div>
            <strong>{t('usage.costTitle', 'Cost Intelligence')}</strong>
            <span>{t('usage.costEmpty', 'Nenhuma execução atribuída no período.')}</span>
          </div>
        </div>
      </section>
    );
  }

  const aggregate = summary.aggregate;
  const costKnown = aggregate.known_cost_observations > 0;
  const tokensKnown = aggregate.known_token_observations > 0;

  return (
    <section className={styles.root} aria-label={t('usage.costTitle', 'Cost Intelligence')}>
      <div className={styles.heading}>
        <CircleDollarSign size={16} />
        <div>
          <strong>{t('usage.costTitle', 'Cost Intelligence')}</strong>
          <span>
            {t('usage.costWindow', 'Últimos 7 dias')} · {aggregate.observation_count}{' '}
            {t('usage.executionsObserved', 'execuções observadas')}
          </span>
        </div>
      </div>

      <div className={styles.metrics}>
        <div className={styles.metric}>
          <span>{t('usage.knownCost', 'Custo conhecido')}</span>
          <strong>
            {costKnown ? formatUSD(aggregate.cost_microusd) : t('common.unknown', 'Desconhecido')}
          </strong>
          <small data-state={summary.cost_evidence}>{summary.cost_evidence}</small>
        </div>
        <div className={styles.metric}>
          <span>{t('usage.knownTokens', 'Tokens conhecidos')}</span>
          <strong>
            {tokensKnown ? formatNumber(knownTokens) : t('common.unknown', 'Desconhecido')}
          </strong>
          <small data-state={summary.token_evidence}>{summary.token_evidence}</small>
        </div>
        <div className={styles.metric}>
          <span>{t('usage.observationDuration', 'Tempo observado')}</span>
          <strong>{formatDuration(aggregate.duration_ms)}</strong>
          <small>
            <Clock3 size={11} /> {aggregate.observation_count}
          </small>
        </div>
        <div className={styles.metric}>
          <span>{t('usage.unmeasuredExecutions', 'Sem custo medido')}</span>
          <strong>{aggregate.unknown_cost_observations}</strong>
          <small>
            <TriangleAlert size={11} />
            {t('usage.notZeroCost', 'UNKNOWN ≠ zero')}
          </small>
        </div>
      </div>

      {summary.by_provider.length > 0 ? (
        <div className={styles.providers}>
          <Database size={13} />
          {summary.by_provider.slice(0, 5).map((provider) => (
            <span key={`${provider.provider}/${provider.profile}`}>
              <b>{provider.provider}</b>:{provider.profile}{' '}
              {provider.known_cost_observations > 0
                ? formatUSD(provider.cost_microusd)
                : t('common.unknown', 'Desconhecido')}
            </span>
          ))}
        </div>
      ) : null}
    </section>
  );
};
