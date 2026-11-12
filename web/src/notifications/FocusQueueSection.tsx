import React from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, ListOrdered } from 'lucide-react';
import type { RuntimeSession } from '../types';
import { buildFocusQueue, type FocusQueueItem } from '../app/focusQueueModel';
import { sanitizeAttentionText } from '../components/attentionText';
import styles from './FocusQueueSection.module.scss';

/** Upper bound so a stale timestamp cannot render an absurd "waited" figure. */
const MAX_WAIT_MINUTES = 99 * 60 + 59;

function waitLabel(waitedMs: number, locale: string): string {
  const minutes = Math.min(MAX_WAIT_MINUTES, Math.max(0, Math.round(waitedMs / 60_000)));
  try {
    const format = new Intl.RelativeTimeFormat(locale, { numeric: 'always' });
    return format.format(-minutes, 'minute');
  } catch {
    return `${minutes} min`;
  }
}

/**
 * Single prioritized queue of pending work across every project: severity
 * first, oldest episode first. Rows deduplicate by attention episode and an
 * already answered episode never re-enters the queue, matching the durable
 * one-shot claim on the server.
 */
export const FocusQueueSection: React.FC<{
  runtimes: RuntimeSession[];
  onFocus: (item: FocusQueueItem) => void;
}> = ({ runtimes, onFocus }) => {
  const { t, i18n } = useTranslation();
  const queue = buildFocusQueue(runtimes);

  return (
    <section
      className={`nx-attention-radar__group ${styles.queue}`}
      data-tour="focus-queue"
      aria-label={t('notifications.focusQueueTitle', { defaultValue: 'Focus queue' })}
    >
      <header className={styles.header}>
        <strong>
          <ListOrdered size={12} aria-hidden="true" />
          {t('notifications.focusQueueTitle', { defaultValue: 'Focus queue' })}
        </strong>
        <span className={styles.count}>{queue.length}</span>
      </header>
      {queue.length === 0 ? (
        <p className={styles.empty}>
          {t('notifications.focusQueueEmpty', {
            defaultValue: 'Nothing is waiting for you right now.',
          })}
        </p>
      ) : (
        <ol className={styles.list}>
          {queue.map((item, index) => {
            const label = sanitizeAttentionText(item.context || item.title, item.title);
            return (
              <li key={item.key} className={styles.row}>
                <span className={styles.rank} aria-hidden="true">
                  {index + 1}
                </span>
                <button
                  type="button"
                  className="nx-attention-radar__item"
                  data-badge={item.severity}
                  data-runtime-id={item.runtimeId}
                  onClick={() => onFocus(item)}
                  title={label}
                >
                  <span className="nx-attention-radar__badge" data-badge={item.severity}>
                    <AlertTriangle size={12} aria-hidden="true" />
                    {t(`notifications.radarState.${item.severity}`, {
                      defaultValue: item.severity,
                    })}
                  </span>
                  <span className="nx-attention-radar__title">{label}</span>
                  <span className={styles.wait}>{waitLabel(item.waitedMs, i18n.language)}</span>
                </button>
              </li>
            );
          })}
        </ol>
      )}
    </section>
  );
};
