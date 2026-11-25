import React from 'react';
import { useTranslation } from 'react-i18next';
import { X } from 'lucide-react';
import type { NexV2HudAlert } from './nexV2Types';
import styles from './NexNotificationBubble.module.scss';

export interface NexNotificationBubbleProps {
  alert: NexV2HudAlert | null;
  doNotDisturb: boolean;
  panelOpen: boolean;
  offline: boolean;
  onDismiss: (fingerprint: string) => void;
  onOpenAttention: () => void;
}

export const NexNotificationBubble: React.FC<NexNotificationBubbleProps> = ({
  alert,
  doNotDisturb,
  panelOpen,
  offline,
  onDismiss,
  onOpenAttention,
}) => {
  const { t } = useTranslation();

  if (doNotDisturb || panelOpen || offline || !alert) {
    return null;
  }

  const severity = (alert.severity || 'INFO').toUpperCase();
  const title =
    [alert.provider, alert.profile].filter(Boolean).join(' · ') || alert.category || 'Nexus';

  return (
    <aside
      className={styles.bubble}
      data-severity={severity}
      role="status"
      aria-live="polite"
      aria-label={t('nexV2.bubbleLabel')}
    >
      <div className={styles.content}>
        <span className={styles.title}>{title}</span>
        {alert.reason && <p className={styles.reason}>{alert.reason}</p>}
      </div>
      <div className={styles.actions}>
        <button type="button" className={styles.openBtn} onClick={onOpenAttention}>
          {t('nexV2.bubbleOpen')}
        </button>
        <button
          type="button"
          className={styles.dismissBtn}
          aria-label={t('nexV2.bubbleDismiss')}
          onClick={() => onDismiss(alert.fingerprint)}
        >
          <X size={14} aria-hidden="true" />
        </button>
      </div>
    </aside>
  );
};
