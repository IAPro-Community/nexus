import React from 'react';
import { useTranslation } from 'react-i18next';
import { AlertCircle, ArrowRightLeft, BellOff, CheckCircle2 } from 'lucide-react';
import type { CapacityAlert, CapacityAlertRecommendation } from '../types';
import { Badge, Button } from '../design-system';
import { capacitySeverityTone, formatCapacityRemaining } from './notificationModel';
import styles from './CapacityAlertCard.module.scss';

export interface CapacityAlertCardProps {
  alert: CapacityAlert;
  onAcknowledge: (alertId: string) => void;
  onSnooze: (alertId: string, durationMinutes: number) => void;
  onSwitch?: (recommendation: CapacityAlertRecommendation) => void;
}

export const CapacityAlertCard: React.FC<CapacityAlertCardProps> = ({
  alert,
  onAcknowledge,
  onSnooze,
  onSwitch,
}) => {
  const { t } = useTranslation();
  const tone = capacitySeverityTone(alert.severity);
  const remaining = alert.remaining_percent;
  const hasRemaining = typeof remaining === 'number';
  const remainingStr = formatCapacityRemaining(remaining);
  const severityLower = alert.severity.toLowerCase();

  return (
    <article
      className={styles.alertCard}
      data-severity={severityLower}
      role="region"
      aria-label={t('capacityMonitor.alertAriaLabel', 'Alerta de capacidade')}
    >
      <header className={styles.header}>
        <div className={styles.badges}>
          <Badge tone={tone}>{alert.severity}</Badge>
          <Badge tone="default">{alert.provider}</Badge>
          {alert.profile && alert.profile !== 'default' && (
            <Badge tone="default">{alert.profile}</Badge>
          )}
        </div>
        {alert.window && <span className={styles.metricValue}>{alert.window}</span>}
      </header>

      <div className={styles.body}>
        <h4 className={styles.title}>{alert.subject}</h4>
        {alert.model_name && (
          <p className={styles.metricRow}>
            <span>{t('capacityMonitor.model', 'Modelo')}:</span>
            <span className={styles.metricValue}>{alert.model_name}</span>
          </p>
        )}
        {hasRemaining && (
          <>
            <div className={styles.metricRow}>
              <span>{t('capacityMonitor.remaining', 'Capacidade Restante')}:</span>
              <span className={styles.metricValue}>{remainingStr}</span>
            </div>
            <div
              className={styles.progressBar}
              role="progressbar"
              aria-valuenow={remaining}
              aria-valuemin={0}
              aria-valuemax={100}
            >
              <div
                className={styles.progressFill}
                data-severity={severityLower}
                style={{ width: `${Math.max(0, Math.min(100, remaining))}%` }}
              />
            </div>
          </>
        )}
        {alert.reset_at && (
          <p className={styles.metricRow}>
            <span>{t('capacityMonitor.resetAt', 'Reinício estimado')}:</span>
            <span className={styles.metricValue}>{alert.reset_at}</span>
          </p>
        )}
      </div>

      {alert.recommendation && (
        <section
          className={styles.recommendationBox}
          aria-label={t('capacityMonitor.recommendation', 'Recomendação')}
        >
          <div className={styles.recommendationHeader}>
            <span>{t('capacityMonitor.suggestedAlternative', 'Alternativa Sugerida')}</span>
            <Badge tone="brand">
              {alert.recommendation.provider} / {alert.recommendation.profile}
            </Badge>
          </div>
          {alert.recommendation.continuity_impact && (
            <p className={styles.recommendationDetails}>{alert.recommendation.continuity_impact}</p>
          )}
          {onSwitch && (
            <div className={styles.switchActionWrapper}>
              <Button
                tone="brand"
                onClick={() => onSwitch(alert.recommendation!)}
                aria-label={t('capacityMonitor.switchAction', 'Trocar Provedor')}
              >
                <ArrowRightLeft size={13} />
                {t('capacityMonitor.switchAction', 'Trocar Provedor')}
              </Button>
            </div>
          )}
        </section>
      )}

      <footer className={styles.actions}>
        <Button
          tone="default"
          onClick={() => onSnooze(alert.id, 60)}
          aria-label={t('capacityMonitor.snooze1h', 'Adiar 1h')}
        >
          <BellOff size={13} />
          {t('capacityMonitor.snooze1h', 'Adiar 1h')}
        </Button>
        <Button
          tone="default"
          onClick={() => onAcknowledge(alert.id)}
          aria-label={t('capacityMonitor.acknowledge', 'Confirmar')}
        >
          <CheckCircle2 size={13} />
          {t('capacityMonitor.acknowledge', 'Confirmar')}
        </Button>
      </footer>
    </article>
  );
};
