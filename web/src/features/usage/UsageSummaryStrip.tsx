import React from 'react';
import { useTranslation } from 'react-i18next';
import { Sparkles } from 'lucide-react';
import type { FleetFilterStatus, UsageSummary } from './usageModel';
import styles from './UsageSummaryStrip.module.scss';

export interface UsageSummaryStripProps {
  summary: UsageSummary;
  bestLabel?: string | null;
  activeStatusFilter?: FleetFilterStatus;
  onSelectStatusFilter?: (status: FleetFilterStatus) => void;
  onSelectBest?: () => void;
}

export const UsageSummaryStrip: React.FC<UsageSummaryStripProps> = ({
  summary,
  bestLabel,
  activeStatusFilter = 'all',
  onSelectStatusFilter,
  onSelectBest,
}) => {
  const { t } = useTranslation();

  const handleToggle = (status: FleetFilterStatus) => {
    if (!onSelectStatusFilter) return;
    onSelectStatusFilter(activeStatusFilter === status ? 'all' : status);
  };

  return (
    <div className={styles.strip} role="group" aria-label={t('usage.summary', 'Resumo da frota')}>
      <button
        type="button"
        className={styles.metric}
        data-kind="usable"
        data-active={activeStatusFilter === 'usable'}
        onClick={() => handleToggle('usable')}
      >
        <strong>{summary.usable}</strong>
        <span>{t('usage.usable', 'Utilizáveis')}</span>
      </button>

      {summary.authRequired > 0 ? (
        <button
          type="button"
          className={styles.metric}
          data-kind="auth_required"
          data-active={activeStatusFilter === 'auth_required'}
          onClick={() => handleToggle('auth_required')}
        >
          <strong>{summary.authRequired}</strong>
          <span>{t('usage.authRequiredCount', 'Requer login')}</span>
        </button>
      ) : null}

      <button
        type="button"
        className={styles.metric}
        data-kind="blocked"
        data-active={activeStatusFilter === 'rate_limited'}
        onClick={() => handleToggle('rate_limited')}
      >
        <strong>{summary.blocked}</strong>
        <span>{t('usage.blocked', 'Rate limit')}</span>
      </button>

      <button
        type="button"
        className={styles.metric}
        data-kind="exhausted"
        data-active={activeStatusFilter === 'exhausted'}
        onClick={() => handleToggle('exhausted')}
      >
        <strong>{summary.exhausted}</strong>
        <span>{t('usage.exhausted', 'Esgotadas')}</span>
      </button>

      {bestLabel ? (
        <div
          className={styles.best}
          role="button"
          tabIndex={0}
          onClick={onSelectBest}
          onKeyDown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') onSelectBest?.();
          }}
        >
          <small>
            <Sparkles size={11} />
            {t('usage.bestChoice', 'Melhor escolha')}
          </small>
          <strong>{bestLabel}</strong>
        </div>
      ) : null}
    </div>
  );
};
